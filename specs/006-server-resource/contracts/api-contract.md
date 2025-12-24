# API Contract: Server Resource

**API Client**: `github.com/Zillaforge/cloud-sdk`  
**Module**: VPS Servers  
**Version**: v0.0.0-20251209081935-79e26e215136

---

## Client Access Pattern

```go
import (
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    servermodels "github.com/Zillaforge/cloud-sdk/models/vps/servers"
)

// Initialize client
projectClient := cloudsdk.NewProjectClient(apiEndpoint, apiToken, projectID)

// Access VPS server client
vpsClient := projectClient.VPS()
serverClient := vpsClient.Servers()

// Use CRUD methods
server, err := serverClient.Create(ctx, createRequest)
server, err := serverClient.Get(ctx, serverID)
servers, err := serverClient.List(ctx, listOptions)
server, err := serverClient.Update(ctx, serverID, updateRequest)
err := serverClient.Delete(ctx, serverID)
```

---

## API Methods

### Create Server

**Method**: `Create(ctx context.Context, req *CreateRequest) (*Server, error)`

**Request Model**:
```go
type CreateRequest struct {
    Name             string            `json:"name"`
    Description      string            `json:"description,omitempty"`
    FlavorID         string            `json:"flavor_id"`
    ImageID          string            `json:"image_id"`
    NetworkPorts     []NetworkPortSpec `json:"network_ports"`
    KeypairID        string            `json:"keypair_id,omitempty"`
    Password         string            `json:"password,omitempty"` // Base64 encoded
    UserData         string            `json:"user_data,omitempty"`
}

type NetworkPortSpec struct {
    NetworkID string   `json:"network_id"`
    IPAddress string   `json:"ip_address,omitempty"`
    IsPrimary bool     `json:"is_primary"`
    SGIDs     []string `json:"sg_ids,omitempty"` // Security group IDs to apply to this NIC
}
```

**Response Model**:
```go
type Server struct {
    ID               string        `json:"id"`
    Name             string        `json:"name"`
    Description      string        `json:"description"`
    FlavorID         string        `json:"flavor_id"`
    ImageID          string        `json:"image_id"`
    Status           string        `json:"status"`
    NetworkPorts     []NetworkPort `json:"network_ports"`
    KeypairID        string        `json:"keypair_id"`
    UserData         string        `json:"user_data,omitempty"`
    IPAddresses      []string      `json:"ip_addresses"`
    CreatedAt        time.Time     `json:"created_at"`
}
```

**HTTP Details**:
- **Endpoint**: `POST /v1/projects/{project_id}/vps/servers`
- **Success Code**: `201 Created`
- **Error Codes**:
  - `400 Bad Request`: Validation error (invalid name, missing required fields)
  - `404 Not Found`: Referenced resource not found (flavor, image, network, security group)
  - `409 Conflict`: Name already exists
  - `422 Unprocessable Entity`: Quota exceeded
  - `500 Internal Server Error`: Platform error

**Example Usage**:
```go
req := &servermodels.CreateRequest{
    Name:     "web-server-01",
    FlavorID: "flavor-small",
    ImageID:  "ubuntu-22.04",
    NetworkPorts: []servermodels.NetworkPortSpec{
        {
            NetworkID:        "net-default",
            IsPrimary:        true,
            SecurityGroupIDs: []string{"sg-default"},
        },
    },
}

server, err := serverClient.Create(ctx, req)
if err != nil {
    // Handle error
}

// server.Status == "building"
// Need to poll for "active" status
```

---

### Get Server

**Method**: `Get(ctx context.Context, id string) (*Server, error)`

**Parameters**:
- `id` (string): Server UUID (e.g., "srv-abc123def456")

**Response Model**: `*Server` (same as Create response)

**HTTP Details**:
- **Endpoint**: `GET /v1/projects/{project_id}/vps/servers/{id}`
- **Success Code**: `200 OK`
- **Error Codes**:
  - `404 Not Found`: Server does not exist

