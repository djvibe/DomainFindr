package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/djvibe/domainfindr/internal/consult"
	"github.com/djvibe/domainfindr/internal/input"
	"github.com/djvibe/domainfindr/internal/lookup"
	"github.com/djvibe/domainfindr/internal/model"
	"github.com/djvibe/domainfindr/internal/output"
	"github.com/djvibe/domainfindr/internal/runner"
)

type Config struct {
	Input              string
	Output             string
	Format             string
	Concurrency        int
	Delay              time.Duration
	Retry              int
	Timeout            time.Duration
	RegistrarRetry     int
	RegistrarTimeout   time.Duration
	RegistrarRecheck   int
	Verbose            bool
	Domains            []string
	LogFile            string
	HistoryDir         string
	NoHistory          bool
	RegistrarBaseURL   string
	RegistrarProvider  string
	RegistrarProviders []string
	GoDaddyAPIKey      string
	GoDaddyAPISecret   string
	NamecheapAPIUser   string
	NamecheapAPIKey    string
	NamecheapUserName  string
	NamecheapClientIP  string
	Brief              string
	Styles             []string
	SessionName        string
	ResultsDir         string
}

func Run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := loadDotEnv(".env"); err != nil {
		fmt.Fprintf(stderr, "env error: %v\n", err)
		return 1
	}

	if len(args) > 0 && args[0] == "consult" {
		return runConsult(ctx, args[1:], stdout, stderr)
	}

	cfg, err := parseFlags(args, stderr)
	if err != nil {
		return 1
	}
	if cfg == nil {
		return 0
	}

	history, err := prepareHistory(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "history error: %v\n", err)
		return 1
	}
	defer closeQuietly(history.logFile)
	defer closeQuietly(history.outputFile)

	logWriter := io.Writer(io.Discard)
	if cfg.Verbose {
		logWriter = stderr
	}
	if history.logFile != nil {
		if cfg.Verbose {
			logWriter = io.MultiWriter(stderr, history.logFile)
		} else {
			logWriter = history.logFile
		}
	}
	logger := log.New(logWriter, "", log.LstdFlags)

	results, err := executeLookup(ctx, cfg, logger)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	writer := stdout
	summaryWriter := stderr
	if cfg.Output != "" {
		file, err := os.Create(cfg.Output)
		if err != nil {
			fmt.Fprintf(stderr, "output error: %v\n", err)
			return 1
		}
		defer file.Close()
		writer = file
	} else if history.outputFile != nil {
		writer = io.MultiWriter(stdout, history.outputFile)
		summaryWriter = stdout
	} else {
		summaryWriter = stdout
	}

	if err := output.WriteResults(writer, cfg.Format, results); err != nil {
		fmt.Fprintf(stderr, "format error: %v\n", err)
		return 1
	}

	printSummary(summaryWriter, results)
	if history.dir != "" {
		fmt.Fprintf(summaryWriter, "History saved in %s\n", history.dir)
	}
	return 0
}

func runConsult(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	cfg, err := parseConsultFlags(args, stderr)
	if err != nil {
		return 1
	}
	if cfg == nil {
		return 0
	}

	brief, err := consult.ParseBriefFile(cfg.Brief)
	if err != nil {
		fmt.Fprintf(stderr, "brief error: %v\n", err)
		return 1
	}

	var results []model.Result
	if cfg.Input != "" || len(cfg.Domains) > 0 {
		logger := log.New(io.Discard, "", log.LstdFlags)
		results, err = executeLookup(ctx, cfg, logger)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 1
		}
	}

	artifacts, err := consult.CreateSession(brief, consult.SessionOptions{
		SessionName:       cfg.SessionName,
		Styles:            cfg.Styles,
		ProviderNames:     cfg.RegistrarProviders,
		GeneratedAt:       time.Now(),
		ResultsRoot:       cfg.ResultsDir,
		BriefPath:         cfg.Brief,
		CandidateInput:    cfg.Input,
		PositionalDomains: cfg.Domains,
	}, results)
	if err != nil {
		fmt.Fprintf(stderr, "consult error: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Consult session created in %s\n", artifacts.Dir)
	fmt.Fprintf(stdout, "Executive summary: %s\n", artifacts.ExecutivePath)
	fmt.Fprintf(stdout, "Full report: %s\n", artifacts.FullReportPath)
	if len(results) > 0 {
		printSummary(stdout, results)
	}
	return 0
}

