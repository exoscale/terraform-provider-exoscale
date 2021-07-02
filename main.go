package main

import (
	"github.com/exoscale/terraform-provider-exoscale/exoscale"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: exoscale.Provider,
	})
}
