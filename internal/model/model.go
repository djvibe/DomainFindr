package model

type Entry struct {
	Raw    string
	Domain string
	Valid  bool
}

type Result struct {
	Domain               string   `json:"domain"`
	Available            *bool    `json:"available"`
	Status               string   `json:"status"`
	Source               string   `json:"source"`
	RegistryStatus       string   `json:"registry_status,omitempty"`
	RegistrarStatus      string   `json:"registrar_status,omitempty"`
	RegistrarConsensus   string   `json:"registrar_consensus,omitempty"`
	PricingClass         string   `json:"pricing_class,omitempty"`
	Price                *float64 `json:"price,omitempty"`
	Currency             string   `json:"currency,omitempty"`
	RegistrationPeriod   *int     `json:"registration_period,omitempty"`
	VerificationProvider string   `json:"verification_provider,omitempty"`
	VerificationEnv      string   `json:"verification_environment,omitempty"`
	Verifications        []Check  `json:"verifications,omitempty"`
	Error                *string  `json:"error"`
	Transient            bool     `json:"-"`
	TimedOut             bool     `json:"-"`
}

type Check struct {
	Provider           string   `json:"provider"`
	Environment        string   `json:"environment,omitempty"`
	Source             string   `json:"source"`
	Available          *bool    `json:"available,omitempty"`
	Status             string   `json:"status"`
	PricingClass       string   `json:"pricing_class,omitempty"`
	Price              *float64 `json:"price,omitempty"`
	Currency           string   `json:"currency,omitempty"`
	RegistrationPeriod *int     `json:"registration_period,omitempty"`
	Error              *string  `json:"error,omitempty"`
	Transient          bool     `json:"-"`
	TimedOut           bool     `json:"-"`
}

const (
	StatusAvailable         = "available"
	StatusRegistered        = "registered"
	StatusInvalid           = "invalid"
	StatusLookupError       = "lookup_error"
	StatusStandardAvailable = "standard_available"
	StatusPremiumAvailable  = "premium_available"
	StatusUnavailable       = "unavailable"
	StatusRegistrarUnknown  = "registrar_unknown"

	ConsensusConsensus  = "consensus"
	ConsensusConflict   = "conflict"
	ConsensusIncomplete = "incomplete"

	SourceInput = "input"
	SourceRDAP  = "rdap"
	SourceHTTP  = "http"

	ProviderRDAP = "rdap"
)

func BoolPtr(v bool) *bool {
	return &v
}

func StringPtr(v string) *string {
	return &v
}

func Float64Ptr(v float64) *float64 {
	return &v
}

func IntPtr(v int) *int {
	return &v
}
