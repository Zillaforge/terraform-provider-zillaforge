// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helper

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	resourcemodels "github.com/Zillaforge/terraform-provider-zillaforge/internal/vrm/model"

	"github.com/Zillaforge/cloud-sdk/models/vrm/common"
	vrm "github.com/Zillaforge/cloud-sdk/modules/vrm/core"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// listAllTags retrieves all tags from the VRM API (project-wide).
func ListAllTags(ctx context.Context, vrmClient *vrm.Client) ([]*common.Tag, error) {
	// vrmClient := d.client.VRM()
	// Use default options (let the server apply its defaults)
	tags, err := vrmClient.Tags().List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, nil
}

// listTagsForRepository retrieves tags for a specific repository (FR-022 optimization).
func ListTagsForRepository(ctx context.Context, vrmClient *vrm.Client, repoName string) ([]*common.Tag, error) {
	// vrmClient := d.client.VRM()

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
func FilterTags(ctx context.Context, tags []*common.Tag, data resourcemodels.ImagesDataSourceModel) []*common.Tag {
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
func SortTagsDeterministic(tags []*common.Tag) {
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
func TagToImageModel(tag *common.Tag) resourcemodels.ImageModel {
	model := resourcemodels.ImageModel{
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
