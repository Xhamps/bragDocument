package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Dashboard aggregates the document's logs for a period (PRD-0005). Access is
// checked before the cache so a revoked reader is refused at once. Cache
// errors are misses: the numbers then come from Postgres.
func (s *Logs) Dashboard(ctx context.Context, docID, userID string, from, to *time.Time) (domain.Dashboard, error) {
	if _, err := access(ctx, s.docs, docID, userID, domain.PermRead); err != nil {
		return domain.Dashboard{}, err
	}
	p, err := domain.NewPeriod(from, to, s.now())
	if err != nil {
		return domain.Dashboard{}, err
	}
	ver := "0"
	if v, ok, _ := s.cache.Get(ctx, docVersionKey(docID)); ok {
		ver = string(v)
	}
	key := fmt.Sprintf("dash:%s:v%s:%d:%d", docID, ver, p.From.Unix(), p.To.Unix())
	if b, ok, _ := s.cache.Get(ctx, key); ok {
		var d domain.Dashboard
		if json.Unmarshal(b, &d) == nil {
			return d, nil
		}
	}
	d, err := s.logs.Dashboard(ctx, docID, p)
	if err != nil {
		return domain.Dashboard{}, err
	}
	d.Normalize()
	if b, err := json.Marshal(d); err == nil {
		_ = s.cache.Set(ctx, key, b, dashboardTTL)
	}
	return d, nil
}
