// Package executor changes scheduling without changing the useful task algorithms.
package executor

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"threading-models/internal/task"
)

type Options struct {
	Mode           string
	Workers, Queue int
}

type Summary struct {
	Tasks       int        `json:"tasks_completed"`
	InputBytes  int64      `json:"input_bytes"`
	OutputBytes int64      `json:"compressed_bytes"`
	Stats       task.Stats `json:"stats"`
}

type Report struct {
	Results []task.Result
	Summary Summary
	Elapsed time.Duration
}

func summarize(results []task.Result) Summary {
	s := Summary{Tasks: len(results)}
	for _, r := range results {
		s.InputBytes += int64(r.InputBytes)
		s.OutputBytes += int64(len(r.Compressed))
		s.Stats.Add(r.Stats)
	}
	return s
}

func Run(parent context.Context, options Options, tasks []task.Task) (Report, error) {
	var report Report
	if options.Mode != "seq" && options.Mode != "mn" && options.Mode != "pinned" {
		return report, fmt.Errorf("invalid mode")
	}
	if options.Workers < 1 || options.Workers > 1024 || options.Queue < 0 || options.Queue > 1_000_000 {
		return report, fmt.Errorf("invalid executor parameters")
	}
	for _, t := range tasks {
		if t == nil {
			return report, fmt.Errorf("nil task")
		}
	}
	if err := parent.Err(); err != nil {
		return report, err
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	report.Results = make([]task.Result, len(tasks))
	if options.Mode == "seq" {
		start := time.Now()
		for i, t := range tasks {
			if err := ctx.Err(); err != nil {
				report.Elapsed = time.Since(start)
				return report, err
			}
			result, err := t.Run(ctx)
			if err != nil {
				report.Elapsed = time.Since(start)
				return report, fmt.Errorf("task index %d: %w", i, err)
			}
			report.Results[i] = result
		}
		report.Summary = summarize(report.Results)
		report.Elapsed = time.Since(start)
		return report, ctx.Err()
	}
	jobs := make(chan int, options.Queue)
	startSignal, release := make(chan struct{}), make(chan struct{})
	errors := make(chan error, 1)
	var ready, finished, exited sync.WaitGroup
	ready.Add(options.Workers)
	finished.Add(options.Workers)
	exited.Add(options.Workers)
	for range options.Workers {
		go func() {
			defer exited.Done()
			if options.Mode == "pinned" {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
			}
			ready.Done()
			<-startSignal
			func() {
				defer finished.Done()
				for {
					if ctx.Err() != nil {
						return
					}
					select {
					case <-ctx.Done():
						return
					case i, ok := <-jobs:
						if !ok {
							return
						}
						result, err := tasks[i].Run(ctx)
						if err != nil {
							select {
							case errors <- fmt.Errorf("task index %d: %w", i, err):
							default:
							}
							cancel()
							return
						}
						report.Results[i] = result
					}
				}
			}()
			// Keep pinned threads alive until timing stops; teardown is outside the interval.
			<-release
		}()
	}
	ready.Wait()
	start := time.Now()
	close(startSignal)
feed:
	for i := range tasks {
		select {
		case <-ctx.Done():
			break feed
		case jobs <- i:
		}
	}
	close(jobs)
	finished.Wait()
	if ctx.Err() == nil {
		report.Summary = summarize(report.Results)
	}
	report.Elapsed = time.Since(start)
	close(release)
	exited.Wait()
	select {
	case err := <-errors:
		return report, err
	default:
		return report, ctx.Err()
	}
}
