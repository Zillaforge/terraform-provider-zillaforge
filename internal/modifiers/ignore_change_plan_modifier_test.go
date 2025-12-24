// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package modifiers

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIgnoreChangeAttributePlanModifierString_ReplacesPlanWithState(t *testing.T) {
	mod := IgnoreChangeAttributePlanModifierString("myattr")

	req := planmodifier.StringRequest{
		PlanValue:  types.StringValue("new"),
		StateValue: types.StringValue("old"),
	}

	resp := &planmodifier.StringResponse{}
	mod.PlanModifyString(context.Background(), req, resp)

	if !resp.PlanValue.Equal(req.StateValue) {
		t.Fatalf("expected plan value to be replaced with state value")
	}
}

func TestIgnoreChangeAttributePlanModifierString_AllowsCreate(t *testing.T) {
	mod := IgnoreChangeAttributePlanModifierString("myattr")

	req := planmodifier.StringRequest{
		PlanValue:  types.StringValue("new"),
		StateValue: types.StringNull(),
	}

	resp := &planmodifier.StringResponse{}
	mod.PlanModifyString(context.Background(), req, resp)

	// PlanModifyString should not set the plan value to some other value (like a previous state 'old') during create
	if resp.PlanValue.Equal(types.StringValue("old")) {
		t.Fatalf("expected plan value not to be replaced during create")
	}
}

func TestIgnoreChangeAttributePlanModifierBool_ReplacesPlanWithState(t *testing.T) {
	mod := IgnoreChangeAttributePlanModifierBool("flag")

	req := planmodifier.BoolRequest{
		PlanValue:  types.BoolValue(false),
		StateValue: types.BoolValue(true),
	}

	resp := &planmodifier.BoolResponse{}
	mod.PlanModifyBool(context.Background(), req, resp)

	if !resp.PlanValue.Equal(req.StateValue) {
		t.Fatalf("expected bool plan value to be replaced with state value")
	}
}

func TestIgnoreChangeAttributePlanModifierBool_AllowsCreate(t *testing.T) {
	mod := IgnoreChangeAttributePlanModifierBool("flag")

	req := planmodifier.BoolRequest{
		PlanValue:  types.BoolValue(false),
		StateValue: types.BoolNull(),
	}

	resp := &planmodifier.BoolResponse{}
	mod.PlanModifyBool(context.Background(), req, resp)

	if resp.PlanValue.Equal(types.BoolValue(true)) {
		t.Fatalf("expected plan value not to be replaced during create")
	}
}