func parseFlags(args []string, stderr io.Writer) (*Config, error) {
	fs := flag.NewFlagSet("domainfindr", flag.ContinueOnError)
	fs.SetOutput(stderr)

	cfg := &Config{}
	fs.StringVar(&cfg.Input, "input", "", "Input file (Markdown or CSV)")
	fs.StringVar(&cfg.Input, "i", "", "Input file (Markdown or CSV)")
	fs.StringVar(&cfg.Output, "output", "", "Output file (defaults to stdout)")
	fs.StringVar(&cfg.Output, "o", "", "Output file (defaults to stdout)")
	fs.StringVar(&cfg.Format, "format", "table", "Output format: table, csv, or json")
	fs.StringVar(&cfg.Format, "f", "table", "Output format: table, csv, or json")
	fs.IntVar(&cfg.Concurrency, "concurrency", 4, "Number of worker goroutines")
	fs.IntVar(&cfg.Concurrency, "c", 4, "Number of worker goroutines")
	fs.DurationVar(&cfg.Delay, "delay", time.Second, "Minimum delay between outbound lookups")
	fs.IntVar(&cfg.Retry, "retry", 2, "Retries for transient lookup failures")
	fs.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "Timeout per lookup attempt")
	fs.IntVar(&cfg.RegistrarRetry, "registrar-retry", 1, "Retries for transient registrar verification failures")
	fs.DurationVar(&cfg.RegistrarTimeout, "registrar-timeout", 3*time.Second, "Timeout per registrar verification attempt")
	fs.IntVar(&cfg.RegistrarRecheck, "registrar-recheck", 1, "Additional registrar-only recheck passes for transient registrar_unknown results")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose logging")
	fs.BoolVar(&cfg.Verbose, "v", false, "Enable verbose logging")
	fs.StringVar(&cfg.LogFile, "log-file", "", "Write logs to a file")
	fs.StringVar(&cfg.HistoryDir, "history-dir", "", "Directory for timestamped result and log history")
	fs.BoolVar(&cfg.NoHistory, "no-history", false, "Disable automatic history saving")
	fs.StringVar(&cfg.RegistrarBaseURL, "registrar-base-url", "", "Registrar availability API base URL for secondary verification")
	fs.StringVar(&cfg.RegistrarProvider, "registrar-provider", "", "Registrar provider name for verification output")
	fs.StringVar(&cfg.RegistrarProvider, "provider", "", "Registrar provider name for verification output")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	resolveProviderEnvironment(cfg)
	cfg.RegistrarProviders = normalizeProviderList(cfg.RegistrarProvider)
	if len(cfg.RegistrarProviders) == 0 {
		if cfg.GoDaddyAPIKey != "" && cfg.GoDaddyAPISecret != "" {
			cfg.RegistrarProviders = []string{lookup.GoDaddyProviderName}
		} else if cfg.NamecheapAPIUser != "" && cfg.NamecheapAPIKey != "" && cfg.NamecheapClientIP != "" {
			cfg.RegistrarProviders = []string{lookup.NamecheapProviderName}
		}
	}

	cfg.Domains = fs.Args()
	if cfg.Input == "" && len(cfg.Domains) == 0 {
		fs.Usage()
		return nil, fmt.Errorf("provide --input FILE or one or more domains")
	}

	return cfg, nil
}

