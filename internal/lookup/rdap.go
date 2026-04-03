package lookup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/djvibe/domainfindr/internal/model"
)

type RDAPProvider struct {
	BaseURL string
	Client  *http.Client
}

func NewRDAPChecker(client *http.Client) *RDAPProvider {
	if client == nil {
		client = http.DefaultClient
	}

	return &RDAPProvider{
		BaseURL: "https://rdap.org",
		Client:  client,
	}
}

func (c *RDAPProvider) Check(ctx context.Context, domain string) model.Result {
	endpoint := strings.TrimRight(c.BaseURL, "/") + "/domain/" + url.PathEscape(domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return lookupError(domain, fmt.Errorf("build request: %w", err))
	}
	req.Header.Set("Accept", "application/rdap+json, application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return lookupError(domain, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return model.Result{
			Domain:               domain,
			Available:            model.BoolPtr(false),
			Status:               model.StatusRegistered,
			Source:               model.SourceRDAP,
			VerificationProvider: model.ProviderRDAP,
		}
	case http.StatusNotFound:
		return model.Result{
			Domain:               domain,
			Available:            model.BoolPtr(true),
			Status:               model.StatusAvailable,
			Source:               model.SourceRDAP,
			VerificationProvider: model.ProviderRDAP,
		}
	case http.StatusTooManyRequests:
		return lookupError(domain, errors.New("rdap rate limit exceeded"))
	default:
		msg := readProblemDetail(resp.Body)
		if msg == "" {
			msg = resp.Status
		}
		return lookupError(domain, fmt.Errorf("rdap lookup failed: %s", msg))
	}
}

func lookupError(domain string, err error) model.Result {
	return model.Result{
		Domain:               domain,
		Status:               model.StatusLookupError,
		Source:               model.SourceRDAP,
		VerificationProvider: model.ProviderRDAP,
		Error:                model.StringPtr(err.Error()),
	}
}

func readProblemDetail(r io.Reader) string {
	var payload struct {
		Description interface{} `json:"description"`
		Title       string      `json:"title"`
	}
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return ""
	}

	switch desc := payload.Description.(type) {
	case string:
		return desc
	case []interface{}:
		parts := make([]string, 0, len(desc))
		for _, item := range desc {
			if text, ok := item.(string); ok {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "; ")
		}
	}

	return payload.Title
}
