# Feature Specification: Flavor and Network Data Sources

**Feature Branch**: `002-flavor-network-datasources`  
**Created**: 2025-12-11  
**Status**: Draft  
**Input**: User description: "設計 terraform provider 裡的 flavor 與 network 二種data source"

## Clarifications

### Session 2025-12-11

- Q: How should data sources handle single vs. multiple result returns? → A: Single data source always returns a list (users index with [0] to get first match)
- Q: What matching behavior should the `name` filter use? → A: Exact match only (e.g., "prod" matches only "prod", not "production")
- Q: Should data sources configure explicit API timeouts? → A: Use SDK defaults (no explicit timeout configuration in data sources)
- Q: How should multiple filters combine when specified together? → A: AND logic - all specified filters must match (e.g., vcpus >= 4 AND memory >= 8 returns only flavors meeting both criteria)
- Q: Should the disk attribute be exposed for flavors? → A: Expose disk attribute (optional since some flavors may use separate volumes)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Query Available Flavors (Priority: P1)

As a Terraform user, I need to query available compute flavors from Zillaforge so that I can discover and filter instance sizes based on CPU, memory, and disk specifications.

**Why this priority**: This is the foundational data source that enables users to discover what compute resources are available in their Zillaforge project. Without this, users cannot make informed decisions about instance sizing or reference flavors in their resource configurations.

**Independent Test**: Can be fully tested by configuring the provider with valid credentials, writing a data source block that queries flavors, running `terraform plan`, and verifying that the data source returns a list of available flavors with their specifications (name, CPU count, memory size in GB, disk size). Delivers immediate value by exposing infrastructure options.

**Acceptance Scenarios**:

1. **Given** a configured Zillaforge provider, **When** I define a `zillaforge_flavors` data source without filters, **Then** it returns all available flavors as a list
2. **Given** a `zillaforge_flavors` data source with a name filter, **When** I run `terraform plan`, **Then** it returns only flavors matching that name as a list
3. **Given** a `zillaforge_flavors` data source with vcpus filter, **When** I specify minimum CPU count, **Then** it returns only flavors meeting or exceeding that CPU count as a list
4. **Given** a `zillaforge_flavors` data source with memory filter, **When** I specify minimum memory size in GB, **Then** it returns only flavors meeting or exceeding that memory size as a list
5. **Given** multiple matching flavors, **When** the data source query executes, **Then** it returns all matching flavors as a list
6. **Given** no flavors match the filter criteria, **When** the data source query executes, **Then** it returns an empty list without error
7. **Given** a user wants a single flavor, **When** they access the first element using `data.zillaforge_flavors.example.flavors[0]`, **Then** they can reference that flavor's attributes

---

### User Story 2 - Query Available Networks (Priority: P1)

As a Terraform user, I need to query available networks from Zillaforge so that I can discover and filter networks based on name or status.

**Why this priority**: This is equally foundational as the flavor data source. Networks are essential for any infrastructure deployment, and users must be able to discover and reference existing networks before creating resources. This story is independently valuable and can be developed in parallel with the flavor data source.

**Independent Test**: Can be tested by configuring the provider, writing a data source block that queries networks, running `terraform plan`, and verifying that the data source returns available networks with their attributes (id, name, CIDR, status, description). Delivers immediate value by exposing network topology.

**Acceptance Scenarios**:

1. **Given** a configured Zillaforge provider, **When** I define a `zillaforge_networks` data source without filters, **Then** it returns all available networks as a list
2. **Given** a `zillaforge_networks` data source with a name filter, **When** I run `terraform plan`, **Then** it returns only networks matching that name as a list
3. **Given** a `zillaforge_networks` data source with status filter, **When** I specify a status value, **Then** it returns only networks with that status as a list
4. **Given** multiple matching networks, **When** the data source query executes, **Then** it returns all matching networks as a list
5. **Given** no networks match the filter criteria, **When** the data source query executes, **Then** it returns an empty list without error
6. **Given** a user wants a single network, **When** they access the first element using `data.zillaforge_networks.example.networks[0]`, **Then** they can reference that network's attributes

---

### User Story 3 - Reference Data Sources in Resource Configurations (Priority: P2)

As a Terraform user, I need to reference flavor and network data sources in my resource configurations so that I can create instances using dynamically discovered infrastructure parameters rather than hardcoded values.

**Why this priority**: This story validates the integration value of the data sources. While P1 stories prove the data sources work in isolation, this story demonstrates their practical utility in real infrastructure-as-code workflows. It depends on at least one P1 story being complete.

**Independent Test**: Can be tested by creating a Terraform configuration that uses a flavor data source to populate an instance's flavor attribute and a network data source to populate network attachment, then running `terraform plan` and verifying the plan shows correct attribute references. Delivers the integration pattern that users will follow in production.

**Acceptance Scenarios**:

1. **Given** a `zillaforge_flavors` data source with filters that return a single flavor, **When** I reference `data.zillaforge_flavors.example.flavors[0].id` in a resource, **Then** Terraform correctly substitutes the flavor ID in the plan
2. **Given** a `zillaforge_networks` data source with filters that return a single network, **When** I reference `data.zillaforge_networks.example.networks[0].id` in a resource, **Then** Terraform correctly substitutes the network ID in the plan
3. **Given** a `zillaforge_flavors` (plural) data source with multiple results, **When** I use indexing (e.g., `data.zillaforge_flavors.all.flavors[0].id`) to select a specific item, **Then** Terraform uses the selected item's attributes
4. **Given** data sources with filter criteria, **When** I run `terraform apply`, **Then** resources are created with the dynamically discovered values
5. **Given** a data source query fails or returns empty, **When** a resource depends on that data, **Then** Terraform plan fails with a clear diagnostic message indicating missing data

