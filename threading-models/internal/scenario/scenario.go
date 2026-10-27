package scenario

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"

	"threading-models/internal/config"
	"threading-models/internal/task"
	compressTask "threading-models/internal/tasks/compress"
	logTask "threading-models/internal/tasks/logs"
)

type Entry struct {
	ID       int
	Class    string
	Task     task.Task
	input    []byte
	expected task.Stats
}

type Scenario struct {
	Entries     []Entry
	Fingerprint string
	InputBytes  int64
	HeavyTasks  int
}

func Prepare(ctx context.Context, cfg config.Config) (*Scenario, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	rng := rand.New(rand.NewPCG(cfg.DataSeed, 0))
	s := &Scenario{Entries: make([]Entry, 0, cfg.Tasks)}
	heavyCount := cfg.Tasks * cfg.HeavyPercent / 100
	if cfg.Scenario == "compress" {
		s.HeavyTasks = heavyCount
	}
	for id := 0; id < cfg.Tasks; id++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		entry := Entry{ID: id, Class: "light"}
		if cfg.Scenario == "logs" {
			var input []byte
			for range cfg.RecordsPerTask {
				service := rng.Uint64N(16)
				status := uint64(200)
				if rng.Uint64N(10) == 0 {
					status = 500
				}
				latency, size := rng.Uint64N(1000), rng.Uint64N(4096)
				for i, value := range []uint64{service, status, latency, size} {
					if i > 0 {
						input = append(input, ',')
					}
					input = strconv.AppendUint(input, value, 10)
				}
				input = append(input, '\n')
				entry.expected.Records++
				if status >= 500 {
					entry.expected.Errors++
				}
				entry.expected.Bytes += size
				entry.expected.LatencySum += latency
				entry.expected.Histogram[min(latency/125, 7)]++
			}
			entry.input = input
			t, err := logTask.New(logTask.Config{Wait: cfg.Wait}, input)
			if err != nil {
				return nil, err
			}
			entry.Task = t
		} else {
			size := cfg.BlockBytes
			if id < heavyCount {
				size = cfg.HeavyBlockBytes
				entry.Class = "heavy"
			}
			entry.input = make([]byte, size)
			// Repeated service messages with controlled noise: not uniform random incompressible bytes.
			pattern := []byte("service=telemetry status=ok duration=125 payload=event ")
			for i := range entry.input {
				if i%64 == 0 {
					entry.input[i] = byte(rng.Uint64N(256))
				} else {
					entry.input[i] = pattern[i%len(pattern)]
				}
			}
			t, err := compressTask.New(compressTask.Config{Level: cfg.Level, Wait: cfg.Wait}, entry.input)
			if err != nil {
				return nil, err
			}
			entry.Task = t
		}
		s.InputBytes += int64(len(entry.input))
		s.Entries = append(s.Entries, entry)
	}
	order := rand.New(rand.NewPCG(cfg.OrderSeed, 1))
	order.Shuffle(len(s.Entries), func(i, j int) { s.Entries[i], s.Entries[j] = s.Entries[j], s.Entries[i] })
	hash := sha256.New()
	for _, entry := range s.Entries {
		_ = binary.Write(hash, binary.LittleEndian, uint64(entry.ID))
		_ = binary.Write(hash, binary.LittleEndian, uint64(len(entry.input)))
		_, _ = hash.Write(entry.input)
	}
	s.Fingerprint = hex.EncodeToString(hash.Sum(nil))
	return s, nil
}

// Verify checks each result at its original queue position, independently of completion order.
func (s *Scenario) Verify(ctx context.Context, results []task.Result) error {
	if len(results) != len(s.Entries) {
		return fmt.Errorf("result count mismatch")
	}
	for i, entry := range s.Entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		result := results[i]
		if result.InputBytes != len(entry.input) {
			return fmt.Errorf("task %d: byte count mismatch", entry.ID)
		}
		switch result.Kind {
		case "logs":
			if entry.expected.Records == 0 || result.Stats != entry.expected {
				return fmt.Errorf("task %d: log statistics mismatch", entry.ID)
			}
		case "compress":
			if entry.expected.Records != 0 {
				return fmt.Errorf("task %d: unexpected result kind", entry.ID)
			}
			reader := flate.NewReader(bytes.NewReader(result.Compressed))
			restored, err := io.ReadAll(io.LimitReader(reader, int64(len(entry.input))+1))
			closeErr := reader.Close()
			if err != nil || closeErr != nil || !bytes.Equal(restored, entry.input) {
				return fmt.Errorf("task %d: decompression verification failed", entry.ID)
			}
		default:
			return fmt.Errorf("task %d: unknown result kind %q", entry.ID, result.Kind)
		}
	}
	return nil
}
