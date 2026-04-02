package input

import (
	"strings"
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	t.Parallel()

	data := `
- Example.COM
- invalid domain

| domain |
| --- |
| foo.dev |
| example.com |
`

	entries, err := parseMarkdown(strings.NewReader(data))
	if err != nil {
		t.Fatalf("parseMarkdown() error = %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Domain != "example.com" || !entries[0].Valid {
		t.Fatalf("expected normalized valid example.com, got %#v", entries[0])
	}

	if entries[1].Valid {
		t.Fatalf("expected invalid domain entry, got %#v", entries[1])
	}

	if entries[2].Domain != "foo.dev" {
		t.Fatalf("expected foo.dev, got %#v", entries[2])
	}
}

func TestParseCSVHeaderDetection(t *testing.T) {
	t.Parallel()

	data := "domain,notes\nExample.org,ok\n"
	entries, err := parseCSV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("parseCSV() error = %v", err)
	}

	if len(entries) != 1 || entries[0].Domain != "example.org" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestParseCSVSemicolonFallback(t *testing.T) {
	t.Parallel()

	data := "domain;note\nalpha.io;test\n"
	entries, err := parseCSV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("parseCSV() error = %v", err)
	}

	if len(entries) != 1 || entries[0].Domain != "alpha.io" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestParseDomains(t *testing.T) {
	t.Parallel()

	entries := ParseDomains([]string{"OpenAI.com", "bad domain", "openai.com"})
	if len(entries) != 2 {
		t.Fatalf("expected 2 deduped entries, got %d", len(entries))
	}
	if entries[0].Domain != "openai.com" || !entries[0].Valid {
		t.Fatalf("unexpected first entry: %#v", entries[0])
	}
	if entries[1].Valid {
		t.Fatalf("expected invalid second entry: %#v", entries[1])
	}
}
