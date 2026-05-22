// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Command terraform-provider-infra serves the Terraform provider for the
// control-plane API.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/goabonga/infrastructure/internal/provider"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run with debugger support (attaches a reattach provider)")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(Version), providerserver.ServeOpts{
		Address: "registry.terraform.io/goabonga/infra",
		Debug:   debug,
	})
	if err != nil {
		log.Fatalf("terraform-provider-infra: %v", err)
	}
}
