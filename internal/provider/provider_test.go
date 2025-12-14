// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	frameworkProvider "github.com/hashicorp/terraform-plugin-framework/provider"
	// Provider testing helpers are available in internal/provider/testing.go.
)

// Note: use provider.TestAccProtoV6ProviderFactories and provider.TestAccProtoV6ProviderFactoriesWithEcho
// from internal/provider/testing.go when writing tests requiring provider factories.

// generateTestJWT creates a signed JWT token for use in tests.
// It uses HS256 signing with the provided secret (or "test-secret" if empty)
// and a short lived expiration time.
func generateTestJWT(t *testing.T, secret string) string {
	if secret == "" {
		secret = "test-secret"
	}

	claims := jwt.MapClaims{
		"sub": "test-user",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	s, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign test jwt: %v", err)
	}

	return s
}

func TestZillaforgeProvider_Metadata(t *testing.T) {
	// This test verifies the provider TypeName will be "zillaforge" after renaming.
	ctx := context.Background()
	provider := New("test")()

	req := frameworkProvider.MetadataRequest{}
	resp := &frameworkProvider.MetadataResponse{}

	provider.Metadata(ctx, req, resp)

	if resp.TypeName != "zillaforge" {
		t.Fatalf("Expected TypeName 'zillaforge', got '%s'", resp.TypeName)
	}
}

func TestZillaforgeProvider_Configure_InitializesSDK(t *testing.T) {
	// This test verifies Configure() successfully sets up SDK client when
	// SDK initialization succeeds. We inject a test double via
	// `newClientWrapper` to avoid making network calls.
	ctx := context.Background()
	prov := New("test")()

	validJWT := generateTestJWT(t, "")
	t.Setenv("ZILLAFORGE_API_KEY", validJWT)
	t.Setenv("ZILLAFORGE_PROJECT_ID", "test-project-123")
	// Ensure sys code isn't set in the environment for this test
	t.Setenv("ZILLAFORGE_PROJECT_SYS_CODE", "")

	// Inject a stub that returns a fake project client without error
	oldFactory := newClientWrapper
	defer func() { newClientWrapper = oldFactory }()

	newClientWrapper = func(apiEndpoint, apiKey string) clientWrapper {
		return &testClient{projectResult: struct{}{}, projectErr: nil}
	}

	req := frameworkProvider.ConfigureRequest{}
	resp := &frameworkProvider.ConfigureResponse{}
	prov.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Expected no diagnostics error, got: %v", resp.Diagnostics.Errors())
	}

	if resp.DataSourceData == nil || resp.ResourceData == nil {
		t.Fatalf("Expected DataSourceData and ResourceData to be set after successful Configure")
	}
}

func TestZillaforgeProvider_Configure_InvalidCredentials(t *testing.T) {
	// This test verifies Configure() surfaces SDK initialization errors
	// as diagnostics when the SDK fails to initialize (e.g., invalid creds).
	ctx := context.Background()
	prov := New("test")()

	validJWT := generateTestJWT(t, "")
	t.Setenv("ZILLAFORGE_API_KEY", validJWT)
	t.Setenv("ZILLAFORGE_PROJECT_ID", "test-project-123")
	// Ensure sys code isn't set in the environment for this test
	t.Setenv("ZILLAFORGE_PROJECT_SYS_CODE", "")

	// Inject a stub client that returns an error from Project()
	oldFactory := newClientWrapper
	defer func() { newClientWrapper = oldFactory }()

	newClientWrapper = func(apiEndpoint, apiKey string) clientWrapper {
		return &testClient{projectResult: nil, projectErr: fmt.Errorf("simulated SDK error")}
	}

	req := frameworkProvider.ConfigureRequest{}
	resp := &frameworkProvider.ConfigureResponse{}
	prov.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("Expected diagnostics error due to SDK init failure")
	}

	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "SDK Initialization Failed" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected 'SDK Initialization Failed' diagnostic, got: %v", resp.Diagnostics.Errors())
	}
}

// testClient is a small test double implementing clientWrapper used in
// unit tests above to control Project() behavior.
type testClient struct {
	projectResult interface{}
	projectErr    error
}

