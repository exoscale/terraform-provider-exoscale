package main

import (
	"context"
	"flag"
	"log"

	"github.com/exoscale/terraform-provider-exoscale/exoscale"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

//go:generate terraform fmt -recursive ./examples/

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	var debugMode bool
	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	if debugMode {
		err := plugin.Debug(context.Background(), "registry.terraform.io/exoscale/exoscale", &plugin.ServeOpts{ProviderFunc: exoscale.Provider})
		if err != nil {
			log.Fatal(err.Error())
		}
	} else {
		plugin.Serve(&plugin.ServeOpts{ProviderFunc: exoscale.Provider})
	}
}
