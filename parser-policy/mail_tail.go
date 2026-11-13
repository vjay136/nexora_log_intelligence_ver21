package parserpolicy

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"nexora.local/nli-v20/internal/fieldmapping"
	"nexora.local/nli-v20/internal/mixedkv"
)

const MailTailPackID = "nli_reviewed_smtp_complete_tail"

// Scope the new parser to SMTP records with an explicit pipe frame or the
// reviewed continuation marker. Ordinary SMTP records keep their existing
// routing and typed schemas.
var completeMailHeader = regexp.MustCompile(`^\s*\d{8}\s+\d{9}[+-]\d{4}\s+\S+\s+smtp\s+\d+\s+\d+\s+\d+\s+[^;\s]+;\S+\s+`)
var continuedMailValue = regexp.MustCompile(`^[A-Z][A-Z0-9_]*=`)
var uncodedSieveHeader = regexp.MustCompile(`^\s*\d{8}\s+\d{9}[+-]\d{4}\s+\S+\s+smtp\s+\d+\s+\d+\s+\d+\s+[^;\s]+;SieveRuleMsg\s+`)
var dottedSieveTail = regexp.MustCompile(`^status=[^\s;,|]+\.reason;[^;,|]+,server:[^;,|]+$`)

func HasCompleteSMTPMailTail(input string) bool {
	if fieldmapping.HasMailHeaderExtension(input) {
		return true
	}
	header := completeMailHeader.FindStringIndex(input)
	if header == nil {
		return false
	}
	tail := input[header[1]:]
	return strings.HasPrefix(tail, "|") || hasMailContinuation(tail) || uncodedSieveHeader.MatchString(input)
}

// RequiresSMTPMailContinuation keeps compatibility layouts eligible for
// execution while preventing automatic fallback from merging a continued
// assignment back into its parent field.
func RequiresSMTPMailContinuation(input string) bool {
	header := completeMailHeader.FindStringIndex(input)
	return header != nil && hasMailContinuation(input[header[1]:])
}

func hasMailContinuation(tail string) bool {
	return strings.Contains(tail, ":msg=MISREPORT=") || strings.HasPrefix(tail, "msg=MISREPORT=")
}

func parseUncodedSieveTail(ctx context.Context, input string) (mixedkv.Result, error) {
	// This legacy variant uses a dot between exactly the status and reason.
	// Keep dots in other values and formats out of this narrow grammar.
	if dottedSieveTail.MatchString(input) {
		config := mixedkv.DefaultConfig()
		config.PairSeparators = []string{".", ","}
		config.AssignmentSeparators = []string{"=", ";", ":"}
		config.KeyAllowPatterns = []string{`^(status|reason|server)$`}
		return mixedkv.Parse(input, config)
	}
	profiles, err := fieldmapping.InspectScopedKeyValues(ctx, input, 0)
	if err != nil {
		return mixedkv.Result{}, err
	}
	if len(profiles) != 1 || len(profiles[0].Observations) < 2 {
		return mixedkv.Result{}, fmt.Errorf("no unique complete mail grammar")
	}
	result := mixedkv.Result{Fields: make(map[string]interface{}, len(profiles[0].Observations))}
	for _, observation := range profiles[0].Observations {
		name := mailOutputKey(observation.Key)
		if _, exists := result.Fields[name]; exists {
			return mixedkv.Result{}, fmt.Errorf("mail grammar contains duplicate keys")
		}
		span := mixedkv.Span{Start: observation.Start, End: observation.End}
		result.Fields[name] = observation.Value
		result.Entries = append(result.Entries, mixedkv.Entry{SourceKey: observation.Key, Key: name, Value: observation.Value, ValueSpan: span, PairSpan: span, Emitted: true})
	}
	return result, nil
}

