// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	vps_data "github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/data"
	vps_resource "github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/resource"
	vrm_data "github.com/Zillaforge/terraform-provider-zillaforge/internal/vrm/data"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// clientWrapper is a minimal interface wrapping the parts of the cloudsdk
// that the provider needs during Configure(). Using this small interface
// allows unit tests to inject test doubles without relying on the full SDK
// implementation or network access.
type clientWrapper interface {
	Project(ctx context.Context, projectIDOrCode string) (interface{}, error)
}

// newClientWrapper constructs a clientWrapper from SDK credentials. In
// production this delegates to cloudsdk.NewClient; tests can replace this
// variable with a stub to control behavior in unit tests.
var newClientWrapper = func(apiEndpoint, apiKey string) clientWrapper {
	return &sdkClientWrapper{client: cloudsdk.NewClient(apiEndpoint, apiKey)}
}

type sdkClientWrapper struct {
	client *cloudsdk.Client
}

func (s *sdkClientWrapper) Project(ctx context.Context, projectIDOrCode string) (interface{}, error) {
	return s.client.Project(ctx, projectIDOrCode)
}

// Ensure ZillaforgeProvider satisfies various provider interfaces.
var _ provider.Provider = &ZillaforgeProvider{}
var _ provider.ProviderWithFunctions = &ZillaforgeProvider{}
var _ provider.ProviderWithEphemeralResources = &ZillaforgeProvider{}

// ZillaforgeProvider defines the provider implementation.
type ZillaforgeProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ZillaforgeProviderModel describes the provider data model.
type ZillaforgeProviderModel struct {
	APIEndpoint    types.String `tfsdk:"api_endpoint"`
	APIKey         types.String `tfsdk:"api_key"`
	ProjectID      types.String `tfsdk:"project_id"`
	ProjectSysCode types.String `tfsdk:"project_sys_code"`
}

// T048: JWT token format validation helper (<100ms per NFR-001)
// isValidJWTFormat checks if a token has the format header.payload.signature
// without performing cryptographic validation (which happens in the SDK).
func isValidJWTFormat(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != ""
}

func (p *ZillaforgeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "zillaforge"
	resp.Version = p.version
}

func (p *ZillaforgeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provider for managing Zillaforge cloud resources. Requires API authentication and project context.",
		Attributes: map[string]schema.Attribute{
			"api_endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL for the Zillaforge API. Override this to use a different environment (staging, development) or regional endpoint. Can also be set via `ZILLAFORGE_API_ENDPOINT` environment variable.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for authenticating with Zillaforge services. Must be a valid JWT token. This credential is sensitive and will not be displayed in Terraform plan output or logs. Can be provided via the `ZILLAFORGE_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Numeric or UUID identifier for the Zillaforge project. Exactly one of `project_id` or `project_sys_code` must be specified. Can be set via `ZILLAFORGE_PROJECT_ID` environment variable.",
				Optional:            true,
			},
			"project_sys_code": schema.StringAttribute{
				MarkdownDescription: "Alphanumeric system code for the Zillaforge project. Exactly one of `project_id` or `project_sys_code` must be specified. Can be set via `ZILLAFORGE_PROJECT_SYS_CODE` environment variable.",
				Optional:            true,
			},
		},
	}
}

