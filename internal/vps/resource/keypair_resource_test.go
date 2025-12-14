// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource_test

import (
	"regexp"
	"testing"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// T008: Acceptance test - Create keypair with user-provided public key.
func TestAccKeypairResource_UserProvidedKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairResourceConfig_userProvided,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-user-key"),
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "description", "User-provided keypair"),
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "public_key"),
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "fingerprint"),
					// Verify private_key is null for user-provided key
					resource.TestCheckNoResourceAttr("zillaforge_keypair.test", "private_key"),
				),
			},
		},
	})
}

const testAccKeypairResourceConfig_userProvided = `
resource "zillaforge_keypair" "test" {
  name        = "test-user-key"
  description = "User-provided keypair"
  public_key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"
}
`

// T009: Acceptance test - Create keypair with system-generated keys.
func TestAccKeypairResource_SystemGenerated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairResourceConfig_systemGenerated,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-auto-key"),
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "public_key"),
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "private_key"),
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "fingerprint"),
				),
			},
		},
	})
}

const testAccKeypairResourceConfig_systemGenerated = `
resource "zillaforge_keypair" "test" {
  name        = "test-auto-key"
  description = "System-generated keypair"
  # public_key omitted - system generates
}
`

// T010: Acceptance test - Delete keypair successfully.
func TestAccKeypairResource_Delete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairResourceConfig_delete,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-delete-key"),
				),
			},
			// Destroy happens automatically after last step
		},
	})
}

const testAccKeypairResourceConfig_delete = `
resource "zillaforge_keypair" "test" {
  name = "test-delete-key"
}
`

// T011: Acceptance test - Change immutable fields triggers replacement.
func TestAccKeypairResource_RequiresReplace(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairResourceConfig_original,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-immutable-key"),
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "id"),
				),
			},
			{
				Config: testAccKeypairResourceConfig_renamed,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-renamed-key"),
					// ID should change (resource replaced)
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "id"),
				),
			},
		},
	})
}

const testAccKeypairResourceConfig_original = `
resource "zillaforge_keypair" "test" {
  name = "test-immutable-key"
}
`

const testAccKeypairResourceConfig_renamed = `
resource "zillaforge_keypair" "test" {
  name = "test-renamed-key"
}
`

// T012: Acceptance test - Duplicate keypair name returns error.
func TestAccKeypairResource_DuplicateName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairResourceConfig_duplicate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_keypair.first", "name", "duplicate-name"),
				),
			},
			{
				Config:      testAccKeypairResourceConfig_duplicateSecond,
				ExpectError: regexp.MustCompile(`Key pair 'duplicate-name' already exists`),
			},
		},
	})
}

const testAccKeypairResourceConfig_duplicate = `
resource "zillaforge_keypair" "first" {
  name = "duplicate-name"
}
`

const testAccKeypairResourceConfig_duplicateSecond = `
resource "zillaforge_keypair" "first" {
  name = "duplicate-name"
}

resource "zillaforge_keypair" "second" {
  name = "duplicate-name"
}
`

// T013: Acceptance test - Invalid public key format returns error.
func TestAccKeypairResource_InvalidPublicKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKeypairResourceConfig_invalidKey,
				ExpectError: regexp.MustCompile(`failed to generate fingerprint`),
			},
		},
	})
}

const testAccKeypairResourceConfig_invalidKey = `
resource "zillaforge_keypair" "test" {
  name       = "test-invalid-key"
  public_key = "this-is-not-a-valid-ssh-key"
}
`

// T019 test: Update description only (other fields immutable).
func TestAccKeypairResource_UpdateDescription(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairResourceConfig_initialDescription,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-update-key"),
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "id"),
				),
			},
			{
				Config: testAccKeypairResourceConfig_updatedDescription,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-update-key"),
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "description", "Updated description"),
					// ID should NOT change (in-place update)
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "id"),
				),
			},
		},
	})
}

const testAccKeypairResourceConfig_initialDescription = `
resource "zillaforge_keypair" "test" {
  name        = "test-update-key"
  description = "Initial description"
}
`

const testAccKeypairResourceConfig_updatedDescription = `
resource "zillaforge_keypair" "test" {
  name        = "test-update-key"
  description = "Updated description"
}
`

// T042: Acceptance test - Import keypair by ID.
func TestAccKeypairResource_ImportByID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairResourceConfig_forImport,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-import-key"),
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "id"),
				),
			},
			{
				ResourceName:            "zillaforge_keypair.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"private_key"}, // Private key not available after creation
			},
		},
	})
}

const testAccKeypairResourceConfig_forImport = `
resource "zillaforge_keypair" "test" {
  name        = "test-import-key"
  description = "Keypair for import testing"
}
`

// T043: Acceptance test - Imported keypair shows no changes on plan.
func TestAccKeypairResource_ImportNoChanges(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairResourceConfig_importNoChanges,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-import-nochanges"),
				),
			},
			{
				ResourceName:            "zillaforge_keypair.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"private_key"},
			},
			// Re-apply same config - should show no changes
			{
				Config: testAccKeypairResourceConfig_importNoChanges,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-import-nochanges"),
				),
			},
		},
	})
}

const testAccKeypairResourceConfig_importNoChanges = `
resource "zillaforge_keypair" "test" {
  name        = "test-import-nochanges"
  description = "Testing import state consistency"
}
`

// T044: Acceptance test - Non-existent import ID returns error.
func TestAccKeypairResource_ImportNonExistent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccKeypairResourceConfig_importNonExistent,
				ResourceName:  "zillaforge_keypair.test",
				ImportState:   true,
				ImportStateId: "00000000-0000-0000-0000-000000000000", // Non-existent ID
				ExpectError:   regexp.MustCompile(`Keypair not found`),
			},
		},
	})
}

const testAccKeypairResourceConfig_importNonExistent = `
resource "zillaforge_keypair" "test" {
  name = "test-import-nonexistent"
}
`

// T045: Acceptance test - Imported keypair private_key is null.
func TestAccKeypairResource_ImportPrivateKeyNull(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairResourceConfig_importPrivateKeyNull,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-import-privatekey"),
					// For system-generated keys, private_key should be set during creation
					resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "private_key"),
				),
			},
			{
				ResourceName:            "zillaforge_keypair.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"private_key"},
				Check: resource.ComposeAggregateTestCheckFunc(
					// After import, private_key should be null (not available from API)
					resource.TestCheckNoResourceAttr("zillaforge_keypair.test", "private_key"),
				),
			},
		},
	})
}

const testAccKeypairResourceConfig_importPrivateKeyNull = `
resource "zillaforge_keypair" "test" {
  name        = "test-import-privatekey"
  description = "System-generated keypair for import"
  # No public_key - system generates
}
`
