# Provider Configuration Contract

**Feature**: Zillaforge Provider Migration  
**Contract Type**: Provider Configuration Schema  
**Created**: 2025-12-06

## Overview

This contract defines the Terraform provider configuration schema for the Zillaforge provider. Unlike typical REST/GraphQL API contracts, this describes the **Terraform HCL interface** that users interact with when configuring the provider block.

## Provider Block Schema

### HCL Syntax

```hcl
provider "zillaforge" {
  api_endpoint      = string  # Optional
  api_key           = string  # Required, sensitive
  project_id        = string  # Optional, mutually exclusive with project_sys_code
  project_sys_code  = string  # Optional, mutually exclusive with project_id
}
```

### Attribute Specifications

#### `api_endpoint`

**Type**: `string`  
**Required**: `false`  
**Sensitive**: `false`  
**Default**: Production Zillaforge API endpoint (assumed `https://api.zillaforge.com`)  
**Environment Variable**: `ZILLAFORGE_API_ENDPOINT`

**Description**:
> Base URL for the Zillaforge API. Override this to use a different environment (staging, development) or regional endpoint. If not specified, defaults to the production API endpoint.

**Validation**:
- Must be a valid URL format if provided
- HTTPS recommended (HTTP may be rejected by SDK or API)

**Examples**:
```hcl
# Use default (production)
provider "zillaforge" {
  api_key = "..."
  project_id = "..."
}

# Override for staging
provider "zillaforge" {
  api_endpoint = "https://staging-api.zillaforge.com"
  api_key = "..."
  project_id = "..."
}

# Use environment variable
# export ZILLAFORGE_API_ENDPOINT=https://dev-api.zillaforge.com
provider "zillaforge" {
  api_key = "..."
  project_id = "..."
}
```

---

#### `api_key`

**Type**: `string`  
**Required**: `true`  
**Sensitive**: `true`  
**Default**: None  
**Environment Variable**: `ZILLAFORGE_API_KEY`

**Description**:
> API key for authenticating with Zillaforge services. This credential is sensitive and will not be displayed in Terraform plan output or logs. Can be provided via the provider block or the `ZILLAFORGE_API_KEY` environment variable.

**Validation**:
- Must not be empty after environment variable fallback
- Format validation delegated to SDK (provider checks presence only)

**Error Scenarios**:
- **Missing**: `Error: Missing API Key - api_key must be set via provider block or ZILLAFORGE_API_KEY environment variable`

**Examples**:
```hcl
# Explicit in provider block (not recommended for production)
provider "zillaforge" {
  api_key = "zf_abc123def456ghi789..."
  project_id = "12345"
}

# Use environment variable (recommended)
# export ZILLAFORGE_API_KEY=zf_abc123def456ghi789...
provider "zillaforge" {
  project_id = "12345"
}

# Use Terraform variable (recommended for reusability)
variable "zillaforge_api_key" {
  type      = string
  sensitive = true
}

provider "zillaforge" {
  api_key = var.zillaforge_api_key
  project_id = "12345"
}
```

---

#### `project_id`

**Type**: `string`  
**Required**: `false` (but one of `project_id` or `project_sys_code` is required)  
**Sensitive**: `false`  
**Default**: None  
**Environment Variable**: `ZILLAFORGE_PROJECT_ID`  
**Mutually Exclusive With**: `project_sys_code`

**Description**:
> Numeric or UUID identifier for the Zillaforge project to manage resources within. Exactly one of `project_id` or `project_sys_code` must be specified - they cannot both be set or both be omitted.

**Validation**:
- Exactly one of `project_id` or `project_sys_code` must be provided
- Cannot be empty string if specified
- Format validation delegated to SDK

**Error Scenarios**:
- **Both provided**: `Error: Conflicting Project Identifiers - Only one of project_id or project_sys_code can be specified, not both. Please remove one from your provider configuration.`
- **Neither provided**: `Error: Missing Project Identifier - Either project_id or project_sys_code must be specified. Set one via provider block or environment variables ZILLAFORGE_PROJECT_ID or ZILLAFORGE_PROJECT_SYS_CODE.`

