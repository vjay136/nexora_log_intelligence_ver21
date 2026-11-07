package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"nexora.local/nli-v20/internal/agent"
	"nexora.local/nli-v20/internal/output"
	"nexora.local/nli-v20/internal/quarantine"
)

func replayDeliveryFixture(t *testing.T, url string) (agent.Config, string, string, string) {
	t.Helper()
	root := t.TempDir()
	queue := filepath.Join(root, "quarantine.jsonl")
	normal := filepath.Join(root, "normal.jsonl")
	input := filepath.Join(root, "input.log")
	_ = os.WriteFile(input, nil, 0600)
	raw := "20250811 080053102+0100 aaa-gmailrgi009 smtp 3692 0 140026870728448 Note;SmtpServerConnect(2001/3) id=67F5E2BF38CCA05B:host=192.0.2.1:connection_count=1:local_port=2525"
	record := quarantine.New("node-synthetic", "smtp-in", input, 1, raw, "old", 1, []string{"no_parser"}, time.Now().UTC())
	if err := quarantine.Append(queue, record); err != nil {
		t.Fatal(err)
	}
	cfg := agent.Config{Version: 2, Inputs: []agent.InputConfig{{Path: input, Service: "auto"}}, StateDir: filepath.Join(root, "state"), PollMS: 50, DisableV5Parser: true, Outputs: []output.Config{{Type: "file", Path: normal, Routes: []string{"normal", "warning"}, Families: []string{"smtp-in"}}, {Type: "file", Path: queue, Routes: []string{"quarantine", "unknown"}}, {Type: "loki", URL: url, Required: true, RetryMax: 1, RetryBackoffMS: 1, BatchSize: 1000, Routes: []string{"normal", "warning"}}}, TelemetryOutputs: []output.Config{{Type: "discard"}}}
	encoded, _ := json.Marshal(cfg)
	configPath := filepath.Join(root, "nli.json")
	if err := os.WriteFile(configPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	return cfg, configPath, queue, normal
}

func replayRowCount(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		if json.Unmarshal([]byte(line), &record) != nil {
			return 0, false
		}
		count++
	}
	return count, true
}

