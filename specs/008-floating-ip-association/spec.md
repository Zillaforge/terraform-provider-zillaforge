# Feature Specification: Floating IP Association with Network Attachments

**Feature Branch**: `008-floating-ip-association`  
**Created**: December 25, 2025  
**Status**: Draft  
**Input**: User description: "Associate/Disassociate a floating ip with a network_attachment in server"

## Clarifications

### Session 2025-12-25

- Q: When the cloud-sdk returns an error during floating IP association (e.g., floating IP already in use, server not ACTIVE), should Terraform retry the operation or fail immediately? → A: Fail immediately with descriptive error message (no retries)
- Q: What format should the `floating_ip_id` attribute accept in the network_attachment configuration? → A: String UUID format (e.g., "550e8400-e29b-41d4-a716-446655440000")
- Q: When a floating IP association/disassociation operation is in progress on the cloud platform, how should Terraform handle the operation? → A: Wait with polling until complete
- Q: When swapping floating IPs (changing `floating_ip_id` from one IP to another), should there be a brief moment where the network attachment has no floating IP, or should the transition be seamless? → A: Sequential: disassociate old, then associate new (brief window with no floating IP)
- Q: What should happen if a floating IP association fails midway through a server update operation that includes other changes (e.g., updating security groups)? → A: Partial update with error: apply successful changes, fail operation with clear error about floating IP

---

## Overview

This feature enables infrastructure engineers to associate and disassociate floating IPs (public IP addresses) with specific network attachments on VPS server instances through Terraform. Users can attach floating IPs to server network interfaces to provide public internet access, and detach them to reassign to different servers or release back to the pool. The implementation extends the existing server resource's network_attachment configuration.

---

## Assumptions

- Floating IP and Server resources already exist (from features 007 and 006 respectively)
- Each network_attachment on a server can have at most one floating IP associated
- A floating IP can only be associated with one network_attachment at a time
- Association changes are applied in-place without recreating the server or network attachment
- The cloud-sdk provides API methods via server.NICs() operations for associating/disassociating floating IPs with server network interfaces
- When a floating IP is associated, the server's network interface immediately gains external connectivity
- When a floating IP is disassociated, it becomes available for reassignment but remains allocated
- Floating IPs can only be associated with network attachments on servers in ACTIVE status
- Network attachments always have an internal IP address assigned
- Attempting to associate an already-associated floating IP results in an error from the cloud-sdk
- Disassociating a non-associated floating IP is a no-op that succeeds without error
- Association and disassociation operations are synchronous - Terraform waits with polling until the operation completes on the cloud platform
- When swapping floating IPs on a network_attachment, the old IP is disassociated completely before the new IP is associated, resulting in a brief window where the network_attachment has no floating IP
- If floating IP association fails during a server update with multiple changes, successful changes are applied and the operation fails with a clear error, allowing Terraform state to reflect actual infrastructure

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Associate Floating IP with Network Attachment (Priority: P1)

As an infrastructure engineer, I need to associate a floating IP with a specific network attachment on my server so that the server can receive incoming traffic from the public internet on that interface.

**Why this priority**: This is the core functionality - associating floating IPs with servers is the fundamental capability that provides public internet access to VPS instances.

**Independent Test**: Can be fully tested by adding a `floating_ip_id` attribute to a network_attachment block in an existing server resource, applying the configuration, and verifying the floating IP is associated and the server is accessible via the public IP.

**Acceptance Scenarios**:

1. **Given** an existing server with a network_attachment and an available floating IP, **When** I add `floating_ip_id` to the network_attachment configuration and apply, **Then** the floating IP is associated with that network interface
2. **Given** a server configuration with multiple network_attachments, **When** I specify different floating IPs for different attachments, **Then** each floating IP is associated with its respective network interface
3. **Given** a server with an associated floating IP, **When** I query the server attributes, **Then** the floating IP details are visible in the network_attachment's attributes
4. **Given** an attempt to associate a floating IP that is already associated elsewhere, **When** I apply the configuration, **Then** the system returns an error indicating the floating IP is already in use

