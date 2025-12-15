// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource_test

import (
	"regexp"
	"testing"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// T010: Acceptance test - Create security group with ingress rules.
func TestAccSecurityGroup_CreateIngress(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_ingress,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.web", "name", "test-ingress-sg"),
					resource.TestCheckResourceAttrSet("zillaforge_security_group.web", "id"),
					resource.TestCheckResourceAttr("zillaforge_security_group.web", "ingress_rule.0.protocol", "tcp"),
					resource.TestCheckResourceAttr("zillaforge_security_group.web", "ingress_rule.0.port_range", "80"),
					resource.TestCheckResourceAttr("zillaforge_security_group.web", "ingress_rule.0.source_cidr", "0.0.0.0/0"),
				),
			},
		},
	})
}

const testAccSecurityGroupConfig_ingress = `
resource "zillaforge_security_group" "web" {
  name        = "test-ingress-sg"
  description = "Ingress test"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }
}
`

// T011: Acceptance test - Create security group with egress rules.
func TestAccSecurityGroup_CreateEgress(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_egress,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.egress", "name", "test-egress-sg"),
					resource.TestCheckResourceAttrSet("zillaforge_security_group.egress", "id"),
					resource.TestCheckResourceAttr("zillaforge_security_group.egress", "egress_rule.0.protocol", "any"),
					resource.TestCheckResourceAttr("zillaforge_security_group.egress", "egress_rule.0.port_range", "all"),
					resource.TestCheckResourceAttr("zillaforge_security_group.egress", "egress_rule.0.destination_cidr", "0.0.0.0/0"),
				),
			},
		},
	})
}

const testAccSecurityGroupConfig_egress = `
resource "zillaforge_security_group" "egress" {
  name        = "test-egress-sg"
  description = "Egress test"

  egress_rule {
    protocol         = "any"
    port_range       = "all"
    destination_cidr = "0.0.0.0/0"
  }
}
`

// T012: Acceptance test - Update security group description (in-place).
func TestAccSecurityGroup_UpdateDescription(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_updateDescInitial,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.update", "name", "test-update-sg"),
					resource.TestCheckResourceAttrSet("zillaforge_security_group.update", "id"),
					resource.TestCheckResourceAttr("zillaforge_security_group.update", "description", "Initial description"),
				),
			},
			{
				Config: testAccSecurityGroupConfig_updateDescUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.update", "name", "test-update-sg"),
					resource.TestCheckResourceAttr("zillaforge_security_group.update", "description", "Updated description"),
				),
			},
		},
	})
}

const testAccSecurityGroupConfig_updateDescInitial = `
resource "zillaforge_security_group" "update" {
  name        = "test-update-sg"
  description = "Initial description"
}
`

const testAccSecurityGroupConfig_updateDescUpdated = `
resource "zillaforge_security_group" "update" {
  name        = "test-update-sg"
  description = "Updated description"
}
`

// T013: Acceptance test - Add rules to existing security group.
func TestAccSecurityGroup_AddRules(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_addRulesInitial,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.add", "name", "test-add-rules"),
					resource.TestCheckResourceAttrSet("zillaforge_security_group.add", "id"),
					resource.TestCheckResourceAttr("zillaforge_security_group.add", "ingress_rule.0.port_range", "22"),
				),
			},
			{
				Config: testAccSecurityGroupConfig_addRulesUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.add", "ingress_rule.0.port_range", "22"),
					resource.TestCheckResourceAttr("zillaforge_security_group.add", "ingress_rule.1.port_range", "80"),
				),
			},
		},
	})
}

const testAccSecurityGroupConfig_addRulesInitial = `
resource "zillaforge_security_group" "add" {
  name = "test-add-rules"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "203.0.113.0/24"
  }
}
`

const testAccSecurityGroupConfig_addRulesUpdated = `
resource "zillaforge_security_group" "add" {
  name = "test-add-rules"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "203.0.113.0/24"
  }

  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }
}
`

// T014: Acceptance test - Remove rules from security group.
func TestAccSecurityGroup_RemoveRules(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_removeRulesInitial,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.remove", "name", "test-remove-rules"),
					resource.TestCheckResourceAttrSet("zillaforge_security_group.remove", "id"),
					resource.TestCheckResourceAttr("zillaforge_security_group.remove", "ingress_rule.0.port_range", "22"),
				),
			},
			{
				Config: testAccSecurityGroupConfig_removeRulesUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("zillaforge_security_group.remove", "ingress_rule.1"),
				),
			},
		},
	})
}

const testAccSecurityGroupConfig_removeRulesInitial = `
resource "zillaforge_security_group" "remove" {
  name = "test-remove-rules"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "203.0.113.0/24"
  }

  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }
}
`

const testAccSecurityGroupConfig_removeRulesUpdated = `
resource "zillaforge_security_group" "remove" {
  name = "test-remove-rules"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "203.0.113.0/24"
  }
}
`

// T015: Acceptance test - Modify rule attributes (replace rule).
func TestAccSecurityGroup_ModifyRule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_modifyInitial,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.modify", "ingress_rule.0.port_range", "22"),
				),
			},
			{
				Config: testAccSecurityGroupConfig_modifyUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.modify", "ingress_rule.0.port_range", "2200-2300"),
				),
			},
		},
	})
}

const testAccSecurityGroupConfig_modifyInitial = `
resource "zillaforge_security_group" "modify" {
  name = "test-modify-rule"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "203.0.113.0/24"
  }
}
`

