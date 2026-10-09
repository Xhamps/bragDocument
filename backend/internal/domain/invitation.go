package domain

import (
	"net/mail"
	"strings"
	"time"
)

// Invitation lets an email address join a tenant on first sign-in.
type Invitation struct {
	ID        string
	TenantID  string
	Email     string
	CreatedBy string
	CreatedAt time.Time
	// ForDocument marks a pending document invitation (PRD-0004 FR-4) found at
	// sign-in or listed for admins; DocumentTitle is set in the admin list.
	ForDocument   bool
	DocumentTitle string
}

// NormalizeEmail trims, lowercases, and validates an email address.
// Accepts only a bare local@domain address; display names, comments, and quoted local parts are rejected.
func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return "", NewValidationError(map[string]string{"email": "invalid email address"})
	}
	return email, nil
}
