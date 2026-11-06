package adapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/familyprofile"
	parserpolicy "nexora.local/nli-v20/parser-policy"
	"nexora.local/nli-v20/telemetry"
)

const (
	// ResolveResilient already caps the controller-managed generation at
	// MaxManagedPacks. Keep enough additional capacity for the reviewed embedded
	// families and the SMTP compatibility profile so a valid managed parser is
	// never discarded merely because more than eight parser packs are active.
	maxProfileExecutions       = familyprofile.MaxManagedPacks + 16
	unclassifiedRuntimeService = "__unclassified__"
)

var (
	nativeFamilyHeader = regexp.MustCompile(`^\s*\d{8}\s+\d{9}[+-]\d{4}\s+(\S+)\s+(\S+)\s+`)
	webtopFamilyHeader = regexp.MustCompile(`^\s*\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}:\d{3}[+-]\d{4}\s+`)
)

type ParserPackAdapter struct {
	profiles          []familyprofile.Profile
	smtpCompatibility familyprofile.Profile
	policyHash        string
}

// NewParserPackAdapter creates the default runtime from the embedded,
// product-reviewed family profiles.
func NewParserPackAdapter() (*ParserPackAdapter, error) {
	return NewParserPackAdapterFromDirectory("")
}

// NewParserPackAdapterFromDirectory merges the embedded profiles with one
// bounded controller-managed active generation. The directory may be absent.
func NewParserPackAdapterFromDirectory(directory string) (*ParserPackAdapter, error) {
	profiles, diagnostics, err := familyprofile.ResolveResilient(directory)
	if err != nil {
		return nil, err
	}
	for _, diagnostic := range diagnostics {
		log.Printf("WARNING: managed parser pack %q isolated: %s; built-in and validated managed parsers remain active", diagnostic.Name, diagnostic.Reason)
	}
	if len(profiles) == 0 {
		return nil, &configurationError{message: "no approved parser profiles"}
	}
	compatibilityPack := parserpolicy.DefaultSMTPParserPack()
	compatibilityExecutor, err := parserpolicy.NewParserPackExecutor(compatibilityPack)
	if err != nil {
		return nil, err
	}
	compatibility := familyprofile.Profile{
		Pack: compatibilityPack, Executor: compatibilityExecutor,
		Order: 90, Name: "embedded-smtp-golden-compatibility", Managed: false,
	}
	policyProfiles := append(append([]familyprofile.Profile(nil), profiles...), compatibility)
	return &ParserPackAdapter{
		profiles: profiles, smtpCompatibility: compatibility,
		policyHash: aggregatePolicyHash(policyProfiles),
	}, nil
}

type configurationError struct{ message string }

func (err *configurationError) Error() string { return err.message }

type V5ParseResult struct {
	EventPayload    *telemetry.EventPayload
	ParseTelemetry  *telemetry.ParseTelemetry
	Success         bool
	Quarantined     bool
	ReviewedMapping bool
}

type profileCandidate struct {
	profile     familyprofile.Profile
	specificity int
}

type profileExecution struct {
	candidate profileCandidate
	result    *parserpolicy.ExecutionResult
	err       error
}

func (adapter *ParserPackAdapter) TryParseV5Production(input string, source engine.Source) *V5ParseResult {
	return adapter.tryParseV5Production(input, source, false)
}

