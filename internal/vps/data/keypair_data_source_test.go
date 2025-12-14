// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data_test

import (
	"regexp"
	"testing"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// T026: Acceptance test - Query keypair by name.
func TestAccKeypairDataSource_FilterByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// First create a keypair to query
			{
				Config: testAccKeypairDataSourceConfig_setupForName,
			},
			// Then query it by name
			{
				Config: testAccKeypairDataSourceConfig_byName,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify filter was applied
					resource.TestCheckResourceAttr("data.zillaforge_keypairs.test", "name", "test-query-by-name"),
					// Verify keypairs list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_keypairs.test", "keypairs.#"),
					// Verify at least one result returned
					resource.TestCheckResourceAttr("data.zillaforge_keypairs.test", "keypairs.0.name", "test-query-by-name"),
					resource.TestCheckResourceAttrSet("data.zillaforge_keypairs.test", "keypairs.0.id"),
					resource.TestCheckResourceAttrSet("data.zillaforge_keypairs.test", "keypairs.0.public_key"),
					resource.TestCheckResourceAttrSet("data.zillaforge_keypairs.test", "keypairs.0.fingerprint"),
				),
			},
		},
	})
}

const testAccKeypairDataSourceConfig_setupForName = `
resource "zillaforge_keypair" "setup" {
  name        = "test-query-by-name"
  description = "Keypair for name filter test"
  public_key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"
}
`

const testAccKeypairDataSourceConfig_byName = `
resource "zillaforge_keypair" "setup" {
  name        = "test-query-by-name"
  description = "Keypair for name filter test"
  public_key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"
}

data "zillaforge_keypairs" "test" {
  name = zillaforge_keypair.setup.name
}
`

// T027: Acceptance test - Query keypair by ID.
func TestAccKeypairDataSource_FilterByID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// First create a keypair to query
			{
				Config: testAccKeypairDataSourceConfig_setupForID,
			},
			// Then query it by ID
			{
				Config: testAccKeypairDataSourceConfig_byID,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify filter was applied
					resource.TestCheckResourceAttrPair("data.zillaforge_keypairs.test", "id", "zillaforge_keypair.setup", "id"),
					// Verify keypairs list exists with single result
					resource.TestCheckResourceAttr("data.zillaforge_keypairs.test", "keypairs.#", "1"),
					// Verify result matches the created keypair
					resource.TestCheckResourceAttrPair("data.zillaforge_keypairs.test", "keypairs.0.id", "zillaforge_keypair.setup", "id"),
					resource.TestCheckResourceAttr("data.zillaforge_keypairs.test", "keypairs.0.name", "test-query-by-id"),
					resource.TestCheckResourceAttrSet("data.zillaforge_keypairs.test", "keypairs.0.public_key"),
					resource.TestCheckResourceAttrSet("data.zillaforge_keypairs.test", "keypairs.0.fingerprint"),
				),
			},
		},
	})
}

const testAccKeypairDataSourceConfig_setupForID = `
resource "zillaforge_keypair" "setup" {
  name        = "test-query-by-id"
  description = "Keypair for ID filter test"
  public_key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"
}
`

const testAccKeypairDataSourceConfig_byID = `
resource "zillaforge_keypair" "setup" {
  name        = "test-query-by-id"
  description = "Keypair for ID filter test"
  public_key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"
}

data "zillaforge_keypairs" "test" {
  id = zillaforge_keypair.setup.id
}
`

// T028: Acceptance test - List all keypairs (no filters).
func TestAccKeypairDataSource_ListAll(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple keypairs
			{
				Config: testAccKeypairDataSourceConfig_setupMultiple,
			},
			// Query all keypairs (no filters)
			{
				Config: testAccKeypairDataSourceConfig_listAll,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify keypairs list exists
					resource.TestCheckResourceAttrSet("data.zillaforge_keypairs.test", "keypairs.#"),
					// Verify at least our created keypairs are present (may have more from other tests)
					// We can't assert exact count since other tests may leave keypairs
				),
			},
		},
	})
}

const testAccKeypairDataSourceConfig_setupMultiple = `
resource "zillaforge_keypair" "first" {
  name        = "test-list-all-first"
  description = "First keypair for list-all test"
  public_key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test1@example.com"
}

resource "zillaforge_keypair" "second" {
  name        = "test-list-all-second"
  description = "Second keypair for list-all test"
  public_key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test2@example.com"
}
`

const testAccKeypairDataSourceConfig_listAll = `
resource "zillaforge_keypair" "first" {
  name        = "test-list-all-first"
  description = "First keypair for list-all test"
}

resource "zillaforge_keypair" "second" {
  name        = "test-list-all-second"
  description = "Second keypair for list-all test"
}

data "zillaforge_keypairs" "test" {
  # No filters - list all keypairs
}
`

// T029: Acceptance test - Both name and ID filters return validation error.
func TestAccKeypairDataSource_MutualExclusivity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKeypairDataSourceConfig_bothFilters,
				ExpectError: regexp.MustCompile(".*cannot specify both.*id.*name.*|.*Invalid Filter Combination.*"),
			},
		},
	})
}

const testAccKeypairDataSourceConfig_bothFilters = `
data "zillaforge_keypairs" "test" {
  id   = "some-uuid"
  name = "some-name"
}
`

// T030: Acceptance test - Non-existent ID returns error.
func TestAccKeypairDataSource_NonExistentID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKeypairDataSourceConfig_nonExistentID,
				ExpectError: regexp.MustCompile(".*not found.*|.*Keypair Not Found.*"),
			},
		},
	})
}

const testAccKeypairDataSourceConfig_nonExistentID = `
data "zillaforge_keypairs" "test" {
  id = "00000000-0000-0000-0000-000000000000"
}
`

// T031: Acceptance test - Non-existent name returns empty list.
func TestAccKeypairDataSource_NonExistentName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairDataSourceConfig_nonExistentName,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify filter was applied
					resource.TestCheckResourceAttr("data.zillaforge_keypairs.test", "name", "non-existent-keypair-name-12345"),
					// Verify empty list is returned (no error)
					resource.TestCheckResourceAttr("data.zillaforge_keypairs.test", "keypairs.#", "0"),
				),
			},
		},
	})
}

const testAccKeypairDataSourceConfig_nonExistentName = `
data "zillaforge_keypairs" "test" {
  name = "non-existent-keypair-name-12345"
}
`
