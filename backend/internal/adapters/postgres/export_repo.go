package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ExportRepo implements ports.ExportRepo.
type ExportRepo struct{ db *DB }

// NewExportRepo wires the repository to the pool.
func NewExportRepo(db *DB) *ExportRepo { return &ExportRepo{db: db} }

func toExportJob(j sqlcgen.ExportJob) (domain.ExportJob, error) {
	out := domain.ExportJob{ID: j.ID.String(), TenantID: j.TenantID.String(), DocumentID: j.DocumentID.String(),
		RequestedBy: j.RequestedBy.String(), Status: j.Status, Error: j.Error, FileKey: j.FileKey,
		CreatedAt: j.CreatedAt, ExpiresAt: j.ExpiresAt}
	if err := json.Unmarshal(j.Params, &out.Params); err != nil {
		return domain.ExportJob{}, fmt.Errorf("postgres: export params: %w", err)
	}
	return out, nil
}

func (r *ExportRepo) Create(ctx context.Context, j domain.ExportJob, a domain.AuditEntry) (domain.ExportJob, error) {
	tid, err := parseID(j.TenantID)
	if err != nil {
		return domain.ExportJob{}, err
	}
	did, err := parseID(j.DocumentID)
	if err != nil {
		return domain.ExportJob{}, err
	}
	uid, err := parseID(j.RequestedBy)
	if err != nil {
		return domain.ExportJob{}, err
	}
	params, err := json.Marshal(j.Params)
	if err != nil {
		return domain.ExportJob{}, err
	}
	var out domain.ExportJob
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.CreateExportJob(ctx, sqlcgen.CreateExportJobParams{TenantID: tid, DocumentID: did, RequestedBy: uid, Params: params})
		if err != nil {
			return wrap(err)
		}
		if out, err = toExportJob(row); err != nil {
			return err
		}
		return audit(ctx, q, a)
	})
	return out, err
}

func (r *ExportRepo) Get(ctx context.Context, documentID, id string) (domain.ExportJob, error) {
	did, err := parseID(documentID)
	if err != nil {
		return domain.ExportJob{}, err
	}
	jid, err := parseID(id)
	if err != nil {
		return domain.ExportJob{}, err
	}
	var out domain.ExportJob
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.GetExportJob(ctx, sqlcgen.GetExportJobParams{ID: jid, DocumentID: did})
		if err != nil {
			return wrap(err)
		}
		out, err = toExportJob(row)
		return err
	})
	return out, err
}

func (r *ExportRepo) List(ctx context.Context, documentID, userID string) ([]domain.ExportJob, error) {
	did, err := parseID(documentID)
	if err != nil {
		return nil, err
	}
	uid, err := parseID(userID)
	if err != nil {
		return nil, err
	}
	out := []domain.ExportJob{}
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListExportJobs(ctx, sqlcgen.ListExportJobsParams{DocumentID: did, RequestedBy: uid})
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			j, err := toExportJob(row)
			if err != nil {
				return err
			}
			out = append(out, j)
		}
		return nil
	})
	return out, err
}

// Claim runs under the provisioning flag: the tenant is what we are about to learn.
func (r *ExportRepo) Claim(ctx context.Context) (domain.ExportJob, error) {
	var out domain.ExportJob
	err := r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		row, err := sqlcgen.New(tx).ClaimExportJob(ctx)
		if err != nil {
			return wrap(err) // no row: domain.ErrNotFound
		}
		out, err = toExportJob(row)
		return err
	})
	return out, err
}

func (r *ExportRepo) Finish(ctx context.Context, id, fileKey string) error {
	jid, err := parseID(id)
	if err != nil {
		return err
	}
	return withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		return oneRow(q.FinishExportJob(ctx, sqlcgen.FinishExportJobParams{ID: jid, FileKey: fileKey}))
	})
}

func (r *ExportRepo) Fail(ctx context.Context, id, reason string) error {
	jid, err := parseID(id)
	if err != nil {
		return err
	}
	return withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		return oneRow(q.FailExportJob(ctx, sqlcgen.FailExportJobParams{ID: jid, Error: reason}))
	})
}

// Expired runs under the provisioning flag: cleanup spans tenants.
func (r *ExportRepo) Expired(ctx context.Context) ([]domain.ExportJob, error) {
	out := []domain.ExportJob{}
	err := r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := sqlcgen.New(tx).ListExpiredExportJobs(ctx)
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			j, err := toExportJob(row)
			if err != nil {
				return err
			}
			out = append(out, j)
		}
		return nil
	})
	return out, err
}

// Delete runs under the provisioning flag, after Expired.
func (r *ExportRepo) Delete(ctx context.Context, id string) error {
	jid, err := parseID(id)
	if err != nil {
		return err
	}
	return r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return wrap(sqlcgen.New(tx).DeleteExportJob(ctx, jid))
	})
}

func (r *ExportRepo) Settings(ctx context.Context, documentID string) (domain.ReportSettings, error) {
	did, err := parseID(documentID)
	if err != nil {
		return domain.ReportSettings{}, err
	}
	var out domain.ReportSettings
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.GetReportSettings(ctx, did)
		if err != nil {
			return wrap(err)
		}
		out = domain.ReportSettings{GoalsThisYear: row.GoalsThisYear, GoalsNextYear: row.GoalsNextYear}
		if err := json.Unmarshal(row.SectionMap, &out.SectionMap); err != nil {
			return fmt.Errorf("postgres: section map: %w", err)
		}
		return nil
	})
	return out, err
}

func (r *ExportRepo) SaveSettings(ctx context.Context, tenantID, documentID string, s domain.ReportSettings) error {
	tid, err := parseID(tenantID)
	if err != nil {
		return err
	}
	did, err := parseID(documentID)
	if err != nil {
		return err
	}
	m, err := json.Marshal(orEmptyMap(s.SectionMap))
	if err != nil {
		return err
	}
	return withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		return wrap(q.UpsertReportSettings(ctx, sqlcgen.UpsertReportSettingsParams{DocumentID: did, TenantID: tid,
			GoalsThisYear: s.GoalsThisYear, GoalsNextYear: s.GoalsNextYear, SectionMap: m}))
	})
}

// oneRow maps "no running job changed" to domain.ErrNotFound: the job was
// reclaimed or already settled, and a late outcome must not overwrite it.
func oneRow(n int64, err error) error {
	if err != nil {
		return wrap(err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// orEmptyMap keeps jsonb a JSON object: nil marshals to null.
func orEmptyMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