func parseConsultFlags(args []string, stderr io.Writer) (*Config, error) {
	fs := flag.NewFlagSet("domainfindr consult", flag.ContinueOnError)
	fs.SetOutput(stderr)

	cfg := &Config{}
	var styles string
	fs.StringVar(&cfg.Brief, "brief", "", "Structured brief file (Markdown)")
	fs.StringVar(&cfg.Input, "input", "", "Optional candidate domain file (Markdown or CSV)")
	fs.StringVar(&cfg.Input, "i", "", "Optional candidate domain file (Markdown or CSV)")
	fs.StringVar(&cfg.RegistrarProvider, "provider", "", "Registrar provider name for verification output")
	fs.StringVar(&cfg.RegistrarProvider, "registrar-provider", "", "Registrar provider name for verification output")
	fs.StringVar(&cfg.RegistrarBaseURL, "registrar-base-url", "", "Registrar availability API base URL for secondary verification")
	fs.IntVar(&cfg.Concurrency, "concurrency", 4, "Number of worker goroutines")
	fs.DurationVar(&cfg.Delay, "delay", time.Second, "Minimum delay between outbound lookups")
	fs.IntVar(&cfg.Retry, "retry", 2, "Retries for transient lookup failures")
	fs.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "Timeout per lookup attempt")
	fs.IntVar(&cfg.RegistrarRetry, "registrar-retry", 1, "Retries for transient registrar verification failures")
	fs.DurationVar(&cfg.RegistrarTimeout, "registrar-timeout", 3*time.Second, "Timeout per registrar verification attempt")
	fs.IntVar(&cfg.RegistrarRecheck, "registrar-recheck", 1, "Additional registrar-only recheck passes for transient registrar_unknown results")
	fs.StringVar(&styles, "style", "", "Comma-separated naming lanes, for example authority,startup,directory")
	fs.StringVar(&cfg.SessionName, "session-name", "", "Optional session folder name override")
	fs.StringVar(&cfg.ResultsDir, "results-dir", filepath.Join("workspace", "results"), "Directory for consultant session artifacts")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Brief) == "" {
		fs.Usage()
		return nil, fmt.Errorf("provide --brief FILE")
	}

	resolveProviderEnvironment(cfg)
	cfg.RegistrarProviders = normalizeProviderList(cfg.RegistrarProvider)
	if len(cfg.RegistrarProviders) == 0 {
		if cfg.GoDaddyAPIKey != "" && cfg.GoDaddyAPISecret != "" {
			cfg.RegistrarProviders = []string{lookup.GoDaddyProviderName}
		} else if cfg.NamecheapAPIUser != "" && cfg.NamecheapAPIKey != "" && cfg.NamecheapClientIP != "" {
			cfg.RegistrarProviders = []string{lookup.NamecheapProviderName}
		}
	}

	cfg.Styles = splitCSVValues(styles)
	cfg.Domains = fs.Args()
	return cfg, nil
}

func resolveProviderEnvironment(cfg *Config) {
	cfg.GoDaddyAPIKey = os.Getenv("GODADDY_API_KEY")
	cfg.GoDaddyAPISecret = os.Getenv("GODADDY_API_SECRET")
	cfg.NamecheapAPIUser = os.Getenv("NAMECHEAP_API_USER")
	cfg.NamecheapAPIKey = os.Getenv("NAMECHEAP_API_KEY")
	cfg.NamecheapUserName = os.Getenv("NAMECHEAP_USERNAME")
	cfg.NamecheapClientIP = os.Getenv("NAMECHEAP_CLIENT_IP")
	resolveGoDaddyCredentials(cfg)
	resolveNamecheapCredentials(cfg)
}

func splitCSVValues(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	items := strings.Split(raw, ",")
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, trimmed)
	}
	return out
}

