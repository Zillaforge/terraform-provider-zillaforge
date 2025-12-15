# Research: Security Group Data Source and Resource

**Feature**: 004-security-group-data-resource  
**Phase**: 0 (Outline & Research)  
**Date**: 2025-12-14

## Purpose

Resolve all NEEDS CLARIFICATION items from Technical Context and research best practices for implementing Terraform security group resources using the Plugin Framework and ZillaForge cloud-sdk.

## Research Tasks

### 1. Cloud-SDK Security Group API Structure

**Question**: What is the exact API structure for security groups in `github.com/Zillaforge/cloud-sdk`?

**Findings**:
Based on analysis of existing code patterns (keypairs, networks, flavors), the expected structure is:

```go
// Import pattern
import (
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    securitygroupsmodels "github.com/Zillaforge/cloud-sdk/models/vps/securitygroups"
)

// Client access chain
vpsClient := projectClient.VPS()
sgClient := vpsClient.SecurityGroups()

// Expected API methods
securityGroup, err := sgClient.Get(ctx, id)
securityGroups, err := sgClient.List(ctx, &securitygroupsmodels.ListOptions{Name: "filter"})
securityGroup, err := sgClient.Create(ctx, &securitygroupsmodels.CreateRequest{...})
securityGroup, err := sgClient.Update(ctx, id, &securitygroupsmodels.UpdateRequest{...})
err := sgClient.Delete(ctx, id)

// Rule management (if separate from main resource)
rule, err := sgClient.CreateRule(ctx, sgID, &securitygroupsmodels.RuleCreateRequest{...})
err := sgClient.DeleteRule(ctx, sgID, ruleID)
```

**Decision**: Use cloud-sdk security group client following established VPS resource patterns. Model structure will be confirmed during implementation (cloud-sdk may not have security group support yet, requiring SDK updates).

**Alternatives Considered**:
- Direct HTTP client: Rejected - violates consistency with existing resources
- Custom SDK wrapper: Rejected - cloud-sdk is the official client

---

### 2. Terraform Plugin Framework Best Practices for Nested Rules

**Question**: How should security group rules be modeled in Terraform schema - nested attributes vs separate resource?

**Findings**:

**Option A: Nested Rules (Recommended)**
```hcl
resource "zillaforge_security_group" "web" {
  name = "web-servers"
  
  ingress_rule {
    protocol  = "tcp"
    port_range = "80"
    source_cidr = "0.0.0.0/0"
  }
  
  ingress_rule {
    protocol  = "tcp"
    port_range = "443"
    source_cidr = "0.0.0.0/0"
  }
  
    egress_rule {
        protocol  = "any"
        port_range = "all"
        destination_cidr = "0.0.0.0/0"
    }
}
```

**Option B: Separate Rule Resource**
```hcl
resource "zillaforge_security_group" "web" {
  name = "web-servers"
}

resource "zillaforge_security_group_rule" "http" {
  security_group_id = zillaforge_security_group.web.id
  direction = "ingress"
  protocol  = "tcp"
  port_range = "80"
  source_cidr = "0.0.0.0/0"
}
```

**Decision**: **Option A - Nested Rules** using `schema.ListNestedAttribute`

**Rationale**:
1. **Atomic Updates**: Security groups with rules are conceptually a single unit; nested approach ensures atomic apply/destroy
2. **Simpler User Experience**: Users define entire security posture in one resource block
3. **State Consistency**: No orphaned rules if security group is destroyed
4. **Matches AWS Pattern**: AWS security groups use inline rules as primary pattern
5. **Plugin Framework Support**: `ListNestedAttribute` with `NestedAttributeObject` natively supports this

**Implementation Pattern**:
```go
"ingress_rules": schema.ListNestedAttribute{
    MarkdownDescription: "List of inbound rules allowing traffic to instances",
    Optional: true,
    Computed: true, // Empty list if not specified
    NestedObject: schema.NestedAttributeObject{
        Attributes: map[string]schema.Attribute{
            "protocol": schema.StringAttribute{
                MarkdownDescription: "Protocol (tcp, udp, icmp, any)",
                Required: true,
            },
            "port_range": schema.StringAttribute{
                MarkdownDescription: "Port or range (22, 80-443, all)",
                Required: true,
            },
            "source_cidr": schema.StringAttribute{
                MarkdownDescription: "Source CIDR block (0.0.0.0/0, 10.0.0.0/16)",
                Required: true,
            },
        },
    },
},
```

