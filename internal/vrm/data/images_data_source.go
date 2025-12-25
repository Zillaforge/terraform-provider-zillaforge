// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data

// Package data provides VRM-focused data sources for the provider. The
// images data source queries VRM Tags to enumerate image repository:tag
// pairs and expose them to Terraform configurations.

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vrm/helper"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vrm/model"

	"github.com/Zillaforge/cloud-sdk/models/vrm/common"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
	var data model.ImagesDataSourceModel

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
		tags, err = helper.ListTagsForRepository(ctx, d.client.VRM(), data.Repository.ValueString())
	} else {
		// Project-wide tag listing
		tags, err = helper.ListAllTags(ctx, d.client.VRM())
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to retrieve images",
			fmt.Sprintf("Unable to query VRM tags: %s", err.Error()),
		)
		return
	}

	// Apply client-side filtering
	filteredTags := helper.FilterTags(ctx, tags, data)

	// Sort deterministically by repository_name asc, then tag_name asc (FR-015)
	helper.SortTagsDeterministic(filteredTags)

	// Convert to ImageModel slice
	images := make([]model.ImageModel, 0, len(filteredTags))
	for _, tag := range filteredTags {
		images = append(images, helper.TagToImageModel(tag))
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
