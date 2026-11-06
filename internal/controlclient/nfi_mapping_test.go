package controlclient

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	runtimeadapter "nexora.local/nli-v20/internal/adapter"
	"nexora.local/nli-v20/internal/canonicaljson"
	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/familyprofile"
	"nexora.local/nli-v20/internal/fieldmapping"
	"nexora.local/nli-v20/parser-policy"
)

func TestNFIReviewedMappingSynchronizesAndExecutesOnClient(t *testing.T) {
	root := t.TempDir()
	serviceExecutor := &fakeExecutor{}
	client, err := New(Config{ControllerURL: "http://127.0.0.1:12345", AllowInsecureHTTP: true, NodeTokenFile: filepath.Join(root, "node.token"), InstallIDFile: filepath.Join(root, "install.id"), JournalFile: filepath.Join(root, "journal.json"), AgentConfigPath: filepath.Join(root, "nli.json"), ParserDir: filepath.Join(root, "parsers"), AssetDir: filepath.Join(root, "parser-assets"), PackageSpoolDir: filepath.Join(root, "packages"), PollSeconds: 5}, serviceExecutor)
	if err != nil {
		t.Fatal(err)
	}
	raw := "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed:reason=notneeded:server=cloud"
	plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "pid", Method: "position", Source: "5", Type: "integer"}, {Name: "loglevel", Method: "delimiter", Source: "8", Delimiter: ";", Part: "1", Confirmed: true}, {Name: "event", Method: "delimiter", Source: "8", Delimiter: ";", Part: "2", Confirmed: true}, {Name: "status", Method: "key_value", Source: "status"}, {Name: "reason", Method: "key_value", Source: "reason"}, {Name: "server", Method: "key_value", Source: "server"}}}
	compiled, err := fieldmapping.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := compiled.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	pack := &parserpolicy.ParserPack{ID: "nfi-client-fixture", Version: "1.0.0", Revision: 1, SourceType: "manual", Author: "administrator", ReviewState: parserpolicy.ReviewStateApproved, CreatedAt: time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC), ResourceLimits: parserpolicy.DefaultResourceLimits(), Metadata: map[string]string{"mapping_engine": "nfi-v1", "selection_policy": "reviewed_mapping"}, Rules: []parserpolicy.ParserRule{{ID: "nfi-rule", Name: "Reviewed NFI mapping", Priority: 1, ServiceSelectors: []string{"*"}, Stages: []parserpolicy.Stage{{ID: "nfi-stage", Name: "Native Field Intelligence", Type: parserpolicy.StageTypeMapping, Config: plan, FailurePolicy: parserpolicy.FailurePolicyRejectRule}}, FailurePolicy: parserpolicy.FailurePolicyQuarantine, ProjectionMode: parserpolicy.ProjectionModeAllowlist, AllowedFields: []string{"pid", "loglevel", "event", "status", "reason", "server"}, CountPolicy: parserpolicy.CountPolicyProjection}}, RegressionCases: []parserpolicy.RegressionCase{{Name: "reviewed-mail", Input: raw, Service: "smtp", SourceType: "file", ExpectedPayload: preview.Values, ExpectedParser: "nfi-client-fixture/nfi-rule", ExpectedRoute: "normal"}}}
	pack.ResourceLimits.MaxAttempts = 1
	if err := parserpolicy.SealParserPack(pack); err != nil {
		t.Fatal(err)
	}
	packJSON, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	packJSON, err = canonicaljson.Normalize(packJSON)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(packJSON)
	bundle := synchronizedAssetBundle{SchemaVersion: 1, Assets: []synchronizedAsset{{AssetClass: "parser-packs", AssetID: pack.ID, Name: "NFI client fixture", Revision: 1, SHA256: fmt.Sprintf("%x", digest), Payload: packJSON}}}
	bundle.BundleHash, err = synchronizedBundleHash(bundle)
	if err != nil {
		t.Fatal(err)
	}
	bundleJSON, _ := json.Marshal(bundle)
	payload, _ := json.Marshal(map[string]json.RawMessage{"asset_bundle": bundleJSON})
	message, err := client.applyParsers(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(message, "1 revisioned parser assets validated and synchronized") || serviceExecutor.restarts != 1 {
		t.Fatalf("sync did not restart atomically: %s %+v", message, serviceExecutor)
	}
	installed, err := os.ReadFile(filepath.Join(client.cfg.ParserDir, pack.ID+".json"))
	if err != nil || string(installed) != string(packJSON) {
		t.Fatal("runtime bytes did not match reviewed artifact", err)
	}
	hash, _ := activeAssetBundleStatus(filepath.Join(client.cfg.AssetDir, "bundle.json"))
	if hash != bundle.BundleHash {
		t.Fatal("active acknowledgement hash differs from exact bundle")
	}
	profiles, err := familyprofile.Resolve(client.cfg.ParserDir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, profile := range profiles {
		if profile.Managed && profile.Pack != nil && profile.Pack.ID == pack.ID && profile.Executor != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("applied NFI parser absent from executable client generation")
	}
	adapter, err := runtimeadapter.NewParserPackAdapterFromDirectory(client.cfg.ParserDir)
	if err != nil {
		t.Fatal(err)
	}
	parsed := adapter.TryParseV5Production(raw, engine.Source{Hostname: "node", Service: "smtp", File: "/var/log/custom.log"})
	if !parsed.Success || parsed.Quarantined || parsed.ParseTelemetry == nil || parsed.ParseTelemetry.ParserID != "nfi-client-fixture/nfi-rule" {
		t.Fatalf("reviewed mapping did not execute in actual production adapter: %+v", parsed)
	}
	actualJSON, _ := json.Marshal(parsed.EventPayload.Fields)
	wantJSON, _ := json.Marshal(preview.Values)
	if string(actualJSON) != string(wantJSON) {
		t.Fatalf("preview/runtime drift: actual=%s expected=%s", actualJSON, wantJSON)
	}
	bundle.BundleHash = strings.Repeat("f", 64)
	badJSON, _ := json.Marshal(bundle)
	badPayload, _ := json.Marshal(map[string]json.RawMessage{"asset_bundle": badJSON})
	if _, err := client.applyParsers(context.Background(), badPayload); err == nil {
		t.Fatal("tampered bundle hash accepted")
	}
	current, _ := activeAssetBundleStatus(filepath.Join(client.cfg.AssetDir, "bundle.json"))
	if current != hash || serviceExecutor.restarts != 1 {
		t.Fatal("rejected bundle changed active state")
	}
}
