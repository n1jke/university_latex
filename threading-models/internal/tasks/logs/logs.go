// Package logs parses records formatted as service,status,latency_us,bytes.
package logs

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"threading-models/internal/task"
)

type Config struct{ Wait time.Duration }

type Task struct {
	cfg   Config
	input []byte
}

// New borrows immutable input for the lifetime of the task.
func New(cfg Config, input []byte) (*Task, error) {
	if cfg.Wait < 0 {
		return nil, fmt.Errorf("negative wait duration")
	}
	if len(input) == 0 {
		return nil, fmt.Errorf("empty log packet")
	}
	return &Task{cfg: cfg, input: input}, nil
}

func (t *Task) Run(ctx context.Context) (task.Result, error) {
	result := task.Result{Kind: "logs", InputBytes: len(t.input)}
	for line := range bytes.Lines(t.input) {
		if err := ctx.Err(); err != nil {
			return task.Result{}, err
		}
		fields := bytes.Split(bytes.TrimSuffix(line, []byte{'\n'}), []byte{','})
		if len(fields) != 4 {
			return task.Result{}, fmt.Errorf("expected four log fields")
		}
		values := [4]uint64{}
		for i, field := range fields {
			value, err := strconv.ParseUint(string(field), 10, 32)
			if err != nil {
				return task.Result{}, fmt.Errorf("field %d: %w", i, err)
			}
			values[i] = value
		}
		result.Stats.Records++
		if values[1] >= 500 {
			result.Stats.Errors++
		}
		result.Stats.LatencySum += values[2]
		result.Stats.Bytes += values[3]
		bucket := min(values[2]/125, 7)
		result.Stats.Histogram[bucket]++
	}
	if err := task.Wait(ctx, t.cfg.Wait); err != nil {
		return task.Result{}, err
	}
	return result, nil
}
