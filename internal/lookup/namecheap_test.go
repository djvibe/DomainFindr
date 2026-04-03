package lookup

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/djvibe/domainfindr/internal/model"
)

func TestNamecheapProviderStandardAvailableUsesPricingLookup(t *testing.T) {
	t.Parallel()

	var calls int
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.Query().Get("ApiUser") != "api-user" {
				t.Fatalf("unexpected ApiUser %q", req.URL.Query().Get("ApiUser"))
			}
			if req.URL.Query().Get("UserName") != "api-user" {
				t.Fatalf("unexpected UserName %q", req.URL.Query().Get("UserName"))
			}
			if req.URL.Query().Get("ClientIp") != "203.0.113.10" {
				t.Fatalf("unexpected ClientIp %q", req.URL.Query().Get("ClientIp"))
			}
			switch req.URL.Query().Get("Command") {
			case "namecheap.domains.check":
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`<?xml version="1.0" encoding="utf-8"?>
<ApiResponse xmlns="http://api.namecheap.com/xml.response" Status="OK">
  <Errors />
  <CommandResponse Type="namecheap.domains.check">
    <DomainCheckResult Domain="example.com" Available="true" ErrorNo="0" Description="" IsPremiumName="false" PremiumRegistrationPrice="0" />
  </CommandResponse>
</ApiResponse>`)),
					Header: make(http.Header),
				}, nil
			case "namecheap.users.getPricing":
				if req.URL.Query().Get("ProductName") != "COM" {
					t.Fatalf("unexpected ProductName %q", req.URL.Query().Get("ProductName"))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`<?xml version="1.0" encoding="UTF-8"?>
<ApiResponse xmlns="http://api.namecheap.com/xml.response" Status="OK">
  <Errors />
  <CommandResponse Type="namecheap.users.getPricing">
    <UserGetPricingResult>
      <ProductType Name="DOMAIN">
        <ProductCategory Name="REGISTER">
          <Product Name="com">
            <Price Duration="1" DurationType="YEAR" Price="12.98" Currency="USD" />
          </Product>
        </ProductCategory>
      </ProductType>
    </UserGetPricingResult>
  </CommandResponse>
</ApiResponse>`)),
					Header: make(http.Header),
				}, nil
			default:
				t.Fatalf("unexpected command %q", req.URL.Query().Get("Command"))
			}
			return nil, nil
		}),
	}

	provider := NewNamecheapProvider("api-user", "api-key", "", "203.0.113.10", client, "https://api.sandbox.namecheap.test/xml.response")
	result := provider.Check(context.Background(), "example.com")

	if result.Status != model.StatusStandardAvailable {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Price == nil || *result.Price != 12.98 {
		t.Fatalf("expected register pricing, got %#v", result)
	}
	if calls != 2 {
		t.Fatalf("expected 2 API calls, got %d", calls)
	}
	if result.VerificationEnv != EnvironmentSandbox {
		t.Fatalf("expected sandbox environment, got %#v", result)
	}
}

func TestNamecheapProviderPremiumAvailable(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`<?xml version="1.0" encoding="utf-8"?>
<ApiResponse xmlns="http://api.namecheap.com/xml.response" Status="OK">
  <Errors />
  <CommandResponse Type="namecheap.domains.check">
    <DomainCheckResult Domain="us.xyz" Available="true" ErrorNo="0" Description="" IsPremiumName="true" PremiumRegistrationPrice="13000.0000" />
  </CommandResponse>
</ApiResponse>`)),
				Header: make(http.Header),
			}, nil
		}),
	}

	provider := NewNamecheapProvider("api-user", "api-key", "ncuser", "203.0.113.10", client, "")
	result := provider.Check(context.Background(), "us.xyz")

	if result.Status != model.StatusPremiumAvailable || result.Price == nil || *result.Price != 13000 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestNamecheapProviderUnavailable(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`<?xml version="1.0" encoding="utf-8"?>
<ApiResponse xmlns="http://api.namecheap.com/xml.response" Status="OK">
  <Errors />
  <CommandResponse Type="namecheap.domains.check">
    <DomainCheckResult Domain="taken.com" Available="false" ErrorNo="0" Description="" IsPremiumName="false" PremiumRegistrationPrice="0" />
  </CommandResponse>
</ApiResponse>`)),
				Header: make(http.Header),
			}, nil
		}),
	}

	provider := NewNamecheapProvider("api-user", "api-key", "ncuser", "203.0.113.10", client, "")
	result := provider.Check(context.Background(), "taken.com")

	if result.Status != model.StatusUnavailable || result.Available == nil || *result.Available {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestNamecheapProviderAPIErrorReturnsRegistrarUnknown(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`<?xml version="1.0" encoding="utf-8"?>
<ApiResponse xmlns="http://api.namecheap.com/xml.response" Status="ERROR">
  <Errors>
    <Error Number="2019166">Validation error from api user</Error>
  </Errors>
</ApiResponse>`)),
				Header: make(http.Header),
			}, nil
		}),
	}

	provider := NewNamecheapProvider("api-user", "api-key", "ncuser", "203.0.113.10", client, "")
	result := provider.Check(context.Background(), "broken.com")

	if result.Status != model.StatusRegistrarUnknown || result.Error == nil {
		t.Fatalf("unexpected result: %#v", result)
	}
}
