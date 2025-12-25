// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helper

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	flavorsmodels "github.com/Zillaforge/cloud-sdk/models/vps/flavors"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// func listFlavorsWithReflection() removed - using typed SDK integration.
func ListFlavorsWithSDK(ctx context.Context, projectClient *cloudsdk.ProjectClient, filters model.FlavorDataSourceModel) ([]model.FlavorModel, error) {
	if projectClient == nil {
		return nil, fmt.Errorf("no project client available")
	}
	vpsClient := projectClient.VPS()
	opts := &flavorsmodels.ListFlavorsOptions{}
	if !filters.Name.IsNull() && filters.Name.ValueString() != "" {
		opts.Name = filters.Name.ValueString()
	}
	flavorList, err := vpsClient.Flavors().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("sdk Flavor List() error: %w", err)
	}

	results := []model.FlavorModel{}
	for _, f := range flavorList {
		// Exact name match
		if !filters.Name.IsNull() && f.Name != filters.Name.ValueString() {
			continue
		}
		// min vcpus
		if !filters.VCPUs.IsNull() {
			if int64(f.VCPU) < filters.VCPUs.ValueInt64() {
				continue
			}
		}
		// min memory - SDK returns GiB
		if !filters.Memory.IsNull() {
			memoryGB := int64(f.Memory)
			if memoryGB < filters.Memory.ValueInt64() {
				continue
			}
		}

		fm := model.FlavorModel{
			ID:          types.StringValue(f.ID),
			Name:        types.StringValue(f.Name),
			VCPUs:       types.Int64Value(int64(f.VCPU)),
			Memory:      types.Int64Value(int64(f.Memory)),
			Disk:        types.Int64Value(int64(f.Disk)),
			Description: types.StringValue(f.Description),
		}
		results = append(results, fm)
	}
	return results, nil
}
