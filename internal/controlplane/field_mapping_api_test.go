package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	runtimeadapter "nexora.local/nli-v20/internal/adapter"
	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/fleet"
	parserpolicy "nexora.local/nli-v20/parser-policy"
)

const nfiExample = "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed:reason=notneeded:server=cloud"

func nfiExampleDefinition() (map[string]any, map[string]any) {
	fields := []any{}
	names := []string{"date", "time_timezone", "hostname", "service", "pid", "minor_pid", "thread_id"}
	for i, name := range names {
		fields = append(fields, map[string]any{"name": name, "type": "string", "policy": "required", "method": "position", "source": string(rune('1' + i))})
	}
	fields = append(fields,
		map[string]any{"name": "loglevel", "type": "string", "policy": "required", "method": "delimiter", "source": "8", "delimiter": ";", "part": "1", "confirmed": true},
		map[string]any{"name": "event", "type": "string", "policy": "required", "method": "delimiter", "source": "8", "delimiter": ";", "part": "2", "confirmed": true})
	for _, key := range []string{"status", "reason", "server"} {
		fields = append(fields, map[string]any{"name": key, "type": "string", "policy": "required", "method": "key_value", "source": key, "duplicate_policy": "reject"})
	}
	return map[string]any{"pattern_type": "mapping", "pattern": ".*", "mapping_plan": map[string]any{"version": 1, "fields": fields}}, map[string]any{
		"date": "20250811", "time_timezone": "080053105+0100", "hostname": "aaa-gmailrgi009", "service": "smtp", "pid": "3692", "minor_pid": "0", "thread_id": "140026854221568", "loglevel": "Note", "event": "SieveRuleMsg", "status": "failed", "reason": "notneeded", "server": "cloud"}
}

func TestNFIExamplePreviewApprovalPublicationAndProductionParity(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "nfi-origin", Hostname: "nfi-origin"})
	definition, desired := nfiExampleDefinition()
	response, body := f.request(http.MethodPost, "/api/v1/assurance/mapping-preview", map[string]any{"raw": nfiExample, "plan": definition["mapping_plan"]})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["passed"] != true {
		t.Fatalf("mapping preview: %d %s", response.StatusCode, body)
	}
	actual := decodeMap(t, body)["actual"].(map[string]any)
	if diff, _ := structuralDiff(actual, desired); len(diff) != 0 {
		t.Fatalf("mapping output: %#v", diff)
	}
	cases := []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: desired}}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parser-preview", map[string]any{"parser": definition, "cases": cases})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["passed"] != true {
		t.Fatalf("candidate preview: %d %s", response.StatusCode, body)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{"id": "nfi-example", "name": "NFI example", "version": "1.0.0", "parser": definition, "cases": cases})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("draft: %d %s", response.StatusCode, body)
	}
	for _, action := range []string{"submit", "approve", "publish"} {
		response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/nfi-example/"+action, map[string]any{"reason": "Administrator reviewed the exact native field mapping."})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d %s", action, response.StatusCode, body)
		}
	}
	doc, err := f.server.metadata.Get(context.Background(), "parser-packs", "nfi-example", 0)
	if err != nil {
		t.Fatal(err)
	}
	pack, encoded, err := runtimeParserPackForDocument(doc)
	if err != nil {
		t.Fatal(err)
	}
	if pack.Rules[0].Stages[0].Type != parserpolicy.StageType("mapping") {
		t.Fatal("native plan was replaced by a different runtime representation")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, pack.ID+".json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	adapter, err := runtimeadapter.NewParserPackAdapterFromDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	result := adapter.TryParseV5Production(nfiExample, engine.Source{Hostname: "nfi-origin", Service: "smtp", File: "/logs/custom.log"})
	if !result.Success || result.Quarantined {
		t.Fatalf("production adapter failed: %+v", result)
	}
	// Marshal through the production output boundary to compare exact JSON
	// semantics rather than registry/editor-only values.
	payload, err := json.Marshal(result.EventPayload.Fields)
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]any
	if err := json.Unmarshal(payload, &output); err != nil {
		t.Fatal(err)
	}
	if diff, _ := structuralDiff(output, desired); len(diff) != 0 {
		t.Fatalf("production differs from native preview: %#v", diff)
	}
	response, body = f.request(http.MethodGet, "/api/v1/field-mapping", nil)
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["profile_count"] != float64(1) {
		t.Fatalf("NFI status: %d %s", response.StatusCode, body)
	}
}

func TestNFIValidationFailsClosedAndPreservesExplicitOperations(t *testing.T) {
	f := newAPIFixture(t)
	plan := map[string]any{"version": 1, "fields": []any{map[string]any{"name": "value", "method": "constant", "type": "string", "policy": "required", "default_value": "$(this is literal text)", "operations": []any{map[string]any{"op": "replace", "find": "literal", "replace": "configured"}}}}}
	response, body := f.request(http.MethodPost, "/api/v1/assurance/mapping-preview", map[string]any{"raw": "evidence", "plan": plan})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["actual"].(map[string]any)["value"] != "$(this is configured text)" {
		t.Fatalf("literal operation: %d %s", response.StatusCode, body)
	}
	plan["fields"].([]any)[0].(map[string]any)["operations"] = []any{map[string]any{"op": "exec", "value": "anything"}}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/mapping-preview", map[string]any{"raw": "evidence", "plan": plan})
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("unapproved operation accepted: %d %s", response.StatusCode, body)
	}
	plan["fields"].([]any)[0].(map[string]any)["operations"] = []any{}
	plan["unknown_executable_config"] = true
	response, body = f.request(http.MethodPost, "/api/v1/assurance/mapping-preview", map[string]any{"raw": "evidence", "plan": plan})
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown executable configuration accepted: %d %s", response.StatusCode, body)
	}
}
