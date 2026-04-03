package lookup

import (
	"context"
	"errors"
	"testing"

	"github.com/djvibe/domainfindr/internal/model"
)

type stubProvider struct {
	result model.Result
}

func (s stubProvider) Check(context.Context, string) model.Result {
	return s.result
}

func TestVerifierFallsBackToRDAPWhenNoRegistrarProvider(t *testing.T) {
	t.Parallel()

	verifier := NewVerifier(
		stubProvider{
			result: model.Result{
				Domain:               "example.com",
				Available:            model.BoolPtr(true),
				Status:               model.StatusAvailable,
				Source:               model.SourceRDAP,
				VerificationProvider: model.ProviderRDAP,
			},
		},
	)

	result := verifier.Check(context.Background(), "example.com")

	if result.Status != model.StatusAvailable {
		t.Fatalf("expected rdap status, got %#v", result)
	}
	if result.RegistryStatus != model.StatusAvailable {
		t.Fatalf("expected registry status to be populated, got %#v", result)
	}
	if result.RegistrarStatus != "" {
		t.Fatalf("expected empty registrar status, got %#v", result)
	}
}

func TestVerifierPrefersRegistrarTruthWhenAvailable(t *testing.T) {
	t.Parallel()

	verifier := NewVerifier(
		stubProvider{
			result: model.Result{
				Domain:               "example.com",
				Available:            model.BoolPtr(true),
				Status:               model.StatusAvailable,
				Source:               model.SourceRDAP,
				VerificationProvider: model.ProviderRDAP,
			},
		},
		stubProvider{
			result: model.Result{
				Domain:               "example.com",
				Available:            model.BoolPtr(true),
				Status:               model.StatusPremiumAvailable,
				Source:               model.SourceHTTP,
				PricingClass:         "premium",
				Price:                model.Float64Ptr(249.0),
				Currency:             "USD",
				RegistrationPeriod:   model.IntPtr(1),
				VerificationProvider: "test_registrar",
			},
		},
	)

	result := verifier.Check(context.Background(), "example.com")

	if result.Status != model.StatusPremiumAvailable {
		t.Fatalf("expected registrar truth to win, got %#v", result)
	}
	if result.RegistryStatus != model.StatusAvailable || result.RegistrarStatus != model.StatusPremiumAvailable {
		t.Fatalf("expected both statuses, got %#v", result)
	}
	if result.Price == nil || *result.Price != 249.0 {
		t.Fatalf("expected price to be carried through, got %#v", result)
	}
}

func TestVerifierMarksRegistrarUnknownWithoutDiscardingRegistrySignal(t *testing.T) {
	t.Parallel()

	verifier := NewVerifier(
		stubProvider{
			result: model.Result{
				Domain:               "example.com",
				Available:            model.BoolPtr(true),
				Status:               model.StatusAvailable,
				Source:               model.SourceRDAP,
				VerificationProvider: model.ProviderRDAP,
			},
		},
		stubProvider{
			result: registrarUnknown("example.com", "test_registrar", errors.New("timeout")),
		},
	)

	result := verifier.Check(context.Background(), "example.com")

	if result.Status != model.StatusRegistrarUnknown {
		t.Fatalf("expected registrar unknown, got %#v", result)
	}
	if result.RegistryStatus != model.StatusAvailable {
		t.Fatalf("expected rdap status to remain visible, got %#v", result)
	}
	if result.Error == nil {
		t.Fatalf("expected registrar error detail, got %#v", result)
	}
}

func TestVerifierContinuesToRegistrarWhenRDAPFails(t *testing.T) {
	t.Parallel()

	verifier := NewVerifier(
		stubProvider{
			result: model.Result{
				Domain:               "example.com",
				Status:               model.StatusLookupError,
				Source:               model.SourceRDAP,
				Error:                model.StringPtr("rdap timeout"),
				VerificationProvider: model.ProviderRDAP,
			},
		},
		stubProvider{
			result: model.Result{
				Domain:               "example.com",
				Available:            model.BoolPtr(true),
				Status:               model.StatusStandardAvailable,
				Source:               model.SourceHTTP,
				PricingClass:         "standard",
				Price:                model.Float64Ptr(10.69),
				Currency:             "USD",
				RegistrationPeriod:   model.IntPtr(1),
				VerificationProvider: "godaddy",
			},
		},
	)

	result := verifier.Check(context.Background(), "example.com")

	if result.Status != model.StatusStandardAvailable {
		t.Fatalf("expected registrar result to win after rdap failure, got %#v", result)
	}
	if result.RegistryStatus != model.StatusLookupError {
		t.Fatalf("expected rdap failure to remain visible, got %#v", result)
	}
	if result.RegistrarStatus != model.StatusStandardAvailable {
		t.Fatalf("expected registrar status to be populated, got %#v", result)
	}
}

func TestVerifierPrefersCleanPremiumOverErroredStandard(t *testing.T) {
	t.Parallel()

	verifier := NewVerifier(
		stubProvider{
			result: model.Result{
				Domain:               "allspas.ai",
				Available:            model.BoolPtr(true),
				Status:               model.StatusAvailable,
				Source:               model.SourceRDAP,
				VerificationProvider: model.ProviderRDAP,
			},
		},
		stubProvider{
			result: model.Result{
				Domain:               "allspas.ai",
				Available:            model.BoolPtr(true),
				Status:               model.StatusPremiumAvailable,
				Source:               model.SourceHTTP,
				PricingClass:         "premium",
				Price:                model.Float64Ptr(423.98),
				Currency:             "USD",
				RegistrationPeriod:   model.IntPtr(2),
				VerificationProvider: "godaddy",
			},
		},
		stubProvider{
			result: model.Result{
				Domain:               "allspas.ai",
				Available:            model.BoolPtr(true),
				Status:               model.StatusStandardAvailable,
				Source:               model.SourceHTTP,
				PricingClass:         "standard",
				Error:                model.StringPtr("pricing not found"),
				VerificationProvider: "namecheap",
			},
		},
	)

	result := verifier.Check(context.Background(), "allspas.ai")

	if result.RegistrarStatus != model.StatusPremiumAvailable || result.Status != model.StatusPremiumAvailable {
		t.Fatalf("expected premium result to survive errored standard result, got %#v", result)
	}
	if result.Price == nil || *result.Price != 423.98 {
		t.Fatalf("expected premium pricing to survive, got %#v", result)
	}
	if result.VerificationProvider != "godaddy" {
		t.Fatalf("expected strongest provider to remain selected, got %#v", result)
	}
}
