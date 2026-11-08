package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"nexora.local/nli-v20/internal/fleet"
	"nexora.local/nli-v20/internal/protocol"
	rolloutmodel "nexora.local/nli-v20/internal/rollout"
	metastore "nexora.local/nli-v20/internal/store"
	"nexora.local/nli-v20/telemetry"
)

func TestDottedClientHeartbeatPreservesAndRecoversParserAcknowledgement(t *testing.T) {
	f := newAPIFixture(t)
	ctx := context.Background()
	const node = "ip-172-31-37-151.ap-southeast-2.compute.internal-client"
	f.server.fleet.Register(&fleet.Node{ID: node, Hostname: node})
	bundle, err := f.server.buildActiveParserBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	files := []protocol.ParserFileDigest{}
	for _, asset := range bundle.Assets {
		if asset.AssetClass == "parser-packs" {
			files = append(files, protocol.ParserFileDigest{ID: asset.AssetID, SHA256: asset.SHA256})
		}
	}
	heartbeat := protocol.HeartbeatRequest{
		NodeID:                node,
		ActiveAssetBundleHash: bundle.BundleHash,
		RuntimeInventory: &protocol.ParserRuntimeInventory{
			BundleHash:         bundle.BundleHash,
			ManagedFilesDigest: protocol.ManagedParserFilesDigest(files),
		},
	}
	report := func() map[string]any {
		t.Helper()
		f.server.updateSyncFromHeartbeat(heartbeat)
		doc, err := f.server.metadata.Get(ctx, "fleet-sync", safeID(node), 0)
		if err != nil {
			t.Fatal(err)
		}
		var state map[string]any
		if err := json.Unmarshal(doc.Payload, &state); err != nil {
			t.Fatal(err)
		}
		return state
	}
	if state := report(); parserBundleAcknowledged(state, bundle.BundleHash) {
		t.Fatalf("unapproved client report must not authorize parser delivery: %+v", state)
	}
	job, err := f.server.startParserBundleRollout(ctx, "operator", "Dotted client rollout", bundle, []string{node})
	if err != nil {
		t.Fatal(err)
	}
	if state := report(); !parserBundleAcknowledged(state, bundle.BundleHash) {
		t.Fatalf("heartbeat erased desired state for dotted client: %+v", state)
	}

	// Simulate the legacy heartbeat overwriting the desired bundle record.
	if _, err := f.server.metadata.Put(ctx, metastore.Document{
		Collection: "fleet-sync", ID: safeID(node), State: "reported",
		Payload: json.RawMessage(`{"asset_sync_status":"drifted"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if state := report(); parserBundleAcknowledged(state, bundle.BundleHash) {
		t.Fatalf("legacy state must remain gated until there is a completed delivery: %+v", state)
	}
	if err := f.server.rollouts.RecordNodeResult(job.ID, node, rolloutStatusCompleted, "applied"); err != nil {
		t.Fatal(err)
	}
	if state := report(); !parserBundleAcknowledged(state, bundle.BundleHash) {
		t.Fatalf("completed exact delivery did not repair legacy state: %+v", state)
	}
}

func TestPublishedAssurancePackIsOnlyRuntimeBundleHandoff(t *testing.T) {
	f := newAPIFixture(t)
	definition := map[string]any{"pattern_type": "regex", "pattern": `status=(?P<status>\w+)`, "fields": []any{map[string]any{"name": "status", "source": "group:status", "type": "string"}}}
	cases := []assuranceRequest{{Raw: "status=OK", Service: "mail", Desired: map[string]any{"status": "OK"}}}
	pack, err := assuranceRuntimePack("recovery-parser", "1.2.0", "author", definition, cases)
	if err != nil {
		t.Fatal(err)
	}
	packJSON, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"draft", "rejected", "approved", "published"} {
		id := "recovery-" + state
		if state == "published" {
			id = "recovery-parser"
		}
		payload := map[string]any{"id": id, "name": "Recovery " + state, "review_state": state}
		if state == "published" {
			payload["runtime_pack"] = json.RawMessage(packJSON)
		}
		encoded, _ := json.Marshal(payload)
		if _, err := f.server.metadata.Put(context.Background(), metastore.Document{Collection: "parser-packs", ID: id, State: state, Payload: encoded, CreatedBy: "author"}); err != nil {
			t.Fatal(err)
		}
	}
	bundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var included *parserBundleAsset
	for i := range bundle.Assets {
		if bundle.Assets[i].AssetClass != "parser-packs" {
			continue
		}
		if included != nil {
			t.Fatalf("more than one Assurance parser entered desired bundle: %#v", bundle.Assets)
		}
		included = &bundle.Assets[i]
	}
	if included == nil || included.AssetID != "recovery-parser" {
		t.Fatalf("non-published Assurance parsers entered desired bundle: %#v", bundle.Assets)
	}
	wantPayloadHash := sha256.Sum256(included.Payload)
	if included.SHA256 != hex.EncodeToString(wantPayloadHash[:]) || included.Revision != 1 {
		t.Fatalf("published parser bundle asset lost immutable payload revision/hash: %+v", included)
	}

	f.server.fleet.Register(&fleet.Node{ID: "recovery-node", Hostname: "recovery-node"})
	rollout, err := f.server.startParserBundleRollout(context.Background(), "operator", "Recovery parser rollout", bundle, []string{"recovery-node"})
	if err != nil || rollout == nil {
		t.Fatalf("start selected-client rollout: rollout=%+v err=%v", rollout, err)
	}
	var rolloutPayload map[string]any
	if json.Unmarshal(rollout.Payload, &rolloutPayload) != nil || rolloutPayload["bundle_hash"] != bundle.BundleHash {
		t.Fatalf("selected-client rollout targets a different bundle hash: %s", rollout.Payload)
	}
	var storedBundle parserAssetBundle
	rolloutBundleJSON, _ := json.Marshal(rolloutPayload["asset_bundle"])
	if err := json.Unmarshal(rolloutBundleJSON, &storedBundle); err != nil || storedBundle.BundleHash != bundle.BundleHash {
		t.Fatalf("rollout bundle does not match inventory hash: %+v err=%v", storedBundle, err)
	}

	if err := f.server.storeQuarantineRecord(context.Background(), quarantineRecordInput{ID: "recovery-q", NodeID: "recovery-node", Status: "waiting", AttemptCount: 5}, false); err != nil {
		t.Fatal(err)
	}
	f.login("admin", "Admin-password-12345!")
	f.refreshCSRF()
	replayBody := map[string]any{"node_id": "recovery-node", "record_ids": []string{"recovery-q"}}
	telemetryUpdatedAt := time.Now().UTC().Add(-2 * time.Second)
	f.server.streamTelemetry.Publish(telemetry.StreamFileTelemetry{
		NodeID: "recovery-node", Source: "/var/log/recovery.log", ReadLines: 11, UpdatedAt: telemetryUpdatedAt,
	})
	response, body := f.request(http.MethodPost, "/api/v1/quarantine/reprocess", replayBody)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("replay must wait for target applied acknowledgement, got %d: %s", response.StatusCode, body)
	}
	conflict := decodeMap(t, body)
	if conflict["code"] != "parser_bundle_not_acknowledged" || conflict["retryable"] != true || conflict["live_sync_impact"] != false {
		t.Fatalf("replay gate is not a stable, actionable conflict: %#v", conflict)
	}
	if conflict["expected_bundle_hash"] != bundle.BundleHash || conflict["desired_bundle_hash"] != bundle.BundleHash {
		t.Fatalf("replay gate omitted the safe expected/desired hash: %#v", conflict)
	}
	snapshot := f.server.streamTelemetry.Snapshot()
	if len(snapshot) != 1 || snapshot[0].ReadLines != 11 || !snapshot[0].UpdatedAt.Equal(telemetryUpdatedAt) {
		t.Fatalf("gated replay changed canonical telemetry freshness: %#v", snapshot)
	}
	stateBytes, _ := json.Marshal(map[string]any{"node_id": "recovery-node", "desired_asset_bundle_hash": "stale-desired-hash", "active_asset_bundle_hash": bundle.BundleHash, "asset_sync_status": "applied"})
	if _, err := f.server.metadata.Put(context.Background(), metastore.Document{Collection: "fleet-sync", ID: "recovery-node", State: "reported", Payload: stateBytes, CreatedBy: "node:recovery-node"}); err != nil {
		t.Fatal(err)
	}
	response, body = f.request(http.MethodPost, "/api/v1/quarantine/reprocess", replayBody)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("matching active hash with stale desired hash must remain gated, got %d: %s", response.StatusCode, body)
	}
	stateBytes, _ = json.Marshal(map[string]any{"node_id": "recovery-node", "desired_asset_bundle_hash": bundle.BundleHash, "active_asset_bundle_hash": bundle.BundleHash, "asset_sync_status": "drifted"})
	if _, err := f.server.metadata.Put(context.Background(), metastore.Document{Collection: "fleet-sync", ID: "recovery-node", State: "reported", Payload: stateBytes, CreatedBy: "node:recovery-node"}); err != nil {
		t.Fatal(err)
	}
	response, body = f.request(http.MethodPost, "/api/v1/quarantine/reprocess", replayBody)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("matching hash without applied acknowledgement must remain gated, got %d: %s", response.StatusCode, body)
	}
	f.server.recordParserSyncRolloutResult(context.Background(), rollout.ID, protocol.CommandResult{
		NodeID:                "recovery-node",
		CommandID:             rollout.ID,
		Status:                rolloutmodel.StatusCompleted,
		ActiveAssetBundleHash: bundle.BundleHash,
		ActiveAssetRevisions:  bundleRevisionMap(bundle),
	})
	acknowledgement, err := f.server.metadata.Get(context.Background(), "fleet-sync", "recovery-node", 0)
	if err != nil {
		t.Fatal(err)
	}
	var acknowledgedState map[string]any
	if json.Unmarshal(acknowledgement.Payload, &acknowledgedState) != nil || acknowledgedState["active_asset_bundle_hash"] != bundle.BundleHash || acknowledgedState["asset_sync_status"] != "applied" {
		t.Fatalf("successful command result did not immediately acknowledge the exact bundle: %#v", acknowledgedState)
	}
	response, body = f.request(http.MethodPost, "/api/v1/quarantine/reprocess", replayBody)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("replay should open only after matching applied acknowledgement, got %d: %s", response.StatusCode, body)
	}
	record, _, err := f.server.getQuarantineRecord(context.Background(), "recovery-q")
	if err != nil || record.AttemptCount != 5 || record.ReplayAttemptCount != 1 || record.LastReplayBundleHash != bundle.BundleHash {
		t.Fatalf("five parser strategy attempts must not exhaust the governed replay budget: %+v, %v", record, err)
	}

	enableTestQuarantineRetention(t, f)
	ctx := context.Background()
	const exhaustedID = "recovery-exhausted"
	if err := f.server.storeQuarantineRecord(ctx, quarantineRecordInput{
		ID: exhaustedID, NodeID: "recovery-node", Status: "dead_letter", AttemptCount: 5, Raw: "retained exact sample",
	}, true); err != nil {
		t.Fatal(err)
	}
	exhausted, doc, err := f.server.getQuarantineRecord(ctx, exhaustedID)
	if err != nil {
		t.Fatal(err)
	}
	replayBody["record_ids"] = []string{exhaustedID}
	response, body = f.request(http.MethodPost, "/api/v1/quarantine/reprocess", replayBody)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("legacy exhausted record without a known prior bundle cannot retry: %d %s", response.StatusCode, body)
	}
	exhausted.LastReplayBundleHash = bundle.BundleHash
	if err := f.server.putQuarantineRecord(ctx, exhausted, doc.CreatedBy); err != nil {
		t.Fatal(err)
	}
	response, body = f.request(http.MethodPost, "/api/v1/quarantine/reprocess", replayBody)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("exhausted record cannot retry the same bundle: %d %s", response.StatusCode, body)
	}
	exhausted.LastReplayBundleHash = strings.Repeat("a", 64)
	if err := f.server.putQuarantineRecord(ctx, exhausted, doc.CreatedBy); err != nil {
		t.Fatal(err)
	}
	replayBody["expected_bundle_hash"] = exhausted.LastReplayBundleHash
	response, body = f.request(http.MethodPost, "/api/v1/quarantine/reprocess", replayBody)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("stale wizard bundle cannot request replay: %d %s", response.StatusCode, body)
	}
	replayBody["expected_bundle_hash"] = bundle.BundleHash
	response, body = f.request(http.MethodPost, "/api/v1/quarantine/reprocess", replayBody)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("exhausted record should get one recovery on a changed acknowledged bundle: %d %s", response.StatusCode, body)
	}
	exhausted, doc, err = f.server.getQuarantineRecord(ctx, exhaustedID)
	if err != nil || exhausted.ReplayAttemptCount != quarantineReplayLimit+1 || !exhausted.RecoveryReplayUsed {
		t.Fatalf("recovery must consume the one extra attempt: %+v, %v", exhausted, err)
	}
	f.server.recordQuarantineReplayResult(ctx, exhausted.ReplayRolloutID, exhausted.NodeID, rolloutStatusFailed, "")
	exhausted, doc, err = f.server.getQuarantineRecord(ctx, exhaustedID)
	if err != nil || exhausted.Status != "dead_letter" {
		t.Fatalf("failed recovery must remain visible as exhausted: %+v, %v", exhausted, err)
	}
	exhausted.LastReplayBundleHash = strings.Repeat("b", 64)
	if err := f.server.putQuarantineRecord(ctx, exhausted, doc.CreatedBy); err != nil {
		t.Fatal(err)
	}
	response, body = f.request(http.MethodPost, "/api/v1/quarantine/reprocess", replayBody)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("a second recovery is never available: %d %s", response.StatusCode, body)
	}
}

func TestLateParserRolloutAcknowledgementCannotRewindReplayGate(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "late-ack-node", Hostname: "late-ack-node"})
	oldBundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	oldRollout, err := f.server.startParserBundleRollout(context.Background(), "operator", "Old parser bundle", oldBundle, []string{"late-ack-node"})
	if err != nil || oldRollout == nil {
		t.Fatalf("start old parser rollout: rollout=%+v err=%v", oldRollout, err)
	}
	if _, err := f.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "regex-patterns", ID: "new-late-ack-rule", State: "active",
		Payload: json.RawMessage(`{"name":"New late acknowledgement rule"}`), CreatedBy: "operator",
	}); err != nil {
		t.Fatal(err)
	}
	newBundle, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if newBundle.BundleHash == oldBundle.BundleHash {
		t.Fatal("fixture did not change the published parser bundle")
	}
	newRollout, err := f.server.startParserBundleRollout(context.Background(), "operator", "New parser bundle", newBundle, []string{"late-ack-node"})
	if err != nil || newRollout == nil {
		t.Fatalf("start new parser rollout: rollout=%+v err=%v", newRollout, err)
	}

	f.server.recordParserSyncRolloutResult(context.Background(), oldRollout.ID, protocol.CommandResult{
		NodeID: "late-ack-node", CommandID: oldRollout.ID, Status: rolloutmodel.StatusCompleted,
		ActiveAssetBundleHash: oldBundle.BundleHash, ActiveAssetRevisions: bundleRevisionMap(oldBundle),
	})
	doc, err := f.server.metadata.Get(context.Background(), "fleet-sync", "late-ack-node", 0)
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	if json.Unmarshal(doc.Payload, &state) != nil {
		t.Fatalf("decode fleet sync state: %s", doc.Payload)
	}
	if state["desired_asset_bundle_hash"] != newBundle.BundleHash || state["active_asset_bundle_hash"] != oldBundle.BundleHash || state["asset_sync_status"] != "drifted" {
		t.Fatalf("late acknowledgement rewound desired state or opened replay gate: %#v", state)
	}
}
