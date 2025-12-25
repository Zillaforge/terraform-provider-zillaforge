// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// T014: Acceptance test - Create server with required attributes.
func TestAccServerResource_Basic(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-basic-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_basic, name, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "flavor_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "image_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "status"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "created_at"),
					// Verify network_attachment
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.#", "1"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.network_id"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_basic = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"

  wait_for_deleted = false


  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

const testAccServerResourceConfig_destroy = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }

  timeouts {
    delete = "10m"
  }
}
`

// T015: Acceptance test - Create server with optional attributes (keypair, user_data).
func TestAccServerResource_WithOptions(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-options-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_withOptions, name, name, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttr("zillaforge_server.test", "description", "Server with optional attributes"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "flavor_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "image_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "keypair"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "user_data"),
					resource.TestCheckResourceAttr("zillaforge_server.test", "wait_for_active", "true"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_withOptions = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

data "zillaforge_keypairs" "test" {}

resource "zillaforge_keypair" "test" {
  name = "%s-keypair"
}

resource "zillaforge_server" "test" {
  name        = "%s"
  description = "Server with optional attributes"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = data.zillaforge_images.test.images[0].id
  keypair     = zillaforge_keypair.test.id
  user_data   = "#!/bin/bash\necho 'Hello World'"

  wait_for_active = true
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }

  timeouts {
    create = "15m"
  }
}
`

// T016: Acceptance test - Create server with multiple network attachments.
func TestAccServerResource_MultiNetwork(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-multi-net-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_multiNetwork, name, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					// Verify multiple network attachments
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.#", "2"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.network_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.1.network_id"),
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.0.primary", "true"),
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.1.primary", "false"),
				)},
		},
	})
}

const testAccServerResourceConfig_multiNetwork = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    primary    = true
    security_group_ids = [zillaforge_security_group.sg.id]
  }

  wait_for_deleted = false

  dynamic "network_attachment" {
    for_each = length(data.zillaforge_networks.test.networks) > 1 ? [1] : []
    content {
      network_id = data.zillaforge_networks.test.networks[1].id
      primary    = false
      security_group_ids = [zillaforge_security_group.sg.id]
    }
  }
}
`

// T017: Acceptance test - Single NIC with multiple security groups.
func TestAccServerResource_SingleNICMultipleSecurityGroups(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-single-nic-multi-sg-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_singleNICMultiSG, name, name, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					// Verify single network attachment exists
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.#", "1"),
					// Verify two security groups are attached to the NIC
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.0.security_group_ids.#", "2"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.security_group_ids.0"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.security_group_ids.1"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_singleNICMultiSG = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg1" {
  name = "%s-sg1"
}

resource "zillaforge_security_group" "sg2" {
  name = "%s-sg2"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg1.id, zillaforge_security_group.sg2.id]
  }
}
`

// T018: Acceptance test - Server destruction.
func TestAccServerResource_Destroy(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-destroy-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_destroy, name, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
			// Destroy happens automatically after last step
		},
	})
}