func executeLookup(ctx context.Context, cfg *Config, logger *log.Logger) ([]model.Result, error) {
	var entries []model.Entry
	var err error
	if cfg.Input != "" {
		entries, err = input.ParseFile(cfg.Input)
		if err != nil {
			return nil, fmt.Errorf("input error: %v", err)
		}
	} else {
		entries = input.ParseDomains(cfg.Domains)
	}

	httpClient := &http.Client{}
	providers := []lookup.Provider{lookup.NewRDAPChecker(httpClient)}
	for _, providerName := range cfg.RegistrarProviders {
		switch providerName {
		case "", "none":
		case lookup.GoDaddyProviderName:
			if cfg.GoDaddyAPIKey == "" || cfg.GoDaddyAPISecret == "" {
				return nil, fmt.Errorf("config error: GoDaddy provider requires GODADDY_API_KEY and GODADDY_API_SECRET")
			}
			providers = append(providers, lookup.NewGoDaddyProvider(cfg.GoDaddyAPIKey, cfg.GoDaddyAPISecret, httpClient, providerBaseURL(providerName, cfg)))
		case lookup.NamecheapProviderName:
			if cfg.NamecheapAPIUser == "" || cfg.NamecheapAPIKey == "" || cfg.NamecheapClientIP == "" {
				return nil, fmt.Errorf("config error: Namecheap provider requires NAMECHEAP_API_USER, NAMECHEAP_API_KEY, and NAMECHEAP_CLIENT_IP")
			}
			providers = append(providers, lookup.NewNamecheapProvider(cfg.NamecheapAPIUser, cfg.NamecheapAPIKey, cfg.NamecheapUserName, cfg.NamecheapClientIP, nil, providerBaseURL(providerName, cfg)))
		default:
			if cfg.RegistrarBaseURL == "" {
				return nil, fmt.Errorf("config error: registrar provider %q requires --registrar-base-url", providerName)
			}
			providers = append(providers, lookup.NewRegistrarHTTPProvider(providerName, httpClient, cfg.RegistrarBaseURL))
		}
	}
	checker := lookup.NewVerifierWithConfig(lookup.VerifierConfig{
		RegistrarRetry:   cfg.RegistrarRetry,
		RegistrarTimeout: cfg.RegistrarTimeout,
		RegistrarRecheck: cfg.RegistrarRecheck,
	}, providers...)
	run := runner.New(checker, nil, runner.Config{
		Concurrency: cfg.Concurrency,
		Delay:       cfg.Delay,
		Retry:       cfg.Retry,
		Timeout:     cfg.Timeout,
		Logger:      logger,
	})
	return run.Run(ctx, entries), nil
}

func printSummary(w io.Writer, results []model.Result) {
	var available, registered, registrarUnknown, invalid, errors, disagreements int
	disagreementDomains := make([]string, 0)
	for _, result := range results {
		switch result.Status {
		case model.StatusAvailable, model.StatusStandardAvailable, model.StatusPremiumAvailable:
			available++
		case model.StatusRegistered, model.StatusUnavailable:
			registered++
		case model.StatusRegistrarUnknown:
			registrarUnknown++
		case model.StatusInvalid:
			invalid++
		case model.StatusLookupError:
			errors++
		}
		if hasDisagreement(result) {
			disagreements++
			disagreementDomains = append(disagreementDomains, result.Domain)
		}
	}

	fmt.Fprintf(
		w,
		"Checked %d domains: %d available, %d registered/unavailable, %d registrar unknown, %d invalid, %d lookup errors, %d disagreements.\n",
		len(results),
		available,
		registered,
		registrarUnknown,
		invalid,
		errors,
		disagreements,
	)
	for _, line := range providerSummary(results) {
		fmt.Fprintln(w, line)
	}
	if len(disagreementDomains) > 0 {
		fmt.Fprintf(w, "Disagreements: %s\n", strings.Join(disagreementDomains, ", "))
	}
}

type historyFiles struct {
	dir        string
	logFile    *os.File
	outputFile *os.File
}

func prepareHistory(cfg *Config) (historyFiles, error) {
	if cfg.NoHistory {
		if cfg.LogFile == "" {
			return historyFiles{}, nil
		}
		logFile, err := os.Create(cfg.LogFile)
		if err != nil {
			return historyFiles{}, err
		}
		return historyFiles{logFile: logFile}, nil
	}

	dir := cfg.HistoryDir
	if dir == "" {
		dir = defaultHistoryRoot()
	}
	runDir := filepath.Join(dir, time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return historyFiles{}, err
	}

	var logPath string
	if cfg.LogFile != "" {
		logPath = cfg.LogFile
	} else {
		logPath = filepath.Join(runDir, "domainfindr.log")
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return historyFiles{}, err
	}

	var outputFile *os.File
	if cfg.Output == "" {
		outputPath := filepath.Join(runDir, "results."+outputExtension(cfg.Format))
		outputFile, err = os.Create(outputPath)
		if err != nil {
			logFile.Close()
			return historyFiles{}, err
		}
	}

	return historyFiles{
		dir:        runDir,
		logFile:    logFile,
		outputFile: outputFile,
	}, nil
}

