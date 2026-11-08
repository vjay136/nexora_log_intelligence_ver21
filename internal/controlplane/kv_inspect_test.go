package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/adapter"
	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/fleet"
)

func TestKVInputEditsPreviewAndClientCompatibility(t *testing.T) {
	f := newAPIFixture(t)
	definition, desired := nfiExampleDefinition()
	raw := strings.Split(nfiExample, "status=")[0] + "[status:failed][reason;notneeded][server=cloud]"
	plan := definition["mapping_plan"].(map[string]any)
	tail := map[string]any{
		"name": "tail", "method": "kv_all", "type": "object", "source": "9",
		"end_of_line": true, "output_policy": "flatten", "kv_profile": "configured",
		"require_complete": true, "assignment_separators": []string{":", ";", "="},
		"kv_pair_separators": []string{"|"},
		"kv_input_edits": []map[string]string{
			{"action": "replace", "find": "][", "replace": "|"},
			{"action": "strip_prefix", "find": "["},
			{"action": "strip_suffix", "find": "]"},
		},
	}
	plan["fields"] = append(plan["fields"].([]any)[:9:9], tail)
	response, body := f.request(http.MethodPost, "/api/v1/assurance/mapping-preview", map[string]any{"raw": raw, "plan": plan})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("preview: %d %s", response.StatusCode, body)
	}
	preview := decodeMap(t, body)
	if preview["passed"] != true {
		t.Fatalf("preview failed: %s", body)
	}
	actual, ok := preview["actual"].(map[string]any)
	if !ok {
		t.Fatalf("preview is missing mapped fields: %s", body)
	}
	if diff, _ := structuralDiff(actual, desired); len(diff) != 0 {
		t.Fatalf("preview differs from expected output: %v", diff)
	}
	pack, err := assuranceMappingRuntimePack("kv-input-edits", "1.0.0", "admin", definition,
		[]assuranceRequest{{Raw: raw, Service: "smtp", Desired: desired}})
	if err != nil {
		t.Fatal(err)
	}
	if pack.Metadata["minimum_client_release"] != kvInputEditsMinimumRelease {
		t.Fatalf("candidate metadata = %v", pack.Metadata)
	}
	delete(pack.Metadata, "minimum_client_release")
	payload, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	bundle := parserAssetBundle{Assets: []parserBundleAsset{{AssetClass: "parser-packs", Payload: payload}}}
	if minimum := parserBundleMinimumKVRelease(bundle); minimum != kvInputEditsMinimumRelease {
		t.Fatalf("imported bundle release floor = %q", minimum)
	}
	f.server.fleet.Register(&fleet.Node{ID: "old-kv-client", Hostname: "old-kv-client", Version: "22.1.0"})
	_, err = f.server.proposeParserBundleRollout(context.Background(), "admin", "KV edits", "test compatibility", "selected", bundle, []string{"old-kv-client"})
	if err == nil || !strings.Contains(err.Error(), kvInputEditsMinimumRelease) {
		t.Fatalf("old client proposal: %v", err)
	}
	bundle.Assets[0].Payload = json.RawMessage(`{"rules":[{"stages":[{"type":"mapping","config":{"fields":[{"key_value_edits":[{"key":"status","action":"blank"}]}]}}]}]}`)
	minimum := parserBundleMinimumKVRelease(bundle)
	if minimum != kvOutputEditsMinimumRelease || !parserClientSupportsKVRelease("22.1.0", minimum) {
		t.Fatalf("old output-only edits release floor = %q", minimum)
	}
}

func TestKVIEInspectAndPublishOnePackForBothReportedTails(t *testing.T) {
	f := newAPIFixture(t)
	definition, desired := nfiExampleDefinition()
	prefix := strings.Split(nfiExample, "status=")[0]
	samples := []string{prefix + "|status:failed|reason:notneeded|server:cloud", prefix + "status:failed reason;notneeded server;cloud"}
	tail := map[string]any{"name": "tail", "method": "kv_all", "type": "object", "source": "9", "unit": "token", "end_of_line": true, "output_policy": "flatten", "kv_profile": "auto", "require_complete": true}
	plan := definition["mapping_plan"].(map[string]any)
	plan["fields"] = append(plan["fields"].([]any)[:9:9], tail)
	var cases []assuranceRequest
	for _, raw := range samples {
		response, body := f.request(http.MethodPost, "/api/v1/assurance/kv-inspect", map[string]any{"raw": raw, "field": tail})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("inspect: %d %s", response.StatusCode, body)
		}
		result := decodeMap(t, body)
		profiles, ok := result["profiles"].([]any)
		if !ok || len(profiles) == 0 || result["persisted"] != false || result["activated"] != false {
			t.Fatalf("inspection not a read-only match: %s", body)
		}
		response, body = f.request(http.MethodPost, "/api/v1/assurance/mapping-preview", map[string]any{"raw": raw, "plan": plan})
		if response.StatusCode != http.StatusOK || decodeMap(t, body)["passed"] != true {
			t.Fatalf("preview: %d %s", response.StatusCode, body)
		}
		cases = append(cases, assuranceRequest{Raw: raw, Service: "smtp", Desired: desired})
	}
	response, body := f.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{"id": "kvie-example", "name": "KVIE example", "version": "1.0.0", "parser": definition, "cases": cases})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("draft: %d %s", response.StatusCode, body)
	}
	for _, action := range []string{"submit", "approve", "publish"} {
		response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/kvie-example/"+action, map[string]any{"reason": "Administrator reviewed both exact source formats."})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d %s", action, response.StatusCode, body)
		}
	}
	doc, err := f.server.metadata.Get(context.Background(), "parser-packs", "kvie-example", 0)
	if err != nil {
		t.Fatal(err)
	}
	pack, encoded, err := runtimeParserPackForDocument(doc)
	if err != nil {
		t.Fatal(err)
	}
	if pack.Metadata["minimum_client_release"] != "21.0.0-10" {
		t.Fatalf("missing R10 runtime requirement: %v", pack.Metadata)
	}
	dir := t.TempDir()
	if err = os.WriteFile(filepath.Join(dir, pack.ID+".json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	runtime, err := adapter.NewParserPackAdapterFromDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range samples {
		result := runtime.TryParseV5Production(raw, engine.Source{Service: "smtp", File: "/logs/custom.log"})
		if !result.Success || result.Quarantined {
			t.Fatalf("production failed: %+v", result)
		}
		payload, err := json.Marshal(result.EventPayload.Fields)
		if err != nil {
			t.Fatal(err)
		}
		if diff, _ := structuralDiff(decodeMap(t, payload), desired); len(diff) != 0 {
			t.Fatalf("production differs from validated output: %v", diff)
		}
	}
	bad := prefix + `|status:"unfinished|reason:notneeded|server:cloud`
	result := runtime.TryParseV5Production(bad, engine.Source{Service: "smtp", File: "/logs/custom.log"})
	if result.Success || !result.Quarantined {
		t.Fatalf("malformed tail escaped quarantine: %+v", result)
	}
}
