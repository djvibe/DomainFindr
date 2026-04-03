package consult

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/djvibe/domainfindr/internal/model"
)

type Brief struct {
	Title                string
	BusinessModel        string
	LaunchWedge          string
	LongTermAmbition     string
	PreferredNamingStyle []string
	Geography            string
	Budget               string
	RegistrarPreference  string
	Notes                []string
	Raw                  string
}

type SessionOptions struct {
	SessionName       string
	Styles            []string
	ProviderNames     []string
	GeneratedAt       time.Time
	ResultsRoot       string
	BriefPath         string
	CandidateInput    string
	PositionalDomains []string
}

type SessionArtifacts struct {
	Dir              string
	ReadmePath       string
	ExecutivePath    string
	FullReportPath   string
	CandidatesPath   string
	AvailabilityPath string
}

func ParseBriefFile(path string) (Brief, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Brief{}, err
	}
	brief := ParseBrief(string(data))
	if brief.Title == "" {
		base := filepath.Base(path)
		brief.Title = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return brief, nil
}

func ParseBrief(raw string) Brief {
	brief := Brief{Raw: strings.TrimSpace(raw)}
	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "#") && brief.Title == "" {
			brief.Title = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			continue
		}

		normalized := strings.TrimSpace(strings.TrimLeft(trimmed, "-*"))
		key, value, ok := splitField(normalized)
		if !ok {
			continue
		}

		switch normalizeKey(key) {
		case "businessmodel":
			brief.BusinessModel = value
		case "launchwedge":
			brief.LaunchWedge = value
		case "longtermplatformambition", "longtermambition":
			brief.LongTermAmbition = value
		case "preferrednamingstyles", "preferredstyles", "styles", "namingstyles":
			brief.PreferredNamingStyle = splitList(value)
		case "geography", "market":
			brief.Geography = value
		case "budget":
			brief.Budget = value
		case "registrarpreference", "registrar":
			brief.RegistrarPreference = value
		case "notes", "constraints":
			brief.Notes = append(brief.Notes, value)
		}
	}

	return brief
}

func CreateSession(brief Brief, options SessionOptions, results []model.Result) (SessionArtifacts, error) {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now()
	}

	root := options.ResultsRoot
	if strings.TrimSpace(root) == "" {
		root = filepath.Join("workspace", "results")
	}

	sessionName := strings.TrimSpace(options.SessionName)
	if sessionName == "" {
		sessionName = brief.Title
	}
	if sessionName == "" {
		sessionName = "domain-consult-session"
	}

	dir := filepath.Join(root, fmt.Sprintf("%s-%s", slugify(sessionName), generatedAt.Format("20060102")))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SessionArtifacts{}, err
	}

	artifacts := SessionArtifacts{
		Dir:              dir,
		ReadmePath:       filepath.Join(dir, "README.md"),
		ExecutivePath:    filepath.Join(dir, "executive-summary.md"),
		FullReportPath:   filepath.Join(dir, "full-report.md"),
		CandidatesPath:   filepath.Join(dir, "candidate-domains.md"),
		AvailabilityPath: filepath.Join(dir, "availability.md"),
	}

	if err := os.WriteFile(artifacts.ReadmePath, []byte(renderReadme(brief, options, results, generatedAt)), 0o644); err != nil {
		return SessionArtifacts{}, err
	}
	if err := os.WriteFile(artifacts.ExecutivePath, []byte(renderExecutiveSummary(brief, options, results, generatedAt)), 0o644); err != nil {
		return SessionArtifacts{}, err
	}
	if err := os.WriteFile(artifacts.FullReportPath, []byte(renderFullReport(brief, options, results, generatedAt)), 0o644); err != nil {
		return SessionArtifacts{}, err
	}
	if err := os.WriteFile(artifacts.CandidatesPath, []byte(renderCandidates(options, results)), 0o644); err != nil {
		return SessionArtifacts{}, err
	}
	if err := os.WriteFile(artifacts.AvailabilityPath, []byte(renderAvailability(results)), 0o644); err != nil {
		return SessionArtifacts{}, err
	}

	return artifacts, nil
}

func renderReadme(brief Brief, options SessionOptions, results []model.Result, generatedAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# Domain Consultant Session\n\n")
	builder.WriteString(fmt.Sprintf("- Generated: %s\n", generatedAt.Format("2006-01-02 15:04 MST")))
	if brief.Title != "" {
		builder.WriteString(fmt.Sprintf("- Brief: %s\n", brief.Title))
	}
	if options.BriefPath != "" {
		builder.WriteString(fmt.Sprintf("- Source brief: `%s`\n", options.BriefPath))
	}
	if len(options.Styles) > 0 {
		builder.WriteString(fmt.Sprintf("- Naming lanes: %s\n", strings.Join(options.Styles, ", ")))
	}
	if len(options.ProviderNames) > 0 {
		builder.WriteString(fmt.Sprintf("- Verification providers: %s\n", strings.Join(options.ProviderNames, ", ")))
	}
	builder.WriteString("\n## Artifacts\n\n")
	builder.WriteString("- `executive-summary.md`: fast recommendation view\n")
	builder.WriteString("- `full-report.md`: full strategy and evaluation notes\n")
	builder.WriteString("- `candidate-domains.md`: candidate pool used in this session\n")
	builder.WriteString("- `availability.md`: availability status snapshot\n")
	if len(results) == 0 {
		builder.WriteString("\n## Status\n\n")
		builder.WriteString("No availability sweep was run in this session. Add `--input` or positional domains to include live domain checks.\n")
	}
	return builder.String()
}

