package domain

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Impacts are the impact levels, lowest first. The order is the sort order.
var Impacts = []string{"low", "medium", "high", "critical"}

// Statuses are the log statuses in lifecycle order. The order is the sort order.
var Statuses = []string{"idea", "in_progress", "done", "dropped"}

// StatusDone is the status of a new log when none is given.
const StatusDone = "done"

const (
	maxLogNameLen        = 120
	maxLogDescriptionLen = 20000
	maxTagLen            = 50
	maxTags              = 20
	maxLinks             = 20
	maxLinkLabelLen      = 100
)

// Link is a piece of evidence. Validate derives Host, which backs the domain filter.
type Link struct {
	URL   string
	Label string
	Host  string
}

// Log is one entry in a brag document (PRD-0002).
type Log struct {
	ID          string
	TenantID    string
	DocumentID  string
	Name        string
	Description string
	Impact      string
	// ImpactStatement is extracted from the text (PRD-0007): nil means not
	// checked, "" means checked and none found.
	ImpactStatement *string
	Status          string
	IsExample       bool
	Tags            []string
	Links           []Link
	CreatedAt       time.Time
	CreatedBy       string
	UpdatedAt       time.Time
	UpdatedBy       string
}

// Validate trims the name, normalizes tags and links, and checks every field.
// It mutates the receiver. Description is deliberately left untrimmed.
func (l *Log) Validate() error {
	l.Name = strings.TrimSpace(l.Name)
	fields := map[string]string{}
	switch n := utf8.RuneCountInString(l.Name); {
	case n == 0:
		fields["name"] = "required"
	case n > maxLogNameLen:
		fields["name"] = "at most " + strconv.Itoa(maxLogNameLen) + " characters"
	}
	if utf8.RuneCountInString(l.Description) > maxLogDescriptionLen {
		fields["description"] = "at most " + strconv.Itoa(maxLogDescriptionLen) + " characters"
	}
	if !slices.Contains(Impacts, l.Impact) {
		fields["impact"] = "must be one of " + strings.Join(Impacts, ", ")
	}
	if !slices.Contains(Statuses, l.Status) {
		fields["status"] = "must be one of " + strings.Join(Statuses, ", ")
	}
	var msg string
	if l.Tags, msg = normalizeTags(l.Tags); msg != "" {
		fields["tags"] = msg
	}
	if msg := l.normalizeLinks(); msg != "" {
		fields["links"] = msg
	}
	if len(fields) > 0 {
		return NewValidationError(fields)
	}
	return nil
}

// normalizeTags lowercases, trims, drops empties, and de-duplicates, keeping
// order. The message is non-empty when a limit is exceeded.
func normalizeTags(in []string) ([]string, string) {
	out := []string{}
	for _, t := range in {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || slices.Contains(out, t) {
			continue
		}
		if utf8.RuneCountInString(t) > maxTagLen {
			return out, "each tag at most " + strconv.Itoa(maxTagLen) + " characters"
		}
		out = append(out, t)
	}
	if len(out) > maxTags {
		return out, "at most " + strconv.Itoa(maxTags) + " tags"
	}
	return out, ""
}

func (l *Log) normalizeLinks() string {
	if len(l.Links) > maxLinks {
		return "at most " + strconv.Itoa(maxLinks) + " links"
	}
	for i := range l.Links {
		k := &l.Links[i]
		k.URL = strings.TrimSpace(k.URL)
		k.Label = strings.TrimSpace(k.Label)
		u, err := url.Parse(k.URL)
		// Scheme allow-list: a javascript: URL rendered as a link is XSS.
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
			return "each link must be an http or https URL"
		}
		if utf8.RuneCountInString(k.Label) > maxLinkLabelLen {
			return "each label at most " + strconv.Itoa(maxLinkLabelLen) + " characters"
		}
		k.Host = NormalizeHost(u.Hostname())
	}
	return ""
}

// NormalizeHost lowercases a host name and strips a leading "www.".
func NormalizeHost(h string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(h)), "www.")
}
