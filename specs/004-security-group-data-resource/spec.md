# Feature Specification: Security Group Data Source and Resource

**Feature Branch**: `004-security-group-data-resource`  
**Created**: December 14, 2025  
**Status**: Draft  
**Input**: User description: "設計 security group 的 data source 與 resource。"

## Clarifications

### Session 2025-12-14

- Q: When a security group that is attached to active VPS instances is deleted, should the platform behavior be: Allow deletion immediately and auto-detach from all instances (warning only), Block deletion with error requiring explicit detachment first, or Allow deletion with force flag option (default blocks, --force allows)? → A: Block deletion with error requiring explicit detachment first
- Q: When a security group has no rules defined (empty rule set), what should be the default traffic behavior? → A: Default deny all traffic
- Q: For security group rules, should ICMP support be: All ICMP types only (no granular control of ICMP type/code), All ICMP types with optional type/code specification for granular control, or Predefined ICMP types only (echo-request, echo-reply, unreachable)? → A: All ICMP types only (no granular control of ICMP type/code)
- Q: When multiple security groups are attached to a single VPS instance and rules overlap or conflict, how should rule evaluation work? → A: Union of all rules (most permissive wins - if any group allows, traffic is permitted)
- Q: Should security group rules be stateful (automatically allow return traffic for established connections) or stateless (require explicit rules for both directions)? → A: Stateful (return traffic automatically allowed for established connections)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create and Manage Security Groups with Rules (Priority: P1)

As an infrastructure engineer, I need to create and manage security groups with firewall rules through Terraform so that I can control network access to my VPS instances in a declarative and version-controlled manner.

**Why this priority**: This is the core functionality - the ability to create security groups and define access rules is essential for securing VPS instances and is the primary use case.

**Independent Test**: Can be fully tested by defining a security group resource with inbound/outbound rules in Terraform configuration, applying it, and verifying the security group exists in the ZillaForge platform with the correct rules.

**Acceptance Scenarios**:

1. **Given** a Terraform configuration with a security group resource, **When** I run `terraform apply`, **Then** a new security group is created in ZillaForge with the specified name and description
2. **Given** a security group resource with inbound rules, **When** I apply the configuration, **Then** the security group is created with the specified inbound rules (protocol, port range, source IP/CIDR)
3. **Given** a security group resource with outbound rules, **When** I apply the configuration, **Then** the security group is created with the specified outbound rules
4. **Given** an existing security group resource, **When** I add new rules to the configuration and apply, **Then** the new rules are added to the security group
5. **Given** an existing security group resource, **When** I remove rules from the configuration and apply, **Then** the rules are removed from the security group
6. **Given** an existing security group resource, **When** I modify rule attributes (port, protocol, source), **Then** Terraform replaces the rule with the updated configuration
7. **Given** an existing security group resource in Terraform state, **When** I run `terraform destroy`, **Then** the security group is removed from ZillaForge
8. **Given** a security group is attached to active instances, **When** I run `terraform plan` to delete it, **Then** Terraform displays a warning message about instances that will lose the security group protection

---

### User Story 2 - Query Existing Security Groups (Priority: P2)

As an infrastructure engineer, I need to query existing security groups through a data source so that I can reference them in my Terraform configurations without importing them as managed resources.

**Why this priority**: Data sources enable read-only access to existing security groups, which is important for referencing shared security groups or those created outside of Terraform.

**Independent Test**: Can be fully tested by creating a security group outside Terraform, then querying it via the data source and verifying all attributes including rules are correctly retrieved.

**Acceptance Scenarios**:

1. **Given** an existing security group in ZillaForge, **When** I query it by name using the data source, **Then** the security group's attributes and all rules are returned accurately
2. **Given** an existing security group in ZillaForge, **When** I query it by ID using the data source, **Then** the security group's details including rules are retrieved correctly
3. **Given** a non-existent security group name, **When** I query it using the data source, **Then** Terraform returns an appropriate error message
4. **Given** multiple security groups exist, **When** I query the data source without specifying name or ID filters, **Then** I can retrieve a list of all available security groups with their basic attributes
5. **Given** the data source is configured with both name and ID filters, **When** I query it, **Then** the system returns an error indicating only one filter should be specified
6. **Given** a queried security group, **When** the data source returns its rules, **Then** each rule includes protocol, port range, direction (inbound/outbound), and source/destination CIDR

