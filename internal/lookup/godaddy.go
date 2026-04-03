package lookup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/djvibe/domainfindr/internal/model"
)

const (
	GoDaddyProviderName = "godaddy"
	goDaddyDefaultURL   = "https://api.godaddy.com"
)

type GoDaddyProvider struct {
	BaseURL string
	Key     string
	Secret  string
	Client  *http.Client
}

func NewGoDaddyProvider(key string, secret string, client *http.Client, baseURL string) *GoDaddyProvider {
	if client == nil {
		client = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = goDaddyDefaultURL
	}

	return &GoDaddyProvider{
		BaseURL: baseURL,
		Key:     key,
		Secret:  secret,
		Client:  client,
	}
}

func (p *GoDaddyProvider) Check(ctx context.Context, domain string) model.Result {
	endpoint := strings.TrimRight(p.BaseURL, "/") + "/v1/domains/available?domain=" + url.QueryEscape(domain) + "&checkType=FULL"
	environment := providerEnvironment(GoDaddyProviderName, p.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		result := registrarUnknown(domain, GoDaddyProviderName, fmt.Errorf("build request: %w", err))
		result.VerificationEnv = environment
		return result
	}
	req.Header.Set("Authorization", "sso-key "+p.Key+":"+p.Secret)
	req.Header.Set("Accept", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		result := registrarUnknown(domain, GoDaddyProviderName, err)
		result.VerificationEnv = environment
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return goDaddyAPIError(domain, environment, resp)
	}

	var payload struct {
		Available  bool   `json:"available"`
		Currency   string `json:"currency"`
		Definitive bool   `json:"definitive"`
		Domain     string `json:"domain"`
		Period     int    `json:"period"`
		Price      int64  `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return registrarUnknown(domain, GoDaddyProviderName, fmt.Errorf("decode response: %w", err))
	}

	result := model.Result{
		Domain:               domain,
		Available:            model.BoolPtr(payload.Available),
		Status:               model.StatusUnavailable,
		Source:               model.SourceHTTP,
		Currency:             payload.Currency,
		VerificationProvider: GoDaddyProviderName,
		VerificationEnv:      environment,
	}
	if payload.Period > 0 {
		result.RegistrationPeriod = model.IntPtr(payload.Period)
	}
	if payload.Price > 0 {
		price := float64(payload.Price) / 1_000_000
		result.Price = model.Float64Ptr(price)
	}

	if payload.Available {
		result.Status = model.StatusStandardAvailable
		result.PricingClass = "standard"
		if result.Price != nil && *result.Price > 100 {
			result.Status = model.StatusPremiumAvailable
			result.PricingClass = "premium"
		}
	}
	if !payload.Definitive && result.Status != model.StatusUnavailable {
		result.Status = model.StatusRegistrarUnknown
		result.Available = nil
	}

	return result
}

func goDaddyAPIError(domain string, environment string, resp *http.Response) model.Result {
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil && (payload.Code != "" || payload.Message != "") {
		msg := strings.TrimSpace(payload.Code + ": " + payload.Message)
		result := registrarUnknown(domain, GoDaddyProviderName, fmt.Errorf("godaddy api %s", strings.Trim(msg, ": ")))
		result.VerificationEnv = environment
		return result
	}

	result := registrarUnknown(domain, GoDaddyProviderName, fmt.Errorf("godaddy api status %s", resp.Status))
	result.VerificationEnv = environment
	return result
}

func NormalizeProviderName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
