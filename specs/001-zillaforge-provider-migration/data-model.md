# Data Model: Zillaforge Provider Migration

**Feature**: [001-zillaforge-provider-migration](./spec.md)  
**Created**: 2025-12-06  
**Purpose**: Define provider configuration schema and SDK client entity

## Overview

This feature introduces the Zillaforge provider configuration schema. Unlike typical Terraform resources with CRUD operations, the provider itself has a **configure-once** lifecycle. The "data model" here describes the provider's configuration schema and the SDK client entity that gets initialized and shared across resources.

## Entities

### Entity 1: ZillaforgeProviderModel

**Description**: The provider configuration schema model that captures user-provided settings for authenticating and connecting to Zillaforge cloud services.

**Attributes**:
| Attribute | Type | Required | Sensitive | Default | Description |
|-----------|------|----------|-----------|---------|-------------|
| `api_endpoint` | string | No | No | `https://api.zillaforge.com` (assumed) | Base URL for Zillaforge API. Override for testing or regional endpoints. |
| `api_key` | string | Yes | Yes | None | API key for authentication with Zillaforge services. Can be set via `ZILLAFORGE_API_KEY` environment variable. |
| `project_id` | string | No* | No | None | Numeric or UUID project identifier. Mutually exclusive with `project_sys_code`. Set via `ZILLAFORGE_PROJECT_ID` environment variable. |
| `project_sys_code` | string | No* | No | None | Alphanumeric system code for project. Mutually exclusive with `project_id`. Set via `ZILLAFORGE_PROJECT_SYS_CODE` environment variable. |

**Constraints**:
- Exactly one of `project_id` or `project_sys_code` MUST be provided
- If both are provided: validation error
- If neither is provided: validation error
- `api_key` is required (either explicit or via environment variable)

**Validation Rules**:
1. `api_key` presence check: Must not be empty after environment variable fallback
2. Project identifier mutual exclusivity: XOR logic for `project_id` and `project_sys_code`
3. `api_endpoint` format validation: Must be valid URL if provided (optional, can delegate to SDK)

**Lifecycle**: Configuration is validated and used once during provider `Configure()` method execution. Values are immutable after provider initialization.

---

### Entity 2: ZillaforgeProvider

**Description**: The provider implementation struct that manages SDK client lifecycle and configuration state.

**Fields**:
| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Provider version string (set at build time: "dev", "test", or semantic version) |

**Methods**:
| Method | Description |
|--------|-------------|
| `Metadata()` | Returns provider TypeName as "zillaforge" and version |
| `Schema()` | Returns provider configuration schema (ZillaforgeProviderModel) |
| `Configure()` | Validates configuration, initializes SDK client, stores client in ResourceData/DataSourceData |
| `Resources()` | Returns list of resource constructors (unchanged from scaffolding) |
| `DataSources()` | Returns list of data source constructors (unchanged from scaffolding) |
| `EphemeralResources()` | Returns list of ephemeral resource constructors (unchanged from scaffolding) |
| `Functions()` | Returns list of function constructors (unchanged from scaffolding) |

**State**: Stateless after Configure() completes. SDK client is passed to resources/data sources via context, not stored in provider struct.

---

### Entity 3: SDK Client (External Dependency)

**Description**: The Zillaforge Cloud SDK client instance from `github.com/Zillaforge/cloud-sdk` that handles HTTP communication, authentication, and API operations.

**Type**: `*zillaforge.Client` (assumed based on Go SDK conventions)

**Initialization**: Created during provider `Configure()` method with:
- API endpoint (from `api_endpoint` or default)
- API key (from `api_key`)
- Project identifier (from `project_id` OR `project_sys_code`)
- Context for timeout handling

**Lifecycle**:
1. Initialized in provider `Configure()`
2. Validated (e.g., test authentication, ping API)
3. Stored in `resp.ResourceData` and `resp.DataSourceData`
4. Retrieved by resources/data sources via type assertion
5. Reused across all resource operations in Terraform run
6. Destroyed when Terraform process exits (Go garbage collection)

**Thread Safety**: SDK client MUST be thread-safe (research assumption). Terraform may execute resource operations concurrently.

---

## Relationships

```text
┌─────────────────────────────┐
│  Terraform Configuration    │
│  (provider "zillaforge" {}) │
└──────────────┬──────────────┘
               │ config values
               ▼
┌─────────────────────────────┐
│ ZillaforgeProviderModel     │
│ - api_endpoint              │
│ - api_key (sensitive)       │
│ - project_id XOR            │
│   project_sys_code          │
└──────────────┬──────────────┘
               │ used by
               ▼
┌─────────────────────────────┐
│ ZillaforgeProvider          │
│ .Configure() method         │
└──────────────┬──────────────┘
               │ creates
               ▼
┌─────────────────────────────┐
│ SDK Client Instance         │
│ (*zillaforge.Client)        │
└──────────────┬──────────────┘
               │ shared with
               ▼
┌─────────────────────────────┐
│ Resources & Data Sources    │
│ - example_resource          │
│ - example_data_source       │
│ - example_ephemeral_resource│
└─────────────────────────────┘
```

## State Transitions

Provider configuration is **static** after initialization - no state transitions during Terraform run.

```text
[Uninitialized] 
    │
    │ Terraform init/plan/apply
    ▼
[Schema Validation]
    │
    │ Validate api_key, project identifier
    ▼
[SDK Client Creation]
    │
    │ Initialize zillaforge.Client
    ▼
[Configuration Complete]
    │
    │ Provider ready
    ▼
[Serving Resources/Data Sources]
    │
    │ Terraform operation continues
    ▼
[Provider Destroyed]
    (End of Terraform process)
```

## Terraform Schema Representation

```hcl
provider "zillaforge" {
  # Optional: defaults to production API
  api_endpoint = "https://api.zillaforge.com"

  # Required: sensitive, can use ZILLAFORGE_API_KEY env var
  api_key = "zf_abc123..."

  # Exactly one of these required:
  project_id = "12345"
  # OR
  # project_sys_code = "PROJ-ABC"
}
```

## Non-Functional Properties

**Security**:
- `api_key` marked as sensitive - won't appear in plan output or logs
- SDK client should use TLS for API communication (SDK responsibility)

**Performance**:
- Single SDK client initialization per Terraform run (not per resource)
- Client reuse amortizes initialization cost across resources
- No caching at provider level (SDK may cache internally)

**Observability**:
- Configuration errors return actionable Terraform diagnostics
- SDK initialization failures include error details
- Debug logging should show (redacted) configuration values

## Notes

- This feature does NOT introduce new Terraform resources - only provider configuration
- Existing resources will be updated to use typed SDK client instead of generic `http.Client`
- Data model is **configuration schema**, not persistent data - no storage backend needed
