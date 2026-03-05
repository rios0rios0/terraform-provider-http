package entities

type Configuration struct {
	URL       string
	BasicAuth *BasicAuth
}

type BasicAuth struct {
	Username string
	Password string //nolint:gosec // G117: this is a struct field name, not a hardcoded credential
}

func NewConfiguration(url string) *Configuration {
	return &Configuration{URL: url}
}

func (it *Configuration) HasAuthentication() bool {
	return it.BasicAuth != nil && it.BasicAuth.Username != "" && it.BasicAuth.Password != ""
}
