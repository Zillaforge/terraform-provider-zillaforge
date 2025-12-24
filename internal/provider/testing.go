// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
)

// TestAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
// This is exported for use in other test packages (e.g., vps_test).
var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"zillaforge": providerserver.NewProtocol6WithError(New("test")()),
}

// TestAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the scaffolding provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
// This is exported for use in other test packages (e.g., scaffolding_test).
var TestAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"zillaforge": providerserver.NewProtocol6WithError(New("test")()),
	"echo":       echoprovider.NewProviderServer(),
}

// TestAccPreCheck runs pre-check validation before acceptance tests.
// This is exported for use in other test packages (e.g., vps_test).
func TestAccPreCheck(t *testing.T) {
	apiKey := os.Getenv("ZILLAFORGE_API_KEY")
	projectID := os.Getenv("ZILLAFORGE_PROJECT_ID")
	projectSys := os.Getenv("ZILLAFORGE_PROJECT_SYS_CODE")

	if apiKey == "" || (projectID == "" && projectSys == "") {
		t.Skip("Zillaforge API credentials or project not configured; skipping acceptance test")
	}
}
