package app

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

type fakeUsers struct {
	users               map[string]domain.User
	tenants             map[string]domain.Tenant
	invitations         map[string]domain.Invitation // by email
	seq                 int
	createUserErrOnce   error // returned by the first CreateUser, then cleared; a conflict inserts the winner's row
	createUserErrAlways error // returned by every CreateUser, no row inserted
	createUserCalls     int
	accepted            []string // user ids passed to AcceptInvitations
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{users: map[string]domain.User{}, tenants: map[string]domain.Tenant{}, invitations: map[string]domain.Invitation{}}
}

func (f *fakeUsers) Provision(ctx context.Context, fn func(context.Context, ports.ProvisionTx) error) error {
	return fn(ctx, f)
}
func (f *fakeUsers) GetUser(_ context.Context, id string) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}
func (f *fakeUsers) GetTenant(_ context.Context, id string) (domain.Tenant, error) {
	t, ok := f.tenants[id]
	if !ok {
		return domain.Tenant{}, domain.ErrNotFound
	}
	return t, nil
}
func (f *fakeUsers) FindInvitationByEmail(_ context.Context, email string) (domain.Invitation, error) {
	i, ok := f.invitations[email]
	if !ok {
		return domain.Invitation{}, domain.ErrNotFound
	}
	return i, nil
}
func (f *fakeUsers) CreateTenant(_ context.Context, name string) (domain.Tenant, error) {
	f.seq++
	t := domain.Tenant{ID: "t" + strconv.Itoa(f.seq), Name: name}
	f.tenants[t.ID] = t
	return t, nil
}
func (f *fakeUsers) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	f.createUserCalls++
	if f.createUserErrAlways != nil {
		return domain.User{}, f.createUserErrAlways
	}
	if err := f.createUserErrOnce; err != nil {
		f.createUserErrOnce = nil
		if errors.Is(err, domain.ErrConflict) {
			// The concurrent winner's row: in the invited tenant if there is an invitation, else t9.
			tenantID := "t9"
			if inv, ok := f.invitations[u.Email]; ok {
				tenantID = inv.TenantID
				delete(f.invitations, u.Email) // the winner consumed it
			}
			f.users[u.ID] = domain.User{ID: u.ID, TenantID: tenantID, Email: u.Email}
		}
		return domain.User{}, err
	}
	f.users[u.ID] = u
	return u, nil
}
func (f *fakeUsers) AcceptInvitations(_ context.Context, u domain.User) error {
	delete(f.invitations, u.Email)
	f.accepted = append(f.accepted, u.ID)
	return nil
}

type fakeDocs struct {
	docs     map[string]domain.Document
	grants   map[string]map[string]domain.Role // document → user → role
	seen     []string                          // "doc/user" passed to MarkSeen
	examples []domain.Log
	seq      int
}

func newFakeDocs() *fakeDocs {
	return &fakeDocs{docs: map[string]domain.Document{}, grants: map[string]map[string]domain.Role{}}
}

func (f *fakeDocs) grant(docID, userID string, r domain.Role) {
	if f.grants[docID] == nil {
		f.grants[docID] = map[string]domain.Role{}
	}
	f.grants[docID][userID] = r
}

func (f *fakeDocs) roleOf(d domain.Document, userID string) domain.Role {
	if d.OwnerID == userID {
		return domain.RoleOwner
	}
	return f.grants[d.ID][userID]
}

func (f *fakeDocs) sorted() []domain.Document {
	out := slices.Collect(maps.Values(f.docs))
	slices.SortFunc(out, func(a, b domain.Document) int { return strings.Compare(a.ID, b.ID) })
	return out
}

