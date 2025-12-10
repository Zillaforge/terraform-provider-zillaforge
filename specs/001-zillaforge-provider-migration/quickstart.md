# Quickstart Guide: Zillaforge Provider Migration

**Feature**: [001-zillaforge-provider-migration](./spec.md)  
**Created**: 2025-12-06  
**Audience**: Developers implementing this feature

## Overview

This quickstart guide walks through implementing the Zillaforge Provider Migration in a test-driven manner, following the project constitution's TDD principles. Complete the user stories in priority order (P1 → P2 → P3) for incremental delivery.

## Prerequisites

- Go 1.22.4 or later installed
- Terraform 1.0+ installed for testing
- Git repository cloned and on branch `001-zillaforge-provider-migration`
- Familiarity with Terraform Plugin Framework
- Access to `github.com/Zillaforge/cloud-sdk` (or ability to create mock for testing)

## Phase 0: Environment Setup

### Step 0.1: Verify Current State

```bash
# Ensure you're on the correct branch
git branch --show-current
# Should output: 001-zillaforge-provider-migration

# Verify provider compiles in current state
cd /workspaces/terraform-provider-zillaforge
go build

# Run existing tests to establish baseline
make testacc
# Note: These should pass before starting changes
```

### Step 0.2: Review Specifications

Read in order:
1. [spec.md](./spec.md) - Functional requirements and user stories
2. [data-model.md](./data-model.md) - Provider configuration schema
3. [contracts/provider-config-schema.md](./contracts/provider-config-schema.md) - Configuration contract
4. [research.md](./research.md) - Resolved clarifications

---

## User Story 1: Provider Rebranding (P1) 🎯 MVP

**Goal**: Rename ScaffoldingProvider to ZillaforgeProvider  
**Testable Outcome**: Provider compiles, `terraform init` recognizes "zillaforge"

### Step 1.1: Write Failing Acceptance Test (RED)

Create test for provider metadata:

```bash
# Edit: internal/provider/provider_test.go
```

Add test:
```go
func TestZillaforgeProvider_Metadata(t *testing.T) {
    // This test verifies the provider TypeName is "zillaforge"
    ctx := context.Background()
    provider := New("test")()
    
    req := provider.MetadataRequest{}
    resp := &provider.MetadataResponse{}
    
    provider.Metadata(ctx, req, resp)
    
    if resp.TypeName != "zillaforge" {
        t.Errorf("Expected TypeName 'zillaforge', got '%s'", resp.TypeName)
    }
}
```

**Run test (should FAIL)**:
```bash
go test ./internal/provider -v -run TestZillaforgeProvider_Metadata
# Expected: FAIL (TypeName is still "scaffolding")
```

### Step 1.2: Implement Provider Renaming (GREEN)

**File 1: internal/provider/provider.go**

```go
// Rename type
type ZillaforgeProvider struct {
    version string
}

// Rename model
type ZillaforgeProviderModel struct {
    Endpoint types.String `tfsdk:"endpoint"`
}

// Update Metadata method
func (p *ZillaforgeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "zillaforge"  // Changed from "scaffolding"
    resp.Version = p.version
}

// Update New() function
func New(version string) func() provider.Provider {
    return func() provider.Provider {
        return &ZillaforgeProvider{  // Changed from ScaffoldingProvider
            version: version,
        }
    }
}

// Update interface checks
var _ provider.Provider = &ZillaforgeProvider{}
var _ provider.ProviderWithFunctions = &ZillaforgeProvider{}
var _ provider.ProviderWithEphemeralResources = &ZillaforgeProvider{}
```

**File 2: main.go**

Update provider instantiation (if needed):
```go
// Verify New() call uses correct function name
```

**File 3: go.mod**

```bash
# Update module path
module github.com/Zillaforge/terraform-provider-zillaforge

# Run go mod tidy to update dependencies
go mod tidy
```

**File 4: examples/provider/provider.tf**