---

### User Story 3 - Import Existing Security Groups (Priority: P3)

As an infrastructure engineer, I need to import manually-created security groups into Terraform management so that I can bring existing security configurations under version control.

**Why this priority**: While useful for migration scenarios, this is less critical than create/read operations and typically used during initial adoption or when adopting existing infrastructure.

**Independent Test**: Can be fully tested by creating a security group manually with rules, importing it into Terraform state, and verifying subsequent apply operations correctly detect drift.

**Acceptance Scenarios**:

1. **Given** a manually-created security group in ZillaForge, **When** I run `terraform import` with the security group ID, **Then** the security group is added to Terraform state with all attributes and rules
2. **Given** an imported security group in Terraform state, **When** I run `terraform plan`, **Then** Terraform shows no changes if the configuration matches the actual state
3. **Given** an imported security group with configuration drift (rules added/removed outside Terraform), **When** I run `terraform plan`, **Then** Terraform identifies the differences and proposes corrections

---

### User Story 4 - Reference Security Groups Between Resources (Priority: P2)

As an infrastructure engineer, I need to reference security groups in VPS instance configurations so that I can apply firewall rules to instances in a composable way.

**Why this priority**: This enables the primary use case of security groups - attaching them to instances to control network access.

**Independent Test**: Can be fully tested by creating a security group resource, referencing it in a VPS instance resource, applying the configuration, and verifying the instance has the security group attached.

**Acceptance Scenarios**:

1. **Given** a security group resource and a VPS instance resource, **When** I reference the security group ID in the instance configuration, **Then** the instance is created with the security group attached
2. **Given** an instance with an attached security group, **When** I add additional security groups to the instance configuration, **Then** the instance is updated to have multiple security groups attached
3. **Given** an instance with attached security groups, **When** I remove a security group reference from the configuration, **Then** the security group is detached from the instance

---

### Edge Cases

