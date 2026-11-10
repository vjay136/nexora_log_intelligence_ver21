package adapter

import (
	"context"
	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/fieldmapping"
	parserpolicy "nexora.local/nli-v20/parser-policy"
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
