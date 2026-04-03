package lookup

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/djvibe/domainfindr/internal/model"
)

func TestGoDaddyProviderStandardAvailable(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Authorization") != "sso-key key:secret" {
				t.Fatalf("unexpected auth header %q", req.Header.Get("Authorization"))
			}
			if req.URL.Path != "/v1/domains/available" {
				t.Fatalf("unexpected path %s", req.URL.Path)
			}
			if req.URL.Query().Get("checkType") != "FULL" {
				t.Fatalf("unexpected checkType %q", req.URL.Query().Get("checkType"))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"domain":"example.com",
					"available":true,
					"definitive":true,
					"currency":"USD",
					"period":1,
					"price":12990000
				}`)),
				Header: make(http.Header),
			}, nil
		}),
	}

	provider := NewGoDaddyProvider("key", "secret", client, "https://api.godaddy.test")
	result := provider.Check(context.Background(), "example.com")

	if result.Status != model.StatusStandardAvailable {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Price == nil || *result.Price != 12.99 {
		t.Fatalf("expected price conversion, got %#v", result)
	}
	if result.VerificationProvider != GoDaddyProviderName {
		t.Fatalf("unexpected provider: %#v", result)
	}
	if result.VerificationEnv != EnvironmentCustom {
		t.Fatalf("expected custom environment, got %#v", result)
	}
}

func TestGoDaddyProviderPremiumAvailable(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"domain":"example.ai",
					"available":true,
					"definitive":true,
					"currency":"USD",
					"period":1,
					"price":349990000
				}`)),
				Header: make(http.Header),
			}, nil
		}),
	}

	provider := NewGoDaddyProvider("key", "secret", client, "https://api.godaddy.test")
	result := provider.Check(context.Background(), "example.ai")

	if result.Status != model.StatusPremiumAvailable || result.PricingClass != "premium" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestGoDaddyProviderUnavailable(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"domain":"taken.com","available":false,"definitive":true}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	provider := NewGoDaddyProvider("key", "secret", client, "https://api.godaddy.test")
	result := provider.Check(context.Background(), "taken.com")

	if result.Status != model.StatusUnavailable || result.Available == nil || *result.Available {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestGoDaddyProviderUnknownWhenNotDefinitive(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"domain":"maybe.com","available":true,"definitive":false}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	provider := NewGoDaddyProvider("key", "secret", client, "https://api.godaddy.test")
	result := provider.Check(context.Background(), "maybe.com")

	if result.Status != model.StatusRegistrarUnknown {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestGoDaddyProviderMarksTransientHTTPFailures(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader(`{"code":"SERVICE_UNAVAILABLE","message":"try again later"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	provider := NewGoDaddyProvider("key", "secret", client, "https://api.godaddy.test")
	result := provider.Check(context.Background(), "maybe.com")

	if result.Status != model.StatusRegistrarUnknown || !result.Transient {
		t.Fatalf("expected transient registrar_unknown, got %#v", result)
	}
}