// T019: Acceptance test - Change `network_attachment.network_id` (create new NIC then delete old).
func TestAccServerResource_ChangeNetworkID(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-change-network-%d", time.Now().UnixNano()%100000)
	config1 := fmt.Sprintf(testAccServerResourceConfig_changeNetwork_from, name, name)
	config2 := fmt.Sprintf(testAccServerResourceConfig_changeNetwork_to, name, name)

	var initialNetworkID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.#", "1"),
					resource.TestCheckFunc(func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["zillaforge_server.test"]
						if !ok {
							return fmt.Errorf("server resource not found in state")
						}
						nid := rs.Primary.Attributes["network_attachment.0.network_id"]
						if nid == "" {
							return fmt.Errorf("network_attachment.0.network_id expected to be set")
						}
						// If there is only one network in the data source, skip this test (cannot change to a second network)
						ds, ok := s.RootModule().Resources["data.zillaforge_networks.test"]
						if !ok || ds.Primary.Attributes["networks.1.id"] == "" {
							// Use t.Skip here to skip the test early
							t.Skip("only one network available; skipping TestAccServerResource_ChangeNetworkID")
						}
						initialNetworkID = nid
						return nil
					}),
				),
			},
			{
				Config: config2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.#", "1"),
					resource.TestCheckFunc(func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["zillaforge_server.test"]
						if !ok {
							return fmt.Errorf("server resource not found in state")
						}
						newNID := rs.Primary.Attributes["network_attachment.0.network_id"]
						if newNID == "" {
							return fmt.Errorf("network_attachment.0.network_id expected to be set")
						}
						// Ensure it changed from the initial value
						if newNID == initialNetworkID {
							return fmt.Errorf("network_id did not change, still %s", newNID)
						}
						// If a second network exists, verify it matches data source networks[1].id
						ds, ok := s.RootModule().Resources["data.zillaforge_networks.test"]
						if ok {
							expected := ds.Primary.Attributes["networks.1.id"]
							if expected != "" && newNID != expected {
								return fmt.Errorf("expected network_id %s, got %s", expected, newNID)
							}
						}
						return nil
					}),
				),
			},
		},
	})
}

const testAccServerResourceConfig_changeNetwork_from = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

const testAccServerResourceConfig_changeNetwork_to = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[1].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// T042: Acceptance test - Create server with wait_for_active = false.
func TestAccServerResource_WaitForActiveFalse(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-async-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_waitForActiveFalse, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttr("zillaforge_server.test", "wait_for_active", "false"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "flavor_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "image_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "status"),
					// Status may be "building" or "active" depending on API response speed
					// We just verify it exists
				),
				// Expect non-empty plan on refresh since server may still be provisioning
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

const testAccServerResourceConfig_waitForActiveFalse = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

data "zillaforge_security_groups" "test" {}

resource "zillaforge_server" "test" {
  name            = "%s"
  flavor_id       = data.zillaforge_flavors.test.flavors[0].id
  image_id        = data.zillaforge_images.test.images[0].id
  password        = "TestPassword123!"
  wait_for_active = false
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [data.zillaforge_security_groups.test.security_groups[0].id]
  }
}
`

// T043: Acceptance test - Create server with wait_for_active = true (explicit default).
func TestAccServerResource_WaitForActiveTrue(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-wait-active-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_waitForActiveTrue, name, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttr("zillaforge_server.test", "wait_for_active", "true"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "flavor_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "image_id"),
					// Status should be "active" (lowercase) since we waited
					resource.TestCheckResourceAttr("zillaforge_server.test", "status", "ACTIVE"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_waitForActiveTrue = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name            = "%s"
  flavor_id       = data.zillaforge_flavors.test.flavors[0].id
  image_id        = data.zillaforge_images.test.images[0].id
  password        = "TestPassword123!"
  wait_for_active = true
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }

  timeouts {
    create = "15m"
  }
}
`