**Examples**:
```hcl
# Using project_id
provider "zillaforge" {
  api_key = "..."
  project_id = "12345"
}

# Using environment variable
# export ZILLAFORGE_PROJECT_ID=12345
provider "zillaforge" {
  api_key = "..."
}

# Using Terraform variable
variable "zillaforge_project_id" {
  type = string
}

provider "zillaforge" {
  api_key = var.zillaforge_api_key
  project_id = var.zillaforge_project_id
}
```

---

#### `project_sys_code`

**Type**: `string`  
**Required**: `false` (but one of `project_id` or `project_sys_code` is required)  
**Sensitive**: `false`  
**Default**: None  
**Environment Variable**: `ZILLAFORGE_PROJECT_SYS_CODE`  
**Mutually Exclusive With**: `project_id`

**Description**:
> Alphanumeric system code for the Zillaforge project. Alternative to `project_id` for organizations that use system codes instead of numeric IDs. Exactly one of `project_id` or `project_sys_code` must be specified - they cannot both be set or both be omitted.

**Validation**:
- Exactly one of `project_id` or `project_sys_code` must be provided
- Cannot be empty string if specified
- Format validation delegated to SDK

**Error Scenarios**:
- Same as `project_id` (see above)

**Examples**:
```hcl
# Using project_sys_code
provider "zillaforge" {
  api_key = "..."
  project_sys_code = "PROJ-ABC-123"
}

# Using environment variable
# export ZILLAFORGE_PROJECT_SYS_CODE=PROJ-ABC-123
provider "zillaforge" {
  api_key = "..."
}

# ERROR: both specified
provider "zillaforge" {
  api_key = "..."
  project_id = "12345"
  project_sys_code = "PROJ-ABC-123"  # This will cause validation error
}
```

---

## Configuration Validation Logic

### Precedence Order

For each attribute:
1. Explicit provider block value (highest priority)
2. Environment variable (`ZILLAFORGE_*`)
3. Default value (if applicable)
4. Error if required and not found

### Validation Sequence

```text
1. Load explicit values from provider block
2. Apply environment variable fallbacks for unset values
3. Validate api_key presence
4. Validate project identifier mutual exclusivity:
   - Check if both project_id and project_sys_code are set → ERROR
   - Check if neither project_id nor project_sys_code is set → ERROR
   - Otherwise → PASS
5. Initialize SDK client with validated configuration
6. Handle SDK initialization errors with diagnostics
```

### Pseudo-code

```go
// Provider Configure() method validation logic
func (p *ZillaforgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var data ZillaforgeProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    
    // Step 1: Environment variable fallbacks
    apiKey := getConfigOrEnv(data.APIKey, "ZILLAFORGE_API_KEY")
    apiEndpoint := getConfigOrEnv(data.APIEndpoint, "ZILLAFORGE_API_ENDPOINT", DEFAULT_ENDPOINT)
    projectID := getConfigOrEnv(data.ProjectID, "ZILLAFORGE_PROJECT_ID")
    projectSysCode := getConfigOrEnv(data.ProjectSysCode, "ZILLAFORGE_PROJECT_SYS_CODE")
    
    // Step 2: Validate api_key
    if apiKey == "" {
        resp.Diagnostics.AddError("Missing API Key", "...")
        return
    }
    
    // Step 3: Validate project identifier mutual exclusivity
    hasProjectID := projectID != ""
    hasProjectSysCode := projectSysCode != ""
    
    if hasProjectID && hasProjectSysCode {
        resp.Diagnostics.AddError("Conflicting Project Identifiers", "...")
        return
    }
    
    if !hasProjectID && !hasProjectSysCode {
        resp.Diagnostics.AddError("Missing Project Identifier", "...")
        return
    }
    
    // Step 4: Initialize SDK client
    client, err := zillaforge.NewClient(zillaforge.Config{
        APIEndpoint: apiEndpoint,
        APIKey: apiKey,
        ProjectID: projectID,
        ProjectSysCode: projectSysCode,
    })
    
    if err != nil {
        resp.Diagnostics.AddError("SDK Initialization Failed", fmt.Sprintf("Unable to create Zillaforge client: %s", err))
        return
    }
    
    // Step 5: Share client with resources
    resp.ResourceData = client
    resp.DataSourceData = client
}
```