func (p *ZillaforgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ZillaforgeProviderModel

	// Only attempt to read from request config if a value was provided. In
	// some programmatic tests the tfsdk.Config may be zero-valued which would
	// otherwise panic. Avoid calling Get in that case and rely on environment
	// fallbacks for tests.
	if req.Config.Raw.Type() != nil {
		resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	// T025: INFO level logging for provider configuration start
	tflog.Info(ctx, "Configuring Zillaforge provider")

	// T049: Environment variable fallback for all 4 attributes
	// Get api_endpoint with fallback chain: explicit config → env var → default
	apiEndpoint := data.APIEndpoint.ValueString()
	if apiEndpoint == "" {
		apiEndpoint = os.Getenv("ZILLAFORGE_API_ENDPOINT")
	}
	if apiEndpoint == "" {
		apiEndpoint = "https://api.zillaforge.com"
	}

	// Get api_key with fallback chain: explicit config → env var
	apiKey := data.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("ZILLAFORGE_API_KEY")
	}

	// T050: Validate api_key presence
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"api_key must be set via provider block or ZILLAFORGE_API_KEY environment variable.",
		)
		return
	}

	// T048 & T051: JWT token format validation (<100ms per NFR-001)
	if !isValidJWTFormat(apiKey) {
		resp.Diagnostics.AddError(
			"Invalid API Key Format",
			"api_key must be a valid JWT token in the format header.payload.signature. "+
				"Verify your API key is correctly formatted.",
		)
		return
	}

	// Get project identifiers with fallback chain
	projectID := data.ProjectID.ValueString()
	if projectID == "" {
		projectID = os.Getenv("ZILLAFORGE_PROJECT_ID")
	}

	projectSysCode := data.ProjectSysCode.ValueString()
	if projectSysCode == "" {
		projectSysCode = os.Getenv("ZILLAFORGE_PROJECT_SYS_CODE")
	}

	// T052: Validate project identifier mutual exclusivity
	hasProjectID := projectID != ""
	hasProjectSysCode := projectSysCode != ""

	// T053: Detailed diagnostic messages per contracts/provider-config-schema.md
	if !hasProjectID && !hasProjectSysCode {
		resp.Diagnostics.AddError(
			"Missing Project Identifier",
			"Either project_id or project_sys_code must be specified. Set one via provider block or environment variables ZILLAFORGE_PROJECT_ID or ZILLAFORGE_PROJECT_SYS_CODE.",
		)
		return
	}

	if hasProjectID && hasProjectSysCode {
		resp.Diagnostics.AddError(
			"Conflicting Project Identifiers",
			"Only one of project_id or project_sys_code can be specified, not both. Please remove one from your provider configuration.",
		)
		return
	}

	projectIDOrCode := projectID
	if projectIDOrCode == "" {
		projectIDOrCode = projectSysCode
	}

	// T054: Structured logging with provider context for multi-instance support
	tflog.Debug(ctx, "Initializing Zillaforge SDK client", map[string]interface{}{
		"api_endpoint":       apiEndpoint,
		"project_id_or_code": projectIDOrCode,
		"provider_version":   p.version,
	})

	// T055: Initialize SDK client with validated config values
	sdkClient := newClientWrapper(apiEndpoint, apiKey)

	// Get project-specific client
	projectClient, err := sdkClient.Project(ctx, projectIDOrCode)
	if err != nil {
		// T056: Add retry count and error details to SDK initialization error diagnostics (NFR-003)
		// Note: The SDK handles retries internally with exponential backoff
		resp.Diagnostics.AddError(
			"SDK Initialization Failed",
			fmt.Sprintf(
				"Unable to create Zillaforge project client: %s. "+
					"Verify that your API key is valid and the project ID/code '%s' exists. "+
					"The SDK performs automatic retries with exponential backoff for transient failures.",
				err.Error(),
				projectIDOrCode,
			),
		)
		return
	}

	tflog.Info(ctx, "Zillaforge SDK client initialized successfully", map[string]interface{}{
		"project_id_or_code": projectIDOrCode,
	})

	// T028: Share SDK client via resp.ResourceData and resp.DataSourceData
	resp.DataSourceData = projectClient
	resp.ResourceData = projectClient
}

func (p *ZillaforgeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		vps_resource.NewKeypairResource,
		vps_resource.NewSecurityGroupResource,
	}
}

func (p *ZillaforgeProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *ZillaforgeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		vps_data.NewFlavorDataSource,
		vps_data.NewNetworkDataSource,
		vps_data.NewKeypairDataSource,
		vps_data.NewSecurityGroupsDataSource,
		vrm_data.NewImagesDataSource,
	}
}

func (p *ZillaforgeProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ZillaforgeProvider{
			version: version,
		}
	}
}