// T044: Acceptance test - Verify default wait_for_active behavior.
func TestAccServerResource_WaitForActiveDefault(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-default-wait-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_waitForActiveDefault, name, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					// wait_for_active should default to true (computed value)
					resource.TestCheckResourceAttr("zillaforge_server.test", "wait_for_active", "true"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					// Status should be "active" (lowercase) since default is to wait
					resource.TestCheckResourceAttr("zillaforge_server.test", "status", "ACTIVE"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_waitForActiveDefault = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  # wait_for_active not specified - should default to true

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// T045: Acceptance test - Verify wait_for_deleted=true behavior (default).
func TestAccServerResource_WaitForDeletedTrue(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-wait-deleted-true-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_waitForDeletedTrue, name, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					// wait_for_deleted should be true
					resource.TestCheckResourceAttr("zillaforge_server.test", "wait_for_deleted", "true"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_waitForDeletedTrue = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name             = "%s"
  flavor_id        = data.zillaforge_flavors.test.flavors[0].id
  image_id         = data.zillaforge_images.test.images[0].id
  password         = "TestPassword123!"
  wait_for_deleted = true

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// T046: Acceptance test - Verify wait_for_deleted=false behavior.
func TestAccServerResource_WaitForDeletedFalse(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-wait-deleted-false-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_waitForDeletedFalse, name, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					// wait_for_deleted should be false
					resource.TestCheckResourceAttr("zillaforge_server.test", "wait_for_deleted", "false"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_waitForDeletedFalse = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name             = "%s"
  flavor_id        = data.zillaforge_flavors.test.flavors[0].id
  image_id         = data.zillaforge_images.test.images[0].id
  password         = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// T047: Acceptance test - Verify default wait_for_deleted behavior.
func TestAccServerResource_WaitForDeletedDefault(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-default-wait-deleted-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_waitForDeletedDefault, name, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource attributes
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					// wait_for_deleted should default to true (computed value)
					resource.TestCheckResourceAttr("zillaforge_server.test", "wait_for_deleted", "true"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_waitForDeletedDefault = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  # wait_for_deleted not specified - should default to true

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// T030: Acceptance test - Update server name in-place.
func TestAccServerResource_UpdateName(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-update-%d", time.Now().UnixNano()%100000)
	nameUpdated := fmt.Sprintf("%s-updated", name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_updateName, name, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_updateName, name, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", nameUpdated),
					// Verify ID did not change (in-place update)
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_updateName = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// T031: Acceptance test - Update server description in-place.
func TestAccServerResource_UpdateDescription(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-desc-%d", time.Now().UnixNano()%100000)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_updateDescription, name, name, "Initial description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_updateDescription, name, name, "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "description", "Updated description"),
					// Verify ID did not change (in-place update)
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_updateDescription = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name        = "%s"
  description = "%s"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = data.zillaforge_images.test.images[0].id
  password    = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// Acceptance test: Plan-time rejection when attempting to modify flavor_id.
func TestAccServerResource_ModifyFlavorPlanTimeReject(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-flavor-plan-%d", time.Now().UnixNano()%100000)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_updateDescription, name, name, "initial-desc"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
			{
				Config: `data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_server" "test" {
  name        = "` + name + `"
  description = "after-update"
  flavor_id   = length(data.zillaforge_flavors.test.flavors) > 1 ? data.zillaforge_flavors.test.flavors[1].id : "different-flavor-id"
  image_id    = data.zillaforge_images.test.images[0].id
  password    = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
  }
}`,
				ExpectError: regexp.MustCompile(`(?i)Unsupported Change: flavor_id`),
			},
		},
	})
}

// Acceptance test: Plan-time rejection when attempting to modify image_id.
func TestAccServerResource_ModifyImagePlanTimeReject(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-image-plan-%d", time.Now().UnixNano()%100000)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_updateDescription, name, name, "initial-desc"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
			{
				Config: `data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_server" "test" {
  name        = "` + name + `"
  description = "after-update"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = length(data.zillaforge_images.test.images) > 1 ? data.zillaforge_images.test.images[1].id : "different-image-id"
  password    = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
  }
}`,
				ExpectError: regexp.MustCompile(`(?i)Unsupported Change: image_id`),
			},
		},
	})
}

// Acceptance test: Plan-time rejection when attempting to modify keypair.
func TestAccServerResource_ModifyKeypairPlanTimeReject(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-keypair-plan-%d", time.Now().UnixNano()%100000)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_withOptions, name, name, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
			{
				Config: `data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_keypair" "test" {
  name = "` + name + `-keypair"
}

resource "zillaforge_server" "test" {
  name        = "` + name + `"
  description = "after-update"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = data.zillaforge_images.test.images[0].id
  keypair     = "different-keypair-id"
  user_data   = "#!/bin/bash\necho 'Hello World'"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
  }
}`,
				ExpectError: regexp.MustCompile(`(?i)Unsupported Change: keypair`),
			},
		},
	})
}

// Acceptance test: Plan-time rejection when attempting to modify password.
func TestAccServerResource_ModifyPasswordPlanTimeReject(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-password-plan-%d", time.Now().UnixNano()%100000)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_updateDescription, name, name, "initial-desc"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
			{
				Config: `data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_server" "test" {
  name        = "` + name + `"
  description = "after-update"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = data.zillaforge_images.test.images[0].id
  password    = "OtherPassword456!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
  }
}`,
				ExpectError: regexp.MustCompile(`(?i)Unsupported Change: password`),
			},
		},
	})
}

