# Feature Specification: Floating IP Resource and Data Source

**Feature Branch**: `007-floating-ip-resource`  
**Created**: December 24, 2025  
**Status**: Draft  
**Input**: User description: "設計 floating ip resource 與 data source. 透過cloud-sdk存取"

## Clarifications

### Session 2025-12-24

- Q: What are all possible status values for a floating IP? → A: ACTIVE, DOWN, PENDING, REJECTED
- Q: What details should be included in pool exhaustion error messages? → A: Error message from API as-is
- Q: Can device_id be null/empty for unassociated floating IPs? → A: Yes, null/empty when not associated

---

## Overview

This feature provides Terraform resource and data source support for managing floating IPs (public IP addresses) in ZillaForge. Users can allocate floating IPs, release them when no longer needed, and query existing floating IPs by various attributes. Association of floating IPs with VPS instances is out of scope for this feature. The implementation uses the ZillaForge cloud-sdk for all API interactions.

---

## Assumptions

- Currently there is only one floating IP pool; users allocate from this pool without specifying a pool name
- Multiple IP pools may be supported in the future, but this is not in scope for this feature
- Floating IPs are billable resources that persist independently of VPS instances
- The cloud-sdk provides VPS floating IP management APIs following RESTful patterns with direct attribute access
- SDK responses include all floating IP attributes: id, name, description, status, device_id
- Status attribute has four possible values: ACTIVE (IP is active and operational), DOWN (IP is allocated but not operational), PENDING (IP allocation or operation is in progress), REJECTED (IP allocation or operation was rejected)
- Only name and description attributes are modifiable after allocation
- Floating IP addresses are globally unique within the platform
- Standard TCP/IP address validation rules apply
- device_id indicates which device (if any) the floating IP is associated with; this association is managed outside of Terraform
- device_id is null or empty when the floating IP is not associated with any device
- Association of floating IPs with VPS instances will be handled outside of Terraform or in a future feature

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Allocate and Manage Floating IPs (Priority: P1)

As an infrastructure engineer, I need to allocate floating IPs through Terraform so that I can obtain public IP addresses for my VPS instances and manage them as infrastructure-as-code.

**Why this priority**: This is the core functionality - the ability to allocate floating IPs is fundamental for providing public internet access to VPS instances.

**Independent Test**: Can be fully tested by defining a floating IP resource in Terraform configuration with optional name and description, applying it, and verifying the floating IP is allocated in ZillaForge with a unique public IP address.

**Acceptance Scenarios**:

1. **Given** a Terraform configuration with a floating IP resource, **When** I run `terraform apply`, **Then** a new floating IP is allocated with a unique public IP address
2. **Given** a Terraform configuration with a floating IP resource specifying name and description, **When** I apply it, **Then** the floating IP is created with the specified name and description
3. **Given** an existing floating IP resource in Terraform state, **When** I run `terraform destroy`, **Then** the floating IP is released back to the pool
4. **Given** a floating IP allocation request when the pool is exhausted, **When** I apply the configuration, **Then** the system returns an error indicating insufficient IP addresses available
5. **Given** multiple floating IP resources, **When** I apply the configuration, **Then** each floating IP is allocated with a unique IP address
6. **Given** an allocated floating IP, **When** queried via cloud-sdk or platform UI, **Then** the floating IP attributes (id, name, description, status, device_id) are accessible

---

### User Story 2 - Query Existing Floating IPs (Priority: P2)

As an infrastructure engineer, I need to query existing floating IPs through a data source so that I can reference them in my Terraform configurations without importing them as managed resources.

**Why this priority**: Data sources enable read-only access to floating IPs, which is important for referencing shared IPs or those created outside of Terraform. This is less critical than allocation and association.

**Independent Test**: Can be fully tested by creating a floating IP outside Terraform, then querying it via the data source by IP address or ID and verifying all attributes are correctly retrieved.

**Acceptance Scenarios**:

1. **Given** an existing floating IP in ZillaForge, **When** I query it by ID using the data source, **Then** the floating IP's attributes (id, name, description, status, device_id, ip_address) are returned accurately as a list
2. **Given** an existing floating IP, **When** I query it by IP address using the data source, **Then** the floating IP's details are retrieved correctly as a list
3. **Given** an existing floating IP, **When** I query it by name using the data source, **Then** the floating IP's details are retrieved correctly as a list
4. **Given** multiple floating IPs exist, **When** I query the data source without filters, **Then** I can retrieve a list of all available floating IPs with their attributes
5. **Given** the data source is configured with status filter (e.g., "ACTIVE", "DOWN", "PENDING", or "REJECTED"), **When** I query it, **Then** only floating IPs matching that status are returned as a list
6. **Given** a non-existent floating IP ID or name, **When** I query it using the data source, **Then** the data source returns an empty list

---

### User Story 3 - Update Floating IP Attributes (Priority: P3)

As an infrastructure engineer, I need to update certain floating IP attributes (name and description) through Terraform so that I can maintain accurate identification and documentation without recreating the resource.

**Why this priority**: In-place updates for metadata improve operational efficiency but are less critical than core allocation operations. Most users will set these once during creation.

**Independent Test**: Can be tested by modifying the name or description attributes in the Terraform configuration, applying the changes, and verifying the floating IP is updated without reallocation.

**Acceptance Scenarios**:

1. **Given** an existing floating IP, **When** I modify the name in Terraform, **Then** the floating IP is updated in-place without reallocation
2. **Given** an existing floating IP, **When** I modify the description in Terraform, **Then** the floating IP is updated in-place without reallocation
3. **Given** an existing floating IP, **When** I modify both name and description in Terraform, **Then** both attributes are updated in-place without reallocation

---

### User Story 4 - Import Existing Floating IPs (Priority: P3)

As an infrastructure engineer, I need to import manually-allocated floating IPs into Terraform management so that I can bring existing public IPs under version control.

**Why this priority**: Import functionality supports migration scenarios but is less critical than creation and query operations. Typically used during initial adoption of infrastructure-as-code.

**Independent Test**: Can be fully tested by allocating a floating IP manually, importing it into Terraform state using the floating IP ID, and verifying subsequent apply operations correctly detect drift.

**Acceptance Scenarios**:

1. **Given** a manually-allocated floating IP in ZillaForge, **When** I run `terraform import` with the floating IP ID, **Then** the floating IP is added to Terraform state with all attributes
2. **Given** an imported floating IP in Terraform state, **When** I run `terraform plan` with matching configuration, **Then** Terraform shows no changes
3. **Given** an imported floating IP with configuration drift, **When** I run `terraform plan`, **Then** Terraform identifies the differences and proposes corrections

---

### Edge Cases

- **Pool exhaustion**: When a user attempts to allocate a floating IP when no available addresses remain, the system returns an error using the message from the API response as-is
- **Deleting allocated floating IP**: When a user attempts to destroy a floating IP, Terraform releases it back to the pool regardless of whether it's in use (device_id is set) on the platform, since association management is handled outside Terraform
- **Concurrent allocation**: When multiple requests attempt to allocate floating IPs concurrently, each request either succeeds with a unique IP or fails if the pool is exhausted
- **Quota limits**: When a user attempts to allocate a floating IP that would exceed their account quota, the system returns an error with current usage and maximum allowed
- **Empty query results**: When querying by name, IP address, status, or ID that doesn't match any floating IPs, the data source returns an empty list (not an error)
- **Duplicate names**: When a user creates a floating IP with a name that matches an existing floating IP, the system allows creation (names are not required to be unique)
- **Floating IP lifecycle outside Terraform**: When a floating IP managed by Terraform is associated with an instance outside of Terraform (device_id is set), the next `terraform apply` detects no configuration changes since association is not tracked by Terraform
- **device_id attribute**: The device_id attribute is read-only and reflects associations made outside of Terraform; device_id is null/empty when the floating IP is unassociated; changes to device_id outside Terraform do not cause drift in Terraform state since it's a computed attribute
- **Status attribute**: The status attribute (ACTIVE, DOWN, PENDING, REJECTED) is informational and reflects the operational state; all status values are valid states that should be stored in Terraform state

---

## Requirements *(mandatory)*

### Functional Requirements

#### Resource Requirements

