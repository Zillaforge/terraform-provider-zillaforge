// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ validator.List = &networkAttachmentPrimaryConstraint{}

// networkAttachmentPrimaryConstraint validates at most one primary=true in network_attachment list.
type networkAttachmentPrimaryConstraint struct{}

// NetworkAttachmentPrimaryConstraint returns a validator that ensures at most one network attachment has primary=true.
func NetworkAttachmentPrimaryConstraint() validator.List {
	return &networkAttachmentPrimaryConstraint{}
}

func (v *networkAttachmentPrimaryConstraint) Description(ctx context.Context) string {
	return "ensures at most one network attachment has primary=true"
}

func (v *networkAttachmentPrimaryConstraint) MarkdownDescription(ctx context.Context) string {
	return "ensures at most one network attachment has `primary=true`"
}

func (v *networkAttachmentPrimaryConstraint) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Extract list elements
	elements := req.ConfigValue.Elements()
	if len(elements) == 0 {
		return
	}

	primaryCount := 0

	// Iterate through network attachments
	for _, elem := range elements {
		obj, ok := elem.(types.Object)
		if !ok {
			continue
		}

		// Get the "primary" attribute value
		attrs := obj.Attributes()
		primaryAttr, exists := attrs["primary"]
		if !exists {
			continue
		}

		primaryBool, ok := primaryAttr.(types.Bool)
		if !ok || primaryBool.IsNull() || primaryBool.IsUnknown() {
			continue
		}

		if primaryBool.ValueBool() {
			primaryCount++
		}
	}

	if primaryCount > 1 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Multiple Primary Network Attachments",
			fmt.Sprintf("Only one network attachment can have primary=true, found %d. Set primary=true on exactly one network attachment or leave it unset.", primaryCount),
		)
	}
}
