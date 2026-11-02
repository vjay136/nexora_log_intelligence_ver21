package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/fleet"
	metastore "nexora.local/nli-v20/internal/store"
)

func loginParserSyncApprover(t *testing.T, f *apiFixture) {
	t.Helper()
	response, body := f.request(http.MethodPost, "/api/v1/users", map[string]any{
		"username": "independent-approver", "email": "independent@example.test", "role": "admin", "password": "Independent-password-12345!",
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create independent approver %d: %s", response.StatusCode, body)
	}
	f.login("independent-approver", "Independent-password-12345!")
	f.refreshCSRF()
}

func putParserSyncProposal(t *testing.T, f *apiFixture, id, creator, hash string, bundle parserAssetBundle, target string) metastore.Document {
	t.Helper()
	bundle.BundleHash = hash
	payload, err := json.Marshal(map[string]any{
		"name": "Legacy parser rollout", "type": rolloutTypeParserSync,
		"target_nodes": []string{target}, "target_mode": "selected",
		"payload":           map[string]any{"asset_bundle": bundle, "bundle_hash": hash},
		"approval_required": true, "approval_reason": "verified compatibility test", "bundle_hash": hash,
	})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := f.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "rollout-proposals", ID: id, State: "pending_approval", Payload: payload, CreatedBy: creator,
	})
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestParserBundleDigestIsStableAcrossSemanticJSONFormatting(t *testing.T) {
	left := parserAssetBundle{SchemaVersion: 1, Assets: []parserBundleAsset{{
		AssetClass: "regex-patterns", AssetID: "semantic", Revision: 1, SHA256: strings.Repeat("a", 64),
		Payload: json.RawMessage(`{"weight":1.0,"nested":{"z":2e0,"a":0.0100}}`),
	}}}
	right := left
	right.Assets = append([]parserBundleAsset(nil), left.Assets...)
	right.Assets[0].Payload = json.RawMessage(`{ "nested" : { "a" : 1e-2, "z" : 2.00 }, "weight" : 10e-1 }`)
	leftHash, _, leftErr := bundleDigest(left)
	rightHash, _, rightErr := bundleDigest(right)
	if leftErr != nil || rightErr != nil || leftHash != rightHash {
		t.Fatalf("semantic bundle hashes differ: left=%s right=%s leftErr=%v rightErr=%v", leftHash, rightHash, leftErr, rightErr)
	}
	if len(leftHash) != 64 {
		t.Fatalf("canonical bundle digest is not a full SHA-256: %q", leftHash)
	}
}

