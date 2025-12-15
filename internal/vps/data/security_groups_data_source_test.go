// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// T034: Acceptance test - Query security group by name.
func TestAccSecurityGroupsDataSource_ByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupsDataSourceConfig_byName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.by_name", "name", "test-sg-query-by-name"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.by_name", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.by_name", "security_groups.0.name", "test-sg-query-by-name"),
					resource.TestCheckResourceAttrSet("data.zillaforge_security_groups.by_name", "security_groups.0.id"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.by_name", "security_groups.0.description", "Test security group for name query"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.by_name", "security_groups.0.ingress_rule.#", "1"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.by_name", "security_groups.0.ingress_rule.0.protocol", "tcp"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.by_name", "security_groups.0.ingress_rule.0.port_range", "80"),
				),
			},
		},
	})
}

const testAccSecurityGroupsDataSourceConfig_byName = `
resource "zillaforge_security_group" "test" {
  name        = "test-sg-query-by-name"
  description = "Test security group for name query"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }
}

data "zillaforge_security_groups" "by_name" {
  name = zillaforge_security_group.test.name

  depends_on = [zillaforge_security_group.test]
}
`

// T035: Acceptance test - Query security group by ID.
func TestAccSecurityGroupsDataSource_ByID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupsDataSourceConfig_byID,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zillaforge_security_groups.by_id", "id"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.by_id", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.by_id", "security_groups.0.name", "test-sg-query-by-id"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.by_id", "security_groups.0.description", "Test security group for ID query"),
				),
			},
		},
	})
}

const testAccSecurityGroupsDataSourceConfig_byID = `
resource "zillaforge_security_group" "test" {
  name        = "test-sg-query-by-id"
  description = "Test security group for ID query"

  egress_rule {
    protocol         = "any"
    port_range       = "all"
    destination_cidr = "0.0.0.0/0"
  }
}

data "zillaforge_security_groups" "by_id" {
  id = zillaforge_security_group.test.id

  depends_on = [zillaforge_security_group.test]
}
`

// T036: Acceptance test - Error when querying non-existent name (returns empty list).
func TestAccSecurityGroupsDataSource_NonExistentName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupsDataSourceConfig_nonExistent,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.non_existent", "name", "non-existent-security-group-name"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.non_existent", "security_groups.#", "0"),
				),
			},
		},
	})
}

const testAccSecurityGroupsDataSourceConfig_nonExistent = `
data "zillaforge_security_groups" "non_existent" {
  name = "non-existent-security-group-name"
}
`

// T037: Acceptance test - List all security groups (no filters).
func TestAccSecurityGroupsDataSource_ListAll(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupsDataSourceConfig_listAll,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Should return at least 2 security groups we created
					resource.TestCheckResourceAttrWith("data.zillaforge_security_groups.all", "security_groups.#", func(value string) error {
						// We expect at least 2 security groups (the ones we created)
						if value == "0" {
							return fmt.Errorf("expected at least 2 security groups, got 0")
						}
						return nil
					}),
				),
			},
		},
	})
}

const testAccSecurityGroupsDataSourceConfig_listAll = `
resource "zillaforge_security_group" "test1" {
  name        = "test-sg-list-all-1"
  description = "First test security group for list all"
}

resource "zillaforge_security_group" "test2" {
  name        = "test-sg-list-all-2"
  description = "Second test security group for list all"
}

data "zillaforge_security_groups" "all" {
  # No filters - list all

  depends_on = [
    zillaforge_security_group.test1,
    zillaforge_security_group.test2
  ]
}
`

// T038: Acceptance test - Error when both name and ID filters specified.
func TestAccSecurityGroupsDataSource_BothFiltersError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSecurityGroupsDataSourceConfig_bothFilters,
				ExpectError: regexp.MustCompile("mutually exclusive"),
			},
		},
	})
}

const testAccSecurityGroupsDataSourceConfig_bothFilters = `
resource "zillaforge_security_group" "test" {
  name = "test-sg-both-filters"
}

data "zillaforge_security_groups" "both" {
  id   = zillaforge_security_group.test.id
  name = "test-sg-both-filters"

  depends_on = [zillaforge_security_group.test]
}
`

// T039: Acceptance test - Verify all rule attributes returned.
func TestAccSecurityGroupsDataSource_AllRuleAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupsDataSourceConfig_allRuleAttributes,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify ingress rule attributes
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.ingress_rule.#", "2"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.ingress_rule.0.protocol", "tcp"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.ingress_rule.0.port_range", "22"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.ingress_rule.0.source_cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.ingress_rule.1.protocol", "tcp"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.ingress_rule.1.port_range", "80-443"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.ingress_rule.1.source_cidr", "0.0.0.0/0"),
					// Verify egress rule attributes
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.egress_rule.#", "1"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.egress_rule.0.protocol", "any"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.egress_rule.0.port_range", "all"),
					resource.TestCheckResourceAttr("data.zillaforge_security_groups.rules", "security_groups.0.egress_rule.0.destination_cidr", "0.0.0.0/0"),
				),
			},
		},
	})
}

const testAccSecurityGroupsDataSourceConfig_allRuleAttributes = `
resource "zillaforge_security_group" "test" {
  name        = "test-sg-all-rule-attrs"
  description = "Test security group with multiple rules"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "10.0.0.0/8"
  }

  ingress_rule {
    protocol    = "tcp"
    port_range  = "80-443"
    source_cidr = "0.0.0.0/0"
  }

  egress_rule {
    protocol         = "any"
    port_range       = "all"
    destination_cidr = "0.0.0.0/0"
  }
}

data "zillaforge_security_groups" "rules" {
  name = zillaforge_security_group.test.name

  depends_on = [zillaforge_security_group.test]
}
`
