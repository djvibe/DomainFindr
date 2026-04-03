package lookup

import (
	"context"

	"github.com/djvibe/domainfindr/internal/model"
)

type Checker interface {
	Check(context.Context, string) model.Result
}

type Provider interface {
	Check(context.Context, string) model.Result
}

type Verifier struct {
	providers []Provider
}

func NewVerifier(providers ...Provider) *Verifier {
	filtered := make([]Provider, 0, len(providers))
	for _, provider := range providers {
		if provider != nil {
			filtered = append(filtered, provider)
		}
	}
	return &Verifier{providers: filtered}
}

func (v *Verifier) Check(ctx context.Context, domain string) model.Result {
	result := model.Result{
		Domain: domain,
		Source: model.SourceInput,
	}
	hasRegistrarProvider := false

	for _, provider := range v.providers {
		current := provider.Check(ctx, domain)
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

	result.RegistrarConsensus = summarizeRegistrarConsensus(result.Verifications)

	if result.Status == "" {
		result.Status = model.StatusLookupError
		result.Source = model.SourceInput
		result.Error = model.StringPtr("no availability providers configured")
	}

	return result
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
