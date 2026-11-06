package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimeadapter "nexora.local/nli-v20/internal/adapter"
	"nexora.local/nli-v20/internal/controlclient"
	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/familyprofile"
	"nexora.local/nli-v20/internal/fleet"
	"nexora.local/nli-v20/internal/middleware"
	metastore "nexora.local/nli-v20/internal/store"
)

type lifecycleExecutor struct {
	restarts  int
	parserDir string
	runtime   *runtimeadapter.ParserPackAdapter
}

func (e *lifecycleExecutor) RestartAgent(context.Context) error {
	e.restarts++
	runtime, err := runtimeadapter.NewParserPackAdapterFromDirectory(e.parserDir)
	if err != nil {
		return err
	}
	e.runtime = runtime
	return nil
}
func (e *lifecycleExecutor) Replay(context.Context, string, []string) (string, error) { return "", nil }
func (e *lifecycleExecutor) UpgradePackage(context.Context, string) error             { return nil }

func TestR9CatalogDeletionUnloadsActualClientAndRestoreStaysDraft(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "lifecycle-client", Hostname: "lifecycle-client"})
	if err := f.server.fleet.SetToken("lifecycle-client", "lifecycle-client-token"); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	tokenPath := filepath.Join(root, "node.token")
	idPath := filepath.Join(root, "install.id")
	parserDir := filepath.Join(root, "parsers")
	_ = os.WriteFile(tokenPath, []byte("lifecycle-client-token"), 0600)
	_ = os.WriteFile(idPath, []byte("lifecycle-client"), 0600)
	executor := &lifecycleExecutor{parserDir: parserDir}
	client, err := controlclient.New(controlclient.Config{ControllerURL: f.http.URL, AllowInsecureHTTP: true, NodeTokenFile: tokenPath, InstallIDFile: idPath, JournalFile: filepath.Join(root, "journal.json"), AgentConfigPath: filepath.Join(root, "agent.json"), ParserDir: parserDir, AssetDir: filepath.Join(root, "assets"), PackageSpoolDir: filepath.Join(root, "packages"), MetricsURL: "http://127.0.0.1:1/metrics", PollSeconds: 5}, executor)
	if err != nil {
		t.Fatal(err)
	}
	definition, desired := nfiExampleDefinition()
	response, body := f.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{"id": "lifecycle-pack", "name": "Lifecycle sample", "version": "1.0.0", "parser": definition, "cases": []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: desired}}})
	if response.StatusCode != 201 {
		t.Fatalf("create %d %s", response.StatusCode, body)
	}
	for _, action := range []string{"submit", "approve", "publish"} {
		response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/lifecycle-pack/"+action, map[string]any{"reason": "Administrator verified all parser fields"})
		if response.StatusCode != 200 {
			t.Fatalf("%s %d %s", action, response.StatusCode, body)
		}
	}
	published := decodeMap(t, body)
	initialHash := published["bundle_hash"].(string)
	approve := func(proposal map[string]any) {
		t.Helper()
		response, body := f.request(http.MethodPost, "/api/v1/rollouts/"+proposal["id"].(string)+"/approve", map[string]any{"reason": "Administrator reviewed exact parser synchronization"})
		if response.StatusCode != 200 {
			t.Fatalf("rollout approve %d %s", response.StatusCode, body)
		}
	}
	approve(published["rollout_proposal"].(map[string]any))
	if err := client.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	parserPath := filepath.Join(parserDir, "lifecycle-pack.json")
	if _, err := os.Stat(parserPath); err != nil {
		t.Fatal("published parser not installed", err)
	}
	parsed := executor.runtime.TryParseV5Production(nfiExample, engine.Source{Service: "smtp"})
	if !parsed.Success || parsed.ParseTelemetry == nil || !strings.Contains(parsed.ParseTelemetry.ParserID, "lifecycle-pack/") {
		t.Fatalf("installed client runtime did not use reviewed managed parser: %+v", parsed)
	}

	response, body = f.request(http.MethodPost, "/api/v1/parser-sync/verify", map[string]any{"node_ids": []string{"lifecycle-client"}})
	verified := decodeMap(t, body)
	if response.StatusCode != 200 || verified["client_acknowledgements"].([]any)[0].(map[string]any)["exact_acknowledgement"] != true {
		t.Fatalf("actual client apply not acknowledged %s", body)
	}
	originalBytes, err := os.ReadFile(parserPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parserPath, append(append([]byte{}, originalBytes...), byte(' ')), 0640); err != nil {
		t.Fatal(err)
	}
	if err := client.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	response, body = f.request(http.MethodPost, "/api/v1/parser-sync/verify", map[string]any{"node_ids": []string{"lifecycle-client"}})
	if decodeMap(t, body)["client_acknowledgements"].([]any)[0].(map[string]any)["exact_acknowledgement"] != false {
		t.Fatal("modified parser bytes falsely acknowledged current bundle")
	}
	if err := os.WriteFile(parserPath, originalBytes, 0640); err != nil {
		t.Fatal(err)
	}
	if err := client.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	response, body = f.request(http.MethodDelete, "/api/v1/parser-packs/lifecycle-pack", map[string]any{})
	if response.StatusCode != 200 {
		t.Fatalf("delete %d %s", response.StatusCode, body)
	}
	removed := decodeMap(t, body)
	if removed["bundle_hash"] == initialHash || removed["client_sync_status"] != "pending_approval" {
		t.Fatalf("removal not governed %s", body)
	}
	// The UI must distinguish controller removal from real client acknowledgement.
	response, body = f.request(http.MethodPost, "/api/v1/parser-sync/verify", map[string]any{"node_ids": []string{"lifecycle-client"}})
	pending := decodeMap(t, body)["client_acknowledgements"].([]any)[0].(map[string]any)
	if pending["exact_acknowledgement"] != false || pending["blocking_code"] == "" {
		t.Fatalf("stale client falsely aligned %s", body)
	}
	approve(removed["rollout_proposal"].(map[string]any))
	if err := client.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(parserPath); !os.IsNotExist(err) {
		t.Fatal("deleted parser still installed", err)
	}
	if executor.restarts != 2 {
		t.Fatalf("runtime was not restarted for both generations: %d", executor.restarts)
	}
	parsed = executor.runtime.TryParseV5Production(nfiExample, engine.Source{Service: "smtp"})
	if parsed.ParseTelemetry != nil && strings.Contains(parsed.ParseTelemetry.ParserID, "lifecycle-pack/") {
		t.Fatalf("deleted client parser still won runtime dispatch: %+v", parsed)
	}
	profiles, err := familyprofile.Resolve(parserDir)
	if err != nil || len(profiles) == 0 {
		t.Fatal("embedded baseline disappeared after removal", err)
	}
	for _, profile := range profiles {
		if profile.Managed {
			t.Fatalf("deleted managed pack remained in actual client runtime: %s", profile.Pack.ID)
		}
	}

	response, body = f.request(http.MethodGet, "/api/v1/parser-sync/runtime", nil)
	inventory := decodeMap(t, body)
	clients := inventory["clients"].([]any)
	if response.StatusCode != 200 || clients[0].(map[string]any)["exact_acknowledgement"] != true || clients[0].(map[string]any)["runtime_inventory"] == nil {
		t.Fatalf("missing actual client runtime inventory %s", body)
	}
	if _, err := os.Stat(filepath.Join(f.state, "parser-assets", "active-parsers", "lifecycle-pack.json")); !os.IsNotExist(err) {
		t.Fatal("stale controller projection remains", err)
	}
	after := f.server.runAssurance(assuranceRequest{Raw: nfiExample, Service: "auto"})
	if strings.Contains(after["parser"].(map[string]any)["name"].(string), "lifecycle-pack") {
		t.Fatal("deleted parser remained in controller automatic runtime")
	}
	response, body = f.request(http.MethodGet, "/api/v1/parser-packs/deleted", nil)
	if response.StatusCode != 200 || decodeMap(t, body)["count"] != float64(1) {
		t.Fatalf("missing restore history %s", body)
	}
	response, body = f.request(http.MethodPost, "/api/v1/parser-packs/deleted/lifecycle-pack/restore", map[string]any{"id": "lifecycle-restored", "reason": "Repair previous field mapping"})
	if response.StatusCode != 201 {
		t.Fatalf("restore %d %s", response.StatusCode, body)
	}
	restored, err := f.server.metadata.Get(context.Background(), "parser-packs", "lifecycle-restored", 0)
	if err != nil || restored.State != "draft" {
		t.Fatal("restore activated parser", err)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/lifecycle-restored/submit", map[string]any{})
	if response.StatusCode != 409 {
		t.Fatalf("restore bypassed fresh validation %d %s", response.StatusCode, body)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/lifecycle-restored/validate", map[string]any{})
	if response.StatusCode != 200 {
		t.Fatalf("retained cases cannot revalidate restored draft %d %s", response.StatusCode, body)
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/lifecycle-restored/submit", map[string]any{})
	if response.StatusCode != 200 {
		t.Fatalf("restored draft cannot enter normal review %d %s", response.StatusCode, body)
	}

	bundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range bundle.Assets {
		if asset.AssetClass == "parser-packs" && asset.AssetID == "lifecycle-restored" {
			t.Fatal("restored draft entered published bundle")
		}
	}
}

func TestR9DeleteLegacyAliasesRemovesSingleRuntimeIdentity(t *testing.T) {
	f := newAPIFixture(t)
	definition, desired := nfiExampleDefinition()
	pack, err := assuranceMappingRuntimePack("alias-runtime", "1.0.0", "admin", definition, []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: desired}})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"alias-one", "alias-two"} {
		payload, _ := json.Marshal(map[string]any{"runtime_pack": pack, "catalog_id": id, "runtime_id": pack.ID})
		_, err = f.server.metadata.Put(context.Background(), metastore.Document{Collection: "parser-packs", ID: id, State: "published", Payload: payload, CreatedBy: "admin"})
		if err != nil {
			t.Fatal(err)
		}
	}
	response, body := f.request(http.MethodDelete, "/api/v1/parser-packs/alias-one", map[string]any{})
	if response.StatusCode != 200 {
		t.Fatalf("delete aliases %d %s", response.StatusCode, body)
	}
	if len(decodeMap(t, body)["removed_catalog_ids"].([]any)) != 2 {
		t.Fatalf("alias remained active %s", body)
	}
	bundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range bundle.Assets {
		if asset.AssetClass == "parser-packs" && asset.AssetID == "alias-runtime" {
			t.Fatal("legacy alias resurrected deleted runtime")
		}
	}
}

