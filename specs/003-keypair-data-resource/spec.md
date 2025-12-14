# Feature Specification: Keypair Data Source and Resource

**Feature Branch**: `003-keypair-data-resource`  
**Created**: December 13, 2025  
**Status**: Draft  
**Input**: User description: "設計 keypair的data-source和 resource"

## Clarifications

### Session 2025-12-13

- Q: Should the data source support both single keypair queries AND listing all keypairs, or separate them? → A: Support both single queries AND list all in same data source with optional filters
- Q: How should deletion warnings for in-use keypairs be displayed in Terraform? → A: Terraform plan shows warning message, no blocking during apply
- Q: What happens when a user tries to create a keypair with a name that already exists? → A: Return error immediately, do not create keypair
- Q: How should private keys for system-generated keypairs be handled in Terraform state? → A: Mark as sensitive attribute in schema
- Q: What are the validation rules for keypair names (character restrictions, length limits)? → A: Defer to API validation

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create and Manage SSH Keypairs (Priority: P1)

As an infrastructure engineer, I need to create and manage SSH keypairs through Terraform so that I can securely access VPS instances without manually generating and uploading keys through a web interface.

**Why this priority**: This is the core functionality - the ability to create and manage keypairs is essential for VPS instance access and is the primary use case.

**Independent Test**: Can be fully tested by defining a keypair resource in Terraform configuration, applying it, and verifying the keypair exists in the ZillaForge platform with the correct properties.

**Acceptance Scenarios**:

1. **Given** a Terraform configuration with a keypair resource, **When** I run `terraform apply`, **Then** a new keypair is created in ZillaForge with the specified name
2. **Given** an existing keypair resource in Terraform state, **When** I run `terraform destroy`, **Then** the keypair is removed from ZillaForge
3. **Given** an existing keypair, **When** I modify non-updatable attributes in the configuration, **Then** Terraform indicates the resource must be replaced
4. **Given** a keypair resource with a provided public key, **When** I apply the configuration, **Then** the keypair is created with the specified public key
5. **Given** a keypair resource without a provided public key, **When** I apply the configuration, **Then** the system generates a keypair and returns both public and private keys
6. **Given** a keypair is in use by active instances, **When** I run `terraform plan` to delete it, **Then** Terraform displays a warning message about potential loss of access to those instances

---

### User Story 2 - Query Existing Keypairs (Priority: P2)

As an infrastructure engineer, I need to query existing keypairs through a data source so that I can reference them in my Terraform configurations without importing them as managed resources.

**Why this priority**: Data sources enable read-only access to existing keypairs, which is important for referencing shared keypairs or those created outside of Terraform.

**Independent Test**: Can be fully tested by creating a keypair outside Terraform, then querying it via the data source and verifying all attributes are correctly retrieved.

**Acceptance Scenarios**:

1. **Given** an existing keypair in ZillaForge, **When** I query it by name using the data source, **Then** the keypair's attributes are returned accurately
2. **Given** an existing keypair in ZillaForge, **When** I query it by ID using the data source, **Then** the keypair's details are retrieved correctly
3. **Given** a non-existent keypair name, **When** I query it using the data source, **Then** Terraform returns an appropriate error message
4. **Given** multiple keypairs exist, **When** I query the data source without specifying name or ID filters, **Then** I can retrieve a list of all available keypairs with their attributes
5. **Given** the data source is configured with both name and ID filters, **When** I query it, **Then** the system returns an error indicating only one filter should be specified

---

### User Story 3 - Import Existing Keypairs (Priority: P3)

As an infrastructure engineer, I need to import manually-created keypairs into Terraform management so that I can bring existing infrastructure under version control.

**Why this priority**: While useful for migration scenarios, this is less critical than create/read operations and typically used during initial adoption.

**Independent Test**: Can be fully tested by creating a keypair manually, importing it into Terraform state, and verifying subsequent apply operations correctly detect drift.

**Acceptance Scenarios**:

1. **Given** a manually-created keypair in ZillaForge, **When** I run `terraform import` with the keypair ID, **Then** the keypair is added to Terraform state with all attributes
2. **Given** an imported keypair in Terraform state, **When** I run `terraform plan`, **Then** Terraform shows no changes if the configuration matches the actual state
3. **Given** an imported keypair with configuration drift, **When** I run `terraform plan`, **Then** Terraform identifies the differences and proposes corrections

---

### Edge Cases

