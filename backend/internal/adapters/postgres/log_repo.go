package postgres

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// LogRepo implements ports.LogRepo for the tenant in the context.
type LogRepo struct{ db *DB }

// NewLogRepo wires the repository to the pool.
func NewLogRepo(db *DB) *LogRepo { return &LogRepo{db: db} }

func (r *LogRepo) tx(ctx context.Context, fn func(ctx context.Context, q *sqlcgen.Queries) error) error {
	return withQueries(ctx, r.db, fn)
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)

func listParams(did uuid.UUID, f domain.LogFilter) sqlcgen.ListLogsParams {
	p := sqlcgen.ListLogsParams{
		DocumentID:   did,
		Statuses:     orEmpty(f.Statuses),
		Impacts:      orEmpty(f.Impacts),
		Tags:         orEmpty(f.Tags),
		HideExamples: f.HideExamples,
		Sort:         f.Sort,
		Descending:   f.Desc,
		Lim:          int32(f.PerPage),
		Off:          int32((f.Page - 1) * f.PerPage),
	}
	if f.Query != "" {
		p.Q = pgtype.Text{String: likeEscaper.Replace(f.Query), Valid: true}
	}
	if f.Domain != "" {
		p.Host = pgtype.Text{String: f.Domain, Valid: true}
	}
	if f.From != nil {
		p.FromAt = pgtype.Timestamptz{Time: *f.From, Valid: true}
	}
	if f.To != nil {
		p.ToAt = pgtype.Timestamptz{Time: *f.To, Valid: true}
	}
	return p
}

// List returns one page of the document's logs. f must have passed
// domain.LogFilter.Validate (defaults, clamps, whitelisted sort).
func (r *LogRepo) List(ctx context.Context, documentID string, f domain.LogFilter) (domain.LogPage, error) {
	did, err := parseID(documentID)
	if err != nil {
		return domain.LogPage{}, err
	}
	page := domain.LogPage{Items: []domain.Log{}}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		p := listParams(did, f)
		rows, err := q.ListLogs(ctx, p)
		if err != nil {
			return wrap(err)
		}
		if len(rows) == 0 && p.Off > 0 {
			// Past the last page: count(*) OVER () has no row to ride on, so
			// ask for the first row only to learn the total.
			p.Off, p.Lim = 0, 1
			if rows, err = q.ListLogs(ctx, p); err != nil {
				return wrap(err)
			}
			if len(rows) > 0 {
				page.Total = int(rows[0].Total)
			}
			return nil
		}
		logs := make([]sqlcgen.Log, len(rows))
		for i, row := range rows {
			logs[i] = row.Log
		}
		if len(rows) > 0 {
			page.Total = int(rows[0].Total)
		}
		page.Items, err = hydrate(ctx, q, logs)
		return err
	})
	return page, err
}

// hydrate attaches tags and links with two queries per page, not per row.
func hydrate(ctx context.Context, q *sqlcgen.Queries, rows []sqlcgen.Log) ([]domain.Log, error) {
	out := make([]domain.Log, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, len(rows))
	byID := make(map[uuid.UUID]int, len(rows))
	for i, l := range rows {
		ids[i], byID[l.ID], out[i] = l.ID, i, toLog(l)
	}
	tags, err := q.ListTagsForLogs(ctx, ids)
	if err != nil {
		return nil, wrap(err)
	}
	for _, t := range tags {
		i := byID[t.LogID]
		out[i].Tags = append(out[i].Tags, t.TagName)
	}
	links, err := q.ListLinksForLogs(ctx, ids)
	if err != nil {
		return nil, wrap(err)
	}
	for _, k := range links {
		i := byID[k.LogID]
		out[i].Links = append(out[i].Links, domain.Link{URL: k.Url, Label: k.Label, Host: k.Host})
	}
	return out, nil
}

func (r *LogRepo) Get(ctx context.Context, documentID, id string) (domain.Log, error) {
	did, err := parseID(documentID)
	if err != nil {
		return domain.Log{}, err
	}
	lid, err := parseID(id)
	if err != nil {
		return domain.Log{}, err
	}
	var out domain.Log
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.GetLog(ctx, sqlcgen.GetLogParams{ID: lid, DocumentID: did})
		if err != nil {
			return wrap(err)
		}
		logs, err := hydrate(ctx, q, []sqlcgen.Log{row})
		if err != nil {
			return err
		}
		out = logs[0]
		return nil
	})
	return out, err
}

func (r *LogRepo) Create(ctx context.Context, l domain.Log) (domain.Log, error) {
	var out domain.Log
	err := r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		var err error
		out, err = insertLog(ctx, q, l)
		return err
	})
	return out, err
}

