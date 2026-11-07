package controlplane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"nexora.local/nli-v20/internal/adapter"
	"nexora.local/nli-v20/internal/fieldmapping"
	metastore "nexora.local/nli-v20/internal/store"
	parserpolicy "nexora.local/nli-v20/parser-policy"
)

// One declarative plan is used by the editor, Assurance, sealed bundles and
// enrolled clients. A browser preview never authorizes or publishes a parser.
func decodeMappingPlan(value any) (fieldmapping.Plan, error) {
	var plan fieldmapping.Plan
	data, err := json.Marshal(value)
	if err != nil {
		return plan, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return plan, fmt.Errorf("invalid NFI mapping plan: %w", err)
	}
	_, err = fieldmapping.Compile(plan)
	return plan, err
}

func (s *Server) handleMappingPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	ctx, done, admitted := s.beginFieldSenseAnalysis(w, r)
	if !admitted {
		return
	}
	defer done()
	var input struct {
		Raw  string          `json:"raw"`
		Plan json.RawMessage `json:"plan"`
	}
	if err := decodeJSON(r, &input, 2<<20); err != nil || len(input.Raw) > 1<<20 || len(input.Plan) == 0 {
		jsonError(w, "bounded raw evidence and a mapping plan are required", http.StatusBadRequest)
		return
	}
	plan, err := decodeMappingPlan(input.Plan)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	compiled, err := fieldmapping.Compile(plan)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if fieldSenseContextError(w, ctx) {
		return
	}
	result, err := compiled.ExecuteTrace(ctx, input.Raw)
	if err == nil {
		err = compiled.ValidateComplete(input.Raw, result)
	}
	if fieldSenseContextError(w, ctx) {
		return
	}
	if err != nil {
		jsonOK(w, map[string]any{"passed": false, "actual": map[string]any{}, "trace": result.Trace, "evidence": result.Evidence, "errors": []string{err.Error()}, "persisted": false, "activated": false, "engine": "nfi-v1"})
		return
	}
	jsonOK(w, map[string]any{"passed": true, "actual": result.Values, "fields": result.Evidence, "evidence": result.Evidence, "trace": result.Trace, "assessment": assessMappingEvidence(result, len(input.Raw)), "errors": []string{}, "persisted": false, "activated": false, "engine": "nfi-v1"})
}

// Assess only fields emitted for this sample. Defaults and derived values
// can be valid output without having a direct source span.
func assessMappingEvidence(result fieldmapping.Result, inputBytes int) map[string]any {
	observed := 0
	for name := range result.Values {
		evidence, ok := result.Evidence[name]
		if ok && evidence.Start >= 0 && evidence.End > evidence.Start && evidence.End <= inputBytes {
			observed++
		}
	}
	total := len(result.Values)
	assessment := map[string]any{
		"observed_fields": observed, "output_fields": total,
		"fields_without_direct_span": total - observed,
		"structured_coverage_passed": true,
	}
	if total > 0 {
		assessment["source_evidence_pct"] = 100 * observed / total
	}
	return assessment
}

func (s *Server) handleFieldMappingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	docs, err := s.metadata.List(r.Context(), "parser-packs", metastore.Query{Limit: 1000})
	if err != nil {
		jsonError(w, "NFI parser lifecycle status unavailable", http.StatusServiceUnavailable)
		return
	}
	states := map[string]int{}
	profiles := []map[string]any{}
	for _, doc := range docs {
		var body struct {
			Parser map[string]any `json:"parser"`
			Name   string         `json:"name"`
		}
		if json.Unmarshal(doc.Payload, &body) != nil || body.Parser["pattern_type"] != "mapping" {
			continue
		}
		states[doc.State]++
		profiles = append(profiles, map[string]any{"id": doc.ID, "name": body.Name, "state": doc.State, "revision": doc.Revision, "mapping_plan": body.Parser["mapping_plan"]})
	}
	sort.Slice(profiles, func(i, j int) bool { return fmt.Sprint(profiles[i]["id"]) < fmt.Sprint(profiles[j]["id"]) })
	admission := s.fieldSenseAnalysis()
	jsonOK(w, map[string]any{
		"subsystem": "nfi", "name": "NLI FieldSense Studio", "engine": "nfi-v1", "available": true,
		"analysis":     admission.status(),
		"kvie":         admission.kvStatus(),
		"generated_at": time.Now().UTC(), "states": states, "profiles": profiles, "profile_count": len(profiles),
		"methods": fieldmapping.Methods, "operations": fieldmapping.Operations, "types": fieldmapping.Types,
		"limits":                              map[string]int{"max_input_bytes": fieldmapping.MaxInputBytes, "max_fields": fieldmapping.MaxFields, "max_operations_per_field": fieldmapping.MaxOperations, "max_learning_samples": 100},
		"minimum_client_release":              "21.0.0-7",
		"dynamic_tail_minimum_client_release": "21.0.0-9",
		"execution":                           "The same bounded native plan runs in controller preview and enrolled clients. Legacy dynamic EOL plans require R9; KVIE profiles require R10. Saved drafts require validation, approval and exact-bundle acknowledgement before replay.",
		"learning":                            "Deterministic structural suggestions from operator samples; ambiguous meanings require operator labels. No external AI calls or automatic parser activation.",
	})
}

