package experiment_test

import (
	"context"
	"testing"
	"threading-models/internal/config"
	"threading-models/internal/executor"
	"threading-models/internal/experiment"
)

func TestModesProcessIdenticalWork(t *testing.T) {
	for _, kind := range []string{"logs", "compress"} {
		t.Run(kind, func(t *testing.T) {
			cfg := config.Default()
			cfg.Scenario = kind
			cfg.Tasks = 10
			cfg.RecordsPerTask = 13
			cfg.BlockBytes = 1024
			cfg.HeavyBlockBytes = 4096
			cfg.HeavyPercent = 30
			cfg.Procs = 2
			cfg.Workers = 7
			cfg.Queue = 0
			var fingerprint string
			var summary executor.Summary
			for i, mode := range []string{"seq", "mn", "pinned"} {
				cfg.Mode = mode
				record, err := experiment.Run(context.Background(), cfg)
				if err != nil {
					t.Fatalf("%s: %v", mode, err)
				}
				if !record.Verified || record.Status != "ok" || record.Summary.Tasks != 10 {
					t.Fatalf("%s: incomplete result: %+v", mode, record)
				}
				if i == 0 {
					fingerprint = record.Fingerprint
					summary = record.Summary
				}
				if record.Fingerprint != fingerprint || record.Summary != summary {
					t.Fatalf("%s changed workload or useful output", mode)
				}
			}
		})
	}
}

func TestRejectInvalidConfig(t *testing.T) {
	cfg := config.Default()
	cfg.BlockBytes = 0
	record, err := experiment.Run(context.Background(), cfg)
	if err == nil || record.Verified || record.Status != "failed" || record.Error == "" {
		t.Fatalf("invalid config reported as success: %+v, %v", record, err)
	}
}
