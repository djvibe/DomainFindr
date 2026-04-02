package app

import (
	"bytes"
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
