// Package email holds the ports.Mailer adapters (PRD-0004 FR-6, ADR-0014).
package email

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/resend/resend-go/v2"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Disabled is the Mailer without RESEND_API_KEY.
type Disabled struct{}

// Send always reports the mailer as unavailable.
func (Disabled) Send(context.Context, string, string, string) error { return domain.ErrUnavailable }

// Resend sends through the Resend API.
type Resend struct {
	client   *resend.Client
	from     string
	timeout  time.Duration
	failures prometheus.Counter
}

// NewResend registers email_failures_total on reg, so call it once per registry.
func NewResend(apiKey, from string, timeout time.Duration, reg prometheus.Registerer) *Resend {
	failures := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "email_failures_total",
		Help: "Emails that could not be sent; the action that triggered them still succeeded.",
	})
	reg.MustRegister(failures)
	return &Resend{client: resend.NewClient(apiKey), from: from, timeout: timeout, failures: failures}
}

// Send delivers one HTML email within the configured timeout.
func (r *Resend) Send(ctx context.Context, to, subject, html string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	_, err := r.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From: r.from, To: []string{to}, Subject: subject, Html: html,
	})
	if err != nil {
		r.failures.Inc()
		return fmt.Errorf("resend: %w", err)
	}
	return nil
}
