package main

import (
	"context"
	"flag"
	"log"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/exoscale/terraform-provider-exoscale/pkg/provider"
)

//go:generate ./scripts/fmt.sh

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	var debugMode bool
	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(
		context.Background(),
		func() fwprovider.Provider { return &provider.ExoscaleProvider{} },
		providerserver.ServeOpts{
			Address: "registry.terraform.io/exoscale/exoscale",
			Debug:   debugMode,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}