---

## Complete Provider Configuration Examples

### Minimal Configuration (Environment Variables)

```bash
export ZILLAFORGE_API_KEY=zf_abc123...
export ZILLAFORGE_PROJECT_ID=12345
```

```hcl
provider "zillaforge" {
  # All configuration from environment variables
}
```

### Full Explicit Configuration

```hcl
provider "zillaforge" {
  api_endpoint     = "https://api.zillaforge.com"
  api_key          = "zf_abc123..."
  project_id       = "12345"
}
```

### Production-Ready Configuration (Variables)

```hcl
variable "zillaforge_api_key" {
  description = "Zillaforge API key"
  type        = string
  sensitive   = true
}

variable "zillaforge_project_id" {
  description = "Zillaforge project ID"
  type        = string
}

variable "zillaforge_api_endpoint" {
  description = "Zillaforge API endpoint"
  type        = string
  default     = "https://api.zillaforge.com"
}

provider "zillaforge" {
  api_endpoint = var.zillaforge_api_endpoint
  api_key      = var.zillaforge_api_key
  project_id   = var.zillaforge_project_id
}
```

### Multi-Project Configuration

```hcl
# Provider alias for production project
provider "zillaforge" {
  alias      = "prod"
  api_key    = var.prod_api_key
  project_id = var.prod_project_id
}

# Provider alias for staging project
provider "zillaforge" {
  alias      = "staging"
  api_key    = var.staging_api_key
  project_id = var.staging_project_id
}

# Use specific provider for resource
resource "zillaforge_example" "prod_resource" {
  provider = zillaforge.prod
  # ...
}

resource "zillaforge_example" "staging_resource" {
  provider = zillaforge.staging
  # ...
}
```

---

## Diagnostic Messages

All error messages must be actionable and guide users to fix configuration issues.

| Scenario | Severity | Summary | Detail |
|----------|----------|---------|--------|
| Missing API key | Error | Missing API Key | api_key must be set via provider block or ZILLAFORGE_API_KEY environment variable |
| Both project identifiers | Error | Conflicting Project Identifiers | Only one of project_id or project_sys_code can be specified, not both. Please remove one from your provider configuration. |
| Neither project identifier | Error | Missing Project Identifier | Either project_id or project_sys_code must be specified. Set one via provider block or environment variables ZILLAFORGE_PROJECT_ID or ZILLAFORGE_PROJECT_SYS_CODE. |
| SDK init failure | Error | SDK Initialization Failed | Unable to create Zillaforge client: [specific error]. Verify your api_endpoint and api_key are correct. |
| Invalid endpoint format | Error | Invalid API Endpoint | api_endpoint must be a valid URL (e.g., https://api.zillaforge.com) |

---

## Testing Contract

Provider configuration must be tested with acceptance tests covering:

1. **Valid minimal configuration**: api_key + project_id
2. **Valid alternative configuration**: api_key + project_sys_code
3. **Environment variable fallback**: All values from env vars
4. **Explicit precedence**: Explicit values override env vars
5. **Missing api_key**: Validation error
6. **Missing project identifier**: Validation error
7. **Both project identifiers**: Validation error
8. **Invalid endpoint format**: Validation error (if implemented)
9. **SDK initialization failure**: Proper error diagnostic
10. **Multiple provider instances**: Aliased providers work independently

---

## Contract Version

**Version**: 1.0.0  
**Status**: Draft  
**Breaking Changes**: None (initial implementation)