- **Duplicate keypair name**: When a user tries to create a keypair with a name that already exists in their account, the system returns an error immediately and does not create the keypair
- **Invalid public key format**: The system validates the public key format and returns a clear error message specifying the invalid format and accepted formats (RSA, ECDSA, ED25519)
- **Deleted keypair query**: When querying a keypair that was deleted outside of Terraform, the data source returns a "not found" error
- **Concurrent creation**: When multiple requests attempt to create keypairs with the same name concurrently, the first request succeeds and subsequent requests receive a duplicate name error
- **Keypair in use by instances**: When a user deletes a keypair that is used by active VPS instances, Terraform shows a warning in the plan output but allows the deletion to proceed
- **Account keypair limit**: When a user attempts to create a keypair that would exceed their account quota, the system returns an error with the current count and maximum allowed
- **Non-existent import ID**: When importing a keypair with an ID that doesn't exist, Terraform returns an error indicating the keypair was not found

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to create SSH keypairs with a unique name
- **FR-002**: System MUST support both user-provided public keys and system-generated keypairs
- **FR-003**: System MUST return the private key only once during creation for system-generated keypairs, and mark it as a sensitive attribute to prevent exposure in plan output or logs
- **FR-004**: System MUST allow users to delete keypairs they own
- **FR-005**: System MUST provide a data source to query individual keypairs by name or ID, and support listing all keypairs when no filters are specified
- **FR-006**: System MUST prevent creation of keypairs with duplicate names within the same account by returning an error immediately
- **FR-007**: System MUST validate public key format when users provide their own keys and return clear error messages for invalid formats
- **FR-008**: System MUST support importing existing keypairs into Terraform state
- **FR-009**: System MUST track keypair metadata including fingerprint
- **FR-010**: System MUST handle keypair resource lifecycle (create, read, update, delete) operations, where update is limited to description field only
- **FR-011**: System MUST return unique identifiers for each created keypair
- **FR-012**: Data source MUST return error when querying non-existent keypairs
- **FR-013**: System MUST support listing all keypairs associated with the account
- **FR-014**: Resource updates that require replacement MUST be clearly indicated in Terraform plan output
- **FR-015**: System MUST allow deletion of keypairs even when in use by instances, and display a warning in the Terraform plan output about potential loss of access to those instances
- **FR-016**: Warning messages for in-use keypair deletion MUST NOT block the apply operation
- **FR-017**: Keypair name validation MUST be enforced by the underlying API, with validation errors reported clearly to users

### Key Entities

- **Keypair**: Represents an SSH keypair with attributes including:
  - Unique identifier (ID)
  - User-defined name
  - Public key content
  - Private key content (only for system-generated, returned once, marked as sensitive)
  - Fingerprint (cryptographic hash of public key)
  - Owner/account association

- **Keypair Data Source**: Read-only reference to existing keypairs with flexible query modes:
  - Single keypair lookup by name (optional filter)
  - Single keypair lookup by ID (optional filter)
  - List all keypairs when no filters specified
  - Mutual exclusivity: name and ID filters cannot be used simultaneously

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create a new keypair and receive confirmation within 2 seconds
- **SC-002**: Users can query existing keypairs and receive results within 1 second
- **SC-003**: 100% of valid public keys are accepted, and 100% of invalid public keys are rejected with clear error messages
- **SC-004**: Users can complete the full lifecycle (create, read, update, delete) of a keypair within a single Terraform workflow
- **SC-005**: 95% of users successfully create and use keypairs without manual intervention or support
- **SC-006**: System-generated keypairs are cryptographically secure with minimum 2048-bit RSA or equivalent strength
- **SC-007**: Import operations complete within 3 seconds and preserve all keypair attributes

## Assumptions

- Users have basic knowledge of SSH keypairs and their purpose
- The ZillaForge API supports standard SSH key formats (RSA, ECDSA, ED25519)
- Public key validation follows OpenSSH format standards
- Keypair name validation rules (character restrictions, length limits) are enforced by the ZillaForge API
- Keypair names must be unique within an account scope, not globally
- System-generated keypairs use industry-standard key generation algorithms
- Private keys for user-provided public keys are never stored or accessible through the API
- Keypair deletion is immediate and cannot be recovered
- The data source queries use eventual consistency (changes may take a moment to reflect)
- Account-level keypair quotas are enforced by the underlying API
- Keypair fingerprints use standard SSH fingerprint format (MD5 or SHA256)
