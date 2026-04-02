package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/djvibe/domainfindr/internal/model"
)

func TestWriteResultsCSV(t *testing.T) {
	t.Parallel()

	results := []model.Result{
		{Domain: "example.com", Available: model.BoolPtr(false), Status: model.StatusRegistered, Source: model.SourceRDAP},
	}

	var buf bytes.Buffer
	if err := WriteResults(&buf, "csv", results); err != nil {
		t.Fatalf("WriteResults() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "domain,available,status,source,error") {
		t.Fatalf("unexpected csv output: %s", output)
	}
	if !strings.Contains(output, "example.com,false,registered,rdap,") {
		t.Fatalf("unexpected csv row: %s", output)
	}
}

func TestWriteResultsJSON(t *testing.T) {
	t.Parallel()

	results := []model.Result{
		{Domain: "example.com", Available: model.BoolPtr(true), Status: model.StatusAvailable, Source: model.SourceRDAP},
	}

	var buf bytes.Buffer
	if err := WriteResults(&buf, "json", results); err != nil {
		t.Fatalf("WriteResults() error = %v", err)
	}

	if !strings.Contains(buf.String(), `"status": "available"`) {
		t.Fatalf("unexpected json output: %s", buf.String())
	}
}

func TestWriteResultsTable(t *testing.T) {
	t.Parallel()

	results := []model.Result{
		{Domain: "example.com", Available: model.BoolPtr(false), Status: model.StatusRegistered, Source: model.SourceRDAP},
	}

	var buf bytes.Buffer
	if err := WriteResults(&buf, "table", results); err != nil {
		t.Fatalf("WriteResults() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "DOMAIN") || !strings.Contains(output, "example.com") {
		t.Fatalf("unexpected table output: %s", output)
	}
}
