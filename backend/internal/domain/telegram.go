package domain

import (
	"regexp"
	"slices"
	"strings"
	"time"
)

// TelegramLink ties a user to a Telegram account (PRD-0003). DocumentID is ""
// until the user picks a target with /use.
type TelegramLink struct {
	UserID         string
	TenantID       string
	TelegramUserID int64
	DocumentID     string
	LinkedAt       time.Time
}

// LogDraft is a log parsed from a chat message. Validation is left to Log.Validate.
type LogDraft struct {
	Name        string
	Description string
	Impact      string
	Tags        []string
	Links       []Link
}

var (
	msgTagRe    = regexp.MustCompile(`^#[\p{L}\p{N}_-]+$`)
	msgImpactRe = regexp.MustCompile(`(?i)(^|\s)!(low|medium|high|critical)\b`)
	msgURLRe    = regexp.MustCompile(`https?://\S+`)
)

// ParseLogMessage reads a one-shot chat message: first line is the name, the
// rest the description; #tag, !impact, and URLs anywhere become fields.
// Markers and URLs are removed from the name; only !impact from the description.
// Tags and impacts are whole whitespace-separated tokens, so "# Heading" and
// URL fragments are not tags.
func ParseLogMessage(text string) (LogDraft, error) {
	d := LogDraft{Impact: "medium", Tags: []string{}, Links: []Link{}}
	for _, u := range msgURLRe.FindAllString(text, -1) {
		u = strings.TrimRight(u, ".,;:!?)")
		if !slices.ContainsFunc(d.Links, func(l Link) bool { return l.URL == u }) {
			d.Links = append(d.Links, Link{URL: u})
		}
	}
	for _, tok := range strings.Fields(msgURLRe.ReplaceAllString(text, " ")) {
		switch {
		case msgTagRe.MatchString(tok):
			if tag := tok[1:]; !slices.Contains(d.Tags, tag) {
				d.Tags = append(d.Tags, tag)
			}
		case isImpactToken(tok):
			d.Impact = strings.ToLower(tok[1:]) // last one wins
		}
	}

	first, rest, _ := strings.Cut(text, "\n")
	var name []string
	for _, tok := range strings.Fields(msgURLRe.ReplaceAllString(first, " ")) {
		if !msgTagRe.MatchString(tok) && !isImpactToken(tok) {
			name = append(name, tok)
		}
	}
	d.Name = strings.Join(name, " ")
	d.Description = strings.TrimSpace(stripImpacts(rest))
	if d.Name == "" {
		return d, NewValidationError(map[string]string{"name": "first line needs some text besides tags and links"})
	}
	return d, nil
}

func isImpactToken(tok string) bool {
	return strings.HasPrefix(tok, "!") && slices.Contains(Impacts, strings.ToLower(tok[1:]))
}

// stripImpacts removes !impact markers, keeping the rest verbatim. It loops
// because adjacent markers share the separating space, so one pass skips every other.
func stripImpacts(s string) string {
	for {
		n := msgImpactRe.ReplaceAllString(s, "$1")
		if n == s {
			return s
		}
		s = n
	}
}