func (adapter *ParserPackAdapter) tryParseV5Production(input string, source engine.Source, reviewedOnly bool) *V5ParseResult {
	started := time.Now()
	trimmed := strings.TrimSpace(input)
	service := runtimeService(source.Service, trimmed)
	if service == "" {
		return &V5ParseResult{Success: false}
	}
	candidates := adapter.candidates(service)
	if reviewedOnly {
		filtered := candidates[:0]
		for _, candidate := range candidates {
			if reviewedMappingProfile(candidate.profile) {
				filtered = append(filtered, candidate)
			}
		}
		candidates = filtered
	}
	if !reviewedOnly && (service == "smtp" || service == "smtp-in" || service == "smtp-out") {
		candidates = append([]profileCandidate{{profile: adapter.smtpCompatibility, specificity: 0}}, candidates...)
	}
	hasReviewedMapping := false
	for index := range candidates {
		if reviewedMappingProfile(candidates[index].profile) {
			candidates[index].specificity = -1
			hasReviewedMapping = true
		}
	}
	if hasReviewedMapping {
		sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].specificity < candidates[j].specificity })
	}
	if len(candidates) == 0 {
		return &V5ParseResult{Success: false}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	limitExceeded := len(candidates) > maxProfileExecutions
	if len(candidates) > maxProfileExecutions {
		candidates = candidates[:maxProfileExecutions]
	}

	// Execute candidates in selector tiers: exact, compatible family, then
	// wildcard recovery. Parser executors are intentionally evaluated in stable
	// order because some reviewed parsing stages reuse mutable scratch objects;
	// concurrent evaluation made a selected parser's field types nondeterministic.
	// Once a more-specific tier succeeds, less-specific tiers are never evaluated,
	// preserving both the fast path and exact-parser authority.
	executions := make([]profileExecution, 0, len(candidates))
	execute := func(candidate profileCandidate) profileExecution {
		executionService := service
		if candidate.profile.Pack.ID == adapter.smtpCompatibility.Pack.ID {
			executionService = "smtp"
		}
		executionInput := trimmed
		if mappingProfile(candidate.profile) {
			executionInput = input
		}
		result, err := candidate.profile.Executor.Execute(&parserpolicy.ExecutionRequest{
			Input: executionInput, NodeID: source.Hostname, Service: executionService, SourceType: "file", Context: ctx,
			SourceMetadata: map[string]string{
				"hostname": source.Hostname,
				"service":  service,
				"file":     source.File,
			},
		})
		return profileExecution{candidate: candidate, result: result, err: err}
	}
	currentSpecificity := candidates[0].specificity
	tierSucceeded := false
	for _, candidate := range candidates {
		if candidate.specificity != currentSpecificity {
			if tierSucceeded {
				break
			}
			currentSpecificity = candidate.specificity
			tierSucceeded = false
		}
		execution := execute(candidate)
		executions = append(executions, execution)
		if successfulExecution(execution.result, execution.err) {
			tierSucceeded = true
		}
	}
	selectedIndex := -1
	for index := range executions {
		if successfulExecution(executions[index].result, executions[index].err) && (selectedIndex < 0 || betterExecution(executions[index], executions[selectedIndex])) {
			selectedIndex = index
		}
	}

	metadata := adapter.aggregateTelemetry(trimmed, source.Hostname, executions, selectedIndex, started)
	if limitExceeded {
		metadata.ErrorCodes = appendUnique(metadata.ErrorCodes, "pack_attempt_limit")
	}
	if selectedIndex < 0 {
		metadata.Route = "quarantine"
		metadata.Result = "quarantined"
		metadata.Confidence = 0
		metadata.ErrorCodes = appendUnique(metadata.ErrorCodes, "all_packs_failed")
		return &V5ParseResult{ParseTelemetry: metadata, Success: false, Quarantined: true}
	}

	selected := executions[selectedIndex].result
	if selected.EventPayload == nil || selected.EventPayload.FieldCount() == 0 {
		metadata.Route = "quarantine"
		metadata.Result = "quarantined"
		metadata.Confidence = 0
		metadata.ErrorCodes = appendUnique(metadata.ErrorCodes, "empty_event_payload")
		return &V5ParseResult{ParseTelemetry: metadata, Success: false, Quarantined: true}
	}
	return &V5ParseResult{EventPayload: selected.EventPayload, ParseTelemetry: metadata, Success: true, ReviewedMapping: reviewedMappingProfile(executions[selectedIndex].candidate.profile)}
}

func (adapter *ParserPackAdapter) candidates(service string) []profileCandidate {
	tiers := [3][]profileCandidate{}
	for _, profile := range adapter.profiles {
		specificity, eligible := profileSpecificity(profile.Pack, service)
		if !eligible {
			continue
		}
		tiers[specificity] = append(tiers[specificity], profileCandidate{profile: profile, specificity: specificity})
	}
	candidates := make([]profileCandidate, 0, len(adapter.profiles))
	for specificity := range tiers {
		candidates = append(candidates, tiers[specificity]...)
	}
	return candidates
}

