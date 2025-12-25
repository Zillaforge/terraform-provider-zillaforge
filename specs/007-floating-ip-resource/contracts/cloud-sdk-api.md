# Cloud-SDK Floating IP API Contract

**Feature**: 007-floating-ip-resource  
**Date**: December 24, 2025  
**Purpose**: Document expected cloud-sdk API interface for floating IP management

---

## Client Access Pattern

```go
import cloudsdk "github.com/Zillaforge/cloud-sdk"

// Initialize project client (provided by provider configuration)
projectClient := cloudsdk.NewProjectClient(...)

// Access VPS floating IP client
vpsClient := projectClient.VPS()
floatingIPClient := vpsClient.FloatingIPs()
```

---

## API Methods

### Create Floating IP

**Method**: `Create(ctx context.Context, req *FloatingIPCreateRequest) (*FloatingIP, error)`

**Purpose**: Allocate a new floating IP from the pool

**Request**:
```go
type FloatingIPCreateRequest struct {
	Name        string `json:"name,omitempty"`        // Optional
	Description string `json:"description,omitempty"` // Optional
}
```

**Response**:
```go
type FloatingIP struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IPAddress   string `json:"ip_address"` // or "address" - verify field name
	Status      string `json:"status"`
	DeviceID    string `json:"device_id,omitempty"` // null/empty when unassociated
}
```

**Example**:
```go
req := &FloatingIPCreateRequest{
	Name:        "web-server-ip",
	Description: "Public IP for web server",
}

floatingIP, err := floatingIPClient.Create(ctx, req)
if err != nil {
	// Handle error (pool exhaustion, quota exceeded, etc.)
	return err
}

// floatingIP.ID -> "fip-uuid-123"
// floatingIP.IPAddress -> "203.0.113.42"
// floatingIP.Status -> "ACTIVE"
```

**Error Scenarios**:
- Pool exhaustion → error message from API
- Quota exceeded → error with current usage and limits
- Network error → connection/timeout errors

---

### Get Floating IP

**Method**: `Get(ctx context.Context, id string) (*FloatingIP, error)`

**Purpose**: Retrieve a specific floating IP by ID

**Parameters**:
- `id`: Floating IP unique identifier (UUID)

**Response**: `*FloatingIP` (same structure as Create)

**Example**:
```go
floatingIP, err := floatingIPClient.Get(ctx, "fip-uuid-123")
if err != nil {
	// Handle error (not found, network error, etc.)
	return err
}

// Access attributes
fmt.Println(floatingIP.IPAddress) // "203.0.113.42"
```

**Error Scenarios**:
- Not found → error with ID reference
- Network error → connection/timeout errors

---

### List Floating IPs

**Method**: `List(ctx context.Context) ([]FloatingIP, error)`

**Purpose**: List all floating IPs in the project

**Parameters**: None

**Response**: `[]FloatingIP`

**Example**:
```go
floatingIPs, err := floatingIPClient.List(ctx)
if err != nil {
	// Handle error
	return err
}

for _, ip := range floatingIPs {
	fmt.Printf("%s: %s (%s)\n", ip.Name, ip.IPAddress, ip.Status)
}
```

**Known Issues**:
- SDK List() filter parameters have known bugs
- **MUST use client-side filtering** (fetch all, filter in-memory)

**Error Scenarios**:
- Network error → connection/timeout errors
- Authorization error → permission denied

---

### Update Floating IP

**Method**: `Update(ctx context.Context, id string, req *FloatingIPUpdateRequest) (*FloatingIP, error)`

**Purpose**: Update modifiable floating IP attributes (name, description)

**Parameters**:
- `id`: Floating IP unique identifier
- `req`: Update request with changed attributes

**Request**:
```go
type FloatingIPUpdateRequest struct {
	Name        *string `json:"name,omitempty"`        // Optional (nil = no change)
	Description *string `json:"description,omitempty"` // Optional (nil = no change)
}
```

**Response**: `*FloatingIP` with updated attributes

**Example**:
```go
newName := "production-web-ip"
newDesc := "Updated description"

req := &FloatingIPUpdateRequest{
	Name:        &newName,
	Description: &newDesc,
}

floatingIP, err := floatingIPClient.Update(ctx, "fip-uuid-123", req)
if err != nil {
	// Handle error
	return err
}

// floatingIP.Name -> "production-web-ip"
```

**Notes**:
- Only name and description are modifiable
- Omit fields that should not change (or use nil pointers)
- Returns full FloatingIP object with all attributes

