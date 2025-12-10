# Feature Specification: Zillaforge Provider Migration

**Feature Branch**: `001-zillaforge-provider-migration`  
**Created**: 2025-12-06  
**Status**: Draft  
**Input**: User description: "Modify existing ScaffoldingProvider to ZillaforgeProvider and modify the Provider Client configuration using Zillaforge api sdk"

## Clarifications

### Session 2025-12-06

- Q: How should api_key be validated at configuration time? → A: Validate api_key format at configuration time (JWT token format)
- Q: What timeout should be set for SDK client initialization? → A: No explicit timeout (rely on SDK defaults)
- Q: How should network timeout errors be reported when retries are exhausted? → A: Return detailed diagnostic with retry attempts and last error
- Q: What logging levels should be used for provider operations? → A: Structured logging with INFO (lifecycle) and DEBUG (details) levels
- Q: Should multiple provider instances be supported in the same workspace? → A: Support multiple provider instances (allow provider alias)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Provider Rebranding (Priority: P1)

As a Terraform provider developer, I need to rename the ScaffoldingProvider to ZillaforgeProvider so that the provider identity reflects the actual service it manages and follows Terraform naming conventions.

**Why this priority**: This is the foundational change that establishes the provider's identity. Without this, all subsequent work would still reference scaffolding terminology, causing confusion for users and breaking the provider's branding.

**Independent Test**: Can be fully tested by building the provider, checking the provider binary name, verifying `terraform init` recognizes "zillaforge" as the provider name, and confirming all internal references use ZillaforgeProvider instead of ScaffoldingProvider. Delivers a properly branded but still functional provider.

**Acceptance Scenarios**:

1. **Given** the provider codebase uses ScaffoldingProvider, **When** I rename all references to ZillaforgeProvider, **Then** the provider compiles without errors and maintains backward compatibility with the Plugin Framework
2. **Given** a Terraform configuration file, **When** I specify `provider "zillaforge"`, **Then** Terraform recognizes the provider and can initialize it
3. **Given** the provider is built, **When** I check the provider metadata, **Then** it reports the TypeName as "zillaforge"
4. **Given** example Terraform configurations, **When** I update them to use `zillaforge_*` resource naming, **Then** they remain syntactically valid

---

### User Story 2 - Zillaforge SDK Integration (Priority: P2)

As a Terraform provider developer, I need to integrate the Zillaforge API SDK into the provider configuration so that the provider can authenticate and communicate with actual Zillaforge cloud services instead of using generic HTTP client placeholders.

**Why this priority**: This enables real API functionality. While P1 establishes naming, this story enables actual cloud resource management. It builds on P1 but is independently testable against a real or mock Zillaforge API endpoint.

**Independent Test**: Can be tested by configuring the provider with Zillaforge API credentials, verifying the SDK client initializes successfully, making a test API call (e.g., list resources or validate credentials), and confirming proper error handling for invalid credentials. Delivers a provider that can actually connect to Zillaforge services.

**Acceptance Scenarios**:

1. **Given** the provider configuration includes Zillaforge API endpoint and credentials, **When** the provider Configure method runs, **Then** it initializes a Zillaforge SDK client successfully
2. **Given** invalid Zillaforge API credentials, **When** the provider attempts to configure, **Then** it returns actionable diagnostic errors explaining authentication failure
3. **Given** the Zillaforge SDK client is initialized, **When** resources or data sources attempt API calls, **Then** they use the SDK client instead of generic HTTP client
4. **Given** the provider is configured with environment variables, **When** explicit configuration is omitted, **Then** the provider reads credentials from standard Zillaforge SDK environment variables
5. **Given** API rate limiting or transient failures, **When** API calls are made, **Then** the SDK client implements retry logic with exponential backoff

---

### User Story 3 - Provider Schema Update (Priority: P3)

As a Terraform user, I need the provider schema to accept Zillaforge-specific configuration attributes (API endpoint, API key, project identifier, etc.) so that I can properly configure the provider for my Zillaforge account and project environment.

**Why this priority**: This improves usability by exposing proper configuration options. While P2 can work with hardcoded or default SDK settings, this story enables users to customize behavior and specify their target project. It's the polish layer that makes the provider production-ready.

**Independent Test**: Can be tested by writing Terraform configurations with various provider block configurations, running `terraform validate` and `terraform plan`, and verifying that all documented attributes are accepted, validated correctly, and appear in generated documentation. Delivers a provider with complete, user-friendly configuration options.

**Acceptance Scenarios**:

1. **Given** a Terraform configuration with provider block, **When** I specify `api_endpoint`, `api_key`, and either `project_id` or `project_sys_code` attributes, **Then** the provider validates and accepts these values
2. **Given** both `project_id` and `project_sys_code` are provided, **When** the provider validates configuration, **Then** it returns a diagnostic error explaining that only one project identifier should be specified
3. **Given** neither `project_id` nor `project_sys_code` is provided, **When** the provider validates configuration, **Then** it returns a diagnostic error requiring at least one project identifier
4. **Given** required attributes like `api_key` are missing, **When** the provider validates configuration, **Then** it returns clear diagnostic messages identifying missing required fields
5. **Given** optional attributes like `api_endpoint` are omitted, **When** the provider initializes, **Then** it uses sensible defaults (e.g., production Zillaforge API endpoint)
6. **Given** sensitive attributes like `api_key`, **When** the provider schema is defined, **Then** they are marked as sensitive to prevent accidental exposure in logs or plan output
7. **Given** the provider schema is updated, **When** documentation is generated, **Then** it includes MarkdownDescription for all attributes explaining their purpose and valid values

---

### Edge Cases

