package internal

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	providerConfig = `
provider "http" {
  url     = "https://jsonplaceholder.typicode.com"
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
		"http": providerserver.NewProtocol6WithError(NewProvider("test")()),
	}
)
