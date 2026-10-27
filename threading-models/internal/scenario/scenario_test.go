package scenario_test

import (
	"context"
	"testing"
	"threading-models/internal/config"
	"threading-models/internal/scenario"
)

func TestSeedsAndComposition(t *testing.T) {
	cfg := config.Default()
	cfg.Scenario = "compress"
	cfg.Tasks = 20
	cfg.BlockBytes = 128
	cfg.HeavyBlockBytes = 512
	cfg.HeavyPercent = 30
	a, err := scenario.Prepare(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := scenario.Prepare(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if a.Fingerprint != b.Fingerprint || a.HeavyTasks != 6 || a.InputBytes != 14*128+6*512 {
		t.Fatal("seed or composition is not reproducible")
	}
	cfg.OrderSeed++
	c, err := scenario.Prepare(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if a.Fingerprint == c.Fingerprint || a.InputBytes != c.InputBytes || a.HeavyTasks != c.HeavyTasks {
		t.Fatal("order seed must change order while preserving composition")
	}
}