func renderExecutiveSummary(brief Brief, options SessionOptions, results []model.Result, generatedAt time.Time) string {
	available, premium, registered, unknown := categorizeResults(results)
	var builder strings.Builder
	builder.WriteString("# Executive Summary\n\n")
	if brief.Title != "" {
		builder.WriteString(fmt.Sprintf("## %s\n\n", brief.Title))
	}
	builder.WriteString("## Brief Snapshot\n\n")
	writeBriefSection(&builder, brief)
	builder.WriteString("\n## Recommendation Frame\n\n")
	for _, lane := range recommendationLanes(options, brief) {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", lane.Title, lane.Note))
	}
	builder.WriteString("\n## Availability Snapshot\n\n")
	builder.WriteString(fmt.Sprintf("- Standard-available candidates: %d\n", len(available)))
	builder.WriteString(fmt.Sprintf("- Premium-available candidates: %d\n", len(premium)))
	builder.WriteString(fmt.Sprintf("- Registered or unavailable candidates: %d\n", len(registered)))
	builder.WriteString(fmt.Sprintf("- Needs recheck: %d\n", len(unknown)))
	if len(available) > 0 {
		builder.WriteString("- Buy-now shortlist: ")
		builder.WriteString(strings.Join(domainNames(available, 5), ", "))
		builder.WriteString("\n")
	}
	if len(premium) > 0 {
		builder.WriteString("- Premium candidates needing pricing review: ")
		builder.WriteString(strings.Join(domainNames(premium, 3), ", "))
		builder.WriteString("\n")
	}
	if len(unknown) > 0 {
		builder.WriteString("- Recheck queue: ")
		builder.WriteString(strings.Join(domainNames(unknown, 5), ", "))
		builder.WriteString("\n")
	}
	builder.WriteString("\n## Next Steps\n\n")
	builder.WriteString("- Confirm which naming lane wins: company brand, directory/media, or authority/ranking.\n")
	builder.WriteString("- Verify buy-now candidates in registrar checkout before purchase.\n")
	builder.WriteString("- Move durable lessons into `docs/domain_research_lessons.md` after the session closes.\n")
	return builder.String()
}

func renderFullReport(brief Brief, options SessionOptions, results []model.Result, generatedAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# Full Report\n\n")
	builder.WriteString(fmt.Sprintf("Generated on %s.\n\n", generatedAt.Format("2006-01-02 15:04 MST")))
	builder.WriteString("## 1. Brief\n\n")
	writeBriefSection(&builder, brief)
	builder.WriteString("\n## 2. Naming Lanes\n\n")
	for _, lane := range recommendationLanes(options, brief) {
		builder.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", lane.Title, lane.Note))
	}
	builder.WriteString("## 3. Evaluation Rules\n\n")
	builder.WriteString("- Prioritize clarity, memorability, and expansion potential over raw availability.\n")
	builder.WriteString("- Treat premium domains as a separate acquisition class.\n")
	builder.WriteString("- Trust registrar checkout over RDAP alone for purchase decisions.\n\n")
	builder.WriteString("## 4. Availability Review\n\n")
	builder.WriteString(renderAvailability(results))
	return builder.String()
}

func renderCandidates(options SessionOptions, results []model.Result) string {
	var builder strings.Builder
	builder.WriteString("# Candidate Domains\n\n")
	if options.CandidateInput != "" {
		builder.WriteString(fmt.Sprintf("- Input file: `%s`\n", options.CandidateInput))
	}
	if len(options.PositionalDomains) > 0 {
		builder.WriteString(fmt.Sprintf("- Positional domains: %s\n", strings.Join(options.PositionalDomains, ", ")))
	}
	builder.WriteString("\n")

	if len(results) == 0 && len(options.PositionalDomains) == 0 {
		builder.WriteString("No candidate domains were supplied.\n")
		return builder.String()
	}

	for _, domain := range sortedCandidates(options, results) {
		builder.WriteString(fmt.Sprintf("- `%s`\n", domain))
	}
	return builder.String()
}

func renderAvailability(results []model.Result) string {
	if len(results) == 0 {
		return "No availability sweep was run.\n"
	}

	var builder strings.Builder
	builder.WriteString("| domain | status | provider | price |\n")
	builder.WriteString("| --- | --- | --- | --- |\n")
	for _, result := range results {
		price := "-"
		if result.Price != nil {
			currency := result.Currency
			if currency == "" {
				currency = "USD"
			}
			price = fmt.Sprintf("%.2f %s", *result.Price, currency)
		}
		provider := result.VerificationProvider
		if provider == "" {
			provider = result.Source
		}
		builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", result.Domain, result.Status, provider, price))
	}
	return builder.String()
}

