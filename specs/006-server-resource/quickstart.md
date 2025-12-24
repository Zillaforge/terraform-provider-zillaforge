# Quickstart: zillaforge_server Resource

**Resource**: `zillaforge_server`  
**Purpose**: Manage ZillaForge VPS virtual machine instances

---

## Basic Example

Create a server with minimum required configuration:

```hcl
# Configure the ZillaForge provider
provider "zillaforge" {
  api_endpoint = "https://api.zillaforge.com"
  api_token    = var.zillaforge_token
}

# Data source for available flavors
data "zillaforge_flavors" "available" {}

# Data source for available images
data "zillaforge_images" "ubuntu" {
  name = "Ubuntu 22.04 LTS"
}

# Data source for default network
data "zillaforge_networks" "default" {
  name = "default"
}

# Data source for default security group
data "zillaforge_security_groups" "default" {
  name = "default"
}

# Create a server
resource "zillaforge_server" "web" {
  name   = "web-server-01"
  flavor = data.zillaforge_flavors.available.flavors[0].id
  image  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    security_group_ids = [
      data.zillaforge_security_groups.default.security_groups[0].id
    ]
  }
}

# Output server details
output "server_id" {
  value = zillaforge_server.web.id
}

output "server_ip" {
  value = zillaforge_server.web.ip_addresses
}

output "server_status" {
  value = zillaforge_server.web.status
}
```

---

## Complete Example with All Options

Create a server with all available configuration options:

```hcl
# Keypair for SSH access
resource "zillaforge_keypair" "admin" {
  name       = "admin-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

# Custom security group for web traffic
resource "zillaforge_security_group" "web" {
  name        = "web-sg"
  description = "Allow HTTP/HTTPS traffic"

  rule {
    direction   = "ingress"
    protocol    = "tcp"
    port_range  = "80"
    cidr_blocks = ["0.0.0.0/0"]
  }

  rule {
    direction   = "ingress"
    protocol    = "tcp"
    port_range  = "443"
    cidr_blocks = ["0.0.0.0/0"]
  }

  rule {
    direction   = "ingress"
    protocol    = "tcp"
    port_range  = "22"
    cidr_blocks = ["10.0.0.0/8"]
  }
}

# Server with full configuration
resource "zillaforge_server" "app" {
  name        = "app-server-01"
  description = "Application server for production environment"

  # Compute resources
  flavor = data.zillaforge_flavors.available.flavors[0].id  # 4 vCPU, 8GB RAM
  image  = data.zillaforge_images.ubuntu.images[0].id

  # Network configuration
  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
  }

  # Security: assign security groups per network attachment via `sg_ids`

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    security_group_ids = [
      zillaforge_security_group.web.id,
      data.zillaforge_security_groups.default.security_groups[0].id,
    ]
  }

  keypair = zillaforge_keypair.admin.name

  # Cloud-init configuration
  user_data = base64encode(<<-EOF
    #cloud-config
    package_update: true
    package_upgrade: true
    packages:
      - nginx
      - docker.io
    runcmd:
      - systemctl enable nginx
      - systemctl start nginx
      - usermod -aG docker ubuntu
  EOF
  )

  # Wait for server to become active (default: true)
  wait_for_active = true

  # Custom timeouts
  timeouts {
    create = "15m"
    update = "10m"
    delete = "5m"
  }
}
```

---

## Multi-Network Example

Create a server with multiple network interfaces:

```hcl
# Data sources for networks
data "zillaforge_networks" "public" {
  name = "public-network"
}

data "zillaforge_networks" "private" {
  name = "private-network"
}

data "zillaforge_networks" "management" {
  name = "management-network"
}

# Server with three network interfaces
resource "zillaforge_server" "database" {
  name   = "db-server-01"
  flavor = data.zillaforge_flavors.available.flavors[1].id
  image  = data.zillaforge_images.ubuntu.images[0].id

  # Primary network (public-facing)
  network_attachment {
    network_id = data.zillaforge_networks.public.networks[0].id
    primary    = true
    security_group_ids = [
      data.zillaforge_security_groups.default.security_groups[0].id,
    ]
  }

  # Secondary network (private backend)
  network_attachment {
    network_id = data.zillaforge_networks.private.networks[0].id
    ip_address = "192.168.1.100"  # Fixed IP for internal communication
    security_group_ids = [
      data.zillaforge_security_groups.default.security_groups[0].id,
    ]
  }

  # Tertiary network (management)
  network_attachment {
    network_id = data.zillaforge_networks.management.networks[0].id
    sg_ids = [
      data.zillaforge_security_groups.default.security_groups[0].id,
    ]
  }
}

# Access different IPs
output "db_public_ip" {
  value = zillaforge_server.database.ip_addresses[0]
}

output "db_private_ip" {
  value = "192.168.1.100"  # Fixed IP from network_attachment
}
```

