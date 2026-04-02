package model

type Entry struct {
	Raw    string
	Domain string
	Valid  bool
}

type Result struct {
	Domain    string  `json:"domain"`
	Available *bool   `json:"available"`
	Status    string  `json:"status"`
	Source    string  `json:"source"`
	Error     *string `json:"error"`
}

const (
	StatusAvailable   = "available"
	StatusRegistered  = "registered"
	StatusInvalid     = "invalid"
	StatusLookupError = "lookup_error"

	SourceInput = "input"
	SourceRDAP  = "rdap"
)

func BoolPtr(v bool) *bool {
	return &v
}

func StringPtr(v string) *string {
	return &v
}
