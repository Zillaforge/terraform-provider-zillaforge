// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// skipIfNoMatchingImages queries VRM tags directly and skips the test when there are no tags matching the provided filters in the current test environment. This avoids passing tests that relied on an empty result set.
func skipIfNoMatchingImages(t *testing.T, repo, tag, pattern string) {
	apiEndpoint := os.Getenv("ZILLAFORGE_API_ENDPOINT")
	apiKey := os.Getenv("ZILLAFORGE_API_KEY")
	projectID := os.Getenv("ZILLAFORGE_PROJECT_ID")
	projectSys := os.Getenv("ZILLAFORGE_PROJECT_SYS_CODE")
	project := projectID
	if project == "" {
		project = projectSys
	}

	if apiKey == "" || project == "" {
		t.Skip("Zillaforge API credentials or project not configured; skipping acceptance test")
	}

	client := cloudsdk.NewClient(apiEndpoint, apiKey)
	proj, err := client.Project(context.Background(), project)
	if err != nil {
		t.Skipf("unable to initialize SDK client: %v; skipping acceptance test", err)
	}

	vrm := proj.VRM()
	tags, err := vrm.Tags().List(context.Background(), nil)
	if err != nil {
		t.Skipf("unable to list tags from VRM: %v; skipping acceptance test", err)
	}

	// Apply same client-side filtering as the data source
	for _, tt := range tags {
		if repo != "" {
			if tt.Repository == nil || tt.Repository.Name != repo {
				continue
			}
		}
		if tag != "" {
			if tt.Name != tag {
				continue
			}
		}
		if pattern != "" {
			matched, err := filepath.Match(pattern, tt.Name)
			if err != nil {
				continue
			}
			if !matched {
				continue
			}
		}
		return
	}

	t.Skip("No matching images in this environment; skipping acceptance test")
}

// T010: Acceptance test - Query images with both repository and tag filters.
// Expected: Returns a list containing exactly one image (length 1) for the specified repository:tag.
func TestAccImagesDataSource_RepositoryAndTag(t *testing.T) {
	repo := os.Getenv("TF_ACC_IMAGES_REPOSITORY")
	tag := os.Getenv("TF_ACC_IMAGES_TAG")
	if repo == "" || tag == "" {
		t.Skip("TF_ACC_IMAGES_REPOSITORY and TF_ACC_IMAGES_TAG must be set for this test")
	}

	cfg := fmt.Sprintf(`data "zillaforge_images" "test" {
  repository = "%s"
  tag        = "%s"
}
`, repo, tag)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			provider.TestAccPreCheck(t)
			skipIfNoMatchingImages(t, repo, tag, "")
		},
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zillaforge_images.test", "images.#"),
					resource.TestCheckFunc(func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.zillaforge_images.test"]
						if !ok {
							return fmt.Errorf("data source not found")
						}
						if rs.Primary.Attributes["images.0.id"] == "" {
							return fmt.Errorf("images.0.id expected to be set")
						}
						if rs.Primary.Attributes["images.0.repository_name"] != repo {
							return fmt.Errorf("images.0.repository_name expected %s, got %s", repo, rs.Primary.Attributes["images.0.repository_name"])
						}
						if rs.Primary.Attributes["images.0.tag_name"] != tag {
							return fmt.Errorf("images.0.tag_name expected %s, got %s", tag, rs.Primary.Attributes["images.0.tag_name"])
						}
						if rs.Primary.Attributes["images.0.size"] == "" {
							return fmt.Errorf("images.0.size expected to be set")
						}
						if rs.Primary.Attributes["images.0.operating_system"] == "" {
							return fmt.Errorf("images.0.operating_system expected to be set")
						}
						if rs.Primary.Attributes["images.0.type"] == "" {
							return fmt.Errorf("images.0.type expected to be set")
						}
						if rs.Primary.Attributes["images.0.status"] == "" {
							return fmt.Errorf("images.0.status expected to be set")
						}
						return nil
					}),
				),
			},
		},
	})
}

// T011: Acceptance test - Query images with repository filter only.
// Expected: Returns all tags for that repository sorted deterministically.
func TestAccImagesDataSource_RepositoryOnly(t *testing.T) {
	repo := os.Getenv("TF_ACC_IMAGES_REPOSITORY")
	if repo == "" {
		t.Skip("TF_ACC_IMAGES_REPOSITORY must be set for this test")
	}

	cfg := fmt.Sprintf(`data "zillaforge_images" "test" {
  repository = "%s"
}
`, repo)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			provider.TestAccPreCheck(t)
			skipIfNoMatchingImages(t, repo, "", "")
		},
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zillaforge_images.test", "images.#"),
					resource.TestCheckFunc(func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.zillaforge_images.test"]
						if !ok {
							return fmt.Errorf("data source not found")
						}
						if rs.Primary.Attributes["images.0.id"] == "" {
							return fmt.Errorf("images.0.id expected to be set")
						}
						if rs.Primary.Attributes["images.0.repository_name"] != repo {
							return fmt.Errorf("images.0.repository_name expected %s, got %s", repo, rs.Primary.Attributes["images.0.repository_name"])
						}
						if rs.Primary.Attributes["images.0.tag_name"] == "" {
							return fmt.Errorf("images.0.tag_name expected to be set")
						}
						if rs.Primary.Attributes["images.0.size"] == "" {
							return fmt.Errorf("images.0.size expected to be set")
						}
						return nil
					}),
				),
			},
		},
	})
}

