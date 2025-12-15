# Quickstart: Security Group Data Source and Resource

**Feature**: 004-security-group-data-resource  
**Audience**: Terraform users managing ZillaForge VPS infrastructure  
**Prerequisites**: ZillaForge provider configured with valid credentials

## Overview

Security groups provide stateful firewall rules for VPS instances. This guide shows how to:
1. Create security groups with inbound/outbound rules
2. Query existing security groups
3. Import manually-created security groups
4. Common patterns and best practices

## Rule conventions

- **Protocol**: one of `tcp`, `udp`, `icmp`, or `any` (lowercased). Use `any` when the rule should apply to all protocols.
- **Port range**: a single port (`22`), a range (`8000-8100`), or the literal `all`. The value `all` is equivalent to the numeric range `1-65535`. For `icmp`, `port_range` should be `all` (ICMP does not use ports).

## 5-Minute Quick Start

### Step 1: Create Your First Security Group

Create a file `security-groups.tf`:

```hcl
# Allow SSH and HTTP/HTTPS traffic
resource "zillaforge_security_group" "web" {
  name        = "web-servers"
  description = "Security group for web tier"
  
  # Allow SSH from your IP only
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "203.0.113.100/32"  # Replace with your IP
  }
  
  # Allow HTTP from anywhere
  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }
  
  # Allow HTTPS from anywhere
  ingress_rule {
    protocol    = "tcp"
    port_range  = "443"
    source_cidr = "0.0.0.0/0"
  }
  
  # Allow all outbound traffic
  egress_rule {
    protocol          = "any"
    port_range        = "all"
    destination_cidr  = "0.0.0.0/0"
  }
}

output "security_group_id" {
  value = zillaforge_security_group.web.id
}
```

### Step 2: Apply Configuration

```bash
terraform init
terraform plan
terraform apply
```

**Result**: Security group created with ID like `sg-12345678-1234-1234-1234-123456789abc`

### Step 3: Query the Security Group

Create `data-sources.tf`:

```hcl
data "zillaforge_security_groups" "web" {
  name = "web-servers"
}

output "web_sg_rules" {
  value = data.zillaforge_security_groups.web.security_groups[0].ingress_rules
}
```

Run `terraform apply` to see the rules.

---

## Common Use Cases

### Use Case 1: Web Application Tier

**Scenario**: Public-facing web servers with SSH access restricted to office network

```hcl
resource "zillaforge_security_group" "web_tier" {
  name        = "web-tier-prod"
  description = "Web application servers"
  
  # HTTP from anywhere
  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }
  
  # HTTPS from anywhere
  ingress_rule {
    protocol    = "tcp"
    port_range  = "443"
    source_cidr = "0.0.0.0/0"
  }
  
  # SSH from office network only
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "198.51.100.0/24"  # Office CIDR
  }
  
  # Unrestricted outbound (for package updates, API calls)
  egress_rule {
    protocol          = "any"
    port_range        = "all"
    destination_cidr  = "0.0.0.0/0"
  }
}
```

---

### Use Case 2: Database Tier

**Scenario**: PostgreSQL database accessible only from application tier

```hcl
resource "zillaforge_security_group" "db_tier" {
  name        = "database-tier-prod"
  description = "PostgreSQL database servers"
  
  # PostgreSQL from app tier subnet only
  ingress_rule {
    protocol    = "tcp"
    port_range  = "5432"
    source_cidr = "10.0.2.0/24"  # App tier private subnet
  }
  
  # SSH from bastion host
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "10.0.0.10/32"  # Bastion server IP
  }
  
  # No outbound rules = deny all egress (database doesn't initiate connections)
}
```

---

### Use Case 3: Application Tier (Mid-Tier)

**Scenario**: App servers communicating with database and internet

```hcl
resource "zillaforge_security_group" "app_tier" {
  name        = "app-tier-prod"
  description = "Application backend servers"
  
  # Application port from load balancer subnet
  ingress_rule {
    protocol    = "tcp"
    port_range  = "8080"
    source_cidr = "10.0.1.0/24"  # Load balancer subnet
  }
  
  # SSH from bastion
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "10.0.0.10/32"
  }
  
  # Allow outbound to database subnet (port 5432)
  egress_rule {
    protocol          = "tcp"
    port_range        = "5432"
    destination_cidr  = "10.0.2.0/24"
  }
  
  # Allow outbound HTTPS for API calls
  egress_rule {
    protocol          = "tcp"
    port_range        = "443"
    destination_cidr  = "0.0.0.0/0"
  }
}
```

