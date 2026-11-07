package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFieldSenseAdmissionIsolationAndRecovery(t *testing.T) {
	s := &Server{cfg: Config{FieldSenseAnalysisConcurrency: 1, FieldSenseAnalysisTimeoutMS: 50}}
	ctx, done, ok := s.beginFieldSenseAnalysis(httptest.NewRecorder(), httptest.NewRequest("POST", "/", nil))
	if !ok {
		t.Fatal("first analysis rejected")
	}
	busy := httptest.NewRecorder()
	s.handleMappingPreview(busy, httptest.NewRequest("POST", "/", strings.NewReader(`{}`)))
	if busy.Code != http.StatusTooManyRequests || busy.Header().Get("Retry-After") != "1" {
		t.Fatalf("busy analysis: %d", busy.Code)
	}
	kvBusy := httptest.NewRecorder()
	s.handleKVInspect(kvBusy, httptest.NewRequest("POST", "/", strings.NewReader(`{}`)))
	if kvBusy.Code != http.StatusTooManyRequests || kvBusy.Header().Get("Retry-After") != "1" {
		t.Fatalf("KV inspection bypassed shared admission: %d", kvBusy.Code)
	}
	// A deadline frees capacity when the handler exits; no background queue starts.
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("analysis deadline missing")
	}
	done()
	malformed := httptest.NewRecorder()
	s.handleMappingPreview(malformed, httptest.NewRequest("POST", "/", strings.NewReader(`{}`)))
	if malformed.Code != http.StatusBadRequest || len(s.fieldSenseAnalysis().slots) != 0 {
		t.Fatal("invalid input leaked an analysis slot")
	}
	valid := httptest.NewRecorder()
	s.handleMappingPreview(valid, httptest.NewRequest("POST", "/", strings.NewReader(`{"raw":"hello","plan":{"version":1,"fields":[{"name":"message","type":"string","method":"position","source":"1"}]}}`)))
	if valid.Code != 200 || !strings.Contains(valid.Body.String(), `"passed":true`) || !strings.Contains(valid.Body.String(), `"trace"`) {
		t.Fatalf("capacity did not recover: %s", valid.Body.String())
	}
	if s.fieldSenseAnalysis().timedOut.Load() != 1 {
		t.Fatal("deadline metric missing")
	}
}

func TestFieldSenseCanceledRequestCannotOccupyCapacity(t *testing.T) {
	s := &Server{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest("POST", "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	_, _, ok := s.beginFieldSenseAnalysis(w, r)
	if ok || w.Code != http.StatusRequestTimeout || len(s.fieldSenseAnalysis().slots) != 0 {
		t.Fatal("canceled request admitted")
	}
}