func defaultHistoryRoot() string {
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "domainfindr", "history")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".domainfindr-history")
	}
	return filepath.Join(home, ".local", "state", "domainfindr", "history")
}

func outputExtension(format string) string {
	switch strings.ToLower(format) {
	case "json":
		return "json"
	case "csv":
		return "csv"
	default:
		return "txt"
	}
}

func closeQuietly(file *os.File) {
	if file != nil {
		_ = file.Close()
	}
}

func resolveGoDaddyCredentials(cfg *Config) {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("GODADDY_API_ENV")))
	if cfg.GoDaddyAPIKey == "" || cfg.GoDaddyAPISecret == "" {
		switch env {
		case "ote":
			cfg.GoDaddyAPIKey = firstNonEmpty(cfg.GoDaddyAPIKey, os.Getenv("GODADDY_OTE_API_KEY"))
			cfg.GoDaddyAPISecret = firstNonEmpty(cfg.GoDaddyAPISecret, os.Getenv("GODADDY_OTE_API_SECRET"))
		case "prd", "prod", "production":
			cfg.GoDaddyAPIKey = firstNonEmpty(cfg.GoDaddyAPIKey, os.Getenv("GODADDY_PRD_API_KEY"))
			cfg.GoDaddyAPISecret = firstNonEmpty(cfg.GoDaddyAPISecret, os.Getenv("GODADDY_PRD_API_SECRET"))
		}
	}
	if cfg.RegistrarBaseURL == "" {
		switch env {
		case "ote":
			cfg.RegistrarBaseURL = firstNonEmpty(
				os.Getenv("GODADDY_API_BASE_URL"),
				os.Getenv("GODADDY_OTE_API_BASE_URL"),
				"https://api.ote-godaddy.com",
			)
		case "prd", "prod", "production":
			cfg.RegistrarBaseURL = firstNonEmpty(
				os.Getenv("GODADDY_API_BASE_URL"),
				os.Getenv("GODADDY_PRD_API_BASE_URL"),
				"https://api.godaddy.com",
			)
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeProviderList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		name := lookup.NormalizeProviderName(item)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func providerBaseURL(provider string, cfg *Config) string {
	if cfg.RegistrarBaseURL != "" && len(cfg.RegistrarProviders) == 1 {
		return cfg.RegistrarBaseURL
	}
	switch provider {
	case lookup.GoDaddyProviderName:
		switch strings.ToLower(strings.TrimSpace(os.Getenv("GODADDY_API_ENV"))) {
		case "prd", "prod", "production":
			return firstNonEmpty(os.Getenv("GODADDY_API_BASE_URL"), os.Getenv("GODADDY_PRD_API_BASE_URL"), os.Getenv("GODADDY_OTE_API_BASE_URL"))
		case "ote":
			return firstNonEmpty(os.Getenv("GODADDY_API_BASE_URL"), os.Getenv("GODADDY_OTE_API_BASE_URL"), os.Getenv("GODADDY_PRD_API_BASE_URL"))
		default:
			return firstNonEmpty(os.Getenv("GODADDY_API_BASE_URL"), os.Getenv("GODADDY_OTE_API_BASE_URL"), os.Getenv("GODADDY_PRD_API_BASE_URL"))
		}
	case lookup.NamecheapProviderName:
		switch strings.ToLower(strings.TrimSpace(os.Getenv("NAMECHEAP_API_ENV"))) {
		case "prd", "prod", "production":
			return firstNonEmpty(os.Getenv("NAMECHEAP_API_BASE_URL"), os.Getenv("NAMECHEAP_PRD_API_BASE_URL"), os.Getenv("NAMECHEAP_SANDBOX_API_BASE_URL"))
		case "sandbox", "test":
			return firstNonEmpty(os.Getenv("NAMECHEAP_API_BASE_URL"), os.Getenv("NAMECHEAP_SANDBOX_API_BASE_URL"), os.Getenv("NAMECHEAP_PRD_API_BASE_URL"))
		default:
			return firstNonEmpty(os.Getenv("NAMECHEAP_API_BASE_URL"), os.Getenv("NAMECHEAP_SANDBOX_API_BASE_URL"), os.Getenv("NAMECHEAP_PRD_API_BASE_URL"))
		}
	default:
		return cfg.RegistrarBaseURL
	}
}

func hasDisagreement(result model.Result) bool {
	rdap := result.RegistryStatus
	for _, check := range result.Verifications {
		if check.Provider == model.ProviderRDAP || check.Status == "" {
			continue
		}
		if rdap != "" && rdap != model.StatusLookupError && check.Status != rdap {
			return true
		}
	}
	return false
}

func providerSummary(results []model.Result) []string {
	counts := map[string]int{}
	for _, result := range results {
		for _, check := range result.Verifications {
			if check.Provider == "" {
				continue
			}
			counts[providerSummaryKey(check.Provider, check.Environment)]++
		}
	}
	lines := make([]string, 0, len(counts))
	for provider, count := range counts {
		lines = append(lines, fmt.Sprintf("Provider %s returned %d checks.", provider, count))
	}
	return lines
}

func providerLabel(provider string) string {
	switch provider {
	case lookup.GoDaddyProviderName:
		return "GoDaddy"
	case lookup.NamecheapProviderName:
		return "Namecheap"
	case model.ProviderRDAP:
		return "RDAP"
	default:
		return provider
	}
}

func providerSummaryKey(provider string, environment string) string {
	label := providerLabel(provider)
	env := strings.TrimSpace(environment)
	if env == "" || provider == model.ProviderRDAP {
		return label
	}
	return fmt.Sprintf("%s (%s)", label, env)
}

func resolveNamecheapCredentials(cfg *Config) {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("NAMECHEAP_API_ENV")))
	if cfg.NamecheapAPIUser == "" || cfg.NamecheapAPIKey == "" || cfg.NamecheapClientIP == "" {
		switch env {
		case "sandbox", "test":
			cfg.NamecheapAPIUser = firstNonEmpty(cfg.NamecheapAPIUser, os.Getenv("NAMECHEAP_SANDBOX_API_USER"))
			cfg.NamecheapAPIKey = firstNonEmpty(cfg.NamecheapAPIKey, os.Getenv("NAMECHEAP_SANDBOX_API_KEY"))
			cfg.NamecheapUserName = firstNonEmpty(cfg.NamecheapUserName, os.Getenv("NAMECHEAP_SANDBOX_USERNAME"))
			cfg.NamecheapClientIP = firstNonEmpty(cfg.NamecheapClientIP, os.Getenv("NAMECHEAP_SANDBOX_CLIENT_IP"))
		case "prd", "prod", "production":
			cfg.NamecheapAPIUser = firstNonEmpty(cfg.NamecheapAPIUser, os.Getenv("NAMECHEAP_PRD_API_USER"))
			cfg.NamecheapAPIKey = firstNonEmpty(cfg.NamecheapAPIKey, os.Getenv("NAMECHEAP_PRD_API_KEY"))
			cfg.NamecheapUserName = firstNonEmpty(cfg.NamecheapUserName, os.Getenv("NAMECHEAP_PRD_USERNAME"))
			cfg.NamecheapClientIP = firstNonEmpty(cfg.NamecheapClientIP, os.Getenv("NAMECHEAP_PRD_CLIENT_IP"))
		}
	}
	if cfg.NamecheapUserName == "" {
		cfg.NamecheapUserName = cfg.NamecheapAPIUser
	}
	if cfg.RegistrarBaseURL == "" {
		switch env {
		case "sandbox", "test":
			cfg.RegistrarBaseURL = firstNonEmpty(
				os.Getenv("NAMECHEAP_API_BASE_URL"),
				os.Getenv("NAMECHEAP_SANDBOX_API_BASE_URL"),
				"https://api.sandbox.namecheap.com/xml.response",
			)
		case "prd", "prod", "production":
			cfg.RegistrarBaseURL = firstNonEmpty(
				os.Getenv("NAMECHEAP_API_BASE_URL"),
				os.Getenv("NAMECHEAP_PRD_API_BASE_URL"),
				"https://api.namecheap.com/xml.response",
			)
		}
	}
}
