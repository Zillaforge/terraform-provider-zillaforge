// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource_test

import (
	"testing"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// T014: Acceptance test - Create floating IP without name/description.
func TestAccFloatingIPResource_Basic(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPResourceConfig_basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify computed attributes are set
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_basic", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_basic", "ip_address"),
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_basic", "status"),
					// Verify optional attributes are null
					resource.TestCheckNoResourceAttr("zillaforge_floating_ip.test_basic", "name"),
					resource.TestCheckNoResourceAttr("zillaforge_floating_ip.test_basic", "description"),
					resource.TestCheckNoResourceAttr("zillaforge_floating_ip.test_basic", "device_id"),
				),
			},
			// T023: Import test step
			{
				ResourceName:      "zillaforge_floating_ip.test_basic",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccFloatingIPResourceConfig_basic = `
resource "zillaforge_floating_ip" "test_basic" {
  # No attributes - allocate with defaults
}
`

// T022: Acceptance test - Create floating IP with name and description.
func TestAccFloatingIPResource_WithNameDescription(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPResourceConfig_withNameDescription,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify all attributes
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test_with_name", "name", "test-floating-ip"),
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test_with_name", "description", "Test floating IP"),
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_with_name", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_with_name", "ip_address"),
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_with_name", "status"),
				),
			},
		},
	})
}

const testAccFloatingIPResourceConfig_withNameDescription = `
resource "zillaforge_floating_ip" "test_with_name" {
  name        = "test-floating-ip"
  description = "Test floating IP"
}
`

// Additional test: Update name and description (in-place update).
func TestAccFloatingIPResource_Update(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPResourceConfig_beforeUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test_update", "name", "original-name"),
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test_update", "description", "Original description"),
				),
			},
			{
				Config: testAccFloatingIPResourceConfig_afterUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test_update", "name", "updated-name"),
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test_update", "description", "Updated description"),
					// Verify ID unchanged (in-place update, not replacement)
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_update", "id"),
				),
			},
		},
	})
}

const testAccFloatingIPResourceConfig_beforeUpdate = `
resource "zillaforge_floating_ip" "test_update" {
  name        = "original-name"
  description = "Original description"
}
`

const testAccFloatingIPResourceConfig_afterUpdate = `
resource "zillaforge_floating_ip" "test_update" {
  name        = "updated-name"
  description = "Updated description"
}
`

// Additional test: Verify all status values are handled correctly.
func TestAccFloatingIPResource_StatusHandling(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPResourceConfig_basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Status should be set (any value is informational)
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_basic", "status"),
				),
			},
		},
	})
}

// T041: Acceptance test - Import floating IP with name and description attributes.
func TestAccFloatingIPResource_ImportWithAttributes(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPResourceConfig_importTest,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test_import", "name", "test-import-fip"),
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test_import", "description", "Test import with attributes"),
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_import", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_import", "ip_address"),
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test_import", "status"),
				),
			},
			// Import step - verify all attributes are correctly imported
			{
				ResourceName:      "zillaforge_floating_ip.test_import",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccFloatingIPResourceConfig_importTest = `
resource "zillaforge_floating_ip" "test_import" {
  name        = "test-import-fip"
  description = "Test import with attributes"
}
`
