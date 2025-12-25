// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helper

import (
	"context"
	"fmt"
	"sort"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	networksmodels "github.com/Zillaforge/cloud-sdk/models/vps/networks"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ListNetworksWithSDK(ctx context.Context, projectClient *cloudsdk.ProjectClient, filters model.NetworkDataSourceModel) ([]model.NetworkModel, error) {
	if projectClient == nil {
		return nil, fmt.Errorf("no project client available")
	}
	vpsClient := projectClient.VPS()
	opts := &networksmodels.ListNetworksOptions{}
	if !filters.Name.IsNull() {
		opts.Name = filters.Name.ValueString()
	}
	if !filters.Status.IsNull() {
		opts.Status = filters.Status.ValueString()
	}
	networkList, err := vpsClient.Networks().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("sdk Network List() error: %w", err)
	}
	results := []model.NetworkModel{}
	for _, nr := range networkList {
		// Apply exact name filter: if provided and mismatched -> skip
		if !filters.Name.IsNull() && nr.Network.Name != filters.Name.ValueString() {
			continue
		}
		if !filters.Status.IsNull() && nr.Network.Status != filters.Status.ValueString() {
			continue
		}
		nm := model.NetworkModel{
			ID:          types.StringValue(nr.Network.ID),
			Name:        types.StringValue(nr.Network.Name),
			CIDR:        types.StringValue(nr.Network.CIDR),
			Status:      types.StringValue(nr.Network.Status),
			Description: types.StringValue(nr.Network.Description),
		}
		results = append(results, nm)
	}
	// Deterministic sort: by id asc.
	sortNetworksDeterministic(results)
	return results, nil
}

// sortNetworksDeterministic sorts networks by id asc (deterministic).
func sortNetworksDeterministic(results []model.NetworkModel) {
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].ID.ValueString() < results[j].ID.ValueString()
	})
}
