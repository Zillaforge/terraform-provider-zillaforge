// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data

// Package data provides VRM-focused data sources for the provider. The
// images data source queries VRM Tags to enumerate image repository:tag
// pairs and expose them to Terraform configurations.

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	"github.com/Zillaforge/cloud-sdk/models/vrm/common"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ImagesDataSource{}

// NewImagesDataSource creates a new instance of the images data source.
func NewImagesDataSource() datasource.DataSource {
	return &ImagesDataSource{}
}

// ImagesDataSource defines the images data source implementation.
type ImagesDataSource struct {
	client *cloudsdk.ProjectClient
}

// ImagesDataSourceModel describes the data source config and filters.
type ImagesDataSourceModel struct {
	Repository types.String `tfsdk:"repository"`  // Optional filter
	Tag        types.String `tfsdk:"tag"`         // Optional filter (mutually exclusive with tag_pattern)
	TagPattern types.String `tfsdk:"tag_pattern"` // Optional filter (mutually exclusive with tag)
	Images     []ImageModel `tfsdk:"images"`      // Computed results
}

// ImageModel represents a single image (tag) in the results list.
type ImageModel struct {
	ID              types.String `tfsdk:"id"`
	RepositoryName  types.String `tfsdk:"repository_name"`
	TagName         types.String `tfsdk:"tag_name"`
	Size            types.Int64  `tfsdk:"size"`
	OperatingSystem types.String `tfsdk:"operating_system"`
	Description     types.String `tfsdk:"description"`
	Type            types.String `tfsdk:"type"`
	Status          types.String `tfsdk:"status"`
}

func (d *ImagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images"
}

