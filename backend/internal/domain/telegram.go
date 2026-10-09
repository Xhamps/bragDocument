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
	msgTagRe   = regexp.MustCompile(`^#[\p{L}\p{N}_-]*[\p{L}\p{N}][\p{L}\p{N}_-]*$`)
	msgURLRe   = regexp.MustCompile(`https?://\S+`)
	msgTokenRe = regexp.MustCompile(`[ \t]*\S+`)
)

// markerPunct is trailing punctuation allowed after a #tag or !impact.
const markerPunct = ".,;:!?)"

// ParseLogMessage reads a one-shot chat message: first line is the name, the
// rest the description; #tag, !impact, and URLs anywhere become fields.
// Markers and URLs are removed from the name; only !impact from the description.
// Tags and impacts are whole whitespace-separated tokens (trailing punctuation
// allowed), so "# Heading" and URL fragments are not tags.
func ParseLogMessage(text string) (LogDraft, error) {
	text = strings.TrimLeft(strings.ReplaceAll(text, "\r\n", "\n"), " \t\r\n")
	d := LogDraft{Impact: "medium", Tags: []string{}, Links: []Link{}}
	for _, u := range msgURLRe.FindAllString(text, -1) {
		u = trimURL(u)
		if !slices.ContainsFunc(d.Links, func(l Link) bool { return l.URL == u }) {
			d.Links = append(d.Links, Link{URL: u})
		}
	}
	for _, tok := range strings.Fields(msgURLRe.ReplaceAllString(text, " ")) {
		if tag, ok := tagToken(tok); ok && !slices.Contains(d.Tags, tag) {
			d.Tags = append(d.Tags, tag)
		}
		if impact, ok := impactToken(tok); ok {
			d.Impact = impact // last one wins
		}
	}

	first, rest, _ := strings.Cut(text, "\n")
	var name []string
	for _, tok := range strings.Fields(msgURLRe.ReplaceAllString(first, " ")) {
		_, isTag := tagToken(tok)
		_, isImpact := impactToken(tok)
		if !isTag && !isImpact {
			name = append(name, tok)
		}
	}
	d.Name = strings.Join(name, " ")
	// Drop each !impact token with its leading space, keeping its punctuation: "it !high." → "it.".
	d.Description = strings.TrimSpace(msgTokenRe.ReplaceAllStringFunc(rest, func(m string) string {
		tok := strings.TrimLeft(m, " \t")
		if _, ok := impactToken(tok); ok {
			return tok[len(strings.TrimRight(tok, markerPunct)):]
		}
		return m
	}))
	if d.Name == "" {
		return d, NewValidationError(map[string]string{"name": "first line needs some text besides tags and links"})
	}
	return d, nil
}

func tagToken(tok string) (string, bool) {
	tok = strings.TrimRight(tok, markerPunct)
	if !msgTagRe.MatchString(tok) {
		return "", false
	}
	return strings.ToLower(tok[1:]), true
}

func impactToken(tok string) (string, bool) {
	tok = strings.ToLower(strings.TrimRight(tok, markerPunct))
	if !strings.HasPrefix(tok, "!") || !slices.Contains(Impacts, tok[1:]) {
		return "", false
	}
	return tok[1:], true
}

// trimURL drops trailing punctuation, keeping a ")" that closes a "(" in the URL.
func trimURL(u string) string {
	for {
		n := strings.TrimRight(u, ".,;:!?>\"")
		if strings.HasSuffix(n, ")") && strings.Count(n, ")") > strings.Count(n, "(") {
			n = n[:len(n)-1]
		}
		if n == u {
			return u
		}
		u = n
	}
}
