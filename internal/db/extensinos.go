package db

import "time"

func (e *AddHistoryEntryParams) SetLatency(latency time.Duration) {
	e.LatencyMs = int64(latency / time.Millisecond)
}

func (e *AddHistoryEntryParams) SetJitter(jitter time.Duration) {
	e.JitterMs = int64(jitter / time.Millisecond)
}