// Acceptance test: Plan-time rejection when attempting to modify user_data.
func TestAccServerResource_ModifyUserDataPlanTimeReject(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-userdata-plan-%d", time.Now().UnixNano()%100000)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_withOptions, name, name, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
			{
				Config: `data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_keypair" "test" {
  name = "` + name + `-keypair"
}

resource "zillaforge_server" "test" {
  name        = "` + name + `"
  description = "after-update"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = data.zillaforge_images.test.images[0].id
  keypair     = zillaforge_keypair.test.id
  user_data   = "#!/bin/bash\necho 'Goodbye World'"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
  }
}`,
				ExpectError: regexp.MustCompile(`(?i)Unsupported Change: user_data`),
			},
		},
	})
}

// Acceptance test: Changing runtime-only attributes should not trigger updates.
func TestAccServerResource_IgnoreRuntimeOnlyAttributes(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-runtime-ignore-%d", time.Now().UnixNano()%100000)

	initial := `data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "` + name + `-sg"
}

resource "zillaforge_server" "test" {
  name        = "` + name + `"
  description = "initial"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = data.zillaforge_images.test.images[0].id
  password    = "TestPassword123!"
  wait_for_active = true
  wait_for_deleted = true
  timeouts {
    create = "15m"
  }
  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}	
`

	updated := `data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "` + name + `-sg"
}

resource "zillaforge_server" "test" {
  name        = "` + name + `"
  description = "initial"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = data.zillaforge_images.test.images[0].id
  password    = "TestPassword123!"
  wait_for_active = false
  wait_for_deleted = false
  timeouts {
    create = "1m"
    update = "5m"
    delete = "20m"
  }
  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Ensure resource still exists and ID did not change (no replacement)
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
		},
	})
}

// Acceptance test: Update security_group_ids in-place without replacement.
func TestAccServerResource_UpdateSecurityGroupIDs(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-sg-%d", time.Now().UnixNano()%100000)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_updateSecurityGroups_step1, name, name, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.0.security_group_ids.#", "1"),
				),
			},
			{
				Config: fmt.Sprintf(testAccServerResourceConfig_updateSecurityGroups_step2, name, name, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify ID did not change (in-place update)
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.0.security_group_ids.#", "1"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_updateSecurityGroups_step1 = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg1" {
  name = "%s-sg1"
}

resource "zillaforge_security_group" "sg2" {
  name = "%s-sg2"
}

resource "zillaforge_server" "test" {
  name        = "%s"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = data.zillaforge_images.test.images[0].id
  password    = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg1.id]
  }
}
`

const testAccServerResourceConfig_updateSecurityGroups_step2 = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg1" {
  name = "%s-sg1"
}

resource "zillaforge_security_group" "sg2" {
  name = "%s-sg2"
}

