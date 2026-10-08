package domain

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Document states.
const (
	DocumentActive   = "active"
	DocumentArchived = "archived"
)

const (
	maxTitleLen       = 200
	maxDescriptionLen = 2000
)

// Document is the container for logs and the unit of sharing and reporting.
type Document struct {
	ID          string
	TenantID    string
	OwnerID     string
	Title       string
	Description string
	State       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate trims the title and checks lengths and state. It mutates the receiver.
// Description is deliberately left untrimmed.
func (d *Document) Validate() error {
	d.Title = strings.TrimSpace(d.Title)
	fields := map[string]string{}
	switch n := utf8.RuneCountInString(d.Title); {
	case n == 0:
		fields["title"] = "required"
	case n > maxTitleLen:
		fields["title"] = "at most " + strconv.Itoa(maxTitleLen) + " characters"
	}
	if utf8.RuneCountInString(d.Description) > maxDescriptionLen {
		fields["description"] = "at most " + strconv.Itoa(maxDescriptionLen) + " characters"
	}
	if d.State != DocumentActive && d.State != DocumentArchived {
		fields["state"] = "must be active or archived"
	}
	if len(fields) > 0 {
		return NewValidationError(fields)
	}
	return nil
}