func parseReviewedMailTail(input string, config mixedkv.Config, continuationName string) (mixedkv.Result, error) {
	if strings.HasPrefix(input, "|") {
		return parseReviewedPipe(input, config)
	}
	parser, err := mixedkv.Compile(config)
	if err != nil {
		return mixedkv.Result{}, err
	}
	result, err := parser.ParseWithPipeBoundaries(input)
	if err != nil || result.RemainderSpan != nil || continuationName == "" {
		return result, err
	}
	for i, entry := range result.Entries {
		if entry.SourceKey != "msg" {
			continue
		}
		raw := input[entry.ValueSpan.Start:entry.ValueSpan.End]
		marker := continuedMailValue.FindStringIndex(raw)
		if marker == nil {
			continue
		}
		if _, exists := result.Fields[continuationName]; exists || len(result.Entries) >= config.MaxFields || len(raw) == marker[1] {
			return mixedkv.Result{}, fmt.Errorf("mail continuation is empty or conflicts with a captured field")
		}
		if _, ok := result.Fields[entry.Key].(string); !ok {
			return mixedkv.Result{}, fmt.Errorf("repeated mail continuation is ambiguous")
		}
		split := entry.ValueSpan.Start + marker[1]
		prefix, suffix := strings.Clone(raw[:marker[1]-1]), strings.Clone(raw[marker[1]:])
		result.Entries[i].Value = prefix
		result.Entries[i].ValueSpan.End = split - 1
		result.Entries[i].PairSpan.End = split - 1
		result.Fields[entry.Key] = prefix
		result.Fields[continuationName] = suffix
		span := mixedkv.Span{Start: split, End: entry.ValueSpan.End}
		result.Entries = append(result.Entries, mixedkv.Entry{
			Key: continuationName, Value: suffix, ValueSpan: span, PairSpan: span, Emitted: true,
		})
	}
	return result, nil
}

func parseReviewedPipe(input string, config mixedkv.Config) (mixedkv.Result, error) {
	config.PairSeparators = []string{"|"}
	config.AssignmentSeparators = []string{"=>", ":=", "=", ":", ";"}
	parser, err := mixedkv.Compile(config)
	if err != nil {
		return mixedkv.Result{}, err
	}
	segments, err := parser.ScanPipeTail(input)
	if err != nil {
		return mixedkv.Result{}, err
	}
	reserved := make(map[string]bool, len(segments))
	for _, segment := range segments {
		reserved[mailOutputKey(segment.Key)] = true
	}
	result := mixedkv.Result{Fields: make(map[string]interface{}, len(segments))}
	next := 1
	for _, segment := range segments {
		name := mailOutputKey(segment.Key)
		for name == "" {
			name = fmt.Sprintf("field%d", next)
			next++
			if reserved[name] {
				name = ""
			}
		}
		switch previous := result.Fields[name].(type) {
		case nil:
			result.Fields[name] = segment.Text
		case string:
			result.Fields[name] = []interface{}{previous, segment.Text}
		case []interface{}:
			result.Fields[name] = append(previous, segment.Text)
		}
		result.Entries = append(result.Entries, mixedkv.Entry{
			SourceKey: segment.Key, Key: name, Value: segment.Text,
			ValueSpan: segment.Value, PairSpan: segment.Pair, Emitted: true,
		})
	}
	return result, nil
}

func mailOutputKey(key string) string {
	switch key {
	case "date", "time", "hostname", "service", "major_pid", "pid_status", "minor_pid", "event_type", "event", "code",
		"raw", "nli", "meta", "source", "envelope":
		return "input." + key
	}
	return key
}

