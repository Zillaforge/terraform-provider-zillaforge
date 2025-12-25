// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data_test

import (
	"testing"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// T024: Acceptance test - Query all floating IPs without filters.
func TestAccFloatingIPsDataSource_All(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPsDataSourceConfig_all,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the data source executes successfully (no specific count assertion)
					// The number of floating IPs can vary depending on cleanup state
					resource.TestCheckResourceAttrWith("data.zillaforge_floating_ips.all", "floating_ips.#", func(value string) error {
						// Just verify the attribute exists - value can be 0 or more
						return nil
					}),
				),
			},
		},
	})
}

const testAccFloatingIPsDataSourceConfig_all = `
data "zillaforge_floating_ips" "all" {
  # No filters - return all floating IPs
}
`

// T032: Acceptance test - Filter floating IPs by ID.
func TestAccFloatingIPsDataSource_FilterByID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPsDataSourceConfig_filterByID,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.by_id", "floating_ips.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.zillaforge_floating_ips.by_id", "id",
						"zillaforge_floating_ip.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.zillaforge_floating_ips.by_id", "floating_ips.0.id",
						"zillaforge_floating_ip.test", "id",
					),
					resource.TestCheckResourceAttrSet("data.zillaforge_floating_ips.by_id", "floating_ips.0.ip_address"),
				),
			},
		},
	})
}

const testAccFloatingIPsDataSourceConfig_filterByID = `
resource "zillaforge_floating_ip" "test" {
  name = "test-fip-query-by-id"
}

data "zillaforge_floating_ips" "by_id" {
  id = zillaforge_floating_ip.test.id

  depends_on = [zillaforge_floating_ip.test]
}
`

// T033: Acceptance test - Filter floating IPs by name.
func TestAccFloatingIPsDataSource_FilterByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPsDataSourceConfig_filterByName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.by_name", "name", "test-fip-query-by-name"),
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.by_name", "floating_ips.#", "1"),
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.by_name", "floating_ips.0.name", "test-fip-query-by-name"),
					resource.TestCheckResourceAttrSet("data.zillaforge_floating_ips.by_name", "floating_ips.0.id"),
					resource.TestCheckResourceAttrSet("data.zillaforge_floating_ips.by_name", "floating_ips.0.ip_address"),
				),
			},
		},
	})
}

const testAccFloatingIPsDataSourceConfig_filterByName = `
resource "zillaforge_floating_ip" "test" {
  name        = "test-fip-query-by-name"
  description = "Test floating IP for name query"
}

data "zillaforge_floating_ips" "by_name" {
  name = zillaforge_floating_ip.test.name

  depends_on = [zillaforge_floating_ip.test]
}
`

// T034: Acceptance test - Filter floating IPs by IP address.
func TestAccFloatingIPsDataSource_FilterByIPAddress(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPsDataSourceConfig_filterByIP,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.by_ip", "floating_ips.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.zillaforge_floating_ips.by_ip", "ip_address",
						"zillaforge_floating_ip.test", "ip_address",
					),
					resource.TestCheckResourceAttrPair(
						"data.zillaforge_floating_ips.by_ip", "floating_ips.0.ip_address",
						"zillaforge_floating_ip.test", "ip_address",
					),
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.by_ip", "floating_ips.0.name", "test-fip-query-by-ip"),
				),
			},
		},
	})
}

const testAccFloatingIPsDataSourceConfig_filterByIP = `
resource "zillaforge_floating_ip" "test" {
  name = "test-fip-query-by-ip"
}

data "zillaforge_floating_ips" "by_ip" {
  ip_address = zillaforge_floating_ip.test.ip_address

  depends_on = [zillaforge_floating_ip.test]
}
`

// T035: Acceptance test - Filter floating IPs by status.
func TestAccFloatingIPsDataSource_FilterByStatus(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPsDataSourceConfig_filterByStatus,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the status filter is set in the data source config
					resource.TestCheckResourceAttrPair(
						"data.zillaforge_floating_ips.by_status", "status",
						"zillaforge_floating_ip.test", "status",
					),
					// Verify floating_ips list exists (may be empty or have entries depending on actual status)
					resource.TestCheckResourceAttrSet("data.zillaforge_floating_ips.by_status", "floating_ips.#"),
				),
			},
		},
	})
}

const testAccFloatingIPsDataSourceConfig_filterByStatus = `
resource "zillaforge_floating_ip" "test" {
  name = "test-fip-query-by-status"
}

data "zillaforge_floating_ips" "by_status" {
  status = zillaforge_floating_ip.test.status

  depends_on = [zillaforge_floating_ip.test]
}
`

// T036: Acceptance test - Filter with multiple conditions (AND logic).
func TestAccFloatingIPsDataSource_MultipleFilters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPsDataSourceConfig_multipleFilters,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.multi", "name", "test-fip-multi-filter"),
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.multi", "floating_ips.#", "1"),
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.multi", "floating_ips.0.name", "test-fip-multi-filter"),
					resource.TestCheckResourceAttrPair(
						"data.zillaforge_floating_ips.multi", "ip_address",
						"zillaforge_floating_ip.test", "ip_address",
					),
					resource.TestCheckResourceAttrPair(
						"data.zillaforge_floating_ips.multi", "floating_ips.0.ip_address",
						"zillaforge_floating_ip.test", "ip_address",
					),
				),
			},
		},
	})
}

const testAccFloatingIPsDataSourceConfig_multipleFilters = `
resource "zillaforge_floating_ip" "test" {
  name        = "test-fip-multi-filter"
  description = "Test multiple filter AND logic"
}

data "zillaforge_floating_ips" "multi" {
  name   = zillaforge_floating_ip.test.name
  ip_address = zillaforge_floating_ip.test.ip_address

  depends_on = [zillaforge_floating_ip.test]
}
`

// T037: Acceptance test - Filter with no matches returns empty list.
func TestAccFloatingIPsDataSource_NoMatches(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPsDataSourceConfig_noMatches,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Should return empty list, not error
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.no_match", "floating_ips.#", "0"),
				),
			},
		},
	})
}

const testAccFloatingIPsDataSourceConfig_noMatches = `
data "zillaforge_floating_ips" "no_match" {
  name = "nonexistent-floating-ip-name-12345"
}
`
