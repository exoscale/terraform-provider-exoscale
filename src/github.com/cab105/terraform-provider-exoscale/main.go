package main

/* Bootstrap the plugin for Terraform */

import (
	"github.com/cab105/terraform-provider-exoscale/exoscale"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: exoscale.Provider,
	})
}
