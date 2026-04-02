package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/djvibe/domainfindr/internal/model"
)

func WriteResults(w io.Writer, format string, results []model.Result) error {
	switch strings.ToLower(format) {
	case "", "table":
		return writeTable(w, results)
	case "csv":
		return writeCSV(w, results)
	case "json":
		return writeJSON(w, results)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writeCSV(w io.Writer, results []model.Result) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"domain", "available", "status", "source", "error"}); err != nil {
		return err
	}

	for _, result := range results {
		record := []string{
			result.Domain,
			boolString(result.Available),
			result.Status,
			result.Source,
			stringValue(result.Error),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

func writeJSON(w io.Writer, results []model.Result) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

func writeTable(w io.Writer, results []model.Result) error {
	headers := []string{"DOMAIN", "AVAILABLE", "STATUS", "SOURCE", "ERROR"}
	widths := []int{len(headers[0]), len(headers[1]), len(headers[2]), len(headers[3]), len(headers[4])}
	rows := make([][]string, 0, len(results))

	for _, result := range results {
		row := []string{
			result.Domain,
			tableAvailability(result.Available),
			result.Status,
			result.Source,
			stringValue(result.Error),
		}
		rows = append(rows, row)
		for idx, cell := range row {
			if len(cell) > widths[idx] {
				widths[idx] = len(cell)
			}
		}
	}

	if _, err := fmt.Fprintln(w, formatRow(headers, widths)); err != nil {
		return err
	}
	divider := []string{
		strings.Repeat("-", widths[0]),
		strings.Repeat("-", widths[1]),
		strings.Repeat("-", widths[2]),
		strings.Repeat("-", widths[3]),
		strings.Repeat("-", widths[4]),
	}
	if _, err := fmt.Fprintln(w, formatRow(divider, widths)); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(w, formatRow(row, widths)); err != nil {
			return err
		}
	}

	return nil
}

func formatRow(row []string, widths []int) string {
	parts := make([]string, len(row))
	for idx, cell := range row {
		parts[idx] = fmt.Sprintf("%-*s", widths[idx], cell)
	}
	return strings.Join(parts, " | ")
}

func boolString(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "true"
	}
	return "false"
}

func tableAvailability(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "Yes"
	}
	return "No"
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