**Example Usage**:
```go
server, err := serverClient.Get(ctx, "srv-abc123def456")
if err != nil {
    // Handle error (404 = not found)
}

// server.Status could be: "building", "active", "error", "deleted"
```

**Special Behavior**:
- `user_data` field is **not returned** by GET requests for security reasons (will be empty string)
- Use for status polling during create operation
- Use for import functionality

---

### List Servers

**Method**: `List(ctx context.Context, opts *ListOptions) ([]Server, error)`

**Request Model**:
```go
type ListOptions struct {
    Name   string `json:"name,omitempty"`   // Filter by name (substring)
    Status string `json:"status,omitempty"` // Filter by status
}
```

**Response**: `[]Server` (array of Server models)

**HTTP Details**:
- **Endpoint**: `GET /v1/projects/{project_id}/vps/servers?name={name}&status={status}`
- **Success Code**: `200 OK`
- **Error Codes**:
  - `400 Bad Request`: Invalid filter parameters

**Example Usage**:
```go
// List all servers
servers, err := serverClient.List(ctx, nil)

// List servers by name filter
servers, err := serverClient.List(ctx, &servermodels.ListOptions{
    Name: "web-server",
})

// List only active servers
servers, err := serverClient.List(ctx, &servermodels.ListOptions{
    Status: "active",
})
```

---

### Update Server

**Method**: `Update(ctx context.Context, id string, req *UpdateRequest) (*Server, error)`

**Request Model**:
```go
type UpdateRequest struct {
    Name           *string           `json:"name,omitempty"`
    Description    *string           `json:"description,omitempty"`
    NetworkPorts   []NetworkPortSpec `json:"network_ports,omitempty"`
    SecurityGroups []string          `json:"security_groups,omitempty"`
}
```

**Response Model**: `*Server` (updated server)

**HTTP Details**:
- **Endpoint**: `PATCH /v1/projects/{project_id}/vps/servers/{id}`
- **Success Code**: `200 OK`
- **Error Codes**:
  - `400 Bad Request`: Validation error
  - `404 Not Found`: Server does not exist
  - `409 Conflict`: Name conflict
  - `422 Unprocessable Entity`: Invalid operation (e.g., cannot update while building)

**Example Usage**:
```go
// Update name only
name := "new-server-name"
req := &servermodels.UpdateRequest{
    Name: &name,
}
server, err := serverClient.Update(ctx, "srv-abc123", req)

// Update network attachments
req := &servermodels.UpdateRequest{
    NetworkPorts: []servermodels.NetworkPortSpec{
        {NetworkID: "net-1", IsPrimary: true},
        {NetworkID: "net-2", IsPrimary: false},
    },
}
server, err := serverClient.Update(ctx, "srv-abc123", req)

// Update security groups
req := &servermodels.UpdateRequest{
    SecurityGroups: []string{"sg-web", "sg-default"},
}
server, err := serverClient.Update(ctx, "srv-abc123", req)
```

**Immutable Fields** (cannot be updated):
- `flavor_id` - Requires flavor resize operation (out of scope)
- `image_id` - Requires server rebuild (out of scope)
- `keypair_name` - Cannot change after injection
- `user_data` - Only applied at creation

---

### Delete Server

**Method**: `Delete(ctx context.Context, id string) error`

**Parameters**:
- `id` (string): Server UUID

**Response**: `error` (nil on success)

**HTTP Details**:
- **Endpoint**: `DELETE /v1/projects/{project_id}/vps/servers/{id}`
- **Success Code**: `204 No Content`
- **Error Codes**:
  - `404 Not Found`: Server does not exist (not an error, idempotent)
  - `409 Conflict`: Server has dependent resources (rare)
  - `500 Internal Server Error`: Platform error

**Example Usage**:
```go
err := serverClient.Delete(ctx, "srv-abc123def456")
if err != nil {
    // Handle error
}

// Poll to confirm deletion (optional)
for {
    _, err := serverClient.Get(ctx, "srv-abc123def456")
    if err != nil {
        // 404 = successfully deleted
        break
    }
    time.Sleep(2 * time.Second)
}
```

