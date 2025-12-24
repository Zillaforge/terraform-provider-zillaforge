# Feature Specification: Server Resource (VPS Virtual Machine)

**Feature Branch**: `006-server-resource`  
**Created**: December 16, 2025  
**Status**: Draft  
**Input**: User description: "Design server resource, which is virtual machine in zillaforge"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create and Configure VPS Instances (Priority: P1)

As an infrastructure engineer, I need to create and configure VPS (Virtual Private Server) instances through Terraform so that I can provision virtual machines with specified compute resources, storage, and network settings for my applications.

**Why this priority**: This is the core functionality - the ability to create VPS instances is the fundamental use case for the server resource and the primary reason users interact with the provider.

**Independent Test**: Can be fully tested by defining a server resource in Terraform configuration with required attributes (name, flavor (by ID), image (by ID), network (with per-NIC `sg_ids`)), applying it, and verifying the VPS instance is active in ZillaForge with the correct specifications.

**Acceptance Scenarios**:

1. **Given** a Terraform configuration with a server resource specifying name, flavor, image, and network (each network_attachment having `sg_ids`), **When** I run `terraform apply`, **Then** a new VPS instance is created in ZillaForge with the specified configuration
2. **Given** a server resource configuration with optional attributes (keypair, user data), **When** I apply the configuration, **Then** the VPS instance is created with those optional settings applied
3. **Given** an existing server resource in Terraform state, **When** I run `terraform destroy`, **Then** the VPS instance is terminated and removed from ZillaForge
4. **Given** a server configuration with multiple network interfaces, **When** I apply the configuration, **Then** the VPS instance is created with all specified network attachments


> Example Terraform snippet:
>
> resource "zillaforge_server" "example" {
>   name = "web-01"
>   flavor = data.zillaforge_flavors.available.flavors[0].id
>   image = data.zillaforge_images.ubuntu.images[0].id
>   network_attachment {
>     network_id = "net-123"
>     ip_address = "10.0.0.10"
>     sg_ids = ["default", "web-access"]
>     primary = true
>   }
>
>   network_attachment {
>     network_id = "net-456"
>   }
> }
5. **Given** a server configuration with user data scripts, **When** the instance is provisioned, **Then** the user data is executed during instance initialization

---

### User Story 2 - Update Instance Configuration (Priority: P2)

As an infrastructure engineer, I need to update certain VPS instance attributes (such as name, description, network attachments, and security groups) through Terraform so that I can adjust configurations without recreating the instance.

**Why this priority**: In-place updates improve operational efficiency by avoiding downtime from instance replacement. This is essential for maintaining configurations without disrupting running services.

**Independent Test**: Can be tested by modifying updateable attributes (name, description, network attachments, security groups) in the Terraform configuration, applying the changes, and verifying the instance is updated without replacement.

**Acceptance Scenarios**:

1. **Given** an existing VPS instance, **When** I modify the instance name in Terraform, **Then** the instance is updated in-place without recreation
2. **Given** an existing VPS instance, **When** I modify the instance description in Terraform, **Then** the instance is updated in-place without recreation
3. **Given** an existing VPS instance, **When** I add or remove network attachments in the configuration, **Then** the network attachments are updated in-place without recreating the instance
4. **Given** an existing VPS instance, **When** I add or remove security groups in the configuration, **Then** the security group associations are updated in-place without recreating the instance
5. **Given** an existing VPS instance, **When** I modify non-updateable attributes (flavor, image), **Then** Terraform indicates the resource must be replaced
6. **Given** an existing VPS instance, **When** I attempt to modify the flavor (resize), **Then** Terraform indicates that flavor resize is out of scope for this feature and must be performed manually on the platform or by recreating the instance
7. **Given** an instance update that requires replacement, **When** I run `terraform plan`, **Then** Terraform clearly indicates which attribute changes force recreation

---

### User Story 3 - Asynchronous Server Creation (Priority: P3)

As a platform engineer managing large-scale deployments, I need to create multiple server instances without waiting for each to become active so that I can deploy infrastructure faster when external orchestration handles readiness verification.

**Why this priority**: This is an optimization feature for advanced use cases (batch deployments, autoscaling groups, external orchestration). Most users will use the default synchronous behavior, but this flexibility is valuable for high-volume scenarios.

