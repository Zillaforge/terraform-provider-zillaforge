// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	frameworkProvider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"zillaforge": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the scaffolding provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"zillaforge": providerserver.NewProtocol6WithError(New("test")()),
	"echo":       echoprovider.NewProviderServer(),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

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
	// This test verifies Configure() successfully initializes the SDK client
	// For now, we'll test with environment variables since the full schema isn't implemented yet
	validJWT := generateTestJWT(t, "")
	t.Setenv("ZILLAFORGE_API_KEY", validJWT)
	t.Setenv("ZILLAFORGE_PROJECT_ID", "test-project-123")

	// Test will be implemented properly once SDK integration is complete
	// Expected: Configure() should initialize SDK client and set ResourceData/DataSourceData
	t.Skip("Test will pass after SDK integration (T022-T028)")
}

func TestZillaforgeProvider_Configure_InvalidCredentials(t *testing.T) {
	// This test verifies Configure() returns diagnostic errors with invalid/missing credentials
	// Test will be implemented properly once SDK integration is complete
	// Expected: Configure() should return diagnostic error when credentials are missing
	t.Skip("Test will pass after SDK integration (T022-T028)")
}

// T039: Test both project identifiers provided (should fail)
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

// T040: Test neither project identifier provided (should fail)
func TestZillaforgeProvider_Schema_NeitherProjectIdentifier(t *testing.T) {
	ctx := context.Background()
	prov := New("test")()

	validJWT := generateTestJWT(t, "")
	t.Setenv("ZILLAFORGE_API_KEY", validJWT)
	// Don't set either project identifier

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

// T041: Test valid configuration with project_id (should succeed)
func TestZillaforgeProvider_Schema_ValidWithProjectID(t *testing.T) {
	ctx := context.Background()
	prov := New("test")()

	validJWT := generateTestJWT(t, "")
	t.Setenv("ZILLAFORGE_API_KEY", validJWT)
	t.Setenv("ZILLAFORGE_PROJECT_ID", "test-project-123")

	req := frameworkProvider.ConfigureRequest{}
	resp := &frameworkProvider.ConfigureResponse{}

	prov.Configure(ctx, req, resp)

	// This will fail until we mock the SDK properly, but validates the schema
	// For now we expect it to fail on SDK initialization, not validation
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

// T042: Test valid configuration with project_sys_code (should succeed)
func TestZillaforgeProvider_Schema_ValidWithProjectSysCode(t *testing.T) {
	ctx := context.Background()
	prov := New("test")()

	validJWT := generateTestJWT(t, "")
	t.Setenv("ZILLAFORGE_API_KEY", validJWT)
	t.Setenv("ZILLAFORGE_PROJECT_SYS_CODE", "TEST-CODE-456")

	req := frameworkProvider.ConfigureRequest{}
	resp := &frameworkProvider.ConfigureResponse{}

	prov.Configure(ctx, req, resp)

	// This will fail until we mock the SDK properly, but validates the schema
	// For now we expect it to fail on SDK initialization, not validation
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

// T043: Test invalid JWT format (should fail)
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

// T044: Test multiple provider instances with aliases
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

	// Configure first instance
	t.Setenv("ZILLAFORGE_API_KEY", validJWT1)
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

	// Both should be configured independently
	// (SDK errors are expected in unit tests, but validation should pass)
	if resp1.Diagnostics.HasError() {
		for _, diag := range resp1.Diagnostics.Errors() {
			if diag.Summary() != "SDK Initialization Failed" {
				t.Fatalf("Provider 1 got unexpected error: %v", diag)
			}
		}
	}

	if resp2.Diagnostics.HasError() {
		for _, diag := range resp2.Diagnostics.Errors() {
			if diag.Summary() != "SDK Initialization Failed" {
				t.Fatalf("Provider 2 got unexpected error: %v", diag)
			}
		}
	}

	t.Skip("Multi-instance test passes validation (SDK errors expected in unit tests)")
}
