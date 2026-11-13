package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/fieldmapping"
	metastore "nexora.local/nli-v20/internal/store"
	parserpolicy "nexora.local/nli-v20/parser-policy"
)

func TestTemplateComposeExactRevisionsAndRuntimeParity(t *testing.T) {
	f := newAPIFixture(t)
	ctx := context.Background()
	const header = `^(?P<date>[0-9]{8})\s+(?P<timestamp>\S+)\s+(?P<hostname>\S+)\s+(?P<service>\S+)\s+(?P<pid>\d+)\s+(?P<minor_pid>\d+)\s+(?P<thread_id>\d+)\s+(?:(?P<random_number>\d+);(?:(?P<extra_tag1>[^;\s]+);)?(?:(?P<extra_tag2>[^;\s]+);)?)?(?P<event_type>[^;\s]+);(?P<event>\S+)(?:[ \t]+|$)`
	capture := map[string]string{}
	for _, field := range []string{"date", "timestamp", "hostname", "service", "pid", "minor_pid", "thread_id", "event_type", "event", "random_number", "extra_tag1", "extra_tag2"} {
		capture[field] = field
	}
	save := func(collection, id string, config any) templateComposeRef {
		t.Helper()
		payload, err := json.Marshal(map[string]any{"config": config})
		if err != nil {
			t.Fatal(err)
		}
		doc, err := f.server.metadata.Put(ctx, metastore.Document{Collection: collection, ID: id, State: "active", Payload: payload, CreatedBy: "admin"})
		if err != nil {
			t.Fatal(err)
		}
		return templateComposeRef{ID: doc.ID, Revision: doc.Revision}
	}
	headerRef := save("templates", "test-header", map[string]any{"engine": "re2", "pattern": header, "fields": capture, "optional_fields": []string{"random_number", "extra_tag1", "extra_tag2"}})
	kvRef := save("kv-config", "test-kv", map[string]any{
		"pair_separators": []string{":", ";"}, "assignment_separators": []string{"="},
		"bare_field": "messages", "max_fields": 256,
	})
	save("kv-config", "test-kv", map[string]any{"pair_separators": []string{"|"}, "assignment_separators": []string{"="}})
	smtp := "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg"
	card := "20260401 134601279+0100 server001 pabd 250410 0 139764638705408 Note;CARDDAV"
	samples := []string{smtp, smtp + " status=failed;reason=not needed",
		card + " id=002fac3e:HTTP CHALLENGE REQUEST:method=1:realm=mx:address=11.22.33.44",
		smtp + " status=failed;invalid key=hidden",
		strings.Replace(smtp, "Note;", "123456;abcd;efgh;Note;", 1) + " status=failed;reason=not needed",
	}
	response, data := f.request(http.MethodPost, "/api/v1/field-mapping/template-compose", map[string]any{
		"header": headerRef, "kv": kvRef, "samples": samples,
	})
	body := decodeMap(t, data)
	if response.StatusCode != http.StatusOK || body["total"] != float64(5) || body["failed"] != float64(1) || body["persisted"] != false || body["activated"] != false {
		t.Fatalf("compose status %d: %s", response.StatusCode, data)
	}
	if got := body["failures"].([]any)[0].(map[string]any)["line"]; got != float64(4) {
		t.Fatalf("wrong failed line: %v", got)
	}
	examples := body["examples"].([]any)
	if header := examples[0].(map[string]any)["actual"].(map[string]any); len(header) != 9 || header["event"] != "SieveRuleMsg" {
		t.Fatalf("header-only record returned unexpected values: %v", header)
	}
	if variant := examples[3].(map[string]any)["actual"].(map[string]any); variant["extra_tag1"] != "abcd" || variant["extra_tag2"] != "efgh" || variant["random_number"] != "123456" || variant["event"] != "SieveRuleMsg" {
		t.Fatalf("variant mismatch: %v", variant)
	}
	actual := examples[2].(map[string]any)["actual"].(map[string]any)
	kv := actual["kv_pair"].(map[string]any)
	if kv["address"] != "11.22.33.44" || kv["messages"].([]any)[0] != "HTTP CHALLENGE REQUEST" {
		t.Fatalf("wrong CARDDAV values: %v", actual)
	}
	cases := []assuranceRequest{{Raw: samples[1], Service: "smtp"}, {Raw: samples[2], Service: "pabd"}, {Raw: samples[4], Service: "smtp"}}
	pack, err := assuranceMappingRuntimePack("test-compose", "1.0.0", "test", map[string]any{
		"mapping_plan": body["plan"], "scope_pattern": body["scope_pattern"],
	}, cases)
	if err != nil {
		t.Fatal(err)
	}
	if pack.Metadata["minimum_client_release"] != taggedMailMinimumRelease {
		t.Fatalf("wrong runtime client compatibility: %v", pack.Metadata)
	}
	executor, err := parserpolicy.NewParserPackExecutor(pack)
	if err != nil {
		t.Fatal(err)
	}
	output, err := executor.Execute(&parserpolicy.ExecutionRequest{Input: samples[2], Service: "pabd", Context: ctx})
	if err != nil || output == nil || output.Quarantined || output.EventPayload == nil {
		t.Fatalf("native pack failed: %+v, %v", output, err)
	}
	if diff, _ := structuralDiff(output.EventPayload.Fields, actual); len(diff) != 0 {
		t.Fatalf("native pack disagrees with composed preview: %v", diff)
	}
	tagged, err := executor.Execute(&parserpolicy.ExecutionRequest{Input: samples[4], Service: nfiCaseService(cases[2]), Context: ctx})
	if err != nil || tagged == nil || tagged.Quarantined || tagged.EventPayload == nil {
		t.Fatalf("tagged composition failed at runtime: %+v, %v", tagged, err)
	}
	if diff, _ := structuralDiff(tagged.EventPayload.Fields, examples[3].(map[string]any)["actual"].(map[string]any)); len(diff) != 0 {
		t.Fatalf("tagged composition diverged from preview: %v", diff)
	}
	encoded, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	bundle := parserAssetBundle{Assets: []parserBundleAsset{
		{AssetClass: "parser-packs", Payload: json.RawMessage(`{"rules":[{"stages":[{"type":"mapping","config":{"fields":[{"kv_input_edits":[{}]}]}}]}]}`)},
		{AssetClass: "parser-packs", Payload: encoded},
	}}
	if minimum := parserBundleMinimumKVRelease(bundle); minimum != taggedMailMinimumRelease || parserClientSupportsKVRelease("22.10.0", minimum) {
		t.Fatalf("advanced bundle should exclude prior clients: minimum %q", minimum)
	}
}

func TestTemplateComposeCannotHideUncapturedKVBehindGreedyPrefix(t *testing.T) {
	plan, err := fieldmapping.Compile(fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{
		{Name: "category", Type: "string", Method: "regex", Source: `^(?P<category>system).*$`, Part: "category"},
		{Name: "kv_pair", Type: "object", Method: "kv_all", Policy: "optional", KVProfile: "configured",
			KVHeaderPattern: `^(?P<category>system).*$`, KVLiteralBoundaries: true, PairDelimiters: ";",
			KeyOptions: fieldmapping.KeyOptions{RequireComplete: true}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	raw := "system status=failed;reason=missing"
	result, err := plan.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.ValidateComplete(raw, result); err == nil || !strings.Contains(err.Error(), "no mapped") {
		t.Fatalf("unmapped tail was accepted: %v", err)
	}
}