**Independent Test**: Can be tested by setting `wait_for_active = false` in a server resource configuration, applying it, and verifying Terraform returns immediately after the API call without polling for active status.

**Acceptance Scenarios**:

1. **Given** a server resource with `wait_for_active = false`, **When** I run `terraform apply`, **Then** Terraform returns immediately after the create API call without waiting for "active" status
2. **Given** a server resource with `wait_for_active = true` (default), **When** I run `terraform apply`, **Then** Terraform waits for the server to reach "active" status before completion
3. **Given** a server resource with `wait_for_active = false`, **When** creation completes, **Then** the server status may be "building" in the Terraform state
4. **Given** multiple servers with `wait_for_active = false`, **When** deploying via Terraform, **Then** all servers are created concurrently without sequential waiting
5. **Given** a server with `wait_for_active = false` that enters error state, **When** I run `terraform refresh`, **Then** Terraform updates the state with the current error status

---

### User Story 3.1 - Asynchronous Server Deletion (Priority: P3)

As a platform engineer managing large-scale deployments, I need to delete multiple server instances without waiting for each to be fully removed so that I can tear down infrastructure faster when external orchestration handles cleanup verification.

**Why this priority**: This is an optimization feature similar to asynchronous creation, useful for batch operations and external orchestration. Most users will use the default synchronous behavior to ensure resources are fully cleaned up.

**Independent Test**: Can be tested by setting `wait_for_deleted = false` in a server resource configuration, destroying it with `terraform destroy`, and verifying Terraform returns immediately after the delete API call without polling for deletion completion.

**Acceptance Scenarios**:

1. **Given** a server resource with `wait_for_deleted = false`, **When** I run `terraform destroy`, **Then** Terraform returns immediately after the delete API call without waiting for the server to be fully removed
2. **Given** a server resource with `wait_for_deleted = true` (default), **When** I run `terraform destroy`, **Then** Terraform waits for the server to be fully deleted before completion
3. **Given** a server resource with `wait_for_deleted = false`, **When** deletion is initiated, **Then** Terraform state is immediately removed without confirming the server is gone
4. **Given** multiple servers with `wait_for_deleted = false`, **When** destroying via Terraform, **Then** all servers are deleted concurrently without sequential waiting
5. **Given** a server with custom delete timeout, **When** `wait_for_deleted = true`, **Then** Terraform respects the configured timeout value

---

### User Story 4 - Import Existing VPS Instances (Priority: P3)

As an infrastructure engineer, I need to import manually-created VPS instances into Terraform management so that I can bring existing virtual machines under version control and infrastructure-as-code practices.

**Why this priority**: Import functionality supports migration scenarios and is important for Terraform adoption but is less critical than creation and update operations. Typically used during initial migration to infrastructure-as-code.

**Independent Test**: Can be tested by creating a VPS instance manually through the ZillaForge console, importing it into Terraform state using the instance ID, and verifying subsequent plan operations correctly detect configuration drift.

**Acceptance Scenarios**:

1. **Given** a manually-created VPS instance in ZillaForge, **When** I run `terraform import` with the instance ID, **Then** the instance is added to Terraform state with all attributes
2. **Given** an imported instance in Terraform state, **When** I run `terraform plan` with matching configuration, **Then** Terraform shows no changes
3. **Given** an imported instance with configuration drift, **When** I run `terraform plan`, **Then** Terraform identifies the differences and proposes corrections
4. **Given** an invalid instance ID, **When** I attempt to import, **Then** Terraform returns a clear error message indicating the instance was not found

---

### Edge Cases

