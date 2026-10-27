package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"threading-models/internal/config"
	"threading-models/internal/experiment"
	"threading-models/internal/output"
)

func main() { os.Exit(run()) }

func run() int {
	cfg := config.Default()
	flag.StringVar(&cfg.Scenario, "scenario", cfg.Scenario, "logs or compress")
	flag.StringVar(&cfg.Mode, "mode", cfg.Mode, "seq, mn or pinned")
	flag.IntVar(&cfg.Workers, "workers", cfg.Workers, "number of persistent worker goroutines")
	flag.IntVar(&cfg.Procs, "procs", cfg.Procs, "explicit GOMAXPROCS")
	flag.IntVar(&cfg.Queue, "queue", cfg.Queue, "queue capacity (zero means unbuffered)")
	flag.IntVar(&cfg.Tasks, "tasks", cfg.Tasks, "number of tasks")
	flag.IntVar(&cfg.RecordsPerTask, "records", cfg.RecordsPerTask, "records per log task")
	flag.IntVar(&cfg.BlockBytes, "block-bytes", cfg.BlockBytes, "light compression block size")
	flag.IntVar(&cfg.HeavyBlockBytes, "heavy-block-bytes", cfg.HeavyBlockBytes, "heavy compression block size")
	flag.IntVar(&cfg.HeavyPercent, "heavy-percent", cfg.HeavyPercent, "heavy compression tasks by count, rounded down")
	flag.IntVar(&cfg.Level, "level", cfg.Level, "DEFLATE compression level (-2 through 9)")
	flag.Uint64Var(&cfg.DataSeed, "data-seed", cfg.DataSeed, "PCG data seed")
	flag.Uint64Var(&cfg.OrderSeed, "order-seed", cfg.OrderSeed, "PCG task order seed")
	flag.DurationVar(&cfg.Wait, "wait", cfg.Wait, "modeled timer wait after each task, e.g. 1ms")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	record, err := experiment.Run(ctx, cfg)
	if writeErr := output.Write(os.Stdout, record); writeErr != nil {
		fmt.Fprintln(os.Stderr, "write result:", writeErr)
		return 1
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