func TestParserSyncProposalApprovalPreservesBundleHashAndRequiresIndependentApprover(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "parser-node", Hostname: "parser-node"})
	// Keep a decimal in the stored payload to catch float64 normalization while
	// a proposal is decoded and approved.
	assetPayload := []byte(`{"name":"decimal parser","config":{"weight":1.0}}`)
	if _, err := f.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "regex-patterns", ID: "decimal-parser", State: "active", Payload: assetPayload, CreatedBy: "admin",
	}); err != nil {
		t.Fatal(err)
	}
	bundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := f.server.proposeParserBundleRollout(context.Background(), "admin", "Decimal parser rollout", "test approval", "selected", bundle, []string{"parser-node"})
	if err != nil {
		t.Fatal(err)
	}

	response, body := f.request(http.MethodPost, "/api/v1/rollouts/"+proposal.ID+"/approve", map[string]any{})
	if response.StatusCode != http.StatusConflict || !strings.Contains(string(body), "creator cannot approve own change") {
		t.Fatalf("proposal creator approval should be rejected, got %d: %s", response.StatusCode, body)
	}

	loginParserSyncApprover(t, f)
	response, body = f.request(http.MethodPost, "/api/v1/rollouts/"+proposal.ID+"/approve", map[string]any{"reason": "Independent parser bundle review completed"})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("independent parser sync approval %d: %s", response.StatusCode, body)
	}

	approved := decodeMap(t, body)
	payload, ok := approved["payload"].(map[string]any)
	if !ok || payload["approved_by"] != "independent-approver" || payload["approval_reason"] != "Independent parser bundle review completed" || approved["state"] != "active" {
		t.Fatalf("approval attribution/state missing: %s", body)
	}
	liveID, ok := payload["live_rollout_id"].(string)
	if !ok || liveID == "" {
		t.Fatalf("approval did not create live rollout: %s", body)
	}
	live, ok := f.server.rollouts.Get(liveID)
	if !ok {
		t.Fatalf("live rollout %q not found", liveID)
	}
	var rolloutPayload struct {
		Bundle     parserAssetBundle `json:"asset_bundle"`
		BundleHash string            `json:"bundle_hash"`
	}
	if err := json.Unmarshal(live.Payload, &rolloutPayload); err != nil {
		t.Fatal(err)
	}
	if rolloutPayload.BundleHash != bundle.BundleHash || rolloutPayload.Bundle.BundleHash != bundle.BundleHash {
		t.Fatalf("live rollout bundle hash changed: payload=%s bundle=%s expected=%s", rolloutPayload.BundleHash, rolloutPayload.Bundle.BundleHash, bundle.BundleHash)
	}
	hash, _, err := bundleDigest(rolloutPayload.Bundle)
	if err != nil || hash != bundle.BundleHash {
		t.Fatalf("live rollout bundle is not verifiable: calculated=%s expected=%s err=%v", hash, bundle.BundleHash, err)
	}
	var decimalPayload json.RawMessage
	for _, asset := range rolloutPayload.Bundle.Assets {
		if asset.AssetClass == "regex-patterns" && asset.AssetID == "decimal-parser" {
			decimalPayload = asset.Payload
			break
		}
	}
	var decimalValue map[string]any
	if json.Unmarshal(decimalPayload, &decimalValue) != nil || decimalValue["config"].(map[string]any)["weight"] != float64(1) {
		t.Fatalf("live rollout did not preserve parser asset semantics: %s", decimalPayload)
	}
}

func TestParserSyncProposalIsIdempotentAndSupersedesOlderBundle(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "proposal-node", Hostname: "proposal-node"})
	firstBundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	first, err := f.server.proposeParserBundleRollout(context.Background(), "admin", "First", "first", "selected", firstBundle, []string{"proposal-node"})
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := f.server.proposeParserBundleRollout(context.Background(), "admin", "Duplicate click", "duplicate", "selected", firstBundle, []string{"proposal-node"})
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID != first.ID {
		t.Fatalf("identical parser rollout produced duplicate proposals: first=%s duplicate=%s", first.ID, duplicate.ID)
	}
	if _, err := f.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "regex-patterns", ID: "new-parser-rule", State: "active",
		Payload: json.RawMessage(`{"name":"New parser rule"}`), CreatedBy: "admin",
	}); err != nil {
		t.Fatal(err)
	}
	secondBundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.server.proposeParserBundleRollout(context.Background(), "admin", "Second", "second", "selected", secondBundle, []string{"proposal-node"})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID || secondBundle.BundleHash == firstBundle.BundleHash {
		t.Fatal("new parser bundle did not create one new proposal")
	}
	superseded, err := f.server.metadata.Get(context.Background(), "rollout-proposals", first.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if superseded.State != "superseded" {
		t.Fatalf("older proposal remains actionable: state=%s", superseded.State)
	}
	var payload map[string]any
	if json.Unmarshal(superseded.Payload, &payload) != nil || payload["superseded_by"] != second.ID || payload["superseded_by_bundle_hash"] != secondBundle.BundleHash {
		t.Fatalf("supersession evidence missing: %s", superseded.Payload)
	}
}

