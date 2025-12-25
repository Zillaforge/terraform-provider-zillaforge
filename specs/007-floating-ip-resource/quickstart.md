# Floating IP Resource & Data Source - Developer Quickstart

**Feature**: 007-floating-ip-resource  
**Date**: December 24, 2025  
**Purpose**: Quick start guide for implementing floating IP functionality

---

## Overview

This guide provides step-by-step instructions for implementing:
- `zillaforge_floating_ip` resource (allocate, update, import floating IPs)
- `zillaforge_floating_ips` data source (query floating IPs with filters)

**Key Implementation Details**:
- Single IP pool (no pool selection parameter)
- Client-side filtering (SDK List() filter has bugs)
- Models in `internal/vps/model` (shared between resource and data source)
- Test with `make testacc TESTARGS='-run=TestAccXXXX' PARALLEL=1`

---

## Prerequisites

1. **Go 1.22.4+** installed
2. **Terraform Plugin Framework v1.14.1** in go.mod
3. **Cloud-SDK v0.0.0-20251209081935-79e26e215136** in go.mod
4. **API credentials** for acceptance tests (ZILLAFORGE_API_TOKEN, ZILLAFORGE_API_ENDPOINT)

---

## Project Structure

```
internal/vps/
├── model/
│   └── floating_ip.go          # NEW: Shared data models
├── resource/
│   ├── floating_ip_resource.go # NEW: Resource implementation
│   └── floating_ip_resource_test.go # NEW: Acceptance tests
└── data/
    ├── floating_ips_data_source.go # NEW: Data source implementation
    └── floating_ips_data_source_test.go # NEW: Acceptance tests
```

---

## Step 1: Define Models

**File**: `internal/vps/model/floating_ip.go`

```go
package model

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zillaforge/cloud-sdk/vps"
)

// FloatingIPResourceModel represents the Terraform resource state
type FloatingIPResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IPAddress   types.String `tfsdk:"ip_address"`
	Status      types.String `tfsdk:"status"`
	DeviceID    types.String `tfsdk:"device_id"`
}

// FloatingIPDataSourceModel represents the data source state
type FloatingIPDataSourceModel struct {
	ID          types.String      `tfsdk:"id"`
	Name        types.String      `tfsdk:"name"`
	IPAddress   types.String      `tfsdk:"ip_address"`
	Status      types.String      `tfsdk:"status"`
	FloatingIPs []FloatingIPModel `tfsdk:"floating_ips"`
}

// FloatingIPModel represents a single floating IP in the data source results
type FloatingIPModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IPAddress   types.String `tfsdk:"ip_address"`
	Status      types.String `tfsdk:"status"`
	DeviceID    types.String `tfsdk:"device_id"`
}

// MapFloatingIPToResourceModel converts SDK FloatingIP to resource model
func MapFloatingIPToResourceModel(ctx context.Context, fip *vps.FloatingIP, data *FloatingIPResourceModel) {
	data.ID = types.StringValue(fip.ID)
	data.Name = stringPointerOrNull(fip.Name)
	data.Description = stringPointerOrNull(fip.Description)
	data.IPAddress = types.StringValue(fip.IPAddress)
	data.Status = types.StringValue(fip.Status)
	data.DeviceID = stringPointerOrNull(fip.DeviceID)
}

// MapFloatingIPToModel converts SDK FloatingIP to data source model
func MapFloatingIPToModel(fip *vps.FloatingIP) FloatingIPModel {
	return FloatingIPModel{
		ID:          types.StringValue(fip.ID),
		Name:        stringPointerOrNull(fip.Name),
		Description: stringPointerOrNull(fip.Description),
		IPAddress:   types.StringValue(fip.IPAddress),
		Status:      types.StringValue(fip.Status),
		DeviceID:    stringPointerOrNull(fip.DeviceID),
	}
}

// BuildCreateRequest creates FloatingIPCreateRequest from resource model
func BuildCreateRequest(data *FloatingIPResourceModel) *vps.FloatingIPCreateRequest {
	req := &vps.FloatingIPCreateRequest{}
	
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		name := data.Name.ValueString()
		req.Name = &name
	}
	
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		req.Description = &desc
	}
	
	return req
}

// BuildUpdateRequest creates FloatingIPUpdateRequest from resource model
func BuildUpdateRequest(data *FloatingIPResourceModel) *vps.FloatingIPUpdateRequest {
	req := &vps.FloatingIPUpdateRequest{}
	
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		name := data.Name.ValueString()
		req.Name = &name
	}
	
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		req.Description = &desc
	}
	
	return req
}

// FilterFloatingIPs applies client-side filtering to floating IP list
func FilterFloatingIPs(fips []*vps.FloatingIP, filters *FloatingIPDataSourceModel) []*vps.FloatingIP {
	var filtered []*vps.FloatingIP
	
	for _, fip := range fips {
		if matchesFilters(fip, filters) {
			filtered = append(filtered, fip)
		}
	}
	
	return filtered
}

// matchesFilters checks if floating IP matches all specified filters (AND logic)
func matchesFilters(fip *vps.FloatingIP, filters *FloatingIPDataSourceModel) bool {
	// ID filter
	if !filters.ID.IsNull() && !filters.ID.IsUnknown() {
		if fip.ID != filters.ID.ValueString() {
			return false
		}
	}
	
	// Name filter
	if !filters.Name.IsNull() && !filters.Name.IsUnknown() {
		if fip.Name == nil || *fip.Name != filters.Name.ValueString() {
			return false
		}
	}
	
	// IP Address filter
	if !filters.IPAddress.IsNull() && !filters.IPAddress.IsUnknown() {
		if fip.IPAddress != filters.IPAddress.ValueString() {
			return false
		}
	}
	
	// Status filter
	if !filters.Status.IsNull() && !filters.Status.IsUnknown() {
		if fip.Status != filters.Status.ValueString() {
			return false
		}
	}
	
	return true
}

// stringPointerOrNull converts string pointer to types.String (null if nil)
func stringPointerOrNull(s *string) types.String {
	if s == nil || *s == "" {
		return types.StringNull()
	}
	return types.StringValue(*s)
}
```

