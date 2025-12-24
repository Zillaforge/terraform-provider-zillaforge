// Consolidated examples demonstrating common `zillaforge_server` usage patterns.
// This single file replaces the prior per-feature example files and keeps each
// example focused on the feature it intends to demonstrate.

// Provider configuration (shared by all examples)
terraform {
  required_providers {
    zillaforge = {
      source = "Zillaforge/zillaforge"
    }
  }
}

provider "zillaforge" {
  api_endpoint = "https://api.zillaforge.com"
  api_token    = var.zillaforge_token
}

// ---------------------------------------------------------------------------
// Common data sources
// ---------------------------------------------------------------------------

data "zillaforge_flavors" "available" {}

data "zillaforge_images" "ubuntu" {
  name = "Ubuntu 22.04 LTS"
}

data "zillaforge_networks" "default" {
  name = "default"
}

data "zillaforge_networks" "public" {
  name = "public-network"
}

data "zillaforge_networks" "private" {
  name = "private-network"
}

data "zillaforge_networks" "management" {
  name = "management-network"
}

data "zillaforge_security_groups" "default" {
  name = "default"
}

data "zillaforge_security_groups" "web" {
  name = "web-sg"
}

data "zillaforge_security_groups" "database" {
  name = "database-sg"
}

data "zillaforge_security_groups" "management" {
  name = "management-sg"
}

// ---------------------------------------------------------------------------
// Example 1: Basic server
// Demonstrates minimal required attributes and inspecting outputs
// ---------------------------------------------------------------------------

resource "zillaforge_server" "web" {
  name      = "web-server-01"
  flavor_id = data.zillaforge_flavors.available.flavors[0].id
  image_id  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    primary            = true
    security_group_ids = [data.zillaforge_security_groups.default.security_groups[0].id]
  }
}

output "web_server_id" {
  value = zillaforge_server.web.id
}

output "web_server_ip" {
  value = zillaforge_server.web.ip_addresses
}

// ---------------------------------------------------------------------------
// Example 2: Server with optional attributes
// Demonstrates keypair, cloud-init user_data, wait_for_active and timeouts
// ---------------------------------------------------------------------------

resource "zillaforge_keypair" "admin" {
  name       = "admin-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

resource "zillaforge_server" "app" {
  name        = "app-server-01"
  description = "Application server with optional attributes"

  flavor_id = data.zillaforge_flavors.available.flavors[0].id
  image_id  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    primary            = true
    security_group_ids = [data.zillaforge_security_groups.default.security_groups[0].id]
  }

  keypair = zillaforge_keypair.admin.name

  user_data = base64encode(<<-EOF
    #cloud-config
    package_update: true
    package_upgrade: true
    packages:
      - nginx
    runcmd:
      - systemctl enable nginx
      - systemctl start nginx
  EOF
  )

  wait_for_active = true

  timeouts {
    create = "15m"
    delete = "10m"
  }
}

output "app_server_id" {
  value = zillaforge_server.app.id
}

// ---------------------------------------------------------------------------
// Example 3: Server with multiple network attachments
// Demonstrates multiple NICs, fixed IP, and per-NIC security groups
// ---------------------------------------------------------------------------

resource "zillaforge_server" "database" {
  name        = "db-server-01"
  description = "Database server with multiple networks"

  flavor_id = data.zillaforge_flavors.available.flavors[1].id
  image_id  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id         = data.zillaforge_networks.public.networks[0].id
    primary            = true
    security_group_ids = [data.zillaforge_security_groups.web.security_groups[0].id]
  }

  network_attachment {
    network_id         = data.zillaforge_networks.private.networks[0].id
    ip_address         = "192.168.1.100"
    security_group_ids = [data.zillaforge_security_groups.database.security_groups[0].id]
  }

  network_attachment {
    network_id         = data.zillaforge_networks.management.networks[0].id
    security_group_ids = [data.zillaforge_security_groups.management.security_groups[0].id]
  }
}

output "database_server_ips" {
  value = zillaforge_server.database.ip_addresses
}

// ---------------------------------------------------------------------------
// Example 4: Asynchronous server creation
// Demonstrates `wait_for_active = false` for batch deployments
// ---------------------------------------------------------------------------

resource "zillaforge_server" "batch_server_1" {
  name            = "batch-server-01"
  flavor_id       = data.zillaforge_flavors.available.flavors[0].id
  image_id        = data.zillaforge_images.ubuntu.images[0].id
  password        = "SecurePassword123!"
  wait_for_active = false

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    security_group_ids = [data.zillaforge_security_groups.default.security_groups[0].id]
  }
}

resource "zillaforge_server" "batch_server_2" {
  name            = "batch-server-02"
  flavor_id       = data.zillaforge_flavors.available.flavors[0].id
  image_id        = data.zillaforge_images.ubuntu.images[0].id
  password        = "SecurePassword123!"
  wait_for_active = false

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    security_group_ids = [data.zillaforge_security_groups.default.security_groups[0].id]
  }
}

resource "zillaforge_server" "batch_server_3" {
  name            = "batch-server-03"
  flavor_id       = data.zillaforge_flavors.available.flavors[0].id
  image_id        = data.zillaforge_images.ubuntu.images[0].id
  password        = "SecurePassword123!"
  wait_for_active = false

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    security_group_ids = [data.zillaforge_security_groups.default.security_groups[0].id]
  }
}

output "batch_server_ids" {
  value = [
    zillaforge_server.batch_server_1.id,
    zillaforge_server.batch_server_2.id,
    zillaforge_server.batch_server_3.id,
  ]
}

output "batch_server_statuses" {
  value = {
    server_1 = zillaforge_server.batch_server_1.status
    server_2 = zillaforge_server.batch_server_2.status
    server_3 = zillaforge_server.batch_server_3.status
  }
}

// ---------------------------------------------------------------------------
// Example 5: Demonstrate wait_for_deleted behavior
// ---------------------------------------------------------------------------

resource "zillaforge_server" "web_wait_deleted" {
  name             = "web-wait-deleted"
  flavor_id        = data.zillaforge_flavors.available.flavors[0].id
  image_id         = data.zillaforge_images.ubuntu.images[0].id
  description      = "Server that waits for deletion (default behavior)"
  wait_for_deleted = true

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    security_group_ids = [data.zillaforge_security_groups.default.security_groups[0].id]
    primary            = true
  }

  timeouts {
    delete = "15m"
  }
}

resource "zillaforge_server" "temp_no_wait" {
  name             = "temp-no-wait"
  flavor_id        = data.zillaforge_flavors.available.flavors[0].id
  image_id         = data.zillaforge_images.ubuntu.images[0].id
  description      = "Temporary server that does not wait for deletion"
  wait_for_deleted = false

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    security_group_ids = [data.zillaforge_security_groups.default.security_groups[0].id]
    primary            = true
  }
}
