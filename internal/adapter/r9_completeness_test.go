package adapter

import (
	"context"
	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/fieldmapping"
	parserpolicy "nexora.local/nli-v20/parser-policy"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var r9MailSamples = []string{
	"20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed;reason=not needed",
	"20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed:reason=notneeded:server=cloud",
	"20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status:failed reason;notneeded server;cloud",
	"20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg |status:failed|reason:notneeded|server:cloud",
}

func TestR9BaselineMailCompletenessTrace(t *testing.T) {
	a, err := NewParserPackAdapter()
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range r9MailSamples {
		result := a.TryParseV5Production(raw, engine.Source{Service: "smtp"})
		if !result.Success || result.Quarantined {
			t.Fatalf("built-in complete mail pack failed: %+v", result.ParseTelemetry)
		}
		if result.EventPayload.Fields["status"] != "failed" || result.EventPayload.Fields["reason"] == "" {
			t.Fatalf("built-in parser discarded a source value: %+v", result.EventPayload.Fields)
		}
	}
}

func r9HeaderPlan() fieldmapping.Plan {
	p := fieldmapping.Plan{Version: 1}
	for i, name := range []string{"date", "time_timezone", "hostname", "service", "pid", "minor_pid", "thread_id"} {
		p.Fields = append(p.Fields, fieldmapping.Field{Name: name, Method: "position", Source: strconv.Itoa(i + 1)})
	}
	for i, name := range []string{"loglevel", "event"} {
		p.Fields = append(p.Fields, fieldmapping.Field{Name: name, Method: "delimiter", Source: "8", Delimiter: ";", Part: strconv.Itoa(i + 1), Confirmed: true})
	}
	return p
}

func TestR9ManagedHeaderOnlyQuarantinesAllFourReportedLines(t *testing.T) {
	plan := r9HeaderPlan()
	header := strings.Join(strings.Fields(r9MailSamples[0])[:8], " ")
	a, _ := sourceParityAdapter(t, header, plan)
	for i := range a.profiles {
		profile := &a.profiles[i]
		if !profile.Managed {
			continue
		}
		profile.Pack.Rules[0].ServiceSelectors = []string{"*"}
		if err := parserpolicy.SealParserPack(profile.Pack); err != nil {
			t.Fatal(err)
		}
		var err error
		profile.Executor, err = parserpolicy.NewParserPackExecutor(profile.Pack)
		if err != nil {
			t.Fatal(err)
		}
	}
	deps := engine.Default()
	deps.SetV5Parser(a)
	for _, raw := range r9MailSamples {
		result := a.TryParseV5Production(raw, engine.Source{Service: "nfi-fixture"})
		if result.Success {
			t.Fatalf("managed header-only emitted: %+v", result)
		}
		if !result.Quarantined {
			t.Fatal("managed-only completeness failure allowed generic fallback")
		}
		routed := engine.Parse(raw, engine.Source{Service: "smtp"}, deps)
		if parserpolicy.HasCompleteSMTPMailTail(raw) {
			if string(routed.Meta.Route) != "normal" || routed.Data["status"] != "failed" || routed.Data["reason"] == nil ||
				(strings.Contains(raw, "server") && routed.Data["server"] != "cloud") {
				t.Fatalf("complete fallback lost a source field: %+v", routed)
			}
			continue
		}
		if string(routed.Meta.Route) != "quarantine" {
			t.Fatalf("production route lost suffix: %+v", routed)
		}
	}
}

func TestR9DynamicTailProductionAndNewKeys(t *testing.T) {
	plan := r9HeaderPlan()
	header := strings.Join(strings.Fields(r9MailSamples[0])[:8], " ")
	a, _ := sourceParityAdapter(t, header, plan)
	plan.Fields = append(plan.Fields, fieldmapping.Field{Name: "tail_fields", Method: "kv_all", Source: "9", Type: "object", OutputPolicy: "flatten", KeyOptions: fieldmapping.KeyOptions{RequireComplete: true}})
	for i := range a.profiles {
		profile := &a.profiles[i]
		if profile.Pack.ID != "nfi-source-parity" {
			continue
		}
		pack := profile.Pack
		pack.Rules[0].Stages[0].Config = plan
		pack.Rules[0].ServiceSelectors = []string{"smtp-in"}
		pack.Rules[0].ProjectionMode = parserpolicy.ProjectionModeAll
		pack.Rules[0].CountPolicy = parserpolicy.CountPolicyAllowDerived
		pack.Rules[0].AllowedFields = nil
		compiled, err := fieldmapping.Compile(plan)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := compiled.Execute(context.Background(), r9MailSamples[0])
		if err != nil {
			t.Fatal(err)
		}
		pack.RegressionCases[0].Input = r9MailSamples[0]
		pack.RegressionCases[0].Service = "smtp-in"
		pack.RegressionCases[0].ExpectedPayload = expected.Values
		if err := parserpolicy.SealParserPack(pack); err != nil {
			t.Fatal(err)
		}
		profile.Executor, err = parserpolicy.NewParserPackExecutor(pack)
		if err != nil {
			t.Fatal(err)
		}
	}
	deps := engine.Default()
	deps.SetV5Parser(a)
	for i, raw := range r9MailSamples {
		if i < 2 {
			raw += ":future_key=0042"
		}
		r := engine.Parse(raw, engine.Source{Service: "smtp"}, deps)
		if i < 2 {
			if string(r.Meta.Route) == "quarantine" || r.Data["future_key"] != "0042" || r.Data["status"] != "failed" {
				t.Fatalf("dynamic payload lost: %+v", r)
			}
		} else {
			if string(r.Meta.Route) != "normal" || r.Data["status"] != "failed" || r.Data["reason"] != "notneeded" || r.Data["server"] != "cloud" {
				t.Fatalf("built-in mail pack lost the pipe tail: %+v", r)
			}
		}
	}
}

func TestReviewedComposedHeaderPreventsPartialFallback(t *testing.T) {
	baseline, err := NewParserPackAdapter()
	if err != nil {
		t.Fatal(err)
	}
	raw := r9MailSamples[0]
	source := engine.Source{Service: "smtp"}
	if result := baseline.TryParseV5Production(raw, source); !result.Success || result.Quarantined {
		t.Fatalf("expected eligible baseline candidate: %+v", result)
	}
	header := strings.Join(strings.Fields(raw)[:8], " ")
	plan := r9HeaderPlan()
	plan.Fields = append(plan.Fields, fieldmapping.Field{
		Name: "tail", Type: "object", Method: "kv_all", Policy: "optional",
		KVHeaderPattern: `^\d{8}\s+(?:\S+\s+){6}Note;SieveRuleMsg(?:[ \t]+|$)`,
		KVProfile:       "configured", KVLiteralBoundaries: true, PairDelimiters: ";",
		KeyOptions: fieldmapping.KeyOptions{AssignmentSeparators: []string{":"}, RequireComplete: true},
	})
	a, _ := sourceParityAdapter(t, header, plan)
	for i := range a.profiles {
		profile := &a.profiles[i]
		if !profile.Managed {
			continue
		}
		profile.Pack.Rules[0].ServiceSelectors = []string{"*"}
		if err := parserpolicy.SealParserPack(profile.Pack); err != nil {
			t.Fatal(err)
		}
		profile.Executor, err = parserpolicy.NewParserPackExecutor(profile.Pack)
		if err != nil {
			t.Fatal(err)
		}
	}
	if result := a.TryParseV5Production(header, source); !result.Success || result.Quarantined {
		t.Fatalf("composed header without a payload should parse: %+v", result)
	}
	if result := a.TryParseV5Production(raw, source); result.Success || !result.Quarantined {
		t.Fatalf("strict composed rejection allowed a legacy fallback: %+v", result)
	}
}

func TestNumericMailHeaderCannotNormalizeWithShiftedFields(t *testing.T) {
	const raw = "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 123456;Note;SieveRuleMsg status=failed;reason=not needed"
	plan := r9HeaderPlan()
	plan.Fields = append(plan.Fields,
		fieldmapping.Field{Name: "status", Method: "key_value", Source: "status", KVProfile: "configured", KVStart: "9", PairDelimiters: ";", KeyOptions: fieldmapping.KeyOptions{AssignmentSeparators: []string{"="}}},
		fieldmapping.Field{Name: "reason", Method: "key_value", Source: "reason", KVProfile: "configured", KVStart: "9", PairDelimiters: ";", KeyOptions: fieldmapping.KeyOptions{AssignmentSeparators: []string{"="}}},
	)
	legacy, _ := sourceParityAdapter(t, r9MailSamples[0], plan)
	source := engine.Source{Service: "nfi-fixture"}
	if result := legacy.TryParseV5Production(raw, source); result.Success || !result.Quarantined || !slices.Contains(result.ParseTelemetry.ErrorCodes, "incomplete_structured_header") {
		t.Fatalf("shifted legacy mapping escaped the structural check: %+v", result)
	}
	for i, name := range []string{"random_number", "loglevel", "event"} {
		field := fieldmapping.Field{Name: name, Method: "delimiter", Source: "8", Delimiter: ";", Part: strconv.Itoa(i + 1), Confirmed: true}
		if i < 2 {
			plan.Fields[7+i] = field
		} else {
			plan.Fields = append(plan.Fields, field)
		}
	}
	complete, _ := sourceParityAdapter(t, raw, plan)
	result := complete.TryParseV5Production(raw, source)
	if !result.Success || result.Quarantined || result.EventPayload.FieldCount() != 12 {
		t.Fatalf("complete numeric header should emit 12 semantic fields: %+v", result)
	}
	for name, want := range map[string]string{"random_number": "123456", "loglevel": "Note", "event": "SieveRuleMsg", "status": "failed", "reason": "not needed"} {
		if got, ok := result.EventPayload.Get(name); !ok || got != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
		for _, evidence := range result.ParseTelemetry.FieldEvidence {
			if evidence.Field == name && raw[evidence.Start:evidence.End] != want {
				t.Errorf("%s has wrong byte span %d–%d", name, evidence.Start, evidence.End)
			}
		}
	}
	deps := engine.Default()
	deps.SetV5Parser(legacy)
	if result := engine.Parse(raw, source, deps); string(result.Meta.Route) != "quarantine" {
		t.Fatalf("legacy fallback normalized structurally incomplete data: %+v", result)
	}
	plan.Fields[7].OutputPolicy = "omit"
	omission, _ := sourceParityAdapter(t, raw, plan)
	result = omission.TryParseV5Production(raw, source)
	if !result.Success || result.EventPayload.FieldCount() != 11 {
		t.Fatalf("an explicitly omitted exact source component should be accounted for: %+v", result)
	}
	if _, ok := result.EventPayload.Get("random_number"); ok {
		t.Fatal("an explicitly omitted source component entered output")
	}
}

func TestNumericSMTPHeaderBuiltInAndReviewedComposition(t *testing.T) {
	const header = "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 123456;Note;SieveRuleMsg"
	baseline, err := NewParserPackAdapter()
	if err != nil {
		t.Fatal(err)
	}
	source := engine.Source{Service: "auto"}
	for _, test := range []struct {
		tail       string
		wantFields int
	}{
		{wantFields: 10},
		{tail: "\r\n", wantFields: 10},
		{tail: " status=failed;reason=not needed", wantFields: 12},
		{tail: " status=failed;missing_assignment"},
		{tail: " status=failed;status=ok"},
	} {
		result := baseline.TryParseV5Production(header+test.tail, source)
		if test.wantFields == 0 {
			if result.Success || !result.Quarantined {
				t.Fatalf("malformed tail normalized: %+v", result.ParseTelemetry)
			}
			continue
		}
		if !result.Success || result.Quarantined || result.EventPayload.FieldCount() != test.wantFields || result.EventPayload.Fields["event"] != "SieveRuleMsg" {
			t.Fatalf("valid input did not produce %d exact fields: %+v", test.wantFields, result)
		}
	}
	pack := parserpolicy.DefaultCompleteSMTPMailPack()
	plan := pack.Rules[1].Stages[0].Config.(fieldmapping.Plan)
	plan.Fields[len(plan.Fields)-1].OutputPolicy = "include"
	managed, _ := sourceParityAdapter(t, header, plan)
	for i := range managed.profiles {
		profile := &managed.profiles[i]
		if !profile.Managed {
			continue
		}
		profile.Pack.Rules[0].ServiceSelectors = []string{"*"}
		if err := parserpolicy.SealParserPack(profile.Pack); err != nil {
			t.Fatal(err)
		}
		profile.Executor, err = parserpolicy.NewParserPackExecutor(profile.Pack)
		if err != nil {
			t.Fatal(err)
		}
	}
	result := managed.TryParseV5Production(header+" status=failed;reason=not needed", source)
	if !result.Success || result.ParseTelemetry.PackID != "nfi-source-parity" {
		t.Fatalf("reviewed template composition did not take precedence: %+v", result.ParseTelemetry)
	}
	for _, attempt := range result.ParseTelemetry.OrderedAttempts {
		if strings.HasPrefix(attempt.ParserID, parserpolicy.MailTailPackID+"/") {
			return
		}
	}
	t.Fatal("reviewed composition skipped the eligible embedded parser")
}
