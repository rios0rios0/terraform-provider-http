package provider

import (
	"crypto/tls"
	"net/http"
)

type InternalContext struct {
	client *http.Client
	config *Configuration
}

type Configuration struct {
	URL       string
	BasicAuth *BasicAuth
}

func NewConfiguration(url string) *Configuration {
	return &Configuration{URL: url}
}

func (it *Configuration) HasAuthentication() bool {
	return it.BasicAuth != nil && it.BasicAuth.Username != "" && it.BasicAuth.Password != ""
}

type BasicAuth struct {
	Username string
	Password string
}

func NewInternalContext(ignoreTLS bool, config *Configuration) *InternalContext {
	client := &http.Client{}
	if ignoreTLS {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		client.Transport = transport
	}

	return &InternalContext{
		client: client,
		config: config,
	}
}
