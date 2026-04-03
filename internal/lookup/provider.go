package lookup

import (
	"context"
	"time"

	"github.com/djvibe/domainfindr/internal/model"
)

type Checker interface {
	Check(context.Context, string) model.Result
}

type Provider interface {
	Check(context.Context, string) model.Result
	Kind() ProviderKind
}

type ProviderKind string

const (
	ProviderKindRDAP      ProviderKind = "rdap"
	ProviderKindRegistrar ProviderKind = "registrar"
)

type VerifierConfig struct {
	RegistrarRetry   int
	RegistrarTimeout time.Duration
	RegistrarRecheck int
}

type Verifier struct {
	providers []Provider
	config    VerifierConfig
}

func NewVerifier(providers ...Provider) *Verifier {
	return NewVerifierWithConfig(VerifierConfig{}, providers...)
}

func NewVerifierWithConfig(cfg VerifierConfig, providers ...Provider) *Verifier {
	filtered := make([]Provider, 0, len(providers))
	for _, provider := range providers {
		if provider != nil {
			filtered = append(filtered, provider)
		}
	}
	return &Verifier{
		providers: filtered,
		config:    cfg,
	}
}

func (v *Verifier) Check(ctx context.Context, domain string) model.Result {
	providerResults := make([]model.Result, len(v.providers))
	recheckTargets := make([]int, 0)

	for i, provider := range v.providers {
		current := v.checkProvider(ctx, domain, provider)
		providerResults[i] = current
		if shouldRecheckRegistrarResult(current) {
			recheckTargets = append(recheckTargets, i)
		}
	}

	if v.config.RegistrarRecheck > 0 && len(recheckTargets) > 0 {
		remaining := recheckTargets
		for pass := 0; pass < v.config.RegistrarRecheck && len(remaining) > 0; pass++ {
			next := make([]int, 0)
			for _, idx := range remaining {
				current := v.checkProvider(ctx, domain, v.providers[idx])
				providerResults[idx] = current
				if shouldRecheckRegistrarResult(current) {
					next = append(next, idx)
				}
			}
			remaining = next
		}
	}

	result := mergeProviderResults(domain, providerResults)
	result.RegistrarConsensus = summarizeRegistrarConsensus(result.Verifications)

	if result.Status == "" {
		result.Status = model.StatusLookupError
		result.Source = model.SourceInput
		result.Error = model.StringPtr("no availability providers configured")
	}

	return result
}

func (v *Verifier) checkProvider(ctx context.Context, domain string, provider Provider) model.Result {
	if provider.Kind() != ProviderKindRegistrar {
		return provider.Check(ctx, domain)
	}

	attempts := v.config.RegistrarRetry + 1
	if attempts < 1 {
		attempts = 1
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		attemptCtx := ctx
		cancel := func() {}
		if v.config.RegistrarTimeout > 0 {
			attemptCtx, cancel = context.WithTimeout(ctx, v.config.RegistrarTimeout)
		}
		result := provider.Check(attemptCtx, domain)
		cancel()

		if !shouldRetryRegistrarResult(result) || attempt == attempts {
			return result
		}
	}

	return registrarUnknown(domain, "registrar", context.DeadlineExceeded)
}

func mergeProviderResults(domain string, providerResults []model.Result) model.Result {
	result := model.Result{
		Domain: domain,
		Source: model.SourceInput,
	}
	hasRegistrarProvider := false

	for _, current := range providerResults {
		result.Verifications = append(result.Verifications, model.Check{
			Provider:           current.VerificationProvider,
			Environment:        current.VerificationEnv,
			Source:             current.Source,
			Available:          current.Available,
			Status:             current.Status,
			PricingClass:       current.PricingClass,
			Price:              current.Price,
			Currency:           current.Currency,
			RegistrationPeriod: current.RegistrationPeriod,
			Error:              current.Error,
			Transient:          current.Transient,
			TimedOut:           current.TimedOut,
		})
		switch current.Source {
		case model.SourceRDAP:
			result.RegistryStatus = current.Status
			if current.Status == model.StatusLookupError {
				if !hasRegistrarProvider {
					result.Status = current.Status
					result.Source = current.Source
					result.Error = current.Error
				}
				continue
			}
			if result.Status == "" {
				result.Available = current.Available
				result.Status = current.Status
				result.Source = current.Source
				result.Error = current.Error
			}
		default:
			hasRegistrarProvider = true
			applyRegistrarTruth(&result, current)
		}
	}

	return result
}

