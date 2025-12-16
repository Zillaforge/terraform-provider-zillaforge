# Feature Specification: Image Data Source

**Feature Branch**: `005-tag-data-source`  
**Created**: December 15, 2025  
**Status**: Draft  
**Input**: User description: "Design Image data source. Image tag represents a tagged version of an image used to create virtual machines."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Query Image Tags in Repository (Priority: P1)

As a Terraform user, I need to query available images from a ZillaForge repository so that I can discover and reference specific image versions when creating virtual machines. An Image is defined by a repository and a tag (repository:tag), each repository can contain many tags, and the `zillaforge_images` data source provides `repository` and `tag` filters to narrow results.

**Why this priority**: This is the foundational capability - users need to discover what image tags exist in a repository before they can create virtual machines. This enables dynamic image selection and version management in infrastructure-as-code workflows.

**Independent Test**: Can be fully tested by configuring the provider with valid credentials, writing a data source block that queries image tags for a specific repository, running `terraform plan`, and verifying that the data source returns a list of available images with their attributes (name, size). Delivers immediate value by exposing available image versions for VM creation.

**Acceptance Scenarios**:

1. **Given** a configured ZillaForge provider and an existing repository and tag, **When** I define `zillaforge_images` with both `repository` and `tag`, **Then** it returns a list containing exactly one image (length 1) for the specified repository:tag with attributes including `id` (used to create VMs), `repository_name`, `tag_name`, `size` (bytes), `operatingSystem`, `description`, `type`, and `status`.
2. **Given** a configured Zillaforge provider and only a `repository` filter, **When** I define `zillaforge_images` with `repository` only, **Then** it returns a list of images for that repository (one per tag) sorted deterministically by repository name then tag name.
3. **Given** a configured ZillaForge provider and only a `tag` filter, **When** I define `zillaforge_images` with `tag` only, **Then** it returns a list of images across all repositories that have that tag (one per repository) sorted deterministically by repository name then tag name.
4. **Given** no filters are specified, **When** I define `zillaforge_images` without `repository` or `tag`, **Then** the data source returns all available images (one entry per repository:tag) sorted deterministically by repository name then tag name.
5. **Given** a user wants a specific image entry, **When** they access the first element using `data.zillaforge_images.example.images[0]`, **Then** they can reference that image's attributes including `id` (used to create VMs), `repository_name`, `tag_name`, `size` (bytes), `operatingSystem`, `description`, `type`, and `status`.

---

### User Story 2 - Filter Images by Pattern (Priority: P2)

As a Terraform user, I need to filter image tags using pattern matching so that I can discover tags following specific naming conventions (e.g., semantic versioning, environment prefixes).

**Why this priority**: Pattern matching enables advanced workflows like selecting the latest production image tag or finding all tags for a specific version series. While valuable, basic image querying (P1) must work first.

**Independent Test**: Can be tested by creating a repository with tags following different patterns, then using pattern filters to query subsets, and verifying only matching tags are returned. Delivers value for users managing multiple environments or version ranges.

**Acceptance Scenarios**:

1. **Given** a `zillaforge_images` data source with a tag pattern filter (e.g., "v1.*"), **When** I run `terraform plan`, **Then** it returns only images matching that pattern as a list
2. **Given** tags following semantic versioning (v1.0.0, v1.0.1, v2.0.0), **When** I filter by pattern "v1.*", **Then** only v1.x.x tags are returned
3. **Given** tags with environment prefixes (prod-*, staging-*), **When** I filter by pattern "prod-*", **Then** only production tags are returned
4. **Given** a pattern filter that matches no tags, **When** the data source query executes, **Then** it returns an empty list without error

---

### User Story 3 - Reference Image Attributes in VM Provisioning (Priority: P2)

As a Terraform user, I need to reference image tag attribute `id` in my resource configurations so that I can create virtual machines (using `id`).

**Why this priority**: This validates the practical utility of the data source in VM provisioning workflows. It depends on P1 image querying working correctly.

**Independent Test**: Can be tested by creating a Terraform configuration that uses an image data source to populate a VM resource with an image `id`, then verifying the VM is provisioned using the correct image identifier. Delivers the integration pattern for VM provisioning.

**Acceptance Scenarios**:

1. **Given** a `zillaforge_images` data source that returns an image, **When** I reference `data.zillaforge_images.example.images[0].id` in a VM resource, **Then** Terraform correctly substitutes the image `id` when creating the VM
2. **Given** multiple tags for an image, **When** I sort by creation time and select the first element, **Then** Terraform uses the most recently created tag
3. **Given** a tag with size metadata, **When** I reference the size attribute, **Then** I can implement validation or cost estimation logic
4. **Given** a data source query fails or returns empty, **When** a resource depends on that data, **Then** Terraform plan fails with a clear diagnostic message

---

### Edge Cases

- What happens when querying images for a non-existent repository? → Data source returns an empty list (consistent with no-match behavior), allowing Terraform plans to complete successfully
- What happens when no filters are specified? → Data source returns all available images (one entry per repository:tag) up to the server-enforced limit; pagination handles retrieval but results are capped at maximum limit (e.g., 1000 images)
- How does the data source handle repositories with hundreds of images? → Data source implementation handles pagination transparently; returns all images or implements server-side filtering if available
- What happens when a tag is deleted between plan and apply? → Apply phase may fail with tag not found error; users should prefer using the image `id` (unique identifier) when creating VMs to avoid tag-based drift or ambiguity
- How are untagged images handled? → Data source only returns named image tags; untagged images are not included in results
- What happens when both `repository` and `tag` are provided? → Data source returns a list containing exactly one image (length 1) for that repository:tag, including `id` and computed attributes
- What happens when only `tag` is provided? → Data source returns matching images across repositories (one result per repository) to support cross-repo tag queries
- What happens when multiple tags point to the same image content? → Data source returns all tags; each entry has different tag names referencing the same underlying image content (not exposed via a `digest` attribute).
- How does the data source handle special characters in tag names? → Tag name validation follows image repository naming standards; invalid characters return validation error
- What happens when the ZillaForge API returns images in an unexpected order? → Data source normalizes results by sorting by creation time (newest first) with secondary sort by tag_name alphabetically for deterministic ordering
- How are authentication errors differentiated from "no tags found" scenarios? → Authentication errors return Terraform diagnostics with error severity; empty tag lists return empty list with no error

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provider MUST implement a `zillaforge_images` data source that returns images matching the provided `repository` and/or `tag` filters for VM creation
- **FR-002**: `zillaforge_images` data source MUST allow `repository` as an Optional filter (queries may be performed with `repository`, `tag`, or both)
- **FR-003**: `zillaforge_images` data source MUST support filtering by exact tag name (optional)
- **FR-004**: `zillaforge_images` data source MUST support filtering by tag name pattern using glob-style wildcards (* and ?) only; regex patterns are not supported
- **FR-005**: Data source MUST return results as a list attribute named `images`
- **FR-006**: Data source MUST return an empty list when no images match filter criteria (not an error)
- **FR-007**: Data source SHOULD return an empty list when the specified repository does not exist (consistent with no-match behavior per FR-006)
- **FR-008**: Data source MUST use the ZillaForge SDK client initialized by the provider for API calls
- **FR-009**: Data source MUST handle API errors gracefully and return Terraform diagnostics with actionable error messages
- **FR-010**: Image objects MUST expose computed attributes: `id` (string, used to create virtual machines), `repository_name`, `tag_name`, `size` (bytes), `operatingSystem`, `description`, `type`, and `status`.
- **FR-011**: Image objects MAY expose optional computed attributes in future versions: `updated_at`, `image_format`, `platform` (these are out of scope for initial implementation but may be added if Tag.Extra or Tag fields contain this information)
- **FR-012**: Data source schema MUST mark all result attributes as Computed
- **FR-013**: Data source schema MUST mark repository filter as Optional
- **FR-014**: Data source schema MUST mark tag name and pattern filters as Optional and mutually exclusive
- **FR-015**: Data source MUST return images sorted deterministically by repository_name ascending, then tag_name ascending for consistent Terraform state management
- **FR-016**: Data source MUST handle API pagination transparently if the API returns paginated image lists
- **FR-017**: Data source attribute descriptions MUST be documented with MarkdownDescription for documentation generation
- **FR-018**: Data source MUST validate that only one of tag name filter or pattern filter is specified at a time
- **FR-019**: Image size values MUST be returned in bytes in the `size` attribute for consistent measurement
- **FR-020**: Data source MUST return all available images when no filters are specified up to the server-enforced maximum limit (i.e., support an unfiltered global listing with reasonable result cap).
- **FR-021**: When both `repository` and `tag` filters are specified, the data source MUST return a list of length 1 containing the unique image (repository:tag) and include its `id` attribute.
- **FR-022**: When `repository` filter is provided, the data source SHOULD use the VRM Repositories API (`Repositories().Get()`/`List()` + `RepositoryResource.Tags().List()`) to retrieve tags for that repository efficiently.

