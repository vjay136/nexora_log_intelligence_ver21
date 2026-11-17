package agent

import (
	"fmt"
	"io"
	"sync/atomic"

	"nexora.local/nli-v20/internal/output"
)

type remoteDeliveryCounter struct {
	active    int
	delivered atomic.Uint64
	failed    atomic.Uint64
}

func (counter *remoteDeliveryCounter) record(count int, delivered bool) {
	if delivered {
		counter.delivered.Add(uint64(count))
	} else {
		counter.failed.Add(uint64(count))
	}
}

type remoteDeliveryMetrics struct {
	loki       remoteDeliveryCounter
	clickhouse remoteDeliveryCounter
}

func instrumentRemoteDelivery(outputs []output.Config) *remoteDeliveryMetrics {
	metrics := &remoteDeliveryMetrics{}
	for i := range outputs {
		var counter *remoteDeliveryCounter
		switch outputs[i].Type {
		case "loki":
			counter = &metrics.loki
		case "clickhouse":
			counter = &metrics.clickhouse
		default:
			continue
		}
		counter.active++
		outputs[i].RecordBatch = counter.record
	}
	return metrics
}

func (metrics *remoteDeliveryMetrics) write(w io.Writer) {
	for _, destination := range []struct {
		name    string
		counter *remoteDeliveryCounter
	}{{"loki", &metrics.loki}, {"clickhouse", &metrics.clickhouse}} {
		fmt.Fprintf(w, "nli_%s_active_writers %d\nnli_%s_delivered_total %d\nnli_%s_failed_delivery_attempts_total %d\n",
			destination.name, destination.counter.active,
			destination.name, destination.counter.delivered.Load(),
			destination.name, destination.counter.failed.Load())
	}
}
