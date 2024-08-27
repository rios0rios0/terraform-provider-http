package main

import (
	"context"
	"flag"
	"github.com/rios0rios0/terraform-provider-http/internal"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	version string = "1.0.0"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/rios0rios0/http",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), internal.NewProvider(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