// DefaultCompleteSMTPMailPack is shared by controller and agent runtimes.
func DefaultCompleteSMTPMailPack() *ParserPack {
	activated := time.Date(2026, time.November, 10, 0, 0, 0, 0, time.UTC)
	aliases := map[string]string{}
	for _, name := range []string{"date", "time", "hostname", "service", "major_pid", "pid_status", "minor_pid", "event_type", "event", "code", "raw", "nli", "meta", "source", "envelope"} {
		aliases[name] = mailOutputKey(name)
	}
	pack := &ParserPack{
		ID: MailTailPackID, Version: "22.13.0", Revision: 2,
		SourceType: "manual", Author: "nexora-parser-review-board", ReviewState: ReviewStateActivated,
		CreatedAt: activated, ActivatedAt: &activated,
		ResourceLimits: DefaultResourceLimits(),
		RegressionCases: []RegressionCase{{
			Name:    "smtp_mixed_continuation",
			Input:   "20250811 080053105+0100 mail001 smtp 3692 0 140026854221568 Note;SieveRuleMsg(2005/1) id=demo:msg=MISREPORT=eventType;misreport|sender;<demo@example.test>|action;keep",
			Service: "smtp", SourceType: "file",
			ExpectedParser: MailTailPackID + "/smtp_complete_tail", ExpectedRoute: "normal",
			ExpectedPayload: map[string]interface{}{
				"date": "20250811", "time": "080053105+0100", "hostname": "mail001", "service": "smtp",
				"major_pid": "3692", "pid_status": 0, "minor_pid": "140026854221568",
				"event_type": "Note", "event": "SieveRuleMsg", "code": "2005/1",
				"id": "demo", "msg": "MISREPORT", "tailed_field": "eventType;misreport",
				"sender": "<demo@example.test>", "action": "keep",
			},
		}, {
			Name:    "smtp_pipe_duplicate_and_positional",
			Input:   `20250812 130035121+0100 mail001 smtp 3609 0 140293780948736 Note;SmtpServerEHLO(2001/4) |DATA|sid=5f9e\|2c1a|hdr=X-Mailer:Example\, Build 1234|hdr=X-Origin:192.0.2.1, 192.0.2.2`,
			Service: "smtp", SourceType: "file",
			ExpectedParser: MailTailPackID + "/smtp_complete_tail", ExpectedRoute: "normal",
			ExpectedPayload: map[string]interface{}{
				"date": "20250812", "time": "130035121+0100", "hostname": "mail001", "service": "smtp",
				"major_pid": "3609", "pid_status": 0, "minor_pid": "140293780948736",
				"event_type": "Note", "event": "SmtpServerEHLO", "code": "2001/4",
				"field1": "DATA", "sid": "5f9e|2c1a", "hdr": []interface{}{`X-Mailer:Example\, Build 1234`, "X-Origin:192.0.2.1, 192.0.2.2"},
			},
		}, {
			Name:    "smtp_uncoded_adjacent_wrappers",
			Input:   "20250811 080053105+0100 mail001 smtp 3692 0 140026854221568 Note;SieveRuleMsg [status:failed][reason;notneeded][server=cloud]",
			Service: "smtp", SourceType: "file",
			ExpectedParser: MailTailPackID + "/smtp_complete_tail", ExpectedRoute: "normal",
			ExpectedPayload: map[string]interface{}{
				"date": "20250811", "time": "080053105+0100", "hostname": "mail001", "service": "smtp",
				"major_pid": "3692", "pid_status": 0, "minor_pid": "140026854221568",
				"event_type": "Note", "event": "SieveRuleMsg",
				"status": "failed", "reason": "notneeded", "server": "cloud",
			},
		}},
		Rules: []ParserRule{{
			ID: "smtp_complete_tail", Name: "Reviewed SMTP mail tail", Priority: 180,
			ServiceSelectors:  []string{"smtp", "smtp-in", "smtp-out"},
			IncludePredicates: []Predicate{{Type: "regex", Field: "input", Pattern: completeMailHeader.String() + `(?:\||(?:.*:)?msg=MISREPORT=)|` + uncodedSieveHeader.String()}},
			FailurePolicy:     FailurePolicyRejectRule,
			ProjectionMode:    ProjectionModeAll, CountPolicy: CountPolicyExactSource,
			Stages: []Stage{
				{ID: "mail_prefix", Name: "SMTP event prefix", Type: StageTypeRegex, FailurePolicy: FailurePolicyRejectRule,
					Config: regexStageConfig{Pattern: `(?s)^\s*(?P<date>\d{8})\s+(?P<time>\d{9}[+-]\d{4})\s+(?P<hostname>\S+)\s+(?P<service>smtp)\s+(?P<major_pid>\d+)\s+(?P<pid_status>\d+)\s+(?P<minor_pid>\d+)\s+(?P<event_type>[^;\s]+);(?P<event>[^()\s]+)(?:\((?P<code>[^()\s]+)\))?\s+(?P<_kv_tail>.+)$`}},
				{ID: "mail_tail", Name: "Exact mixed assignment tail", Type: StageTypeMixedKV, FailurePolicy: FailurePolicyRejectRule,
					Config: mixedKVStageConfig{
						InputField: "_kv_tail", MailTail: true, ContinuationField: "tailed_field",
						AssignmentSeparators: []string{"=", ";"}, PairSeparators: []string{":", "|"},
						BracketPairs:    map[string]string{"[": "]", "{": "}", "(": ")", "<": ">"},
						KeyAliases:      aliases,
						DuplicatePolicy: mixedkv.DuplicateError,
					},
				},
				{ID: "mail_types", Name: "SMTP process status type", Type: StageTypeTypeConversion, FailurePolicy: FailurePolicyRejectRule,
					Config: conversionStageConfig{Conversions: map[string]string{"pid_status": "int"}}},
			},
		}},
	}
	rule, examples := numericSMTPHeaderRule()
	pack.Rules = append(pack.Rules, rule)
	pack.RegressionCases = append(pack.RegressionCases, examples...)
	if err := SealParserPack(pack); err != nil {
		panic(err)
	}
	return pack
}

