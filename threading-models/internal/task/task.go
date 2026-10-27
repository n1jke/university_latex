// Package task defines useful work independently of its execution model.
package task

import (
	"context"
	"time"
)

type Task interface {
	Run(context.Context) (Result, error)
}

type Stats struct {
	Records    uint64    `json:"records"`
	Errors     uint64    `json:"errors"`
	Bytes      uint64    `json:"bytes"`
	LatencySum uint64    `json:"latency_sum"`
	Histogram  [8]uint64 `json:"histogram"`
}

func (s *Stats) Add(other Stats) {
	s.Records += other.Records
	s.Errors += other.Errors
	s.Bytes += other.Bytes
	s.LatencySum += other.LatencySum
	for i := range s.Histogram {
		s.Histogram[i] += other.Histogram[i]
	}
}

type Result struct {
	Kind       string
	InputBytes int
	Stats      Stats
	Compressed []byte
}

// Wait is a modeled wait, not a substitute for CPU work.
func Wait(ctx context.Context, duration time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if duration == 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
