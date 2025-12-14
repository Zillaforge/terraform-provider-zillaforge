# Research Document: Keypair Data Source and Resource

**Feature**: 003-keypair-data-resource  
**Date**: December 13, 2025  
**Status**: Phase 0 Complete

## Purpose

This document consolidates research findings for implementing SSH keypair management in the Terraform Provider Zillaforge. It resolves technical unknowns, documents best practices, and provides implementation guidance for both data source and resource components.

## Technical Research

### 1. SSH Keypair Formats and Validation

**Decision**: Support RSA, ECDSA, and ED25519 key formats as defined in OpenSSH standards

**Rationale**:
- cloud-sdk Keypair model accepts arbitrary public_key strings (validation on API side per FR-017)
- OpenSSH format is the industry standard (ssh-rsa, ecdsa-sha2-*, ssh-ed25519 prefixes)
- ED25519 is recommended for modern security (smaller keys, better performance)
- RSA 2048+ still widely used for compatibility

**Alternatives Considered**:
1. **Only ED25519**: Rejected - limits user choice and breaks compatibility with existing keys
2. **DSA Support**: Rejected - deprecated due to security vulnerabilities, not in cloud-sdk docs
3. **Custom validation in provider**: Rejected - API handles validation (FR-017), avoid duplicate logic

**Implementation Guidance**:
- Provider passes public_key directly to API without format validation
- API returns clear error messages for invalid formats (FR-007)
- Documentation should recommend ED25519 for new keys while supporting others

### 2. Terraform Plugin Framework Sensitive Attributes

**Decision**: Use `Sensitive: true` in schema.StringAttribute for private_key field

**Rationale**:
- Plugin Framework provides built-in sensitive attribute handling
- Marks values as sensitive in state file (shows as `(sensitive value)` in plan/output)
- Prevents accidental exposure in logs and console output
- Required by FR-003 and clarification decision

**Alternatives Considered**:
1. **Custom encryption**: Rejected - Terraform state encryption is user's responsibility
2. **Ephemeral resource**: Rejected - keypairs are persistent infrastructure, not ephemeral
3. **Output-only**: Rejected - must be in state for resource tracking and drift detection

**Implementation Pattern** (from Plugin Framework docs):
```go
"private_key": schema.StringAttribute{
    MarkdownDescription: "Private key for SSH authentication (only for system-generated keypairs)",
    Computed:            true,
    Sensitive:           true, // Masks value in outputs and logs
}
```

### 3. Terraform Resource Lifecycle for Immutable Resources

**Decision**: Keypair resource supports Create, Read, Delete only (no Update except description via cloud-sdk Update())

**Rationale**:
- cloud-sdk KeypairUpdateRequest only supports description field updates
- Public key and name are immutable after creation (security best practice)
- Changing immutable fields triggers `RequiresReplace` in plan output (FR-014)

**Alternatives Considered**:
1. **Full Update support**: Rejected - API doesn't support updating public_key or name
2. **Delete+Recreate on any change**: Rejected - unnecessary for description-only updates
3. **No Update method**: Considered but description updates are useful for documentation

**Implementation Pattern**:
```go
func (r *KeypairResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "name": schema.StringAttribute{
                MarkdownDescription: "Keypair name (immutable)",
                Required:            true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(), // Force replace if changed
                },
            },
            "public_key": schema.StringAttribute{
                MarkdownDescription: "SSH public key (immutable)",
                Optional:            true,
                Computed:            true, // Generated if not provided
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(),
                    stringplanmodifier.UseStateForUnknown(), // Preserve on update
                },
            },
            "description": schema.StringAttribute{
                MarkdownDescription: "Optional description (updatable)",
                Optional:            true,
                Computed:            true,
                // No RequiresReplace - can be updated
            },
        },
    }
}
```

### 4. Data Source Design Pattern (Consistent with Flavors/Networks)

**Decision**: Single data source supporting both individual lookup and list-all modes with optional filters

**Rationale**:
- Matches flavor_data_source.go and network_data_source.go patterns
- Users familiar with existing data sources have consistent experience
- cloud-sdk List() supports name filtering via ListKeypairsOptions
- Clarification answer: "Support both single queries AND list all in same data source with optional filters"