---

### Use Case 4: Bastion/Jump Host

**Scenario**: SSH bastion for secure access to private instances

```hcl
resource "zillaforge_security_group" "bastion" {
  name        = "bastion-host"
  description = "SSH bastion for private subnet access"
  
  # SSH from office network
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "198.51.100.0/24"
  }
  
  # Allow outbound SSH to private instances
  egress_rule {
    protocol          = "tcp"
    port_range        = "22"
    destination_cidr  = "10.0.0.0/16"  # Private VPC CIDR
  }
  
  # Allow outbound HTTPS for system updates
  egress_rule {
    protocol          = "tcp"
    port_range        = "443"
    destination_cidr  = "0.0.0.0/0"
  }
}
```

---

### Use Case 5: ICMP/Ping Monitoring

**Scenario**: Allow ping for health checks

```hcl
resource "zillaforge_security_group" "monitoring" {
  name        = "monitoring-enabled"
  description = "Allow ICMP for health checks"
  
  # Allow ICMP from monitoring subnet
  ingress_rule {
    protocol    = "icmp"
    port_range  = "all"  # Required but ignored for ICMP
    source_cidr = "10.0.10.0/24"  # Monitoring tools subnet
  }
  
  # Application ports...
  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }
}
```

---

## Data Source Usage

### Query by Name

```hcl
data "zillaforge_security_groups" "existing_web" {
  name = "web-servers"
}

# Check if it exists
output "exists" {
  value = length(data.zillaforge_security_groups.existing_web.security_groups) > 0
}

# Reference in new instance (future capability)
resource "zillaforge_vps_instance" "app" {
  name = "app-server"
  # ... other attributes ...
  security_group_ids = [
    data.zillaforge_security_groups.existing_web.security_groups[0].id
  ]
}
```

### Query by ID

```hcl
data "zillaforge_security_groups" "specific" {
  id = "sg-12345678-1234-1234-1234-123456789abc"
}

output "security_group_name" {
  value = data.zillaforge_security_groups.specific.security_groups[0].name
}
```

### List All Security Groups

```hcl
data "zillaforge_security_groups" "all" {
  # No filters
}

output "all_names" {
  value = [
    for sg in data.zillaforge_security_groups.all.security_groups : sg.name
  ]
}

# Find groups with specific characteristics
locals {
  ssh_groups = [
    for sg in data.zillaforge_security_groups.all.security_groups :
    sg if anytrue([
      for rule in sg.ingress_rules :
      rule.protocol == "tcp" && rule.port_range == "22"
    ])
  ]
}

output "ssh_enabled_groups" {
  value = [for sg in local.ssh_groups : sg.name]
}
```

---

## Importing Existing Security Groups

### Step 1: Identify Security Group ID

Use the ZillaForge console or CLI to find the security group ID (format: `sg-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`).

### Step 2: Create Configuration Stub

```hcl
resource "zillaforge_security_group" "imported" {
  # Empty block - will be populated after import
}
```

### Step 3: Import

```bash
terraform import zillaforge_security_group.imported sg-12345678-1234-1234-1234-123456789abc
```

### Step 4: Fill Configuration

Run `terraform show` to see the imported state, then add matching configuration:

```hcl
resource "zillaforge_security_group" "imported" {
  name        = "legacy-web-servers"  # From terraform show
  description = "Imported security group"
  
  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }
  
  # ... match all rules from state ...
}
```

### Step 5: Verify

```bash
terraform plan
# Should show "No changes" if configuration matches imported state
```

---

## Updating Security Groups

### Adding Rules

```hcl
resource "zillaforge_security_group" "web" {
  name        = "web-servers"
  description = "Security group for web tier"
  
  # Existing rules...
  ingress_rule {
    protocol    = "tcp"
    port_range  = "443"
    source_cidr = "0.0.0.0/0"
  }
  
  # NEW: Add SSH access
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "203.0.113.0/24"
  }
}
```

Run `terraform apply` - SSH rule is added without disrupting existing rules (stateful update).

### Removing Rules

Delete the `ingress_rule` block and run `terraform apply`. The rule is removed from the security group.

### Changing Description

```hcl
resource "zillaforge_security_group" "web" {
  name        = "web-servers"
  description = "Updated description"  # Changed
  
  # Rules unchanged...
}
```

Run `terraform apply` - description updated in-place (no disruption).

### Changing Name (Replacement)

