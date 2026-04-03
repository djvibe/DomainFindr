package lookup

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/djvibe/domainfindr/internal/model"
)

func TestRegistrarHTTPProviderPremiumAvailable(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/availability/example.com" {
				t.Fatalf("unexpected path %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"available": true,
					"pricing_class": "premium",
					"price": 299.99,
					"currency": "USD",
					"registration_period": 1
				}`)),
				Header: make(http.Header),
			}, nil
		}),
	}

	provider := NewRegistrarHTTPProvider("test_registrar", client, "https://registrar.test")
	result := provider.Check(context.Background(), "example.com")

	if result.Status != model.StatusPremiumAvailable {
		t.Fatalf("expected premium availability, got %#v", result)
	}
	if result.Price == nil || *result.Price != 299.99 {
		t.Fatalf("expected price, got %#v", result)
	}
	if result.RegistrationPeriod == nil || *result.RegistrationPeriod != 1 {
		t.Fatalf("expected registration period, got %#v", result)
	}
}

func TestRegistrarHTTPProviderUnavailable(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"available": false}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	provider := NewRegistrarHTTPProvider("test_registrar", client, "https://registrar.test")
	result := provider.Check(context.Background(), "taken.com")

	if result.Status != model.StatusUnavailable || result.Available == nil || *result.Available {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestRegistrarHTTPProviderBadResponseReturnsUnknown(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader(`bad gateway`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	provider := NewRegistrarHTTPProvider("test_registrar", client, "https://registrar.test")
	result := provider.Check(context.Background(), "broken.com")

	if result.Status != model.StatusRegistrarUnknown || result.Error == nil {
		t.Fatalf("unexpected result: %#v", result)
	}
}