- **Invalid flavor/image combination**: When a user specifies a flavor and image that are incompatible (e.g., Windows image with flavor that doesn't support Windows), the system returns a validation error before attempting creation
- **Insufficient quota**: When a user attempts to create an instance that would exceed their account quota for CPU, memory, or instances, the system returns an error with current usage and limits
- **Network unavailable**: When specified network IDs are invalid or inaccessible, the instance creation fails with a clear error identifying the problematic network
- **Keypair reference invalid**: When a referenced keypair does not exist, the instance creation fails with an error indicating the missing keypair
- **Instance stuck in transitional state**: When an instance remains in a transitional state (e.g., "building") beyond the timeout period, Terraform reports the timeout with the last known state
- **Concurrent modification**: When an instance is modified outside of Terraform between plan and apply, Terraform detects the drift during apply and reports the conflict
- **Volume attachment limits**: When user attempts to attach more volumes than the flavor supports, the system returns an error with the maximum allowed attachments
- **Name conflicts**: When a user specifies an instance name that conflicts with an existing instance, the system allows creation but may append a unique identifier depending on platform requirements

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to create VPS instances with required attributes: name, flavor (instance type ID), image (operating system ID), network configuration, and security_groups (list of security group IDs). Flavor and image MUST be specified using platform IDs (not human-readable names).
- **FR-002**: System MUST support optional attributes: description, keypair, and user data
- **FR-003**: Users MUST be able to specify multiple network interfaces using the `network_attachment` nested block (fields: `network_id` required, optional `ip_address`, optional `primary`). The system MUST validate that specified `network_id`s exist and that at most one attachment is marked `primary`.
- **FR-004**: System MUST validate that referenced resources (flavors, images, networks, keypairs, security groups) exist before attempting instance creation
- **FR-005**: System MUST create instances in the project specified in the provider configuration
- **FR-006**: System MUST support instance lifecycle operations: create, read, update, delete
- **FR-007**: Users MUST be able to provide user data (cloud-init scripts) for instance initialization
- **FR-008**: System MUST mark sensitive attributes (user data containing secrets, admin passwords) as sensitive in Terraform state
- **FR-009**: System MUST support configurable wait behavior via `wait_for_active` attribute (Boolean, default: true). When true, system waits for instance to reach "active" state after creation. When false, system returns immediately after API responds without polling for status.
- **FR-009-A**: System MUST support configurable delete wait behavior via `wait_for_deleted` attribute (Boolean, default: true). When true, system waits for instance to be fully deleted after delete API call. When false, system returns immediately after delete API call without polling for deletion completion.
- **FR-010**: System MUST support updating in-place for: **name** and **description**. Support for in-place updates of network attachments and security groups is planned but **not yet implemented**; such changes will surface a TODO warning during apply until implemented.
- **FR-011**: System MUST force resource replacement when non-updateable attributes are changed: **image**. Changes to **flavor** constitute a platform 'resize' operation and are out of scope for this feature; users must perform resize manually on the platform or recreate the instance if a different flavor is required. Attempts to change `flavor_id` or `image_id` during an update should return a clear error explaining these operations are not supported in-place.
- **FR-012**: System MUST implement timeout handling for long-running operations (create, update, delete) with configurable timeout values
- **FR-013**: System MUST support importing existing VPS instances by instance ID
- **FR-014**: System MUST retrieve and expose computed attributes: instance ID, IP addresses, status, creation timestamp
- **FR-015**: System MUST return detailed error messages when quota limits are exceeded, including current usage and maximum limits
- **FR-016**: System MUST validate flavor and image compatibility before instance creation
- **FR-017**: System MUST handle instance state transitions gracefully, waiting for completion of in-progress operations

### Key Entities

- **Server/Instance**: A virtual machine in ZillaForge with compute, memory, storage, and network resources. Key attributes include: unique identifier, name, flavor (compute specifications), image (OS template), network attachments, security groups, keypair, power state, IP addresses, and creation timestamp
- **Flavor**: Instance type defining CPU count, memory size, disk size, and other compute specifications. Referenced by ID or name when creating server instances
- **Image**: Operating system template or snapshot used to initialize instance root disk. Referenced by ID or name during instance creation
- **Network**: Virtual network to which instance network interfaces attach. Multiple networks can be specified for multi-homed instances

**Network Attachment (Terraform schema)**: Represented by a nested block `network_attachment` (list) with fields:

- `network_id` (String, required): ID of the network to attach
- `ip_address` (String, optional): Fixed IPv4 address to request on that network
- `primary` (Bool, optional): Whether this attachment is the primary interface (one `primary` true allowed)

This block supports multiple attachments, fixed IP assignment, and explicit primary interface selection.
- **Keypair**: SSH key pair for secure instance access. Referenced by name and injected into instance during creation
- **Security Group**: Firewall ruleset controlling inbound and outbound network traffic. Multiple security groups can be associated with an instance
- **User Data**: Initialization script (typically cloud-init format) executed when instance first boots

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can provision a basic VPS instance (name + flavor + image + network + security_groups) in under 5 minutes from `terraform apply` completion
- **SC-002**: Terraform correctly detects and reports configuration drift when instances are modified outside of Terraform
- **SC-003**: 100% of instance create operations correctly wait for instance to reach "active" state before completing
- **SC-004**: Import operations successfully reconstruct instance configuration for all supported attributes
- **SC-005**: Users can update in-place attributes (name, description, network attachments, security groups) without instance downtime
- **SC-006**: System correctly identifies and prevents invalid configurations (incompatible flavor/image combinations, missing dependencies) before API calls
- **SC-007**: Error messages for quota exhaustion include actionable information (current usage, limits, suggested resolution)

## Assumptions

- **AS-001**: ZillaForge API provides synchronous or poll-able instance creation with status endpoints for monitoring operation progress
- **AS-002**: Default timeout for instance creation is 10 minutes, configurable via resource timeouts block
- **AS-003**: The platform supports cloud-init or equivalent user data injection mechanism
- **AS-004**: Instances can have multiple network interfaces attached to different networks
- **AS-005**: Security group associations can be modified on active instances without requiring reboot
- **AS-006**: Instance names are not required to be globally unique (uniqueness enforced at project level if at all)
- **AS-007**: Flavors and images are managed outside this resource and queried via data sources
- **AS-008**: Instance IP addresses are assigned by the platform (DHCP or allocation pool) unless static IPs are explicitly configured
- **AS-009**: Changing an instance flavor is a platform 'resize' operation and is out of scope for this feature; users must perform a manual resize on the platform or recreate the instance to change flavor
- **AS-010**: Admin/root passwords are generated by the platform and retrievable via API if not using keypair authentication

## Clarifications

### Session 2025-12-17

- Q: How should network attachments be modeled in Terraform? → A: Option B — use a nested `network_attachment` block with fields `network_id` (required), `ip_address` (optional), `primary` (optional boolean).

> Note: Implementation detail — use a `network_attachment` list-of-blocks to support multiple interfaces, optional fixed IPs, and explicit primary interface designation. This maps cleanly to import and state handling.

- Q: How should security group references be modeled in the Terraform resource? → A: `sg_ids` nested inside each `network_attachment` — This models which NIC each security group applies to and avoids ambiguity about target interface.

- Q: Should the `status` attribute be user-settable or computed-only? → A: Computed-only — The `status` attribute should be computed (read-only) since the platform manages instance state and users cannot directly set it (aligns with FR-014 exposing status as a computed attribute).

- Q: What should the default timeout be for instance creation operations? → A: 10 minutes — Aligns with AS-002 assumption and provides reasonable buffer for typical instance provisioning operations while being configurable via Terraform timeouts block.

## Dependencies

- **DEP-001**: Requires `zillaforge_flavors` data source to validate and reference instance types
- **DEP-002**: Requires `zillaforge_images` data source to validate and reference OS templates
- **DEP-003**: Requires `zillaforge_networks` data source to validate and reference network configurations
- **DEP-004**: Requires `zillaforge_keypairs` resource/data source to reference SSH keys (optional dependency)
- **DEP-005**: Requires `zillaforge_security_groups` resource/data source to reference firewall rules (optional dependency)
- **DEP-006**: Depends on ZillaForge API SDK for instance lifecycle management operations
- **DEP-007**: Requires provider configuration with valid API credentials and project identifier

## Scope

### In Scope

- Creating VPS instances with compute, network, and storage configuration
- Managing instance lifecycle (create, read, update, delete)
- Network interface management (multiple networks)
- Security group associations
- Keypair assignment for SSH access
- User data injection for initialization scripts
- Import of existing instances
- In-place updates for mutable attributes (name, description, network attachments, security groups)
- Forced replacement for immutable attributes (flavor, image)
- Timeout configuration for long-running operations
- Quota validation and error reporting

### Out of Scope

- Power state management (start, stop, reboot operations)
- Volume attachment management (separate resource)
- Floating IP assignment (separate resource)
- Instance snapshots/backups (future feature)
- Auto-scaling groups or instance pools (future feature)
- Instance migration between hosts/zones (platform operation)
- Real-time performance monitoring/metrics (separate tooling)
- Console access or VNC connections (platform feature)
- Instance resize / changing flavor (resize operation) — out of scope for this feature (resize not supported by the resource)
- Custom network interface configuration beyond platform defaults
- Load balancer backend pool membership (separate resource)