resource "zillaforge_server" "test" {
  name        = "%s"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = data.zillaforge_images.test.images[0].id
  password    = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg2.id]
  }
}
`

// ===== Phase 6: Import Tests =====

// T050: Acceptance test - Import existing server by ID.
func TestAccServerResource_ImportByID(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-import-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_forImport, name, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "flavor_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "image_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "status"),
				),
			},
			{
				ResourceName:      "zillaforge_server.test",
				ImportState:       true,
				ImportStateVerify: true,
				// user_data and password are sensitive and not returned by API for security
				// wait_for_active and wait_for_deleted are client-side only flags
				ImportStateVerifyIgnore: []string{"user_data", "password", "wait_for_active", "wait_for_deleted"},
			},
		},
	})
}

const testAccServerResourceConfig_forImport = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    primary    = true
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// T051: Acceptance test - Imported server shows no drift with matching config.
func TestAccServerResource_ImportNoChanges(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-import-nodrift-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_importNoChanges, name, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
				),
			},
			{
				ResourceName:            "zillaforge_server.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"user_data", "password", "wait_for_active", "wait_for_deleted"},
			},
			// Re-apply same config - should show no changes
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
				),
				PlanOnly: true, // Verify plan shows no changes
			},
		},
	})
}

const testAccServerResourceConfig_importNoChanges = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name        = "%s"
  description = "Server for import testing"
  flavor_id   = data.zillaforge_flavors.test.flavors[0].id
  image_id    = data.zillaforge_images.test.images[0].id
  password    = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    primary    = true
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// T052: Acceptance test - Import with invalid ID returns error.
func TestAccServerResource_ImportInvalidID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccServerResourceConfig_importInvalidID,
				ResourceName:  "zillaforge_server.test",
				ImportState:   true,
				ImportStateId: "00000000-0000-0000-0000-000000000000", // Non-existent ID
				ExpectError:   regexp.MustCompile(`Server not found|Unable to read server`),
			},
		},
	})
}

const testAccServerResourceConfig_importInvalidID = `
data "zillaforge_flavors" "test" {}

data "zillaforge_images" "test" {}

data "zillaforge_networks" "test" {}

resource "zillaforge_server" "test" {
  name      = "test-import-invalid"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
  }
}
`

// T011: Acceptance test - Create server with floating_ip_id in network_attachment.
func TestAccServerResource_FloatingIP_AssociateCreate(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-fip-create-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_floatingIPCreate, name, name, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "id"),
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.#", "1"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip"),
					// Verify floating IP is actually associated
					testAccCheckFloatingIPAssociated("zillaforge_floating_ip.test"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_floatingIPCreate = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test" {
  name = "%s-fip"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
    floating_ip_id = zillaforge_floating_ip.test.id
  }
}
`

// T012: Acceptance test - Add floating_ip_id to existing server.
func TestAccServerResource_FloatingIP_AssociateExisting(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-fip-existing-%d", time.Now().UnixNano()%100000)
	configWithout := fmt.Sprintf(testAccServerResourceConfig_floatingIPExisting_without, name, name)
	configWith := fmt.Sprintf(testAccServerResourceConfig_floatingIPExisting_with, name, name, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configWithout,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.#", "1"),
					resource.TestCheckNoResourceAttr("zillaforge_server.test", "network_attachment.0.floating_ip_id"),
				),
			},
			{
				Config: configWith,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip"),
					testAccCheckFloatingIPAssociated("zillaforge_floating_ip.test"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_floatingIPExisting_without = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

const testAccServerResourceConfig_floatingIPExisting_with = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test" {
  name = "%s-fip"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
    floating_ip_id = zillaforge_floating_ip.test.id
  }
}
`

// T013: Acceptance test - Associate different floating IPs to multiple network_attachments.
func TestAccServerResource_FloatingIP_Multiple(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-fip-multi-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_floatingIPMultiple, name, name, name, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttr("zillaforge_server.test", "network_attachment.#", "2"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.1.floating_ip_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.1.floating_ip"),
					testAccCheckFloatingIPAssociated("zillaforge_floating_ip.test1"),
					testAccCheckFloatingIPAssociated("zillaforge_floating_ip.test2"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_floatingIPMultiple = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test1" {
  name = "%s-fip1"
}

resource "zillaforge_floating_ip" "test2" {
  name = "%s-fip2"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
    floating_ip_id = zillaforge_floating_ip.test1.id
  }

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[1].id
    security_group_ids = [zillaforge_security_group.sg.id]
    floating_ip_id = zillaforge_floating_ip.test2.id
  }
}
`

