package entities

import (
	"crypto/tls"
	"net/http"
)

type InternalContext struct {
	Client *http.Client
	Config *Configuration
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
		Client: client,
		Config: config,
	}
}
