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
