package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/exoscale/terraform-provider-exoscale/exoscale"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: exoscale.Provider,
	})
}