---

## Step 2: Implement Resource

**File**: `internal/vps/resource/floating_ip_resource.go`

```go
package resource

import (
	"context"
	"fmt"
	
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	
	"github.com/zillaforge/cloud-sdk/vps"
	"terraform-provider-zillaforge/internal/provider"
	"terraform-provider-zillaforge/internal/vps/model"
)

var (
	_ resource.Resource                = &FloatingIPResource{}
	_ resource.ResourceWithConfigure   = &FloatingIPResource{}
	_ resource.ResourceWithImportState = &FloatingIPResource{}
)

func NewFloatingIPResource() resource.Resource {
	return &FloatingIPResource{}
}

type FloatingIPResource struct {
	client *vps.Client
}

func (r *FloatingIPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_floating_ip"
}

func (r *FloatingIPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a ZillaForge floating IP. Allocates a public IP address from the default pool.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the floating IP.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Optional human-readable name for the floating IP. Can be updated in-place.",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description for the floating IP. Can be updated in-place.",
				Optional:            true,
				Computed:            true,
			},
			"ip_address": schema.StringAttribute{
				MarkdownDescription: "Public IPv4 address allocated from the pool.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Current status of the floating IP. Possible values: ACTIVE, DOWN, PENDING, REJECTED.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"device_id": schema.StringAttribute{
				MarkdownDescription: "Device ID of the associated VPS instance. Null when not associated. Association is managed outside Terraform.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *FloatingIPResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*provider.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *provider.ClientConfig, got: %T", req.ProviderData),
		)
		return
	}

	r.client = clients.VPSClient
}

func (r *FloatingIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data model.FloatingIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build create request
	createReq := model.BuildCreateRequest(&data)

	// Call API
	fip, err := r.client.CreateFloatingIP(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Floating IP Allocation Failed",
			fmt.Sprintf("Unable to allocate floating IP: %s", err.Error()),
		)
		return
	}

	// Map to state (status is just informational)
	model.MapFloatingIPToResourceModel(ctx, fip, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FloatingIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data model.FloatingIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	fip, err := r.client.GetFloatingIP(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Floating IP Not Found",
			fmt.Sprintf("Floating IP with ID '%s' not found: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	// Map to state
	model.MapFloatingIPToResourceModel(ctx, fip, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FloatingIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data model.FloatingIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request
	updateReq := model.BuildUpdateRequest(&data)

	// Call API
	fip, err := r.client.UpdateFloatingIP(ctx, data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Floating IP Update Failed",
			fmt.Sprintf("Unable to update floating IP: %s", err.Error()),
		)
		return
	}

	// Map to state
	model.MapFloatingIPToResourceModel(ctx, fip, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FloatingIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data model.FloatingIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	err := r.client.DeleteFloatingIP(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Floating IP Deletion Failed",
			fmt.Sprintf("Unable to delete floating IP: %s", err.Error()),
		)
		return
	}
}

func (r *FloatingIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

---

## Step 3: Implement Data Source

**File**: `internal/vps/data/floating_ips_data_source.go`

```go
package data

import (
	"context"
	"fmt"
	
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	
	"github.com/zillaforge/cloud-sdk/vps"
	"terraform-provider-zillaforge/internal/provider"
	"terraform-provider-zillaforge/internal/vps/model"
)

var (
	_ datasource.DataSource              = &FloatingIPsDataSource{}
	_ datasource.DataSourceWithConfigure = &FloatingIPsDataSource{}
)