**Alternatives Considered**:
- Separate rule resource: More complex dependency management, risk of orphaned rules
- Single string with rule syntax: Poor UX, validation complexity, not idiomatic Terraform

---

### 3. Port Range Validation Strategy

**Question**: How to validate port ranges efficiently (single ports, ranges, "all")?

**Findings**:

**Validator Pattern** (Terraform Plugin Framework validators):
```go
import "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

"port_range": schema.StringAttribute{
    Required: true,
    Validators: []validator.String{
        stringvalidator.RegexMatches(
            regexp.MustCompile(`^(all|[1-9][0-9]{0,4}(-[1-9][0-9]{0,4})?)$`),
            "must be 'all', a single port (1-65535), or range (80-443)",
        ),
        // Custom validator for range logic
        validatePortRange(),
    },
},
```

**Custom Validator Implementation**:
```go
type portRangeValidator struct{}

func (v portRangeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
    if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
        return
    }
    
    value := req.ConfigValue.ValueString()
    if value == "all" {
        return
    }
    
    // Parse single port
    if !strings.Contains(value, "-") {
        port, err := strconv.Atoi(value)
        if err != nil || port < 1 || port > 65535 {
            resp.Diagnostics.AddAttributeError(
                req.Path,
                "Invalid Port",
                fmt.Sprintf("Port must be between 1-65535, got: %s", value),
            )
        }
        return
    }
    
    // Parse range
    parts := strings.Split(value, "-")
    if len(parts) != 2 {
        resp.Diagnostics.AddAttributeError(req.Path, "Invalid Range", "Range must be start-end")
        return
    }
    
    start, err1 := strconv.Atoi(parts[0])
    end, err2 := strconv.Atoi(parts[1])
    
    if err1 != nil || err2 != nil || start < 1 || end > 65535 || start > end {
        resp.Diagnostics.AddAttributeError(
            req.Path,
            "Invalid Port Range",
            fmt.Sprintf("Range must be 1-65535 with start <= end, got: %s", value),
        )
    }
}
```

**Decision**: Use combination of regex validator + custom validator for comprehensive validation

**Alternatives Considered**:
- API-side validation only: Rejected - poor UX (errors during apply instead of plan)
- String parsing in Create/Update: Rejected - doesn't prevent invalid config from being saved

---

### 4. CIDR Validation Best Practices

**Question**: How to validate CIDR blocks for IPv4 and IPv6?

**Findings**:

**Standard Library Approach** (Go net package):
```go
import (
    "net"
    "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
)

type cidrValidator struct {
    allowIPv4 bool
    allowIPv6 bool
}

func (v cidrValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
    if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
        return
    }
    
    cidr := req.ConfigValue.ValueString()
    ip, ipNet, err := net.ParseCIDR(cidr)
    
    if err != nil {
        resp.Diagnostics.AddAttributeError(
            req.Path,
            "Invalid CIDR Notation",
            fmt.Sprintf("Must be valid CIDR (e.g., 10.0.0.0/24, ::/0): %s", err),
        )
        return
    }
    
    // Validate network address (not host address)
    if !ip.Equal(ipNet.IP) {
        resp.Diagnostics.AddAttributeWarning(
            req.Path,
            "CIDR Host Bits Set",
            fmt.Sprintf("CIDR %s has host bits set, network is %s", cidr, ipNet.String()),
        )
    }
    
    // Check IPv4/IPv6
    isIPv4 := ip.To4() != nil
    if !v.allowIPv4 && isIPv4 {
        resp.Diagnostics.AddAttributeError(req.Path, "IPv4 Not Allowed", "Only IPv6 CIDRs permitted")
    }
    if !v.allowIPv6 && !isIPv4 {
        resp.Diagnostics.AddAttributeError(req.Path, "IPv6 Not Allowed", "Only IPv4 CIDRs permitted")
    }
}
```

**Decision**: Use `net.ParseCIDR` with custom validator supporting both IPv4 and IPv6