**Pattern Analysis from existing code**:
```go
// FlavorDataSourceModel pattern
type FlavorDataSourceModel struct {
    Name    types.String   `tfsdk:"name"`    // Optional filter
    VCPUs   types.Int64    `tfsdk:"vcpus"`   // Optional filter
    Memory  types.Int64    `tfsdk:"memory"`  // Optional filter
    Flavors []FlavorModel  `tfsdk:"flavors"` // Computed list result
}

// Applied to Keypairs
type KeypairDataSourceModel struct {
    Name     types.String     `tfsdk:"name"`     // Optional filter (exact match)
    ID       types.String     `tfsdk:"id"`       // Optional filter (single lookup)
    Keypairs []KeypairModel   `tfsdk:"keypairs"` // Computed list result
}
```

**Alternatives Considered**:
1. **Separate data sources**: zillaforge_keypair (singular) and zillaforge_keypairs (plural) - Rejected for consistency
2. **Required filters**: Rejected - must support list-all per clarification
3. **Pagination params**: Rejected - cloud-sdk handles internally

**Implementation Guidance**:
- If `id` is set: call client.Get(ctx, id) and return single item in list
- If `name` is set: call client.List(ctx, &ListKeypairsOptions{Name: name}) and filter exact matches
- If both set: return validation error (mutually exclusive per clarification)
- If neither set: call client.List(ctx, nil) for all keypairs

### 5. Warning Messages for In-Use Keypair Deletion

**Decision**: Implement plan-time warning using terraform-plugin-log without blocking apply

**Rationale**:
- Clarification decision: "Terraform plan shows warning message, no blocking during apply"
- Terraform providers cannot inject warnings directly into plan output (framework limitation)
- Best practice: use tflog.Warn() during resource Read() to log warnings
- Users see warnings in verbose output but deletion proceeds per FR-015, FR-016

**Alternatives Considered**:
1. **Plan modifier with diagnostics**: Rejected - diagnostics block the operation
2. **API-side warnings**: Rejected - API doesn't track instance associations per spec assumptions
3. **Custom diff function**: Rejected - Plugin Framework doesn't support plan-time warnings without blocking

**Implementation Pattern**:
```go
func (r *KeypairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state KeypairResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    
    // Log warning (visible in TF_LOG=WARN output) but don't block
    tflog.Warn(ctx, "Deleting keypair that may be in use by VPS instances",
        map[string]interface{}{
            "keypair_id":   state.ID.ValueString(),
            "keypair_name": state.Name.ValueString(),
        })
    
    // Proceed with deletion per FR-015
    if err := r.client.VPS().Keypairs().Delete(ctx, state.ID.ValueString()); err != nil {
        resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to delete keypair: %s", err))
        return
    }
}
```

**Note**: For enhanced UX, documentation should advise users to check instance associations before deletion manually.

### 6. Import Functionality Pattern

**Decision**: Support import by keypair ID with full state reconstruction

**Rationale**:
- FR-008 requires import support
- cloud-sdk Get(ctx, keypairID) returns full Keypair object
- Private key NOT available after creation (API security), so imported resources won't have private_key
- Standard Terraform import pattern: `terraform import zillaforge_keypair.example <keypair-id>`

**Implementation Pattern**:
```go
func (r *KeypairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // req.ID contains the keypair ID from command line
    keypairID := req.ID
    
    // Fetch full keypair details from API
    keypair, err := r.client.VPS().Keypairs().Get(ctx, keypairID)
    if err != nil {
        resp.Diagnostics.AddError("Import Error", 
            fmt.Sprintf("Unable to read keypair %s: %s", keypairID, err))
        return
    }
    
    // Map to resource model
    state := KeypairResourceModel{
        ID:          types.StringValue(keypair.ID),
        Name:        types.StringValue(keypair.Name),
        Description: types.StringValue(keypair.Description),
        PublicKey:   types.StringValue(keypair.PublicKey),
        Fingerprint: types.StringValue(keypair.Fingerprint),
        // PrivateKey is NOT set - not available from API after creation
    }
    
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

### 7. Error Handling Best Practices

**Decision**: Use terraform-plugin-framework diagnostics with actionable error messages

**Rationale**:
- Constitution Principle III requires actionable errors
- FR-007 requires clear error messages for invalid public key formats
- API errors should be wrapped with context for user understanding

**Pattern from existing code**:
```go
// Good: Actionable with context
resp.Diagnostics.AddError(
    "Keypair Creation Failed",
    fmt.Sprintf("Unable to create keypair '%s': %s. Verify the public key format is valid OpenSSH (ssh-rsa, ecdsa-sha2-*, ssh-ed25519).", 
        data.Name.ValueString(), err))

