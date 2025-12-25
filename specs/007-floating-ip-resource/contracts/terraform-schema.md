# Terraform Schema Contract

**Feature**: 007-floating-ip-resource  
**Date**: December 24, 2025  
**Purpose**: Define Terraform resource and data source schemas

---

## Resource: zillaforge_floating_ip

### Schema Definition

```hcl
resource "zillaforge_floating_ip" "example" {
  # Optional attributes
  name        = "web-server-ip"              # string, optional
  description = "Public IP for web server"   # string, optional

  # Computed attributes (read-only, shown in output)
  # id          = "fip-uuid-123"             # string, computed
  # ip_address  = "203.0.113.42"             # string, computed
  # status      = "ACTIVE"                    # string, computed
  # device_id   = "device-uuid-456"           # string, computed (null when unassociated)
}
```

### Attribute Reference

| Attribute | Type | Required | Computed | Updateable | Description |
|-----------|------|----------|----------|------------|-------------|
| `name` | string | No | Yes | **Yes** | Optional human-readable name for the floating IP |
| `description` | string | No | Yes | **Yes** | Optional description for documentation |
| `id` | string | No | Yes | No | Unique identifier (UUID format) assigned by API |
| `ip_address` | string | No | Yes | No | Public IPv4 address allocated from pool |
| `status` | string | No | Yes | No | Current status (ACTIVE, DOWN, PENDING, REJECTED) |
| `device_id` | string | No | Yes | No | Associated device ID (null when unassociated) |

### Import

```bash
# Import by floating IP ID
terraform import zillaforge_floating_ip.example fip-uuid-123
```

### Example Usage

**Basic allocation**:
```hcl
resource "zillaforge_floating_ip" "basic" {
  # No required attributes - allocates from default pool
}

output "floating_ip_address" {
  value = zillaforge_floating_ip.basic.ip_address
}
```

**With name and description**:
```hcl
resource "zillaforge_floating_ip" "named" {
  name        = "production-api-ip"
  description = "Floating IP for production API gateway"
}
```

**Update name/description**:
```hcl
resource "zillaforge_floating_ip" "updateable" {
  name        = "updated-name"         # Can be changed in-place
  description = "updated description"  # Can be changed in-place
}
```

---

## Data Source: zillaforge_floating_ips

### Schema Definition

```hcl
data "zillaforge_floating_ips" "example" {
  # Optional filters (all optional, AND logic)
  id         = "fip-uuid-123"        # string, optional
  name       = "web-server-ip"       # string, optional
  ip_address = "203.0.113.42"        # string, optional
  status     = "ACTIVE"              # string, optional
  
  # Computed result list
  # floating_ips = [...]              # list of objects, computed
}
```

### Attribute Reference

**Filter Attributes** (all optional):

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Filter by exact floating IP ID |
| `name` | string | Filter by exact name (case-sensitive) |
| `ip_address` | string | Filter by exact IP address |
| `status` | string | Filter by status (ACTIVE, DOWN, PENDING, REJECTED) |

**Result Attribute**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `floating_ips` | list(object) | List of matching floating IPs (empty if no matches) |

**floating_ips Object Structure**:
```hcl
floating_ips = [
  {
    id          = "fip-uuid-123"
    name        = "web-server-ip"
    description = "Public IP for web server"
    ip_address  = "203.0.113.42"
    status      = "ACTIVE"
    device_id   = "device-uuid-456"  # null when unassociated
  },
  # ... more floating IPs
]
```

### Example Usage

**List all floating IPs**:
```hcl
data "zillaforge_floating_ips" "all" {
  # No filters - returns all floating IPs
}

output "all_ips" {
  value = data.zillaforge_floating_ips.all.floating_ips
}
```

**Filter by ID**:
```hcl
data "zillaforge_floating_ips" "by_id" {
  id = "fip-uuid-123"
}

# Access first (and only) result
output "found_ip" {
  value = data.zillaforge_floating_ips.by_id.floating_ips[0].ip_address
}
```

**Filter by name**:
```hcl
data "zillaforge_floating_ips" "by_name" {
  name = "production-api-ip"
}

output "matching_ips" {
  value = [
    for ip in data.zillaforge_floating_ips.by_name.floating_ips :
    ip.ip_address
  ]
}
```

**Filter by IP address**:
```hcl
data "zillaforge_floating_ips" "by_ip" {
  ip_address = "203.0.113.42"
}

# Returns list (even for single match)
locals {
  found = length(data.zillaforge_floating_ips.by_ip.floating_ips) > 0
}
```

**Filter by status**:
```hcl
data "zillaforge_floating_ips" "active_only" {
  status = "ACTIVE"
}

output "active_ips" {
  value = data.zillaforge_floating_ips.active_only.floating_ips
}
```

