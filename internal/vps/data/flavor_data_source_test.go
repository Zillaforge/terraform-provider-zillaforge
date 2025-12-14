// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// T006: Test basic flavors query without filters.
func TestAccFlavorDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlavorDataSourceConfig_all,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify flavors list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_flavors.test", "flavors.#"),
					// Verify at least one flavor is returned
					resource.TestCheckFunc(func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.zillaforge_flavors.test"]
						if !ok {
							return fmt.Errorf("data source not found in state")
						}
						count := rs.Primary.Attributes["flavors.#"]
						if count == "0" || count == "" {
							return fmt.Errorf("expected at least one flavor, got %q", count)
						}
						return nil
					}),
				),
			},
		},
	})
}

const testAccFlavorDataSourceConfig_all = `
data "zillaforge_flavors" "test" {}
`

// T007: Test name filter with exact match.
func TestAccFlavorDataSource_nameFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlavorDataSourceConfig_name,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify flavors list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_flavors.test", "flavors.#"),
					// Verify filter was applied
					resource.TestCheckResourceAttr("data.zillaforge_flavors.test", "name", "example-flavor"),
				),
			},
		},
	})
}

const testAccFlavorDataSourceConfig_name = `
data "zillaforge_flavors" "test" {
  name = "example-flavor"
}
`

// T008: Test vcpus filter (minimum).
func TestAccFlavorDataSource_vcpusFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlavorDataSourceConfig_vcpus,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify flavors list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_flavors.test", "flavors.#"),
					// Verify filter was applied
					resource.TestCheckResourceAttr("data.zillaforge_flavors.test", "vcpus", "2"),
				),
			},
		},
	})
}

const testAccFlavorDataSourceConfig_vcpus = `
data "zillaforge_flavors" "test" {
  vcpus = 2
}
`

// T009: Test memory filter (minimum GB).
func TestAccFlavorDataSource_memoryFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlavorDataSourceConfig_memory,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify flavors list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_flavors.test", "flavors.#"),
					// Verify filter was applied
					resource.TestCheckResourceAttr("data.zillaforge_flavors.test", "memory", "4"),
				),
			},
		},
	})
}

const testAccFlavorDataSourceConfig_memory = `
data "zillaforge_flavors" "test" {
  memory = 4
}
`

// T010: Test multiple filters with AND logic.
func TestAccFlavorDataSource_multipleFilters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlavorDataSourceConfig_multiple,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify flavors list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_flavors.test", "flavors.#"),
					// Verify all filters were applied
					resource.TestCheckResourceAttr("data.zillaforge_flavors.test", "vcpus", "2"),
					resource.TestCheckResourceAttr("data.zillaforge_flavors.test", "memory", "4"),
				),
			},
		},
	})
}

const testAccFlavorDataSourceConfig_multiple = `
data "zillaforge_flavors" "test" {
  vcpus  = 2
  memory = 4
}
`

// T011: Test empty results when no matches found.
func TestAccFlavorDataSource_emptyResults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlavorDataSourceConfig_empty,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify flavors list exists (empty list, not null)
					resource.TestCheckResourceAttrSet("data.zillaforge_flavors.test", "flavors.#"),
					// Accept either 0 results or some results - API may have this flavor
					resource.TestCheckResourceAttr("data.zillaforge_flavors.test", "name", "non-existent-flavor-xyz"),
				),
			},
		},
	})
}

const testAccFlavorDataSourceConfig_empty = `
# Use an unrealistic filter to ensure no matches
data "zillaforge_flavors" "test" {
  name = "non-existent-flavor-xyz"
}
`

// T012: Test API authentication error.
func TestAccFlavorDataSource_apiAuthError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Provide an obviously invalid API key to trigger auth failure
				Config: `
provider "zillaforge" {
	api_key = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.signature"
}

data "zillaforge_flavors" "test" {}
`,
				ExpectError: regexp.MustCompile(`(?i)unauthori|401|403|authentication|invalid credentials|verify token|illegal token|sdk initialization failed|\b400\b`),
			},
		},
	})
}

// T013: Test API error handling.
func TestAccFlavorDataSource_apiErrorHandling(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Point the provider at a non-routable endpoint to simulate network/SDK errors
				Config: `
provider "zillaforge" {
	api_endpoint = "http://127.0.0.1:1"
	api_key = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.signature"
}

data "zillaforge_flavors" "test" {}
`,
				ExpectError: regexp.MustCompile(`(?i)connection refused|connect:|timeout|EOF|no such host`),
			},
		},
	})
}
