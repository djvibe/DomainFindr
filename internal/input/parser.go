package input

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/djvibe/domainfindr/internal/domain"
	"github.com/djvibe/domainfindr/internal/model"
)

var (
	bulletPrefix  = regexp.MustCompile(`^\s*(?:[-*+]\s+|\d+\.\s+)`)
	markdownSplit = regexp.MustCompile(`\s*\|\s*`)
)

func ParseFile(path string) ([]model.Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return parseMarkdown(file)
	case ".csv":
		return parseCSV(file)
	default:
		return nil, fmt.Errorf("unsupported input format for %q", path)
	}
}

func ParseDomains(values []string) []model.Entry {
	seen := map[string]struct{}{}
	entries := make([]model.Entry, 0, len(values))
	for _, value := range values {
		addCandidate(&entries, seen, value)
	}
	return entries
}

func parseMarkdown(r io.Reader) ([]model.Entry, error) {
	scanner := bufio.NewScanner(r)
	seen := map[string]struct{}{}
	var entries []model.Entry

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if isMarkdownSeparator(line) {
			continue
		}

		if strings.Contains(line, "|") {
			cells := splitMarkdownCells(line)
			if looksLikeMarkdownHeader(cells) {
				continue
			}
			for _, cell := range cells {
				addCandidate(&entries, seen, cell)
			}
			continue
		}

		line = bulletPrefix.ReplaceAllString(line, "")
		addCandidate(&entries, seen, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func parseCSV(r io.Reader) ([]model.Entry, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	if strings.Count(firstLine(string(data)), ";") > strings.Count(firstLine(string(data)), ",") {
		reader.Comma = ';'
	}

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	start := 0
	column := 0
	for idx, value := range rows[0] {
		if strings.EqualFold(strings.TrimSpace(value), "domain") {
			column = idx
			start = 1
			break
		}
	}

	seen := map[string]struct{}{}
	var entries []model.Entry
	for _, row := range rows[start:] {
		if column >= len(row) {
			continue
		}
		addCandidate(&entries, seen, row[column])
	}

	return entries, nil
}

func addCandidate(entries *[]model.Entry, seen map[string]struct{}, candidate string) {
	raw := strings.TrimSpace(candidate)
	if raw == "" {
		return
	}

	raw = strings.Trim(raw, "`")
	normalized := domain.Normalize(raw)
	if normalized == "" {
		return
	}

	if _, ok := seen[normalized]; ok {
		return
	}
	seen[normalized] = struct{}{}

	*entries = append(*entries, model.Entry{
		Raw:    raw,
		Domain: normalized,
		Valid:  domain.IsValid(normalized),
	})
}

func isMarkdownSeparator(line string) bool {
	if strings.HasPrefix(line, "#") {
		return true
	}

	trimmed := strings.ReplaceAll(line, "|", "")
	trimmed = strings.ReplaceAll(trimmed, "-", "")
	trimmed = strings.ReplaceAll(trimmed, ":", "")
	return strings.TrimSpace(trimmed) == ""
}

func splitMarkdownCells(line string) []string {
	parts := markdownSplit.Split(strings.Trim(line, "|"), -1)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			out = append(out, part)
		}
	}
	return out
}

func looksLikeMarkdownHeader(cells []string) bool {
	if len(cells) == 0 {
		return false
	}

	for _, cell := range cells {
		value := strings.ToLower(strings.TrimSpace(cell))
		if value != "domain" && value != "domains" {
			return false
		}
	}

	return true
}

func firstLine(value string) string {
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return value[:idx]
	}
	return value
}
