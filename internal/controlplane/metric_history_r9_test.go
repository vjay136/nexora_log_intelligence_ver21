package controlplane

import (
	"encoding/json"
	"net/http"
	"net/url"
	"nexora.local/nli-v20/telemetry"
	"path/filepath"
	"testing"
	"time"
)

func TestR9HistoryAPIReportsEffectiveMinuteBounds(t *testing.T) {
	f := newAPIFixture(t)
	from := time.Now().UTC().Truncate(time.Minute).Add(-2*time.Hour + 30*time.Second)
	to := from.Add(time.Hour)
	if err := f.server.metricHistory.observe([]telemetry.StreamFileTelemetry{{NodeID: "one", Source: "input", ProcessGeneration: "g", ReadLines: 49, Normalized: 45, Quarantined: 4}}, from.Add(20*time.Second)); err != nil {
		t.Fatal(err)
	}
	query := url.Values{"from": {from.Format(time.RFC3339)}, "to": {to.Format(time.RFC3339)}}
	response, data := f.request(http.MethodGet, "/api/v1/metrics/timeseries?"+query.Encode(), nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", response.StatusCode, data)
	}
	var result struct {
		From          string `json:"from"`
		To            string `json:"to"`
		RequestedFrom string `json:"requested_from"`
		Points        []struct {
			Read uint64 `json:"received"`
		} `json:"points"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	var total uint64
	for _, point := range result.Points {
		total += point.Read
	}
	if total != 49 || result.From != from.Truncate(time.Minute).Format(time.RFC3339) || result.To != to.Truncate(time.Minute).Add(time.Minute).Format(time.RFC3339) || result.RequestedFrom != from.Format(time.RFC3339) {
		t.Fatalf("minute-cut query lost records or misreported bounds: %+v", result)
	}
}

func TestR9HistoryAPIClientFilterAndExactCheckpoint(t *testing.T) {
	f := newAPIFixture(t)
	now := time.Now().UTC()
	const large = uint64(9007199254740993)
	if err := f.server.metricHistory.observe([]telemetry.StreamFileTelemetry{
		{NodeID: "one", Source: "input", ProcessGeneration: "g", ReadLines: large, Normalized: large - 2, Quarantined: 2},
		{NodeID: "two", Source: "input", ProcessGeneration: "g", ReadLines: 7, Normalized: 7},
	}, now); err != nil {
		t.Fatal(err)
	}
	response, data := f.request(http.MethodGet, "/api/v1/metrics/timeseries?range=1h&node_id=one", nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", response.StatusCode, data)
	}
	var result struct {
		Canonical struct {
			Scope    string `json:"scope"`
			Baseline map[string]struct {
				Read string `json:"read"`
			} `json:"counter_baseline"`
			Points []struct {
				Read string `json:"read_delta"`
			} `json:"points"`
		} `json:"canonical_history"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.Canonical.Scope != "selected_node" || len(result.Canonical.Baseline) != 1 || result.Canonical.Baseline["one\x00input"].Read != "9007199254740993" {
		t.Fatalf("incorrect client checkpoint: %s", data)
	}
	found := false
	for _, point := range result.Canonical.Points {
		if point.Read == "9007199254740993" {
			found = true
		}
	}
	if !found {
		t.Fatalf("exact client interval omitted: %s", data)
	}
}

func TestR9HistoryMillionRecordsNodeRangesAndCheckpoint(t *testing.T) {
	h, err := newMetricHistory(filepath.Join(t.TempDir(), "history.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Minute)
	old := now.Add(-2 * time.Hour)
	sample := telemetry.StreamFileTelemetry{NodeID: "one", Source: "input", ProcessGeneration: "g", ReadLines: 49, Normalized: 45, Quarantined: 4}
	if err = h.observe([]telemetry.StreamFileTelemetry{sample}, old); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	for batch := 1; batch <= 1000; batch++ {
		sample.ReadLines = 49 + uint64(batch)*1000
		sample.Normalized = 45 + uint64(batch)*900
		sample.Quarantined = 4 + uint64(batch)*100
		if err = h.observe([]telemetry.StreamFileTelemetry{sample}, now.Add(time.Duration(batch)*time.Millisecond)); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("one million aggregated records, 1000 publications: %s", time.Since(start))
	other := telemetry.StreamFileTelemetry{NodeID: "two", Source: "input", ProcessGeneration: "g", ReadLines: 7, Normalized: 7}
	if err = h.observe([]telemetry.StreamFileTelemetry{sample, other}, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	recent, baseline, observed := h.query(now.Add(-time.Hour), now.Add(time.Minute), time.Minute, "one")
	var read, parsed, quarantine uint64
	for _, bucket := range recent {
		read += bucket.Read
		parsed += bucket.Parsed
		quarantine += bucket.Quarantine
	}
	if read != 1000000 || parsed != 900000 || quarantine != 100000 {
		t.Fatalf("recent node counts %d/%d/%d", read, parsed, quarantine)
	}
	if baseline["one\x00input"].Read != 1000049 || len(baseline) != 1 || !observed.Equal(now.Add(2*time.Second)) {
		t.Fatalf("checkpoint %+v at %s", baseline, observed)
	}
	all := h.window(now.Add(-3*time.Hour), now.Add(time.Minute), time.Minute)
	read = 0
	for _, b := range all {
		read += b.Read
	}
	if read != 1000056 {
		t.Fatalf("all-node historical total %d", read)
	}
	second := h.windowNode(now, now.Add(time.Minute), time.Minute, "two")
	if second[0].Read != 7 {
		t.Fatalf("node two %+v", second)
	}
	h.mu.Lock()
	err = h.persistLocked()
	h.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := newMetricHistory(h.path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.windowNode(now, now.Add(time.Minute), time.Minute, "two")[0].Read != 7 {
		t.Fatal("node history lost after restart")
	}
}