type recommendationLane struct {
	Title string
	Note  string
}

func recommendationLanes(options SessionOptions, brief Brief) []recommendationLane {
	styles := options.Styles
	if len(styles) == 0 {
		styles = brief.PreferredNamingStyle
	}
	if len(styles) == 0 {
		styles = []string{"startup", "directory", "authority"}
	}

	lanes := make([]recommendationLane, 0, len(styles))
	for _, style := range styles {
		switch normalizeKey(style) {
		case "authority", "editorial", "ranking":
			lanes = append(lanes, recommendationLane{
				Title: "Authority / Ranking Lane",
				Note:  "Use names that signal trust, category coverage, and editorial confidence.",
			})
		case "directory", "category", "discovery", "marketplace":
			lanes = append(lanes, recommendationLane{
				Title: "Directory / Discovery Lane",
				Note:  "Use descriptive names that explain the category quickly and support search intent.",
			})
		case "software", "saas", "startup", "coined", "brandable":
			lanes = append(lanes, recommendationLane{
				Title: "Startup / Software Lane",
				Note:  "Use cleaner, brandable names that can stretch into product and company identity.",
			})
		default:
			lanes = append(lanes, recommendationLane{
				Title: strings.Title(strings.TrimSpace(style)) + " Lane",
				Note:  "Evaluate this lane against clarity, memorability, and expansion potential.",
			})
		}
	}

	return uniqueLanes(lanes)
}

func writeBriefSection(builder *strings.Builder, brief Brief) {
	startLen := builder.Len()
	writeBullet(builder, "Business model", brief.BusinessModel)
	writeBullet(builder, "Launch wedge", brief.LaunchWedge)
	writeBullet(builder, "Long-term ambition", brief.LongTermAmbition)
	if len(brief.PreferredNamingStyle) > 0 {
		writeBullet(builder, "Preferred naming styles", strings.Join(brief.PreferredNamingStyle, ", "))
	}
	writeBullet(builder, "Geography", brief.Geography)
	writeBullet(builder, "Budget", brief.Budget)
	writeBullet(builder, "Registrar preference", brief.RegistrarPreference)
	for _, note := range brief.Notes {
		writeBullet(builder, "Notes", note)
	}
	if builder.Len() == startLen {
		builder.WriteString("- No structured brief fields detected. Review the source brief directly.\n")
	}
}

func writeBullet(builder *strings.Builder, label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	builder.WriteString(fmt.Sprintf("- %s: %s\n", label, value))
}

func categorizeResults(results []model.Result) (available, premium, registered, unknown []model.Result) {
	for _, result := range results {
		switch result.Status {
		case model.StatusStandardAvailable, model.StatusAvailable:
			available = append(available, result)
		case model.StatusPremiumAvailable:
			premium = append(premium, result)
		case model.StatusRegistered, model.StatusUnavailable:
			registered = append(registered, result)
		default:
			unknown = append(unknown, result)
		}
	}
	return available, premium, registered, unknown
}

func domainNames(results []model.Result, limit int) []string {
	if len(results) == 0 {
		return nil
	}
	names := make([]string, 0, len(results))
	for _, result := range results {
		names = append(names, result.Domain)
	}
	sort.Strings(names)
	if limit > 0 && len(names) > limit {
		return names[:limit]
	}
	return names
}

func sortedCandidates(options SessionOptions, results []model.Result) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(options.PositionalDomains)+len(results))
	for _, domain := range options.PositionalDomains {
		domain = strings.TrimSpace(domain)
		if domain == "" || seen[domain] {
			continue
		}
		seen[domain] = true
		out = append(out, domain)
	}
	for _, result := range results {
		if result.Domain == "" || seen[result.Domain] {
			continue
		}
		seen[result.Domain] = true
		out = append(out, result.Domain)
	}
	sort.Strings(out)
	return out
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "domain-consult-session"
	}
	return slug
}

func splitField(line string) (string, string, bool) {
	for _, separator := range []string{":", "-"} {
		parts := strings.SplitN(line, separator, 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			return "", "", false
		}
		return key, value, true
	}
	return "", "", false
}

func splitList(value string) []string {
	rawItems := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '/'
	})
	out := make([]string, 0, len(rawItems))
	seen := map[string]bool{}
	for _, item := range rawItems {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := normalizeKey(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, trimmed)
	}
	return out
}

func normalizeKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", "/", "", ".", "")
	return replacer.Replace(value)
}

func uniqueLanes(lanes []recommendationLane) []recommendationLane {
	seen := map[string]bool{}
	out := make([]recommendationLane, 0, len(lanes))
	for _, lane := range lanes {
		key := normalizeKey(lane.Title)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, lane)
	}
	return out
}
