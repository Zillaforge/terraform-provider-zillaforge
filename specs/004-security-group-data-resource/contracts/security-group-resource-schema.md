# Security Group Resource Schema

**Resource Type**: `zillaforge_security_group`  
**API Mapping**: `POST/PUT/GET/DELETE /api/v1/projects/{project_id}/vps/security-groups`

## Schema Definition

### Resource Attributes

| Attribute | Type | Required | Computed | ForceNew | Sensitive | Validators | Description |
|-----------|------|----------|----------|----------|-----------|------------|-------------|
| `id` | String | No | Yes | No | No | - | Unique identifier (UUID format). Assigned by API during creation. |
| `name` | String | Yes | No | Yes | No | Length 1-255, alphanumeric + hyphens/underscores | Human-readable security group name. Must be unique within project. **Immutable** - changing forces resource replacement. |
| `description` | String | No | Yes | No | No | Max length 500 | Optional description of security group purpose. Empty string if not provided. **Mutable** - can be updated in-place. |
| `ingress_rules` | List[Object] | No | Yes | No | No | - | List of inbound firewall rules. Empty list if not specified (default deny all inbound). **Mutable** - rules can be added/removed. |
| `egress_rules` | List[Object] | No | Yes | No | No | - | List of outbound firewall rules. Empty list if not specified (default deny all outbound). **Mutable** - rules can be added/removed. |

### Nested Attribute: `ingress_rules` / `egress_rules`

Each rule object contains:

| Attribute | Type | Required | Validators | Description |
|-----------|------|----------|------------|-------------|
| `protocol` | String | Yes | Enum: `tcp`, `udp`, `icmp`, `any` (case-insensitive) | Network protocol for the rule. Normalized to lowercase. |
| `port_range` | String | Yes | Regex: `^(all\|[1-9][0-9]{0,4}(-[1-9][0-9]{0,4})?)$` + custom port range validator | Port or port range. Examples: `22`, `80-443`, `all`. For ICMP, `port_range` must be `all`. The literal `all` is equivalent to `1-65535` for TCP/UDP semantics. |
| `source_cidr` | String | Yes (ingress only) | CIDR validator (IPv4/IPv6) | Source IP address range in CIDR notation. Examples: `0.0.0.0/0`, `10.0.0.0/24`, `::/0`. **Ingress rules only**. |
| `destination_cidr` | String | Yes (egress only) | CIDR validator (IPv4/IPv6) | Destination IP address range in CIDR notation. Examples: `0.0.0.0/0`, `192.168.1.0/24`, `2001:db8::/32`. **Egress rules only**. |

## Terraform HCL Example

```hcl
resource "zillaforge_security_group" "web" {
  name        = "web-servers-prod"
  description = "Security group for production web tier"
  
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
  
  # Allow SSH from admin network only
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "203.0.113.0/24"
  }
  
  # Allow all outbound traffic (IPv4)
  egress_rule {
    protocol          = "any"
    port_range        = "all"
    destination_cidr  = "0.0.0.0/0"
  }
  
  # Allow all outbound traffic (IPv6)
  egress_rule {
    protocol          = "any"
    port_range        = "all"
    destination_cidr  = "::/0"
  }
}
```

## Plan Modifiers

| Attribute | Plan Modifier | Reason |
|-----------|---------------|--------|
| `id` | `stringplanmodifier.UseStateForUnknown()` | ID unknown until API returns it; preserve in state once known |
| `name` | `stringplanmodifier.RequiresReplace()` | Name is immutable; changing forces new resource |
| `description` | - | Mutable attribute, normal update |
| `ingress_rules` | - | Mutable list, normal update with diff detection |
| `egress_rules` | - | Mutable list, normal update with diff detection |

## CRUD Operations

### Create

**API Call**: `POST /api/v1/projects/{project_id}/vps/security-groups`

**Request Body**:
```json
{
  "name": "web-servers-prod",
  "description": "Security group for production web tier",
  "ingress_rules": [
    {
      "protocol": "tcp",
      "port_range": "80",
      "source_cidr": "0.0.0.0/0"
    },
    {
      "protocol": "tcp",
      "port_range": "443",
      "source_cidr": "0.0.0.0/0"
    }
  ],
  "egress_rules": [
    {
      "protocol": "any",
      "port_range": "all",
      "destination_cidr": "0.0.0.0/0"
    }
  ]
}
```