func profileSpecificity(pack *parserpolicy.ParserPack, service string) (int, bool) {
	best := 3
	for _, rule := range pack.Rules {
		for _, selector := range rule.ServiceSelectors {
			normalized := canonicalService(selector)
			switch {
			case normalized == service:
				best = 0
			case (service == "smtp-in" || service == "smtp-out") && normalized == "smtp" && best > 1:
				best = 1
			case selector == "*" && best > 2:
				best = 2
			}
		}
	}
	return best, best < 3
}

func successfulExecution(result *parserpolicy.ExecutionResult, err error) bool {
	return err == nil && result != nil && !result.Quarantined && result.EventPayload != nil && result.EventPayload.FieldCount() > 0
}

func betterExecution(candidate, current profileExecution) bool {
	candidateTelemetry := candidate.result.ParseTelemetry
	currentTelemetry := current.result.ParseTelemetry
	// NFI output schemas are an explicit, reviewed operator choice. A managed
	// mapping may replace a built-in interpretation only after its complete
	// bounded plan and required fields succeed. Other profiles retain the
	// established specificity and evidence ordering below.
	if reviewedMappingProfile(candidate.candidate.profile) != reviewedMappingProfile(current.candidate.profile) {
		return reviewedMappingProfile(candidate.candidate.profile)
	}
	// A successful exact or family parser must remain authoritative over a
	// successful wildcard recovery parser. Field count only breaks ties inside
	// the same selector tier.
	if candidate.candidate.specificity != current.candidate.specificity {
		return candidate.candidate.specificity < current.candidate.specificity
	}
	if candidateTelemetry.SourceFieldCount != currentTelemetry.SourceFieldCount {
		return candidateTelemetry.SourceFieldCount > currentTelemetry.SourceFieldCount
	}
	if candidateTelemetry.OutputFieldCount != currentTelemetry.OutputFieldCount {
		return candidateTelemetry.OutputFieldCount > currentTelemetry.OutputFieldCount
	}
	if candidate.candidate.profile.Order != current.candidate.profile.Order {
		return candidate.candidate.profile.Order < current.candidate.profile.Order
	}
	if candidate.candidate.profile.Pack.Revision != current.candidate.profile.Pack.Revision {
		return candidate.candidate.profile.Pack.Revision > current.candidate.profile.Pack.Revision
	}
	return candidate.candidate.profile.Pack.ID < current.candidate.profile.Pack.ID
}

// Mapping selectors address the original source; legacy profiles retain trimmed input.
func mappingProfile(profile familyprofile.Profile) bool {
	if profile.Pack == nil {
		return false
	}
	for _, rule := range profile.Pack.Rules {
		for _, stage := range rule.Stages {
			if stage.Type == parserpolicy.StageTypeMapping {
				return true
			}
		}
	}
	return false
}

func reviewedMappingProfile(profile familyprofile.Profile) bool {
	return profile.Managed && profile.Pack != nil && profile.Pack.Metadata["selection_policy"] == "reviewed_mapping" && profile.Pack.Metadata["mapping_engine"] == "nfi-v1"
}

func (adapter *ParserPackAdapter) aggregateTelemetry(input, nodeID string, executions []profileExecution, selectedIndex int, started time.Time) *telemetry.ParseTelemetry {
	var metadata *telemetry.ParseTelemetry
	if selectedIndex >= 0 && executions[selectedIndex].result != nil && executions[selectedIndex].result.ParseTelemetry != nil {
		copyOfSelected := *executions[selectedIndex].result.ParseTelemetry
		metadata = &copyOfSelected
	} else {
		metadata = telemetry.NewParseTelemetry(input, nodeID)
		metadata.PackID = "runtime_family_router"
		metadata.ParserID = "runtime_family_router"
		metadata.ParserVersion = parserpolicy.CurrentSchemaVersion
		metadata.PolicyHash = adapter.policyHash
	}

	metadata.OrderedAttempts = make([]telemetry.ParserAttempt, 0, familyprofile.MaxAttemptsPerPack*len(executions))
	metadata.ErrorCodes = make([]string, 0)
	metadata.AttemptCount = 0
	metadata.StageCount = 0
	metadata.FallbackDepth = 0
	attemptsBeforeSelected := 0
	for index, execution := range executions {
		attempts := executionAttempts(execution)
		if index < selectedIndex {
			attemptsBeforeSelected += len(attempts)
		}
		metadata.OrderedAttempts = append(metadata.OrderedAttempts, attempts...)
		metadata.AttemptCount += len(attempts)
		if execution.result != nil && execution.result.ParseTelemetry != nil {
			metadata.StageCount += execution.result.ParseTelemetry.StageCount
			for _, code := range execution.result.ParseTelemetry.ErrorCodes {
				metadata.ErrorCodes = appendUnique(metadata.ErrorCodes, code)
			}
		}
		if execution.err != nil {
			metadata.ErrorCodes = appendUnique(metadata.ErrorCodes, "parser_execution_failed")
		}
	}
	if selectedIndex >= 0 {
		metadata.FallbackDepth = attemptsBeforeSelected
	}
	metadata.LatencyMs = elapsedMillis(started)
	return metadata
}

