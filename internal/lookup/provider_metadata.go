package lookup

import "strings"

const (
	EnvironmentProduction = "production"
	EnvironmentSandbox    = "sandbox"
	EnvironmentOTE        = "ote"
	EnvironmentCustom     = "custom"
	EnvironmentRegistry   = "registry"
)

func providerEnvironment(provider string, baseURL string) string {
	base := strings.ToLower(strings.TrimSpace(baseURL))

	switch provider {
	case GoDaddyProviderName:
		switch {
		case strings.Contains(base, "ote-godaddy"):
			return EnvironmentOTE
		case strings.Contains(base, "api.godaddy.com"):
			return EnvironmentProduction
		case base != "":
			return EnvironmentCustom
		default:
			return EnvironmentProduction
		}
	case NamecheapProviderName:
		switch {
		case strings.Contains(base, "sandbox.namecheap"):
			return EnvironmentSandbox
		case strings.Contains(base, "api.namecheap.com"):
			return EnvironmentProduction
		case base != "":
			return EnvironmentCustom
		default:
			return EnvironmentSandbox
		}
	case "":
		return ""
	default:
		if base == "" {
			return EnvironmentCustom
		}
		return EnvironmentCustom
	}
}
