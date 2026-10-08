package domain

import "time"

// Tenant is a company or team. Every other entity hangs off one.
type Tenant struct {
	ID        string
	Name      string
	CreatedAt time.Time
}