func (t *testClient) Project(ctx context.Context, projectIDOrCode string) (interface{}, error) {
	return t.projectResult, t.projectErr
}

// T039: Test both project identifiers provided (should fail).
func TestZillaforgeProvider_Schema_BothProjectIdentifiers(t *testing.T) {
	ctx := context.Background()
	prov := New("test")()

	validJWT := generateTestJWT(t, "")
	t.Setenv("ZILLAFORGE_API_KEY", validJWT)
	t.Setenv("ZILLAFORGE_PROJECT_ID", "test-project-123")
	t.Setenv("ZILLAFORGE_PROJECT_SYS_CODE", "TEST-CODE-456")

	req := frameworkProvider.ConfigureRequest{}
	resp := &frameworkProvider.ConfigureResponse{}

	prov.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Expected error when both project_id and project_sys_code are set")
	}

	// Verify error message contains expected text
	foundError := false
	for _, diag := range resp.Diagnostics.Errors() {
		if diag.Summary() == "Conflicting Project Identifiers" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Fatalf("Expected 'Conflicting Project Identifiers' error, got: %v", resp.Diagnostics.Errors())
	}
}

// T040: Test neither project identifier provided (should fail).
func TestZillaforgeProvider_Schema_NeitherProjectIdentifier(t *testing.T) {
	ctx := context.Background()
	prov := New("test")()

	validJWT := generateTestJWT(t, "")
	t.Setenv("ZILLAFORGE_API_KEY", validJWT)
	// Ensure no project identifiers are set in the environment for deterministic testing
	t.Setenv("ZILLAFORGE_PROJECT_ID", "")
	t.Setenv("ZILLAFORGE_PROJECT_SYS_CODE", "")

	req := frameworkProvider.ConfigureRequest{}
	resp := &frameworkProvider.ConfigureResponse{}

	prov.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Expected error when neither project_id nor project_sys_code are set")
	}

	// Verify error message contains expected text
	foundError := false
	for _, diag := range resp.Diagnostics.Errors() {
		if diag.Summary() == "Missing Project Identifier" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Fatalf("Expected 'Missing Project Identifier' error, got: %v", resp.Diagnostics.Errors())
	}
}

// T041: Test valid configuration with project_id (should succeed).
func TestZillaforgeProvider_Schema_ValidWithProjectID(t *testing.T) {
	ctx := context.Background()
	prov := New("test")()

	validJWT := generateTestJWT(t, "")
	t.Setenv("ZILLAFORGE_API_KEY", validJWT)
	t.Setenv("ZILLAFORGE_PROJECT_ID", "test-project-123")
	// Clear sys code to avoid interference from environment
	t.Setenv("ZILLAFORGE_PROJECT_SYS_CODE", "")

	// Inject a stub client so SDK initialization succeeds during unit tests
	oldFactory := newClientWrapper
	defer func() { newClientWrapper = oldFactory }()

	newClientWrapper = func(apiEndpoint, apiKey string) clientWrapper {
		return &testClient{projectResult: struct{}{}, projectErr: nil}
	}

	req := frameworkProvider.ConfigureRequest{}
	resp := &frameworkProvider.ConfigureResponse{}

	prov.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Expected no diagnostics error, got: %v", resp.Diagnostics.Errors())
	}
}