func TestLegacyParserSyncProposalIsSafelyUpgradedAndPersisted(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "legacy-node", Hostname: "legacy-node"})
	if _, err := f.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "regex-patterns", ID: "legacy-rule", State: "active",
		Payload: json.RawMessage(`{"name":"Legacy rule","config":{"weight":1.0}}`), CreatedBy: "admin",
	}); err != nil {
		t.Fatal(err)
	}
	bundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	legacyBundle := bundle
	legacyBundle.Assets = append([]parserBundleAsset(nil), bundle.Assets...)
	for index := range legacyBundle.Assets {
		if legacyBundle.Assets[index].AssetID == "legacy-rule" {
			legacyPayload := json.RawMessage(`{ "config": { "weight": 1.0 }, "name": "Legacy rule" }`)
			legacyBundle.Assets[index].Payload = legacyPayload
			legacyDigest := sha256.Sum256(legacyPayload)
			legacyBundle.Assets[index].SHA256 = fmt.Sprintf("%x", legacyDigest[:])
		}
	}
	legacyHash, _, err := legacyBundleDigest(legacyBundle)
	if err != nil {
		t.Fatal(err)
	}
	if legacyHash == bundle.BundleHash {
		t.Fatal("test fixture did not produce a legacy/canonical hash transition")
	}
	doc := putParserSyncProposal(t, f, "legacy-parser-sync", "admin", legacyHash, legacyBundle, "legacy-node")
	loginParserSyncApprover(t, f)
	response, body := f.request(http.MethodPost, "/api/v1/rollouts/"+doc.ID+"/approve", map[string]any{"reason": "Independent review of legacy proposal"})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("legacy proposal approval failed %d: %s", response.StatusCode, body)
	}
	approved := decodeMap(t, body)
	payload := approved["payload"].(map[string]any)
	if payload["bundle_hash"] != bundle.BundleHash || payload["approval_mode"] != "independent" {
		t.Fatalf("legacy proposal was not canonically approved: %s", body)
	}
	migration, ok := payload["bundle_hash_migration"].(map[string]any)
	if !ok || migration["from"] != legacyHash || migration["to"] != bundle.BundleHash || migration["migrated_by"] != "independent-approver" {
		t.Fatalf("legacy migration evidence was not persisted: %s", body)
	}
	liveID, _ := payload["live_rollout_id"].(string)
	live, ok := f.server.rollouts.Get(liveID)
	if !ok {
		t.Fatalf("legacy approval did not create rollout %q", liveID)
	}
	var livePayload struct {
		Bundle     parserAssetBundle `json:"asset_bundle"`
		BundleHash string            `json:"bundle_hash"`
	}
	if err := json.Unmarshal(live.Payload, &livePayload); err != nil {
		t.Fatal(err)
	}
	if livePayload.BundleHash != bundle.BundleHash || livePayload.Bundle.BundleHash != bundle.BundleHash {
		t.Fatalf("rollout did not receive canonical bundle hash: %#v", livePayload)
	}
}

func TestR4JSONBRoundTripHashMigratesOnlyWhenSemanticSnapshotIsCurrent(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "r4-jsonb-node", Hostname: "r4-jsonb-node"})
	if _, err := f.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "regex-patterns", ID: "r4-jsonb-rule", State: "active",
		Payload: json.RawMessage(`{"z":2,"name":"R4 jsonb rule","weight":1.0}`), CreatedBy: "admin",
	}); err != nil {
		t.Fatal(err)
	}
	bundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the unreconstructible raw-byte digest persisted by R4 before a
	// jsonb round trip. The full digest fields agree, while semantic content is
	// still independently compared with the current published bundle.
	r4RawDigest := strings.Repeat("a", 64)
	doc := putParserSyncProposal(t, f, "r4-jsonb-parser-sync", "admin", r4RawDigest, bundle, "r4-jsonb-node")
	loginParserSyncApprover(t, f)
	response, body := f.request(http.MethodPost, "/api/v1/rollouts/"+doc.ID+"/approve", map[string]any{"reason": "Reviewed R4 JSONB migration"})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("R4 JSONB proposal approval failed %d: %s", response.StatusCode, body)
	}
	payload := decodeMap(t, body)["payload"].(map[string]any)
	migration, ok := payload["bundle_hash_migration"].(map[string]any)
	if !ok || migration["from"] != r4RawDigest || migration["to"] != bundle.BundleHash || migration["mode"] != "verified_semantic_snapshot" {
		t.Fatalf("R4 JSONB migration evidence missing: %s", body)
	}
}

