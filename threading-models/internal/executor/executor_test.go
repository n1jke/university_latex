package executor_test

import (
	"context"
	"errors"
	"testing"
	"threading-models/internal/executor"
	"threading-models/internal/task"
	"time"
)

type functionTask func(context.Context) (task.Result, error)

func (f functionTask) Run(ctx context.Context) (task.Result, error) { return f(ctx) }

func TestCancellationWhileQueueIsFull(t *testing.T) {
	for _, mode := range []string{"seq", "mn", "pinned"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			started := make(chan struct{}, 1)
			waiting := functionTask(func(ctx context.Context) (task.Result, error) {
				select {
				case started <- struct{}{}:
				default:
				}
				<-ctx.Done()
				return task.Result{}, ctx.Err()
			})
			tasks := []task.Task{waiting, waiting, waiting, waiting}
			done := make(chan error, 1)
			go func() {
				_, err := executor.Run(ctx, executor.Options{Mode: mode, Workers: 2, Queue: 0}, tasks)
				done <- err
			}()
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("task never started")
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("unexpected error: %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("executor did not stop")
			}
		})
	}
}

func TestTaskFailureCancelsOtherWorkers(t *testing.T) {
	want := errors.New("useful work failed")
	failing := functionTask(func(context.Context) (task.Result, error) { return task.Result{}, want })
	waiting := functionTask(func(ctx context.Context) (task.Result, error) { <-ctx.Done(); return task.Result{}, ctx.Err() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := executor.Run(ctx, executor.Options{Mode: "pinned", Workers: 2, Queue: 1}, []task.Task{failing, waiting, waiting})
	if !errors.Is(err, want) {
		t.Fatalf("lost task error: %v", err)
	}
}
