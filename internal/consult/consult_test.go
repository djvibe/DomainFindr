package consult

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djvibe/domainfindr/internal/model"
)

func TestParseBriefExtractsStructuredFields(t *testing.T) {
	t.Parallel()

	brief := ParseBrief(`# Spa Atlas

- Business model: directory for premium spas
- Launch wedge: British Columbia spa discovery
- Long-term ambition: wellness platform
- Preferred naming styles: authority, startup, directory
- Geography: Canada
- Budget: under $2,000
- Registrar preference: GoDaddy
- Notes: avoid weak prefixed .com names
`)

	if brief.Title != "Spa Atlas" {
		t.Fatalf("unexpected title: %q", brief.Title)
	}
	if brief.BusinessModel != "directory for premium spas" {
		t.Fatalf("unexpected business model: %q", brief.BusinessModel)
	}
	if len(brief.PreferredNamingStyle) != 3 {
		t.Fatalf("unexpected styles: %#v", brief.PreferredNamingStyle)
	}
	if brief.RegistrarPreference != "GoDaddy" {
		t.Fatalf("unexpected registrar preference: %q", brief.RegistrarPreference)
	}
}

func TestCreateSessionWritesConsultantArtifacts(t *testing.T) {
	root := t.TempDir()

	artifacts, err := CreateSession(
		Brief{
			Title:               "Spa Atlas",
			BusinessModel:       "wellness directory",
			LaunchWedge:         "BC spa discovery",
			RegistrarPreference: "GoDaddy",
		},
		SessionOptions{
			SessionName:       "Spa Atlas Consultant",
			Styles:            []string{"authority", "startup"},
			ProviderNames:     []string{"godaddy"},
			GeneratedAt:       time.Date(2026, 4, 3, 11, 0, 0, 0, time.FixedZone("PDT", -7*60*60)),
			ResultsRoot:       root,
			BriefPath:         "brief.md",
			CandidateInput:    "candidates.md",
			PositionalDomains: []string{"spaatlas.ai"},
		},
		[]model.Result{
			{Domain: "spaatlas.ai", Status: model.StatusStandardAvailable, VerificationProvider: "godaddy"},
			{Domain: "spaatlas.com", Status: model.StatusRegistered, VerificationProvider: "rdap"},
		},
	)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if _, err := os.Stat(artifacts.Dir); err != nil {
		t.Fatalf("expected session dir: %v", err)
	}

	executive, err := os.ReadFile(artifacts.ExecutivePath)
	if err != nil {
		t.Fatalf("ReadFile(executive) error = %v", err)
	}
	if !strings.Contains(string(executive), "Buy-now shortlist") {
		t.Fatalf("expected shortlist in executive summary, got:\n%s", executive)
	}

	availability, err := os.ReadFile(artifacts.AvailabilityPath)
	if err != nil {
		t.Fatalf("ReadFile(availability) error = %v", err)
	}
	if !strings.Contains(string(availability), "| spaatlas.ai | standard_available | godaddy | - |") {
		t.Fatalf("unexpected availability report:\n%s", availability)
	}

	if filepath.Base(artifacts.Dir) != "spa-atlas-consultant-20260403" {
		t.Fatalf("unexpected session dir name: %q", filepath.Base(artifacts.Dir))
	}
}
