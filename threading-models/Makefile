GO ?= go
SCENARIO ?= logs
MODE ?= mn
WORKERS ?= 4
# Empty PROCS preserves the program default: runtime.NumCPU().
PROCS ?=
QUEUE ?= 16
TASKS ?= 100
RECORDS ?= 256
BLOCK_BYTES ?= 16384
HEAVY_BLOCK_BYTES ?= 262144
HEAVY_PERCENT ?= 10
LEVEL ?= -1
DATA_SEED ?= 42
ORDER_SEED ?= 2026
WAIT ?= 0s
REPEATS ?= 3
RESULTS_DIR ?= results

RUN_ARGS = -scenario "$(SCENARIO)" -workers "$(WORKERS)" \
	$(if $(PROCS),-procs "$(PROCS)") -queue "$(QUEUE)" \
	-tasks "$(TASKS)" -records "$(RECORDS)" \
	-block-bytes "$(BLOCK_BYTES)" -heavy-block-bytes "$(HEAVY_BLOCK_BYTES)" \
	-heavy-percent "$(HEAVY_PERCENT)" -level "$(LEVEL)" \
	-data-seed "$(DATA_SEED)" -order-seed "$(ORDER_SEED)" -wait "$(WAIT)"

.DEFAULT_GOAL := help
.PHONY: help build build-linux run run-seq run-mn run-pinned compare cli-help fmt test race vet check

help:
	@printf '%s\n' \
	  'make build / build-linux  Build native / Linux amd64 binary' \
	  'make run                 Run once (SCENARIO=logs MODE=mn)' \
	  'make run-seq/run-mn/run-pinned  Run a specific mode' \
	  'make compare             Run three modes, REPEATS=3, save fresh JSONL' \
	  'make cli-help            Show program flags' \
	  'make check               Run vet, tests and race detector' \
	  'make fmt                 Format Go sources' \
	  'Example: make run SCENARIO=compress MODE=pinned WORKERS=16 PROCS=4' \
	  'See USAGE.md for parameters, measurements and result fields.'

build:
	@$(GO) build -o bin/experiment ./cmd/experiment

build-linux:
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o bin/experiment-linux-amd64 ./cmd/experiment

run: build
	@./bin/experiment -mode "$(MODE)" $(RUN_ARGS)

run-seq: build
	@./bin/experiment -mode seq $(RUN_ARGS)

run-mn: build
	@./bin/experiment -mode mn $(RUN_ARGS)

run-pinned: build
	@./bin/experiment -mode pinned $(RUN_ARGS)

# Every invocation creates a new file; each measurement runs in a fresh process.
# Rotate mode order between repetitions. Keep original records if a later run fails.
compare: build
	@set -eu; \
	case "$(REPEATS)" in ''|*[!0-9]*|0) echo 'REPEATS must be a positive integer' >&2; exit 2;; esac; \
	mkdir -p "$(RESULTS_DIR)"; \
	result_file=$$(mktemp "$(RESULTS_DIR)/compare.XXXXXX"); \
	printf 'Results (JSONL): %s\n' "$$result_file"; \
	repeat=1; \
	while [ "$$repeat" -le "$(REPEATS)" ]; do \
	  case $$((repeat % 3)) in \
	    1) modes='seq mn pinned';; \
	    2) modes='mn pinned seq';; \
	    0) modes='pinned seq mn';; \
	  esac; \
	  for mode in $$modes; do \
	    ./bin/experiment -mode "$$mode" $(RUN_ARGS) >> "$$result_file"; \
	  done; \
	  repeat=$$((repeat + 1)); \
	done

cli-help: build
	@./bin/experiment -help

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

check: vet test race