**Multiple filters (AND logic)**:
```hcl
data "zillaforge_floating_ips" "active_named" {
  name   = "production-api-ip"
  status = "ACTIVE"
}

# Returns only floating IPs matching BOTH filters
```

---

## Schema Validation Rules

### Resource Schema

**name**:
- Type: string
- Optional + Computed
- Updatable (in-place)
- No format restrictions (validated by API)

**description**:
- Type: string
- Optional + Computed
- Updatable (in-place)
- No length restrictions (validated by API)

**id**:
- Type: string
- Computed only
- Plan Modifier: `UseStateForUnknown()`

**ip_address**:
- Type: string
- Computed only
- Plan Modifier: `UseStateForUnknown()`

**status**:
- Type: string
- Computed only
- Plan Modifier: `UseStateForUnknown()`
- Values: ACTIVE, DOWN, PENDING, REJECTED

**device_id**:
- Type: string
- Computed only
- Plan Modifier: `UseStateForUnknown()`
- Nullable (null when unassociated)

### Data Source Schema

**Filter attributes**:
- All filters are Optional
- Type: string
- No validators (exact match filtering)

**floating_ips**:
- Type: ListNestedAttribute
- Computed only
- Contains objects with same structure as resource computed attributes

---

## Behavior Specifications

### Resource Lifecycle

**Create**:
```terraform
# terraform apply
# → API: Create(name, description)
# → Response: FloatingIP with id, ip_address, status, device_id
# → State: All attributes saved
```

**Read**:
```terraform
# terraform refresh
# → API: Get(id)
# → Response: FloatingIP with current state
# → State: All computed attributes updated
```

**Update**:
```terraform
# terraform apply (after changing name or description)
# → API: Update(id, {name, description})
# → Response: FloatingIP with updated state
# → State: Updated in-place (no replacement)
```

**Delete**:
```terraform
# terraform destroy
# → API: Delete(id)
# → State: Resource removed
# → Note: Succeeds even if device_id is set (association managed outside TF)
```

**Import**:
```terraform
# terraform import zillaforge_floating_ip.example fip-uuid-123
# → API: Get(fip-uuid-123)
# → State: All attributes populated from API response
```

### Data Source Behavior

**No filters**:
```terraform
# terraform plan/apply
# → API: List()
# → Filter: None (returns all)
# → Result: All floating IPs sorted by ID
```

**Single filter**:
```terraform
# id = "fip-123"
# → API: List()
# → Filter: Client-side match on id
# → Result: List with 0 or 1 items
```

**Multiple filters**:
```terraform
# name = "prod-ip", status = "ACTIVE"
# → API: List()
# → Filter: Client-side match on name AND status
# → Result: List with matching items
```

**No matches**:
```terraform
# name = "nonexistent"
# → API: List()
# → Filter: Client-side match on name
# → Result: Empty list [] (NOT an error)
```

---

## Error Handling

### Resource Errors

**Pool exhaustion**:
```
Error: Floating IP Allocation Failed

Unable to allocate floating IP: [API error message]
```

**Quota exceeded**:
```
Error: Floating IP Allocation Failed

Unable to allocate floating IP: quota exceeded: current=10, maximum=10
```

**Not found (on read/update/delete)**:
```
Error: Floating IP Not Found

Floating IP with ID 'fip-123' not found: [API error]
```

### Data Source Errors

**API failure**:
```
Error: Unable to Read Floating IPs

Failed to list floating IPs: [API error message]
```

**No other errors** - empty results return empty list, not error

---

## MarkdownDescription Requirements

All attributes MUST have MarkdownDescription per constitution. Examples:

```go
"name": schema.StringAttribute{
	MarkdownDescription: "Optional human-readable name for the floating IP. " +
		"Can be updated in-place without recreating the resource.",
	Optional: true,
	Computed: true,
},

"device_id": schema.StringAttribute{
	MarkdownDescription: "Device ID of the associated VPS instance or other device. " +
		"Null/empty when the floating IP is not associated with any device. " +
		"Association is managed outside of Terraform.",
	Computed: true,
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.UseStateForUnknown(),
	},
},
```

---

## Documentation Generation

All documentation is generated by `tfplugindocs`:

```bash
# Generate provider documentation
make generate

# Output files:
# docs/resources/floating_ip.md
# docs/data-sources/floating_ips.md
```

**Manual documentation NOT ALLOWED** per constitution.

---

## Notes

- Schema follows Terraform Plugin Framework conventions
- All computed attributes use `UseStateForUnknown()` plan modifier
- Data source always returns list (even for single-item queries)
- Empty filter set returns all floating IPs
- Client-side filtering due to SDK List() filter bugs
- Association (device_id) is read-only, managed outside Terraform