### Key Entities

- **Image**: Represents a named version/reference of an image used to create virtual machines within a repository. Each Image corresponds to a Tag object in the cloud-sdk. Key attributes include:
  - Unique identifier (`id`) — the Tag object's ID from cloud-sdk; used to create virtual machines; unique per repository:tag pair
  - `repository_name` (repository containing the tag)
  - `tag_name` (user-defined label like "latest", "v1.0.0", "prod-2024")
  - Image content may be identified by an immutable cryptographic hash in the VRM service, but the data source does not expose a `digest` attribute; multiple tags may reference the same image content.

  - `size` (bytes)
  - `operatingSystem` (e.g., linux, windows)
  - `description` (human-readable text)
  - `type` (image format or catalogue type)
  - `status` (e.g., available, deprecated)
  - Optional platform information (architecture, OS)

- **Repository**: The image repository that contains the images (referenced by name as an Optional filter). Note: the data source requires at least one of `repository` or `tag` to be specified.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can query all images in a repository and receive results within 3 seconds
- **SC-002**: Users can filter images by exact name and retrieve the matching image in under 2 seconds
- **SC-003**: Users can filter images by pattern and receive all matching results in under 3 seconds
- **SC-004**: Data sources return empty lists when no images match criteria, allowing Terraform plans to complete successfully
- **SC-005**: Data sources handle API errors with clear diagnostic messages indicating root cause (authentication, repository not found, network errors)
- **SC-006**: Users can successfully use the image `id` to create VMs.
- **SC-007**: 90% of users can successfully query and reference image tags in their first Terraform configuration attempt
- **SC-008**: Data source handles repositories with up to 1000 images without performance degradation
- **SC-009**: Provider documentation includes working examples for common use cases (latest image tag, semantic version filtering, immutable VM provisioning)

## Clarifications

### Session 2025-12-15

- Q: How does `id` relate to repository:tag vs digest? → A: `id` attribute in `zillaforge_images` is the id field from a gove Tag in cloud-sdk (each repository:tag pair has unique ID)
- Q: Are images account-scoped or do they have visibility controls? → A: Project-scoped; API endpoint `vrm/api/v1/project/tags` returns all available images within the project context
- Q: What's the maximum number of images returned in unfiltered query? → A: Server-enforced limit (e.g., 1000 images max)
- Q: Does pattern support only glob (* ?) or also regex? → A: Only glob wildcards (* and ?)
- Q: How to handle images with missing or identical timestamps? → A: Secondary sort by tag_name alphabetically

## Assumptions

- Users have basic knowledge of image repositories and image tagging conventions
- The ZillaForge API supports querying image tags by repository name via the `vrm/api/v1/project/tags` endpoint; for repository-scoped queries, `vrm/api/v1/project/repositories` and `GET /repository/{id}/tags` are available and preferred for efficiency
- Images are scoped to the project context; the data source returns images available within the authenticated project
- Tag names follow standard image repository naming conventions (alphanumeric, hyphens, underscores, periods, limited to 128 characters)
- Image digests use standard cryptographic hash formats (sha256, sha512)
- Repository names are unique within a ZillaForge account/project scope
- The ZillaForge API provides pagination support for repositories with many images; server enforces a maximum result limit to prevent excessive memory usage and timeouts
- Image tag deletion is eventual consistency (may take moments to reflect in queries)
- Untagged images (with only digest references) are not returned by the images data source
- Image size includes manifest and layer sizes as reported by the registry API
- Pattern matching supports standard glob-style wildcards (* and ?)
- Users cannot create image tags through this data source (read-only; tag creation happens through separate mechanisms like image push operations) 
