// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &portRangeValidator{}

// portRangeValidator validates port range strings according to security group requirements:
// - Literal "all" (case-insensitive) for all ports (1-65535)
// - Single port: "80" (1-65535)
// - Port range: "8000-8100" (start <= end, both 1-65535).
type portRangeValidator struct{}

// PortRange returns a validator for port range strings.
func PortRange() validator.String {
	return &portRangeValidator{}
}

func (v *portRangeValidator) Description(ctx context.Context) string {
	return "value must be 'all', a single port (1-65535), or a port range in format 'start-end' where start <= end"
}

func (v *portRangeValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be `all`, a single port (`1-65535`), or a port range in format `start-end` where start ≤ end"
}

func (v *portRangeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is unknown or null
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Check for literal "all" (case-insensitive)
	if strings.ToLower(value) == "all" {
		return
	}

	// Check for port range pattern: "start-end"
	rangePattern := regexp.MustCompile(`^(\d+)-(\d+)$`)
	if matches := rangePattern.FindStringSubmatch(value); matches != nil {
		startPort, err1 := strconv.Atoi(matches[1])
		endPort, err2 := strconv.Atoi(matches[2])

		if err1 != nil || err2 != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Port Range",
				fmt.Sprintf("Port range '%s' contains invalid numeric values.", value),
			)
			return
		}

		// Validate port range bounds
		if startPort < 1 || startPort > 65535 || endPort < 1 || endPort > 65535 {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Port Out of Range",
				fmt.Sprintf("Port range '%s' contains ports outside valid range (1-65535).", value),
			)
			return
		}

		// Validate start <= end
		if startPort > endPort {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Port Range",
				fmt.Sprintf("Port range '%s' has start port (%d) greater than end port (%d). Start port must be less than or equal to end port.", value, startPort, endPort),
			)
			return
		}

		return
	}

	// Check for single port: numeric value 1-65535
	port, err := strconv.Atoi(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Port Format",
			fmt.Sprintf("Port range '%s' must be 'all', a single port number (1-65535), or a range in format 'start-end'.", value),
		)
		return
	}

	if port < 1 || port > 65535 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Port Out of Range",
			fmt.Sprintf("Port '%s' is outside valid range (1-65535).", value),
		)
		return
	}
}
