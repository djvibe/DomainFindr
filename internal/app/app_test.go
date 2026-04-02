package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFlagsAcceptsPositionalDomains(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	cfg, err := parseFlags([]string{"openai.com", "example.com"}, stderr)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config")
	}
	if len(cfg.Domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(cfg.Domains))
	}
}

func TestParseFlagsRequiresInputOrDomains(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	_, err := parseFlags(nil, stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "provide --input FILE or one or more domains") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareHistoryDefaults(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", root)

	history, err := prepareHistory(&Config{Format: "json"})
	if err != nil {
		t.Fatalf("prepareHistory() error = %v", err)
	}
	defer closeQuietly(history.logFile)
	defer closeQuietly(history.outputFile)

	if history.dir == "" {
		t.Fatal("expected history directory")
	}
	if _, err := os.Stat(history.dir); err != nil {
		t.Fatalf("expected history dir to exist: %v", err)
	}
	if history.logFile == nil {
		t.Fatal("expected log file")
	}
	if history.outputFile == nil {
		t.Fatal("expected output file")
	}
}

func TestPrepareHistoryNoHistoryWithLogFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "run.log")
	history, err := prepareHistory(&Config{NoHistory: true, LogFile: logPath})
	if err != nil {
		t.Fatalf("prepareHistory() error = %v", err)
	}
	defer closeQuietly(history.logFile)

	if history.dir != "" {
		t.Fatalf("expected no history dir, got %q", history.dir)
	}
	if history.logFile == nil {
		t.Fatal("expected log file")
	}
	if history.outputFile != nil {
		t.Fatal("expected no output file")
	}
}