const testAccSecurityGroupConfig_modifyUpdated = `
resource "zillaforge_security_group" "modify" {
  name = "test-modify-rule"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "2200-2300"
    source_cidr = "203.0.113.0/24"
  }
}
`

// T016: Acceptance test - Delete security group (not attached).
func TestAccSecurityGroup_Delete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_delete,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.delete", "name", "test-delete-sg"),
				),
			},
		},
	})
}

const testAccSecurityGroupConfig_delete = `
resource "zillaforge_security_group" "delete" {
  name = "test-delete-sg"
}
`

// T017: Acceptance test - Block deletion when attached to instances (handle 409).
// NOTE: This test requires attaching the security group to a server instance which
// is outside the scope of this feature; left as a TODO to implement when server
// provisioning helper is available in test environment.
func TestAccSecurityGroup_DeleteBlockedWhenAttached(t *testing.T) {
	t.Skip("TODO: Implement after server provisioning helper is available in test environment")
}

// T018: Acceptance test - ForceNew on name change.
func TestAccSecurityGroup_RequiresReplaceOnNameChange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_renameOriginal,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.rename", "name", "test-rename-sg"),
					resource.TestCheckResourceAttrSet("zillaforge_security_group.rename", "id"),
				),
			},
			{
				Config: testAccSecurityGroupConfig_renameNew,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.rename", "name", "test-renamed-sg"),
					resource.TestCheckResourceAttrSet("zillaforge_security_group.rename", "id"),
				),
			},
		},
	})
}

const testAccSecurityGroupConfig_renameOriginal = `
resource "zillaforge_security_group" "rename" {
  name = "test-rename-sg"
}
`

const testAccSecurityGroupConfig_renameNew = `
resource "zillaforge_security_group" "rename" {
  name = "test-renamed-sg"
}
`

// T055: Acceptance test - Import security group by ID.
func TestAccSecurityGroup_ImportByID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_importByID,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "name", "test-import-sg"),
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "description", "Security group for import testing"),
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "ingress_rule.#", "1"),
				),
			},
			{
				ResourceName:      "zillaforge_security_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccSecurityGroupConfig_importByID = `
resource "zillaforge_security_group" "test" {
  name        = "test-import-sg"
  description = "Security group for import testing"

  ingress_rule {
	protocol    = "tcp"
	port_range  = "22"
	source_cidr = "10.0.0.0/8"
  }
}
`

// T056: Acceptance test - Plan after import shows no changes (matching config).
func TestAccSecurityGroup_ImportNoChanges(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_importNoChanges,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "name", "test-import-nochanges"),
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "ingress_rule.#", "2"),
				),
			},
			{
				ResourceName:            "zillaforge_security_group.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ingress_rule", "egress_rule"}, // API may return rules in different order
			},
			// Re-apply same config - should show no changes
			{
				Config: testAccSecurityGroupConfig_importNoChanges,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "name", "test-import-nochanges"),
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "ingress_rule.#", "2"),
				),
			},
		},
	})
}

const testAccSecurityGroupConfig_importNoChanges = `
resource "zillaforge_security_group" "test" {
  name        = "test-import-nochanges"
  description = "Testing import state consistency"

  ingress_rule {
	protocol    = "tcp"
	port_range  = "80"
	source_cidr = "0.0.0.0/0"
  }

  ingress_rule {
	protocol    = "tcp"
	port_range  = "443"
	source_cidr = "0.0.0.0/0"
  }

  egress_rule {
	protocol         = "any"
	port_range       = "all"
	destination_cidr = "0.0.0.0/0"
  }
}
`

// T057: Acceptance test - Plan after import detects drift (config mismatch).
func TestAccSecurityGroup_ImportDetectsDrift(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_importDriftOriginal,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "name", "test-import-drift"),
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "ingress_rule.#", "1"),
				),
			},
			{
				ResourceName:      "zillaforge_security_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Apply different config - should detect drift and update
			{
				Config: testAccSecurityGroupConfig_importDriftModified,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "name", "test-import-drift"),
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "ingress_rule.#", "2"),
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "description", "Modified after import"),
				),
			},
		},
	})
}

const testAccSecurityGroupConfig_importDriftOriginal = `
resource "zillaforge_security_group" "test" {
  name        = "test-import-drift"
  description = "Original config"

  ingress_rule {
	protocol    = "tcp"
	port_range  = "22"
	source_cidr = "10.0.0.0/8"
  }
}
`

const testAccSecurityGroupConfig_importDriftModified = `
resource "zillaforge_security_group" "test" {
  name        = "test-import-drift"
  description = "Modified after import"

  ingress_rule {
	protocol    = "tcp"
	port_range  = "22"
	source_cidr = "10.0.0.0/8"
  }

  ingress_rule {
	protocol    = "tcp"
	port_range  = "80"
	source_cidr = "0.0.0.0/0"
  }
}
`

// T058: Acceptance test - Error on invalid import ID format.
func TestAccSecurityGroup_ImportInvalidID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_importInvalidID,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_security_group.test", "name", "test-import-invalid"),
				),
			},
			{
				ResourceName:  "zillaforge_security_group.test",
				ImportState:   true,
				ImportStateId: "invalid-id-format", // Not a valid UUID
				ExpectError:   regexp.MustCompile(`Invalid Import ID Format|not a valid UUID`),
			},
		},
	})
}

const testAccSecurityGroupConfig_importInvalidID = `
resource "zillaforge_security_group" "test" {
  name = "test-import-invalid"
}
`