⚠️ **Warning**: Changing `name` forces resource replacement (destroy + recreate).

```hcl
resource "zillaforge_security_group" "web" {
  name = "web-servers-v2"  # Changed from "web-servers"
  # ...
}
```

`terraform plan` shows:
```
# zillaforge_security_group.web must be replaced
-/+ resource "zillaforge_security_group" "web" {
      ~ name = "web-servers" -> "web-servers-v2" # forces replacement
      # ...
    }
```

To avoid downtime, create new security group first, update instances, then delete old:

```hcl
resource "zillaforge_security_group" "web_v2" {
  name = "web-servers-v2"
  # ... same rules ...
}

# Update instances to use new security group
# Then remove old zillaforge_security_group.web
```

---

## Deletion Safety

### Attempting to Delete Attached Security Group

If a security group is attached to instances, deletion will fail:

```bash
terraform destroy -target=zillaforge_security_group.web
```

**Error**:
```
Error: Cannot Delete Security Group

Security group "web-servers" is attached to instances: [i-abc123, i-def456].
Detach the security group from all instances before deletion.
```

**Solution**: Remove security group from instances first, then delete:

```hcl
# Remove security_group_ids reference from instances
resource "zillaforge_vps_instance" "server" {
  # ... other attributes ...
  security_group_ids = []  # Removed sg reference
}
```

Apply changes, then delete security group.

---

## Advanced Patterns

### Default Security Group for All Instances

```hcl
# Create base security group
resource "zillaforge_security_group" "default" {
  name        = "default-instance-rules"
  description = "Base rules for all instances"
  
  # SSH from office
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "198.51.100.0/24"
  }
  
  # ICMP for monitoring
  ingress_rule {
    protocol    = "icmp"
    port_range  = "all"
    source_cidr = "10.0.10.0/24"
  }
  
  # Outbound internet access
  egress_rule {
    protocol          = "any"
    port_range        = "all"
    destination_cidr  = "0.0.0.0/0"
  }
}

# Use in all instances
locals {
  default_sg_id = zillaforge_security_group.default.id
}
```

### Multiple Security Groups per Instance (Future)

```hcl
resource "zillaforge_security_group" "base" {
  name = "base-rules"
  # SSH, monitoring, etc.
}

resource "zillaforge_security_group" "web" {
  name = "web-rules"
  # HTTP, HTTPS
}

resource "zillaforge_vps_instance" "web_server" {
  name = "web-1"
  # Attach multiple security groups (union of all rules)
  security_group_ids = [
    zillaforge_security_group.base.id,
    zillaforge_security_group.web.id
  ]
}
```

**Evaluation**: Union of all rules (most permissive wins). If ANY security group allows traffic, it's permitted.

### IPv6 Support

```hcl
resource "zillaforge_security_group" "dual_stack" {
  name = "ipv4-and-ipv6"
  
  # Allow HTTP from IPv4
  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }
  
  # Allow HTTP from IPv6
  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "::/0"
  }
  
  # Outbound for both protocols
  egress_rule {
    protocol          = "any"
    port_range        = "all"
    destination_cidr  = "0.0.0.0/0"
  }
  
  egress_rule {
    protocol          = "any"
    port_range        = "all"
    destination_cidr  = "::/0"
  }
}
```

### Attaching Security Groups to VPS Instances

**Note**: VPS instance resource (`zillaforge_vps_instance`) is planned for future implementation. This section describes the expected pattern.

#### Basic Attachment Pattern

```hcl
# Create security group
resource "zillaforge_security_group" "web" {
  name = "web-servers"
  
  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }
}

# Attach to VPS instance (future resource)
resource "zillaforge_vps_instance" "web_server" {
  name              = "web-prod-01"
  flavor_id         = "flavor-uuid"
  security_group_id = zillaforge_security_group.web.id
}
```

#### Referencing by Data Source Lookup

```hcl
# Look up existing security group
data "zillaforge_security_groups" "existing" {
  name = "manually-created-sg"
}

# Reference in VPS instance
resource "zillaforge_vps_instance" "app_server" {
  name              = "app-01"
  flavor_id         = "flavor-uuid"
  security_group_id = data.zillaforge_security_groups.existing.security_groups[0].id
}
```

#### Multi-Instance Deployment

```hcl
# Create shared security group
resource "zillaforge_security_group" "web_tier" {
  name = "web-tier-shared"
  # ... rules ...
}

# Deploy multiple instances with same security group
resource "zillaforge_vps_instance" "web_servers" {
  count = 3
  
  name              = "web-${count.index + 1}"
  flavor_id         = var.web_flavor_id
  security_group_id = zillaforge_security_group.web_tier.id
}
```