```hcl
terraform {
  required_providers {
    zillaforge = {  # Changed from scaffolding
      source = "registry.terraform.io/Zillaforge/zillaforge"
    }
  }
}

provider "zillaforge" {  # Changed from scaffolding
  endpoint = "https://api.zillaforge.com"
}
```

**Run test (should PASS)**:
```bash
go test ./internal/provider -v -run TestZillaforgeProvider_Metadata
# Expected: PASS
```

### Step 1.3: Verify Terraform Integration

```bash
# Rebuild provider
go install

# Test terraform init recognizes provider
cd examples/provider
terraform init
# Should recognize "zillaforge" provider

# Return to root
cd ../..
```

### Step 1.4: Run Full Test Suite (GREEN)

```bash
# All existing tests should still pass
make testacc
# Expected: PASS (no functional changes, just naming)
```

**✅ Checkpoint**: User Story 1 complete. Provider is rebranded and functional.

---

## User Story 2: Zillaforge SDK Integration (P2)

**Goal**: Integrate Zillaforge SDK, initialize client in Configure()  
**Testable Outcome**: SDK client initializes, resources receive client

### Step 2.1: Add SDK Dependency

```bash
# Add Zillaforge SDK to go.mod
go get github.com/Zillaforge/cloud-sdk@latest

# Or specific version if known:
# go get github.com/Zillaforge/cloud-sdk@v0.1.0

go mod tidy
```

### Step 2.2: Write Failing Acceptance Test (RED)

```bash
# Edit: internal/provider/provider_test.go
```

Add SDK initialization test:
```go
func TestZillaforgeProvider_Configure_InitializesSDK(t *testing.T) {
    // Set up environment for SDK initialization
    os.Setenv("ZILLAFORGE_API_KEY", "test-key")
    os.Setenv("ZILLAFORGE_PROJECT_ID", "test-project")
    defer os.Unsetenv("ZILLAFORGE_API_KEY")
    defer os.Unsetenv("ZILLAFORGE_PROJECT_ID")
    
    // Test provider configuration
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: providerConfig(""),
                Check: resource.ComposeAggregateTestCheckFunc(
                    // Verify provider configured without errors
                    func(s *terraform.State) error {
                        // This test verifies Configure() succeeds
                        return nil
                    },
                ),
            },
        },
    })
}
```

**Run test (should FAIL)**:
```bash
go test ./internal/provider -v -run TestZillaforgeProvider_Configure_InitializesSDK
# Expected: FAIL (SDK client not initialized yet)
```

### Step 2.3: Implement SDK Integration (GREEN)

**Edit: internal/provider/provider.go**

```go
import (
    // ... existing imports
    zillaforge "github.com/Zillaforge/cloud-sdk"  // Add SDK import
)

func (p *ZillaforgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var data ZillaforgeProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    
    if resp.Diagnostics.HasError() {
        return
    }
    
    // Get configuration values (will expand in User Story 3)
    endpoint := data.Endpoint.ValueString()
    if endpoint == "" {
        endpoint = "https://api.zillaforge.com"  // Default
    }
    
    // For now, use placeholder credentials (will add schema in US3)
    apiKey := os.Getenv("ZILLAFORGE_API_KEY")
    projectID := os.Getenv("ZILLAFORGE_PROJECT_ID")
    
    if apiKey == "" {
        resp.Diagnostics.AddError(
            "Missing API Key",
            "ZILLAFORGE_API_KEY environment variable must be set",
        )
        return
    }
    
    if projectID == "" {
        resp.Diagnostics.AddError(
            "Missing Project ID",
            "ZILLAFORGE_PROJECT_ID environment variable must be set",
        )
        return
    }
    
    // Initialize Zillaforge SDK client
    client, err := zillaforge.NewClient(zillaforge.Config{
        APIEndpoint: endpoint,
        APIKey:      apiKey,
        ProjectID:   projectID,
    })
    
    if err != nil {
        resp.Diagnostics.AddError(
            "SDK Initialization Failed",
            fmt.Sprintf("Unable to create Zillaforge client: %s", err),
        )
        return
    }
    
    // Share client with resources and data sources
    resp.ResourceData = client
    resp.DataSourceData = client
}
```

