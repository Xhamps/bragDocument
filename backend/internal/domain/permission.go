package domain

import (
	"fmt"
	"slices"
)

// Role is a user's relation to one document (PRD-0004 FR-1). The owner is
// documents.owner_id; editor and viewer are grants.
type Role string

// Document roles.
const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// Grantable reports whether the role can be granted; ownership is only transferred.
func (r Role) Grantable() bool { return r == RoleEditor || r == RoleViewer }

// Permission is one action of the PRD-0004 matrix. The value is the phrase
// AccessError uses, so a denial reads as a sentence.
type Permission string

// Permissions. Generating the PDF report (PRD-0006) is PermRead.
const (
	PermRead      Permission = "read the document"
	PermWriteLogs Permission = "create, edit, or delete logs"
	PermManage    Permission = "rename or archive the document"
	PermDelete    Permission = "delete the document"
	PermShare     Permission = "change sharing"
	PermTransfer  Permission = "transfer ownership"
)

// permissions mirrors the matrix in docs/prd/0004-sharing-and-rbac.md §7.
var permissions = map[Role][]Permission{
	RoleOwner:  {PermRead, PermWriteLogs, PermManage, PermDelete, PermShare, PermTransfer},
	RoleEditor: {PermRead, PermWriteLogs},
	RoleViewer: {PermRead},
}

// Can reports whether role r allows p. The empty role allows nothing.
func Can(r Role, p Permission) bool { return slices.Contains(permissions[r], p) }

// AccessError explains a denial (PRD-0004 Goal 3). It matches ErrForbidden.
type AccessError struct {
	Role Role
	Perm Permission
}

func (e *AccessError) Error() string {
	return fmt.Sprintf("you are %s on this document and cannot %s", e.Role, e.Perm)
}

// Is makes errors.Is(err, ErrForbidden) true.
func (e *AccessError) Is(target error) bool { return target == ErrForbidden }
