package server

import (
	"sync/atomic"
	"time"
)

// Stats tracks service-level counters.
type Stats struct {
	started      time.Time
	totalRequests atomic.Int64
}

// NewStats creates a Stats instance stamped with the current time.
func NewStats() *Stats {
	return &Stats{started: time.Now()}
}

// Snapshot is a JSON-serializable view of current stats.
type Snapshot struct {
	TotalRequests int64            `json:"totalRequests"`
	UptimeSeconds float64          `json:"uptimeSeconds"`
	PerPath       map[string]int64 `json:"perPath"`
}

// RecordRequest increments request counters for the given API path.
func (s *Stats) RecordRequest(path string) {
	s.totalRequests.Add(1)
}

// Reset clears accumulated request counters.
func (s *Stats) Reset() {
	s.totalRequests.Store(0)
}

// Snapshot returns the current stats view.
func (s *Stats) Snapshot(perPath map[string]int64) Snapshot {
	return Snapshot{
		TotalRequests: s.totalRequests.Load(),
		UptimeSeconds: time.Since(s.started).Seconds(),
		PerPath:       perPath,
	}
}
