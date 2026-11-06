package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	runtimeadapter "nexora.local/nli-v20/internal/adapter"
	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/fleet"
)

// This acceptance crosses authoritative controller preview, exact candidate
// validation, immutable metadata, administrator review, sealed bundles and the
// actual agent engine. It does not substitute a browser-only transformation.
func TestFieldSenseAdvancedPlanLifecycleProductionParity(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "fieldsense-node", Hostname: "fieldsense-node"})
	raw := "account=Alice@example.test trace=0042 status=failed:reason=notneeded blocks=first[one]second[two]"
	digest := sha256.Sum256([]byte("0042"))
	desired := map[string]any{"contact": "alice@example.test", "trace_fingerprint": hex.EncodeToString(digest[:]), "selected_labels": map[string]any{"status": "failed", "reason": "notneeded"}, "second_block": "two", "summary": "failed"}
	plan := map[string]any{"version": 1, "fields": []any{
		map[string]any{"name": "contact", "method": "key_value", "source": "account", "type": "email", "transform": "lowercase"},
		map[string]any{"name": "trace_fingerprint", "method": "key_value", "source": "trace", "type": "string", "operations": []any{map[string]any{"op": "hash", "algorithm": "sha256"}}},
		map[string]any{"name": "selected_labels", "method": "kv_all", "type": "object", "key_include": []string{"status", "reason"}},
		map[string]any{"name": "second_block", "method": "between", "boundary_start": "[", "boundary_end": "]", "occurrence": "nth", "occurrence_index": 2, "preserve_boundary": false, "type": "string"},
		map[string]any{"name": "temporary_status", "method": "key_value", "source": "status", "output_policy": "intermediate"},
		map[string]any{"name": "summary", "method": "field", "source": "temporary_status", "operations": []any{map[string]any{"op": "conditional_merge", "fields": []string{"temporary_status"}, "condition_field": "temporary_status", "condition_operator": "equals", "condition_value": "failed", "else_value": "ok", "separator": " / "}}},
	}}
	definition := map[string]any{"pattern_type": "mapping", "mapping_plan": plan, "service_scope": "fieldsense-fixture"}
	cases := []assuranceRequest{{Raw: raw, Service: "fieldsense-fixture", Desired: desired}}
	response, body := f.request(http.MethodPost, "/api/v1/assurance/mapping-preview", map[string]any{"raw": raw, "plan": plan})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["passed"] != true {
		t.Fatalf("advanced preview: %d %s", response.StatusCode, body)
	}
	if diff, _ := structuralDiff(decodeMap(t, body)["actual"].(map[string]any), desired); len(diff) != 0 {
		t.Fatalf("advanced native mismatch: %+v", diff)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parser-preview", map[string]any{"parser": definition, "cases": cases})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["passed"] != true {
		t.Fatalf("exact candidate: %d %s", response.StatusCode, body)
	}
	before, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{"id": "fieldsense-advanced", "name": "Advanced operator mapping", "version": "1.0.0", "parser": definition, "cases": cases})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("save: %d %s", response.StatusCode, body)
	}
	draftBundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil || draftBundle.BundleHash != before.BundleHash {
		t.Fatal("unapproved draft changed active bundle", err)
	}
	for _, action := range []string{"submit", "approve", "publish"} {
		response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/fieldsense-advanced/"+action, map[string]any{"reason": "Administrator checked exact FieldSense advanced output."})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d %s", action, response.StatusCode, body)
		}
	}
	bundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	installed := false
	for _, asset := range bundle.Assets {
		if asset.AssetClass != "parser-packs" {
			continue
		}
		digest := sha256.Sum256(asset.Payload)
		if hex.EncodeToString(digest[:]) != asset.SHA256 {
			t.Fatal("bundle bytes differ from sealed digest")
		}
		if err := os.WriteFile(filepath.Join(directory, asset.AssetID+".json"), asset.Payload, 0600); err != nil {
			t.Fatal(err)
		}
		if asset.AssetID == "fieldsense-advanced" {
			installed = true
		}
	}
	if !installed {
		t.Fatal("published advanced pack missing from canonical fleet bundle")
	}
	adapter, err := runtimeadapter.NewParserPackAdapterFromDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	deps := engine.Default()
	deps.SetV5Parser(adapter)
	actual := engine.Parse(raw, engine.Source{Service: "fieldsense-fixture"}, deps)
	serialized, err := json.Marshal(actual.Data)
	if err != nil {
		t.Fatal(err)
	}
	output := map[string]any{}
	if err := json.Unmarshal(serialized, &output); err != nil {
		t.Fatal(err)
	}
	if diff, _ := structuralDiff(output, desired); len(diff) != 0 {
		t.Fatalf("actual agent differs from controller: %+v parser=%s", diff, actual.NLI.Parser)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/fieldsense-advanced/rollback", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("rollback: %d %s", response.StatusCode, body)
	}
	rolledBack, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil || rolledBack.BundleHash != before.BundleHash {
		t.Fatal("rollback failed to restore prior active bundle", err)
	}
}

