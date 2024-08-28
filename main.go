package main

import (
	"context"
	"flag"
	"log"

	"github.com/rios0rios0/terraform-provider-http/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// According to Terraform SDK documentation, the `main.go` should be on the root of the project.
// Otherwise, `tfplugindocs generate` will not work and `.goreleaser.yml` should be changed.

const version string = "1.0.0"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/rios0rios0/http",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
