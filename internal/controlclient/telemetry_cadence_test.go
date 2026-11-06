package controlclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"nexora.local/nli-v20/internal/protocol"
	"nexora.local/nli-v20/telemetry"
)

func TestR9TelemetryDefaultFastCadenceAndExplicitLegacyCompatibility(t *testing.T) {
	cases := []struct {
		config Config
		want   time.Duration
	}{{Config{}, 250 * time.Millisecond}, {Config{TelemetryPollSeconds: 3}, 3 * time.Second}, {Config{TelemetryPollSeconds: 3, TelemetryPollMS: 100}, 100 * time.Millisecond}}
	for _, tc := range cases {
		tc.config.setDefaults()
		if got := tc.config.telemetryInterval(); got != tc.want {
			t.Fatalf("cadence=%s want%s", got, tc.want)
		}
	}
}

func TestR9TelemetryMillionCounterBurstCoalescesAndKeepsExactFinal(t *testing.T) {
	var count atomic.Uint64
	metrics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"files": []telemetry.StreamFileTelemetry{{Source: "/logs/source.log", Phase: "reading", ReadLines: count.Load(), Normalized: count.Load(), UpdatedAt: time.Now()}}})
	}))
	defer metrics.Close()
	root := t.TempDir()
	client, err := New(Config{ControllerURL: "http://127.0.0.1:12345", AllowInsecureHTTP: true, MetricsURL: metrics.URL + "/metrics", NodeTokenFile: filepath.Join(root, "token"), InstallIDFile: filepath.Join(root, "id"), JournalFile: filepath.Join(root, "journal"), AgentConfigPath: filepath.Join(root, "config"), ParserDir: filepath.Join(root, "parsers"), AssetDir: filepath.Join(root, "assets"), PackageSpoolDir: filepath.Join(root, "packages")}, &fakeExecutor{})
	if err != nil {
		t.Fatal(err)
	}
	client.nodeID = "burst-node"
	client.token = "burst-token"
	var batches []protocol.StreamTelemetryRequest
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var batch protocol.StreamTelemetryRequest
		if err := json.NewDecoder(request.Body).Decode(&batch); err != nil {
			return nil, err
		}
		batches = append(batches, batch)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: request}, nil
	})
	if err := client.sendStreamTelemetry(context.Background()); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	for i := 0; i < 1000000; i++ {
		count.Add(1)
	}
	// One publication tick carries the aggregate million-record burst, not a
	// million HTTP calls. This tests telemetry transport, not storage throughput.
	time.Sleep(client.cfg.telemetryInterval())
	if err := client.sendStreamTelemetry(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(batches) != 2 || batches[1].StreamFiles[0].ReadLines != 1000000 || batches[1].StreamFiles[0].Normalized != 1000000 || batches[1].PublicationIntervalMS != 250 {
		t.Fatalf("exact final counters not published in bounded batch: %+v", batches)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("final burst publication took %s", elapsed)
	}
	for i := 0; i < 10; i++ {
		if err := client.sendStreamTelemetry(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(batches) != 2 {
		t.Fatalf("timestamp-only changes flooded controller: %d requests", len(batches))
	}
	client.lastStreamSent = time.Now().Add(-6 * time.Second)
	if err := client.sendStreamTelemetry(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(batches) != 3 {
		t.Fatal("idle keepalive suppressed")
	}
}

func TestR9RuntimeInventoryContainsManagedAndOperatorDigestsOnly(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "parsers")
	_ = os.MkdirAll(dir, 0750)
	_ = os.WriteFile(filepath.Join(dir, "managed.json"), []byte(`{"id":"managed"}`), 0640)
	_ = os.WriteFile(filepath.Join(dir, "operator.json"), []byte(`{"id":"operator"}`), 0640)
	_ = os.WriteFile(filepath.Join(dir, managedParserManifest), []byte(`["managed"]`), 0640)
	client := &Client{cfg: Config{ParserDir: dir, AssetDir: filepath.Join(root, "assets")}}
	inventory := client.runtimeParserInventory()
	if len(inventory.Files) != 2 || !inventory.Files[0].ControllerManaged || inventory.Files[1].ControllerManaged || len(inventory.Files[0].SHA256) != 64 {
		t.Fatalf("wrong runtime ownership inventory: %+v", inventory)
	}
	data, err := os.ReadFile(inventory.InventoryPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"id":"operator"`) {
		t.Fatal("inventory included parser contents")
	}
}

func TestR9ReplayRejectsParserGenerationChangeBeforeExecution(t *testing.T) {
	executor := &fakeExecutor{}
	client := &Client{cfg: Config{AssetDir: t.TempDir(), QuarantinePath: "/quarantine.jsonl"}, executor: executor}
	payload := json.RawMessage(`{"record_ids":["retained-record"],"expected_asset_bundle_hash":"expected-reviewed-generation"}`)
	_, err := client.apply(context.Background(), &protocol.Command{Type: "replay", Payload: payload})
	if err == nil || !strings.Contains(err.Error(), "replay parser bundle changed") || len(executor.replayIDs) != 0 {
		t.Fatalf("replay used an unverified parser generation: %v %+v", err, executor.replayIDs)
	}
}
