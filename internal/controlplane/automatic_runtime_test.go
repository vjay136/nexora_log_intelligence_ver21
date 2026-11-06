package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"nexora.local/nli-v20/internal/adapter"
	"nexora.local/nli-v20/internal/engine"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"nexora.local/nli-v20/internal/fleet"
	metastore "nexora.local/nli-v20/internal/store"
	parserpolicy "nexora.local/nli-v20/parser-policy"
)

type automaticWriteCounter struct {
	metastore.Store
	writes atomic.Int64
}

func (s *automaticWriteCounter) Put(ctx context.Context, doc metastore.Document) (metastore.Document, error) {
	s.writes.Add(1)
	return s.Store.Put(ctx, doc)
}
func (s *automaticWriteCounter) Delete(ctx context.Context, collection, id string, purge bool, actor string) error {
	s.writes.Add(1)
	return s.Store.Delete(ctx, collection, id, purge, actor)
}

func TestAutomaticPublishedRuntimeRefreshAndReadOnlySuggestions(t *testing.T) {
	f := newAPIFixture(t)
	counter := &automaticWriteCounter{Store: f.server.metadata}
	f.server.metadata = counter
	f.server.fleet.Register(&fleet.Node{ID: "automatic-node", Hostname: "automatic-node"})
	shared := f.server.assuranceDeps.V5Parser
	definition, desired := nfiExampleDefinition()
	cases := []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: desired}}
	response, body := f.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{"id": "automatic-managed", "name": "Automatic managed sample", "version": "1.0.0", "parser": definition, "cases": cases})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("save: %d %s", response.StatusCode, body)
	}
	before := f.server.runAssurance(assuranceRequest{Raw: nfiExample, Service: "auto"})
	if strings.Contains(before["parser"].(map[string]any)["name"].(string), "automatic-managed") {
		t.Fatal("draft entered automatic pool")
	}
	for _, action := range []string{"submit", "approve", "publish"} {
		response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/automatic-managed/"+action, map[string]any{"reason": "Administrator checked exact runtime output"})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d %s", action, response.StatusCode, body)
		}
	}
	after := f.server.runAssurance(assuranceRequest{Raw: nfiExample, Service: "auto", Desired: desired})
	if after["passed"] != true || after["parser"].(map[string]any)["name"] != "automatic-managed/automatic-managed-rule" {
		t.Fatalf("published parser absent from automatic Assurance: %+v", after)
	}
	first, err := f.server.automaticParserRuntime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.server.automaticParserRuntime(context.Background())
	if err != nil || first != second {
		t.Fatal("unchanged bundle not cached", err)
	}
	beforeSuggestionWrites := counter.writes.Load()
	response, body = f.request(http.MethodPost, "/api/v1/field-mapping/suggest", map[string]any{"samples": []string{nfiExample}, "service": "smtp"})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("suggest: %d %s", response.StatusCode, body)
	}
	suggestions := decodeMap(t, body)
	runtime := suggestions["runtime"].(map[string]any)
	if runtime["available"] != true || len(runtime["managed_profiles"].([]any)) != 1 || len(runtime["builtin_profiles"].([]any)) == 0 {
		t.Fatalf("missing complete pool evidence: %s", body)
	}
	match := runtime["matches"].([]any)[0].(map[string]any)
	if match["parser_id"] != "automatic-managed/automatic-managed-rule" || match["attempt_count"].(float64) < 2 {
		t.Fatalf("new and built-in attempts not both considered: %s", body)
	}
	if len(suggestions["runtime_field_suggestions"].([]any)) == 0 {
		t.Fatalf("observed exact runtime fields absent: %s", body)
	}
	response, wildcardBody := f.request(http.MethodPost, "/api/v1/field-mapping/suggest", map[string]any{"samples": []string{nfiExample}, "service": " * "})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("wildcard scope analysis: %d %s", response.StatusCode, wildcardBody)
	}
	wildcardRuntime := decodeMap(t, wildcardBody)["runtime"].(map[string]any)
	wildcardMatch := wildcardRuntime["matches"].([]any)[0].(map[string]any)
	if wildcardMatch["parser_id"] != "automatic-managed/automatic-managed-rule" || wildcardMatch["attempt_count"].(float64) < 2 {
		t.Fatalf("wildcard mapping scope suppressed family candidates: %s", wildcardBody)
	}
	if counter.writes.Load() != beforeSuggestionWrites {
		t.Fatalf("read-only discovery persisted %d changes", counter.writes.Load())
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/automatic-managed/rollback", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("rollback: %d %s", response.StatusCode, body)
	}
	restored := f.server.runAssurance(assuranceRequest{Raw: nfiExample, Service: "auto"})
	if strings.Contains(restored["parser"].(map[string]any)["name"].(string), "automatic-managed") {
		t.Fatal("rolled-back parser remained cached")
	}
	if f.server.assuranceDeps.V5Parser != shared {
		t.Fatal("automatic analysis mutated shared engine dependencies")
	}
}

