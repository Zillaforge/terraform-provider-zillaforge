# Data Model: Security Group Data Source and Resource

**Feature**: 004-security-group-data-resource  
**Phase**: 1 (Design)  
**Date**: 2025-12-14

## Overview

This document defines the data structures for security group management in the Terraform ZillaForge provider. The model supports stateful firewall rules with inbound/outbound traffic control using CIDR-based filtering.

## Core Entities

### Security Group

Represents a named collection of firewall rules that can be applied to VPS instances.

| Attribute | Type | Required | Computed | Description |
|-----------|------|----------|----------|-------------|
| `id` | String | No | Yes | Unique identifier (UUID format, assigned by API) |
| `name` | String | Yes | No | Human-readable name, unique within project. **Immutable** - changing forces replacement |
| `description` | String | No | Yes | Optional description of security group purpose. **Updatable** |
| `ingress_rules` | List[SecurityRule] | No | Yes | Inbound rules allowing traffic TO instances. Empty list if not specified (default deny) |
| `egress_rules` | List[SecurityRule] | No | Yes | Outbound rules allowing traffic FROM instances. Empty list if not specified (default deny) |

**Immutability**:
- `name`: Immutable (requires replacement if changed)
- `id`: Computed, never changes
- `description`: Mutable (can be updated in-place)
- `ingress_rules`, `egress_rules`: Mutable (rules can be added/removed/modified)

**Default Behavior**:
- Empty security group (no rules): Denies all inbound and outbound traffic (FR-025)
- Stateful: Return traffic for established connections automatically allowed (FR-027)

---

### Security Rule

Represents an individual firewall rule within a security group.

| Attribute | Type | Required | Computed | Description |
|-----------|------|----------|----------|-------------|
| `protocol` | String | Yes | No | Protocol type: `tcp`, `udp`, `icmp`, or `any`. Case-insensitive |
| `port_range` | String | Yes | No | Port specification: single port (`22`), range (`80-443`), or `all`. Ignored for ICMP protocol |
| `source_cidr` | String | Yes (ingress) | No | Source CIDR block for ingress rules (e.g., `0.0.0.0/0`, `10.0.1.0/24`, `::/0`). IPv4 or IPv6 |
| `destination_cidr` | String | Yes (egress) | No | Destination CIDR block for egress rules. IPv4 or IPv6 |

**Validation Rules**:
- `protocol`:
  - Must be one of: `tcp`, `udp`, `icmp`, `any`
  - Case-insensitive (normalized to lowercase)
  - ICMP has no granular type/code control (clarification answer)
  
- `port_range`:
  - For TCP/UDP: Valid formats are:
    - Single port: `22`, `443` (1-65535)
    - Port range: `8000-8100` (start ≤ end, both 1-65535)
    - Wildcard: `all`
  - For ICMP: `port_range` must be `all` (ports not applicable). The literal `all` is equivalent to the numeric range `1-65535` for TCP/UDP semantics.

  | `protocol` | String | Yes | No | Protocol type: `tcp`, `udp`, `icmp`, or `any`. Case-insensitive |

  - For ICMP/`any` protocols: For ICMP, `port_range` is ignored and must be `all`. When `port_range` is `all`, it is equivalent to the numeric range `1-65535` for TCP/UDP semantics.
  
- `source_cidr` / `destination_cidr`:
  - Must be valid CIDR notation (validated with `net.ParseCIDR`)
  - IPv4 examples: `0.0.0.0/0` (all), `192.168.1.0/24`, `10.0.0.1/32` (single IP)
  - IPv6 examples: `::/0` (all), `2001:db8::/32`
  - Both IPv4 and IPv6 supported (FR-020)

**Rule Evaluation**:
- Multiple security groups on one instance: Union of all rules (most permissive wins) - FR-026
- Stateful: Matching outbound connection automatically allows return inbound traffic

---

## Terraform Schema Representation

### Resource: `zillaforge_security_group`

```hcl
resource "zillaforge_security_group" "example" {
  name        = "web-servers"
  description = "Security group for web tier"
  
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
  
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "203.0.113.0/24"  # Admin network only
  }
  
  egress_rule {
    protocol          = "any"
    port_range        = "all"
    destination_cidr  = "0.0.0.0/0"
  }
}

# Reference in instance (future capability)
resource "zillaforge_vps_instance" "web" {
  name = "web-1"
  # ... other attributes ...
  security_group_ids = [zillaforge_security_group.example.id]
}
```

### Data Source: `zillaforge_security_groups`

