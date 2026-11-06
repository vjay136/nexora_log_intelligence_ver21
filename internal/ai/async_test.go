package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAsyncDiscoveryNeverBlocksAndRetryQueueRemainsBounded(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	discoverer := New(Config{Enabled: true, Provider: "ollama", Endpoint: server.URL, Model: "test", MinClusterSize: 1, MaxPendingClusters: 1}, nil, nil, t.TempDir())
	discoverer.Submit(1, "sample status=failed")
	if !discoverer.StartBatch(context.Background(), map[int]int{1: 1}) {
		t.Fatal("batch not started")
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("provider was not invoked")
	}
	if discoverer.StartBatch(context.Background(), map[int]int{1: 1}) {
		t.Fatal("parallel batch was allowed")
	}
	discoverer.Submit(2, "next sample")
	close(release)
	deadline := time.Now().Add(time.Second)
	for {
		if discoverer.batchMu.TryLock() {
			discoverer.batchMu.Unlock()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("worker did not release its gate")
		}
		time.Sleep(time.Millisecond)
	}
	stats := discoverer.Stats()
	if stats.PendingClusters > 1 || stats.DroppedPending != 1 {
		t.Fatalf("retry exceeded queue bound: %+v", stats)
	}
}
