package main

/* Bootstrap the plugin for Terraform */

import (
	"github.com/exoscale/terraform-provider-exoscale/exoscale"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: exoscale.Provider,
	})
}