func TestTargetedReplayAcknowledgesOnlySelectedRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	_, configPath, queue, _ := replayDeliveryFixture(t, server.URL)
	data, err := os.ReadFile(queue)
	if err != nil {
		t.Fatal(err)
	}
	var selected quarantine.Record
	if err := json.Unmarshal(data, &selected); err != nil {
		t.Fatal(err)
	}
	queueFile, err := os.OpenFile(queue, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer queueFile.Close()
	encoder := json.NewEncoder(queueFile)
	for i := 0; i < 1001; i++ {
		record := quarantine.New(selected.NodeID, selected.Service, selected.SourcePath, uint64(i+2), "unrelated queued sample", "old", 5, nil, time.Now().UTC())
		if err := encoder.Encode(record); err != nil {
			t.Fatal(err)
		}
	}
	if err := queueFile.Close(); err != nil {
		t.Fatal(err)
	}
	var response strings.Builder
	if err := replayQueue(queue, configPath, "", []string{selected.ID}, &response); err != nil {
		t.Fatal(err)
	}
	var summary replaySummary
	if err := json.Unmarshal([]byte(response.String()), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Replayed != 1 || summary.Resolved != 1 || len(summary.ResolvedIDs) != 1 || summary.ResolvedIDs[0] != selected.ID || summary.Waiting != 0 || len(summary.WaitingIDs) != 0 {
		t.Fatalf("selected replay returned unrelated outcomes: replayed=%d resolved=%d waiting=%d", summary.Replayed, summary.Resolved, summary.Waiting)
	}
	if count, ok := replayRowCount(queue); !ok || count != 1001 {
		t.Fatalf("targeted replay changed the unrelated queue: count=%d valid=%t", count, ok)
	}
}

func TestReplayRequiredRemoteRetryDoesNotDuplicateLocalPayload(t *testing.T) {
	var unavailable atomic.Bool
	unavailable.Store(true)
	var accepted atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if unavailable.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var push struct {
			Streams []struct {
				Values [][2]string `json:"values"`
			} `json:"streams"`
		}
		_ = json.NewDecoder(r.Body).Decode(&push)
		for _, stream := range push.Streams {
			accepted.Add(int64(len(stream.Values)))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	cfg, configPath, queue, normal := replayDeliveryFixture(t, server.URL)
	var summary strings.Builder
	if err := replayQueue(queue, configPath, "", nil, &summary); err == nil {
		t.Fatal("required remote outage was acknowledged")
	}
	if count, _ := replayRowCount(queue); count != 1 {
		t.Fatal("required failure removed replayable evidence")
	}
	if count, ok := replayRowCount(normal); !ok || count != 1 {
		t.Fatal("first local delivery missing")
	}
	wal := output.DeliveryJournalPath(cfg.StateDir, normal, "")
	beforeCleanup, err := os.ReadFile(wal)
	if err != nil || len(beforeCleanup) == 0 {
		t.Fatal("unacknowledged replay receipt was retired prematurely")
	}
	var pending quarantine.Record
	data, _ := os.ReadFile(queue)
	if err = json.Unmarshal(data, &pending); err != nil || pending.ReplayAttempt != 1 || pending.ReplayRevision == "" {
		t.Fatal("delivery attempt identity was not retained across transport retry")
	}
	unavailable.Store(false)
	summary.Reset()
	if err = replayQueue(queue, configPath, "", nil, &summary); err != nil {
		t.Fatal(err)
	}
	if count, ok := replayRowCount(normal); !ok || count != 1 || accepted.Load() != 1 {
		t.Fatalf("retry produced local=%d acceptedremote=%d", count, accepted.Load())
	}
	if count, _ := replayRowCount(queue); count != 0 {
		t.Fatal("resolved evidence remained in active queue")
	}
	// Simulate a stop after queue commit but before receipt compaction.
	if err = os.WriteFile(wal, beforeCleanup, 0600); err != nil {
		t.Fatal(err)
	}
	if err = replayQueue(queue, configPath, "", nil, io.Discard); err != nil {
		t.Fatal(err)
	}
	remaining, _ := os.ReadFile(wal)
	if len(remaining) != 0 {
		t.Fatal("startup replay did not retire post-commit stale receipts")
	}
	if count, _ := replayRowCount(normal); count != 1 {
		t.Fatal("empty replay emitted another customer record")
	}
}

func TestLiveAgentAppendsSurviveReplaySnapshotAndQueueReplacement(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	cfg, configPath, queue, normal := replayDeliveryFixture(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agentDone := make(chan error, 1)
	go func() { agentDone <- agent.Run(ctx, cfg, false) }()
	replayDone := make(chan error, 1)
	go func() { replayDone <- replayQueue(queue, configPath, "", nil, io.Discard) }()
	select {
	case <-entered:
	case err := <-replayDone:
		t.Fatalf("replay stopped before network barrier: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("replay did not reach delivery")
	}
	appendInput := func(label string) {
		f, err := os.OpenFile(cfg.Inputs[0].Path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, err = fmt.Fprintf(f, "new-%s\x00\n", label)
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
	waitRows := func(expected int) {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if count, ok := replayRowCount(queue); ok && count == expected {
				return
			}
			select {
			case err := <-agentDone:
				t.Fatalf("live agent stopped: %v", err)
			default:
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("active queue did not reach %d rows", expected)
	}
	appendInput("during-http")
	waitRows(2) // This must progress while the replay push is blocked.
	close(release)
	select {
	case err := <-replayDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("replay did not finish")
	}
	waitRows(1)
	appendInput("after-replacement")
	waitRows(2)
	cancel()
	select {
	case err := <-agentDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("agent did not stop")
	}
	if count, _ := replayRowCount(normal); count != 1 {
		t.Fatal("replay did not deliver exactly one resolved payload")
	}
	data, _ := os.ReadFile(queue)
	if !strings.Contains(string(data), "during-http") || !strings.Contains(string(data), "after-replacement") {
		t.Fatal("new raw evidence was lost during queue replacement")
	}
}
