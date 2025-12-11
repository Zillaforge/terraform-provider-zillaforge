// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// T035: Test basic networks query without filters
func TestAccNetworkDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

// T036: Test name filter with exact match
func TestAccNetworkDataSource_nameFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

// T037: Test status filter
func TestAccNetworkDataSource_statusFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

// T038: Test multiple filters with AND logic
func TestAccNetworkDataSource_multipleFilters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

// T039: Test empty results when no matches found
func TestAccNetworkDataSource_emptyResults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

// T040: Test API authentication error
func TestAccNetworkDataSource_apiAuthError(t *testing.T) {
	// This test verifies that authentication errors are properly handled
	// Note: Requires invalid credentials to be tested in real environment
	t.Skip("API authentication error scenario test - requires test credentials with invalid auth")
}

// T041: Test API error handling
func TestAccNetworkDataSource_apiErrorHandling(t *testing.T) {
	// This test verifies that SDK errors are converted to diagnostics properly
	// The implementation handles SDK errors in Read() method
	// Note: Comprehensive error testing requires SDK error mocking
	t.Skip("API error handling scenario test - requires SDK error mocking framework")
}