// Bad: Generic
resp.Diagnostics.AddError("Error", err.Error())
```

**Common Error Scenarios**:
- Duplicate name (FR-006): "Keypair name '%s' already exists. Choose a unique name."
- Invalid public key: "Invalid public key format. Expected OpenSSH format (ssh-rsa, ecdsa-sha2-nistp256, ssh-ed25519)."
- Not found on import: "Keypair ID '%s' not found. Verify the ID is correct."
- Quota exceeded: "Account keypair limit reached. Delete unused keypairs or contact support to increase quota."

## Best Practices Summary

### Code Organization
- **Location**: `internal/vps/data/keypair_data_source.go` and `internal/vps/resource/keypair_resource.go`
- **Testing**: Colocated `*_test.go` files with acceptance tests
- **Pattern Consistency**: Follow flavor/network data source structure exactly

### Schema Definitions
- All attributes MUST have MarkdownDescription
- Use `Required`, `Optional`, `Computed` appropriately
- Mark `private_key` as `Sensitive: true`
- Use `RequiresReplace()` plan modifier for immutable fields (name, public_key)
- Use `UseStateForUnknown()` for computed fields to preserve state

### API Integration via cloud-sdk
- Use `projectClient.VPS().Keypairs()` for all operations
- List: `client.List(ctx, &keypairs.ListKeypairsOptions{Name: filterName})`
- Get: `client.Get(ctx, keypairID)`
- Create: `client.Create(ctx, &keypairs.KeypairCreateRequest{...})`
- Update: `client.Update(ctx, keypairID, &keypairs.KeypairUpdateRequest{Description: desc})`
- Delete: `client.Delete(ctx, keypairID)`

### Testing Strategy (TDD)
1. Write acceptance test that fails (no implementation)
2. Implement minimal code to pass test
3. Verify test passes
4. Refactor for quality
5. Repeat for next behavior

**Test Coverage Required**:
- Data source: list all, filter by name, filter by ID, both filters error
- Resource Create: with public_key, without public_key (generated)
- Resource Read: fetch existing keypair
- Resource Update: description only
- Resource Delete: successful deletion
- Import: by keypair ID

### Documentation Requirements
- Generate docs with `tfplugindocs` after implementation
- Examples in `examples/` directory with realistic configurations
- Import command examples in resource documentation
- Recommend ED25519 keys for new keypairs

## Technology Decisions Log

| Decision | Technology | Rationale |
|----------|-----------|-----------|
| SSH Key Formats | RSA, ECDSA, ED25519 | Industry standards, API-validated |
| Sensitive Handling | Plugin Framework `Sensitive: true` | Built-in security, prevents exposure |
| Resource Lifecycle | Create/Read/Update(desc)/Delete | API constraints, security best practice |
| Data Source Pattern | Single source with optional filters | Consistency with flavors/networks |
| Warning Mechanism | tflog.Warn() in Delete() | Framework limitation, no plan warnings |
| Import Strategy | ID-based with Get() call | Standard pattern, full state reconstruction |
| Error Messages | Diagnostics with context | Constitution requirement, UX consistency |

## References

- [Terraform Plugin Framework - Sensitive Data](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes/string#sensitive)
- [Terraform Plugin Framework - Plan Modifiers](https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification)
- [OpenSSH Public Key Format](https://www.ietf.org/rfc/rfc4253.txt) - Section 6.6
- [cloud-sdk keypairs module](https://github.com/Zillaforge/cloud-sdk/tree/main/modules/vps/keypairs)
- [Existing flavor_data_source.go](../../internal/vps/data/flavor_data_source.go) - Pattern reference
- [Existing network_data_source.go](../../internal/vps/data/network_data_source.go) - Pattern reference

## Phase 0 Completion Checklist

- [x] SSH key format standards researched and decided
- [x] Sensitive attribute pattern identified
- [x] Resource lifecycle constraints documented
- [x] Data source design pattern aligned with existing code
- [x] Warning mechanism approach defined
- [x] Import functionality pattern established
- [x] Error handling best practices documented
- [x] No NEEDS CLARIFICATION items remaining
- [x] All technical unknowns from plan.md Technical Context resolved

**Status**: ✅ Ready for Phase 1 (Design & Contracts)
