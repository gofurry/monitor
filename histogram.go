package monitor

import (
	"sync/atomic"
	"time"
)

var latencyBoundsNS = [...]uint64{
	uint64(time.Millisecond),
	5 * uint64(time.Millisecond),
	10 * uint64(time.Millisecond),
	25 * uint64(time.Millisecond),
	50 * uint64(time.Millisecond),
	100 * uint64(time.Millisecond),
	250 * uint64(time.Millisecond),
	500 * uint64(time.Millisecond),
	uint64(time.Second),
	2500 * uint64(time.Millisecond),
	5 * uint64(time.Second),
}

const latencyHistogramShards = 32

type latencyHistogram struct {
	buckets [latencyHistogramShards][len(latencyBoundsNS) + 1]atomic.Uint64
	maxNS   atomic.Uint64
}

type latencyWindow struct {
	buckets [len(latencyBoundsNS) + 1]uint64
	maxNS   uint64
}

func (h *latencyHistogram) observe(ns uint64) {
	h.observeSharded(ns, 0)
}

func (h *latencyHistogram) observeSharded(ns, shard uint64) {
	bucket := len(latencyBoundsNS)
	for i, bound := range latencyBoundsNS {
		if ns <= bound {
			bucket = i
			break
		}
	}
	h.buckets[shard&(latencyHistogramShards-1)][bucket].Add(1)
	updateMaxUint64(&h.maxNS, ns)
}

func (h *latencyHistogram) snapshotAndReset() latencyWindow {
	var window latencyWindow
	for shard := range h.buckets {
		for bucket := range h.buckets[shard] {
			window.buckets[bucket] += h.buckets[shard][bucket].Swap(0)
		}
	}
	window.maxNS = h.maxNS.Swap(0)
	return window
}

func (w latencyWindow) percentile(percent uint64) uint64 {
	var total uint64
	for _, count := range w.buckets {
		total += count
	}
	if total == 0 || percent == 0 {
		return 0
	}

	rank := (total*percent + 99) / 100
	var cumulative uint64
	for i, count := range w.buckets {
		cumulative += count
		if cumulative < rank {
			continue
		}
		if i < len(latencyBoundsNS) {
			return latencyBoundsNS[i]
		}
		return w.maxNS
	}
	return w.maxNS
}
