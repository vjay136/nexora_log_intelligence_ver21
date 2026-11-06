package controlclient

import (
	"crypto/sha256"
	"encoding/json"
	"nexora.local/nli-v20/telemetry"
	"sort"
	"time"
)

func (c Config) telemetryInterval() time.Duration {
	if c.TelemetryPollMS > 0 {
		return time.Duration(c.TelemetryPollMS) * time.Millisecond
	}
	if c.TelemetryPollSeconds > 0 {
		return time.Duration(c.TelemetryPollSeconds) * time.Second
	}
	return 250 * time.Millisecond
}

// Timestamp-only refreshes do not need a network publication. Full cumulative
// counter snapshots preserve exact totals through coalescing and retries.
func streamSnapshotDigest(files []telemetry.StreamFileTelemetry) [32]byte {
	stable := append([]telemetry.StreamFileTelemetry(nil), files...)
	for i := range stable {
		f := &stable[i]
		f.UpdatedAt = time.Time{}
		f.CollectedAt = time.Time{}
		f.PublishedAt = time.Time{}
		f.ControllerReceivedAt = time.Time{}
		f.Sequence = 0
		f.FreshnessAgeMS = 0
		f.ControllerProcessingMS = 0
		f.ObservedPropagationDelayMS = nil
	}
	sort.Slice(stable, func(i, j int) bool { return stable[i].Source < stable[j].Source })
	data, _ := json.Marshal(stable)
	return sha256.Sum256(data)
}