**Response** (201 Created):
```json
{
  "id": "sg-12345678-1234-1234-1234-123456789abc",
  "name": "web-servers-prod",
  "description": "Security group for production web tier",
  "ingress_rules": [...],
  "egress_rules": [...],
  "created_at": "2025-12-14T10:30:00Z"
}
```

**Error Handling**:
- 409 Conflict: Security group with same name already exists
- 400 Bad Request: Invalid rule configuration (port range, CIDR, protocol)
- 422 Unprocessable Entity: Quota exceeded

### Read

**API Call**: `GET /api/v1/projects/{project_id}/vps/security-groups/{id}`

**Response** (200 OK):
```json
{
  "id": "sg-12345678-1234-1234-1234-123456789abc",
  "name": "web-servers-prod",
  "description": "Security group for production web tier",
  "ingress_rules": [
    {
      "protocol": "tcp",
      "port_range": "80",
      "source_cidr": "0.0.0.0/0"
    }
  ],
  "egress_rules": [
    {
      "protocol": "any",
      "port_range": "all",
      "destination_cidr": "0.0.0.0/0"
    }
  ],
  "created_at": "2025-12-14T10:30:00Z",
  "updated_at": "2025-12-14T11:00:00Z"
    
      "port_range": "all",
      "destination_cidr": "0.0.0.0/0"
    }

### Update

**API Call**: `PUT /api/v1/projects/{project_id}/vps/security-groups/{id}`

**Request Body** (partial update):
```json
{
  "description": "Updated description",
  "ingress_rules": [
    {
      "protocol": "tcp",
      "port_range": "80",
      "source_cidr": "0.0.0.0/0"
    },
    {
      "protocol": "tcp",
      "port_range": "22",
      "source_cidr": "10.0.0.0/8"
    }
  ]
}
```

**Response** (200 OK): Full security group object with updates applied

**Error Handling**:
- 404 Not Found: Security group deleted outside Terraform
- 400 Bad Request: Invalid rule configuration
- 409 Conflict: Concurrent modification detected

**Note**: Only `description` and rules are updatable. `name` changes trigger `RequiresReplace` (destroy + recreate).

### Delete

**API Call**: `DELETE /api/v1/projects/{project_id}/vps/security-groups/{id}`

**Response** (204 No Content): Successful deletion

**Error Handling**:
- 404 Not Found: Already deleted (idempotent)
- 409 Conflict: Security group attached to active instances
  - SDK returns generic error: `"(neutron)Security Group {id} in use."`
  - Provider must return actionable diagnostic with workaround guidance
  - Example error message:
    ```
    Error: Security Group In Use
    
    Cannot delete security group "web-servers": it is currently in use by one or
    more instances. Please detach the security group from all instances before
    deletion.
    
    To find instances using this security group, check the ZillaForge console or run:
      zillaforge instances list --security-group sg-12345678-1234-1234-1234-123456789abc
    ```

**Pre-Delete Check**:
Per FR-007 (clarification B), provider must verify security group is not attached before attempting deletion:

```go
// Check attachments (if API provides endpoint)
attachments, err := vpsClient.SecurityGroups().GetAttachments(ctx, id)
if len(attachments.InstanceIDs) > 0 {
    return fmt.Errorf("Cannot delete security group %s: attached to instances %v. Detach from instances first.", name, attachments.InstanceIDs)
}
```

### Import

**Command**:
```bash
terraform import zillaforge_security_group.web sg-12345678-1234-1234-1234-123456789abc
```

**Implementation**:
- Parse `id` from import string
- Call Read operation to fetch current state
- Populate all attributes into Terraform state
- Requires user to manually add matching configuration block

**Error Handling**:
- Invalid UUID format: Error with format guidance
- Security group not found: Clear error message

## Validation Rules

### Name Validation
- Pattern: `^[a-zA-Z0-9_-]+$`
- Length: 1-255 characters
- Uniqueness: Enforced by API (409 Conflict on duplicate)

### Description Validation
- Max length: 500 characters
- Optional (empty string if not provided)

### Protocol Validation
### Protocol Validation
- Enum: `tcp`, `udp`, `icmp`, `any`
- Case-insensitive (normalized to lowercase)
- Implemented with `stringvalidator.OneOf("tcp", "udp", "icmp", "any")`

### Port Range Validation

**Regex Pattern**: `^(all|[1-9][0-9]{0,4}(-[1-9][0-9]{0,4})?)$`

**Custom Validator Logic**:
```go
func validatePortRange(protocol, portRange string) error {
  // ICMP must use "all"; "any" indicates all protocols and may use "all" for port_range
  if protocol == "icmp" || protocol == "any" {
        if portRange != "all" {
            return fmt.Errorf("port_range must be 'all' for protocol '%s'", protocol)
        }
        return nil
    }
    
    // TCP/UDP can use single port, range, or all
    if portRange == "all" {
        return nil
    }
    
    // Single port
    if !strings.Contains(portRange, "-") {
        port, _ := strconv.Atoi(portRange)
        if port < 1 || port > 65535 {
            return fmt.Errorf("port must be 1-65535, got %d", port)
        }
        return nil
    }
    
    // Port range
    parts := strings.Split(portRange, "-")
    start, _ := strconv.Atoi(parts[0])
    end, _ := strconv.Atoi(parts[1])
    
    if start < 1 || end > 65535 || start > end {
        return fmt.Errorf("port range must be 1-65535 with start <= end, got %s", portRange)
    }
    
    return nil
}
```

### CIDR Validation

**Validator**:
```go
func validateCIDR(cidr string) error {
    ip, ipNet, err := net.ParseCIDR(cidr)
    if err != nil {
        return fmt.Errorf("invalid CIDR notation: %w", err)
    }
    
    // Warn if host bits are set (e.g., 192.168.1.5/24 should be 192.168.1.0/24)
    if !ip.Equal(ipNet.IP) {
        // Warning diagnostic, not error
    }
    
    return nil
}
```

**Accepted Formats**:
- IPv4: `0.0.0.0/0`, `192.168.1.0/24`, `10.0.0.1/32`
- IPv6: `::/0`, `2001:db8::/32`, `fe80::1/128`

## State Behavior

### ForceNew (Replacement) Triggers
- `name` attribute change

### In-Place Updates
- `description` change
- `ingress_rules` modifications (add/remove/modify rules)
- `egress_rules` modifications (add/remove/modify rules)

### Computed Attributes
- `id`: Set once during create, never changes
- `description`: Computed with default empty string if not provided

### Null vs Empty Handling
- `ingress_rules`: `null` or `[]` both result in empty list (default deny)
- `egress_rules`: `null` or `[]` both result in empty list (default deny)
- `description`: `null` becomes empty string `""`

## Timeouts

**Default Timeouts** (configurable):
```hcl
resource "zillaforge_security_group" "web" {
  # ... attributes ...
  
  timeouts {
    create = "10m"
    update = "10m"
    delete = "5m"
  }
}
```

**Rationale**: Rule propagation may take time; allow users to configure based on their infrastructure scale.

## Examples

### Minimal Security Group
```hcl
resource "zillaforge_security_group" "minimal" {
  name = "minimal-sg"
  # No rules = deny all (secure by default)
}
```

### SSH-Only Access
```hcl
resource "zillaforge_security_group" "ssh" {
  name        = "ssh-access"
  description = "Allow SSH from corporate network"
  
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "198.51.100.0/24"
  }
  
  egress_rule {
    protocol          = "any"
    port_range        = "all"
    destination_cidr  = "0.0.0.0/0"
  }
}
```

### Database Security Group
```hcl
resource "zillaforge_security_group" "postgres" {
  name        = "postgres-db"
  description = "PostgreSQL database access"
  
  ingress_rule {
    protocol    = "tcp"
    port_range  = "5432"
    source_cidr = "10.0.1.0/24"  # App tier subnet
  }
  
  # No egress rules = deny all outbound (database doesn't initiate connections)
}
```

### ICMP Ping Allowed
```hcl
resource "zillaforge_security_group" "pingable" {
  name = "allow-ping"
  
  ingress_rule {
    protocol    = "icmp"
    port_range  = "all"  # Required but ignored for ICMP
    source_cidr = "0.0.0.0/0"
  }
}
```
