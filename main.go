package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"

	"github.com/exoscale/terraform-provider-exoscale/exoscale"
	"github.com/exoscale/terraform-provider-exoscale/pkg/provider"
)

//go:generate terraform fmt -recursive ./examples/

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var debugMode bool
	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	ctx := context.Background()

	upgradedProvider, err := tf5to6server.UpgradeServer(
		ctx,
		exoscale.Provider().GRPCProvider,
	)
	check(err)

	newProvider := providerserver.NewProtocol6(&provider.ExoscaleProvider{})
	providers := []func() tfprotov6.ProviderServer{
		func() tfprotov6.ProviderServer {
			return upgradedProvider
		},
		newProvider,
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	check(err)

	var serveOpts []tf6server.ServeOpt

	if debugMode {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	err = tf6server.Serve(
		"registry.terraform.io/exoscale/exoscale",
		muxServer.ProviderServer,
		serveOpts...,
	)
	check(err)
}