func TestAutomaticRuntimeInvalidIsolationCancellationAndManualSelection(t *testing.T) {
	f := newAPIFixture(t)
	_, err := f.server.metadata.Put(context.Background(), metastore.Document{Collection: "parser-packs", ID: "broken-active", State: "published", Payload: []byte(`{"runtime_pack":{"id":"broken-active"}}`), CreatedBy: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := f.server.automaticParserRuntime(context.Background())
	if err != nil || snapshot.parser == nil || len(snapshot.builtinIDs) == 0 {
		t.Fatal("invalid add-on disabled baseline", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.server.automaticParserRuntime(ctx); err == nil {
		t.Fatal("canceled runtime compile succeeded")
	}
	manual := f.server.runAssurance(assuranceRequest{Raw: "alpha=one", Service: "auto", ParserID: "generic.kv"})
	if manual["parser"].(map[string]any)["name"] != "generic.kv" || manual["runtime_pool"] != nil {
		t.Fatalf("explicit manual selector changed: %+v", manual)
	}
}

func TestAutomaticRuntimeToleratesIdenticalAliasesAndIsolatesConflicts(t *testing.T) {
	f := newAPIFixture(t)
	definition, desired := nfiExampleDefinition()
	cases := []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: desired}}
	good, err := assuranceMappingRuntimePack("automatic-good", "1.0.0", "admin", definition, cases)
	if err != nil {
		t.Fatal(err)
	}
	conflict, err := assuranceMappingRuntimePack("automatic-conflict", "1.0.0", "admin", definition, cases)
	if err != nil {
		t.Fatal(err)
	}
	put := func(id string, pack *parserpolicy.ParserPack) {
		t.Helper()
		payload, err := json.Marshal(map[string]any{"runtime_pack": pack, "catalog_id": id, "runtime_id": pack.ID})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.server.metadata.Put(context.Background(), metastore.Document{Collection: "parser-packs", ID: id, State: "published", Payload: payload, CreatedBy: "admin"}); err != nil {
			t.Fatal(err)
		}
	}
	put("good-alias-a", good)
	put("good-alias-b", good)
	put("conflict-alias-a", conflict)
	changed := *conflict
	changed.Version = "1.0.1"
	if err := parserpolicy.SealParserPack(&changed); err != nil {
		t.Fatal(err)
	}
	put("conflict-alias-b", &changed)
	snapshot, err := f.server.automaticParserRuntime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.managedIDs) != 1 || snapshot.managedIDs[0] != "automatic-good" || len(snapshot.builtinIDs) == 0 {
		t.Fatalf("conflict removed unrelated valid pool: %+v", snapshot.managedIDs)
	}
	if !strings.Contains(strings.Join(snapshot.diagnostics, " "), "Conflicting published runtime identity") {
		t.Fatal("conflicting identity not diagnosed")
	}
	result := f.server.runAssurance(assuranceRequest{Raw: nfiExample, Service: "smtp", Desired: desired})
	if result["passed"] != true || result["parser"].(map[string]any)["name"] != "automatic-good/automatic-good-rule" {
		t.Fatalf("unrelated good parser not selected: %+v", result)
	}
	summary := automaticRuntimeSummary(snapshot, nil)
	if summary["pool_hash"] == nil || summary["bundle_hash"] != nil {
		t.Fatal("tolerant analysis identity misrepresented as acknowledged fleet bundle")
	}
	if _, err := f.server.buildActiveParserBundle(context.Background()); err == nil {
		t.Fatal("strict canonical fleet duplicate protection was weakened")
	}
}

func TestAutomaticRuntimePreserveSourceMatchesClientAndLegacyCanonicalization(t *testing.T) {
	f := newAPIFixture(t)
	raw := " A\r\nB  "
	desired := map[string]any{"leading": " ", "line_boundary": "\r\n", "trailing": "  "}
	definition := map[string]any{"pattern_type": "mapping", "service_scope": "source-preserve-fixture", "mapping_plan": map[string]any{"version": 1, "fields": []any{
		map[string]any{"name": "leading", "method": "character", "source": "1", "source_end": "1"},
		map[string]any{"name": "line_boundary", "method": "byte", "source": "3", "source_end": "4"},
		map[string]any{"name": "trailing", "method": "character", "source": "6", "source_end": "7"},
	}}}
	pack, err := assuranceMappingRuntimePack("source-preserve", "1.0.0", "admin", definition, []assuranceRequest{{Raw: raw, Service: "source-preserve-fixture", Desired: desired}})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"runtime_pack": pack})
	if _, err := f.server.metadata.Put(context.Background(), metastore.Document{Collection: "parser-packs", ID: pack.ID, State: "published", Payload: payload, CreatedBy: "admin"}); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, pack.ID+".json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	client, err := adapter.NewParserPackAdapterFromDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	parsed := client.TryParseV5Production(raw, engine.Source{Service: "source-preserve-fixture"})
	if !parsed.Success {
		t.Fatal("actual client fixture failed")
	}
	response, body := f.request(http.MethodPost, "/api/v1/assurance/run", map[string]any{"raw": raw, "service": "source-preserve-fixture", "desired": desired, "preserve_source": true})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("preserved source HTTP: %d %s", response.StatusCode, body)
	}
	result := decodeMap(t, body)
	if result["passed"] != true || result["raw"] != raw {
		t.Fatalf("source mutated before automatic parser: %s", body)
	}
	if differences, _ := structuralDiff(result["actual"].(map[string]any), parsed.EventPayload.Fields); len(differences) != 0 {
		t.Fatalf("client/controller exact-source drift: %+v", differences)
	}
	legacy := f.server.runAssurance(assuranceRequest{Raw: raw, Service: "source-preserve-fixture"})
	if legacy["raw"] != normalizeAssuranceRaw(raw) {
		t.Fatal("legacy default canonicalization changed")
	}
}

func TestAutomaticRuntimeProtectedBuiltinIDCannotBeCreated(t *testing.T) {
	f := newAPIFixture(t)
	definition, desired := nfiExampleDefinition()
	id := parserpolicy.DefaultSMTPParserPack().ID
	response, body := f.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{"id": id, "name": "Forbidden built-in replacement", "version": "1.0.0", "parser": definition, "cases": []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: desired}}})
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("protected built-in identity accepted: %d %s", response.StatusCode, body)
	}
	if _, err := f.server.metadata.Get(context.Background(), "parser-packs", id, 0); err == nil {
		t.Fatal("rejected built-in override persisted a draft")
	}
}
