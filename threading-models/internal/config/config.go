package config

import (
	"compress/flate"
	"fmt"
	"runtime"
	"time"
)

type Config struct {
	Scenario        string        `json:"scenario"`
	Mode            string        `json:"mode"`
	Workers         int           `json:"workers"`
	Procs           int           `json:"gomaxprocs"`
	Queue           int           `json:"queue_capacity"`
	Tasks           int           `json:"tasks"`
	RecordsPerTask  int           `json:"records_per_task"`
	BlockBytes      int           `json:"block_bytes"`
	HeavyBlockBytes int           `json:"heavy_block_bytes"`
	HeavyPercent    int           `json:"heavy_percent"`
	Level           int           `json:"compression_level"`
	DataSeed        uint64        `json:"data_seed"`
	OrderSeed       uint64        `json:"order_seed"`
	Wait            time.Duration `json:"wait_ns"`
}

// Default is the hardcoded MVP scenario description. CLI flags override its fields.
func Default() Config {
	return Config{Scenario: "logs", Mode: "mn", Workers: 4, Procs: runtime.NumCPU(), Queue: 16,
		Tasks: 100, RecordsPerTask: 256, BlockBytes: 16 * 1024, HeavyBlockBytes: 256 * 1024,
		HeavyPercent: 10, Level: flate.DefaultCompression, DataSeed: 42, OrderSeed: 2026}
}

func (c Config) Validate() error {
	if c.Scenario != "logs" && c.Scenario != "compress" {
		return fmt.Errorf("unknown scenario %q", c.Scenario)
	}
	if c.Mode != "seq" && c.Mode != "mn" && c.Mode != "pinned" {
		return fmt.Errorf("unknown mode %q", c.Mode)
	}
	if c.Workers < 1 || c.Workers > 1024 || c.Procs < 1 || c.Procs > 1024 {
		return fmt.Errorf("workers and procs must be in [1,1024]")
	}
	if c.Tasks < 1 || c.Tasks > 1_000_000 || c.Queue < 0 || c.Queue > 1_000_000 {
		return fmt.Errorf("invalid task count or queue capacity")
	}
	if c.Wait < 0 || c.Wait > time.Hour {
		return fmt.Errorf("wait must be between zero and one hour")
	}
	if c.RecordsPerTask < 1 || c.BlockBytes < 1 || c.HeavyBlockBytes < c.BlockBytes {
		return fmt.Errorf("invalid packet sizes")
	}
	if c.HeavyPercent < 0 || c.HeavyPercent > 100 {
		return fmt.Errorf("heavy percent must be in [0,100]")
	}
	if c.Level < flate.HuffmanOnly || c.Level > flate.BestCompression {
		return fmt.Errorf("invalid compression level")
	}
	// Reject oversized inputs before allocating; products are checked using division.
	const maxInput = 512 * 1024 * 1024
	if c.Scenario == "logs" {
		if c.RecordsPerTask > maxInput/40/c.Tasks {
			return fmt.Errorf("generated logs exceed input budget (512 MiB)")
		}
	} else {
		heavy := c.Tasks * c.HeavyPercent / 100
		if c.BlockBytes > maxInput/c.Tasks || (heavy > 0 && c.HeavyBlockBytes > maxInput/heavy) {
			return fmt.Errorf("compression input exceeds budget (512 MiB)")
		}
		total := int64(c.Tasks-heavy)*int64(c.BlockBytes) + int64(heavy)*int64(c.HeavyBlockBytes)
		if total > maxInput {
			return fmt.Errorf("compression input exceeds budget (512 MiB)")
		}
	}
	return nil
}
