package main

import (
	"flag"

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
		plugin.Serve(&plugin.ServeOpts{
			ProviderAddr: "registry.terraform.io/exoscale/exoscale",
			ProviderFunc: exoscale.Provider,
			Debug:        true,
		})
	} else {
		plugin.Serve(&plugin.ServeOpts{ProviderFunc: exoscale.Provider})
	}
}
