package adapter

import (
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/fieldmapping"
	parserpolicy "nexora.local/nli-v20/parser-policy"
)

const additiveSMTP = "20250812 130034198+0100 mail001 smtp 3609 0 140290550523648 Note;SmtpServerConnect(2001/3) id=6864BD5E047384B0:host=11.22.33.44:connection_count=1:local_port=2525"

func TestAutomaticReviewedMappingDoesNotShortCircuitExistingParsers(t *testing.T) {
	adapter, _ := sourceParityAdapter(t, additiveSMTP, fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "custom_message", Method: "raw"}}})
	for index := range adapter.profiles {
		profile := &adapter.profiles[index]
		if !profile.Managed {
			continue
		}
		profile.Pack.Rules[0].ServiceSelectors = []string{"smtp-in"}
		profile.Pack.RegressionCases[0].Service = "smtp-in"
		if err := parserpolicy.SealParserPack(profile.Pack); err != nil {
			t.Fatal(err)
		}
		executor, err := parserpolicy.NewParserPackExecutor(profile.Pack)
		if err != nil {
			t.Fatal(err)
		}
		profile.Executor = executor
	}
	result := adapter.TryParseV5Production(additiveSMTP, engine.Source{Service: "smtp-in"})
	if !result.Success || !result.ReviewedMapping {
		t.Fatalf("reviewed mapping lost its approved output choice: %+v", result)
	}
	attempted := map[string]bool{}
	for _, attempt := range result.ParseTelemetry.OrderedAttempts {
		attempted[attempt.ParserID] = true
	}
	for _, id := range []string{"smtp_native_v5", "nli_reviewed_smtp_in/reviewed_smtp_in", "nli_reviewed_smtp_out", "nfi-source-parity/mapping"} {
		found := attempted[id]
		for parser := range attempted {
			if strings.HasPrefix(parser, id+"/") {
				found = true
			}
		}
		if !found {
			t.Fatalf("existing/add-on parser omitted from automatic pool: %s: %+v", id, attempted)
		}
	}
	if result.EventPayload.FieldCount() != 1 {
		t.Fatal("candidate evaluation emitted multiple customer results")
	}
}

func TestAutomaticWildcardAddOnRunsAlongsideSuccessfulExistingParser(t *testing.T) {
	directory := t.TempDir()
	writeWildcardRuntimePack(t, directory, "add-on-wildcard")
	adapter, err := NewParserPackAdapterFromDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	result := adapter.TryParseV5Production(additiveSMTP, engine.Source{Service: "smtp-in"})
	if !result.Success || result.ParseTelemetry.ParserID != "nli_reviewed_smtp_in/reviewed_smtp_in" {
		t.Fatalf("add-on changed existing successful format: %+v", result)
	}
	found := false
	for _, attempt := range result.ParseTelemetry.OrderedAttempts {
		if strings.HasPrefix(attempt.ParserID, "add-on-wildcard/") {
			found = true
		}
	}
	if !found {
		t.Fatal("wildcard add-on was skipped after embedded success")
	}
	if len(result.ParseTelemetry.ErrorCodes) != 0 {
		t.Fatal("failed add-on contaminated successful embedded parser health")
	}
}

func TestAutomaticAddOnCannotSuppressExistingJSONFallback(t *testing.T) {
	directory := t.TempDir()
	writeWildcardRuntimePack(t, directory, "add-on-wildcard")
	adapter, err := NewParserPackAdapterFromDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	deps := engine.Default()
	deps.SetV5Parser(adapter)
	for _, message := range []string{"unchanged", "status=ok"} {
		raw := `{"message":"` + message + `","original":true}`
		result := engine.Parse(raw, engine.Source{Service: "auto"}, deps)
		if result.Data["message"] != message || result.Data["original"] != true || string(result.Meta.Route) != "normal" {
			t.Fatalf("add-on suppressed structural JSON: %+v", result)
		}
		if result.Meta.ParserAttempts < 1 {
			t.Fatal("add-on not considered before preserved JSON fallback")
		}
	}
}
