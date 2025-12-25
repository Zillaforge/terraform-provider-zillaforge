// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helper

import (
	"context"
	"sort"

	floatingipmodels "github.com/Zillaforge/cloud-sdk/models/vps/floatingips"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stringPointerOrNull returns nil for empty strings (converts to types.StringNull).
func stringPointerOrNull(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// MapFloatingIPToResourceModel converts SDK FloatingIP to resource model.
func MapFloatingIPToResourceModel(ctx context.Context, fip *floatingipmodels.FloatingIP, data *model.FloatingIPResourceModel) {
	data.ID = types.StringValue(fip.ID)
	data.Name = types.StringPointerValue(stringPointerOrNull(fip.Name))
	data.Description = types.StringPointerValue(stringPointerOrNull(fip.Description))
	data.IPAddress = types.StringValue(fip.Address)
	data.Status = types.StringValue(string(fip.Status))
	data.DeviceID = types.StringPointerValue(stringPointerOrNull(fip.DeviceID))
}

// MapFloatingIPToModel converts SDK FloatingIP to data source result model.
func MapFloatingIPToModel(fip *floatingipmodels.FloatingIP) model.FloatingIPModel {
	return model.FloatingIPModel{
		ID:          types.StringValue(fip.ID),
		Name:        types.StringPointerValue(stringPointerOrNull(fip.Name)),
		Description: types.StringPointerValue(stringPointerOrNull(fip.Description)),
		IPAddress:   types.StringValue(fip.Address),
		Status:      types.StringValue(string(fip.Status)),
		DeviceID:    types.StringPointerValue(stringPointerOrNull(fip.DeviceID)),
	}
}

// BuildCreateRequest creates FloatingIPCreateRequest from resource model.
func BuildCreateRequest(data *model.FloatingIPResourceModel) *floatingipmodels.FloatingIPCreateRequest {
	req := &floatingipmodels.FloatingIPCreateRequest{}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		req.Name = data.Name.ValueString()
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		req.Description = data.Description.ValueString()
	}

	return req
}

// BuildUpdateRequest creates FloatingIPUpdateRequest from resource model.
func BuildUpdateRequest(data *model.FloatingIPResourceModel) *floatingipmodels.FloatingIPUpdateRequest {
	req := &floatingipmodels.FloatingIPUpdateRequest{}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		req.Name = data.Name.ValueString()
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		req.Description = data.Description.ValueString()
	}

	return req
}

// FilterFloatingIPs applies client-side filtering to floating IP list.
func FilterFloatingIPs(fips []*floatingipmodels.FloatingIP, filters *model.FloatingIPDataSourceModel) []*floatingipmodels.FloatingIP {
	var filtered []*floatingipmodels.FloatingIP

	for _, fip := range fips {
		if matchesFilters(fip, filters) {
			filtered = append(filtered, fip)
		}
	}

	// Sort by ID for deterministic ordering
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})

	return filtered
}

// matchesFilters checks if floating IP matches all specified filters (AND logic).
func matchesFilters(fip *floatingipmodels.FloatingIP, filters *model.FloatingIPDataSourceModel) bool {
	// ID filter
	if !filters.ID.IsNull() && !filters.ID.IsUnknown() {
		if fip.ID != filters.ID.ValueString() {
			return false
		}
	}

	// Name filter
	if !filters.Name.IsNull() && !filters.Name.IsUnknown() {
		if fip.Name != filters.Name.ValueString() {
			return false
		}
	}

	// IP Address filter
	if !filters.IPAddress.IsNull() && !filters.IPAddress.IsUnknown() {
		if fip.Address != filters.IPAddress.ValueString() {
			return false
		}
	}

	// Status filter
	if !filters.Status.IsNull() && !filters.Status.IsUnknown() {
		if string(fip.Status) != filters.Status.ValueString() {
			return false
		}
	}

	return true
}