func (f *fakeDocs) ListByOwner(_ context.Context, ownerID string) ([]domain.Document, error) {
	out := []domain.Document{}
	for _, d := range f.sorted() {
		if d.OwnerID == ownerID {
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeDocs) ListShared(_ context.Context, userID string) ([]domain.Document, error) {
	out := []domain.Document{}
	for _, d := range f.sorted() {
		if r := f.grants[d.ID][userID]; r != "" {
			d.Role = r
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeDocs) ListWritable(_ context.Context, userID string) ([]domain.Document, error) {
	out := []domain.Document{}
	for _, d := range f.sorted() {
		if d.State == domain.DocumentActive && domain.Can(f.roleOf(d, userID), domain.PermWriteLogs) {
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeDocs) GetForUser(_ context.Context, id, userID string) (domain.Document, error) {
	d, ok := f.docs[id]
	if !ok {
		return domain.Document{}, domain.ErrNotFound
	}
	if d.Role = f.roleOf(d, userID); d.Role == "" {
		return domain.Document{}, domain.ErrNotFound
	}
	d.IsNew = d.Role != domain.RoleOwner && !slices.Contains(f.seen, id+"/"+userID)
	return d, nil
}
func (f *fakeDocs) MarkSeen(_ context.Context, id, userID string) error {
	f.seen = append(f.seen, id+"/"+userID)
	return nil
}
func (f *fakeDocs) Create(_ context.Context, d domain.Document, examples []domain.Log) (domain.Document, error) {
	f.examples = examples
	f.seq++
	d.ID = "d" + strconv.Itoa(f.seq)
	f.docs[d.ID] = d
	return d, nil
}
func (f *fakeDocs) Update(_ context.Context, d domain.Document) (domain.Document, error) {
	if _, ok := f.docs[d.ID]; !ok {
		return domain.Document{}, domain.ErrNotFound
	}
	f.docs[d.ID] = d
	return d, nil
}
func (f *fakeDocs) Delete(_ context.Context, id string) error {
	if _, ok := f.docs[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.docs, id)
	return nil
}

type fakeTenants struct {
	members     map[string]domain.User
	invitations map[string]domain.Invitation
	seq         int
}

func newFakeTenants() *fakeTenants {
	return &fakeTenants{members: map[string]domain.User{}, invitations: map[string]domain.Invitation{}}
}

func (f *fakeTenants) ListMembers(context.Context) ([]domain.User, error) {
	out := []domain.User{}
	for _, u := range f.members {
		out = append(out, u)
	}
	return out, nil
}
func (f *fakeTenants) DeleteMember(_ context.Context, id string) error {
	if _, ok := f.members[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.members, id)
	return nil
}
func (f *fakeTenants) ListInvitations(context.Context) ([]domain.Invitation, error) {
	out := []domain.Invitation{}
	for _, i := range f.invitations {
		out = append(out, i)
	}
	return out, nil
}
func (f *fakeTenants) CreateInvitation(_ context.Context, inv domain.Invitation) (domain.Invitation, error) {
	for _, i := range f.invitations {
		if i.Email == inv.Email {
			return domain.Invitation{}, domain.ErrConflict
		}
	}
	f.seq++
	inv.ID = "i" + strconv.Itoa(f.seq)
	f.invitations[inv.ID] = inv
	return inv, nil
}
func (f *fakeTenants) DeleteInvitation(_ context.Context, id string) error {
	if _, ok := f.invitations[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.invitations, id)
	return nil
}

type fakeLogs struct {
	logs      map[string]domain.Log
	filter    domain.LogFilter
	examples  int // DeleteExamples calls
	seq       int
	dashCalls int
}

func newFakeLogs() *fakeLogs { return &fakeLogs{logs: map[string]domain.Log{}} }

func (f *fakeLogs) List(_ context.Context, documentID string, fl domain.LogFilter) (domain.LogPage, error) {
	f.filter = fl
	p := domain.LogPage{Items: []domain.Log{}}
	for _, l := range f.logs {
		if l.DocumentID == documentID {
			p.Items = append(p.Items, l)
		}
	}
	// Newest first; same timestamp → later ID first, as created_at desc in Postgres.
	slices.SortFunc(p.Items, func(a, b domain.Log) int {
		return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), cmp.Compare(len(b.ID), len(a.ID)), strings.Compare(b.ID, a.ID))
	})
	p.Total = len(p.Items)
	if fl.PerPage > 0 {
		start := min((max(fl.Page, 1)-1)*fl.PerPage, len(p.Items))
		p.Items = p.Items[start:min(start+fl.PerPage, len(p.Items))]
	}
	return p, nil
}
func (f *fakeLogs) Get(_ context.Context, documentID, id string) (domain.Log, error) {
	l, ok := f.logs[id]
	if !ok || l.DocumentID != documentID {
		return domain.Log{}, domain.ErrNotFound
	}
	return l, nil
}
func (f *fakeLogs) Create(_ context.Context, l domain.Log) (domain.Log, error) {
	f.seq++
	l.ID = "l" + strconv.Itoa(f.seq)
	f.logs[l.ID] = l
	return l, nil
}
func (f *fakeLogs) Update(_ context.Context, l domain.Log) (domain.Log, error) {
	if _, ok := f.logs[l.ID]; !ok {
		return domain.Log{}, domain.ErrNotFound
	}
	f.logs[l.ID] = l
	return l, nil
}
func (f *fakeLogs) Delete(ctx context.Context, documentID, id string) error {
	if _, err := f.Get(ctx, documentID, id); err != nil {
		return err
	}
	delete(f.logs, id)
	return nil
}
func (f *fakeLogs) DeleteExamples(context.Context, string) error { f.examples++; return nil }
func (f *fakeLogs) ListTags(context.Context) ([]string, error)   { return []string{"project"}, nil }
func (f *fakeLogs) Dashboard(_ context.Context, documentID string, p domain.Period) (domain.Dashboard, error) {
	f.dashCalls++
	d := domain.Dashboard{From: p.From, To: p.To}
	for _, l := range f.logs {
		if l.DocumentID == documentID && !l.IsExample {
			d.Total++
		}
	}
	return d, nil
}

type fakeImpact struct {
	statement string
	err       error
	calls     int
}

func (f *fakeImpact) Extract(context.Context, string, string) (string, error) {
	f.calls++
	return f.statement, f.err
}

type fakeCache struct {
	data map[string][]byte
	ttl  map[string]time.Duration
	err  error
}

func newFakeCache() *fakeCache {
	return &fakeCache{data: map[string][]byte{}, ttl: map[string]time.Duration{}}
}
func (f *fakeCache) Get(_ context.Context, k string) ([]byte, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	v, ok := f.data[k]
	return v, ok, nil
}
func (f *fakeCache) GetDel(ctx context.Context, k string) ([]byte, bool, error) {
	v, ok, err := f.Get(ctx, k)
	delete(f.data, k)
	return v, ok, err
}
func (f *fakeCache) Set(_ context.Context, k string, v []byte, ttl time.Duration) error {
	if f.err != nil {
		return f.err
	}
	f.data[k], f.ttl[k] = v, ttl
	return nil
}
func (f *fakeCache) Delete(_ context.Context, k string) error { delete(f.data, k); return f.err }

type fakeTelegramLinks struct {
	byUser  map[string]domain.TelegramLink
	err     error // returned by Link when set
	findErr error // returned by FindByTelegramID when set
}

func newFakeTelegramLinks() *fakeTelegramLinks {
	return &fakeTelegramLinks{byUser: map[string]domain.TelegramLink{}}
}
func (f *fakeTelegramLinks) FindByTelegramID(_ context.Context, id int64) (domain.TelegramLink, error) {
	if f.findErr != nil {
		return domain.TelegramLink{}, f.findErr
	}
	for _, l := range f.byUser {
		if l.TelegramUserID == id {
			return l, nil
		}
	}
	return domain.TelegramLink{}, domain.ErrNotFound
}
func (f *fakeTelegramLinks) Get(_ context.Context, userID string) (domain.TelegramLink, error) {
	l, ok := f.byUser[userID]
	if !ok {
		return domain.TelegramLink{}, domain.ErrNotFound
	}
	return l, nil
}
func (f *fakeTelegramLinks) Link(_ context.Context, l domain.TelegramLink) error {
	if f.err != nil {
		return f.err
	}
	for _, o := range f.byUser {
		if o.TelegramUserID == l.TelegramUserID && o.UserID != l.UserID {
			return domain.ErrConflict
		}
	}
	f.byUser[l.UserID] = l
	return nil
}
func (f *fakeTelegramLinks) SetDocument(_ context.Context, userID, docID string) error {
	l := f.byUser[userID]
	l.DocumentID = docID
	f.byUser[userID] = l
	return nil
}
func (f *fakeTelegramLinks) Delete(_ context.Context, userID string) error {
	delete(f.byUser, userID)
	return nil
}

type fakeSharing struct {
	docs    *fakeDocs
	members map[string]domain.User // by id
	invs    map[string][]domain.DocumentInvitation
	audit   []domain.AuditEntry
	seq     int
}

func newFakeSharing(docs *fakeDocs) *fakeSharing {
	return &fakeSharing{docs: docs, members: map[string]domain.User{}, invs: map[string][]domain.DocumentInvitation{}}
}

func (f *fakeSharing) Get(_ context.Context, docID string) (domain.Sharing, error) {
	sh := domain.Sharing{Grants: []domain.Grant{}, Invitations: append([]domain.DocumentInvitation{}, f.invs[docID]...)}
	for uid, r := range f.docs.grants[docID] {
		sh.Grants = append(sh.Grants, domain.Grant{DocumentID: docID, UserID: uid, Email: f.members[uid].Email, Role: r})
	}
	slices.SortFunc(sh.Grants, func(a, b domain.Grant) int { return strings.Compare(a.UserID, b.UserID) })
	return sh, nil
}
func (f *fakeSharing) MemberByEmail(_ context.Context, email string) (domain.User, error) {
	for _, u := range f.members {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}
func (f *fakeSharing) Member(_ context.Context, id string) (domain.User, error) {
	u, ok := f.members[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}
func (f *fakeSharing) Grant(_ context.Context, g domain.Grant, a domain.AuditEntry) error {
	if _, ok := f.docs.grants[g.DocumentID][g.UserID]; ok {
		return domain.ErrConflict
	}
	f.docs.grant(g.DocumentID, g.UserID, g.Role)
	f.audit = append(f.audit, a)
	return nil
}
func (f *fakeSharing) Invite(_ context.Context, inv domain.DocumentInvitation, a domain.AuditEntry) (domain.DocumentInvitation, error) {
	for _, i := range f.invs[inv.DocumentID] {
		if i.Email == inv.Email {
			return domain.DocumentInvitation{}, domain.ErrConflict
		}
	}
	f.seq++
	inv.ID = "inv" + strconv.Itoa(f.seq)
	f.invs[inv.DocumentID] = append(f.invs[inv.DocumentID], inv)
	f.audit = append(f.audit, a)
	return inv, nil
}
func (f *fakeSharing) SetRole(_ context.Context, docID, userID string, r domain.Role, a domain.AuditEntry) error {
	if _, ok := f.docs.grants[docID][userID]; !ok {
		return domain.ErrNotFound
	}
	f.docs.grants[docID][userID] = r
	f.audit = append(f.audit, a)
	return nil
}
func (f *fakeSharing) Revoke(_ context.Context, docID, userID string, a domain.AuditEntry) error {
	if _, ok := f.docs.grants[docID][userID]; !ok {
		return domain.ErrNotFound
	}
	delete(f.docs.grants[docID], userID)
	f.audit = append(f.audit, a)
	return nil
}
func (f *fakeSharing) CancelInvitation(_ context.Context, docID, invID string, a domain.AuditEntry) error {
	i := slices.IndexFunc(f.invs[docID], func(x domain.DocumentInvitation) bool { return x.ID == invID })
	if i < 0 {
		return domain.ErrNotFound
	}
	f.invs[docID] = slices.Delete(f.invs[docID], i, i+1)
	f.audit = append(f.audit, a)
	return nil
}
func (f *fakeSharing) Transfer(_ context.Context, docID, from, to string, a domain.AuditEntry) error {
	d := f.docs.docs[docID]
	d.OwnerID = to
	f.docs.docs[docID] = d
	delete(f.docs.grants[docID], to)
	f.docs.grant(docID, from, domain.RoleEditor)
	f.audit = append(f.audit, a)
	return nil
}
func (f *fakeSharing) Audit(_ context.Context, docID string) ([]domain.AuditEntry, error) {
	out := []domain.AuditEntry{}
	for _, a := range slices.Backward(f.audit) {
		if docID == "" || a.DocumentID == docID {
			out = append(out, a)
		}
	}
	return out, nil
}

type fakeMailer struct {
	sent []string // "to: subject"
	html string   // last body
	err  error
}

func (f *fakeMailer) Send(_ context.Context, to, subject, html string) error {
	f.sent = append(f.sent, to+": "+subject)
	f.html = html
	return f.err
}

type fakeExports struct {
	jobs     map[string]domain.ExportJob
	order    []string
	settings map[string]domain.ReportSettings
	seq      int
}

func newFakeExports() *fakeExports {
	return &fakeExports{jobs: map[string]domain.ExportJob{}, settings: map[string]domain.ReportSettings{}}
}

func (f *fakeExports) Create(_ context.Context, j domain.ExportJob) (domain.ExportJob, error) {
	f.seq++
	j.ID, j.Status, j.CreatedAt = "j"+strconv.Itoa(f.seq), domain.ExportQueued, time.Now()
	j.ExpiresAt = j.CreatedAt.Add(24 * time.Hour)
	f.jobs[j.ID] = j
	f.order = append(f.order, j.ID)
	return j, nil
}
func (f *fakeExports) Get(_ context.Context, docID, id string) (domain.ExportJob, error) {
	j, ok := f.jobs[id]
	if !ok || j.DocumentID != docID {
		return domain.ExportJob{}, domain.ErrNotFound
	}
	return j, nil
}
func (f *fakeExports) List(_ context.Context, docID, userID string) ([]domain.ExportJob, error) {
	out := []domain.ExportJob{}
	for _, id := range slices.Backward(f.order) {
		if j := f.jobs[id]; j.DocumentID == docID && j.RequestedBy == userID {
			out = append(out, j)
		}
	}
	return out, nil
}
func (f *fakeExports) Claim(context.Context) (domain.ExportJob, error) {
	for _, id := range f.order {
		if j := f.jobs[id]; j.Status == domain.ExportQueued {
			j.Status = domain.ExportRunning
			f.jobs[id] = j
			return j, nil
		}
	}
	return domain.ExportJob{}, domain.ErrNotFound
}
func (f *fakeExports) Finish(_ context.Context, id, key string) error {
	j := f.jobs[id]
	j.Status, j.FileKey, j.ExpiresAt = domain.ExportDone, key, time.Now().Add(24*time.Hour)
	f.jobs[id] = j
	return nil
}
func (f *fakeExports) Fail(_ context.Context, id, reason string) error {
	j := f.jobs[id]
	j.Status, j.Error = domain.ExportFailed, reason
	f.jobs[id] = j
	return nil
}
func (f *fakeExports) Expired(context.Context) ([]domain.ExportJob, error) {
	out := []domain.ExportJob{}
	for _, id := range f.order {
		if j, ok := f.jobs[id]; ok && !time.Now().Before(j.ExpiresAt) {
			out = append(out, j)
		}
	}
	return out, nil
}
func (f *fakeExports) Delete(_ context.Context, id string) error { delete(f.jobs, id); return nil }
func (f *fakeExports) Settings(_ context.Context, docID string) (domain.ReportSettings, error) {
	s, ok := f.settings[docID]
	if !ok {
		return domain.ReportSettings{}, domain.ErrNotFound
	}
	return s, nil
}
func (f *fakeExports) SaveSettings(_ context.Context, _, docID string, s domain.ReportSettings) error {
	f.settings[docID] = s
	return nil
}

type fakeRenderer struct {
	got domain.Report
	err error
}

func (f *fakeRenderer) Render(_ context.Context, r domain.Report) ([]byte, error) {
	f.got = r
	return []byte("%PDF"), f.err
}

type fakeFiles struct{ files map[string][]byte }

func newFakeFiles() *fakeFiles                                       { return &fakeFiles{files: map[string][]byte{}} }
func (f *fakeFiles) Put(_ context.Context, k string, b []byte) error { f.files[k] = b; return nil }
func (f *fakeFiles) Get(_ context.Context, k string) ([]byte, error) {
	b, ok := f.files[k]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return b, nil
}
func (f *fakeFiles) Delete(_ context.Context, k string) error { delete(f.files, k); return nil }
