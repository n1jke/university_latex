package experiment

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"threading-models/internal/config"
	"threading-models/internal/executor"
	"threading-models/internal/scenario"
	"threading-models/internal/task"
)

type Record struct {
	RunID          string           `json:"run_id"`
	Status         string           `json:"status"`
	Error          string           `json:"error,omitempty"`
	Config         config.Config    `json:"config"`
	GoVersion      string           `json:"go_version"`
	OS             string           `json:"os"`
	Arch           string           `json:"arch"`
	LogicalCPUs    int              `json:"logical_cpus"`
	Generator      string           `json:"generator"`
	Fingerprint    string           `json:"workload_sha256,omitempty"`
	HeavyTasks     int              `json:"heavy_tasks"`
	DurationNS     int64            `json:"duration_ns"`
	DurationMS     float64          `json:"duration_ms"`
	BytesPerSecond float64          `json:"bytes_per_second"`
	Summary        executor.Summary `json:"summary"`
	Verified       bool             `json:"verified"`
}

func Run(ctx context.Context, cfg config.Config) (record Record, err error) {
	record = Record{RunID: time.Now().UTC().Format("20060102T150405.000000000Z"), Status: "failed",
		Config: cfg, GoVersion: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH,
		LogicalCPUs: runtime.NumCPU(), Generator: "math/rand/v2.PCG; data(seed,0); order(seed,1); format=v1"}
	defer func() {
		if err != nil {
			record.Error = err.Error()
		}
	}()
	if err = cfg.Validate(); err != nil {
		return record, err
	}
	oldProcs := runtime.GOMAXPROCS(cfg.Procs)
	defer runtime.GOMAXPROCS(oldProcs)
	prepared, err := scenario.Prepare(ctx, cfg)
	if err != nil {
		return record, fmt.Errorf("prepare: %w", err)
	}
	record.Fingerprint = prepared.Fingerprint
	record.HeavyTasks = prepared.HeavyTasks
	tasks := make([]task.Task, len(prepared.Entries))
	for i, e := range prepared.Entries {
		tasks[i] = e.Task
	}
	report, err := executor.Run(ctx, executor.Options{Mode: cfg.Mode, Workers: cfg.Workers, Queue: cfg.Queue}, tasks)
	record.DurationNS = report.Elapsed.Nanoseconds()
	record.DurationMS = float64(record.DurationNS) / 1e6
	if err != nil {
		return record, fmt.Errorf("execute: %w", err)
	}
	if err = prepared.Verify(ctx, report.Results); err != nil {
		return record, fmt.Errorf("verify: %w", err)
	}
	record.Summary = report.Summary
	if report.Summary.InputBytes != prepared.InputBytes {
		return record, fmt.Errorf("total input size mismatch")
	}
	record.Verified = true
	record.Status = "ok"
	if report.Elapsed > 0 {
		record.BytesPerSecond = float64(prepared.InputBytes) / report.Elapsed.Seconds()
	}
	return record, nil
}
