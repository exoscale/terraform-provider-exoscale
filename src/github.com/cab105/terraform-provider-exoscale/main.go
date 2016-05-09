package main

/* Bootstrap the plugin for Terraform */

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/cab105/terraform-provider-exoscale/exoscale"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: exoscale.Provider,
	})
}