**Rationale**:
- Standard library is battle-tested for CIDR parsing
- Catches common errors (e.g., `192.168.1.1/24` should be `192.168.1.0/24`)
- Provides clear error messages
- Supports both protocol versions as required by FR-020

**Alternatives Considered**:
- Regex validation: Rejected - regex for CIDR is complex and error-prone
- Third-party library: Rejected - unnecessary dependency when stdlib sufficient

---

### 5. Stateful Firewall Implementation Pattern

**Question**: How to communicate stateful behavior in schema/documentation?

**Findings**:

**Documentation Strategy**:
- Mark in resource-level description: "Security groups are stateful - return traffic for established connections is automatically allowed."
- Explain in ingress/egress rule descriptions: No need for explicit response rules
- Provide examples showing typical patterns (SSH, HTTP/HTTPS with egress-all)

**Schema Documentation**:
```go
resp.Schema = schema.Schema{
    MarkdownDescription: `Manages a security group with stateful firewall rules for VPS instances.

**Stateful Behavior**: Security groups automatically allow return traffic for established connections. 
For example, allowing inbound TCP port 22 (SSH) automatically permits the SSH server to send 
responses back to the client without requiring an explicit outbound rule.

**Default Behavior**: Security groups with no rules deny all traffic (secure by default).`,
    
    Attributes: map[string]schema.Attribute{
        "ingress_rules": schema.ListNestedAttribute{
            MarkdownDescription: `Inbound rules controlling traffic TO instances. Due to stateful 
behavior, response traffic is automatically allowed.`,
            // ...
        },
    },
}
```

**Implementation Note**: Statefulness is handled by the ZillaForge platform (not Terraform). Provider only needs to document behavior clearly.

**Decision**: Document stateful behavior prominently in schema descriptions and quickstart guide

---

### 6. Security Group Attachment Pattern

**Question**: How should security groups be attached to VPS instances?

**Findings**:

Based on existing provider structure and common Terraform patterns:

**Option A: Instance Resource References Security Group IDs** (Recommended)
```hcl
resource "zillaforge_security_group" "web" {
  name = "web-servers"
  # ... rules ...
}

resource "zillaforge_vps_instance" "server" {
  name = "web-1"
  flavor_id = data.zillaforge_flavors.small.flavors[0].id
  security_group_ids = [zillaforge_security_group.web.id]
}
```

**Option B: Separate Attachment Resource**
```hcl
resource "zillaforge_security_group_attachment" "web_to_server" {
  security_group_id = zillaforge_security_group.web.id
  instance_id = zillaforge_vps_instance.server.id
}
```

**Decision**: **Option A** - instance resource has `security_group_ids` attribute

**Rationale**:
- Matches AWS EC2 pattern (`vpc_security_group_ids`)
- Simpler dependency graph (no third resource)
- Instance "owns" its security posture
- Better state consistency

**Implementation Impact**: 
- This feature only implements security group resource/data source
- VPS instance resource (future work) will reference security group IDs
- Security group resource does NOT need attachment logic
- Deletion check (FR-007) requires querying which instances use the security group

---

### 7. Rule Change Detection Strategy

**Question**: How to efficiently detect rule changes for update operations?

**Findings**:

**Terraform Plugin Framework Approach**:
Plugin Framework automatically handles change detection through state comparison:

```go
func (r *SecurityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan, state SecurityGroupResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    
    // Framework already detected changes by comparing plan vs state
    // Just apply the planned changes to API
    
    // Update basic attributes if changed
    if !plan.Description.Equal(state.Description) {
        // Update description via API
    }
    
    // Handle rules - framework detects list changes automatically
    // Option 1: Full replacement (simple, may cause brief downtime)
    if !reflect.DeepEqual(plan.IngressRules, state.IngressRules) {
        // Delete all old rules, create all new rules
    }
    
    // Option 2: Differential update (complex, zero downtime)
    toAdd, toRemove := diffRules(state.IngressRules, plan.IngressRules)
    for _, rule := range toRemove {
        sgClient.DeleteRule(ctx, state.ID.ValueString(), rule.ID)
    }
    for _, rule := range toAdd {
        sgClient.CreateRule(ctx, state.ID.ValueString(), rule)
    }
}
```

