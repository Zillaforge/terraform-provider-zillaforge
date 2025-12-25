# Data Model: Floating IP Resource and Data Source

**Feature**: 007-floating-ip-resource  
**Date**: December 24, 2025  
**Purpose**: Define Go struct models for Terraform state management

---

## Model Package Structure

All models are defined in `internal/vps/model/floating_ip.go` to maintain consistency with existing patterns (server, network models).

---

## Resource Model

### FloatingIPResourceModel

**Purpose**: Represents Terraform state for `zillaforge_floating_ip` resource

**File**: `internal/vps/model/floating_ip.go`

```go
// FloatingIPResourceModel represents the Terraform state for a floating IP resource.
type FloatingIPResourceModel struct {
	// Optional user-provided attributes
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	// Computed attributes (read-only)
	ID        types.String `tfsdk:"id"`
	IPAddress types.String `tfsdk:"ip_address"`
	Status    types.String `tfsdk:"status"`
	DeviceID  types.String `tfsdk:"device_id"`
}
```

**Attribute Details**:

| Attribute | Type | Required | Computed | Sensitive | Plan Modifier | Description |
|-----------|------|----------|----------|-----------|---------------|-------------|
| `name` | string | No | Yes | No | None | Optional human-readable name (updatable) |
| `description` | string | No | Yes | No | None | Optional description (updatable) |
| `id` | string | No | Yes | No | UseStateForUnknown | Unique identifier (UUID) |
| `ip_address` | string | No | Yes | No | UseStateForUnknown | Public IP address (e.g., "203.0.113.42") |
| `status` | string | No | Yes | No | UseStateForUnknown | Status: ACTIVE, DOWN, PENDING, REJECTED |
| `device_id` | string | No | Yes | No | UseStateForUnknown | Associated device ID (null/empty when unassociated) |

**State Transitions**:
- **Create**: User provides optional name/description → API returns all computed attributes
- **Update**: User modifies name or description → API updates, returns new state
- **Read**: Fetch current state from API, update all computed attributes
- **Delete**: Remove from API, clear from state

---

## Data Source Models

### FloatingIPDataSourceModel

**Purpose**: Represents configuration and results for `zillaforge_floating_ips` data source

**File**: `internal/vps/model/floating_ip.go`

```go
// FloatingIPDataSourceModel describes the data source config and results.
type FloatingIPDataSourceModel struct {
	// Optional filters (all are optional, AND logic when multiple specified)
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	IPAddress types.String `tfsdk:"ip_address"`
	Status    types.String `tfsdk:"status"`

	// Computed results (list of matching floating IPs)
	FloatingIPs []FloatingIPModel `tfsdk:"floating_ips"`
}
```

**Filter Attributes**:

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | No | Filter by exact floating IP ID |
| `name` | string | No | Filter by exact name (case-sensitive) |
| `ip_address` | string | No | Filter by exact IP address |
| `status` | string | No | Filter by status (ACTIVE, DOWN, PENDING, REJECTED) |
| `floating_ips` | list | Computed | List of matching floating IPs (empty if no matches) |

**Filter Logic**:
- All filters are optional
- Multiple filters use AND logic (all must match)
- Empty filter set returns all floating IPs
- No matches returns empty list (not an error)

### FloatingIPModel

**Purpose**: Represents a single floating IP in data source results (shared model)

**File**: `internal/vps/model/floating_ip.go`

```go
// FloatingIPModel represents a single floating IP in data source results.
type FloatingIPModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IPAddress   types.String `tfsdk:"ip_address"`
	Status      types.String `tfsdk:"status"`
	DeviceID    types.String `tfsdk:"device_id"`
}
```

**Result Attributes**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Unique identifier (UUID) |
| `name` | string | Optional human-readable name (may be empty) |
| `description` | string | Optional description (may be empty) |
| `ip_address` | string | Public IP address |
| `status` | string | Status: ACTIVE, DOWN, PENDING, REJECTED |
| `device_id` | string | Associated device ID (empty when unassociated) |

**Sorting**: Results are sorted deterministically by ID (ascending) for consistent Terraform state.

---

## SDK Integration Models

### SDK Request Models

**Purpose**: Request structures for API calls (expected SDK types)

```go
// These types are expected to be provided by cloud-sdk
// (documented here for reference, not implemented in provider)

type FloatingIPCreateRequest struct {
	Name        string // Optional
	Description string // Optional
}

type FloatingIPUpdateRequest struct {
	Name        *string // Optional (null means no change)
	Description *string // Optional (null means no change)
}
```

### SDK Response Model

**Purpose**: Response structure from API (expected SDK type)

```go
// This type is expected to be provided by cloud-sdk
// (documented here for reference, not implemented in provider)

type FloatingIP struct {
	ID          string
	Name        string
	Description string
	IPAddress   string // Field name in SDK (may be "IP", "Address", or "IPAddress")
	Status      string
	DeviceID    string // May be null/empty when unassociated
}
```

---

## Type Conversion Functions

### SDK to Terraform Model

**Purpose**: Convert SDK response to Terraform state

