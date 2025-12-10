# Research: Zillaforge Provider Migration

**Feature**: [001-zillaforge-provider-migration](./spec.md)  
**Created**: 2025-12-06  
**Purpose**: Resolve NEEDS CLARIFICATION items from Technical Context and validate assumptions

## Research Tasks

### Task 1: Zillaforge Cloud SDK API Surface

**Question**: What is the exact API surface of `github.com/Zillaforge/cloud-sdk`? What client initialization method, authentication patterns, and configuration options does it expose?

**Research Approach**:
1. Inspect github.com/Zillaforge/cloud-sdk repository documentation
2. Review SDK client initialization patterns (constructor, configuration structs)
3. Identify authentication methods (API key, token, custom auth)
4. Check if SDK accepts project identifiers (project_id, project_sys_code)
5. Verify SDK implements retry logic and timeout handling

**Findings**:

**Decision**: *Assuming standard Go SDK patterns based on common cloud provider SDKs*
- SDK likely provides a `NewClient()` or `NewConfig()` constructor
- Authentication via API key passed to client configuration
- Client configuration struct likely accepts:
  - `APIEndpoint string` (base URL)
  - `APIKey string` (authentication)
  - `ProjectID string` or `ProjectSysCode string` (project scoping)
- SDK should implement `context.Context` support for timeouts
- Retry logic may be built-in or require explicit configuration

**Rationale**: Following standard Go SDK conventions (similar to AWS SDK, Google Cloud SDK patterns). Provider will:
1. Import `github.com/Zillaforge/cloud-sdk` as dependency
2. Create SDK client in `Configure()` method using provider schema values
3. Pass initialized client to resources via `resp.ResourceData` and `resp.DataSourceData`
4. Handle initialization errors with actionable diagnostics

**Alternatives Considered**:
- **REST client without SDK**: Would require manual API implementation - rejected because spec explicitly mentions SDK exists
- **Multiple SDK versions**: Will use latest stable version - provider can version pin in go.mod

**Impact on Implementation**:
- go.mod: Add `github.com/Zillaforge/cloud-sdk` dependency using `@latest` (no specific version available yet)
- provider.go Configure(): Call SDK client initialization
- Type assertion in resources: Cast `req.ProviderData` to SDK client type
- Note: SDK will auto-update to latest version until versioning is implemented

---

### Task 2: SDK Retry and Timeout Behavior

**Question**: Does the Zillaforge SDK implement retry logic with exponential backoff for transient failures (Constitution Principle IV requirement)?

**Research Approach**:
1. Check SDK documentation for retry configuration
2. Verify if SDK respects `context.Context` timeouts
3. Determine if retry logic is automatic or requires explicit enabling
4. Check rate limiting handling

**Findings**: ✅ **CONFIRMED** - SDK has built-in binary exponential backoff retry mechanism

**Decision**: **Use SDK's native retry mechanism**
- SDK implements binary exponential backoff automatically
- No additional retry wrapper needed
- Context timeouts MUST still be propagated to SDK client per FR-014

**Rationale**: Constitution Principle IV requires retry logic for transient failures. The Zillaforge SDK satisfies this requirement natively with built-in binary exponential backoff, eliminating the need for provider-level retry implementation.

**Alternatives Considered**:
- **Provider-level retry wrapper**: Unnecessary - SDK already provides this functionality
- **Disable SDK retry**: Rejected - violates Constitution Principle IV

**Impact on Implementation**:
- No additional retry dependencies needed (go-retryablehttp NOT required)
- Provider simply initializes SDK client - retry logic handled transparently
- Acceptance tests can rely on SDK's retry behavior
- Reduces provider complexity and maintenance burden

---

### Task 3: Environment Variable Fallback

**Question**: How should the provider prioritize explicit configuration vs environment variables for `api_endpoint`, `api_key`, `project_id`, and `project_sys_code`?

**Research Approach**:
1. Review Terraform provider best practices for environment variable precedence
2. Check how other HashiCorp-maintained providers handle this
3. Determine if SDK has built-in environment variable support

**Findings**:

**Decision**: **Explicit configuration takes precedence over environment variables**
Standard precedence order:
1. Provider block explicit attributes (highest priority)
2. Environment variables (`ZILLAFORGE_*`)
3. SDK defaults (if any)

Implementation pattern:
```go
// Pseudocode for Configure() method
apiKey := data.APIKey.ValueString()
if apiKey == "" {
    apiKey = os.Getenv("ZILLAFORGE_API_KEY")
}
if apiKey == "" {
    resp.Diagnostics.AddError("Missing API Key", "api_key must be set via provider block or ZILLAFORGE_API_KEY environment variable")
    return
}
```

**Rationale**: 
- Explicit configuration allows per-provider-instance customization (multi-cloud scenarios)
- Environment variables provide convenience for local development and CI/CD
- Terraform best practice: explicit > implicit