```hcl
# Query by name
data "zillaforge_security_groups" "web" {
  name = "web-servers"
}

# Query by ID
data "zillaforge_security_groups" "specific" {
  id = "sg-12345678-1234-1234-1234-123456789abc"
}

# List all
data "zillaforge_security_groups" "all" {
  # No filters = list all security groups
}

# Access attributes
output "first_group_id" {
  value = data.zillaforge_security_groups.web.security_groups[0].id
}

output "ingress_rules" {
  value = data.zillaforge_security_groups.web.security_groups[0].ingress_rules
}
```

**Data Source Attributes**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Optional filter - query specific security group by ID |
| `name` | String | Optional filter - query security groups by exact name |
| `security_groups` | List[SecurityGroupModel] | Computed - list of matching security groups |

**Filtering Behavior**:
- `id` specified: Returns single security group or error if not found
- `name` specified: Returns all security groups with matching name
- Both `id` and `name`: Error - filters are mutually exclusive
- Neither specified: Returns all security groups in project

---

## State Representation

### Terraform State Structure

```json
{
  "version": 4,
  "terraform_version": "1.6.0",
  "resources": [
    {
      "mode": "managed",
      "type": "zillaforge_security_group",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/zillaforge/zillaforge\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "sg-12345678-1234-1234-1234-123456789abc",
            "name": "web-servers",
            "description": "Security group for web tier",
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
              },
              {
                "protocol": "tcp",
                "port_range": "22",
                "source_cidr": "203.0.113.0/24"
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
        }
      ]
    }
  ]
}
```

---

## Go Model Structures

### Terraform Provider Models

```go
// Resource model
type SecurityGroupResourceModel struct {
    ID           types.String      `tfsdk:"id"`
    Name         types.String      `tfsdk:"name"`
    Description  types.String      `tfsdk:"description"`
    IngressRules []SecurityRuleModel `tfsdk:"ingress_rules"`
    EgressRules  []SecurityRuleModel `tfsdk:"egress_rules"`
}

// Nested rule model
type SecurityRuleModel struct {
    Protocol        types.String `tfsdk:"protocol"`
    PortRange       types.String `tfsdk:"port_range"`
    SourceCIDR      types.String `tfsdk:"source_cidr"`       // For ingress
    DestinationCIDR types.String `tfsdk:"destination_cidr"`  // For egress
}

// Data source model
type SecurityGroupDataSourceModel struct {
    ID             types.String            `tfsdk:"id"`     // Filter
    Name           types.String            `tfsdk:"name"`   // Filter
    SecurityGroups []SecurityGroupModel    `tfsdk:"security_groups"`  // Results
}

// Data source result item model
type SecurityGroupModel struct {
    ID           types.String        `tfsdk:"id"`
    Name         types.String        `tfsdk:"name"`
    Description  types.String        `tfsdk:"description"`
    IngressRules []SecurityRuleModel `tfsdk:"ingress_rules"`
    EgressRules  []SecurityRuleModel `tfsdk:"egress_rules"`
}
```

### Cloud-SDK Models (Expected)

```go
package securitygroupsmodels

// API response model
type SecurityGroup struct {
    ID           string
    Name         string
    Description  string
    IngressRules []SecurityRule
    EgressRules  []SecurityRule
    CreatedAt    time.Time
}

// Rule model
type SecurityRule struct {
    ID              string  // May or may not be present
    Protocol        string  // "tcp", "udp", "icmp", "any"
    PortRange       string  // "22", "80-443", "all"
    SourceCIDR      string  // For ingress rules
    DestinationCIDR string  // For egress rules
}

// Create request
type SecurityGroupCreateRequest struct {
    Name         string
    Description  string
    IngressRules []SecurityRuleSpec
    EgressRules  []SecurityRuleSpec
}

// Update request
type SecurityGroupUpdateRequest struct {
    Description  string
    // Rules may be updated via separate calls or included here
}

// Rule specification for create/update
type SecurityRuleSpec struct {
    Protocol        string
    PortRange       string
    SourceCIDR      string
    DestinationCIDR string
}

// List options
type ListSecurityGroupsOptions struct {
    Name string  // Filter by name
}
```

---

## Relationships

### Entity Relationships

```
SecurityGroup (1) ──────> (0..N) SecurityRule (ingress_rules)
                ├─────> (0..N) SecurityRule (egress_rules)
                │
                └─────> (0..N) VPSInstance (future: attached instances)

VPSInstance (1) ──────> (0..N) SecurityGroup (future: security_group_ids)
```

### State Dependencies

```
Terraform Configuration
  │
  ├─> zillaforge_security_group.web (managed resource)
  │     ├─ id: computed after create
  │     ├─ name: user-defined (immutable)
  │     ├─ description: user-defined (mutable)
  │     ├─ ingress_rules: user-defined (mutable list)
  │     └─ egress_rules: user-defined (mutable list)
  │
  └─> zillaforge_vps_instance.server (future)
        └─ security_group_ids: references zillaforge_security_group.web.id
```