// T042: Test valid configuration with project_sys_code (should succeed).
func TestZillaforgeProvider_Schema_ValidWithProjectSysCode(t *testing.T) {
	ctx := context.Background()
	prov := New("test")()

	validJWT := generateTestJWT(t, "")
	t.Setenv("ZILLAFORGE_API_KEY", validJWT)
	t.Setenv("ZILLAFORGE_PROJECT_SYS_CODE", "TEST-CODE-456")
	// Clear project id to avoid interference from environment
	t.Setenv("ZILLAFORGE_PROJECT_ID", "")

	// Inject a stub client so SDK initialization succeeds during unit tests
	oldFactory := newClientWrapper
	defer func() { newClientWrapper = oldFactory }()

	newClientWrapper = func(apiEndpoint, apiKey string) clientWrapper {
		return &testClient{projectResult: struct{}{}, projectErr: nil}
	}

	req := frameworkProvider.ConfigureRequest{}
	resp := &frameworkProvider.ConfigureResponse{}

	prov.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Expected no diagnostics error, got: %v", resp.Diagnostics.Errors())
	}
	if resp.Diagnostics.HasError() {
		// Check it's an SDK error, not a validation error
		for _, diag := range resp.Diagnostics.Errors() {
			if diag.Summary() == "Conflicting Project Identifiers" || diag.Summary() == "Missing Project Identifier" {
				t.Fatalf("Got unexpected validation error: %v", diag)
			}
		}
		// SDK initialization error is expected in unit tests
		t.Skip("SDK initialization fails in unit test (expected)")
	}
}

// T043: Test invalid JWT format (should fail).
func TestZillaforgeProvider_Schema_InvalidJWTFormat(t *testing.T) {
	ctx := context.Background()
	prov := New("test")()

	// Invalid JWT - not in header.payload.signature format
	t.Setenv("ZILLAFORGE_API_KEY", "not-a-valid-jwt-token")
	t.Setenv("ZILLAFORGE_PROJECT_ID", "test-project-123")

	req := frameworkProvider.ConfigureRequest{}
	resp := &frameworkProvider.ConfigureResponse{}

	prov.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Expected error for invalid JWT format")
	}

	// Verify error message indicates JWT validation issue
	foundError := false
	for _, diag := range resp.Diagnostics.Errors() {
		summary := diag.Summary()
		if summary == "Invalid API Key Format" || summary == "SDK Initialization Failed" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Fatalf("Expected JWT format validation error, got: %v", resp.Diagnostics.Errors())
	}
}

// T044: Test multiple provider instances with aliases.
func TestZillaforgeProvider_MultiInstance_Aliases(t *testing.T) {
	// This test verifies that multiple provider instances can coexist
	// Each instance should have independent SDK clients

	ctx := context.Background()

	// Create first provider instance
	prov1 := New("test")()
	validJWT1 := generateTestJWT(t, "secret1")

	// Create second provider instance
	prov2 := New("test")()
	validJWT2 := generateTestJWT(t, "secret2")

	// Inject a stub factory that returns a distinct fake client per apiKey so
	// we can validate that multiple instances configure independently.
	oldFactory := newClientWrapper
	defer func() { newClientWrapper = oldFactory }()

	newClientWrapper = func(apiEndpoint, apiKey string) clientWrapper {
		return &testClient{projectResult: apiKey, projectErr: nil}
	}

	// Configure first instance
	t.Setenv("ZILLAFORGE_API_KEY", validJWT1)
	// Ensure no sys code is set globally to avoid interfering with validation
	t.Setenv("ZILLAFORGE_PROJECT_SYS_CODE", "")
	t.Setenv("ZILLAFORGE_PROJECT_ID", "project-1")

	req1 := frameworkProvider.ConfigureRequest{}
	resp1 := &frameworkProvider.ConfigureResponse{}
	prov1.Configure(ctx, req1, resp1)

	// Configure second instance with different values
	t.Setenv("ZILLAFORGE_API_KEY", validJWT2)
	t.Setenv("ZILLAFORGE_PROJECT_ID", "project-2")

	req2 := frameworkProvider.ConfigureRequest{}
	resp2 := &frameworkProvider.ConfigureResponse{}
	prov2.Configure(ctx, req2, resp2)

	if resp1.Diagnostics.HasError() {
		t.Fatalf("Provider 1 got unexpected diagnostics: %v", resp1.Diagnostics.Errors())
	}

	if resp2.Diagnostics.HasError() {
		t.Fatalf("Provider 2 got unexpected diagnostics: %v", resp2.Diagnostics.Errors())
	}

	// Verify the DataSourceData/ResourceData reflect distinct clients
	if fmt.Sprint(resp1.DataSourceData) == fmt.Sprint(resp2.DataSourceData) {
		t.Fatalf("Expected distinct SDK clients per provider instance, got identical values")
	}
}