---

## In-Place Update Examples

### Update Server Name and Description

```hcl
resource "zillaforge_server" "web" {
  name        = "web-server-02"  # Changed from "web-server-01"
  description = "Updated description"  # Added description
  
  flavor = "flavor-small"
  image  = "ubuntu-22.04"

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    sg_ids     = [data.zillaforge_security_groups.default.security_groups[0].id]
  }

# Terraform will update in-place without destroying the instance
```

### Add/Remove Network Attachments

```hcl
resource "zillaforge_server" "web" {
  name   = "web-server-01"
  flavor = data.zillaforge_flavors.available.flavors[0].id
  image  = data.zillaforge_images.ubuntu.images[0].id

  # Primary network (existing)
  network_attachment {
    network_id = data.zillaforge_networks.public.networks[0].id
    primary    = true
    sg_ids     = [data.zillaforge_security_groups.default.security_groups[0].id]
  }

  # Add new secondary network
  network_attachment {
    network_id = data.zillaforge_networks.private.networks[0].id
    sg_ids     = [data.zillaforge_security_groups.default.security_groups[0].id]
  }
}

# Terraform will attach the new network interface without destroying the instance
```

### Update Security Groups

```hcl
resource "zillaforge_server" "web" {
  name   = "web-server-01"
  flavor = data.zillaforge_flavors.available.flavors[0].id
  image  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
  }

  # Add additional security group on the primary NIC
  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    sg_ids = [
      data.zillaforge_security_groups.default.security_groups[0].id,
      zillaforge_security_group.web.id,  # New security group
    ]
  }
}

# Terraform will update security groups on the NIC in-place
```

---

## Force Replacement Examples

### Change Flavor (Out of Scope - Force Replacement)

```hcl
resource "zillaforge_server" "web" {
  name   = "web-server-01"
  flavor = data.zillaforge_flavors.available.flavors[1].id  # Changed from data.zillaforge_flavors.available.flavors[0].id
  image  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    sg_ids     = [data.zillaforge_security_groups.default.security_groups[0].id]
  }

# Terraform will DESTROY and RECREATE the instance
# Use `terraform plan` to preview this behavior
```

### Change Image

```hcl
resource "zillaforge_server" "web" {
  name   = "web-server-01"
  flavor = data.zillaforge_flavors.available.flavors[0].id
  image  = data.zillaforge_images.test.images[0].id  # Changed from data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    sg_ids     = [data.zillaforge_security_groups.default.security_groups[0].id]
  }

# Terraform will DESTROY and RECREATE the instance
```



---

## Import Example

Import an existing server into Terraform state:

```bash
# Import by server ID
terraform import zillaforge_server.web srv-abc123def456

# After import, write the configuration to match the existing server
```

Configuration after import:

```hcl
resource "zillaforge_server" "web" {
  name   = "existing-web-server"  # Must match imported server
  flavor = data.zillaforge_flavors.available.flavors[0].id          # Must match imported server
  image  = data.zillaforge_images.ubuntu.images[0].id          # Must match imported server

  network_attachment {
    network_id = "net-xyz789"
    primary    = true
    sg_ids     = ["sg-default123"]
  }
}

# Run `terraform plan` to verify configuration matches imported state
```

---

## Computed Attributes Example

Access computed (read-only) attributes:

```hcl
resource "zillaforge_server" "web" {
  name   = "web-server-01"
  flavor = data.zillaforge_flavors.available.flavors[0].id
  image  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    sg_ids     = [data.zillaforge_security_groups.default.security_groups[0].id]
  }

# Use computed attributes in outputs or other resources
output "server_details" {
  value = {
    id          = zillaforge_server.web.id
    status      = zillaforge_server.web.status
    ip_addresses = zillaforge_server.web.ip_addresses
    created_at  = zillaforge_server.web.created_at
  }
}

# Use server IP in another resource
resource "dns_a_record" "web" {
  name  = "web.example.com"
  value = zillaforge_server.web.ip_addresses[0]
}
```

---

## Asynchronous Creation Example

Create servers without waiting for active status (for faster deployments):

```hcl
# Server that returns immediately without waiting for active status
resource "zillaforge_server" "batch" {
  name   = "batch-server-01"
  flavor = "flavor-small"
  image  = "ubuntu-22.04"

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    security_group_ids = [
      data.zillaforge_security_groups.default.security_groups[0].id
    ]
  }

  # Don't wait for server to become active - return as soon as API responds
  wait_for_active = false
}

# Use this for:
# - Batch server creation where individual status doesn't matter
# - Scenarios where external orchestration will manage server readiness
# - Faster Terraform apply times when immediate feedback isn't needed

# Note: With wait_for_active=false, the server status may be "building" 
# immediately after creation. Use data source refresh or manual checks 
# to verify server is active before use.
```

---

## Conditional Creation Example

Create servers conditionally based on environment:

```hcl
variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "dev"
}

variable "enable_ha" {
  description = "Enable high availability"
  type        = bool
  default     = false
}

# Conditional flavor based on environment
locals {
  flavor_map = {
    dev  = data.zillaforge_flavors.available.flavors[0].id
    stag = data.zillaforge_flavors.available.flavors[1].id
    prod = data.zillaforge_flavors.available.flavors[2].id
  }
}

resource "zillaforge_server" "app" {
  count = var.enable_ha ? 2 : 1  # Create 2 instances for HA

  name   = "app-server-${count.index + 1}"
  flavor = local.flavor_map[var.environment]
  image  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    sg_ids     = [data.zillaforge_security_groups.default.security_groups[0].id]
  }

  # Production servers in specific AZ (not supported via resource)  
}
```

---

## Validation Examples

### ❌ Invalid: Multiple Primary Networks

```hcl
resource "zillaforge_server" "invalid" {
  name   = "web-server"
  flavor = data.zillaforge_flavors.available.flavors[0].id
  image  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id = "net-1"
    primary    = true  # ❌ First primary
    sg_ids = ["sg-default"]
  }

  network_attachment {
    network_id = "net-2"
    primary    = true  # ❌ Second primary - VALIDATION ERROR
    sg_ids = ["sg-default"]
  }
}

# Error: Multiple Primary Network Attachments
# Only one network attachment can have primary=true, found 2
```

### ❌ Invalid: Invalid IP Address Format

```hcl
resource "zillaforge_server" "invalid" {
  name   = "web-server"
  flavor = "flavor-small"
  image  = "ubuntu-22.04"

  network_attachment {
    network_id = "net-1"
    ip_address = "192.168.1.999"  # ❌ Invalid IP - VALIDATION ERROR
    primary    = true
    sg_ids = ["sg-default"]
  }
}

# Error: Invalid IP Address
# Value must be a valid IPv4 address
```

### ✅ Valid: Single Primary, Valid IP

```hcl
resource "zillaforge_server" "valid" {
  name   = "web-server"
  flavor = "flavor-small"
  image  = "ubuntu-22.04"

  network_attachment {
    network_id = "net-1"
    ip_address = "192.168.1.100"  # ✅ Valid IPv4
    primary    = true              # ✅ Only one primary
    sg_ids     = ["sg-default"]
  }

  network_attachment {
    network_id = "net-2"
    # primary not set (defaults to false)
    sg_ids     = ["sg-default"]
  }
}
```

---

## Lifecycle Management Examples

### Prevent Accidental Deletion

```hcl
resource "zillaforge_server" "production_db" {
  name   = "prod-db-01"
  flavor = "flavor-xlarge"
  image  = "ubuntu-22.04"

  network_attachment {
    network_id = data.zillaforge_networks.private.networks[0].id
    primary    = true
    sg_ids     = [zillaforge_security_group.database.id]
  }

  lifecycle {
    prevent_destroy = true  # Prevent accidental terraform destroy
  }
}
```