func TestFieldSenseRepairClonePreservesActiveUntilApprovedAndRollsBack(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "fieldsense-repair-node", Hostname: "fieldsense-repair-node"})
	definition, desired := nfiExampleDefinition()
	cases := []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: desired}}
	response, body := f.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{"id": "fieldsense-live", "name": "Original live mapping", "version": "1.0.0", "parser": definition, "cases": cases})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("original draft: %d %s", response.StatusCode, body)
	}
	for _, action := range []string{"submit", "approve", "publish"} {
		response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/fieldsense-live/"+action, map[string]any{"reason": "Administrator checked original."})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("original %s: %d %s", action, response.StatusCode, body)
		}
	}
	original, err := f.server.metadata.Get(context.Background(), "parser-packs", "fieldsense-live", 0)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{"id": "fieldsense-live", "name": "Unapproved overwrite", "version": "2.0.0", "parser": definition, "cases": cases})
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("existing id overwritten: %d %s", response.StatusCode, body)
	}
	response, automaticBody := f.request(http.MethodPost, "/api/v1/assurance/parsers/fieldsense-live/clone-draft", map[string]any{"reason": "Automatic repair identifier"})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("automatic clone: %d %s", response.StatusCode, automaticBody)
	}
	automaticID, _ := decodeMap(t, automaticBody)["id"].(string)
	if automaticID == "" || safeID(automaticID) != automaticID || len(automaticID) > 80 {
		t.Fatalf("invalid generated repair id: %q", automaticID)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/fieldsense-live/clone-draft", map[string]any{"id": "fieldsense-repair", "name": "Explicit operator repair", "version": "1.1.0", "reason": "Rename/normalize event without changing current active source"})
	if response.StatusCode != http.StatusCreated || decodeMap(t, body)["original_active_preserved"] != true {
		t.Fatalf("clone: %d %s", response.StatusCode, body)
	}
	afterClone, err := f.server.metadata.Get(context.Background(), "parser-packs", "fieldsense-live", 0)
	if err != nil || afterClone.Revision != original.Revision || afterClone.Hash != original.Hash {
		t.Fatal("cloning modified original active document", err)
	}
	draft, err := f.server.metadata.Get(context.Background(), "parser-packs", "fieldsense-repair", 0)
	if err != nil {
		t.Fatal(err)
	}
	draftBody := map[string]any{}
	if err := json.Unmarshal(draft.Payload, &draftBody); err != nil {
		t.Fatal(err)
	}
	if draft.State != "draft" || draftBody["validation_passed"] != false || draftBody["activated"] == true {
		t.Fatalf("clone inherited validation/activation: %+v", draftBody)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/fieldsense-repair/submit", map[string]any{})
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("clone submitted without revalidation: %d %s", response.StatusCode, body)
	}
	unchanged, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil || unchanged.BundleHash != baseline.BundleHash {
		t.Fatal("repair draft changed prior active bundle", err)
	}
	repaired := draftBody["parser"].(map[string]any)
	fields := repaired["mapping_plan"].(map[string]any)["fields"].([]any)
	for _, item := range fields {
		field := item.(map[string]any)
		if field["name"] == "event" {
			field["transform"] = "uppercase"
		}
	}
	repairedDesired := map[string]any{}
	for key, value := range desired {
		repairedDesired[key] = value
	}
	repairedDesired["event"] = "SIEVERULEMSG"
	repairedCases := []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: repairedDesired}}
	response, body = f.request(http.MethodPut, "/api/v1/assurance/parsers/fieldsense-repair", map[string]any{"name": "Explicit operator repair", "version": "1.1.0", "parser": repaired, "cases": repairedCases})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("edit clone: %d %s", response.StatusCode, body)
	}
	for _, action := range []string{"validate", "submit", "approve", "publish"} {
		response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/fieldsense-repair/"+action, map[string]any{"reason": "Administrator validated changed exact output."})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("repair %s: %d %s", action, response.StatusCode, body)
		}
	}
	published, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	for _, asset := range published.Assets {
		if asset.AssetClass == "parser-packs" {
			if err := os.WriteFile(filepath.Join(directory, asset.AssetID+".json"), asset.Payload, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	adapter, err := runtimeadapter.NewParserPackAdapterFromDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	deps := engine.Default()
	deps.SetV5Parser(adapter)
	parsed := engine.Parse(nfiExample, engine.Source{Service: "smtp"}, deps)
	if parsed.Data["event"] != "SIEVERULEMSG" || parsed.NLI.Parser != "fieldsense-repair/fieldsense-repair-rule" {
		t.Fatalf("approved repair not selected: %s %+v", parsed.NLI.Parser, parsed.Data)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/fieldsense-repair/rollback", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("repair rollback: %d %s", response.StatusCode, body)
	}
	restored, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil || restored.BundleHash != baseline.BundleHash {
		t.Fatal("repair rollback did not restore old bundle", err)
	}
	preserved, err := f.server.metadata.Get(context.Background(), "parser-packs", "fieldsense-live", 0)
	if err != nil || preserved.Revision != original.Revision || preserved.Hash != original.Hash {
		t.Fatal("repair lifecycle mutated old parser", err)
	}
}

func TestFieldSenseMappingPriorityIsBoundedAndExplicit(t *testing.T) {
	for _, invalid := range []any{-1, 1001, 1.5, "1", true, []any{1}} {
		if _, err := nfiMappingPriority(map[string]any{"mapping_priority": invalid}); err == nil {
			t.Fatalf("invalid mapping priority accepted: %+v", invalid)
		}
	}
	for _, valid := range []int{0, 1, 1000} {
		if actual, err := nfiMappingPriority(map[string]any{"mapping_priority": valid}); err != nil || actual != valid {
			t.Fatalf("valid mapping priority rejected: %d %d %v", valid, actual, err)
		}
	}
}