func TestLegacyShortHashEmergencyApprovalUpgradesExactBundle(t *testing.T) {
	f := newAPIFixture(t)
	f.server.cfg.AllowAdminEmergencySelfApproval = true
	f.server.fleet.Register(&fleet.Node{ID: "legacy-emergency-node", Hostname: "legacy-emergency-node"})
	if _, err := f.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "regex-patterns", ID: "legacy-emergency-rule", State: "active",
		Payload: json.RawMessage(`{"name":"Legacy emergency","weight":1.0}`), CreatedBy: "admin",
	}); err != nil {
		t.Fatal(err)
	}
	bundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	legacyHash, _, err := legacyBundleDigest(bundle)
	if err != nil {
		t.Fatal(err)
	}
	shortHash := legacyHash[:16]
	doc := putParserSyncProposal(t, f, "legacy-emergency-parser-sync", "admin", shortHash, bundle, "legacy-emergency-node")
	response, body := f.request(http.MethodPost, "/api/v1/rollouts/"+doc.ID+"/approve", map[string]any{
		"emergency_self_approval": true,
		"reason":                  "Production recovery requires this exact legacy parser bundle.",
		"expected_revision":       doc.Revision,
		"expected_bundle_hash":    shortHash,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("legacy emergency proposal approval failed %d: %s", response.StatusCode, body)
	}
	payload := decodeMap(t, body)["payload"].(map[string]any)
	if payload["bundle_hash"] != bundle.BundleHash || payload["approval_mode"] != "emergency_admin_self" {
		t.Fatalf("legacy emergency approval was not bound to canonical hash: %s", body)
	}
}

func TestMalformedParserSyncProposalIsExplicitlyNonActionable(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "malformed-node", Hostname: "malformed-node"})
	bundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	doc := putParserSyncProposal(t, f, "malformed-parser-sync", "admin", "not-a-sha256", bundle, "malformed-node")
	loginParserSyncApprover(t, f)
	response, body := f.request(http.MethodPost, "/api/v1/rollouts/"+doc.ID+"/approve", map[string]any{})
	if response.StatusCode != http.StatusConflict || !strings.Contains(string(body), "stored bundle hash is malformed") || strings.Contains(string(body), "internal") {
		t.Fatalf("malformed proposal did not return precise conflict: %d %s", response.StatusCode, body)
	}
	if _, ok := f.server.rollouts.Get(doc.ID); ok {
		t.Fatal("malformed proposal created a live rollout")
	}
}

func TestStaleLegacyParserSyncProposalRequiresRebase(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "stale-node", Hostname: "stale-node"})
	bundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	legacyHash, _, err := legacyBundleDigest(bundle)
	if err != nil {
		t.Fatal(err)
	}
	doc := putParserSyncProposal(t, f, "stale-parser-sync", "admin", legacyHash, bundle, "stale-node")
	if _, err := f.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "regex-patterns", ID: "newer-rule", State: "active",
		Payload: json.RawMessage(`{"name":"Newer published rule"}`), CreatedBy: "admin",
	}); err != nil {
		t.Fatal(err)
	}
	loginParserSyncApprover(t, f)
	response, body := f.request(http.MethodPost, "/api/v1/rollouts/"+doc.ID+"/approve", map[string]any{})
	if response.StatusCode != http.StatusConflict || !strings.Contains(string(body), "proposal is stale") || !strings.Contains(string(body), "create a new proposal") {
		t.Fatalf("stale proposal did not return rebase guidance: %d %s", response.StatusCode, body)
	}
}