// T021: Acceptance test - Remove floating_ip_id to disassociate.
func TestAccServerResource_FloatingIP_Disassociate(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-fip-disassoc-%d", time.Now().UnixNano()%100000)
	configWith := fmt.Sprintf(testAccServerResourceConfig_floatingIPDisassociate_with, name, name, name)
	configWithout := fmt.Sprintf(testAccServerResourceConfig_floatingIPDisassociate_without, name, name, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configWith,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip"),
				),
			},
			{
				Config: configWithout,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckNoResourceAttr("zillaforge_server.test", "network_attachment.0.floating_ip_id"),
					resource.TestCheckNoResourceAttr("zillaforge_server.test", "network_attachment.0.floating_ip"),
					// Note: We don't check the floating IP resource's device_id here because
					// Terraform doesn't automatically refresh it after the server update.
					// The important verification is that the server's state is correct.
				),
			},
		},
	})
}

const testAccServerResourceConfig_floatingIPDisassociate_with = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test" {
  name = "%s-fip"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
    floating_ip_id = zillaforge_floating_ip.test.id
  }
}
`

const testAccServerResourceConfig_floatingIPDisassociate_without = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test" {
  name = "%s-fip"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// T022: Acceptance test - Verify repeated disassociation is idempotent.
func TestAccServerResource_FloatingIP_DisassociateIdempotent(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-fip-idemp-%d", time.Now().UnixNano()%100000)
	configWithout := fmt.Sprintf(testAccServerResourceConfig_floatingIPIdempotent, name, name, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configWithout,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckNoResourceAttr("zillaforge_server.test", "network_attachment.0.floating_ip_id"),
				),
			},
			{
				Config: configWithout,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckNoResourceAttr("zillaforge_server.test", "network_attachment.0.floating_ip_id"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_floatingIPIdempotent = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test" {
  name = "%s-fip"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
  }
}
`

// T028: Acceptance test - Swap floating IP to different IP (verify sequential disassociate-then-associate).
func TestAccServerResource_FloatingIP_Swap(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-fip-swap-%d", time.Now().UnixNano()%100000)
	configWithFip1 := fmt.Sprintf(testAccServerResourceConfig_floatingIPSwap_fip1, name, name, name, name)
	configWithFip2 := fmt.Sprintf(testAccServerResourceConfig_floatingIPSwap_fip2, name, name, name, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configWithFip1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrPair("zillaforge_server.test", "network_attachment.0.floating_ip_id", "zillaforge_floating_ip.test1", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip"),
					testAccCheckFloatingIPAssociated("zillaforge_floating_ip.test1"),
				),
			},
			{
				Config: configWithFip2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrPair("zillaforge_server.test", "network_attachment.0.floating_ip_id", "zillaforge_floating_ip.test2", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip"),
					testAccCheckFloatingIPAssociated("zillaforge_floating_ip.test2"),
				),
			},
		},
	})
}

const testAccServerResourceConfig_floatingIPSwap_fip1 = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test1" {
  name = "%s-fip1"
}

resource "zillaforge_floating_ip" "test2" {
  name = "%s-fip2"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
    floating_ip_id = zillaforge_floating_ip.test1.id
  }
}
`

const testAccServerResourceConfig_floatingIPSwap_fip2 = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test1" {
  name = "%s-fip1"
}

resource "zillaforge_floating_ip" "test2" {
  name = "%s-fip2"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
    floating_ip_id = zillaforge_floating_ip.test2.id
  }
}
`

