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
}

// NormalizeEmail trims, lowercases, and validates an email address.
func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return "", NewValidationError(map[string]string{"email": "invalid email address"})
	}
	return email, nil
}
