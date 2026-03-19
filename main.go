package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/rios0rios0/terraform-provider-http/internal/provider"
)

// According to Terraform SDK documentation, the `main.go` should be on the root of the project.
// Otherwise, `tfplugindocs generate` will not work and `.goreleaser.yml` should be changed.

// version is set at build time via ldflags by GoReleaser.
// During development, it defaults to "dev".
var version = "dev"

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
