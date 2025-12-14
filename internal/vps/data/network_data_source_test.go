// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data_test

import (
	"regexp"
	"testing"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// T035: Test basic networks query without filters.
func TestAccNetworkDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkDataSourceConfig_all,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify networks list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_networks.test", "networks.#"),
				),
			},
		},
	})
}

const testAccNetworkDataSourceConfig_all = `
data "zillaforge_networks" "test" {}
`

// T036: Test name filter with exact match.
func TestAccNetworkDataSource_nameFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkDataSourceConfig_name,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify networks list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_networks.test", "networks.#"),
					// Verify filter was applied
					resource.TestCheckResourceAttr("data.zillaforge_networks.test", "name", "private-network"),
				),
			},
		},
	})
}

const testAccNetworkDataSourceConfig_name = `
data "zillaforge_networks" "test" {
  name = "private-network"
}
`

// T037: Test status filter.
func TestAccNetworkDataSource_statusFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkDataSourceConfig_status,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify networks list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_networks.test", "networks.#"),
					// Verify filter was applied
					resource.TestCheckResourceAttr("data.zillaforge_networks.test", "status", "ACTIVE"),
				),
			},
		},
	})
}

const testAccNetworkDataSourceConfig_status = `
data "zillaforge_networks" "test" {
  status = "ACTIVE"
}
`

// T038: Test multiple filters with AND logic.
func TestAccNetworkDataSource_multipleFilters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkDataSourceConfig_multiple,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify networks list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_networks.test", "networks.#"),
					// Verify all filters were applied
					resource.TestCheckResourceAttr("data.zillaforge_networks.test", "name", "dmz"),
					resource.TestCheckResourceAttr("data.zillaforge_networks.test", "status", "ACTIVE"),
				),
			},
		},
	})
}

const testAccNetworkDataSourceConfig_multiple = `
data "zillaforge_networks" "test" {
  name   = "dmz"
  status = "ACTIVE"
}
`

// T039: Test empty results when no matches found.
func TestAccNetworkDataSource_emptyResults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkDataSourceConfig_empty,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify networks list exists (empty list, not null)
					resource.TestCheckResourceAttrSet("data.zillaforge_networks.test", "networks.#"),
					// Verify filter was applied
					resource.TestCheckResourceAttr("data.zillaforge_networks.test", "name", "non-existent-network-xyz"),
				),
			},
		},
	})
}

const testAccNetworkDataSourceConfig_empty = `
# Use an unrealistic filter to ensure no matches
data "zillaforge_networks" "test" {
  name = "non-existent-network-xyz"
}
`

// T040: Test API authentication error.
func TestAccNetworkDataSource_apiAuthError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "zillaforge" {
	api_key = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.signature"
}

data "zillaforge_networks" "test" {}
`,
				ExpectError: regexp.MustCompile(`(?i)unauthori|401|403|authentication|invalid credentials|verify token|illegal token|sdk initialization failed|\b400\b`),
			},
		},
	})
}

// T041: Test API error handling.
func TestAccNetworkDataSource_apiErrorHandling(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "zillaforge" {
	api_endpoint = "http://127.0.0.1:1"
	api_key = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.signature"
}

data "zillaforge_networks" "test" {}
`,
				ExpectError: regexp.MustCompile(`(?i)connection refused|connect:|timeout|EOF|no such host`),
			},
		},
	})
}
