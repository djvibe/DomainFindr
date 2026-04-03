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
	providers := collectProviders(results)
	headers := []string{"domain", "available", "status", "source", "registry_status", "registrar_status", "registrar_consensus", "pricing_class", "price", "currency", "registration_period", "verification_provider", "verification_environment", "error"}
	for _, provider := range providers {
		headers = append(headers, provider+"_result")
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, result := range results {
		record := []string{
			result.Domain,
			boolString(result.Available),
			result.Status,
			result.Source,
			result.RegistryStatus,
			result.RegistrarStatus,
			result.RegistrarConsensus,
			result.PricingClass,
			floatString(result.Price),
			result.Currency,
			intString(result.RegistrationPeriod),
			displayProvider(result.VerificationProvider),
			displayEnvironment(result.VerificationEnv),
			stringValue(result.Error),
		}
		for _, provider := range providers {
			record = append(record, providerCell(findCheck(result, provider)))
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
	providers := collectProviders(results)
	headers := []string{"DOMAIN", "FINAL", "RDAP", "REGISTRAR", "CONSENSUS", "ENV", "PRICE"}
	for _, provider := range providers {
		if provider == model.ProviderRDAP {
			continue
		}
		headers = append(headers, strings.ToUpper(displayProvider(provider)))
	}
	headers = append(headers, "ERROR")

	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}

	rows := make([][]string, 0, len(results))
	for _, result := range results {
		row := []string{
			result.Domain,
			finalCell(result),
			result.RegistryStatus,
			result.RegistrarStatus,
			result.RegistrarConsensus,
			displayEnvironment(result.VerificationEnv),
			priceCell(result.Price, result.Currency, result.RegistrationPeriod),
		}
		for _, provider := range providers {
			if provider == model.ProviderRDAP {
				continue
			}
			row = append(row, providerCell(findCheck(result, provider)))
		}
		row = append(row, stringValue(result.Error))
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
	divider := make([]string, len(headers))
	for i := range divider {
		divider[i] = strings.Repeat("-", widths[i])
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func floatString(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *value)
}

func intString(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func collectProviders(results []model.Result) []string {
	seen := map[string]bool{}
	providers := make([]string, 0)
	for _, result := range results {
		for _, check := range result.Verifications {
			if check.Provider == "" || seen[check.Provider] {
				continue
			}
			seen[check.Provider] = true
			providers = append(providers, check.Provider)
		}
	}
	return providers
}

func findCheck(result model.Result, provider string) *model.Check {
	for i := range result.Verifications {
		if result.Verifications[i].Provider == provider {
			return &result.Verifications[i]
		}
	}
	return nil
}

func finalCell(result model.Result) string {
	return strings.TrimSpace(strings.Join([]string{result.Status, availabilityTag(result.Available)}, " "))
}

func availabilityTag(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "(yes)"
	}
	return "(no)"
}

func priceCell(price *float64, currency string, term *int) string {
	value := floatString(price)
	if value == "" {
		return ""
	}
	if currency != "" {
		value += " " + currency
	}
	if term != nil {
		value += "/" + intString(term) + "y"
	}
	return value
}

func providerCell(check *model.Check) string {
	if check == nil {
		return ""
	}
	cell := check.Status
	if env := displayEnvironment(check.Environment); env != "" {
		cell += " [" + env + "]"
	}
	if check.Price != nil {
		cell += " " + priceCell(check.Price, check.Currency, check.RegistrationPeriod)
	}
	if check.Error != nil && *check.Error != "" {
		cell += " err"
	}
	return strings.TrimSpace(cell)
}

func displayEnvironment(environment string) string {
	switch environment {
	case "":
		return ""
	case "production":
		return "prod"
	case "sandbox":
		return "sandbox"
	case "ote":
		return "ote"
	default:
		return environment
	}
}

func displayProvider(provider string) string {
	switch provider {
	case "godaddy":
		return "GoDaddy"
	case "namecheap":
		return "Namecheap"
	case model.ProviderRDAP:
		return "RDAP"
	default:
		return provider
	}
}