func nfiCaseService(item assuranceRequest) string {
	return adapter.RuntimeService(item.Service, item.Raw)
}

// Scope is reviewed alongside the plan. Inferred mappings cannot take over
// unrelated services; an authoritative global override requires a bounded guard.
// Priority orders only reviewed native mappings within their reviewed scope.
func nfiMappingPriority(definition map[string]any) (int, error) {
	value, exists := definition["mapping_priority"]
	if !exists || value == nil {
		return 0, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return 0, fmt.Errorf("invalid NFI mapping priority")
	}
	priority, err := strconv.Atoi(string(data))
	if err != nil || priority < 0 || priority > 1000 {
		return 0, fmt.Errorf("NFI mapping priority must be an integer from 0 to 1000")
	}
	return priority, nil
}

func nfiMappingScope(definition map[string]any, plan fieldmapping.Plan, cases []assuranceRequest) ([]string, string, bool, error) {
	if _, err := nfiMappingPriority(definition); err != nil {
		return nil, "", false, err
	}
	pattern, _ := definition["scope_pattern"].(string)
	global, _ := definition["reviewed_global_scope"].(bool)
	if value, exists := definition["scope_pattern"]; exists && value != nil {
		if _, ok := value.(string); !ok {
			return nil, "", false, fmt.Errorf("NFI scope pattern must be a string")
		}
	}
	if value, exists := definition["reviewed_global_scope"]; exists && value != nil {
		if _, ok := value.(bool); !ok {
			return nil, "", false, fmt.Errorf("NFI global scope flag must be boolean")
		}
	}
	if len(pattern) > 1024 {
		return nil, "", false, fmt.Errorf("NFI scope pattern exceeds 1024 bytes")
	}
	if pattern != "" {
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, "", false, fmt.Errorf("invalid NFI scope pattern: %w", err)
		}
	}
	if global && pattern == "" {
		return nil, "", false, fmt.Errorf("global NFI override requires a reviewed scope pattern")
	}
	scopes := map[string]bool{}
	add := func(service string) {
		service = strings.ToLower(strings.TrimSpace(service))
		switch service {
		case "imapserv":
			service = "imap"
		case "popserv":
			service = "pop"
		case "webmail":
			service = "webtop"
		case "pss":
			service = "pabd"
		case "mss":
			service = "capd"
		}
		if service != "" && service != "auto" {
			scopes[service] = true
		}
		if service == "smtp" {
			scopes["smtp-in"] = true
			scopes["smtp-out"] = true
		}
	}
	switch selected := definition["service_scope"].(type) {
	case string:
		for _, service := range strings.Split(selected, ",") {
			add(service)
		}
	case []any:
		for _, service := range selected {
			text, ok := service.(string)
			if !ok {
				return nil, "", false, fmt.Errorf("NFI service scope must contain strings")
			}
			add(text)
		}
	case []string:
		for _, service := range selected {
			add(service)
		}
	case nil:
	default:
		return nil, "", false, fmt.Errorf("invalid NFI service scope")
	}
	if global {
		scopes = map[string]bool{"*": true}
	}
	if len(scopes) == 0 {
		for _, item := range cases {
			add(nfiCaseService(item))
		}
	}
	if len(scopes) == 0 {
		add("__unclassified__")
	}
	if len(scopes) > 32 {
		return nil, "", false, fmt.Errorf("NFI scope exceeds 32 services")
	}
	for service := range scopes {
		if service != "*" && !regexp.MustCompile(`^[a-z0-9_][a-z0-9_.-]{0,63}$`).MatchString(service) {
			return nil, "", false, fmt.Errorf("invalid NFI service scope %q", service)
		}
	}
	constantOnly := true
	for _, field := range plan.Fields {
		if field.Method != "constant" && field.Method != "derived" {
			constantOnly = false
		}
	}
	if constantOnly && pattern == "" {
		inputs := []string{}
		seen := map[string]bool{}
		for _, item := range cases {
			if !seen[item.Raw] && item.Raw != "" {
				inputs = append(inputs, regexp.QuoteMeta(item.Raw))
				seen[item.Raw] = true
			}
		}
		if len(inputs) == 0 {
			return nil, "", false, fmt.Errorf("constant-only NFI plans require reviewed input cases or a scope pattern")
		}
		pattern = "^(?:" + strings.Join(inputs, "|") + ")$"
		if len(pattern) > 1024 {
			return nil, "", false, fmt.Errorf("constant-only NFI cases require an explicit bounded scope pattern")
		}
	}
	for _, item := range cases {
		if !scopes["*"] && !scopes[nfiCaseService(item)] {
			return nil, "", false, fmt.Errorf("reviewed case service %q is outside NFI scope", nfiCaseService(item))
		}
		if pattern != "" && !regexp.MustCompile(pattern).MatchString(item.Raw) {
			return nil, "", false, fmt.Errorf("reviewed case does not match NFI scope pattern")
		}
	}
	selectors := make([]string, 0, len(scopes))
	for service := range scopes {
		selectors = append(selectors, service)
	}
	sort.Strings(selectors)
	return selectors, pattern, !scopes["*"] || global, nil
}

