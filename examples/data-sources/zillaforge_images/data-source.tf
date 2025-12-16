# Query specific image by repository and tag
data "zillaforge_images" "ubuntu_2204" {
  repository = "ubuntu"
  tag        = "22.04"
}

# Access the unique image
output "ubuntu_image_id" {
  value = data.zillaforge_images.ubuntu_2204.images[0].id
}

# List all tags for a repository
data "zillaforge_images" "ubuntu_all" {
  repository = "ubuntu"
}

# Find latest (first in deterministically sorted list)
output "ubuntu_images" {
  value = data.zillaforge_images.ubuntu_all.images
}

# Pattern matching for versioned tags
data "zillaforge_images" "v1_series" {
  repository  = "myapp"
  tag_pattern = "v1.*"
}

output "v1_images" {
  value = [
    for img in data.zillaforge_images.v1_series.images : {
      repo = img.repository_name
      tag  = img.tag_name
      id   = img.id
      size = img.size
    }
  ]
}

# Cross-repository search by tag name
data "zillaforge_images" "latest" {
  tag = "latest"
}

output "latest_images" {
  value = data.zillaforge_images.latest.images
}

# Filter by environment prefix
data "zillaforge_images" "production" {
  tag_pattern = "prod-*"
}

output "production_images" {
  value = data.zillaforge_images.production.images
}

# List all images (up to server limit of 1000)
data "zillaforge_images" "all" {
  # No filters specified
}

output "total_images" {
  value = length(data.zillaforge_images.all.images)
}

# Use image ID in VM resource
resource "zillaforge_vm" "web_server" {
  # Reference image ID from data source
  image_id = data.zillaforge_images.ubuntu_2204.images[0].id
  # ... other VM configuration ...
}
