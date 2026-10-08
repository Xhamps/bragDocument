package domain

import "time"

// ExampleLogs are the logs a new document starts with (PRD-0002 §9): written
// in the article's spirit, flagged IsExample, removable in one click. Their
// statements are fixed, so they never go to the extractor (PRD-0007 FR-7).
func ExampleLogs(now time.Time, userID string) []Log {
	ex := func(daysAgo int, name, desc, impact, statement string, tags []string, links []Link) Log {
		return Log{Name: name, Description: desc, Impact: impact, ImpactStatement: &statement,
			Status: StatusDone, IsExample: true, Tags: tags, Links: links,
			CreatedAt: now.AddDate(0, 0, -daysAgo), CreatedBy: userID, UpdatedBy: userID}
	}
	return []Log{
		ex(45, "Moved billing jobs off the legacy cron host",
			"Moved 14 nightly billing jobs to the queue workers and retired the old host.\n\n"+
				"**Result:** failed billing runs went from about 3 a week to zero, and on-call stopped getting paged for billing.",
			"high", "Failed billing runs went from about 3 a week to zero; on-call stopped getting paged for billing.",
			[]string{"project"},
			[]Link{{URL: "https://github.com/example/billing/pull/123", Label: "PR"}}),
		ex(20, "Mentored a new teammate through their first on-call",
			"Paired weekly for a month and wrote a runbook together for the three noisiest alerts.\n\n"+
				"**Result:** they ran their first on-call week alone, and the whole team now uses the runbook.",
			"medium", "They ran their first on-call week alone; the whole team now uses the runbook.",
			[]string{"mentorship", "documentation"},
			[]Link{{URL: "https://docs.example.com/runbooks/alerts", Label: "Runbook"}}),
		ex(5, "Started the team's incident review practice",
			"Set up a fortnightly blameless review and kept the notes in one place.\n\n"+
				"**Result:** six follow-up fixes shipped this quarter came straight out of the reviews.",
			"medium", "Six follow-up fixes shipped this quarter came straight out of the reviews.",
			[]string{"company-building", "collaboration"}, nil),
	}
}