func executionAttempts(execution profileExecution) []telemetry.ParserAttempt {
	if execution.result != nil && execution.result.ParseTelemetry != nil && len(execution.result.ParseTelemetry.OrderedAttempts) > 0 {
		return append([]telemetry.ParserAttempt(nil), execution.result.ParseTelemetry.OrderedAttempts...)
	}
	reason := "no_eligible_rule"
	outcome := telemetry.AttemptOutcomeRejected
	if execution.err != nil {
		reason = "parser_execution_failed"
		outcome = telemetry.AttemptOutcomeFailed
	} else if execution.result != nil && execution.result.ParseTelemetry != nil && len(execution.result.ParseTelemetry.ErrorCodes) > 0 {
		reason = execution.result.ParseTelemetry.ErrorCodes[len(execution.result.ParseTelemetry.ErrorCodes)-1]
	}
	return []telemetry.ParserAttempt{{
		ParserID: execution.candidate.profile.Pack.ID,
		Priority: execution.candidate.profile.Order,
		Outcome:  outcome, FailureReason: reason,
	}}
}

// RuntimeService shares the exact production routing decision with reviewed
// candidate validation so inferred scopes and fleet execution cannot diverge.
func RuntimeService(configured, input string) string { return runtimeService(configured, input) }

func runtimeService(configured, input string) string {
	configured = canonicalService(configured)
	inferred := inferService(input)
	switch configured {
	case "", "auto", "generic", "nli":
		if inferred == "" {
			// Preserve a concrete routing key so globally approved wildcard packs
			// can evaluate unstructured inputs that have no built-in family header.
			return unclassifiedRuntimeService
		}
		return inferred
	case "smtp":
		if inferred == "smtp-in" || inferred == "smtp-out" {
			return inferred
		}
	}
	return configured
}

func inferService(input string) string {
	if webtopFamilyHeader.MatchString(input) {
		return "webtop"
	}
	match := nativeFamilyHeader.FindStringSubmatch(input)
	if len(match) != 3 {
		return ""
	}
	hostname := strings.ToLower(match[1])
	service := canonicalService(match[2])
	if service != "smtp" {
		return service
	}
	lowerInput := strings.ToLower(input)
	switch {
	case strings.Contains(lowerInput, "policy=inbound"), strings.Contains(hostname, "rgi"):
		return "smtp-in"
	case strings.Contains(lowerInput, "policy=outbound"), strings.Contains(hostname, "rgo"):
		return "smtp-out"
	default:
		return "smtp"
	}
}

func canonicalService(service string) string {
	switch strings.ToLower(strings.TrimSpace(service)) {
	case "imapserv":
		return "imap"
	case "popserv":
		return "pop"
	case "webmail":
		return "webtop"
	case "pss":
		return "pabd"
	case "mss":
		return "capd"
	default:
		return strings.ToLower(strings.TrimSpace(service))
	}
}