---

### User Story 2 - Disassociate Floating IP from Network Attachment (Priority: P1)

As an infrastructure engineer, I need to disassociate a floating IP from a server's network attachment so that I can reassign the public IP to another server or release it back to the pool.

**Why this priority**: Disassociation is equally critical as association - users need to manage floating IP lifecycle by moving IPs between servers or freeing them when no longer needed.

**Independent Test**: Can be fully tested by removing the `floating_ip_id` attribute from a network_attachment block that has an associated floating IP, applying the configuration, and verifying the floating IP is disassociated and the public IP is no longer routed to the server.

**Acceptance Scenarios**:

1. **Given** a server with a floating IP associated to a network_attachment, **When** I remove the `floating_ip_id` attribute and apply, **Then** the floating IP is disassociated from the network interface
2. **Given** a server with a floating IP associated to a network_attachment, **When** I set `floating_ip_id` to null or empty string and apply, **Then** the floating IP is disassociated from the network interface
3. **Given** a disassociated floating IP, **When** I query the floating IP resource, **Then** the `device_id` attribute is null or empty
4. **Given** a server being destroyed with associated floating IPs, **When** I run `terraform destroy`, **Then** all floating IPs are disassociated before the server is terminated
5. **Given** a floating IP that is not associated, **When** I attempt to disassociate it from a server, **Then** the operation succeeds without error (idempotent)

---

### User Story 3 - Update Floating IP Association (Priority: P2)

As an infrastructure engineer, I need to change which floating IP is associated with a network attachment so that I can swap public IPs without recreating the server or network interface.

**Why this priority**: In-place updates for floating IP associations improve operational flexibility by allowing IP swaps without downtime. This is important but less critical than basic associate/disassociate operations.

**Independent Test**: Can be tested by modifying the `floating_ip_id` value in a network_attachment block from one floating IP to another, applying the configuration, and verifying the old IP is disassociated and the new IP is associated without recreating the attachment.

**Acceptance Scenarios**:

1. **Given** a server with a floating IP associated to a network_attachment, **When** I change the `floating_ip_id` to a different available floating IP and apply, **Then** the old IP is disassociated and the new IP is associated in-place
2. **Given** a floating IP reassignment operation, **When** Terraform applies the changes, **Then** the disassociation of the old IP completes before associating the new IP, with a brief window where no floating IP is associated
3. **Given** multiple servers sharing a pool of floating IPs, **When** I reassign IPs between servers, **Then** each association change is atomic and prevents conflicts

---

### User Story 4 - Associate Floating IP During Server Creation (Priority: P2)

As an infrastructure engineer, I need to associate a floating IP with a network attachment during server creation so that the server has public internet access immediately when it becomes active.

**Why this priority**: This streamlines server provisioning by enabling public connectivity in a single apply operation. It's a convenience feature that builds on the core association capability.

**Independent Test**: Can be tested by defining a new server resource with `floating_ip_id` specified in a network_attachment block, applying the configuration, and verifying the server is created with the floating IP already associated.

**Acceptance Scenarios**:

1. **Given** a new server resource configuration with `floating_ip_id` in a network_attachment, **When** I run `terraform apply`, **Then** the server is created and the floating IP is associated as part of the creation process
2. **Given** a new server with multiple network_attachments each specifying floating IPs, **When** I apply the configuration, **Then** all floating IPs are associated to their respective interfaces during creation
3. **Given** a server creation with invalid floating IP ID, **When** I attempt to apply, **Then** the system returns an error before creating the server

---

### Edge Cases

