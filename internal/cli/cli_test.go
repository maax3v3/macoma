package cli

import (
	"flag"
	"os"
	"testing"
)

func TestParse_LabelReadabilityDefaultTrue(t *testing.T) {
	cfg, err := parseWithArgs(t, []string{
		"macoma",
		"--in=input.png",
		"--out=output.png",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !cfg.LabelReadability {
		t.Fatal("LabelReadability should default to true")
	}
}

func TestParse_LabelReadabilityFalse(t *testing.T) {
	cfg, err := parseWithArgs(t, []string{
		"macoma",
		"--in=input.png",
		"--out=output.png",
		"--label-readability=false",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.LabelReadability {
		t.Fatal("LabelReadability should be false when flag is set")
	}
}

func TestParse_GradientStrategyAndKnobs(t *testing.T) {
	cfg, err := parseWithArgs(t, []string{
		"macoma",
		"--in=input.png",
		"--out=output.png",
		"--delimiter-strategy=gradient",
		"--gradient-blur-sigma=1.5",
		"--gradient-low-threshold=0.06",
		"--gradient-high-threshold=0.18",
		"--gradient-close-radius=2",
		"--gradient-min-component-size=30",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.DelimiterStrategy != StrategyGradient {
		t.Fatalf("DelimiterStrategy = %q, want %q", cfg.DelimiterStrategy, StrategyGradient)
	}
	if cfg.GradientBlurSigma != 1.5 || cfg.GradientLowThreshold != 0.06 || cfg.GradientHighThreshold != 0.18 {
		t.Fatalf("unexpected gradient thresholds: %+v", cfg)
	}
	if cfg.GradientCloseRadius != 2 || cfg.GradientMinComponentSize != 30 {
		t.Fatalf("unexpected gradient morphology params: %+v", cfg)
	}
}

func parseWithArgs(t *testing.T, args []string) (Config, error) {
	t.Helper()

	origArgs := os.Args
	origFlagSet := flag.CommandLine
	defer func() {
		os.Args = origArgs
		flag.CommandLine = origFlagSet
	}()

	os.Args = args
	flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
	return Parse()
}