func shouldRetryRegistrarResult(result model.Result) bool {
	return result.Status == model.StatusRegistrarUnknown && (result.Transient || result.TimedOut)
}

func shouldRecheckRegistrarResult(result model.Result) bool {
	return shouldRetryRegistrarResult(result)
}

func applyRegistrarTruth(result *model.Result, registrar model.Result) {
	if shouldReplaceRegistrarResult(*result, registrar) {
		result.RegistrarStatus = registrar.Status
		result.VerificationProvider = registrar.VerificationProvider
		result.VerificationEnv = registrar.VerificationEnv
		result.PricingClass = registrar.PricingClass
		result.Price = registrar.Price
		result.Currency = registrar.Currency
		result.RegistrationPeriod = registrar.RegistrationPeriod
		result.Error = registrar.Error
	}

	switch result.RegistrarStatus {
	case model.StatusStandardAvailable, model.StatusPremiumAvailable:
		result.Available = model.BoolPtr(true)
		result.Status = result.RegistrarStatus
		result.Source = registrar.Source
	case model.StatusUnavailable:
		result.Available = model.BoolPtr(false)
		result.Status = result.RegistrarStatus
		result.Source = registrar.Source
	case model.StatusRegistrarUnknown:
		result.Available = nil
		result.Status = result.RegistrarStatus
		result.Source = registrar.Source
	}
}

func shouldReplaceRegistrarResult(current model.Result, candidate model.Result) bool {
	if current.RegistrarStatus == "" {
		return true
	}
	return registrarRank(candidate) > registrarRank(model.Result{
		Status:             current.RegistrarStatus,
		VerificationEnv:    current.VerificationEnv,
		PricingClass:       current.PricingClass,
		Price:              current.Price,
		Currency:           current.Currency,
		RegistrationPeriod: current.RegistrationPeriod,
		Error:              current.Error,
	})
}

func registrarRank(result model.Result) int {
	score := 0
	switch result.Status {
	case model.StatusUnavailable:
		score = 70
	case model.StatusPremiumAvailable:
		score = 55
	case model.StatusStandardAvailable:
		score = 45
	case model.StatusRegistrarUnknown:
		score = 10
	}

	switch result.VerificationEnv {
	case EnvironmentProduction:
		score += 25
	case EnvironmentCustom:
		score += 10
	case EnvironmentOTE:
		score += 5
	case EnvironmentSandbox:
		score += 2
	}

	if result.Error == nil || *result.Error == "" {
		score += 8
	} else {
		score -= 10
	}

	if hasCompleteRegistrarPricing(result) {
		score += 12
	} else if result.Status == model.StatusStandardAvailable || result.Status == model.StatusPremiumAvailable {
		score -= 8
	}

	return score
}

func hasCompleteRegistrarPricing(result model.Result) bool {
	if result.Status != model.StatusStandardAvailable && result.Status != model.StatusPremiumAvailable {
		return true
	}
	return result.Price != nil && result.Currency != "" && result.RegistrationPeriod != nil
}

func summarizeRegistrarConsensus(checks []model.Check) string {
	statuses := make([]string, 0)
	hasIncompleteEvidence := false

	for _, check := range checks {
		if check.Provider == "" || check.Provider == model.ProviderRDAP || check.Status == "" {
			continue
		}
		statuses = append(statuses, check.Status)
		if registrarCheckIncomplete(check) {
			hasIncompleteEvidence = true
		}
	}

	if len(statuses) == 0 {
		return ""
	}
	if hasIncompleteEvidence {
		return model.ConsensusIncomplete
	}

	first := statuses[0]
	for _, status := range statuses[1:] {
		if status != first {
			return model.ConsensusConflict
		}
	}
	return model.ConsensusConsensus
}

func registrarCheckIncomplete(check model.Check) bool {
	if check.Error != nil && *check.Error != "" {
		return true
	}
	if check.Environment == EnvironmentSandbox || check.Environment == EnvironmentOTE || check.Environment == EnvironmentCustom {
		return true
	}
	if check.Status == model.StatusStandardAvailable || check.Status == model.StatusPremiumAvailable {
		return check.Price == nil || check.Currency == "" || check.RegistrationPeriod == nil
	}
	return false
}