- **FR-001**: System MUST allow users to allocate floating IPs through the `zillaforge_floating_ip` resource without specifying a pool (allocated from the single available pool)
- **FR-002**: System MUST support optional user-provided attributes: name, description
- **FR-003**: System MUST support in-place updates for: name, description
- **FR-004**: System MUST release floating IPs back to the pool during destroy operations
- **FR-005**: System MUST support importing existing floating IPs by floating IP ID
- **FR-006**: System MUST retrieve and expose computed attributes directly from SDK response: id, ip_address, status, device_id
- **FR-007**: System MUST mark device_id as a computed read-only attribute since association is managed outside Terraform; device_id is null/empty when unassociated
- **FR-008**: System MUST use the ZillaForge cloud-sdk client initialized by the provider for all API calls
- **FR-009**: System MUST handle API errors gracefully and return Terraform diagnostics with actionable error messages
- **FR-010**: System MUST return error messages from the API as-is when pool exhaustion occurs
- **FR-011**: System MUST return detailed error messages when quota limits are exceeded, including current usage and maximum limits

#### Data Source Requirements

- **FR-012**: System MUST provide a `zillaforge_floating_ips` data source for querying floating IPs (always returns a list)
- **FR-013**: Data source MUST support optional filters: name, ip_address, status, id
- **FR-014**: Data source MUST return results as a list attribute named `floating_ips` even when filtering by single ID
- **FR-015**: Data source MUST return computed attributes directly from SDK response: id, name, description, ip_address, status, device_id
- **FR-016**: Data source MUST return an empty list when no floating IPs match filter criteria (not an error)
- **FR-017**: Data source MUST allow multiple filters to be specified simultaneously (AND logic)
- **FR-018**: Data source schema MUST mark all result attributes as Computed
- **FR-019**: Data source attribute descriptions MUST be documented with MarkdownDescription for documentation generation
- **FR-020**: Data source MUST return floating IPs sorted deterministically by ID for consistent Terraform state management

### Key Entities

- **Floating IP**: Represents a public IP address resource that can be allocated. Key attributes include: unique identifier (id), optional name, optional description, IP address (ip_address), status (ACTIVE, DOWN, PENDING, or REJECTED), device identifier (device_id) indicating association with a device (null/empty when unassociated). Only name and description are modifiable. Association with VPS instances (reflected in device_id) is handled outside of Terraform.
- **IP Pool**: Represents the single collection of public IP addresses managed by the platform. Pool selection is automatic during allocation. Users cannot create or modify pools through Terraform.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can allocate a floating IP in under 15 seconds
- **SC-002**: 100% of floating IP allocations return a unique, valid public IP address
- **SC-003**: Users can query floating IPs by name, IP address, status, or ID and retrieve complete attribute information (id, name, description, ip_address, status, device_id)
- **SC-004**: 95% of floating IP operations succeed on first attempt without retry
- **SC-005**: Users can update name and description in-place without recreating the floating IP
- **SC-006**: Import operations correctly retrieve all floating IP attributes from SDK response
- **SC-007**: Error messages for pool exhaustion use API response messages as-is; quota limit errors include specific counts and limits

---

## Out of Scope

- Associating or disassociating floating IPs with VPS instances (managed outside Terraform or future feature)
- Managing device_id attribute (read-only, reflects associations made outside Terraform)
- Creating or managing custom IP pools (platform-managed only)
- Selecting specific IP pools during allocation (single pool used automatically)
- Supporting multiple IP pools (future enhancement)
- IPv6 floating IP support (defer to future if needed)
- Floating IP reservations or advanced scheduling
- Bulk floating IP operations beyond Terraform's native parallelism
- Port forwarding or NAT configuration (separate feature if needed)
- Floating IP pricing or billing information in Terraform state

---

## Dependencies

- ZillaForge cloud-sdk with VPS floating IP management APIs providing direct attribute access (id, name, description, status, device_id, ip_address)
- Provider authentication and project context configuration
- Single IP pool pre-configuration at the platform level

---

## Notes

*The floating IP resource follows patterns established by keypair and server resources in the provider. This feature focuses solely on floating IP lifecycle management (allocate, query, release, update metadata). The SDK provides all floating IP attributes directly in responses (id, name, description, ip_address, status, device_id), with only name and description being modifiable. Association with VPS instances (reflected in device_id) will be handled through platform UI, CLI, or API outside of Terraform, or may be added as a separate feature in the future. The current implementation uses a single IP pool with potential for multi-pool support in future iterations.*
