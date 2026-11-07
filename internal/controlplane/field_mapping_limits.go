package controlplane

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

// Analysis capacity is separate from ingestion, health, fleet and approval work.
// No waiting queue retains operator samples when all analysis slots are busy.
type fieldSenseAdmission struct {
	slots         chan struct{}
	timeout       time.Duration
	accepted      atomic.Int64
	completed     atomic.Int64
	rejected      atomic.Int64
	timedOut      atomic.Int64
	canceled      atomic.Int64
	kvInspections atomic.Int64
	kvNoMatch     atomic.Int64
}

func (s *Server) fieldSenseAnalysis() *fieldSenseAdmission {
	s.fieldSenseOnce.Do(func() {
		capacity := s.cfg.FieldSenseAnalysisConcurrency
		if capacity < 1 || capacity > 8 {
			capacity = 2
		}
		millis := s.cfg.FieldSenseAnalysisTimeoutMS
		if millis < 50 || millis > 10000 {
			millis = 2000
		}
		s.fieldSense = &fieldSenseAdmission{slots: make(chan struct{}, capacity), timeout: time.Duration(millis) * time.Millisecond}
	})
	return s.fieldSense
}

func (s *Server) beginFieldSenseAnalysis(w http.ResponseWriter, r *http.Request) (context.Context, func(), bool) {
	a := s.fieldSenseAnalysis()
	if r.Context().Err() != nil {
		jsonError(w, "FieldSense analysis canceled", http.StatusRequestTimeout)
		return nil, nil, false
	}
	select {
	case a.slots <- struct{}{}:
	default:
		a.rejected.Add(1)
		w.Header().Set("Retry-After", "1")
		jsonError(w, "FieldSense analysis capacity is busy; retry shortly", http.StatusTooManyRequests)
		return nil, nil, false
	}
	a.accepted.Add(1)
	ctx, cancel := context.WithTimeout(r.Context(), a.timeout)
	return ctx, func() {
		if ctx.Err() == context.DeadlineExceeded {
			a.timedOut.Add(1)
		} else if ctx.Err() == context.Canceled {
			a.canceled.Add(1)
		}
		cancel()
		<-a.slots
		a.completed.Add(1)
	}, true
}

func fieldSenseContextError(w http.ResponseWriter, ctx context.Context) bool {
	if ctx.Err() == nil {
		return false
	}
	jsonError(w, "FieldSense analysis interrupted: "+ctx.Err().Error(), http.StatusRequestTimeout)
	return true
}

func (a *fieldSenseAdmission) status() map[string]any {
	return map[string]any{"concurrency": cap(a.slots), "active": len(a.slots), "timeout_ms": a.timeout.Milliseconds(), "queued": 0, "accepted": a.accepted.Load(), "completed": a.completed.Load(), "busy_rejections": a.rejected.Load(), "timed_out": a.timedOut.Load(), "canceled": a.canceled.Load(), "ingestion_independent": true}
}

func (a *fieldSenseAdmission) kvStatus() map[string]any {
	return map[string]any{"engine": "kvie-v1", "minimum_client_release": "21.0.0-10", "inspection_total": a.kvInspections.Load(), "no_match_total": a.kvNoMatch.Load(), "preview_path": "/api/v1/assurance/kv-inspect", "automatic_publication": false}
}