```go
// mapFloatingIPToResourceModel converts SDK FloatingIP to resource model.
func mapFloatingIPToResourceModel(ip *FloatingIP) FloatingIPResourceModel {
	return FloatingIPResourceModel{
		ID:          types.StringValue(ip.ID),
		Name:        types.StringPointerValue(stringPointerOrNull(ip.Name)),
		Description: types.StringPointerValue(stringPointerOrNull(ip.Description)),
		IPAddress:   types.StringValue(ip.IPAddress),
		Status:      types.StringValue(ip.Status),
		DeviceID:    types.StringPointerValue(stringPointerOrNull(ip.DeviceID)),
	}
}

// mapFloatingIPToDataModel converts SDK FloatingIP to data source result model.
func mapFloatingIPToDataModel(ip *FloatingIP) FloatingIPModel {
	return FloatingIPModel{
		ID:          types.StringValue(ip.ID),
		Name:        types.StringPointerValue(stringPointerOrNull(ip.Name)),
		Description: types.StringPointerValue(stringPointerOrNull(ip.Description)),
		IPAddress:   types.StringValue(ip.IPAddress),
		Status:      types.StringValue(ip.Status),
		DeviceID:    types.StringPointerValue(stringPointerOrNull(ip.DeviceID)),
	}
}

// stringPointerOrNull returns nil for empty strings (converts to types.StringNull)
func stringPointerOrNull(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
```

### Terraform Model to SDK Request

**Purpose**: Convert Terraform plan to SDK request

```go
// buildCreateRequest creates SDK request from Terraform plan.
func buildCreateRequest(plan FloatingIPResourceModel) *FloatingIPCreateRequest {
	req := &FloatingIPCreateRequest{}
	
	if !plan.Name.IsNull() && plan.Name.ValueString() != "" {
		req.Name = plan.Name.ValueString()
	}
	
	if !plan.Description.IsNull() && plan.Description.ValueString() != "" {
		req.Description = plan.Description.ValueString()
	}
	
	return req
}

// buildUpdateRequest creates SDK update request from plan and state.
func buildUpdateRequest(plan, state FloatingIPResourceModel) *FloatingIPUpdateRequest {
	req := &FloatingIPUpdateRequest{}
	
	// Only include changed attributes
	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		req.Name = &name
	}
	
	if !plan.Description.Equal(state.Description) {
		desc := plan.Description.ValueString()
		req.Description = &desc
	}
	
	return req
}
```

---

## Client-Side Filtering Logic

### Filter Matching

**Purpose**: Client-side filtering for data source (due to SDK List() bugs)

```go
// filterFloatingIPs applies client-side filters to floating IP list.
func filterFloatingIPs(allIPs []FloatingIP, filters FloatingIPDataSourceModel) []FloatingIPModel {
	var results []FloatingIPModel
	
	for _, ip := range allIPs {
		if matchesFilters(ip, filters) {
			results = append(results, mapFloatingIPToDataModel(&ip))
		}
	}
	
	// Sort by ID for deterministic ordering
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID.ValueString() < results[j].ID.ValueString()
	})
	
	return results
}

// matchesFilters checks if floating IP matches all provided filters (AND logic).
func matchesFilters(ip FloatingIP, filters FloatingIPDataSourceModel) bool {
	// ID filter
	if !filters.ID.IsNull() && filters.ID.ValueString() != "" {
		if ip.ID != filters.ID.ValueString() {
			return false
		}
	}
	
	// Name filter
	if !filters.Name.IsNull() && filters.Name.ValueString() != "" {
		if ip.Name != filters.Name.ValueString() {
			return false
		}
	}
	
	// IP Address filter
	if !filters.IPAddress.IsNull() && filters.IPAddress.ValueString() != "" {
		if ip.IPAddress != filters.IPAddress.ValueString() {
			return false
		}
	}
	
	// Status filter
	if !filters.Status.IsNull() && filters.Status.ValueString() != "" {
		if ip.Status != filters.Status.ValueString() {
			return false
		}
	}
	
	return true
}
```

---

## Null/Empty Handling

### device_id Nullability

**Specification**: device_id is null/empty when floating IP is not associated with any device

**Implementation**:
```go
// When mapping from SDK to Terraform:
DeviceID: types.StringPointerValue(stringPointerOrNull(ip.DeviceID))

// Helper converts empty string to nil pointer → types.StringNull()
func stringPointerOrNull(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
```

**State Representation**:
- Unassociated: `device_id = null` in state
- Associated: `device_id = "device-uuid-123"` in state

---

## Validation Logic

### Status Value Validation

**Valid Values**: ACTIVE, DOWN, PENDING, REJECTED

**Implementation**:
```go
// No schema-level validation needed - status is computed
// Validation occurs in business logic:

func validateStatus(status string) error {
	validStatuses := map[string]bool{
		"ACTIVE":   true,
		"DOWN":     true,
		"PENDING":  true,
		"REJECTED": true,
	}
	
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s (expected ACTIVE, DOWN, PENDING, or REJECTED)", status)
	}
	
	return nil
}
```

---

## Summary

**Model Files**:
- `internal/vps/model/floating_ip.go` - All model structs

**Key Decisions**:
1. Separate models for resource and data source (different attribute sets)
2. Shared `FloatingIPModel` for data source results
3. Client-side filtering due to SDK List() bugs
4. Deterministic sorting by ID for consistent state
5. Null handling for optional attributes (name, description, device_id)
6. No schema validators for status (computed attribute)

**Next Phase**: Create API contracts and quickstart documentation