---

## Validation & Constraints

### Attribute Constraints

1. **Name Uniqueness**: Security group names must be unique within a project (enforced by API)

2. **Port Range Constraints**:
   - Single port: Integer 1-65535
   - Range: `start-end` where 1 ≤ start ≤ end ≤ 65535
   - Wildcard: Literal string `all`
  - ICMP: Must use `all` (ports not applicable). The literal `all` for `port_range` is equivalent to `1-65535` when used to represent all TCP/UDP ports.

3. **CIDR Constraints**:
   - Must parse successfully with `net.ParseCIDR()`
   - Can be IPv4 or IPv6
   - Special values: `0.0.0.0/0` (all IPv4), `::/0` (all IPv6)

4. **Protocol Constraints**:
   - Enumeration: `tcp`, `udp`, `icmp`, `all`
   - Case-insensitive input, normalized to lowercase

5. **Rule List Constraints**:
   - Empty lists allowed (secure by default - deny all)
   - No enforced maximum (API may have limits)
   - Duplicate rules allowed (idempotent application)

### Deletion Constraints

Per FR-007 (clarification B):
- Cannot delete security group if attached to active VPS instances
- API must return error listing attached instance IDs
- User must explicitly detach before deletion

---

## Type Conversions

### Terraform ↔ Cloud-SDK

```go
// Convert Terraform model to SDK create request
func toCreateRequest(model SecurityGroupResourceModel) *securitygroupsmodels.SecurityGroupCreateRequest {
    req := &securitygroupsmodels.SecurityGroupCreateRequest{
        Name:        model.Name.ValueString(),
        Description: model.Description.ValueString(),
    }
    
    for _, rule := range model.IngressRules {
        req.IngressRules = append(req.IngressRules, securitygroupsmodels.SecurityRuleSpec{
            Protocol:   rule.Protocol.ValueString(),
            PortRange:  rule.PortRange.ValueString(),
            SourceCIDR: rule.SourceCIDR.ValueString(),
        })
    }
    
    for _, rule := range model.EgressRules {
        req.EgressRules = append(req.EgressRules, securitygroupsmodels.SecurityRuleSpec{
            Protocol:        rule.Protocol.ValueString(),
            PortRange:       rule.PortRange.ValueString(),
            DestinationCIDR: rule.DestinationCIDR.ValueString(),
        })
    }
    
    return req
}

// Convert SDK response to Terraform model
func fromAPIResponse(sg *securitygroupsmodels.SecurityGroup) SecurityGroupResourceModel {
    model := SecurityGroupResourceModel{
        ID:          types.StringValue(sg.ID),
        Name:        types.StringValue(sg.Name),
        Description: types.StringValue(sg.Description),
    }
    
    for _, rule := range sg.IngressRules {
        model.IngressRules = append(model.IngressRules, SecurityRuleModel{
            Protocol:   types.StringValue(rule.Protocol),
            PortRange:  types.StringValue(rule.PortRange),
            SourceCIDR: types.StringValue(rule.SourceCIDR),
        })
    }
    
    for _, rule := range sg.EgressRules {
        model.EgressRules = append(model.EgressRules, SecurityRuleModel{
            Protocol:        types.StringValue(rule.Protocol),
            PortRange:       types.StringValue(rule.PortRange),
            DestinationCIDR: types.StringValue(rule.DestinationCIDR),
        })
    }
    
    return model
}
```

---

## Edge Cases

### Empty Security Group

```hcl
resource "zillaforge_security_group" "isolated" {
  name = "isolated-tier"
  # No rules defined
}
```

**Behavior**: Denies all inbound and outbound traffic (FR-025)

### All-Allow Egress

```hcl
egress_rule {
  protocol         = "any"
  port_range       = "all"
  destination_cidr = "0.0.0.0/0"
}

egress_rule {
  protocol         = "any"
  port_range       = "all"
  destination_cidr = "::/0"
}
```

**Behavior**: Allows all outbound traffic for both IPv4 and IPv6

### ICMP Rules

```hcl
ingress_rule {
  protocol    = "icmp"
  port_range  = "all"  # Required but ignored
  source_cidr = "0.0.0.0/0"
}
```

**Behavior**: Allows all ICMP types (no granular type/code control per clarification)

---

## Summary

- **2 main entities**: SecurityGroup, SecurityRule
- **Nested structure**: Rules embedded in security group (not separate resources)
- **Immutability**: Name is immutable; description and rules are mutable
- **Validation**: Port ranges (1-65535), CIDR blocks (IPv4/IPv6), protocols (tcp/udp/icmp/any)
- **Default deny**: Empty security group blocks all traffic
- **Stateful**: Return traffic auto-allowed
- **Multi-SG evaluation**: Union of rules (most permissive wins)
