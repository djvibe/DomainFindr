package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djvibe/domainfindr/internal/model"
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

func TestParseFlagsAcceptsRegistrarVerificationConfig(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	cfg, err := parseFlags([]string{
		"--registrar-base-url", "https://registrar.test",
		"--registrar-provider", "test_registrar",
		"--registrar-retry", "3",
		"--registrar-timeout", "4s",
		"--registrar-recheck", "2",
		"example.com",
	}, stderr)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}

	if cfg.RegistrarBaseURL != "https://registrar.test" {
		t.Fatalf("unexpected registrar base url: %#v", cfg)
	}
	if cfg.RegistrarProvider != "test_registrar" {
		t.Fatalf("unexpected registrar provider: %#v", cfg)
	}
	if cfg.RegistrarRetry != 3 || cfg.RegistrarTimeout != 4*time.Second || cfg.RegistrarRecheck != 2 {
		t.Fatalf("unexpected registrar hardening config: %#v", cfg)
	}
}

func TestParseConsultFlagsRequiresBrief(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	_, err := parseConsultFlags(nil, stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "provide --brief FILE") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseConsultFlagsAcceptsStylesAndDomains(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	cfg, err := parseConsultFlags([]string{
		"--brief", "brief.md",
		"--style", "authority,startup,authority",
		"--provider", "godaddy,namecheap",
		"spaatlas.ai",
	}, stderr)
	if err != nil {
		t.Fatalf("parseConsultFlags() error = %v", err)
	}

	if cfg.Brief != "brief.md" {
		t.Fatalf("unexpected brief: %#v", cfg)
	}
	if len(cfg.Styles) != 2 || cfg.Styles[0] != "authority" || cfg.Styles[1] != "startup" {
		t.Fatalf("unexpected styles: %#v", cfg.Styles)
	}
	if len(cfg.RegistrarProviders) != 2 {
		t.Fatalf("unexpected providers: %#v", cfg.RegistrarProviders)
	}
	if len(cfg.Domains) != 1 || cfg.Domains[0] != "spaatlas.ai" {
		t.Fatalf("unexpected domains: %#v", cfg.Domains)
	}
}

func TestParseFlagsDefaultsToGoDaddyWhenCredentialsPresent(t *testing.T) {
	t.Setenv("GODADDY_API_KEY", "key")
	t.Setenv("GODADDY_API_SECRET", "secret")

	stderr := &bytes.Buffer{}
	cfg, err := parseFlags([]string{"example.com"}, stderr)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	if len(cfg.RegistrarProviders) != 1 || cfg.RegistrarProviders[0] != "godaddy" {
		t.Fatalf("expected godaddy provider, got %#v", cfg)
	}
}

func TestParseFlagsResolvesOTEGoDaddyCredentials(t *testing.T) {
	t.Setenv("GODADDY_API_ENV", "ote")
	t.Setenv("GODADDY_OTE_API_KEY", "ote-key")
	t.Setenv("GODADDY_OTE_API_SECRET", "ote-secret")
	t.Setenv("GODADDY_OTE_API_BASE_URL", "https://ote.example.test")

	stderr := &bytes.Buffer{}
	cfg, err := parseFlags([]string{"example.com"}, stderr)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	if cfg.GoDaddyAPIKey != "ote-key" || cfg.GoDaddyAPISecret != "ote-secret" {
		t.Fatalf("expected OTE credentials, got %#v", cfg)
	}
	if cfg.RegistrarBaseURL != "https://ote.example.test" {
		t.Fatalf("expected OTE base URL, got %#v", cfg)
	}
}

func TestParseFlagsResolvesProductionGoDaddyBaseURL(t *testing.T) {
	t.Setenv("GODADDY_API_ENV", "prd")
	t.Setenv("GODADDY_PRD_API_KEY", "prd-key")
	t.Setenv("GODADDY_PRD_API_SECRET", "prd-secret")
	t.Setenv("GODADDY_PRD_API_BASE_URL", "https://prd.example.test")

	stderr := &bytes.Buffer{}
	cfg, err := parseFlags([]string{"example.com"}, stderr)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	if cfg.GoDaddyAPIKey != "prd-key" || cfg.GoDaddyAPISecret != "prd-secret" {
		t.Fatalf("expected PRD credentials, got %#v", cfg)
	}
	if cfg.RegistrarBaseURL != "https://prd.example.test" {
		t.Fatalf("expected PRD base URL, got %#v", cfg)
	}
}

func TestParseFlagsDefaultsToNamecheapWhenCredentialsPresent(t *testing.T) {
	t.Setenv("NAMECHEAP_API_USER", "api-user")
	t.Setenv("NAMECHEAP_API_KEY", "api-key")
	t.Setenv("NAMECHEAP_CLIENT_IP", "203.0.113.10")

	stderr := &bytes.Buffer{}
	cfg, err := parseFlags([]string{"example.com"}, stderr)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	if len(cfg.RegistrarProviders) != 1 || cfg.RegistrarProviders[0] != "namecheap" {
		t.Fatalf("expected namecheap provider, got %#v", cfg)
	}
}

func TestParseFlagsResolvesSandboxNamecheapCredentials(t *testing.T) {
	t.Setenv("NAMECHEAP_API_ENV", "sandbox")
	t.Setenv("NAMECHEAP_SANDBOX_API_USER", "sandbox-user")
	t.Setenv("NAMECHEAP_SANDBOX_API_KEY", "sandbox-key")
	t.Setenv("NAMECHEAP_SANDBOX_USERNAME", "sandbox-login")
	t.Setenv("NAMECHEAP_SANDBOX_CLIENT_IP", "203.0.113.10")
	t.Setenv("NAMECHEAP_SANDBOX_API_BASE_URL", "https://sandbox.example.test/xml.response")

	stderr := &bytes.Buffer{}
	cfg, err := parseFlags([]string{"example.com"}, stderr)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	if cfg.NamecheapAPIUser != "sandbox-user" || cfg.NamecheapAPIKey != "sandbox-key" {
		t.Fatalf("expected sandbox namecheap credentials, got %#v", cfg)
	}
	if cfg.NamecheapUserName != "sandbox-login" {
		t.Fatalf("expected sandbox username, got %#v", cfg)
	}
	if cfg.NamecheapClientIP != "203.0.113.10" {
		t.Fatalf("expected sandbox client ip, got %#v", cfg)
	}
	if cfg.RegistrarBaseURL != "https://sandbox.example.test/xml.response" {
		t.Fatalf("expected sandbox base url, got %#v", cfg)
	}
}

func TestProviderSummaryKeyIncludesEnvironmentForRegistrars(t *testing.T) {
	t.Parallel()

	if got := providerSummaryKey("godaddy", "ote"); got != "GoDaddy (ote)" {
		t.Fatalf("unexpected provider summary key: %q", got)
	}
	if got := providerSummaryKey("rdap", "registry"); got != "RDAP" {
		t.Fatalf("unexpected rdap provider summary key: %q", got)
	}
}

func TestParseFlagsAcceptsBudgetAndSortOptions(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	cfg, err := parseFlags([]string{
		"--budget-min", "10",
		"--budget-max", "100",
		"--sort", "price",
		"--only-standard-price",
		"example.com",
	}, stderr)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	if !cfg.HasBudgetMin || cfg.BudgetMin != 10 {
		t.Fatalf("unexpected budget min: %#v", cfg)
	}
	if !cfg.HasBudgetMax || cfg.BudgetMax != 100 {
		t.Fatalf("unexpected budget max: %#v", cfg)
	}
	if cfg.Sort != "price" {
		t.Fatalf("unexpected sort: %#v", cfg)
	}
	if !cfg.OnlyStandardPrice {
		t.Fatalf("expected only-standard-price enabled: %#v", cfg)
	}
}

func TestParseFlagsRejectsInvalidBudgetRange(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	_, err := parseFlags([]string{
		"--budget-min", "100",
		"--budget-max", "10",
		"example.com",
	}, stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--budget-min cannot be greater than --budget-max") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseFlagsRejectsUnsupportedSort(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	_, err := parseFlags([]string{"--sort", "domain", "example.com"}, stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `unsupported sort "domain"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyResultOptionsFiltersAndSortsByPrice(t *testing.T) {
	t.Parallel()

	results := []model.Result{
		{Domain: "premium.com", Status: model.StatusPremiumAvailable, PricingClass: "premium", Price: model.Float64Ptr(999), Available: model.BoolPtr(true)},
		{Domain: "standard-mid.com", Status: model.StatusStandardAvailable, PricingClass: "standard", Price: model.Float64Ptr(20), Available: model.BoolPtr(true)},
		{Domain: "standard-low.com", Status: model.StatusStandardAvailable, PricingClass: "standard", Price: model.Float64Ptr(12), Available: model.BoolPtr(true)},
		{Domain: "unknown-price.com", Status: model.StatusStandardAvailable, PricingClass: "standard", Available: model.BoolPtr(true)},
	}

	cfg := &Config{
		HasBudgetMin:      true,
		BudgetMin:         10,
		HasBudgetMax:      true,
		BudgetMax:         25,
		Sort:              "price",
		OnlyStandardPrice: true,
	}

	filtered := applyResultOptions(results, cfg)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 results, got %#v", filtered)
	}
	if filtered[0].Domain != "standard-low.com" || filtered[1].Domain != "standard-mid.com" {
		t.Fatalf("unexpected result order: %#v", filtered)
	}
}

func TestPriceTierSummary(t *testing.T) {
	t.Parallel()

	lines := priceTierSummary([]model.Result{
		{Domain: "low.com", Status: model.StatusStandardAvailable, Price: model.Float64Ptr(12), Available: model.BoolPtr(true)},
		{Domain: "mid.com", Status: model.StatusPremiumAvailable, Price: model.Float64Ptr(250), Available: model.BoolPtr(true)},
		{Domain: "high.com", Status: model.StatusPremiumAvailable, Price: model.Float64Ptr(1200), Available: model.BoolPtr(true)},
		{Domain: "unknown.com", Status: model.StatusAvailable, Available: model.BoolPtr(true)},
		{Domain: "taken.com", Status: model.StatusUnavailable, Price: model.Float64Ptr(15), Available: model.BoolPtr(false)},
	})

	output := strings.Join(lines, "\n")
	if !strings.Contains(output, "Price tier <$100: 1") {
		t.Fatalf("missing low tier summary: %v", lines)
	}
	if !strings.Contains(output, "Price tier $100-$499.99: 1") {
		t.Fatalf("missing mid tier summary: %v", lines)
	}
	if !strings.Contains(output, "Price tier $500+: 1") {
		t.Fatalf("missing high tier summary: %v", lines)
	}
	if !strings.Contains(output, "Price tier unpriced available: 1") {
		t.Fatalf("missing unpriced tier summary: %v", lines)
	}
}

func TestProviderBaseURLPrefersProductionWhenConfigured(t *testing.T) {
	t.Setenv("GODADDY_API_ENV", "prd")
	t.Setenv("GODADDY_OTE_API_BASE_URL", "https://ote.example.test")
	t.Setenv("GODADDY_PRD_API_BASE_URL", "https://prd.example.test")

	got := providerBaseURL("godaddy", &Config{RegistrarProviders: []string{"godaddy", "namecheap"}})
	if got != "https://prd.example.test" {
		t.Fatalf("expected production base url, got %q", got)
	}
}