**Run test (should PASS)**:
```bash
go test ./internal/provider -v -run TestZillaforgeProvider_Configure_InitializesSDK
# Expected: PASS
```

### Step 2.4: Update Resources to Use SDK Client

**Edit: internal/provider/example_resource.go** (and similar for data sources)

Update Configure method:
```go
func (r *ExampleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    
    client, ok := req.ProviderData.(*zillaforge.Client)  // Changed from http.Client
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            fmt.Sprintf("Expected *zillaforge.Client, got: %T", req.ProviderData),
        )
        return
    }
    
    r.client = client
}
```

Update struct:
```go
type ExampleResource struct {
    client *zillaforge.Client  // Changed from *http.Client
}
```

**Run tests**:
```bash
make testacc
# Expected: PASS
```

**✅ Checkpoint**: User Story 2 complete. SDK integrated, client available to resources.

---

## User Story 3: Provider Schema Update (P3)

**Goal**: Add proper provider schema attributes with validation  
**Testable Outcome**: Schema accepts config, validates mutual exclusivity

### Step 3.1: Write Failing Acceptance Tests (RED)

```go
func TestZillaforgeProvider_Schema_ValidatesProjectIdentifiers(t *testing.T) {
    // Test case 1: Both project_id and project_sys_code provided (should error)
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: `
                    provider "zillaforge" {
                        api_key          = "test-key"
                        project_id       = "12345"
                        project_sys_code = "PROJ-ABC"
                    }
                `,
                ExpectError: regexp.MustCompile("Conflicting Project Identifiers"),
            },
        },
    })
    
    // Test case 2: Neither project identifier provided (should error)
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: `
                    provider "zillaforge" {
                        api_key = "test-key"
                    }
                `,
                ExpectError: regexp.MustCompile("Missing Project Identifier"),
            },
        },
    })
    
    // Test case 3: Valid with project_id (should succeed)
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: `
                    provider "zillaforge" {
                        api_key    = "test-key"
                        project_id = "12345"
                    }
                `,
            },
        },
    })
}
```

**Run tests (should FAIL)**:
```bash
go test ./internal/provider -v -run TestZillaforgeProvider_Schema_ValidatesProjectIdentifiers
# Expected: FAIL (validation not implemented yet)
```

### Step 3.2: Implement Schema and Validation (GREEN)

**Edit: internal/provider/provider.go**

Update model:
```go
type ZillaforgeProviderModel struct {
    APIEndpoint    types.String `tfsdk:"api_endpoint"`
    APIKey         types.String `tfsdk:"api_key"`
    ProjectID      types.String `tfsdk:"project_id"`
    ProjectSysCode types.String `tfsdk:"project_sys_code"`
}
```

Update Schema method:
```go
func (p *ZillaforgeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Zillaforge provider for managing cloud resources.",
        Attributes: map[string]schema.Attribute{
            "api_endpoint": schema.StringAttribute{
                MarkdownDescription: "Base URL for Zillaforge API. Defaults to production endpoint if not specified. Can be set via ZILLAFORGE_API_ENDPOINT environment variable.",
                Optional:            true,
            },
            "api_key": schema.StringAttribute{
                MarkdownDescription: "API key for authenticating with Zillaforge. Can be set via ZILLAFORGE_API_KEY environment variable.",
                Optional:            true,
                Sensitive:           true,
            },
            "project_id": schema.StringAttribute{
                MarkdownDescription: "Numeric project identifier. Mutually exclusive with project_sys_code. Can be set via ZILLAFORGE_PROJECT_ID environment variable.",
                Optional:            true,
            },
            "project_sys_code": schema.StringAttribute{
                MarkdownDescription: "Alphanumeric project system code. Mutually exclusive with project_id. Can be set via ZILLAFORGE_PROJECT_SYS_CODE environment variable.",
                Optional:            true,
            },
        },
    }
}
```