#### Outputs for Cross-Stack References

```hcl
# In security-groups.tf (separate module/stack)
resource "zillaforge_security_group" "shared" {
  name = "shared-infrastructure"
  # ... rules ...
}

output "shared_security_group_id" {
  description = "Security group ID for shared infrastructure"
  value       = zillaforge_security_group.shared.id
}

# In instances.tf (different module/stack)
variable "security_group_id" {
  description = "Security group ID from infrastructure stack"
  type        = string
}

resource "zillaforge_vps_instance" "server" {
  name              = "server-01"
  security_group_id = var.security_group_id
}
```

---

## Best Practices

### 1. Principle of Least Privilege

Only allow necessary ports and sources:

❌ **Bad**: Overly permissive
```hcl
ingress_rule {
  protocol    = "any"
  port_range  = "all"
  source_cidr = "0.0.0.0/0"
}
```

✅ **Good**: Specific rules
```hcl
ingress_rule {
  protocol    = "tcp"
  port_range  = "443"
  source_cidr = "0.0.0.0/0"
}
```

### 2. Use Descriptive Names

✅ **Good**:
```hcl
name = "prod-web-servers"
description = "HTTPS access for production web tier (deployed 2025-12-14)"
```

❌ **Bad**:
```hcl
name = "sg1"
description = ""
```

### 3. Default Deny Approach

Start with no rules (deny all), then explicitly allow:

```hcl
resource "zillaforge_security_group" "secure" {
  name        = "secure-tier"
  description = "Explicit allow-list only"
  
  # Only necessary rules added here
  ingress_rule {
    protocol    = "tcp"
    port_range  = "443"
    source_cidr = "10.0.1.0/24"
  }
  
  # No egress rules = deny all outbound
}
```

### 4. Organize by Environment

```hcl
locals {
  env = "prod"
}

resource "zillaforge_security_group" "web" {
  name        = "${local.env}-web-servers"
  description = "Web tier for ${local.env} environment"
  # ...
}
```

### 5. Document Special Rules

```hcl
ingress_rule {
  protocol    = "tcp"
  port_range  = "8443"
  source_cidr = "203.0.113.50/32"
  # NOTE: VPN endpoint IP - do not remove without ops team approval
}
```

Use description field:
```hcl
description = "Updated 2025-12-14: Added port 8443 for VPN endpoint (ticket #12345)"
```

---

## Troubleshooting

### "Security group not found" on plan/apply

**Cause**: Security group deleted outside Terraform

**Solution**: Remove from state or recreate:
```bash
terraform state rm zillaforge_security_group.missing
terraform apply  # Recreates resource
```

### "Port must be 1-65535" error

**Cause**: Invalid port number

**Solution**: Check port_range syntax:
- ✅ `"22"`, `"80-443"`, `"all"`
- ❌ `"0"`, `"70000"`, `"22-"`, `"-443"`

### "Invalid CIDR notation" error

**Cause**: Malformed CIDR block

**Solution**: Verify CIDR format:
- ✅ `"192.168.1.0/24"`, `"10.0.0.1/32"`, `"0.0.0.0/0"`
- ❌ `"192.168.1.0"`, `"192.168.1.0/255.255.255.0"`, `"192.168.1.5/24"` (host bits set)

### Rules not taking effect

**Cause**: Stateful firewall behavior misunderstanding

**Remember**: 
- Inbound rules automatically allow response traffic
- No need for explicit egress rule to allow SSH responses
- If instance can't reach internet, check egress rules

### "Cannot delete: attached to instances" error

**Cause**: Security group in use

**Solution**: 
1. Run `terraform state show` to see attached instances
2. Update instance configurations to remove security group reference
3. Apply changes
4. Then delete security group

---

## Next Steps

- Attach security groups to VPS instances (future capability: `zillaforge_vps_instance` resource with `security_group_ids` attribute)
- Explore combining multiple security groups per instance for modular firewall policies
- Review ZillaForge audit logs to track security group changes
- Set up monitoring alerts for unauthorized security group modifications

---

## Reference

- [Security Group Resource Schema](contracts/security-group-resource-schema.md)
- [Security Groups Data Source Schema](contracts/security-groups-data-source-schema.md)
- [Data Model Documentation](data-model.md)
- [Research Document](research.md)
