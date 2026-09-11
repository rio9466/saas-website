// Package analytics holds the page-view entity, traffic-source resolution, and
// the administrator permission codes for the analytics surface.
package analytics

// PermissionRead gates the administrator analytics overview (seeded by
// migration 000011 and granted to super_admin).
const PermissionRead = "admin.analytics.read"