**Decision**: Start with **Option 1 (Full Replacement)** for simplicity; optimize to differential updates if performance issues arise

**Rationale**:
- Plugin Framework handles state comparison automatically
- Full replacement is simpler to implement and test
- Brief rule update window acceptable for initial version
- Can optimize later based on user feedback

---

### 8. Deletion Safety Pattern

**Question**: How to implement "block deletion if attached" (FR-007, clarification B)?

**Findings**:

**Implementation Pattern**:

Option A — SDK provides attachments API (preferred):

```go
// Pre-delete attachment check (when SDK supports it)
attachments, err := vpsClient.SecurityGroups().GetAttachments(ctx, state.ID.ValueString())
if err != nil {
    resp.Diagnostics.AddError("Delete Check Failed", fmt.Sprintf("Unable to verify attachments: %s", err))
    return
}
if len(attachments.InstanceIDs) > 0 {
    resp.Diagnostics.AddError("Security Group In Use", fmt.Sprintf("Cannot delete security group %s: attached to instances %v. Detach before deletion.", state.Name.ValueString(), attachments.InstanceIDs))
    return
}
// Safe to delete
err = vpsClient.SecurityGroups().Delete(ctx, state.ID.ValueString())
if err != nil {
    resp.Diagnostics.AddError("Delete Failed", fmt.Sprintf("Unable to delete: %s", err))
    return
}
```

Option B — SDK does NOT provide attachments API (current reality):

```go
// Attempt delete and handle 409 Conflict if attached
err := vpsClient.SecurityGroups().Delete(ctx, state.ID.ValueString())
if err != nil {
    if apiErr, ok := detectAPIError(err); ok && apiErr.StatusCode == 409 {
        // SDK only returns generic "(neutron)Security Group {id} in use." message
        // Cannot extract specific instance IDs from error response
        resp.Diagnostics.AddError(
            "Security Group In Use",
            fmt.Sprintf(
                "Cannot delete security group %s: it is currently in use by one or more instances. "+
                "Please detach the security group from all instances before deletion.\n\n"+
                "To find instances using this security group, check the ZillaForge console or run:\n"+
                "  zillaforge instances list --security-group %s",
                state.Name.ValueString(),
                state.ID.ValueString(),
            ),
        )
        return
    }

    resp.Diagnostics.AddError("Delete Failed", fmt.Sprintf("Unable to delete: %s", err))
    return
}
```

**Fallback**: If `Instances().List()` exposes security group attachments, provider may list instances and filter by security group ID as a last-resort pre-check (be mindful of pagination and performance).

**Current SDK Behavior**: SDK returns 409 with generic error message `"(neutron)Security Group {id} in use."` without instance details.

**API/SDK Recommendation**: Ideally the `cloud-sdk` should provide either:

- a `GetAttachments(projectID, sgID)` convenience method returning instance IDs, or
- ensure DELETE conflict responses (409) include a machine-readable list of attached instance IDs in the response body (JSON field `attached_instances`).

**Workaround**: Error message includes CLI command suggestion for users to manually identify attached instances.

**Decision**: Implement Option B (attempt delete and handle 409) for initial implementation because the SDK currently lacks an attachments API. Revisit and switch to Option A when the SDK adds attachment support.

**Alternatives Considered**:
- Terraform force flag: Rejected — not appropriate for provider behavior
- Allow deletion silently: Rejected — violates FR-007 (clarification B)

---

## Summary

All research tasks completed. Key decisions:

1. **API Structure**: Use cloud-sdk `securitygroups` client following established patterns
2. **Rule Modeling**: Nested `ListNestedAttribute` for ingress/egress rules (not separate resource)
3. **Port Validation**: Regex + custom validator supporting single ports, ranges, and "all"
4. **CIDR Validation**: `net.ParseCIDR` with custom validator for IPv4/IPv6
5. **Stateful Behavior**: Document in schema; platform handles implementation
6. **Instance Attachment**: Instance resources reference security group IDs (future work)
7. **Rule Updates**: Full replacement initially; optimize to differential if needed
8. **Deletion Safety**: Pre-check attachments or rely on API error

**No NEEDS CLARIFICATION items remain.** Ready for Phase 1 design.
