// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helper

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	keypairsmodels "github.com/Zillaforge/cloud-sdk/models/vps/keypairs"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// T038: KeypairToModel() conversion helper.
func KeypairToModel(kp keypairsmodels.Keypair) model.KeypairModel {
	model := model.KeypairModel{
		ID:          types.StringValue(kp.ID),
		Name:        types.StringValue(kp.Name),
		PublicKey:   types.StringValue(kp.PublicKey),
		Fingerprint: types.StringValue(kp.Fingerprint),
	}

	// Handle optional description
	if kp.Description != "" {
		model.Description = types.StringValue(kp.Description)
	} else {
		model.Description = types.StringNull()
	}

	return model
}

// ListKeypairsWithSDK queries keypairs using cloud-sdk List() method.
func ListKeypairsWithSDK(ctx context.Context, projectClient *cloudsdk.ProjectClient, filters model.KeypairDataSourceModel) ([]model.KeypairModel, error) {
	if projectClient == nil {
		return nil, fmt.Errorf("no project client available")
	}

	vpsClient := projectClient.VPS()
	opts := &keypairsmodels.ListKeypairsOptions{}

	// Apply name filter if provided
	if !filters.Name.IsNull() && filters.Name.ValueString() != "" {
		opts.Name = filters.Name.ValueString()
	}

	keypairList, err := vpsClient.Keypairs().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("sdk Keypair List() error: %w", err)
	}

	results := []model.KeypairModel{}
	for _, kp := range keypairList {
		// Apply exact name filter if provided
		if !filters.Name.IsNull() && kp.Name != filters.Name.ValueString() {
			continue
		}

		results = append(results, KeypairToModel(*kp))
	}

	return results, nil
}