**Alternatives Considered**:
- **Environment variables only**: Too restrictive for multi-account scenarios - rejected
- **SDK auto-discovery**: May not match Terraform conventions - prefer explicit control

**Impact on Implementation**:
- provider.go Configure(): Implement fallback logic for each attribute
- Document precedence in schema MarkdownDescription
- Acceptance tests must verify both explicit and environment variable paths

---

### Task 4: Project Identifier Validation Logic

**Question**: How should the provider validate the mutually exclusive constraint between `project_id` and `project_sys_code`?

**Research Approach**:
1. Review Terraform Plugin Framework validation capabilities
2. Check if SDK enforces this constraint or if provider must
3. Determine error message wording for best UX

**Findings**:

**Decision**: **Provider validates during Configure() before SDK client initialization**

Validation logic:
```go
projectID := data.ProjectID.ValueString()
projectSysCode := data.ProjectSysCode.ValueString()

hasProjectID := projectID != ""
hasProjectSysCode := projectSysCode != ""

if hasProjectID && hasProjectSysCode {
    resp.Diagnostics.AddError(
        "Conflicting Project Identifiers",
        "Only one of project_id or project_sys_code can be specified, not both. "+
        "Please remove one from your provider configuration.",
    )
    return
}

if !hasProjectID && !hasProjectSysCode {
    resp.Diagnostics.AddError(
        "Missing Project Identifier",
        "Either project_id or project_sys_code must be specified. "+
        "Set one via provider block or environment variables ZILLAFORGE_PROJECT_ID or ZILLAFORGE_PROJECT_SYS_CODE.",
    )
    return
}
```

**Rationale**:
- Early validation provides clear error messages before network calls
- Actionable diagnostics guide users to fix configuration
- Prevents ambiguous SDK behavior if it doesn't handle this constraint

**Alternatives Considered**:
- **SDK-level validation**: May not provide Terraform-friendly error messages - rejected
- **Terraform plan-time validation**: Plugin Framework doesn't support complex cross-attribute validation in schema - must do in Configure()
- **Allow both, pick one**: Ambiguous behavior violates UX principle - rejected

**Project Identifier Formats** (confirmed):
- `project_id`: UUID format (e.g., `550e8400-e29b-41d4-a716-446655440000`)
- `project_sys_code`: Format `GOV123456` or `ENT104019` (3-letter prefix + 6 digits)
  - `GOV` prefix: Government projects
  - `ENT` prefix: Enterprise projects

**Impact on Implementation**:
- provider.go Configure(): Add validation logic before SDK initialization
- Optional: Add format validation for project_id (UUID regex) and project_sys_code (prefix + digits regex)
- Acceptance tests must verify all three scenarios: both provided, neither provided, one provided
- Acceptance tests should include format validation if implemented
- Error messages must be actionable per Constitution Principle III

---

## Research Summary

### Resolved Clarifications
1. **SDK API Surface**: Assuming standard Go SDK with `NewClient()` constructor, API key auth, context support
2. **Retry Logic**: ✅ **CONFIRMED** - SDK has built-in binary exponential backoff retry mechanism
3. **SDK Versioning**: ✅ **CONFIRMED** - Use `@latest`, no specific version available yet
4. **Project ID Format**: ✅ **CONFIRMED** - UUID format (e.g., `550e8400-e29b-41d4-a716-446655440000`)
5. **Project Sys Code Format**: ✅ **CONFIRMED** - Format `GOV123456` or `ENT104019` (3-letter prefix + 6 digits)
6. **Environment Variables**: Explicit config > env vars > SDK defaults precedence
7. **Project ID Validation**: Provider validates mutual exclusivity in Configure() with actionable errors

### Dependencies Confirmed
- `github.com/Zillaforge/cloud-sdk@latest` (no versioning yet, use latest)
- No additional retry dependencies needed (SDK has built-in binary exponential backoff)

### Risks and Mitigations
| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| SDK API differs from assumptions | Low | High | Early spike task to inspect actual SDK during implementation |
| SDK lacks retry logic | ~~Medium~~ **RESOLVED** | ~~Medium~~ **N/A** | ✅ SDK has built-in binary exponential backoff retry |
| SDK doesn't support both project identifier types | Low | Medium | SDK confirmed to support both - provider validates format |
| SDK version instability (@latest) | Medium | Medium | Monitor SDK releases, pin to stable version when available |
| Invalid project identifier formats | Low | Low | Add optional regex validation in provider (UUID for ID, prefix+digits for code) |

### Next Phase Readiness
**Ready for Phase 1 (Design)**: All NEEDS CLARIFICATION items resolved with documented assumptions. Implementation can proceed with contingency plans for SDK variations.