func NewFloatingIPsDataSource() datasource.DataSource {
	return &FloatingIPsDataSource{}
}

type FloatingIPsDataSource struct {
	client *vps.Client
}

func (d *FloatingIPsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_floating_ips"
}

func (d *FloatingIPsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Queries ZillaForge floating IPs with optional filters. Returns all floating IPs if no filters are specified.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Filter by exact floating IP ID.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Filter by exact name (case-sensitive).",
				Optional:            true,
			},
			"ip_address": schema.StringAttribute{
				MarkdownDescription: "Filter by exact IP address.",
				Optional:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Filter by status (ACTIVE, DOWN, PENDING, REJECTED).",
				Optional:            true,
			},
			"floating_ips": schema.ListNestedAttribute{
				MarkdownDescription: "List of floating IPs matching the filters. Empty list if no matches found.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for the floating IP.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Human-readable name for the floating IP.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Description for the floating IP.",
							Computed:            true,
						},
						"ip_address": schema.StringAttribute{
							MarkdownDescription: "Public IPv4 address.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Current status (ACTIVE, DOWN, PENDING, REJECTED).",
							Computed:            true,
						},
						"device_id": schema.StringAttribute{
							MarkdownDescription: "Associated device ID (null when unassociated).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *FloatingIPsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*provider.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *provider.ClientConfig, got: %T", req.ProviderData),
		)
		return
	}

	d.client = clients.VPSClient
}

func (d *FloatingIPsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data model.FloatingIPDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to get all floating IPs (SDK filter has bugs)
	allFips, err := d.client.ListFloatingIPs(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Floating IPs",
			fmt.Sprintf("Failed to list floating IPs: %s", err.Error()),
		)
		return
	}

	// Apply client-side filtering
	filteredFips := model.FilterFloatingIPs(allFips, &data)

	// Convert to model
	data.FloatingIPs = make([]model.FloatingIPModel, 0, len(filteredFips))
	for _, fip := range filteredFips {
		data.FloatingIPs = append(data.FloatingIPs, model.MapFloatingIPToModel(fip))
	}

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

---

## Step 4: Register with Provider

**File**: `internal/provider/provider.go`

Add to `Resources()` method:
```go
func (p *zillaforgeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// ... existing resources ...
		resource.NewFloatingIPResource,  // ADD THIS
	}
}
```

Add to `DataSources()` method:
```go
func (p *zillaforgeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// ... existing data sources ...
		data.NewFloatingIPsDataSource,  // ADD THIS
	}
}
```

---

## Step 5: Write Acceptance Tests

**File**: `internal/vps/resource/floating_ip_resource_test.go`

```go
package resource_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"terraform-provider-zillaforge/internal/provider"
)

func TestAccFloatingIPResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without name/description
			{
				Config: testAccFloatingIPResourceConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test", "id"),
					resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test", "ip_address"),
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test", "status", "ACTIVE"),
				),
			},
			// Import
			{
				ResourceName:      "zillaforge_floating_ip.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFloatingIPResource_WithNameDescription(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with name/description
			{
				Config: testAccFloatingIPResourceConfig_named(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test", "name", "test-ip"),
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test", "description", "test description"),
				),
			},
			// Update name/description
			{
				Config: testAccFloatingIPResourceConfig_updated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test", "name", "updated-ip"),
					resource.TestCheckResourceAttr("zillaforge_floating_ip.test", "description", "updated description"),
				),
			},
		},
	})
}

func testAccFloatingIPResourceConfig_basic() string {
	return `
resource "zillaforge_floating_ip" "test" {}
`
}

func testAccFloatingIPResourceConfig_named() string {
	return `
resource "zillaforge_floating_ip" "test" {
  name        = "test-ip"
  description = "test description"
}
`
}

func testAccFloatingIPResourceConfig_updated() string {
	return `
resource "zillaforge_floating_ip" "test" {
  name        = "updated-ip"
  description = "updated description"
}
`
}
```

**File**: `internal/vps/data/floating_ips_data_source_test.go`

```go
package data_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"terraform-provider-zillaforge/internal/provider"
)

func TestAccFloatingIPsDataSource_All(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPsDataSourceConfig_all(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zillaforge_floating_ips.test", "floating_ips.#"),
				),
			},
		},
	})
}

func TestAccFloatingIPsDataSource_FilterByID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFloatingIPsDataSourceConfig_filterByID(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zillaforge_floating_ips.test", "floating_ips.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.zillaforge_floating_ips.test", "floating_ips.0.id",
						"zillaforge_floating_ip.test", "id",
					),
				),
			},
		},
	})
}

func testAccFloatingIPsDataSourceConfig_all() string {
	return `
