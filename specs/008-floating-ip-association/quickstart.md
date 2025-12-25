# Quickstart: Floating IP Association with Network Attachments

**Feature**: 008-floating-ip-association  
**Audience**: Terraform users configuring ZillaForge servers with floating IPs

## Overview

This guide shows how to associate floating IPs (public IP addresses) with network interfaces on ZillaForge servers using the `zillaforge_server` resource. Floating IPs provide public internet access to your server instances.

---

## Prerequisites

- Terraform >= 1.0
- zillaforge provider configured with valid credentials
- At least one available floating IP in your ZillaForge account
- Basic understanding of Terraform and networking concepts

---

## Basic Usage

### 1. Create Server with Floating IP

Associate a floating IP when creating a new server:

```hcl
# Fetch available floating IPs
data "zillaforge_floating_ips" "available" {
  status = "DOWN" # Unassociated floating IPs
}

# Create server with floating IP on primary interface
resource "zillaforge_server" "web" {
  name      = "web-server-01"
  flavor_id = data.zillaforge_flavors.small.flavors[0].id
  image_id  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id       = data.zillaforge_networks.default.networks[0].id
    security_group_ids = ["default", "web-access"]
    primary          = true
    
    # Associate floating IP
    floating_ip_id   = data.zillaforge_floating_ips.available.floating_ips[0].id
  }
}

# Output the public IP address
output "server_public_ip" {
  value = zillaforge_server.web.network_attachment[0].floating_ip
  description = "Public IP address of the web server"
}
```

**Result**: Server is created with a floating IP associated to its primary network interface. The `floating_ip` computed attribute shows the actual IP address (e.g., "203.0.113.10").

---

### 2. Add Floating IP to Existing Server

Associate a floating IP with an already-running server:

```hcl
# Existing server without floating IP
resource "zillaforge_server" "app" {
  name      = "app-server-01"
  flavor_id = data.zillaforge_flavors.small.flavors[0].id
  image_id  = data.zillaforge_images.ubuntu.images[0].id

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    security_group_ids = ["default"]
    primary            = true
    
    # Add this line to associate floating IP
    floating_ip_id     = data.zillaforge_floating_ips.available.floating_ips[1].id
  }
}
```

**Terraform Operation**:
```bash
$ terraform plan
# Shows: network_attachment[0].floating_ip_id will be set
# Shows: network_attachment[0].floating_ip will be known after apply

$ terraform apply
# Terraform associates the floating IP with the server
# Operation completes in <30 seconds
```

---

### 3. Multiple Network Interfaces with Different Floating IPs

Associate different floating IPs to different network interfaces:

```hcl
resource "zillaforge_server" "multi_nic" {
  name      = "multi-nic-server"
  flavor_id = data.zillaforge_flavors.medium.flavors[0].id
  image_id  = data.zillaforge_images.ubuntu.images[0].id

  # Primary interface with floating IP for public access
  network_attachment {
    network_id         = data.zillaforge_networks.public.networks[0].id
    security_group_ids = ["public-access"]
    primary            = true
    floating_ip_id     = data.zillaforge_floating_ips.available.floating_ips[0].id
  }

  # Secondary interface with different floating IP for management
  network_attachment {
    network_id         = data.zillaforge_networks.management.networks[0].id
    security_group_ids = ["management"]
    floating_ip_id     = data.zillaforge_floating_ips.available.floating_ips[1].id
  }
}

output "public_ip" {
  value = zillaforge_server.multi_nic.network_attachment[0].floating_ip
}

output "management_ip" {
  value = zillaforge_server.multi_nic.network_attachment[1].floating_ip
}
```

**Result**: Server has two network interfaces, each with its own floating IP for different purposes.

---

## Advanced Scenarios

### Swap Floating IPs Between Servers

Move a floating IP from one server to another:

```hcl
# Step 1: Remove floating IP from server A
resource "zillaforge_server" "server_a" {
  name      = "server-a"
  # ... other config ...

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    security_group_ids = ["default"]
    # Remove or comment out floating_ip_id
    # floating_ip_id   = "550e8400-e29b-41d4-a716-446655440000"
  }
}

# Step 2: Assign floating IP to server B
resource "zillaforge_server" "server_b" {
  name      = "server-b"
  # ... other config ...

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    security_group_ids = ["default"]
    # Add floating_ip_id
    floating_ip_id     = "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**Terraform Operation**:
```bash
$ terraform apply
# Terraform performs operations sequentially:
# 1. Disassociate floating IP from server A (completes in <15s)
# 2. Associate floating IP with server B (completes in <30s)
```

**Note**: There will be a brief window (typically <1 second) where the floating IP is not associated with any server. Existing connections through that IP will be disrupted.

---

### Change Floating IP on Same Server

Swap one floating IP for another on the same network interface:

```hcl
resource "zillaforge_server" "web" {
  name      = "web-server"
  # ... other config ...

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    security_group_ids = ["web-access"]
    primary            = true
    
    # Change from old IP to new IP
    floating_ip_id     = "new-floating-ip-uuid-here"  # Was: "old-floating-ip-uuid-here"
  }
}
```

**Terraform Operation**:
```bash
$ terraform apply
# Terraform performs sequential operations:
# 1. Disassociate old floating IP (completes in <15s)
# 2. Associate new floating IP (completes in <30s)
# Brief window where no floating IP is associated
```

---

### Remove Floating IP (Disassociate)

Remove a floating IP from a server to make it private-only:

```hcl
resource "zillaforge_server" "app" {
  name      = "app-server"
  # ... other config ...

  network_attachment {
    network_id         = data.zillaforge_networks.default.networks[0].id
    security_group_ids = ["internal"]
    primary            = true
    
    # Remove or comment out floating_ip_id to disassociate
    # floating_ip_id   = "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**Result**: Floating IP is disassociated from the server. The floating IP returns to the pool and can be associated with a different server. The `floating_ip` computed attribute becomes null.

---

## Error Handling

### Invalid Floating IP UUID

```hcl
resource "zillaforge_server" "bad_uuid" {
  # ... config ...

  network_attachment {
    network_id     = "..."
    floating_ip_id = "not-a-valid-uuid"  # ERROR: Must be UUID format
  }
}
```

**Error**:
```
Error: Invalid Attribute Value

floating_ip_id must be a valid UUID format 
(e.g., 550e8400-e29b-41d4-a716-446655440000)
```

**Resolution**: Use a valid UUID. Query available floating IPs with `data.zillaforge_floating_ips`.

---

### Floating IP Already In Use

```hcl
resource "zillaforge_server" "duplicate" {
  # ... config ...

  network_attachment {
    network_id     = "..."
    floating_ip_id = "550e8400-..."  # Already associated with another server
  }
}
```

**Error**:
```
Error: Failed to Associate Floating IP

Floating IP 550e8400-e29b-41d4-a716-446655440000 is already associated 
with server abc-123-def-456. Disassociate it first or choose a different 
floating IP.
```

**Resolution**: Either:
1. Disassociate the floating IP from the other server first
2. Choose a different available floating IP
3. Query available (unassociated) floating IPs: `data.zillaforge_floating_ips` with `status = "DOWN"`

---

### Floating IP Not Found

```hcl
resource "zillaforge_server" "missing_fip" {
  # ... config ...

  network_attachment {
    network_id     = "..."
    floating_ip_id = "nonexistent-uuid"  # Floating IP doesn't exist
  }
}
```

**Error**:
```
Error: Floating IP Not Found

Could not find floating IP nonexistent-uuid. Verify the floating IP ID 
is correct using data.zillaforge_floating_ips data source.
```

**Resolution**: Verify the floating IP exists in your account:
```hcl
data "zillaforge_floating_ips" "all" {}

output "available_floating_ips" {
  value = data.zillaforge_floating_ips.all.floating_ips[*].id
}
```

---

## Import Existing Servers with Floating IPs

Import a server that already has floating IPs associated:

```bash
# Import server by ID
terraform import zillaforge_server.imported server-uuid-here
```

```hcl
# Define the resource (Terraform will populate floating_ip_id from state)
resource "zillaforge_server" "imported" {
  name      = "imported-server"
  flavor_id = "..."
  image_id  = "..."

  network_attachment {
    network_id         = "..."
    security_group_ids = ["..."]
    # After import, floating_ip_id will be populated automatically
    # You can reference it in your config or let Terraform manage it
  }
}
```

**Result**: After import, `terraform show` will display the `floating_ip_id` and `floating_ip` values that were discovered from the server's current state.

---

## Best Practices

### 1. Use Data Sources for Dynamic IPs

Query available floating IPs instead of hardcoding UUIDs:

```hcl
data "zillaforge_floating_ips" "available" {
  status = "DOWN"  # Unassociated floating IPs only
}

resource "zillaforge_server" "web" {
  # ... config ...
  
  network_attachment {
    network_id     = "..."
    floating_ip_id = data.zillaforge_floating_ips.available.floating_ips[0].id
  }
}
```

### 2. Output Public IPs for Easy Access

Make public IPs visible in Terraform output:

```hcl
output "web_server_public_ip" {
  value       = zillaforge_server.web.network_attachment[0].floating_ip
  description = "Public IP address to access the web server"
}
```

### 3. Document IP Swap Downtime

When swapping IPs, document expected downtime:

```hcl
# WARNING: Changing floating_ip_id will cause a brief (<1s) network interruption
# as the old IP is disassociated before the new IP is associated.
resource "zillaforge_server" "critical" {
  # ... config ...
  
  network_attachment {
    network_id     = "..."
    floating_ip_id = var.floating_ip_id  # Use variable for flexibility
  }
}
```

### 4. Plan Before Apply

Always review changes to floating IP associations:

```bash
$ terraform plan
# Review:
# - Which floating IPs will be disassociated
# - Which floating IPs will be associated
# - Expected downtime impact
```

---

## Troubleshooting

### Association Takes Too Long

**Symptom**: Terraform hangs during `terraform apply`

**Possible Causes**:
- Server not in ACTIVE status
- Network connectivity issues
- Cloud platform under heavy load

**Resolution**:
1. Check server status in ZillaForge console
2. Verify server is ACTIVE before associating floating IP
3. Wait and retry if platform is temporarily unavailable

---

### Disassociation Doesn't Complete

**Symptom**: Error: "Floating IP disassociation did not complete within 15 seconds"

**Resolution**:
1. Check floating IP status in ZillaForge console
2. Manually disassociate via console if stuck
3. Run `terraform refresh` to sync state
4. Retry `terraform apply`

---

## Related Resources

- **Floating IP Resource**: `zillaforge_floating_ip` - Allocate and manage floating IPs
- **Floating IP Data Source**: `zillaforge_floating_ips` - Query available floating IPs
- **Server Resource**: `zillaforge_server` - Full server configuration options
- **Network Data Source**: `zillaforge_networks` - Query available networks

---

## Summary

**Key Takeaways**:
- Use `floating_ip_id` attribute in `network_attachment` blocks to associate floating IPs
- The `floating_ip` computed attribute shows the actual IP address
- Operations are synchronous - Terraform waits for completion
- Swapping IPs causes a brief network interruption (<1s)
- Use data sources to query available floating IPs dynamically
- Always `terraform plan` before applying changes to IP associations

**Next Steps**:
1. Query available floating IPs: `data.zillaforge_floating_ips`
2. Update server configuration to add `floating_ip_id`
3. Run `terraform plan` to review changes
4. Run `terraform apply` to associate floating IPs
5. Verify connectivity to public IP addresses