Update Configure method with validation:
```go
func (p *ZillaforgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var data ZillaforgeProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    
    if resp.Diagnostics.HasError() {
        return
    }
    
    // Environment variable fallbacks
    apiEndpoint := data.APIEndpoint.ValueString()
    if apiEndpoint == "" {
        apiEndpoint = os.Getenv("ZILLAFORGE_API_ENDPOINT")
    }
    if apiEndpoint == "" {
        apiEndpoint = "https://api.zillaforge.com"
    }
    
    apiKey := data.APIKey.ValueString()
    if apiKey == "" {
        apiKey = os.Getenv("ZILLAFORGE_API_KEY")
    }
    
    projectID := data.ProjectID.ValueString()
    if projectID == "" {
        projectID = os.Getenv("ZILLAFORGE_PROJECT_ID")
    }
    
    projectSysCode := data.ProjectSysCode.ValueString()
    if projectSysCode == "" {
        projectSysCode = os.Getenv("ZILLAFORGE_PROJECT_SYS_CODE")
    }
    
    // Validate api_key
    if apiKey == "" {
        resp.Diagnostics.AddError(
            "Missing API Key",
            "api_key must be set via provider block or ZILLAFORGE_API_KEY environment variable",
        )
        return
    }
    
    // Validate project identifier mutual exclusivity
    hasProjectID := projectID != ""
    hasProjectSysCode := projectSysCode != ""
    
    if hasProjectID && hasProjectSysCode {
        resp.Diagnostics.AddError(
            "Conflicting Project Identifiers",
            "Only one of project_id or project_sys_code can be specified, not both. Please remove one from your provider configuration.",
        )
        return
    }
    
    if !hasProjectID && !hasProjectSysCode {
        resp.Diagnostics.AddError(
            "Missing Project Identifier",
            "Either project_id or project_sys_code must be specified. Set one via provider block or environment variables ZILLAFORGE_PROJECT_ID or ZILLAFORGE_PROJECT_SYS_CODE.",
        )
        return
    }
    
    // Initialize SDK client
    client, err := zillaforge.NewClient(zillaforge.Config{
        APIEndpoint:    apiEndpoint,
        APIKey:         apiKey,
        ProjectID:      projectID,
        ProjectSysCode: projectSysCode,
    })
    
    if err != nil {
        resp.Diagnostics.AddError(
            "SDK Initialization Failed",
            fmt.Sprintf("Unable to create Zillaforge client: %s", err),
        )
        return
    }
    
    resp.ResourceData = client
    resp.DataSourceData = client
}
```

**Run tests (should PASS)**:
```bash
go test ./internal/provider -v -run TestZillaforgeProvider_Schema_ValidatesProjectIdentifiers
# Expected: PASS

# Run all tests
make testacc
# Expected: PASS
```

### Step 3.3: Update Examples

**Edit: examples/provider/provider.tf**

```hcl
terraform {
  required_providers {
    zillaforge = {
      source = "registry.terraform.io/Zillaforge/zillaforge"
    }
  }
}

provider "zillaforge" {
  # Optional: Override API endpoint (defaults to production)
  # api_endpoint = "https://staging-api.zillaforge.com"

  # Required: API key for authentication
  # Recommended: Set via ZILLAFORGE_API_KEY environment variable
  api_key = "your-api-key-here"

  # Required: Exactly one of project_id or project_sys_code
  project_id = "12345"
  # OR
  # project_sys_code = "PROJ-ABC-123"
}
```

### Step 3.4: Generate Documentation

```bash
make generate
# This runs tfplugindocs to generate provider documentation

# Verify docs/index.md was updated with new schema
```

**✅ Checkpoint**: User Story 3 complete. Provider schema fully implemented with validation.

---

## Final Validation

### Validation Checklist