data "zillaforge_floating_ips" "test" {}
`
}

func testAccFloatingIPsDataSourceConfig_filterByID() string {
	return `
resource "zillaforge_floating_ip" "test" {
  name = "data-source-test"
}

data "zillaforge_floating_ips" "test" {
  id = zillaforge_floating_ip.test.id
}
`
}
```

---

## Step 6: Run Tests

### Unit Tests (if any)
```bash
cd /workspaces/terraform-provider-zillaforge
go test ./internal/vps/model -v
```

### Acceptance Tests

**Resource tests**:
```bash
make testacc TESTARGS='-run=TestAccFloatingIPResource' PARALLEL=1
```

**Data source tests**:
```bash
make testacc TESTARGS='-run=TestAccFloatingIPsDataSource' PARALLEL=1
```

**Single test**:
```bash
make testacc TESTARGS='-run=TestAccFloatingIPResource_Basic' PARALLEL=1
```

**All floating IP tests**:
```bash
make testacc TESTARGS='-run=TestAccFloatingIP' PARALLEL=1
```

---

## Step 7: Generate Documentation

After implementation passes tests:

```bash
make generate
```

This creates:
- `docs/resources/floating_ip.md`
- `docs/data-sources/floating_ips.md`

**DO NOT manually edit documentation** - it's generated by `tfplugindocs`.

---

## Common Patterns

### Pattern 1: Optional + Computed Attributes

```go
"name": schema.StringAttribute{
	MarkdownDescription: "Optional name...",
	Optional:            true,  // User CAN provide
	Computed:            true,  // API CAN provide default
},
```

### Pattern 2: Computed-Only Attributes

```go
"ip_address": schema.StringAttribute{
	MarkdownDescription: "Public IP address...",
	Computed:            true,
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.UseStateForUnknown(),  // Preserve during plan
	},
},
```

### Pattern 3: Client-Side Filtering

```go
// Fetch all from API
allItems, err := client.ListFloatingIPs(ctx)

// Apply filters locally (SDK List() filter has bugs)
filtered := model.FilterFloatingIPs(allItems, &filters)
```

### Pattern 4: Null Handling

```go
func stringPointerOrNull(s *string) types.String {
	if s == nil || *s == "" {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// Usage
data.DeviceID = stringPointerOrNull(fip.DeviceID)
```

---

## Testing Checklist

Before marking implementation complete:

- [ ] Resource: Create without name/description
- [ ] Resource: Create with name/description
- [ ] Resource: Update name/description (in-place)
- [ ] Resource: Import by ID
- [ ] Resource: Delete (even if associated)
- [ ] Data Source: List all floating IPs
- [ ] Data Source: Filter by ID
- [ ] Data Source: Filter by name
- [ ] Data Source: Filter by IP address
- [ ] Data Source: Filter by status
- [ ] Data Source: Multiple filters (AND logic)
- [ ] Data Source: No matches returns empty list
- [ ] Documentation generated by `make generate`
- [ ] All MarkdownDescription fields present

---

## Troubleshooting

### Test Hangs

```bash
# Kill hung tests
pkill -f "go test"

# Run with verbose output
make testacc TESTARGS='-run=TestAccFloatingIP -v' PARALLEL=1
```

### API Credentials

Set environment variables:
```bash
export ZILLAFORGE_API_TOKEN="your-token"
export ZILLAFORGE_API_ENDPOINT="https://api.zillaforge.com"
```

### SDK Field Names

Check SDK struct for exact field names:
```go
import "github.com/zillaforge/cloud-sdk/vps"

// Verify fields match:
// vps.FloatingIP
// vps.FloatingIPCreateRequest
// vps.FloatingIPUpdateRequest
```

### Client-Side Filtering Not Working

Ensure filters use AND logic and exact match:
```go
// This matches: name=="prod-ip" AND status=="ACTIVE"
filters := &model.FloatingIPDataSourceModel{
	Name:   types.StringValue("prod-ip"),
	Status: types.StringValue("ACTIVE"),
}
```

---

## Next Steps

After implementation completes:

1. **Run full test suite**: `make testacc PARALLEL=1`
2. **Generate documentation**: `make generate`
3. **Update CHANGELOG.md**: Add feature under "Unreleased"
4. **Create example**: Add to `examples/resources/zillaforge_floating_ip/`
5. **PR review**: Follow constitution checklist

---

## References

- [Terraform Plugin Framework Docs](https://developer.hashicorp.com/terraform/plugin/framework)
- [Cloud-SDK VPS Package](https://github.com/zillaforge/cloud-sdk/tree/main/vps)
- [Constitution](.specify/memory/constitution.md)
- [Research Document](./research.md)
- [Data Model](./data-model.md)
- [API Contract](./contracts/cloud-sdk-api.md)
- [Terraform Schema](./contracts/terraform-schema.md)