func numericSMTPHeaderRule() (ParserRule, []RegressionCase) {
	const pattern = `^[ \t]*(?P<date>[0-9]{8})[ \t]+(?P<time_timezone>[0-9]{9}[+-][0-9]{4})[ \t]+(?P<hostname>\S+)[ \t]+(?P<service>smtp)[ \t]+(?P<pid>[0-9]+)[ \t]+(?P<minor_pid>[0-9]+)[ \t]+(?P<thread_id>[0-9]+)[ \t]+(?P<random_number>[0-9]+);(?P<loglevel>[^;\s]+);(?P<event>[^;()\s]+)(?:\((?P<event_code>[^()\s]+)\))?(?:[ \t]+|[\r\n]*$)`
	const ruleID = "smtp_numeric_prefix"
	plan := fieldmapping.Plan{Version: 1}
	for _, name := range []string{"date", "time_timezone", "hostname", "service", "pid", "minor_pid", "thread_id", "random_number", "loglevel", "event"} {
		typeName := "string"
		if name == "pid" || name == "minor_pid" {
			typeName = "integer"
		}
		plan.Fields = append(plan.Fields, fieldmapping.Field{Name: name, Type: typeName, Method: "regex", Source: pattern, Part: name, Policy: "required"})
	}
	plan.Fields = append(plan.Fields, fieldmapping.Field{Name: "event_code", Type: "string", Method: "regex", Source: pattern, Part: "event_code", Policy: "optional"})
	plan.Fields = append(plan.Fields, fieldmapping.Field{
		Name: "kv_pair", Type: "object", Method: "kv_all", Policy: "optional", OutputPolicy: "flatten",
		KVHeaderPattern: pattern, KVProfile: "configured", KVLiteralBoundaries: true,
		KVPairSeparators: []string{";"}, DuplicatePolicy: "reject",
		KeyOptions: fieldmapping.KeyOptions{AssignmentSeparators: []string{"="}, RequireComplete: true},
	})
	rule := ParserRule{
		ID: ruleID, Name: "Reviewed SMTP numeric event prefix", Priority: 181,
		ServiceSelectors:  []string{"smtp", "smtp-in", "smtp-out"},
		IncludePredicates: []Predicate{{Type: "regex", Field: "input", Pattern: pattern}},
		FailurePolicy:     FailurePolicyRejectRule, ProjectionMode: ProjectionModeAll, CountPolicy: CountPolicyProjection,
		Stages: []Stage{{ID: "map_numeric_prefix", Name: "Exact SMTP header and semicolon KV tail", Type: StageTypeMapping, Config: plan, FailurePolicy: FailurePolicyRejectRule}},
	}
	const header = "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 123456;Note;SieveRuleMsg"
	output := func() map[string]interface{} {
		return map[string]interface{}{
			"date": "20250811", "time_timezone": "080053105+0100", "hostname": "aaa-gmailrgi009", "service": "smtp",
			"pid": 3692, "minor_pid": 0, "thread_id": "140026854221568", "random_number": "123456", "loglevel": "Note", "event": "SieveRuleMsg",
		}
	}
	withTail := output()
	withTail["status"] = "failed"
	withTail["reason"] = "not needed"
	return rule, []RegressionCase{
		{Name: "smtp_numeric_prefix_only", Input: header, Service: "smtp", SourceType: "file", ExpectedParser: MailTailPackID + "/" + ruleID, ExpectedRoute: "normal", ExpectedPayload: output()},
		{Name: "smtp_numeric_prefix_and_tail", Input: header + " status=failed;reason=not needed", Service: "smtp", SourceType: "file", ExpectedParser: MailTailPackID + "/" + ruleID, ExpectedRoute: "normal", ExpectedPayload: withTail},
	}
}