func aggregatePolicyHash(profiles []familyprofile.Profile) string {
	var material strings.Builder
	for _, profile := range profiles {
		material.WriteString(profile.Pack.ID)
		material.WriteByte(0)
		material.WriteString(profile.Pack.SHA256)
		material.WriteByte('\n')
	}
	digest := sha256.Sum256([]byte(material.String()))
	return hex.EncodeToString(digest[:])
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func elapsedMillis(started time.Time) int64 {
	elapsed := time.Since(started)
	if elapsed <= 0 {
		return 0
	}
	milliseconds := elapsed.Milliseconds()
	if milliseconds == 0 {
		return 1
	}
	return milliseconds
}

func (adapter *ParserPackAdapter) TryParseV5Raw(input string, source engine.Source) *engine.V5ParseResult {
	return adapter.TryParseV5(input, source)
}

// Only reviewed managed mappings may precede the existing JSON fast path.
func (adapter *ParserPackAdapter) TryParseReviewedV5Raw(input string, source engine.Source) *engine.V5ParseResult {
	result := adapter.tryParseV5Production(input, source, true)
	return adapter.engineResult(result)
}

func (adapter *ParserPackAdapter) TryParseV5(input string, source engine.Source) *engine.V5ParseResult {
	return adapter.engineResult(adapter.TryParseV5Production(input, source))
}

func (adapter *ParserPackAdapter) engineResult(result *V5ParseResult) *engine.V5ParseResult {
	metadata := map[string]interface{}{}
	parserID := ""
	confidence, coverage, latency := 0.0, 0.0, int64(0)
	attemptCount, fallbackDepth := 0, 0
	trace := []map[string]interface{}{}
	fieldEvidence := []map[string]interface{}{}
	if result.ParseTelemetry != nil {
		telemetryResult := result.ParseTelemetry
		metadata["event_id"] = telemetryResult.EventID
		metadata["parser_id"] = telemetryResult.ParserID
		metadata["confidence"] = telemetryResult.Confidence
		metadata["evidence_coverage"] = telemetryResult.EvidenceCoverage
		metadata["route"] = telemetryResult.Route
		metadata["attempt_count"] = telemetryResult.AttemptCount
		metadata["fallback_depth"] = telemetryResult.FallbackDepth
		metadata["stage_count"] = telemetryResult.StageCount
		parserID = telemetryResult.ParserID
		confidence = telemetryResult.Confidence
		coverage = telemetryResult.EvidenceCoverage
		latency = telemetryResult.LatencyMs
		attemptCount = telemetryResult.AttemptCount
		fallbackDepth = telemetryResult.FallbackDepth
		for attemptIndex, attempt := range telemetryResult.OrderedAttempts {
			trace = append(trace, map[string]interface{}{
				"stage": "parser_attempt", "attempt": attemptIndex + 1,
				"parser": attempt.ParserID, "priority": attempt.Priority,
				"outcome": string(attempt.Outcome), "success": attempt.Outcome == telemetry.AttemptOutcomeSuccess,
				"failure_reason": attempt.FailureReason, "latency_ms": attempt.LatencyMs,
				"fields": attempt.FieldsEmitted, "selected": attempt.ParserID == telemetryResult.ParserID,
			})
			for stageIndex, stage := range attempt.StageResults {
				trace = append(trace, map[string]interface{}{
					"stage": "stage_execution", "attempt": attemptIndex + 1, "step": stageIndex + 1,
					"parser": attempt.ParserID, "stage_id": stage.StageID, "stage_name": stage.StageName,
					"stage_type": stage.StageType, "outcome": string(stage.Outcome),
					"success": stage.Outcome == telemetry.StageOutcomeSuccess,
					"fields":  stage.FieldsAdded, "fields_modified": stage.FieldsModified,
					"latency_ms": stage.LatencyMs, "failure_reason": stage.ErrorCode,
				})
			}
		}
		for _, evidence := range telemetryResult.FieldEvidence {
			fieldEvidence = append(fieldEvidence, map[string]interface{}{
				"field": evidence.Field, "source_backed": evidence.Source,
				"start": evidence.Start, "end": evidence.End, "derived_rule": evidence.DerivedRule,
			})
		}
	}
	payload := map[string]interface{}{}
	if result.EventPayload != nil {
		payload = result.EventPayload.Fields
	}
	return &engine.V5ParseResult{
		EventPayload: payload, ParseTelemetry: metadata,
		Success: result.Success, Quarantined: result.Quarantined, ReviewedMapping: result.ReviewedMapping,
		ParserID: parserID, Confidence: confidence, EvidenceCoverage: coverage, LatencyMs: latency,
		AttemptCount: attemptCount, FallbackDepth: fallbackDepth, Trace: trace, FieldEvidence: fieldEvidence,
	}
}
