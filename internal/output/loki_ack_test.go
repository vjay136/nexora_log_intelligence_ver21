package output

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLokiBlocked260IsRetriedAndNeverAcknowledged(t *testing.T) {
	var blocked atomic.Bool
	blocked.Store(true)
	var calls, accepted atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if blocked.Load() {
			w.WriteHeader(260)
			return
		}
		accepted.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	w, err := NewLokiWriter(Config{URL: server.URL, Synchronous: true, BatchSize: 100, RetryMax: 1, RetryBackoffMS: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err = w.Write([]byte(`{"value":1}`), "normal"); err != nil {
		t.Fatal(err)
	}
	if err = w.Flush(); err == nil {
		t.Fatal("blockedHTTP260 falsely acknowledged")
	}
	if calls.Load() != 2 || accepted.Load() != 0 {
		t.Fatal("blocked push was not retried exactly once")
	}
	blocked.Store(false)
	if err = w.Flush(); err != nil {
		t.Fatal(err)
	}
	if accepted.Load() != 1 {
		t.Fatal("unacknowledged batch was lost instead of retained for retry")
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLokiUnexpected202IsNotAcknowledged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) }))
	defer server.Close()
	w, err := NewLokiWriter(Config{URL: server.URL, Synchronous: true, BatchSize: 100, RetryMax: 1, RetryBackoffMS: 1})
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Write([]byte(`{"value":1}`), "normal")
	if err = w.Flush(); err == nil {
		t.Fatal("unexpected asynchronous status acknowledged as completed Loki push")
	}
	_ = w.Close()
}

func TestLokiIdenticalConcurrentOccurrencesHaveDistinctMonotonicTimestamps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer server.Close()
	w, err := NewLokiWriter(Config{URL: server.URL, Synchronous: true, BatchSize: 1000})
	if err != nil {
		t.Fatal(err)
	}
	w.now = func() time.Time { return time.Unix(1700000000, 0) }
	var wg sync.WaitGroup
	for i := 0; i < 49; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := w.Write([]byte(`{"message":"identical"}`), "normal"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if len(w.batch) != 49 {
		t.Fatal("legitimate identical occurrences collapsed")
	}
	for i := 1; i < len(w.batch); i++ {
		if w.batch[i].ts <= w.batch[i-1].ts {
			t.Fatal("timestamp collision or ordering regression")
		}
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
}