func (d *ImagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Queries VM images (represented as repository:tag pairs) from the ZillaForge VRM service. " +
			"Images can be filtered by repository name, exact tag name, or tag name pattern. " +
			"Each image includes metadata such as size, operating system, status, and a unique ID used for VM creation.",

		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				MarkdownDescription: "Filter images by exact repository name (case-sensitive). " +
					"When combined with `tag`, returns a single image. " +
					"When used alone, returns all tags for the specified repository. " +
					"Optional - omit to query across all repositories.",
				Optional: true,
			},

			"tag": schema.StringAttribute{
				MarkdownDescription: "Filter images by exact tag name (case-sensitive). " +
					"When combined with `repository`, returns a single image. " +
					"When used alone, returns matching tags across all repositories. " +
					"Mutually exclusive with `tag_pattern`. " +
					"Optional - omit to list all tags (up to server limit).",
				Optional: true,
			},

			"tag_pattern": schema.StringAttribute{
				MarkdownDescription: "Filter images by tag name pattern using glob-style wildcards (`*` matches any characters, `?` matches single character). " +
					"Examples: `v1.*` matches all v1.x tags, `prod-*` matches tags starting with 'prod-'. " +
					"Mutually exclusive with `tag`. " +
					"Optional - omit for exact matching or no tag filtering.",
				Optional: true,
			},

			"images": schema.ListNestedAttribute{
				MarkdownDescription: "List of images matching the filter criteria. " +
					"Returns an empty list if no images match. " +
					"Sorted deterministically by repository name then tag name. " +
					"Maximum 1000 images returned (server-enforced limit).",
				Computed: true,

				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for the image tag (UUID). " +
								"This is the Tag object ID from the cloud-sdk, used to create virtual machines. " +
								"Each repository:tag pair has a unique ID.",
							Computed: true,
						},

						"repository_name": schema.StringAttribute{
							MarkdownDescription: "Name of the repository containing this image tag. " +
								"Repository names are unique within a project.",
							Computed: true,
						},

						"tag_name": schema.StringAttribute{
							MarkdownDescription: "Tag name or label for this image version. " +
								"Examples: `latest`, `v1.0.0`, `prod-2024`, `ubuntu-22.04`. " +
								"Tag names follow image repository naming conventions.",
							Computed: true,
						},

						"size": schema.Int64Attribute{
							MarkdownDescription: "Image size in bytes. " +
								"Includes manifest and layer sizes as reported by the VRM API.",
							Computed: true,
						},

						"operating_system": schema.StringAttribute{
							MarkdownDescription: "Operating system type for this image. " +
								"Valid values: `linux`, `windows`.",
							Computed: true,
						},

						"description": schema.StringAttribute{
							MarkdownDescription: "Human-readable description of the image repository. " +
								"May be empty if no description is provided.",
							Computed: true,
						},

						"type": schema.StringAttribute{
							MarkdownDescription: "Tag type classification. " +
								"Valid values: `common` (standard image), `increase` (incremental image).",
							Computed: true,
						},

						"status": schema.StringAttribute{
							MarkdownDescription: "Current status of the image tag. " +
								"Common values: `active` (ready for use), `queued`, `saving`, `creating`, `error`, `deleted`. " +
								"Only `active` and `available` images are typically usable for VM creation.",
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *ImagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*cloudsdk.ProjectClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *cloudsdk.ProjectClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ImagesDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate mutual exclusivity of tag and tag_pattern
	hasExactTag := !data.Tag.IsNull() && data.Tag.ValueString() != ""
	hasPatternTag := !data.TagPattern.IsNull() && data.TagPattern.ValueString() != ""

	if hasExactTag && hasPatternTag {
		resp.Diagnostics.AddError(
			"Invalid Filter Combination",
			"Cannot specify both 'tag' and 'tag_pattern' filters. Please use only one tag filter at a time.",
		)
		return
	}

	tflog.Debug(ctx, "Reading images from VRM API", map[string]interface{}{
		"repository":  data.Repository.ValueString(),
		"tag":         data.Tag.ValueString(),
		"tag_pattern": data.TagPattern.ValueString(),
	})

	var tags []*common.Tag
	var err error

	// FR-022: Use repository-scoped API when repository filter is provided for efficiency
	if !data.Repository.IsNull() && data.Repository.ValueString() != "" {
		tags, err = d.listTagsForRepository(ctx, data.Repository.ValueString())
	} else {
		// Project-wide tag listing
		tags, err = d.listAllTags(ctx)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to retrieve images",
			fmt.Sprintf("Unable to query VRM tags: %s", err.Error()),
		)
		return
	}

	// Apply client-side filtering
	filteredTags := d.filterTags(ctx, tags, data)

	// Sort deterministically by repository_name asc, then tag_name asc (FR-015)
	d.sortTagsDeterministic(filteredTags)

	// Convert to ImageModel slice
	images := make([]ImageModel, 0, len(filteredTags))
	for _, tag := range filteredTags {
		images = append(images, d.tagToImageModel(tag))
	}

	data.Images = images

	// Log retrieved image repository:tag pairs for debugging
	var imagePairs []string
	for _, img := range images {
		imagePairs = append(imagePairs, img.RepositoryName.ValueString()+":"+img.TagName.ValueString())
	}
	tflog.Debug(ctx, "Successfully retrieved images", map[string]interface{}{
		"count":  len(images),
		"images": imagePairs,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// listAllTags retrieves all tags from the VRM API (project-wide).
func (d *ImagesDataSource) listAllTags(ctx context.Context) ([]*common.Tag, error) {
	vrmClient := d.client.VRM()
	// Use default options (let the server apply its defaults)
	tags, err := vrmClient.Tags().List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, nil
}

// listTagsForRepository retrieves tags for a specific repository (FR-022 optimization).
func (d *ImagesDataSource) listTagsForRepository(ctx context.Context, repoName string) ([]*common.Tag, error) {
	vrmClient := d.client.VRM()

	// List repositories to find the one matching the name
	repos, err := vrmClient.Repositories().List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	// Log repository names for debugging acceptance test mismatch
	var repoNames []string
	for _, r := range repos {
		repoNames = append(repoNames, r.Repository.Name)
	}
	tflog.Debug(ctx, "Repositories found for project", map[string]interface{}{
		"count":     len(repos),
		"names":     repoNames,
		"requested": repoName,
	})

	var repoID string
	for _, r := range repos {
		if r.Repository.Name == repoName {
			repoID = r.Repository.ID
			break
		}
	}

	// FR-007: If repository not found in Repositories().List(), fall back to
	// scanning all project tags for matching repository name. This ensures
	// repository-name filtering works even if the Repositories index is stale
	// or inconsistent with tag data.
	if repoID == "" {
		tflog.Debug(ctx, "Repository not found in repository list, falling back to tag scan", map[string]interface{}{
			"repository": repoName,
		})

		// Project-wide tag listing fallback
		tags, err := vrmClient.Tags().List(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list tags during fallback for repository %s: %w", repoName, err)
		}

		var fallback []*common.Tag
		for _, t := range tags {
			if t.Repository != nil && t.Repository.Name == repoName {
				fallback = append(fallback, t)
			}
		}

		return fallback, nil
	}

	// Use repository-scoped tags listing for efficiency
	repoRes, err := vrmClient.Repositories().Get(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s: %w", repoID, err)
	}

	tags, err := repoRes.Tags().List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags for repository %s: %w", repoName, err)
	}

	return tags, nil
}

// filterTags applies client-side filtering based on tag name or pattern.
//
// Notes:
// - Exact tag matches are applied first when `tag` is provided.
// - Glob-style matching (via filepath.Match) is applied when `tag_pattern` is provided.
// - Invalid patterns are skipped with a warning to avoid failing the data source read.
func (d *ImagesDataSource) filterTags(ctx context.Context, tags []*common.Tag, data ImagesDataSourceModel) []*common.Tag {
	var filtered []*common.Tag

	for _, tag := range tags {
		// Filter by exact tag name
		if !data.Tag.IsNull() && data.Tag.ValueString() != "" {
			if tag.Name != data.Tag.ValueString() {
				continue
			}
		}

		// Filter by tag pattern (glob matching)
		if !data.TagPattern.IsNull() && data.TagPattern.ValueString() != "" {
			pattern := data.TagPattern.ValueString()
			matched, err := filepath.Match(pattern, tag.Name)
			if err != nil {
				tflog.Warn(ctx, "Invalid glob pattern", map[string]interface{}{
					"pattern": pattern,
					"error":   err.Error(),
				})
				continue
			}
			if !matched {
				continue
			}
		}

		filtered = append(filtered, tag)
	}

	return filtered
}

// sortTagsDeterministic sorts tags by repository_name asc, then tag_name asc (FR-015).
func (d *ImagesDataSource) sortTagsDeterministic(tags []*common.Tag) {
	sort.SliceStable(tags, func(i, j int) bool {
		if tags[i].Repository != nil && tags[j].Repository != nil {
			if tags[i].Repository.Name != tags[j].Repository.Name {
				return tags[i].Repository.Name < tags[j].Repository.Name
			}
		}
		return tags[i].Name < tags[j].Name
	})
}

// tagToImageModel converts a cloud-sdk Tag to an ImageModel.
//
// Implementation details:
//   - Repository-level metadata is copied into the image model when available.
//   - Empty repository descriptions are represented as types.StringNull() so Terraform
//     does not treat an empty string as an explicitly-set value.
func (d *ImagesDataSource) tagToImageModel(tag *common.Tag) ImageModel {
	model := ImageModel{
		ID:      types.StringValue(tag.ID),
		TagName: types.StringValue(tag.Name),
		Size:    types.Int64Value(tag.Size),
		Type:    types.StringValue(tag.Type.String()),
		Status:  types.StringValue(tag.Status.String()),
	}

	// Extract repository-level attributes
	if tag.Repository != nil {
		model.RepositoryName = types.StringValue(tag.Repository.Name)
		model.OperatingSystem = types.StringValue(tag.Repository.OperatingSystem)
		// Treat empty repository description as null to avoid empty-string "set" values
		if tag.Repository.Description != "" {
			model.Description = types.StringValue(tag.Repository.Description)
		} else {
			model.Description = types.StringNull()
		}
	} else {
		model.RepositoryName = types.StringValue("")
		model.OperatingSystem = types.StringValue("")
		model.Description = types.StringNull()
	}

	return model
}
