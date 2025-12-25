## 0.1.0 (Unreleased)

FEATURES:

- Add `zillaforge_images` data source to query VRM image tags (repository:tag). Supports filters: `repository`, `tag`, and `tag_pattern`. Includes acceptance tests and generated documentation.- Add `zillaforge_floating_ip` resource to manage floating IP addresses. Supports allocation, update (name/description), deletion, and import operations. Floating IPs are public IPv4 addresses from a shared pool.
- Add `zillaforge_floating_ips` data source to query existing floating IPs. Supports client-side filtering by `id`, `name`, `ip_address`, and `status` with AND logic. Returns a list of matching floating IPs sorted by ID.