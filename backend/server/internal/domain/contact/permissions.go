// Package contact holds the contact-inbox entity, submission validation, and
// the administrator permission codes for the inbox surface.
package contact

// Permission codes for contact-inbox administration (seeded by migration
// 000009 alongside the content permissions).
const (
	PermissionRead   = "admin.contact.read"
	PermissionManage = "admin.contact.manage"
)
