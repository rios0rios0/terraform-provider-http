package entities

type Configuration struct {
	URL       string
	BasicAuth *BasicAuth
}

type BasicAuth struct {
	Username string
	Password string
}

func NewConfiguration(url string) *Configuration {
	return &Configuration{URL: url}
}

func (it *Configuration) HasAuthentication() bool {
	return it.BasicAuth != nil && it.BasicAuth.Username != "" && it.BasicAuth.Password != ""
}