**Special Behavior**:
- Idempotent: Deleting non-existent server returns success (not error)
- Asynchronous: Delete may take time (poll for 404 to confirm)
- Cleanup: Platform automatically detaches networks and removes security group associations

---

## Status Polling Pattern

### Server Status Values

| Status | Description | Next States |
|--------|-------------|-------------|
| `building` | Server is being provisioned | `active`, `error` |
| `active` | Server is running and ready | `deleted`, `error` |
| `error` | Server entered error state | `deleted` |
| `deleted` | Server has been deleted | (terminal) |

### Polling Implementation

```go
func waitForServerActive(ctx context.Context, client ServerClient, id string, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    backoff := 1 * time.Second
    maxBackoff := 30 * time.Second

    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout waiting for server to become active")
        case <-ticker.C:
            server, err := client.Get(ctx, id)
            if err != nil {
                return fmt.Errorf("failed to get server status: %w", err)
            }

            switch server.Status {
            case "active":
                return nil
            case "error":
                return fmt.Errorf("server entered error state")
            case "building":
                // Continue polling
            default:
                // Unknown status, continue polling
            }

            // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
            if backoff < maxBackoff {
                backoff *= 2
                if backoff > maxBackoff {
                    backoff = maxBackoff
                }
            }
            ticker.Reset(backoff)
        }
    }
}
```

**Usage in Create**:
```go
// Create server
server, err := serverClient.Create(ctx, createRequest)
if err != nil {
    return err
}

// Wait for active status
err = waitForServerActive(ctx, serverClient, server.ID, 10*time.Minute)
if err != nil {
    return fmt.Errorf("server creation failed: %w", err)
}
```

---

## Error Handling

### HTTP Error Response Format

```go
type APIError struct {
    StatusCode int    `json:"-"`
    Code       string `json:"code"`
    Message    string `json:"message"`
    Details    string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

### Error Code Mapping

| HTTP Code | Error Code | Terraform Diagnostic Severity | Action |
|-----------|------------|-------------------------------|--------|
| 400 | `bad_request` | Error | Show API message, fix configuration |
| 404 | `not_found` | Error | Resource doesn't exist (or import failed) |
| 409 | `conflict` | Error | Name conflict, choose different name |
| 422 | `quota_exceeded` | Error | Quota limit, contact support |
| 429 | `rate_limit` | Warning | Retry with backoff |
| 500 | `internal_error` | Error | Platform issue, retry or contact support |

### Error Handling Example

```go
server, err := serverClient.Create(ctx, req)
if err != nil {
    var apiErr *servermodels.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.StatusCode {
        case 400:
            resp.Diagnostics.AddError(
                "Invalid Server Configuration",
                fmt.Sprintf("The API rejected the server configuration: %s", apiErr.Message),
            )
        case 404:
            resp.Diagnostics.AddError(
                "Resource Not Found",
                fmt.Sprintf("Referenced resource not found: %s. Verify flavor, image, network, and security group IDs exist.", apiErr.Message),
            )
        case 409:
            resp.Diagnostics.AddError(
                "Server Name Conflict",
                fmt.Sprintf("A server with name '%s' already exists. Choose a different name.", req.Name),
            )
        case 422:
            resp.Diagnostics.AddError(
                "Quota Exceeded",
                fmt.Sprintf("Cannot create server: %s. Contact support to increase quota limits.", apiErr.Message),
            )
        default:
            resp.Diagnostics.AddError(
                "API Error",
                fmt.Sprintf("Failed to create server: %s", err.Error()),
            )
        }
    } else {
        // Network error, context timeout, etc.
        resp.Diagnostics.AddError("Request Failed", err.Error())
    }
    return
}
```

---

## Context and Timeout Handling

### Context Propagation

All API methods accept `context.Context` as the first parameter:

```go
// Respect Terraform operation timeout
ctx, cancel := context.WithTimeout(ctx, createTimeout)
defer cancel()