- What happens when the Zillaforge API endpoint is unreachable during provider configuration? → Provider returns detailed diagnostic with retry attempts and connection failure details
- How does the provider handle SDK version compatibility issues if the API changes? → Relies on SDK's built-in version handling and retry logic
- What happens if API credentials expire mid-operation during a long-running Terraform apply? → SDK refresh logic handles token expiration; provider logs at DEBUG level
- How does the provider behave when the API returns unexpected error formats not covered by the SDK? → Provider wraps SDK errors in diagnostics with original error details
- What happens if multiple provider configurations are used in the same Terraform workspace? → Each aliased provider instance maintains independent SDK client for different projects
- How does the provider handle proxy or network configuration requirements for SDK API calls? → SDK respects standard Go HTTP proxy environment variables (HTTP_PROXY, HTTPS_PROXY, NO_PROXY)
- What happens if api_key is provided but not in valid JWT format? → Provider validation fails fast at configuration time with clear format error before SDK initialization
- How are lifecycle events logged for debugging multi-instance scenarios? → Each provider instance logs with structured context including provider alias at INFO level

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provider MUST rename all occurrences of "Scaffolding" to "Zillaforge" in type names, function names, and identifiers
- **FR-002**: Provider MUST update the TypeName metadata to "zillaforge" for Terraform recognition
- **FR-003**: Provider MUST update go.mod module path to reflect zillaforge naming
- **FR-004**: Provider MUST integrate the Zillaforge API SDK as a dependency
- **FR-005**: Provider Configure method MUST initialize a Zillaforge SDK client instance
- **FR-006**: Provider MUST pass the initialized SDK client to all resources and data sources via ResourceData and DataSourceData
- **FR-007**: Provider schema MUST include an `api_endpoint` attribute (optional, with default value for production Zillaforge API)
- **FR-008**: Provider schema MUST include an `api_key` attribute (required, marked as sensitive, validated as JWT token format)
- **FR-009**: Provider schema MUST include a `project_id` attribute (optional, mutually exclusive with project_sys_code)
- **FR-010**: Provider schema MUST include a `project_sys_code` attribute (optional, mutually exclusive with project_id)
- **FR-011**: Provider MUST validate that exactly one of `project_id` or `project_sys_code` is provided (not both, not neither)
- **FR-012**: Provider MUST validate that API credentials are present before attempting API calls
- **FR-013**: Provider MUST return detailed diagnostics for configuration errors including retry attempts and last error when network failures occur (missing credentials, invalid endpoint format, conflicting project identifiers, connection timeouts, etc.)
- **FR-014**: Provider MUST respect SDK client context timeouts derived from Terraform operation context
- **FR-015**: All provider schema attributes MUST have MarkdownDescription for documentation generation
- **FR-016**: Provider MUST handle SDK client initialization failures gracefully without panics
- **FR-017**: Provider MUST update example Terraform configurations to use zillaforge provider syntax
- **FR-018**: Provider MUST implement structured logging with INFO level for lifecycle events (initialization, configuration) and DEBUG level for detailed SDK operations
- **FR-019**: Provider MUST support multiple provider instances via Terraform alias configuration to enable managing resources across different Zillaforge projects in the same workspace

### Non-Functional Requirements

- **NFR-001**: API key validation MUST complete within 100ms using JWT format checks (header.payload.signature structure)
- **NFR-002**: SDK initialization timeout relies on SDK defaults (no explicit provider-level timeout override)
- **NFR-003**: Diagnostic error messages MUST include retry count and final error details when SDK operations fail after retry exhaustion
- **NFR-004**: Logging MUST use terraform-plugin-log framework with structured fields (level=INFO for Configure/Metadata methods, level=DEBUG for SDK client operations)
- **NFR-005**: Each provider instance MUST maintain independent SDK client state to support multi-project management via provider aliases

### Assumptions

- **A-001**: Zillaforge API SDK exists at `github.com/Zillaforge/cloud-sdk` and is a Go package compatible with the provider
- **A-002**: Zillaforge API uses API key authentication (common pattern for cloud providers)
- **A-003**: Zillaforge API has a stable base URL that can be used as default for `api_endpoint` when not specified
- **A-004**: SDK client is thread-safe and can be shared across resources/data sources (standard Go SDK pattern)
- **A-005**: SDK client accepts both project_id and project_sys_code as valid project identifiers
- **A-006**: Environment variables follow convention: `ZILLAFORGE_API_KEY`, `ZILLAFORGE_API_ENDPOINT`, `ZILLAFORGE_PROJECT_ID`, `ZILLAFORGE_PROJECT_SYS_CODE`
- **A-007**: Existing scaffolding resource/data source implementations will be updated in separate features (this feature focuses on provider core only)

### Key Entities

- **ZillaforgeProvider**: The main provider implementation managing SDK client lifecycle and configuration
- **ZillaforgeProviderModel**: The provider configuration schema model containing API credentials (api_endpoint, api_key), and mutually exclusive project identifiers (project_id or project_sys_code)
- **SDK Client**: The Zillaforge API client instance from `github.com/Zillaforge/cloud-sdk` initialized during provider configuration and shared across all operations

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Provider compiles successfully with all references renamed from Scaffolding to Zillaforge (zero compilation errors)
- **SC-002**: `terraform init` successfully initializes the provider when configured with `provider "zillaforge"` block
- **SC-003**: Provider Configure method successfully initializes Zillaforge SDK client when given valid credentials (verifiable via debug logs)
- **SC-004**: Provider returns clear diagnostic errors within 5 seconds when configured with invalid credentials or unreachable endpoint
- **SC-005**: Generated provider documentation includes all configuration attributes with MarkdownDescription (100% attribute coverage)
- **SC-006**: All existing acceptance tests pass after renaming (ensuring no regression in Plugin Framework compliance)