// T012: Acceptance test - Query images with tag filter only.
// Expected: Returns matching tags across all repositories.
func TestAccImagesDataSource_TagOnly(t *testing.T) {
	tag := os.Getenv("TF_ACC_IMAGES_TAG")
	if tag == "" {
		t.Skip("TF_ACC_IMAGES_TAG must be set for this test")
	}

	cfg := fmt.Sprintf(`data "zillaforge_images" "test" {
  tag = "%s"
}
`, tag)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			provider.TestAccPreCheck(t)
			skipIfNoMatchingImages(t, "", tag, "")
		},
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zillaforge_images.test", "images.#"),
					resource.TestCheckFunc(func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.zillaforge_images.test"]
						if !ok {
							return fmt.Errorf("data source not found")
						}
						if rs.Primary.Attributes["images.0.id"] == "" {
							return fmt.Errorf("images.0.id expected to be set")
						}
						if rs.Primary.Attributes["images.0.tag_name"] != tag {
							return fmt.Errorf("images.0.tag_name expected %s, got %s", tag, rs.Primary.Attributes["images.0.tag_name"])
						}
						if rs.Primary.Attributes["images.0.repository_name"] == "" {
							return fmt.Errorf("images.0.repository_name expected to be set")
						}
						return nil
					}),
				),
			},
		},
	})
}

// T013: Acceptance test - List all images (no filters).
// Expected: Returns all available images sorted deterministically (up to server limit).
func TestAccImagesDataSource_NoFilters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			provider.TestAccPreCheck(t)
			skipIfNoMatchingImages(t, "", "", "")
		},
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImagesDataSourceConfig_noFilters,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zillaforge_images.test", "images.#"),
					resource.TestCheckFunc(func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.zillaforge_images.test"]
						if !ok {
							return fmt.Errorf("data source not found")
						}
						if rs.Primary.Attributes["images.0.id"] == "" {
							return fmt.Errorf("images.0.id expected to be set")
						}
						if rs.Primary.Attributes["images.0.repository_name"] == "" {
							return fmt.Errorf("images.0.repository_name expected to be set")
						}
						if rs.Primary.Attributes["images.0.tag_name"] == "" {
							return fmt.Errorf("images.0.tag_name expected to be set")
						}
						if rs.Primary.Attributes["images.0.size"] == "" {
							return fmt.Errorf("images.0.size expected to be set")
						}
						if rs.Primary.Attributes["images.0.operating_system"] == "" {
							return fmt.Errorf("images.0.operating_system expected to be set")
						}
						if rs.Primary.Attributes["images.0.type"] == "" {
							return fmt.Errorf("images.0.type expected to be set")
						}
						if rs.Primary.Attributes["images.0.status"] == "" {
							return fmt.Errorf("images.0.status expected to be set")
						}
						return nil
					}),
				),
			},
		},
	})
}

const testAccImagesDataSourceConfig_noFilters = `
data "zillaforge_images" "test" {
  # No filters specified - returns all images
}
`