- [ ] All tests pass: `make testacc`
- [ ] Linting passes: `golangci-lint run`
- [ ] Documentation generated: `make generate`
- [ ] Provider compiles: `go build`
- [ ] Manual terraform init works with new provider name
- [ ] Success criteria met:
  - [ ] SC-001: Zero compilation errors ✅
  - [ ] SC-002: `terraform init` recognizes "zillaforge" ✅
  - [ ] SC-003: SDK client initializes successfully ✅
  - [ ] SC-004: Clear diagnostic errors (tested via acceptance tests) ✅
  - [ ] SC-005: 100% attribute coverage in generated docs ✅
  - [ ] SC-006: All existing tests pass ✅

### Manual Testing

```bash
# Set up test environment
export ZILLAFORGE_API_KEY="test-key"
export ZILLAFORGE_PROJECT_ID="test-project"

# Create minimal Terraform configuration
cat > test.tf <<EOF
terraform {
  required_providers {
    zillaforge = {
      source = "registry.terraform.io/Zillaforge/zillaforge"
    }
  }
}

provider "zillaforge" {}
EOF

# Test terraform commands
terraform init
terraform validate
terraform plan

# Clean up
rm -rf .terraform .terraform.lock.hcl test.tf
```

---

## Commit Strategy

Follow constitutional requirement for clear commits:

```bash
# Commit User Story 1
git add internal/provider/provider.go internal/provider/provider_test.go main.go go.mod examples/
git commit -m "feat(P1): rebrand ScaffoldingProvider to ZillaforgeProvider

- Rename provider type and model structs
- Update TypeName metadata to 'zillaforge'
- Update go.mod module path
- Update example configurations
- All existing acceptance tests pass (SC-006)
"

# Commit User Story 2
git add internal/provider/provider.go internal/provider/*_resource.go internal/provider/*_data_source.go go.mod go.sum
git commit -m "feat(P2): integrate Zillaforge SDK client

- Add github.com/Zillaforge/cloud-sdk dependency
- Initialize SDK client in Configure() method
- Share client with resources and data sources
- Handle SDK initialization errors with diagnostics
- Update resource Configure methods for typed client
"

# Commit User Story 3
git add internal/provider/provider.go internal/provider/provider_test.go examples/ docs/
git commit -m "feat(P3): implement provider configuration schema

- Add api_endpoint, api_key, project_id, project_sys_code attributes
- Implement mutual exclusivity validation for project identifiers
- Add environment variable fallback logic
- Mark api_key as sensitive
- Add MarkdownDescription for all attributes (SC-005)
- Generate provider documentation (FR-015)
- Add acceptance tests for validation scenarios
"
```

---

## Troubleshooting

### SDK Not Found

**Error**: `package github.com/Zillaforge/cloud-sdk: cannot find package`

**Solution**:
1. Verify SDK repository exists and is accessible
2. Check go.mod has correct SDK import path
3. Run `go mod download` to fetch dependencies
4. If SDK is in private repo, configure GOPRIVATE environment variable

### Test Failures

**Error**: `Expected TypeName 'zillaforge', got 'scaffolding'`

**Solution**:
- Ensure all references to "Scaffolding" are renamed to "Zillaforge"
- Check Metadata() method returns correct TypeName
- Rebuild provider: `go build`

### Validation Not Working

**Error**: Provider accepts both project_id and project_sys_code

**Solution**:
- Check Configure() method has validation logic before SDK init
- Verify tests use correct ExpectError regex
- Add debug logging to trace validation flow

---

## Next Steps

After this feature is complete:
1. Create PR for review
2. Address any code review feedback
3. Merge to main branch
4. Tag release version
5. Proceed to implement actual Zillaforge resources (separate features)

## References

- [Terraform Plugin Framework Docs](https://developer.hashicorp.com/terraform/plugin/framework)
- [Provider Configuration Schema Guide](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas)
- [Zillaforge Provider Constitution](../../.specify/memory/constitution.md)
- [Feature Specification](./spec.md)
