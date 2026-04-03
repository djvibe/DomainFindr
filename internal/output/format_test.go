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
		{
			Domain:               "example.com",
			Available:            model.BoolPtr(false),
			Status:               model.StatusRegistered,
			Source:               model.SourceRDAP,
			RegistryStatus:       model.StatusRegistered,
			VerificationProvider: model.ProviderRDAP,
			Verifications: []model.Check{
				{Provider: model.ProviderRDAP, Source: model.SourceRDAP, Status: model.StatusRegistered, Available: model.BoolPtr(false)},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteResults(&buf, "csv", results); err != nil {
		t.Fatalf("WriteResults() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "domain,available,status,source,registry_status,registrar_status,pricing_class,price,currency,registration_period,verification_provider,error") {
		t.Fatalf("unexpected csv output: %s", output)
	}
	if !strings.Contains(output, "example.com,false,registered,rdap,registered,,,,,,RDAP,,registered") {
		t.Fatalf("unexpected csv row: %s", output)
	}
}

func TestWriteResultsJSON(t *testing.T) {
	t.Parallel()

	results := []model.Result{
		{
			Domain:               "example.com",
			Available:            model.BoolPtr(true),
			Status:               model.StatusStandardAvailable,
			Source:               model.SourceHTTP,
			RegistryStatus:       model.StatusAvailable,
			RegistrarStatus:      model.StatusStandardAvailable,
			PricingClass:         "standard",
			Price:                model.Float64Ptr(12.99),
			Currency:             "USD",
			RegistrationPeriod:   model.IntPtr(1),
			VerificationProvider: "test_registrar",
			Verifications: []model.Check{
				{Provider: model.ProviderRDAP, Source: model.SourceRDAP, Status: model.StatusAvailable, Available: model.BoolPtr(true)},
				{Provider: "test_registrar", Source: model.SourceHTTP, Status: model.StatusStandardAvailable, Available: model.BoolPtr(true), Price: model.Float64Ptr(12.99), Currency: "USD", RegistrationPeriod: model.IntPtr(1)},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteResults(&buf, "json", results); err != nil {
		t.Fatalf("WriteResults() error = %v", err)
	}

	if !strings.Contains(buf.String(), `"status": "standard_available"`) {
		t.Fatalf("unexpected json output: %s", buf.String())
	}
	if !strings.Contains(buf.String(), `"price": 12.99`) {
		t.Fatalf("expected pricing output: %s", buf.String())
	}
}

func TestWriteResultsTable(t *testing.T) {
	t.Parallel()

	results := []model.Result{
		{
			Domain:               "example.com",
			Available:            model.BoolPtr(false),
			Status:               model.StatusUnavailable,
			Source:               model.SourceHTTP,
			RegistryStatus:       model.StatusAvailable,
			RegistrarStatus:      model.StatusUnavailable,
			VerificationProvider: "test_registrar",
			Verifications: []model.Check{
				{Provider: model.ProviderRDAP, Source: model.SourceRDAP, Status: model.StatusAvailable, Available: model.BoolPtr(true)},
				{Provider: "test_registrar", Source: model.SourceHTTP, Status: model.StatusUnavailable, Available: model.BoolPtr(false)},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteResults(&buf, "table", results); err != nil {
		t.Fatalf("WriteResults() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "TEST_REGISTRAR") || !strings.Contains(output, "example.com") || !strings.Contains(output, "unavailable") {
		t.Fatalf("unexpected table output: %s", output)
	}
}