// T014: Acceptance test - Verify all image attributes are accessible.
// Expected: All 8 required attributes can be referenced.
func TestAccImagesDataSource_AttributeReference(t *testing.T) {
	repo := os.Getenv("TF_ACC_IMAGES_REPOSITORY")
	tag := os.Getenv("TF_ACC_IMAGES_TAG")
	if repo == "" || tag == "" {
		t.Skip("TF_ACC_IMAGES_REPOSITORY and TF_ACC_IMAGES_TAG must be set for this test")
	}

	cfg := fmt.Sprintf(`data "zillaforge_images" "test" {
  repository = "%s"
  tag        = "%s"
}

# Verify attributes can be referenced in outputs
output "image_id" {
  value = length(data.zillaforge_images.test.images) > 0 ? data.zillaforge_images.test.images[0].id : ""
}

output "image_size" {
  value = length(data.zillaforge_images.test.images) > 0 ? data.zillaforge_images.test.images[0].size : 0
}
`, repo, tag)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			provider.TestAccPreCheck(t)
			skipIfNoMatchingImages(t, repo, tag, "")
		},
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zillaforge_images.test", "images.#"),
					resource.TestCheckFunc(func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.zillaforge_images.test"]
						if !ok {
							return fmt.Errorf("data source not found")
						}
						if rs.Primary.Attributes["images.0.id"] == "" {
							return fmt.Errorf("images.0.id expected to be set")
						}
						if rs.Primary.Attributes["images.0.repository_name"] == "" {
							return fmt.Errorf("images.0.repository_name expected to be set")
						}
						if rs.Primary.Attributes["images.0.tag_name"] == "" {
							return fmt.Errorf("images.0.tag_name expected to be set")
						}
						if rs.Primary.Attributes["images.0.size"] == "" {
							return fmt.Errorf("images.0.size expected to be set")
						}
						if rs.Primary.Attributes["images.0.operating_system"] == "" {
							return fmt.Errorf("images.0.operating_system expected to be set")
						}
						if rs.Primary.Attributes["images.0.type"] == "" {
							return fmt.Errorf("images.0.type expected to be set")
						}
						if rs.Primary.Attributes["images.0.status"] == "" {
							return fmt.Errorf("images.0.status expected to be set")
						}
						return nil
					}),
				),
			},
		},
	})
}

// T025: Acceptance test - Pattern with * wildcard.
// Expected: Returns all tags matching the wildcard pattern.
func TestAccImagesDataSource_TagPattern_Wildcard(t *testing.T) {
	pattern := os.Getenv("TF_ACC_IMAGES_TAG_PATTERN")
	if pattern == "" {
		t.Skip("TF_ACC_IMAGES_TAG_PATTERN must be set for this test (e.g., 'v1.*' or 'prod-*')")
	}

	cfg := fmt.Sprintf(`data "zillaforge_images" "test" {
  tag_pattern = "%s"
}
`, pattern)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			provider.TestAccPreCheck(t)
			pattern := os.Getenv("TF_ACC_IMAGES_TAG_PATTERN")
			skipIfNoMatchingImages(t, "", "", pattern)
		},
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zillaforge_images.test", "images.#"),
					resource.TestCheckFunc(func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.zillaforge_images.test"]
						if !ok {
							return fmt.Errorf("data source not found")
						}
						if rs.Primary.Attributes["images.0.tag_name"] == "" {
							return fmt.Errorf("images.0.tag_name expected to be set")
						}
						return nil
					}),
				),
			},
		},
	})
}

// T028: Acceptance test - Pattern with no matches.
// Expected: Returns empty list (not an error).
func TestAccImagesDataSource_TagPattern_NoMatches(t *testing.T) {
	cfg := `data "zillaforge_images" "test" {
  tag_pattern = "nonexistent-pattern-xyz-*"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zillaforge_images.test", "images.#", "0"),
				),
			},
		},
	})
}

// T042: Acceptance test - Non-existent repository.
// Expected: Returns empty list (not an error).
func TestAccImagesDataSource_NonExistentRepository(t *testing.T) {
	cfg := `data "zillaforge_images" "test" {
  repository = "nonexistent-repo-xyz-12345"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zillaforge_images.test", "images.#", "0"),
				),
			},
		},
	})
}

// T043: Acceptance test - Mutual exclusivity.
// Expected: Configuration error when both tag and tag_pattern are specified.
func TestAccImagesDataSource_MutualExclusivity(t *testing.T) {
	cfg := `data "zillaforge_images" "test" {
  tag         = "latest"
  tag_pattern = "v1.*"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfg,
				ExpectError: regexp.MustCompile(`Cannot specify both 'tag' and 'tag_pattern'`),
			},
		},
	})
}

// T044: Acceptance test - Special characters in tag names.
// Expected: Handles tags with valid special characters (-, _, .).
func TestAccImagesDataSource_SpecialCharacters(t *testing.T) {
	tag := os.Getenv("TF_ACC_IMAGES_TAG_SPECIAL")
	if tag == "" {
		t.Skip("TF_ACC_IMAGES_TAG_SPECIAL must be set for this test (e.g., 'v1.0.0-rc.1' or 'prod_2024-12-15')")
	}

	cfg := fmt.Sprintf(`data "zillaforge_images" "test" {
  tag = "%s"
}
`, tag)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			provider.TestAccPreCheck(t)
			skipIfNoMatchingImages(t, "", tag, "")
		},
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zillaforge_images.test", "images.#"),
					resource.TestCheckFunc(func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.zillaforge_images.test"]
						if !ok {
							return fmt.Errorf("data source not found")
						}
						if rs.Primary.Attributes["images.0.tag_name"] != tag {
							return fmt.Errorf("images.0.tag_name expected %s, got %s", tag, rs.Primary.Attributes["images.0.tag_name"])
						}
						return nil
					}),
				),
			},
		},
	})
}