- **Duplicate security group name**: When a user tries to create a security group with a name that already exists in their account, the system returns an error immediately and does not create the security group
- **Invalid rule configuration**: When a user specifies an invalid rule (e.g., invalid port range, unsupported protocol), the system validates the rule and returns a clear error message specifying what is invalid
- **Conflicting rules**: When multiple security groups are attached to an instance with overlapping rules, the system evaluates rules as a union (most permissive wins) - if any attached security group allows the traffic, it is permitted
- **Deleted security group query**: When querying a security group that was deleted outside of Terraform, the data source returns a "not found" error
- **Concurrent modification**: When multiple requests attempt to modify the same security group's rules concurrently, the system handles conflicts using last-write-wins or optimistic locking
- **Security group in use by instances**: When a user deletes a security group that is attached to active VPS instances, the API blocks deletion with an error listing the attached instances; the user must explicitly detach the security group from all instances before deletion can proceed
- **Account security group limit**: When a user attempts to create a security group that would exceed their account quota, the system returns an error with the current count and maximum allowed
- **Non-existent import ID**: When importing a security group with an ID that doesn't exist, Terraform returns an error indicating the security group was not found
- **Empty security group**: When a security group has no rules defined, it functions as a default-deny firewall blocking all inbound and outbound traffic; users must explicitly add rules to allow any network access
- **Port range validation**: When specifying port ranges, the system validates that the start port is less than or equal to the end port, and both are within valid range (1-65535)
- **CIDR validation**: When specifying source/destination CIDR blocks, the system validates the CIDR notation format and returns errors for invalid formats
- **Rule limit per security group**: When adding rules exceeds the platform's per-security-group rule limit, the system returns an error indicating the limit and current count

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to create security groups with a unique name and optional description
- **FR-002**: System MUST allow users to define inbound rules specifying protocol, port range, and source CIDR or security group
- **FR-003**: System MUST allow users to define outbound rules specifying protocol, port range, and destination CIDR or security group
- **FR-004**: System MUST support common protocols including TCP, UDP, ICMP (all types without granular type/code control), and `any` (for any protocol)
- **FR-005**: System MUST validate port ranges to ensure start port ≤ end port and both are within 1-65535 range
- **FR-006**: System MUST validate CIDR notation format for source/destination specifications
- **FR-007**: System MUST block deletion of security groups that are attached to active VPS instances, returning an error with a list of attached instances; users MUST explicitly detach the security group from all instances before deletion can proceed
- **FR-008**: System MUST provide a data source to query individual security groups by name or ID, and support listing all security groups when no filters are specified
- **FR-009**: System MUST prevent creation of security groups with duplicate names within the same account by returning an error immediately
- **FR-010**: System MUST support importing existing security groups into Terraform state including all associated rules
- **FR-011**: System MUST track security group metadata including ID, name, description, and creation timestamp
- **FR-012**: System MUST handle security group resource lifecycle (create, read, update, delete) operations
- **FR-013**: System MUST return unique identifiers for each created security group
- **FR-014**: Data source MUST return error when querying non-existent security groups
- **FR-015**: System MUST support listing all security groups associated with the account
- **FR-016**: System MUST allow adding and removing rules from existing security groups without recreating the security group
- **FR-017**: System MUST support referencing security groups by ID when attaching to VPS instances
- **FR-018**: System MUST track which instances are using a security group for dependency management
- **FR-019**: System MUST validate rule configurations before applying them to prevent invalid firewall states
- **FR-020**: System MUST support both IPv4 and IPv6 CIDR blocks in rules
- **FR-021**: System MUST allow specifying individual ports or port ranges (e.g., port 80 or ports 8000-8100)
- **FR-022**: System MUST allow referencing other security groups as source/destination in rules for security group chaining *(Deferred to future enhancement - MVP supports CIDR-based rules only)*
- **FR-023**: Resource updates to description MUST be performed in-place without recreating the security group. Name changes MUST trigger resource replacement (ForceNew behavior).
- **FR-024**: Resource updates that require replacement (such as platform-specific immutable attributes) MUST be clearly indicated in Terraform plan output
- **FR-025**: Security groups with no rules defined MUST default to denying all inbound and outbound traffic (secure-by-default behavior)
- **FR-026**: When multiple security groups are attached to a VPS instance, rule evaluation MUST use union logic where traffic is permitted if ANY attached security group contains a matching allow rule (most permissive wins)
- **FR-027**: Security groups MUST operate in stateful mode, automatically allowing return traffic for connections initiated from allowed inbound or outbound rules without requiring explicit bidirectional rules

### Key Entities

- **Security Group**: Represents a named collection of firewall rules that can be applied to VPS instances; contains ID, name, description, creation timestamp, and associated rules
- **Security Rule**: Represents an individual firewall rule within a security group; contains direction (inbound/outbound), protocol, port range, source/destination specification (CIDR or security group reference); security groups are stateful, automatically allowing return traffic for established connections
- **VPS Instance**: Represents a virtual private server that can have one or more security groups attached to control its network access
- **CIDR Block**: Represents an IP address range in CIDR notation (e.g., 0.0.0.0/0 for all IPv4 addresses, specific subnet like 10.0.1.0/24)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create a security group with basic inbound rules (SSH on port 22 from specific IP) in under 2 minutes of configuration time
- **SC-002**: Security group data source successfully retrieves all security groups and their rules within 5 seconds for accounts with up to 100 security groups
- **SC-003**: Users can update security group rules (add/remove) without recreating the security group, with changes applied within 30 seconds
- **SC-004**: Import operation successfully brings existing security groups into Terraform state with 100% accuracy of all attributes and rules
- **SC-005**: 95% of security group operations (create, update, delete) complete successfully on first attempt without validation errors
- **SC-006**: Security group attachment to VPS instances completes within 10 seconds with network access rules taking effect immediately
- **SC-007**: Terraform plan accurately detects drift when security group rules are modified outside Terraform
- **SC-008**: Documentation enables new users to create their first security group with common rules (SSH, HTTP, HTTPS) without external support