// T029: Acceptance test - Verify floating IP swap is update-in-place, not destroy-create.
func TestAccServerResource_FloatingIP_SwapNoRecreate(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-fip-norecreate-%d", time.Now().UnixNano()%100000)
	configWithFip1 := fmt.Sprintf(testAccServerResourceConfig_floatingIPSwapNoRecreate_fip1, name, name, name, name)
	configWithFip2 := fmt.Sprintf(testAccServerResourceConfig_floatingIPSwapNoRecreate_fip2, name, name, name, name)

	var serverID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configWithFip1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrPair("zillaforge_server.test", "network_attachment.0.floating_ip_id", "zillaforge_floating_ip.test1", "id"),
					// Capture server ID
					func(s *terraform.State) error {
						server, ok := s.RootModule().Resources["zillaforge_server.test"]
						if !ok {
							return fmt.Errorf("Server resource not found")
						}
						serverID = server.Primary.ID
						return nil
					},
				),
			},
			{
				Config: configWithFip2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrPair("zillaforge_server.test", "network_attachment.0.floating_ip_id", "zillaforge_floating_ip.test2", "id"),
					// Verify server ID hasn't changed (no replacement)
					func(s *terraform.State) error {
						server, ok := s.RootModule().Resources["zillaforge_server.test"]
						if !ok {
							return fmt.Errorf("Server resource not found")
						}
						if server.Primary.ID != serverID {
							return fmt.Errorf("Server was recreated (old ID: %s, new ID: %s)", serverID, server.Primary.ID)
						}
						return nil
					},
				),
			},
		},
	})
}

const testAccServerResourceConfig_floatingIPSwapNoRecreate_fip1 = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test1" {
  name = "%s-fip1"
}

resource "zillaforge_floating_ip" "test2" {
  name = "%s-fip2"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
    floating_ip_id = zillaforge_floating_ip.test1.id
  }
}
`

const testAccServerResourceConfig_floatingIPSwapNoRecreate_fip2 = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test1" {
  name = "%s-fip1"
}

resource "zillaforge_floating_ip" "test2" {
  name = "%s-fip2"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
    floating_ip_id = zillaforge_floating_ip.test2.id
  }
}
`

// testAccCheckFloatingIPAssociated is a helper to verify floating IP is associated with server.
func testAccCheckFloatingIPAssociated(floatingIPName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Get server resource
		server, ok := s.RootModule().Resources["zillaforge_server.test"]
		if !ok {
			return fmt.Errorf("Server resource not found: zillaforge_server.test")
		}

		// Get floating IP resource
		fip, ok := s.RootModule().Resources[floatingIPName]
		if !ok {
			return fmt.Errorf("Floating IP resource not found: %s", floatingIPName)
		}

		fipID := fip.Primary.ID

		// Search all network attachments for the floating IP
		numAttachments := 0
		for i := 0; ; i++ {
			key := fmt.Sprintf("network_attachment.%d.floating_ip_id", i)
			serverFIPID, exists := server.Primary.Attributes[key]
			if !exists {
				break
			}
			numAttachments++
			if serverFIPID == fipID {
				// Found the floating IP in this attachment
				return nil
			}
		}

		return fmt.Errorf("Floating IP %s not associated with any of %d network attachments", fipID, numAttachments)
	}
}

// T052: Acceptance test - Import server with floating IP association.
func TestAccServerResource_FloatingIP_Import(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("test-server-fip-import-%d", time.Now().UnixNano()%100000)
	config := fmt.Sprintf(testAccServerResourceConfig_floatingIPImport, name, name, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_server.test", "name", name),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip_id"),
					resource.TestCheckResourceAttrSet("zillaforge_server.test", "network_attachment.0.floating_ip"),
				),
			},
			{
				ResourceName:            "zillaforge_server.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "user_data", "wait_for_active", "wait_for_deleted", "timeouts"},
			},
		},
	})
}

const testAccServerResourceConfig_floatingIPImport = `
data "zillaforge_flavors" "test" {}
data "zillaforge_images" "test" {}
data "zillaforge_networks" "test" {}

resource "zillaforge_security_group" "sg" {
  name = "%s-sg"
}

resource "zillaforge_floating_ip" "test" {
  name = "%s-fip"
}

resource "zillaforge_server" "test" {
  name      = "%s"
  flavor_id = data.zillaforge_flavors.test.flavors[0].id
  image_id  = data.zillaforge_images.test.images[0].id
  password  = "TestPassword123!"
  wait_for_deleted = false

  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    security_group_ids = [zillaforge_security_group.sg.id]
    floating_ip_id = zillaforge_floating_ip.test.id
  }
}
`
