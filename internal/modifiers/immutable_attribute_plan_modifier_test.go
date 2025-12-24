// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package modifiers

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestImmutableAttributePlanModifier_DifferentValues_ProducesError(t *testing.T) {
	t.Parallel()
	mod := ImmutableAttributePlanModifier("flavor_id")

	req := planmodifier.StringRequest{
		Path:       path.Root("flavor_id"),
		StateValue: types.StringValue("small"),
		PlanValue:  types.StringValue("large"),
	}
	resp := &planmodifier.StringResponse{}

	mod.PlanModifyString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostic error when changing immutable attribute, got none: %#v", resp.Diagnostics)
	}
}

func TestImmutableAttributePlanModifier_SameValues_NoError(t *testing.T) {
	t.Parallel()
	mod := ImmutableAttributePlanModifier("image_id")

	req := planmodifier.StringRequest{
		Path:       path.Root("image_id"),
		StateValue: types.StringValue("img-1"),
		PlanValue:  types.StringValue("img-1"),
	}
	resp := &planmodifier.StringResponse{}

	mod.PlanModifyString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostic error when attribute unchanged, got: %#v", resp.Diagnostics)
	}
}

func TestImmutableAttributePlanModifier_StateNull_AllowsChange(t *testing.T) {
	t.Parallel()
	mod := ImmutableAttributePlanModifier("flavor_id")

	req := planmodifier.StringRequest{
		Path:       path.Root("flavor_id"),
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue("large"),
	}
	resp := &planmodifier.StringResponse{}

	mod.PlanModifyString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostic error when state is null (create), got: %#v", resp.Diagnostics)
	}
}

func TestImmutableAttributePlanModifier_PlanUnknown_Allows(t *testing.T) {
	t.Parallel()
	mod := ImmutableAttributePlanModifier("image_id")

	req := planmodifier.StringRequest{
		Path:       path.Root("image_id"),
		StateValue: types.StringValue("img-1"),
		PlanValue:  types.StringUnknown(),
	}
	resp := &planmodifier.StringResponse{}

	mod.PlanModifyString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostic error when plan is unknown, got: %#v", resp.Diagnostics)
	}
}
