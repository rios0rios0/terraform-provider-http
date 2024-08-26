package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	providerConfig = `
provider "http" {
  url     = "http://localhost:19090"
  basic_auth = {
    username = "anything"
    password = "anything"
  }
  ignore_tls = true
}
`
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"http": providerserver.NewProtocol6WithError(New("test")()),
	}
)