- What happens when a floating IP is deleted outside Terraform while still associated with a server? → Terraform detects the inconsistency on next refresh and updates state; user is prompted to reconcile
- How does the system handle association when the server is not in ACTIVE status? → Operation fails with clear error message indicating server must be ACTIVE; user must wait or troubleshoot server status
- What happens if two network_attachments on the same server attempt to use the same floating IP? → Validation error prevents configuration from being applied; only one attachment per floating IP allowed
- How does the system handle concurrent association attempts for the same floating IP? → Cloud API enforces mutual exclusion; the second operation fails with "floating IP already in use" error
- What happens when a floating IP is associated but the network_attachment is removed from the server? → The floating IP is automatically disassociated as part of the network_attachment removal operation
- What happens if the floating IP pool is exhausted when creating a server with floating IP? → Server creation proceeds but floating IP association fails; error indicates insufficient IPs; user must retry or use existing IP
- What happens during the brief window when swapping floating IPs (after old IP disassociated, before new IP associated)? → The server temporarily has no public IP on that interface; existing connections may be disrupted; the window duration is minimized by sequential operations
- What happens if floating IP association fails during a server update with multiple changes (e.g., updating security groups)? → Successful changes (e.g., security group updates) are applied; operation fails with error identifying the floating IP issue; Terraform state reflects partial update; user can fix floating IP issue and re-apply

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to associate a floating IP with a network_attachment by specifying `floating_ip_id` attribute in the network_attachment block
- **FR-002**: System MUST allow users to disassociate a floating IP from a network_attachment by removing or nullifying the `floating_ip_id` attribute
- **FR-003**: System MUST enforce that each network_attachment can have at most one floating IP associated at any time
- **FR-004**: System MUST enforce that each floating IP can only be associated with one network_attachment at any time
- **FR-005**: System MUST validate that the floating_ip_id is a valid UUID format and that the floating IP exists before attempting association
- **FR-006**: System MUST allow in-place updates when changing the `floating_ip_id` value, performing sequential operations: disassociate the old IP completely, then associate the new IP, without recreating the network_attachment
- **FR-007**: System MUST automatically disassociate all floating IPs from a server's network_attachments when the server is destroyed
- **FR-008**: System MUST update the floating IP's `device_id` attribute to reflect the server ID when associated, and null/empty when disassociated
- **FR-009**: System MUST provide clear error messages when association fails due to: floating IP already in use, invalid floating IP ID, or server not in ACTIVE status
- **FR-010**: Association operations MUST fail immediately with descriptive error messages without automatic retries
- **FR-011**: System MUST wait with polling for association/disassociation operations to complete before returning control to Terraform
- **FR-012**: System MUST support associating floating IPs with network_attachments during server creation as an atomic operation
- **FR-013**: Disassociation operations MUST be idempotent - repeated disassociation with the same parameters succeeds without errors
- **FR-014**: When floating IP association fails during a server update with multiple changes, the system MUST apply successful changes, fail the operation with a clear error, and ensure Terraform state reflects the actual infrastructure state

### Key Entities

- **Floating IP**: Public IP address resource that can be associated with network attachments; has attributes: id (string UUID), ip_address, status, device_id, name, description
- **Network Attachment**: Network interface configuration on a server; has attributes: network_id, ip_address, primary, sg_ids, and now floating_ip_id (string UUID format); represents the binding point for floating IP association
- **Server**: VPS instance that contains one or more network_attachments; has attributes: id, name, status, network_attachment blocks

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can associate a floating IP with a server's network attachment in under 30 seconds from applying the Terraform configuration
- **SC-002**: Users can disassociate a floating IP from a server's network attachment in under 15 seconds from applying the Terraform configuration
- **SC-003**: Users can update floating IP associations (swap IPs) in-place without recreating the network_attachment or server
- **SC-004**: 100% of floating IP disassociation operations are idempotent - repeated disassociation with the same parameters succeeds without errors
- **SC-005**: Users can create a server with floating IPs pre-associated to network_attachments in a single Terraform apply operation
- **SC-006**: Error messages for failed associations clearly identify the reason (IP in use, invalid ID, server status) within 5 seconds
- **SC-007**: Terraform state accurately reflects floating IP associations after any apply, refresh, or destroy operation