---

### Delete Floating IP

**Method**: `Delete(ctx context.Context, id string) error`

**Purpose**: Release a floating IP back to the pool

**Parameters**:
- `id`: Floating IP unique identifier

**Response**: `error` (nil on success)

**Example**:
```go
err := floatingIPClient.Delete(ctx, "fip-uuid-123")
if err != nil {
	// Handle error (not found, in use, etc.)
	return err
}

// Floating IP released successfully
```

**Error Scenarios**:
- Not found → error with ID reference
- Network error → connection/timeout errors

**Notes**:
- Association with devices is managed outside Terraform
- Delete proceeds even if device_id is set
- Actual behavior may vary (auto-disassociate or error) - use API response

---

## Data Types

### FloatingIP

**Complete Structure**:
```go
type FloatingIP struct {
	ID          string `json:"id"`                    // UUID format
	Name        string `json:"name"`                  // Optional (may be empty)
	Description string `json:"description"`           // Optional (may be empty)
	IPAddress   string `json:"ip_address"`            // IPv4 address
	Status      string `json:"status"`                // ACTIVE | DOWN | PENDING | REJECTED
	DeviceID    string `json:"device_id,omitempty"`   // UUID or empty when unassociated
}
```

**Field Details**:
- `ID`: Assigned by API, immutable, UUID format
- `Name`: User-provided, optional, modifiable
- `Description`: User-provided, optional, modifiable
- `IPAddress`: Assigned by API from pool, immutable, IPv4 format
- `Status`: Managed by platform, read-only
- `DeviceID`: Managed outside Terraform, read-only, null/empty when unassociated

---

## Status Values

| Value | Meaning | Terraform Handling |
|-------|---------|-------------------|
| `ACTIVE` | Floating IP is operational | Store in state |
| `DOWN` | Floating IP is allocated but not operational | Store in state |
| `PENDING` | Operation in progress | Store in state |
| `REJECTED` | Operation was rejected | Store in state |

**All status values are informational and should be stored in Terraform state. No special error handling is required based on status.**

---

## Error Handling

### Expected Error Types

```go
// Pool exhaustion
err := floatingIPClient.Create(ctx, req)
// err.Error() -> API error message (use as-is per spec clarifications)

// Not found
err := floatingIPClient.Get(ctx, "invalid-id")
// err.Error() -> "floating IP not found: invalid-id"

// Quota exceeded
err := floatingIPClient.Create(ctx, req)
// err.Error() -> "quota exceeded: current=10, maximum=10"
```

### Terraform Provider Handling

```go
// In Create method
floatingIP, err := floatingIPClient.Create(ctx, req)
if err != nil {
	resp.Diagnostics.AddError(
		"Floating IP Allocation Failed",
		fmt.Sprintf("Unable to allocate floating IP: %s", err),
	)
	return
}

// Check for REJECTED status
if floatingIP.Status == "REJECTED" {
	resp.Diagnostics.AddError(
		"Floating IP Rejected",
		fmt.Sprintf("API rejected floating IP allocation (ID: %s)", floatingIP.ID),
	)
	return
}
```

---

## Field Name Verification

**Action Required**: Verify exact field names in cloud-sdk response:

```go
// Check if SDK uses:
// - ip_address, IPAddress, or Address for IP field
// - device_id, DeviceID, or ServerID for device field

// Example verification code:
floatingIP, _ := floatingIPClient.Get(ctx, "test-id")
fmt.Printf("%+v\n", floatingIP) // Print struct to see actual fields
```

**Document actual field names** before implementation to ensure correct mapping.

---

## Testing Endpoints

### Prerequisites

```bash
# Set environment variables for acceptance tests
export ZILLAFORGE_API_URL="https://api.zillaforge.com"
export ZILLAFORGE_API_TOKEN="your-api-token"
export ZILLAFORGE_PROJECT_ID="your-project-id"
```

### Manual API Testing

```bash
# Run acceptance test
make testacc TESTARGS='-run=TestAccFloatingIPResource_basic' PARALLEL=1

# Test data source
make testacc TESTARGS='-run=TestAccFloatingIPDataSource' PARALLEL=1
```

---

## Notes

- All API calls use `context.Context` for timeout/cancellation
- Retry logic for transient failures handled by SDK HTTP client
- Provider should log API calls using `tflog.Debug()`
- Error messages from API should be passed through to user (per spec clarifications)
