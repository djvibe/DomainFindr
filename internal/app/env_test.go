package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvSetsMissingValues(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("ALPHA=one\nBETA=two\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("loadDotEnv() error = %v", err)
	}
	if got := os.Getenv("ALPHA"); got != "one" {
		t.Fatalf("expected ALPHA to be set, got %q", got)
	}
	if got := os.Getenv("BETA"); got != "two" {
		t.Fatalf("expected BETA to be set, got %q", got)
	}
}

func TestLoadDotEnvPreservesExistingValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("ALPHA=file\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	t.Setenv("ALPHA", "process")

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("loadDotEnv() error = %v", err)
	}
	if got := os.Getenv("ALPHA"); got != "process" {
		t.Fatalf("expected existing ALPHA to win, got %q", got)
	}
}
