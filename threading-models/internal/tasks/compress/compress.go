package compress

import (
	"bytes"
	"compress/flate"
	"context"
	"fmt"
	"time"

	"threading-models/internal/task"
)

type Config struct {
	Level int
	Wait  time.Duration
}
type Task struct {
	cfg   Config
	input []byte
}

// New borrows immutable input. Compression state is allocated inside Run.
func New(cfg Config, input []byte) (*Task, error) {
	if cfg.Level < flate.HuffmanOnly || cfg.Level > flate.BestCompression {
		return nil, fmt.Errorf("invalid compression level %d", cfg.Level)
	}
	if cfg.Wait < 0 || len(input) == 0 {
		return nil, fmt.Errorf("invalid wait or empty block")
	}
	return &Task{cfg: cfg, input: input}, nil
}

func (t *Task) Run(ctx context.Context) (task.Result, error) {
	if err := ctx.Err(); err != nil {
		return task.Result{}, err
	}
	var output bytes.Buffer
	writer, err := flate.NewWriter(&output, t.cfg.Level)
	if err != nil {
		return task.Result{}, err
	}
	for start := 0; start < len(t.input); start += 16 * 1024 {
		if err := ctx.Err(); err != nil {
			_ = writer.Close()
			return task.Result{}, err
		}
		if _, err := writer.Write(t.input[start:min(start+16*1024, len(t.input))]); err != nil {
			_ = writer.Close()
			return task.Result{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return task.Result{}, err
	}
	if err := task.Wait(ctx, t.cfg.Wait); err != nil {
		return task.Result{}, err
	}
	return task.Result{Kind: "compress", InputBytes: len(t.input), Compressed: output.Bytes()}, nil
}
