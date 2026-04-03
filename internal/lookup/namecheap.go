package lookup

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/djvibe/domainfindr/internal/model"
)

const (
	NamecheapProviderName   = "namecheap"
	namecheapSandboxBaseURL = "https://api.sandbox.namecheap.com/xml.response"
)

type NamecheapProvider struct {
	BaseURL  string
	APIUser  string
	APIKey   string
	UserName string
	ClientIP string
	Client   *http.Client

	mu           sync.Mutex
	pricingCache map[string]namecheapPricing
}

func (p *NamecheapProvider) Kind() ProviderKind {
	return ProviderKindRegistrar
}

type namecheapPricing struct {
	price    *float64
	currency string
	period   *int
}

func NewNamecheapProvider(apiUser string, apiKey string, userName string, clientIP string, client *http.Client, baseURL string) *NamecheapProvider {
	if client == nil {
		client = newNamecheapHTTPClient()
	}
	if baseURL == "" {
		baseURL = namecheapSandboxBaseURL
	}
	if userName == "" {
		userName = apiUser
	}

	return &NamecheapProvider{
		BaseURL:      baseURL,
		APIUser:      apiUser,
		APIKey:       apiKey,
		UserName:     userName,
		ClientIP:     clientIP,
		Client:       client,
		pricingCache: make(map[string]namecheapPricing),
	}
}

func (p *NamecheapProvider) Check(ctx context.Context, domain string) model.Result {
	environment := providerEnvironment(NamecheapProviderName, p.BaseURL)
	checkResponse, err := p.call(ctx, url.Values{
		"Command":    []string{"namecheap.domains.check"},
		"DomainList": []string{domain},
	})
	if err != nil {
		result := registrarUnknown(domain, NamecheapProviderName, err)
		result.VerificationEnv = environment
		return result
	}

	if len(checkResponse.CommandResponse.DomainCheckResults) == 0 {
		result := registrarUnknown(domain, NamecheapProviderName, fmt.Errorf("namecheap response missing domain check result"))
		result.VerificationEnv = environment
		return result
	}

	item := checkResponse.CommandResponse.DomainCheckResults[0]
	result := model.Result{
		Domain:               domain,
		Available:            model.BoolPtr(item.Available),
		Source:               model.SourceHTTP,
		VerificationProvider: NamecheapProviderName,
		VerificationEnv:      environment,
	}

	if !item.Available {
		result.Status = model.StatusUnavailable
		return result
	}

	result.Status = model.StatusStandardAvailable
	result.PricingClass = "standard"

	if item.IsPremiumName {
		result.Status = model.StatusPremiumAvailable
		result.PricingClass = "premium"
		if price, err := strconv.ParseFloat(item.PremiumRegistrationPrice, 64); err == nil && price > 0 {
			result.Price = model.Float64Ptr(price)
			result.Currency = "USD"
		}
		return result
	}

	tld := domainTLD(domain)
	pricing, err := p.lookupRegisterPricing(ctx, tld)
	if err != nil {
		result.Error = model.StringPtr(err.Error())
		return result
	}
	result.Price = pricing.price
	result.Currency = pricing.currency
	result.RegistrationPeriod = pricing.period
	return result
}

func (p *NamecheapProvider) lookupRegisterPricing(ctx context.Context, tld string) (namecheapPricing, error) {
	p.mu.Lock()
	if pricing, ok := p.pricingCache[tld]; ok {
		p.mu.Unlock()
		return pricing, nil
	}
	p.mu.Unlock()

	response, err := p.call(ctx, url.Values{
		"Command":         []string{"namecheap.users.getPricing"},
		"ProductType":     []string{"DOMAIN"},
		"ProductCategory": []string{"DOMAINS"},
		"ActionName":      []string{"REGISTER"},
		"ProductName":     []string{strings.ToUpper(tld)},
	})
	if err != nil {
		return namecheapPricing{}, err
	}

	for _, category := range response.CommandResponse.UserGetPricingResult.ProductType.ProductCategories {
		if !strings.EqualFold(category.Name, "REGISTER") {
			continue
		}
		for _, product := range category.Products {
			if !strings.EqualFold(product.Name, tld) {
				continue
			}
			for _, price := range product.Prices {
				if price.Duration != 1 || !strings.EqualFold(price.DurationType, "YEAR") {
					continue
				}
				value, err := strconv.ParseFloat(price.Price, 64)
				if err != nil {
					return namecheapPricing{}, fmt.Errorf("parse namecheap price: %w", err)
				}
				pricing := namecheapPricing{
					price:    model.Float64Ptr(value),
					currency: price.Currency,
					period:   model.IntPtr(price.Duration),
				}
				p.mu.Lock()
				p.pricingCache[tld] = pricing
				p.mu.Unlock()
				return pricing, nil
			}
		}
	}

	return namecheapPricing{}, fmt.Errorf("namecheap pricing not found for .%s", tld)
}

