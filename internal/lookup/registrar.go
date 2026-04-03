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

type RegistrarHTTPProvider struct {
	Name    string
	BaseURL string
	Client  *http.Client
}

func NewRegistrarHTTPProvider(name string, client *http.Client, baseURL string) *RegistrarHTTPProvider {
	if client == nil {
		client = http.DefaultClient
	}
	if name == "" {
		name = "registrar_http"
	}

	return &RegistrarHTTPProvider{
		Name:    name,
		BaseURL: baseURL,
		Client:  client,
	}
}

func (p *RegistrarHTTPProvider) Check(ctx context.Context, domain string) model.Result {
	endpoint := strings.TrimRight(p.BaseURL, "/") + "/availability/" + url.PathEscape(domain)
	environment := providerEnvironment(p.Name, p.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		result := registrarUnknown(domain, p.Name, fmt.Errorf("build request: %w", err))
		result.VerificationEnv = environment
		return result
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		result := registrarUnknown(domain, p.Name, err)
		result.VerificationEnv = environment
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := registrarUnknown(domain, p.Name, fmt.Errorf("registrar lookup failed: %s", resp.Status))
		result.VerificationEnv = environment
		return result
	}

	var payload struct {
		Available          *bool    `json:"available"`
		Status             string   `json:"status"`
		PricingClass       string   `json:"pricing_class"`
		Price              *float64 `json:"price"`
		Currency           string   `json:"currency"`
		RegistrationPeriod *int     `json:"registration_period"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		result := registrarUnknown(domain, p.Name, fmt.Errorf("decode registrar response: %w", err))
		result.VerificationEnv = environment
		return result
	}

	result := model.Result{
		Domain:               domain,
		Available:            payload.Available,
		Status:               normalizeRegistrarStatus(payload.Status, payload.Available, payload.PricingClass),
		Source:               model.SourceHTTP,
		PricingClass:         payload.PricingClass,
		Price:                payload.Price,
		Currency:             payload.Currency,
		RegistrationPeriod:   payload.RegistrationPeriod,
		VerificationProvider: p.Name,
		VerificationEnv:      environment,
	}
	if result.Status == model.StatusPremiumAvailable && result.PricingClass == "" {
		result.PricingClass = "premium"
	}
	if result.Status == model.StatusStandardAvailable && result.PricingClass == "" {
		result.PricingClass = "standard"
	}
	return result
}

func normalizeRegistrarStatus(status string, available *bool, pricingClass string) string {
	switch status {
	case model.StatusStandardAvailable, model.StatusPremiumAvailable, model.StatusUnavailable, model.StatusRegistrarUnknown:
		return status
	}

	if available == nil {
		return model.StatusRegistrarUnknown
	}
	if !*available {
		return model.StatusUnavailable
	}
	if strings.EqualFold(pricingClass, "premium") {
		return model.StatusPremiumAvailable
	}
	return model.StatusStandardAvailable
}

func registrarUnknown(domain string, provider string, err error) model.Result {
	return model.Result{
		Domain:               domain,
		Status:               model.StatusRegistrarUnknown,
		Source:               model.SourceHTTP,
		VerificationProvider: provider,
		Error:                model.StringPtr(err.Error()),
	}
}