// insertLog is shared with DocumentRepo.Create, which seeds example logs.
func insertLog(ctx context.Context, q *sqlcgen.Queries, l domain.Log) (domain.Log, error) {
	tid, err := parseID(l.TenantID)
	if err != nil {
		return domain.Log{}, err
	}
	did, err := parseID(l.DocumentID)
	if err != nil {
		return domain.Log{}, err
	}
	uid, err := parseID(l.CreatedBy)
	if err != nil {
		return domain.Log{}, err
	}
	row, err := q.CreateLog(ctx, sqlcgen.CreateLogParams{
		TenantID: tid, DocumentID: did, Name: l.Name, Description: l.Description, Impact: l.Impact,
		ImpactStatement: textOrNull(l.ImpactStatement), Status: l.Status, IsExample: l.IsExample,
		CreatedAt: l.CreatedAt, CreatedBy: uid,
	})
	if err != nil {
		return domain.Log{}, wrap(err)
	}
	if err := setTagsAndLinks(ctx, q, row.TenantID, row.ID, l.Tags, l.Links); err != nil {
		return domain.Log{}, err
	}
	out := toLog(row)
	out.Tags, out.Links = slices.Sorted(slices.Values(orEmpty(l.Tags))), orEmpty(l.Links) // matches Get's ORDER BY tag_name
	return out, nil
}

// setTagsAndLinks replaces a log's tags and links. Tags join the tenant
// vocabulary first so the log_tags foreign key holds.
func setTagsAndLinks(ctx context.Context, q *sqlcgen.Queries, tid, lid uuid.UUID, tags []string, links []domain.Link) error {
	if err := q.DeleteLogTags(ctx, lid); err != nil {
		return wrap(err)
	}
	if err := q.DeleteLogLinks(ctx, lid); err != nil {
		return wrap(err)
	}
	if len(tags) > 0 {
		if err := q.UpsertTags(ctx, sqlcgen.UpsertTagsParams{TenantID: tid, Names: tags}); err != nil {
			return wrap(err)
		}
		if err := q.InsertLogTags(ctx, sqlcgen.InsertLogTagsParams{TenantID: tid, LogID: lid, Names: tags}); err != nil {
			return wrap(err)
		}
	}
	for i, k := range links {
		if err := q.InsertLogLink(ctx, sqlcgen.InsertLogLinkParams{
			TenantID: tid, LogID: lid, Url: k.URL, Label: k.Label, Host: k.Host, Position: int32(i),
		}); err != nil {
			return wrap(err)
		}
	}
	return nil
}

func (r *LogRepo) Update(ctx context.Context, l domain.Log) (domain.Log, error) {
	lid, err := parseID(l.ID)
	if err != nil {
		return domain.Log{}, err
	}
	did, err := parseID(l.DocumentID)
	if err != nil {
		return domain.Log{}, err
	}
	uid, err := parseID(l.UpdatedBy)
	if err != nil {
		return domain.Log{}, err
	}
	var out domain.Log
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.UpdateLog(ctx, sqlcgen.UpdateLogParams{
			ID: lid, DocumentID: did, Name: l.Name, Description: l.Description, Impact: l.Impact,
			ImpactStatement: textOrNull(l.ImpactStatement), Status: l.Status, IsExample: l.IsExample,
			CreatedAt: l.CreatedAt, UpdatedBy: uid,
		})
		if err != nil {
			return wrap(err)
		}
		if err := setTagsAndLinks(ctx, q, row.TenantID, row.ID, l.Tags, l.Links); err != nil {
			return err
		}
		out = toLog(row)
		out.Tags, out.Links = slices.Sorted(slices.Values(orEmpty(l.Tags))), orEmpty(l.Links) // matches Get's ORDER BY tag_name
		return nil
	})
	return out, err
}

func (r *LogRepo) Delete(ctx context.Context, documentID, id string) error {
	did, err := parseID(documentID)
	if err != nil {
		return err
	}
	lid, err := parseID(id)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		n, err := q.DeleteLog(ctx, sqlcgen.DeleteLogParams{ID: lid, DocumentID: did})
		if err != nil {
			return wrap(err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *LogRepo) DeleteExamples(ctx context.Context, documentID string) error {
	did, err := parseID(documentID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		return wrap(q.DeleteExampleLogs(ctx, did))
	})
}

func (r *LogRepo) ListTags(ctx context.Context) ([]string, error) {
	var out []string
	err := r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		var err error
		out, err = q.ListTags(ctx)
		return wrap(err)
	})
	return orEmpty(out), err
}

// Dashboard returns the sparse aggregates for p in one transaction.
func (r *LogRepo) Dashboard(ctx context.Context, documentID string, p domain.Period) (domain.Dashboard, error) {
	did, err := parseID(documentID)
	if err != nil {
		return domain.Dashboard{}, err
	}
	d := domain.Dashboard{From: p.From, To: p.To}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		t, err := q.DashboardTotals(ctx, sqlcgen.DashboardTotalsParams{DocumentID: did, FromAt: p.From, ToAt: p.To})
		if err != nil {
			return wrap(err)
		}
		d.Total, d.InPeriod, d.HighImpact, d.InProgress = int(t.Total), int(t.InPeriod), int(t.HighImpact), int(t.InProgress)
		rows, err := q.DashboardBuckets(ctx, sqlcgen.DashboardBucketsParams{DocumentID: did, FromAt: p.From, ToAt: p.To})
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			b := domain.Bucket{Key: row.Key, Count: int(row.Count)}
			switch row.Kind {
			case "month":
				d.Months = append(d.Months, b)
			case "status":
				d.Statuses = append(d.Statuses, b)
			case "impact":
				d.Impacts = append(d.Impacts, b)
			case "tag":
				d.Tags = append(d.Tags, b)
			}
		}
		return nil
	})
	return d, err
}