func (p *NamecheapProvider) call(ctx context.Context, params url.Values) (namecheapAPIResponse, error) {
	query := url.Values{
		"ApiUser":  []string{p.APIUser},
		"ApiKey":   []string{p.APIKey},
		"UserName": []string{p.UserName},
		"ClientIp": []string{p.ClientIP},
	}
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}

	endpoint := p.BaseURL
	if !strings.Contains(endpoint, "?") {
		endpoint += "?"
	} else if !strings.HasSuffix(endpoint, "&") && !strings.HasSuffix(endpoint, "?") {
		endpoint += "&"
	}
	endpoint += query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return namecheapAPIResponse{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "text/xml, application/xml")
	req.Header.Set("User-Agent", "domainfindr/0.1 (+https://github.com/djvibe/domainfindr)")

	resp, err := p.Client.Do(req)
	if err != nil {
		return namecheapAPIResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return namecheapAPIResponse{}, fmt.Errorf("read response body: %w", err)
	}
	if len(body) == 0 {
		return namecheapAPIResponse{}, fmt.Errorf("empty response body")
	}

	var payload namecheapAPIResponse
	if err := xml.Unmarshal(body, &payload); err != nil {
		return namecheapAPIResponse{}, fmt.Errorf("decode xml response: %w", err)
	}
	if !strings.EqualFold(payload.Status, "OK") {
		return namecheapAPIResponse{}, fmt.Errorf("namecheap api error: %s", payload.errorString())
	}
	if len(payload.Errors) > 0 {
		return namecheapAPIResponse{}, fmt.Errorf("namecheap api error: %s", payload.errorString())
	}

	return payload, nil
}

type namecheapAPIResponse struct {
	XMLName xml.Name `xml:"ApiResponse"`
	Status  string   `xml:"Status,attr"`
	Errors  []struct {
		Number  string `xml:"Number,attr"`
		Message string `xml:",chardata"`
	} `xml:"Errors>Error"`
	CommandResponse struct {
		DomainCheckResults []struct {
			Domain                   string `xml:"Domain,attr"`
			Available                bool   `xml:"Available,attr"`
			IsPremiumName            bool   `xml:"IsPremiumName,attr"`
			PremiumRegistrationPrice string `xml:"PremiumRegistrationPrice,attr"`
		} `xml:"DomainCheckResult"`
		UserGetPricingResult struct {
			ProductType struct {
				ProductCategories []struct {
					Name     string `xml:"Name,attr"`
					Products []struct {
						Name   string `xml:"Name,attr"`
						Prices []struct {
							Duration     int    `xml:"Duration,attr"`
							DurationType string `xml:"DurationType,attr"`
							Price        string `xml:"Price,attr"`
							Currency     string `xml:"Currency,attr"`
						} `xml:"Price"`
					} `xml:"Product"`
				} `xml:"ProductCategory"`
			} `xml:"ProductType"`
		} `xml:"UserGetPricingResult"`
	} `xml:"CommandResponse"`
}

func (r namecheapAPIResponse) errorString() string {
	if len(r.Errors) == 0 {
		return "unknown error"
	}
	parts := make([]string, 0, len(r.Errors))
	for _, item := range r.Errors {
		msg := strings.TrimSpace(item.Message)
		if item.Number != "" {
			msg = item.Number + ": " + msg
		}
		parts = append(parts, strings.TrimSpace(msg))
	}
	return strings.Join(parts, "; ")
}

func domainTLD(domain string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(domain)), ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func newNamecheapHTTPClient() *http.Client {
	return &http.Client{
		Transport: http.DefaultTransport.(*http.Transport).Clone(),
	}
}