// cancelDeleteStore models a remote RPC that commits its tombstone, then loses
// the response as the HTTP caller disconnects. Restoration rejects canceled
// contexts just as a real RPC transport does.
type cancelDeleteStore struct {
	metastore.Store
	cancel      context.CancelFunc
	failRestore bool
	attempted   bool
	restored    bool
}

func (s *cancelDeleteStore) Delete(ctx context.Context, collection, id string, purge bool, actor string) error {
	if err := s.Store.Delete(ctx, collection, id, purge, actor); err != nil {
		return err
	}
	s.attempted = true
	s.cancel()
	return context.Canceled
}
func (s *cancelDeleteStore) Put(ctx context.Context, doc metastore.Document) (metastore.Document, error) {
	if err := ctx.Err(); err != nil {
		return metastore.Document{}, err
	}
	if s.attempted && doc.Collection == "parser-packs" {
		if s.failRestore {
			return metastore.Document{}, errors.New("simulated restore storage failure")
		}
		s.restored = true
	}
	return s.Store.Put(ctx, doc)
}
func (s *cancelDeleteStore) List(ctx context.Context, collection string, query metastore.Query) ([]metastore.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Store.List(ctx, collection, query)
}

func TestR9DeletionCompensatesCanceledRemoteMutationAndReportsFailure(t *testing.T) {
	for _, failRestore := range []bool{false, true} {
		t.Run(fmt.Sprint(failRestore), func(t *testing.T) {
			f := newAPIFixture(t)
			definition, desired := nfiExampleDefinition()
			pack, err := assuranceMappingRuntimePack("canceled-pack", "1.0.0", "admin", definition, []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: desired}})
			if err != nil {
				t.Fatal(err)
			}
			payload, _ := json.Marshal(map[string]any{"runtime_pack": pack})
			doc, err := f.server.metadata.Put(context.Background(), metastore.Document{Collection: "parser-packs", ID: "canceled-pack", State: "published", Payload: payload, CreatedBy: "admin"})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wrapped := &cancelDeleteStore{Store: f.server.metadata, cancel: cancel, failRestore: failRestore}
			f.server.metadata = wrapped
			request := httptest.NewRequest(http.MethodDelete, "/api/v1/parser-packs/canceled-pack", strings.NewReader(`{}`)).WithContext(ctx)
			response := httptest.NewRecorder()
			f.server.deleteParserDocument(response, request, &middleware.SecurityContext{Username: "admin", Role: "admin", Permissions: []string{middleware.PermissionTriggerRollout}}, doc)
			if response.Code != 503 {
				t.Fatalf("ambiguous committed delete should fail with recovery state: %d %s", response.Code, response.Body.String())
			}
			if failRestore {
				if !strings.Contains(response.Body.String(), "parser_delete_rollback_failed") {
					t.Fatalf("rollback failure diagnostic missing: %s", response.Body.String())
				}
				return
			}
			restored, err := wrapped.Store.Get(context.Background(), "parser-packs", "canceled-pack", 0)
			if err != nil || restored.State != "published" || !wrapped.restored {
				t.Fatalf("canceled request prevented independent restoration: %v %+v", err, restored)
			}
			bundle, err := f.server.buildActiveParserBundle(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, asset := range bundle.Assets {
				if asset.AssetClass == "parser-packs" && asset.AssetID == "canceled-pack" {
					found = true
				}
			}
			if !found {
				t.Fatal("restored parser absent from canonical runtime bundle")
			}
		})
	}
}