### Ignore UserData Changes

```hcl
resource "zillaforge_server" "app" {
  name     = "app-server"
  flavor   = "flavor-small"
  image    = "ubuntu-22.04"
  user_data = base64encode(file("cloud-init.yaml"))

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    sg_ids     = [data.zillaforge_security_groups.default.security_groups[0].id]
  }

  lifecycle {
    ignore_changes = [user_data]  # UserData only applied at create time
  }
}
```

### Create Before Destroy

```hcl
resource "zillaforge_server" "web" {
  name   = "web-server"
  flavor = "flavor-small"
  image  = "ubuntu-22.04"

  network_attachment {
    network_id = data.zillaforge_networks.default.networks[0].id
    primary    = true
    sg_ids     = [data.zillaforge_security_groups.default.security_groups[0].id]
  }

  lifecycle {
    create_before_destroy = true  # Create new instance before destroying old one
  }
}
```

---

## Testing Strategy

### Manual Testing Workflow

```bash
# 1. Initialize Terraform
terraform init

# 2. Validate configuration
terraform validate

# 3. Plan changes
terraform plan

# 4. Apply configuration
terraform apply

# 5. Verify server is active
terraform show | grep status

# 6. Test in-place update (change name)
# Edit configuration, then:
terraform plan   # Should show in-place update
terraform apply

# 7. Test force replacement (change flavor)
# Edit configuration, then:
terraform plan   # Should show destroy + create
terraform apply

# 8. Test import
terraform import zillaforge_server.imported srv-abc123

# 9. Destroy resources
terraform destroy
```

### Acceptance Testing

```bash
# Run provider acceptance tests
TF_ACC=1 go test -v ./internal/vps/resource -run TestAccServerResource

# Run specific test case
TF_ACC=1 go test -v ./internal/vps/resource -run TestAccServerResource_basic

# Run with verbose logging
TF_LOG=DEBUG TF_ACC=1 go test -v ./internal/vps/resource -run TestAccServerResource
```

---

## Troubleshooting

### Server Stuck in "building" Status

```bash
# Check timeout configuration
terraform plan

# Increase timeout if needed
resource "zillaforge_server" "web" {
  # ... other configuration ...

  timeouts {
    create = "20m"  # Increase from default 10m
  }
}
```

### Import State Mismatch

```bash
# After import, check for drift
terraform plan

# If drift detected, update configuration to match imported state
# Then run plan again to verify no changes
terraform plan
# Expected: "No changes. Your infrastructure matches the configuration."
```

### Network Attachment Validation Error

```bash
# Error: Only one network attachment can have primary=true
# Solution: Ensure only one network_attachment block has primary=true

# Correct configuration:
network_attachment {
  network_id = "net-1"
  primary    = true  # ✅ Only one primary
}

network_attachment {
  network_id = "net-2"
  # primary not set (defaults to false)
}
```

---

## Best Practices

1. **Use Data Sources**: Reference flavors, images, networks by data source instead of hardcoded IDs
2. **Set Timeouts**: Configure appropriate timeouts based on expected provisioning time
3. **Fixed IPs**: Use fixed IP addresses for servers that need stable internal networking
4. **Security Groups**: Assign at least one security group to each `network_attachment` via `sg_ids` (use default group if none)
5. **Cloud-Init**: Use `user_data` for initial configuration, but use configuration management tools for ongoing changes
6. **Lifecycle Policies**: Use `prevent_destroy` for critical infrastructure
7. **Import**: Use `terraform import` for existing infrastructure, then write configuration to match
8. **Testing**: Test configuration changes with `terraform plan` before applying

---

## Reference

- **Resource Documentation**: See `/docs/resources/server.md`
- **API Documentation**: ZillaForge Cloud SDK VPS Server API
- **Data Sources**: `zillaforge_flavors`, `zillaforge_images`, `zillaforge_networks`, `zillaforge_security_groups`
- **Related Resources**: `zillaforge_keypair`, `zillaforge_security_group`