func assuranceMappingRuntimePack(id, version, author string, definition map[string]any, cases []assuranceRequest) (*parserpolicy.ParserPack, error) {
	plan, err := decodeMappingPlan(definition["mapping_plan"])
	if err != nil {
		return nil, err
	}
	selectors, scopePattern, authoritative, err := nfiMappingScope(definition, plan, cases)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(plan.Fields))
	dynamic := false
	minimumClient := "21.0.0-7"
	for _, field := range plan.Fields {
		if field.OutputPolicy == "flatten" || field.Method == "kv_all" && (field.Source != "" || field.EndOfLine) {
			minimumClient = "21.0.0-9"
		}
		if field.OutputPolicy == "flatten" {
			dynamic = true
			continue
		}
		names = append(names, field.Name)
	}
	for _, field := range plan.Fields {
		if field.KVProfile != "" {
			minimumClient = "21.0.0-10"
			break
		}
	}
	for _, field := range plan.Fields {
		if field.KVNestedObjects {
			minimumClient = "22.0.0-1"
			break
		}
	}
	sort.Strings(names)
	limits := parserpolicy.DefaultResourceLimits()
	limits.MaxAttempts = 1
	// Config is decoded into a map by the strict pack loader. Normalize it
	// before sealing so typed Go structs and JSON reloads use identical keys.
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	var stageConfig map[string]any
	if err := json.Unmarshal(planJSON, &stageConfig); err != nil {
		return nil, err
	}
	pack := &parserpolicy.ParserPack{ID: id, Version: version, Revision: 1, SourceType: "manual", Author: author, ReviewState: parserpolicy.ReviewStateApproved, CreatedAt: time.Now().UTC(), ResourceLimits: limits,
		Metadata: map[string]string{"subsystem": "nfi", "mapping_engine": "nfi-v1", "minimum_client_release": minimumClient},
		Rules: []parserpolicy.ParserRule{{ID: id + "-rule", Name: id, Priority: 1, ServiceSelectors: selectors,
			Stages:        []parserpolicy.Stage{{ID: id + "-mapping", Name: "NFI native field mapping", Type: parserpolicy.StageType("mapping"), Config: stageConfig, FailurePolicy: parserpolicy.FailurePolicyRejectRule}},
			FailurePolicy: parserpolicy.FailurePolicyQuarantine, ProjectionMode: parserpolicy.ProjectionModeAllowlist, AllowedFields: names, CountPolicy: parserpolicy.CountPolicyProjection}}}
	if authoritative {
		pack.Metadata["selection_policy"] = "reviewed_mapping"
		priority, _ := nfiMappingPriority(definition)
		pack.Metadata["mapping_priority"] = strconv.Itoa(priority)
	}
	if dynamic {
		// Dynamic keys must survive projection, including keys first seen after
		// approval. The mapping stage validates their extraction and collisions.
		pack.Rules[0].ProjectionMode = parserpolicy.ProjectionModeAll
		pack.Rules[0].AllowedFields = nil
		pack.Rules[0].CountPolicy = parserpolicy.CountPolicyAllowDerived
	}
	if scopePattern != "" {
		pack.Rules[0].IncludePredicates = []parserpolicy.Predicate{{Type: "regex", Field: "input", Pattern: scopePattern}}
	}
	for index, item := range cases {
		if strings.TrimSpace(item.Raw) == "" || len(item.Desired) == 0 {
			continue
		}
		service := nfiCaseService(item)
		pack.RegressionCases = append(pack.RegressionCases, parserpolicy.RegressionCase{Name: fmt.Sprintf("nfi-reviewed-%03d", index+1), Input: item.Raw, Service: service, SourceType: "file", ExpectedPayload: item.Desired, ExpectedParser: id + "/" + id + "-rule", ExpectedRoute: "normal"})
	}
	if err := parserpolicy.SealParserPack(pack); err != nil {
		return nil, err
	}
	if result := parserpolicy.ValidateParserPack(pack); !result.Valid {
		return nil, fmt.Errorf("NFI runtime pack invalid: %s", strings.Join(result.Errors, "; "))
	}
	encoded, err := json.Marshal(pack)
	if err != nil {
		return nil, err
	}
	if _, err := parserpolicy.LoadParserPackBytes(encoded); err != nil {
		return nil, err
	}
	return pack, nil
}