server, err := serverClient.Create(ctx, req)
// If timeout exceeded, ctx.Err() == context.DeadlineExceeded
```

### Timeout Recommendations

| Operation | Recommended Timeout | Rationale |
|-----------|---------------------|-----------|
| Create | 10-15 minutes | Instance provisioning time (OS boot, network setup) |
| Update | 10 minutes | Network reconfiguration may take time |
| Delete | 5-10 minutes | Resource cleanup time |
| Get | 30 seconds | Simple read operation |
| List | 60 seconds | May return many results |

---

## Rate Limiting

### Rate Limit Headers

API responses include rate limit information:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 42
X-RateLimit-Reset: 1640000000
```

### Handling 429 Response

```go
func createServerWithRetry(ctx context.Context, client ServerClient, req *CreateRequest, maxRetries int) (*Server, error) {
    var server *Server
    var err error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        server, err = client.Create(ctx, req)
        if err == nil {
            return server, nil
        }

        var apiErr *APIError
        if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
            // Rate limited, wait and retry
            resetTime := parseResetHeader(apiErr.Headers["X-RateLimit-Reset"])
            waitDuration := time.Until(resetTime)
            if waitDuration > 1*time.Minute {
                waitDuration = 1 * time.Minute // Cap at 1 minute
            }

            time.Sleep(waitDuration)
            continue
        }

        // Non-retryable error
        return nil, err
    }

    return nil, fmt.Errorf("max retries exceeded: %w", err)
}
```

---

## Pagination (Future Enhancement)

Currently `List()` returns all servers. Future versions may support pagination:

```go
type ListOptions struct {
    Name   string `json:"name,omitempty"`
    Status string `json:"status,omitempty"`
    Limit  int    `json:"limit,omitempty"`  // Max results per page
    Offset int    `json:"offset,omitempty"` // Page offset
}

// Response with pagination metadata
type ListResponse struct {
    Servers    []Server `json:"servers"`
    TotalCount int      `json:"total_count"`
    Limit      int      `json:"limit"`
    Offset     int      `json:"offset"`
}
```

---

## Testing Considerations

### Mock Server Client

For unit tests, create a mock implementation:

```go
type MockServerClient struct {
    CreateFunc func(ctx context.Context, req *CreateRequest) (*Server, error)
    GetFunc    func(ctx context.Context, id string) (*Server, error)
    UpdateFunc func(ctx context.Context, id string, req *UpdateRequest) (*Server, error)
    DeleteFunc func(ctx context.Context, id string) error
    ListFunc   func(ctx context.Context, opts *ListOptions) ([]Server, error)
}

func (m *MockServerClient) Create(ctx context.Context, req *CreateRequest) (*Server, error) {
    if m.CreateFunc != nil {
        return m.CreateFunc(ctx, req)
    }
    return nil, errors.New("not implemented")
}
// ... implement other methods ...
```

### Acceptance Test Environment Variables

```bash
export ZILLAFORGE_API_ENDPOINT="https://api.zillaforge.com"
export ZILLAFORGE_API_TOKEN="test-token-abc123"
export ZILLAFORGE_PROJECT_ID="proj-xyz789"
```

---

## Summary

### API Methods
- **Create**: `POST /servers` → Returns server in "building" status, poll for "active"
- **Get**: `GET /servers/{id}` → Returns server details (user_data excluded)
- **List**: `GET /servers` → Returns array of servers with optional filters
- **Update**: `PATCH /servers/{id}` → In-place updates (name, description, networks, security groups)
- **Delete**: `DELETE /servers/{id}` → Idempotent deletion

### Key Behaviors
- **Asynchronous Create**: Poll status until "active" (default timeout 10 minutes)
- **Immutable Fields**: flavor, image, keypair, user_data
- **User Data Security**: user_data not returned by GET (excluded from import)
- **Error Handling**: Map HTTP codes to actionable Terraform diagnostics
- **Idempotent Delete**: 404 on delete is success (resource already gone)

### Client Pattern
```go
vpsClient := projectClient.VPS()
serverClient := vpsClient.Servers()
// CRUD operations: Create, Get, List, Update, Delete
```

Ready for implementation in `internal/vps/resource/server_resource.go`.