---

### Edge Cases

- What happens when the Zillaforge API returns no flavors (empty project)? → Data source returns an empty list; dependent resources should handle this with appropriate validation
- How does the flavor data source handle flavors with missing optional attributes (e.g., no description)? → Data source populates attributes with null/empty values for missing fields; schema marks optional fields (description, disk) as Optional
- What happens when multiple flavors have identical names? → Data source returns all matching flavors; users should use more specific filters or select from list
- How does the network data source handle networks in different states (active, pending, error)? → Data source includes status attribute; users can filter by status
- What happens when filter criteria are overly restrictive and match nothing? → Data source returns empty list; Terraform plan succeeds but may fail if resource requires non-empty reference
- How does the data source handle API pagination for large flavor/network lists? → Data source implementation handles pagination transparently using SDK's pagination support
- What happens if the Zillaforge API adds new flavor types or network attributes in the future? → Data source schema should be extensible; unknown attributes are ignored gracefully unless schema is updated
- How are authentication errors differentiated from "no results found" scenarios? → Authentication errors return Terraform diagnostics with error severity; empty results return empty list with no error
- How does the data source handle concurrent Terraform runs querying the same data? → Each Terraform run makes independent API calls; SDK handles rate limiting and retries

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provider MUST implement a `zillaforge_flavors` data source that returns a list of available compute flavors
- **FR-002**: Provider MUST implement a `zillaforge_networks` data source that returns a list of available networks
- **FR-003**: `zillaforge_flavors` data source MUST support filtering by `name` (exact match, case-sensitive)
- **FR-004**: `zillaforge_flavors` data source MUST support filtering by `vcpus` (minimum CPU count)
- **FR-005**: `zillaforge_flavors` data source MUST support filtering by `memory` (minimum memory in GB)
- **FR-006**: `zillaforge_networks` data source MUST support filtering by `name` (exact match, case-sensitive)
- **FR-007**: `zillaforge_networks` data source MUST support filtering by `status` (status value match)
- **FR-008**: Both data sources MUST return results as list attributes (in `flavors` or `networks` attribute)
- **FR-009**: Both data sources MUST return an empty list when no matches are found (not an error)
- **FR-010**: When multiple filters are specified, results MUST match ALL filter criteria (AND logic)
- **FR-011**: All data sources MUST use the Zillaforge SDK client initialized by the provider for API calls
- **FR-012**: All data sources MUST handle API errors gracefully and return Terraform diagnostics with actionable error messages
- **FR-013**: `zillaforge_flavors` data source MUST expose a `flavors` attribute containing list of flavor objects
- **FR-014**: `zillaforge_networks` data source MUST expose a `networks` attribute containing list of network objects
- **FR-015**: Flavor objects MUST expose computed attributes: `id`, `name`, `vcpus`, `memory`, `disk`, `description`
- **FR-016**: Network objects MUST expose computed attributes: `id`, `name`, `cidr`, `status`, `description`
- **FR-017**: Data source schemas MUST mark all result attributes as Computed
- **FR-018**: Data source schemas MUST mark all filter attributes as Optional
- **FR-019**: Both data sources MUST support usage without any filters (return all available items)
- **FR-020**: Both data sources MUST handle empty result sets without error (return empty list)
- **FR-021**: All data sources MUST implement proper Read method that fetches data from Zillaforge API
- **FR-022**: Both data sources MUST handle API pagination transparently if the API returns paginated results
- **FR-023**: Data source attribute descriptions MUST be documented with MarkdownDescription for documentation generation

### Key Entities *(include if feature involves data)*

- **Flavor**: Represents a compute instance size template with CPU, memory, and disk specifications. Key attributes include unique identifier, human-readable name, virtual CPU count, memory size (in GB), optional disk size (in GB), and optional description.

- **Network**: Represents a virtual network segment for resource connectivity. Key attributes include unique identifier, network name, CIDR block (IP address range), operational status (active, pending, error, etc.), and optional description.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can query all available flavors in their Zillaforge project without errors
- **SC-002**: Users can filter flavors by name and retrieve matching results in under 3 seconds
- **SC-003**: Users can filter flavors by resource specifications (CPU, memory) and receive accurate matches
- **SC-004**: Users can query all available networks in their Zillaforge project without errors
- **SC-005**: Users can filter networks by name or status and retrieve matching results in under 3 seconds
- **SC-006**: Users can successfully reference data source outputs in resource configurations and generate valid Terraform plans
- **SC-007**: Data sources handle API errors with clear diagnostic messages that indicate the root cause (authentication, network, API unavailable, etc.)
- **SC-008**: Data sources return empty lists when no matches are found, allowing Terraform plans to complete successfully
- **SC-009**: Provider documentation includes working examples for both data sources with common filter patterns
- **SC-010**: 90% of users can successfully query and reference flavors/networks in their first Terraform configuration attempt
