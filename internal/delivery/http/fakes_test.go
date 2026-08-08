package httpdelivery

// In-memory fakes for every domain repository interface RegisterRoutes needs.
// They follow the fake pattern already used by the invoice and lead service
// tests — hand-written structs implementing the domain interface, no mock
// library. Each one keeps its rows in a map or slice so a test can seed the
// state a request should see, then read back what the request wrote.
//
// Every fake embeds fakeErr: calling Fail(err) makes each of its methods return
// that error, which is how handler failure paths ("database error" vs "not
// found") get exercised.
//
// Each fake also carries a Seed method for inserting a row without going through
// Create. Several have no caller yet; they are kept so the set is uniform —
// every repository is seeded the same way, whichever one a later test reaches
// for.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/airport"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/chatbot"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/uow"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

// errNotFound is what every fake returns for a missing row. It IS the error the
// Postgres repositories surface — they return the driver's "no rows" unwrapped —
// so a handler that tells a missing row apart from a broken query behaves the
// same here as in a deployment.
var errNotFound = sql.ErrNoRows

// errWriteFailed stands in for a write the database refused — a constraint
// violation, a dropped connection. What it is does not matter; that the caller
// must leave nothing behind does.
var errWriteFailed = errors.New("write failed")

// fakeErr is embedded by every fake so a test can force its methods to fail.
type fakeErr struct{ err error }

// Fail makes every subsequent call on this repository return err. Passing nil
// clears it, which is how a test replays an operation that failed once.
func (f *fakeErr) Fail(err error) { f.err = err }

// ─── uow.Runner ───────────────────────────────────────────────────────────────

// fakeUnitOfWork runs a unit of work against the in-memory fakes and undoes
// every write when it fails.
//
// The rollback is the whole point of this type. A passthrough that merely called
// fn would leave a half-written conversion sitting in the fakes and report it as
// intended state, so the tests that exist to prove nothing is left behind would
// pass against a service that had never been made transactional at all.
type fakeUnitOfWork struct {
	participants  *fakeParticipantRepo
	leads         *fakeLeadRepo
	portalUsers   *fakePortalUserRepo
	invoices      *fakeInvoiceRepo
	proofs        *fakeProofRepo
	gatewayOrders *fakeGatewayOrderRepo
	// Committed and RolledBack count outcomes, so a test can tell "the unit
	// rolled back" apart from "the unit was never entered".
	Committed  int
	RolledBack int
}

// snapshotRows copies a fake's rows aside and returns the undo. Values are
// deep-copied because operations mutate rows in place, and a restore that shared
// them would hand back rows already carrying the changes it was meant to undo.
func snapshotRows[T any](rows *map[string]*T, order *[]string) func() {
	saved := make(map[string]*T, len(*rows))
	for id, row := range *rows {
		clone := *row
		saved[id] = &clone
	}
	savedOrder := append([]string{}, *order...)
	return func() { *rows, *order = saved, savedOrder }
}

func (u *fakeUnitOfWork) Do(ctx context.Context, fn func(context.Context, uow.Repos) error) error {
	undo := []func(){
		u.participants.snapshot(),
		u.leads.snapshot(),
		u.portalUsers.snapshot(),
		u.invoices.snapshot(),
		u.proofs.snapshot(),
		u.gatewayOrders.snapshot(),
	}
	if err := fn(ctx, uow.Repos{
		Participants:  u.participants,
		Leads:         u.leads,
		PortalUsers:   u.portalUsers,
		Invoices:      u.invoices,
		Proofs:        u.proofs,
		GatewayOrders: u.gatewayOrders,
	}); err != nil {
		for _, restore := range undo {
			restore()
		}
		u.RolledBack++
		return err
	}
	u.Committed++
	return nil
}

// ─── user.Repository ──────────────────────────────────────────────────────────

type fakeUserRepo struct {
	fakeErr
	users map[string]*domainUser.User
	order []string // insertion order, so listings are deterministic
	// panicRole/panicValue make ListByRole panic for one role only. Scoped to a
	// role because the lead flow lists konsultan on the request goroutine and
	// admin on the notification goroutine: it is how a test raises a panic in
	// background work without also raising one in the request.
	panicRole  string
	panicValue any
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[string]*domainUser.User{}}
}

// Seed inserts a user directly, bypassing Create's bookkeeping.
func (r *fakeUserRepo) Seed(u domainUser.User) *domainUser.User {
	if _, exists := r.users[u.ID]; !exists {
		r.order = append(r.order, u.ID)
	}
	r.users[u.ID] = &u
	return r.users[u.ID]
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domainUser.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, id := range r.order {
		if u := r.users[id]; strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, errNotFound
}

func (r *fakeUserRepo) GetByID(_ context.Context, id string) (*domainUser.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	u, ok := r.users[id]
	if !ok {
		return nil, errNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) Create(_ context.Context, u *domainUser.User) error {
	if r.err != nil {
		return r.err
	}
	if u.ID == "" {
		u.ID = fmt.Sprintf("user-%d", len(r.order)+1)
	}
	u.CreatedAt = time.Now()
	r.Seed(*u)
	return nil
}

func (r *fakeUserRepo) Update(_ context.Context, u *domainUser.User) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.users[u.ID]; !ok {
		return errNotFound
	}
	u.UpdatedAt = time.Now()
	r.Seed(*u)
	return nil
}

func (r *fakeUserRepo) Deactivate(_ context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	u, ok := r.users[id]
	if !ok {
		return errNotFound
	}
	u.IsActive = false
	return nil
}

// PanicOnListByRole makes every subsequent ListByRole for role panic with v.
func (r *fakeUserRepo) PanicOnListByRole(role string, v any) {
	r.panicRole, r.panicValue = role, v
}

func (r *fakeUserRepo) ListByRole(_ context.Context, role string) ([]domainUser.User, error) {
	if r.panicValue != nil && r.panicRole == role {
		panic(r.panicValue)
	}
	if r.err != nil {
		return nil, r.err
	}
	out := []domainUser.User{}
	for _, id := range r.order {
		// Deactivated accounts are excluded, as the Postgres query does
		// (WHERE role=$1 AND is_active=true).
		if u := r.users[id]; u.Role == role && u.IsActive {
			out = append(out, *u)
		}
	}
	return out, nil
}

func (r *fakeUserRepo) ListKonsultan(ctx context.Context) ([]domainUser.User, error) {
	return r.ListByRole(ctx, "konsultan")
}

func (r *fakeUserRepo) CountActiveleadsByConsultant(_ context.Context, _ string) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return 0, nil
}

// ─── user.TourLeaderRepository ────────────────────────────────────────────────

type fakeTourLeaderRepo struct {
	fakeErr
	leaders []domainUser.TourLeader
}

func newFakeTourLeaderRepo() *fakeTourLeaderRepo { return &fakeTourLeaderRepo{} }

func (r *fakeTourLeaderRepo) Seed(tl domainUser.TourLeader) { r.leaders = append(r.leaders, tl) }

func (r *fakeTourLeaderRepo) Create(_ context.Context, tl *domainUser.TourLeader) error {
	if r.err != nil {
		return r.err
	}
	if tl.ID == "" {
		tl.ID = fmt.Sprintf("tl-%d", len(r.leaders)+1)
	}
	r.leaders = append(r.leaders, *tl)
	return nil
}

func (r *fakeTourLeaderRepo) GetByUserID(_ context.Context, userID string) (*domainUser.TourLeader, error) {
	if r.err != nil {
		return nil, r.err
	}
	for i := range r.leaders {
		if r.leaders[i].UserID == userID {
			return &r.leaders[i], nil
		}
	}
	return nil, errNotFound
}

func (r *fakeTourLeaderRepo) Update(_ context.Context, tl *domainUser.TourLeader) error {
	if r.err != nil {
		return r.err
	}
	for i := range r.leaders {
		if r.leaders[i].UserID == tl.UserID {
			r.leaders[i] = *tl
			return nil
		}
	}
	return errNotFound
}

func (r *fakeTourLeaderRepo) List(_ context.Context) ([]domainUser.TourLeader, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]domainUser.TourLeader{}, r.leaders...), nil
}

// ─── lead.Repository ──────────────────────────────────────────────────────────

type fakeLeadRepo struct {
	fakeErr
	leads map[string]*domainLead.Lead
	order []string
	// history is the lead_status_history table: every transition, with its
	// actor. Written by the same call that changes the status, as the Postgres
	// repository's chained CTE does.
	history []domainLead.StatusChange
	// StatusChanges records every UpdateStatus call, including the actor, so
	// audit-trail expectations can be asserted without a database.
	StatusChanges []LeadStatusChange
	// panicOnCreate, when set, makes Create panic with it instead of storing the
	// lead — a repository dying mid-request, which is what the runtime-resilience
	// tests need in order to raise a panic from inside a real handler chain.
	panicOnCreate any
	// markConvertedErr fails only MarkConverted, so a test can break the last
	// step of a conversion while the earlier ones succeed.
	markConvertedErr error
}

// FailMarkConverted makes every subsequent MarkConverted return err.
func (r *fakeLeadRepo) FailMarkConverted(err error) { r.markConvertedErr = err }

// LeadStatusChange is one recorded UpdateStatus call.
type LeadStatusChange struct {
	LeadID    string
	From      string
	To        string
	ChangedBy string
}

func newFakeLeadRepo() *fakeLeadRepo {
	return &fakeLeadRepo{leads: map[string]*domainLead.Lead{}}
}

func (r *fakeLeadRepo) Seed(l domainLead.Lead) *domainLead.Lead {
	if _, exists := r.leads[l.ID]; !exists {
		r.order = append(r.order, l.ID)
	}
	r.leads[l.ID] = &l
	return r.leads[l.ID]
}

// PanicOnCreate makes every subsequent Create panic with v.
func (r *fakeLeadRepo) PanicOnCreate(v any) { r.panicOnCreate = v }

func (r *fakeLeadRepo) Create(_ context.Context, l *domainLead.Lead) error {
	if r.panicOnCreate != nil {
		panic(r.panicOnCreate)
	}
	if r.err != nil {
		return r.err
	}
	if l.ID == "" {
		l.ID = fmt.Sprintf("lead-%d", len(r.order)+1)
	}
	l.CreatedAt = time.Now()
	r.Seed(*l)
	return nil
}

func (r *fakeLeadRepo) Update(_ context.Context, l *domainLead.Lead) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.leads[l.ID]; !ok {
		return errNotFound
	}
	r.Seed(*l)
	return nil
}

func (r *fakeLeadRepo) GetByID(_ context.Context, id string) (*domainLead.Lead, error) {
	if r.err != nil {
		return nil, r.err
	}
	l, ok := r.leads[id]
	if !ok || l.DeletedAt != nil {
		return nil, errNotFound
	}
	return l, nil
}

func (r *fakeLeadRepo) List(_ context.Context, f domainLead.Filter) ([]domainLead.Lead, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	out := []domainLead.Lead{}
	for _, id := range r.order {
		l := r.leads[id]
		if l.DeletedAt != nil {
			continue
		}
		if f.Status != nil && l.Status != *f.Status {
			continue
		}
		if f.AssignedTo != nil && (l.AssignedTo == nil || *l.AssignedTo != *f.AssignedTo) {
			continue
		}
		if f.PackageID != nil && l.PackageID != *f.PackageID {
			continue
		}
		out = append(out, *l)
	}
	return out, len(out), nil
}

func (r *fakeLeadRepo) UpdateStatus(_ context.Context, id, status, changedBy string) error {
	if r.err != nil {
		return r.err
	}
	l, ok := r.leads[id]
	if !ok {
		return errNotFound
	}
	r.StatusChanges = append(r.StatusChanges, LeadStatusChange{
		LeadID: id, From: l.Status, To: status, ChangedBy: changedBy,
	})
	r.recordStatusChange(id, l.Status, status, changedBy)
	l.Status = status
	l.UpdatedAt = time.Now()
	return nil
}

// recordStatusChange appends the history row the status update writes with it.
func (r *fakeLeadRepo) recordStatusChange(leadID, from, to, changedBy string) {
	name := domainLead.SystemActor
	if changedBy != "" {
		name = changedBy
	}
	r.history = append(r.history, domainLead.StatusChange{
		ID:         fmt.Sprintf("status-change-%d", len(r.history)+1),
		LeadID:     leadID,
		FromStatus: from,
		ToStatus:   to,
		ChangedBy:  changedBy,
		ChangedAt:  time.Now(),
		// The Postgres query joins the actor's name; the id is enough here.
		ChangedByName: name,
	})
}

func (r *fakeLeadRepo) ListStatusHistory(_ context.Context, leadID string) ([]domainLead.StatusChange, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []domainLead.StatusChange{}
	for _, c := range r.history {
		if c.LeadID == leadID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeLeadRepo) AssignTo(_ context.Context, leadID, consultantID string) error {
	if r.err != nil {
		return r.err
	}
	l, ok := r.leads[leadID]
	if !ok {
		return errNotFound
	}
	l.AssignedTo = &consultantID
	return nil
}

func (r *fakeLeadRepo) MarkConverted(_ context.Context, leadID, changedBy string) error {
	if r.markConvertedErr != nil {
		return r.markConvertedErr
	}
	if r.err != nil {
		return r.err
	}
	l, ok := r.leads[leadID]
	if !ok {
		return errNotFound
	}
	// Only from 'deal', as the Postgres UPDATE's WHERE clause enforces — that
	// condition is what stops one lead becoming two participants.
	if l.Status != "deal" {
		return domainLead.ErrNotConvertible
	}
	r.recordStatusChange(leadID, l.Status, "peserta", changedBy)
	now := time.Now()
	l.ConvertedAt = &now
	l.Status = "peserta"
	return nil
}

func (r *fakeLeadRepo) CountActiveByConsultant(_ context.Context, consultantID string) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	n := 0
	for _, l := range r.leads {
		if l.AssignedTo != nil && *l.AssignedTo == consultantID && l.DeletedAt == nil {
			n++
		}
	}
	return n, nil
}

func (r *fakeLeadRepo) snapshot() func() {
	restoreRows := snapshotRows(&r.leads, &r.order)
	changes := append([]LeadStatusChange{}, r.StatusChanges...)
	// The history table is inside the unit too: a rolled-back conversion left no
	// status change, so the trail must not claim one happened.
	history := append([]domainLead.StatusChange{}, r.history...)
	return func() {
		restoreRows()
		r.StatusChanges, r.history = changes, history
	}
}

// ─── lead.NoteRepository ──────────────────────────────────────────────────────

type fakeNoteRepo struct {
	fakeErr
	notes []domainLead.Note
}

func newFakeNoteRepo() *fakeNoteRepo { return &fakeNoteRepo{} }

func (r *fakeNoteRepo) Seed(n domainLead.Note) { r.notes = append(r.notes, n) }

func (r *fakeNoteRepo) Create(_ context.Context, n *domainLead.Note) error {
	if r.err != nil {
		return r.err
	}
	if n.ID == "" {
		n.ID = fmt.Sprintf("note-%d", len(r.notes)+1)
	}
	n.CreatedAt = time.Now()
	r.notes = append(r.notes, *n)
	return nil
}

func (r *fakeNoteRepo) ListByLead(_ context.Context, leadID string) ([]domainLead.Note, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []domainLead.Note{}
	for _, n := range r.notes {
		if n.LeadID == leadID {
			out = append(out, n)
		}
	}
	return out, nil
}

// ─── participant.Repository ───────────────────────────────────────────────────

type fakeParticipantRepo struct {
	fakeErr
	participants map[string]*domainParticipant.Participant
	order        []string
}

func newFakeParticipantRepo() *fakeParticipantRepo {
	return &fakeParticipantRepo{participants: map[string]*domainParticipant.Participant{}}
}

func (r *fakeParticipantRepo) Seed(p domainParticipant.Participant) *domainParticipant.Participant {
	if _, exists := r.participants[p.ID]; !exists {
		r.order = append(r.order, p.ID)
	}
	r.participants[p.ID] = &p
	return r.participants[p.ID]
}

func (r *fakeParticipantRepo) Create(_ context.Context, p *domainParticipant.Participant) error {
	if r.err != nil {
		return r.err
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("participant-%d", len(r.order)+1)
	}
	p.CreatedAt = time.Now()
	r.Seed(*p)
	return nil
}

func (r *fakeParticipantRepo) Update(_ context.Context, p *domainParticipant.Participant) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.participants[p.ID]; !ok {
		return errNotFound
	}
	r.Seed(*p)
	return nil
}

func (r *fakeParticipantRepo) GetByID(_ context.Context, id string) (*domainParticipant.Participant, error) {
	if r.err != nil {
		return nil, r.err
	}
	p, ok := r.participants[id]
	if !ok {
		return nil, errNotFound
	}
	return p, nil
}

func (r *fakeParticipantRepo) GetByPhone(_ context.Context, phone string) (*domainParticipant.Participant, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, id := range r.order {
		if p := r.participants[id]; p.Phone == phone {
			return p, nil
		}
	}
	return nil, errNotFound
}

func (r *fakeParticipantRepo) ListByPortalUser(_ context.Context, portalUserID, phone string) ([]domainParticipant.Participant, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []domainParticipant.Participant{}
	for _, id := range r.order {
		p := r.participants[id]
		byPortalUser := portalUserID != "" && p.PortalUserID != nil && *p.PortalUserID == portalUserID
		byPhone := phone != "" && p.Phone == phone
		if byPortalUser || byPhone {
			out = append(out, *p)
		}
	}
	// Newest departure first, matching the Postgres repository's ordering.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].BatchDepartureDate == nil || out[j].BatchDepartureDate == nil {
			return false
		}
		return out[i].BatchDepartureDate.After(*out[j].BatchDepartureDate)
	})
	return out, nil
}

func (r *fakeParticipantRepo) List(_ context.Context, f domainParticipant.Filter) ([]domainParticipant.Participant, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	out := []domainParticipant.Participant{}
	for _, id := range r.order {
		p := r.participants[id]
		if f.BatchID != nil && p.BatchID != *f.BatchID {
			continue
		}
		if f.IsActive != nil && p.IsActive != *f.IsActive {
			continue
		}
		out = append(out, *p)
	}
	return out, len(out), nil
}

func (r *fakeParticipantRepo) Activate(_ context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	p, ok := r.participants[id]
	if !ok {
		return errNotFound
	}
	p.IsActive = true
	return nil
}

func (r *fakeParticipantRepo) SetNIK(_ context.Context, id, nik string) error {
	if r.err != nil {
		return r.err
	}
	p, ok := r.participants[id]
	if !ok {
		return errNotFound
	}
	p.NIK = nik
	return nil
}

func (r *fakeParticipantRepo) ListByBatch(_ context.Context, batchID string) ([]domainParticipant.Participant, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []domainParticipant.Participant{}
	for _, id := range r.order {
		if p := r.participants[id]; p.BatchID == batchID {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (r *fakeParticipantRepo) ListByDepartureDaysAhead(_ context.Context, _ int) ([]domainParticipant.Participant, error) {
	if r.err != nil {
		return nil, r.err
	}
	return nil, nil
}

func (r *fakeParticipantRepo) ListWithUnpaidInvoiceDaysOld(_ context.Context, _ int) ([]domainParticipant.Participant, error) {
	if r.err != nil {
		return nil, r.err
	}
	return nil, nil
}

func (r *fakeParticipantRepo) snapshot() func() {
	return snapshotRows(&r.participants, &r.order)
}

// ─── portaluser.Repository ────────────────────────────────────────────────────

type fakePortalUserRepo struct {
	fakeErr
	users map[string]*portaluser.PortalUser
	order []string
}

func newFakePortalUserRepo() *fakePortalUserRepo {
	return &fakePortalUserRepo{users: map[string]*portaluser.PortalUser{}}
}

func (r *fakePortalUserRepo) Seed(u portaluser.PortalUser) *portaluser.PortalUser {
	if _, exists := r.users[u.ID]; !exists {
		r.order = append(r.order, u.ID)
	}
	r.users[u.ID] = &u
	return r.users[u.ID]
}

func (r *fakePortalUserRepo) Create(_ context.Context, u *portaluser.PortalUser) error {
	if r.err != nil {
		return r.err
	}
	if u.ID == "" {
		u.ID = fmt.Sprintf("portal-user-%d", len(r.order)+1)
	}
	u.CreatedAt = time.Now()
	r.Seed(*u)
	return nil
}

func (r *fakePortalUserRepo) GetByID(_ context.Context, id string) (*portaluser.PortalUser, error) {
	if r.err != nil {
		return nil, r.err
	}
	u, ok := r.users[id]
	if !ok {
		return nil, errNotFound
	}
	return u, nil
}

func (r *fakePortalUserRepo) GetByPhone(_ context.Context, phone string) (*portaluser.PortalUser, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, id := range r.order {
		if u := r.users[id]; u.Phone == phone {
			return u, nil
		}
	}
	return nil, errNotFound
}

func (r *fakePortalUserRepo) UpdatePassword(_ context.Context, id, passwordHash string) error {
	if r.err != nil {
		return r.err
	}
	u, ok := r.users[id]
	if !ok {
		return errNotFound
	}
	u.PasswordHash = passwordHash
	return nil
}

func (r *fakePortalUserRepo) snapshot() func() {
	return snapshotRows(&r.users, &r.order)
}

// ─── invoice.Repository ───────────────────────────────────────────────────────

type fakeInvoiceRepo struct {
	fakeErr
	invoices map[string]*domainInvoice.Invoice
	order    []string
}

func newFakeInvoiceRepo() *fakeInvoiceRepo {
	return &fakeInvoiceRepo{invoices: map[string]*domainInvoice.Invoice{}}
}

func (r *fakeInvoiceRepo) Seed(inv domainInvoice.Invoice) *domainInvoice.Invoice {
	if _, exists := r.invoices[inv.ID]; !exists {
		r.order = append(r.order, inv.ID)
	}
	r.invoices[inv.ID] = &inv
	return r.invoices[inv.ID]
}

func (r *fakeInvoiceRepo) Create(_ context.Context, inv *domainInvoice.Invoice) error {
	if r.err != nil {
		return r.err
	}
	if inv.ID == "" {
		inv.ID = "inv-" + inv.InvoiceNumber
	}
	inv.CreatedAt = time.Now()
	r.Seed(*inv)
	return nil
}

func (r *fakeInvoiceRepo) Update(_ context.Context, inv *domainInvoice.Invoice) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.invoices[inv.ID]; !ok {
		return errNotFound
	}
	r.Seed(*inv)
	return nil
}

func (r *fakeInvoiceRepo) snapshot() func() {
	return snapshotRows(&r.invoices, &r.order)
}

// SoftDelete marks an invoice deleted the way a delete endpoint would, so a test
// can ask what the rest of the system does with it afterwards.
func (r *fakeInvoiceRepo) SoftDelete(id string) {
	if inv, ok := r.invoices[id]; ok {
		now := time.Now()
		inv.DeletedAt = &now
	}
}

func (r *fakeInvoiceRepo) GetByID(_ context.Context, id string) (*domainInvoice.Invoice, error) {
	if r.err != nil {
		return nil, r.err
	}
	inv, ok := r.invoices[id]
	if !ok || inv.DeletedAt != nil {
		return nil, errNotFound
	}
	return inv, nil
}

func (r *fakeInvoiceRepo) GetByNumber(_ context.Context, number string) (*domainInvoice.Invoice, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, id := range r.order {
		if inv := r.invoices[id]; inv.InvoiceNumber == number {
			return inv, nil
		}
	}
	return nil, errNotFound
}

func (r *fakeInvoiceRepo) List(_ context.Context, f domainInvoice.Filter) ([]domainInvoice.Invoice, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	out := []domainInvoice.Invoice{}
	for _, id := range r.order {
		inv := r.invoices[id]
		if inv.DeletedAt != nil {
			continue
		}
		if f.Status != nil && inv.Status != *f.Status {
			continue
		}
		if f.ParticipantID != nil && inv.ParticipantID != *f.ParticipantID {
			continue
		}
		out = append(out, *inv)
	}
	return out, len(out), nil
}

func (r *fakeInvoiceRepo) Confirm(_ context.Context, id, confirmedBy string) error {
	if r.err != nil {
		return r.err
	}
	inv, ok := r.invoices[id]
	if !ok {
		return errNotFound
	}
	now := time.Now()
	inv.Status = "lunas"
	inv.ConfirmedAt = &now
	inv.ConfirmedBy = &confirmedBy
	return nil
}

// NextSequence mirrors the Postgres query: the highest sequence yet used in the
// month, plus one — counting soft-deleted rows, so a deleted invoice's number is
// never handed out again (§13.7). A counter that only saw live rows would give
// the deleted one's number straight back.
func (r *fakeInvoiceRepo) NextSequence(_ context.Context, yearMonth string) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	prefix := "INV-" + yearMonth + "-"
	maxSeq := 0
	for _, inv := range r.invoices {
		if !strings.HasPrefix(inv.InvoiceNumber, prefix) {
			continue
		}
		if seq, err := strconv.Atoi(strings.TrimPrefix(inv.InvoiceNumber, prefix)); err == nil && seq > maxSeq {
			maxSeq = seq
		}
	}
	return maxSeq + 1, nil
}

func (r *fakeInvoiceRepo) ListUnpaidOlderThan(_ context.Context, _ int) ([]domainInvoice.Invoice, error) {
	if r.err != nil {
		return nil, r.err
	}
	return nil, nil
}

func (r *fakeInvoiceRepo) GetByOrderID(_ context.Context, orderID string) (*domainInvoice.Invoice, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, id := range r.order {
		if inv := r.invoices[id]; inv.MidtransOrderID == orderID {
			return inv, nil
		}
	}
	return nil, errNotFound
}

func (r *fakeInvoiceRepo) SetSnap(_ context.Context, id, snapToken, orderID string) error {
	if r.err != nil {
		return r.err
	}
	inv, ok := r.invoices[id]
	if !ok {
		return errNotFound
	}
	inv.SnapToken = snapToken
	inv.MidtransOrderID = orderID
	return nil
}

// ─── invoice.PaymentProofRepository ───────────────────────────────────────────

type fakeProofRepo struct {
	fakeErr
	proofs []domainInvoice.PaymentProof
}

func newFakeProofRepo() *fakeProofRepo { return &fakeProofRepo{} }

func (r *fakeProofRepo) snapshot() func() {
	proofs := append([]domainInvoice.PaymentProof{}, r.proofs...)
	return func() { r.proofs = proofs }
}

func (r *fakeProofRepo) Seed(pp domainInvoice.PaymentProof) { r.proofs = append(r.proofs, pp) }

func (r *fakeProofRepo) Create(_ context.Context, pp *domainInvoice.PaymentProof) error {
	if r.err != nil {
		return r.err
	}
	if pp.ID == "" {
		pp.ID = fmt.Sprintf("proof-%d", len(r.proofs)+1)
	}
	pp.UploadedAt = time.Now()
	if pp.Status == "" {
		pp.Status = "menunggu"
	}
	r.proofs = append(r.proofs, *pp)
	return nil
}

func (r *fakeProofRepo) GetByID(_ context.Context, id string) (*domainInvoice.PaymentProof, error) {
	if r.err != nil {
		return nil, r.err
	}
	for i := range r.proofs {
		if r.proofs[i].ID == id {
			return &r.proofs[i], nil
		}
	}
	return nil, errNotFound
}

func (r *fakeProofRepo) GetByInvoice(_ context.Context, invoiceID string) ([]domainInvoice.PaymentProof, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []domainInvoice.PaymentProof{}
	for _, p := range r.proofs {
		if p.InvoiceID == invoiceID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakeProofRepo) Review(_ context.Context, id, status, reviewedBy, notes string) error {
	if r.err != nil {
		return r.err
	}
	for i := range r.proofs {
		if r.proofs[i].ID == id {
			now := time.Now()
			r.proofs[i].Status = status
			r.proofs[i].ReviewedBy = &reviewedBy
			r.proofs[i].ReviewNotes = notes
			r.proofs[i].ReviewedAt = &now
			return nil
		}
	}
	return errNotFound
}

// ─── invoice.GatewayOrderRepository ───────────────────────────────────────────

type fakeGatewayOrderRepo struct {
	fakeErr
	orders []domainInvoice.GatewayOrder
	// applied is the idempotency table: one entry per notification already
	// recorded, keyed the way the unique index keys it.
	applied map[string]bool
}

func newFakeGatewayOrderRepo() *fakeGatewayOrderRepo {
	return &fakeGatewayOrderRepo{applied: map[string]bool{}}
}

func (r *fakeGatewayOrderRepo) Seed(o domainInvoice.GatewayOrder) {
	r.orders = append(r.orders, o)
}

func (r *fakeGatewayOrderRepo) Create(_ context.Context, o *domainInvoice.GatewayOrder) error {
	if r.err != nil {
		return r.err
	}
	if o.ID == "" {
		o.ID = fmt.Sprintf("gateway-order-%d", len(r.orders)+1)
	}
	o.CreatedAt = time.Now()
	r.orders = append(r.orders, *o)
	return nil
}

func (r *fakeGatewayOrderRepo) FindInvoiceIDByOrder(_ context.Context, orderID string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	for _, o := range r.orders {
		if o.OrderID == orderID {
			return o.InvoiceID, nil
		}
	}
	return "", errNotFound
}

func (r *fakeGatewayOrderRepo) ClaimNotification(_ context.Context, n domainInvoice.GatewayNotification) (bool, error) {
	if r.err != nil {
		return false, r.err
	}
	key := n.IdempotencyKey()
	if r.applied[key] {
		return false, nil
	}
	r.applied[key] = true
	return true, nil
}

func (r *fakeGatewayOrderRepo) snapshot() func() {
	orders := append([]domainInvoice.GatewayOrder{}, r.orders...)
	applied := make(map[string]bool, len(r.applied))
	for k, v := range r.applied {
		applied[k] = v
	}
	return func() { r.orders, r.applied = orders, applied }
}

// ─── package.Repository ───────────────────────────────────────────────────────

type fakePackageRepo struct {
	fakeErr
	packages map[string]*domainPkg.Package
	order    []string
}

func newFakePackageRepo() *fakePackageRepo {
	return &fakePackageRepo{packages: map[string]*domainPkg.Package{}}
}

func (r *fakePackageRepo) Seed(p domainPkg.Package) *domainPkg.Package {
	if _, exists := r.packages[p.ID]; !exists {
		r.order = append(r.order, p.ID)
	}
	r.packages[p.ID] = &p
	return r.packages[p.ID]
}

func (r *fakePackageRepo) Create(_ context.Context, p *domainPkg.Package) error {
	if r.err != nil {
		return r.err
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("package-%d", len(r.order)+1)
	}
	p.CreatedAt = time.Now()
	r.Seed(*p)
	return nil
}

func (r *fakePackageRepo) Update(_ context.Context, p *domainPkg.Package) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.packages[p.ID]; !ok {
		return errNotFound
	}
	r.Seed(*p)
	return nil
}

func (r *fakePackageRepo) Delete(_ context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	p, ok := r.packages[id]
	if !ok {
		return errNotFound
	}
	now := time.Now()
	p.DeletedAt = &now
	return nil
}

func (r *fakePackageRepo) GetByID(_ context.Context, id string) (*domainPkg.Package, error) {
	if r.err != nil {
		return nil, r.err
	}
	p, ok := r.packages[id]
	if !ok || p.DeletedAt != nil {
		return nil, errNotFound
	}
	return p, nil
}

func (r *fakePackageRepo) GetBySlug(_ context.Context, slug string) (*domainPkg.Package, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, id := range r.order {
		if p := r.packages[id]; p.Slug == slug && p.DeletedAt == nil {
			return p, nil
		}
	}
	return nil, errNotFound
}

func (r *fakePackageRepo) List(_ context.Context, f domainPkg.Filter) ([]domainPkg.Package, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	out := []domainPkg.Package{}
	for _, id := range r.order {
		p := r.packages[id]
		if p.DeletedAt != nil {
			continue
		}
		if f.Category != nil && p.Category != *f.Category {
			continue
		}
		if f.IsActive != nil && p.IsActive != *f.IsActive {
			continue
		}
		out = append(out, *p)
	}
	return out, len(out), nil
}

// ─── package.ImageRepository ──────────────────────────────────────────────────

type fakeImageRepo struct {
	fakeErr
	images []domainPkg.PackageImage
}

func newFakeImageRepo() *fakeImageRepo { return &fakeImageRepo{} }

func (r *fakeImageRepo) Seed(img domainPkg.PackageImage) { r.images = append(r.images, img) }

func (r *fakeImageRepo) Add(_ context.Context, img *domainPkg.PackageImage) error {
	if r.err != nil {
		return r.err
	}
	if img.ID == "" {
		img.ID = fmt.Sprintf("image-%d", len(r.images)+1)
	}
	img.CreatedAt = time.Now()
	r.images = append(r.images, *img)
	return nil
}

func (r *fakeImageRepo) Delete(_ context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	for i := range r.images {
		if r.images[i].ID == id {
			r.images = append(r.images[:i], r.images[i+1:]...)
			return nil
		}
	}
	return errNotFound
}

func (r *fakeImageRepo) ListByPackage(_ context.Context, packageID string) ([]domainPkg.PackageImage, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []domainPkg.PackageImage{}
	for _, img := range r.images {
		if img.PackageID == packageID {
			out = append(out, img)
		}
	}
	return out, nil
}

// ─── package.BatchRepository ──────────────────────────────────────────────────

type fakeBatchRepo struct {
	fakeErr
	batches map[string]*domainPkg.PackageBatch
	order   []string
}

func newFakeBatchRepo() *fakeBatchRepo {
	return &fakeBatchRepo{batches: map[string]*domainPkg.PackageBatch{}}
}

func (r *fakeBatchRepo) Seed(b domainPkg.PackageBatch) *domainPkg.PackageBatch {
	if _, exists := r.batches[b.ID]; !exists {
		r.order = append(r.order, b.ID)
	}
	r.batches[b.ID] = &b
	return r.batches[b.ID]
}

func (r *fakeBatchRepo) Create(_ context.Context, b *domainPkg.PackageBatch) error {
	if r.err != nil {
		return r.err
	}
	if b.ID == "" {
		b.ID = fmt.Sprintf("batch-%d", len(r.order)+1)
	}
	b.CreatedAt = time.Now()
	r.Seed(*b)
	return nil
}

func (r *fakeBatchRepo) Update(_ context.Context, b *domainPkg.PackageBatch) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.batches[b.ID]; !ok {
		return errNotFound
	}
	r.Seed(*b)
	return nil
}

func (r *fakeBatchRepo) GetByID(_ context.Context, id string) (*domainPkg.PackageBatch, error) {
	if r.err != nil {
		return nil, r.err
	}
	b, ok := r.batches[id]
	if !ok {
		return nil, errNotFound
	}
	return b, nil
}

func (r *fakeBatchRepo) List(_ context.Context, f domainPkg.BatchFilter) ([]domainPkg.PackageBatch, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []domainPkg.PackageBatch{}
	for _, id := range r.order {
		b := r.batches[id]
		if f.PackageID != nil && b.PackageID != *f.PackageID {
			continue
		}
		if f.Status != nil && b.Status != *f.Status {
			continue
		}
		out = append(out, *b)
	}
	return out, nil
}

func (r *fakeBatchRepo) ListByPackage(ctx context.Context, packageID string) ([]domainPkg.PackageBatch, error) {
	return r.List(ctx, domainPkg.BatchFilter{PackageID: &packageID})
}

// ─── document.Repository ──────────────────────────────────────────────────────

type fakeDocumentRepo struct {
	fakeErr
	documents map[string]*document.Document
	order     []string
}

func newFakeDocumentRepo() *fakeDocumentRepo {
	return &fakeDocumentRepo{documents: map[string]*document.Document{}}
}

func (r *fakeDocumentRepo) Seed(d document.Document) *document.Document {
	if _, exists := r.documents[d.ID]; !exists {
		r.order = append(r.order, d.ID)
	}
	r.documents[d.ID] = &d
	return r.documents[d.ID]
}

func (r *fakeDocumentRepo) Create(_ context.Context, d *document.Document) error {
	if r.err != nil {
		return r.err
	}
	if d.ID == "" {
		d.ID = fmt.Sprintf("document-%d", len(r.order)+1)
	}
	d.UploadedAt = time.Now()
	if d.Status == "" {
		d.Status = "menunggu"
	}
	r.Seed(*d)
	return nil
}

func (r *fakeDocumentRepo) GetByID(_ context.Context, id string) (*document.Document, error) {
	if r.err != nil {
		return nil, r.err
	}
	d, ok := r.documents[id]
	if !ok {
		return nil, errNotFound
	}
	return d, nil
}

func (r *fakeDocumentRepo) List(_ context.Context, f document.Filter) ([]document.Document, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []document.Document{}
	for _, id := range r.order {
		d := r.documents[id]
		if f.ParticipantID != nil && d.ParticipantID != *f.ParticipantID {
			continue
		}
		if f.Status != nil && d.Status != *f.Status {
			continue
		}
		out = append(out, *d)
	}
	return out, nil
}

func (r *fakeDocumentRepo) ListByParticipant(ctx context.Context, participantID string) ([]document.Document, error) {
	return r.List(ctx, document.Filter{ParticipantID: &participantID})
}

func (r *fakeDocumentRepo) Review(_ context.Context, id, status, reviewedBy, rejectionReason string) error {
	if r.err != nil {
		return r.err
	}
	d, ok := r.documents[id]
	if !ok {
		return errNotFound
	}
	now := time.Now()
	d.Status = status
	d.ReviewedBy = &reviewedBy
	d.RejectionReason = rejectionReason
	d.ReviewedAt = &now
	return nil
}

func (r *fakeDocumentRepo) Delete(_ context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.documents[id]; !ok {
		return errNotFound
	}
	delete(r.documents, id)
	for i, oid := range r.order {
		if oid == id {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
	return nil
}

// ─── document.OCRResultRepository ─────────────────────────────────────────────

type fakeOCRResultRepo struct {
	fakeErr
	results []document.OCRResult
}

func newFakeOCRResultRepo() *fakeOCRResultRepo { return &fakeOCRResultRepo{} }

func (r *fakeOCRResultRepo) Seed(res document.OCRResult) { r.results = append(r.results, res) }

func (r *fakeOCRResultRepo) Create(_ context.Context, res *document.OCRResult) error {
	if r.err != nil {
		return r.err
	}
	if res.ID == "" {
		res.ID = fmt.Sprintf("ocr-%d", len(r.results)+1)
	}
	res.CreatedAt = time.Now()
	r.results = append(r.results, *res)
	return nil
}

func (r *fakeOCRResultRepo) GetByDocument(_ context.Context, documentID string) (*document.OCRResult, error) {
	if r.err != nil {
		return nil, r.err
	}
	for i := range r.results {
		if r.results[i].DocumentID == documentID {
			return &r.results[i], nil
		}
	}
	return nil, errNotFound
}

func (r *fakeOCRResultRepo) LatestPassportExpiry(_ context.Context, _ string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return "", nil
}

// ─── document.CountryRequirementRepository ────────────────────────────────────

type fakeCountryReqRepo struct {
	fakeErr
	requirements []document.CountryRequirement
}

func newFakeCountryReqRepo() *fakeCountryReqRepo { return &fakeCountryReqRepo{} }

func (r *fakeCountryReqRepo) Seed(req document.CountryRequirement) {
	r.requirements = append(r.requirements, req)
}

func (r *fakeCountryReqRepo) List(_ context.Context, countryCode string) ([]document.CountryRequirement, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []document.CountryRequirement{}
	for _, req := range r.requirements {
		if countryCode == "" || req.CountryCode == countryCode {
			out = append(out, req)
		}
	}
	return out, nil
}

func (r *fakeCountryReqRepo) Create(_ context.Context, req *document.CountryRequirement) error {
	if r.err != nil {
		return r.err
	}
	if req.ID == "" {
		req.ID = fmt.Sprintf("country-req-%d", len(r.requirements)+1)
	}
	req.CreatedAt = time.Now()
	r.requirements = append(r.requirements, *req)
	return nil
}

func (r *fakeCountryReqRepo) Update(_ context.Context, req *document.CountryRequirement) error {
	if r.err != nil {
		return r.err
	}
	for i := range r.requirements {
		if r.requirements[i].ID == req.ID {
			r.requirements[i] = *req
			return nil
		}
	}
	return errNotFound
}

func (r *fakeCountryReqRepo) Delete(_ context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	for i := range r.requirements {
		if r.requirements[i].ID == id {
			r.requirements = append(r.requirements[:i], r.requirements[i+1:]...)
			return nil
		}
	}
	return errNotFound
}

// ─── airport.Repository ───────────────────────────────────────────────────────

type fakeAirportRepo struct {
	fakeErr
	checklists []airport.Checklist
}

func newFakeAirportRepo() *fakeAirportRepo { return &fakeAirportRepo{} }

func (r *fakeAirportRepo) Seed(c airport.Checklist) { r.checklists = append(r.checklists, c) }

func (r *fakeAirportRepo) InitForBatch(_ context.Context, _ string) error { return r.err }

func (r *fakeAirportRepo) GetByParticipant(_ context.Context, participantID, batchID string) (*airport.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	for i := range r.checklists {
		if r.checklists[i].ParticipantID == participantID && r.checklists[i].BatchID == batchID {
			return &r.checklists[i], nil
		}
	}
	return nil, errNotFound
}

func (r *fakeAirportRepo) ListByBatch(_ context.Context, f airport.Filter) ([]airport.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []airport.Checklist{}
	for _, c := range r.checklists {
		if c.BatchID != f.BatchID {
			continue
		}
		if f.Status != nil {
			done := c.BaggageChecked && c.TicketDistributed && c.PassportReturned
			if (*f.Status == "done") != done {
				continue
			}
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *fakeAirportRepo) UpdateBaggage(_ context.Context, participantID, batchID, handledBy string) error {
	return r.mark(participantID, batchID, handledBy, func(c *airport.Checklist, now time.Time) {
		c.BaggageChecked = true
		c.BaggageCheckedAt = &now
	})
}

func (r *fakeAirportRepo) UpdateTicket(_ context.Context, participantID, batchID, handledBy string) error {
	return r.mark(participantID, batchID, handledBy, func(c *airport.Checklist, now time.Time) {
		c.TicketDistributed = true
		c.TicketDistributedAt = &now
	})
}

func (r *fakeAirportRepo) UpdatePassport(_ context.Context, participantID, batchID, handledBy string) error {
	return r.mark(participantID, batchID, handledBy, func(c *airport.Checklist, now time.Time) {
		c.PassportReturned = true
		c.PassportReturnedAt = &now
	})
}

func (r *fakeAirportRepo) mark(participantID, batchID, handledBy string, apply func(*airport.Checklist, time.Time)) error {
	if r.err != nil {
		return r.err
	}
	for i := range r.checklists {
		if r.checklists[i].ParticipantID == participantID && r.checklists[i].BatchID == batchID {
			now := time.Now()
			apply(&r.checklists[i], now)
			r.checklists[i].HandledBy = &handledBy
			r.checklists[i].UpdatedAt = now
			return nil
		}
	}
	return errNotFound
}

func (r *fakeAirportRepo) GetBatchProgress(_ context.Context, batchID string) (*airport.BatchProgress, error) {
	if r.err != nil {
		return nil, r.err
	}
	p := &airport.BatchProgress{BatchID: batchID}
	for _, c := range r.checklists {
		if c.BatchID != batchID {
			continue
		}
		p.TotalPax++
		if c.BaggageChecked && c.TicketDistributed && c.PassportReturned {
			p.DoneCount++
		} else {
			p.PendingCount++
		}
	}
	return p, nil
}

// ─── notification.Repository ──────────────────────────────────────────────────

type fakeNotifRepo struct {
	fakeErr
	// Sent holds every notification row written, in order — but see gap 3 in
	// harness_test.go: FonnteService.Send returns before writing this row when it
	// has no API token, so nothing sent through the harness ever lands here.
	Sent []notification.WANotification
}

func newFakeNotifRepo() *fakeNotifRepo { return &fakeNotifRepo{} }

func (r *fakeNotifRepo) Seed(n notification.WANotification) { r.Sent = append(r.Sent, n) }

func (r *fakeNotifRepo) Create(_ context.Context, n *notification.WANotification) error {
	if r.err != nil {
		return r.err
	}
	if n.ID == "" {
		n.ID = fmt.Sprintf("notif-%d", len(r.Sent)+1)
	}
	n.CreatedAt = time.Now()
	r.Sent = append(r.Sent, *n)
	return nil
}

func (r *fakeNotifRepo) UpdateStatus(_ context.Context, id, status string, response interface{}) error {
	if r.err != nil {
		return r.err
	}
	for i := range r.Sent {
		if r.Sent[i].ID == id {
			r.Sent[i].Status = status
			r.Sent[i].FonnteResponse = response
			return nil
		}
	}
	return errNotFound
}

func (r *fakeNotifRepo) ListByReference(_ context.Context, refID, refType string) ([]notification.WANotification, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []notification.WANotification{}
	for _, n := range r.Sent {
		if n.ReferenceID != nil && *n.ReferenceID == refID &&
			n.ReferenceType != nil && *n.ReferenceType == refType {
			out = append(out, n)
		}
	}
	return out, nil
}

// ─── chatbot.Repository ───────────────────────────────────────────────────────

type fakeChatbotRepo struct {
	fakeErr
	logs []chatbot.Log
}

func newFakeChatbotRepo() *fakeChatbotRepo { return &fakeChatbotRepo{} }

func (r *fakeChatbotRepo) Seed(l chatbot.Log) { r.logs = append(r.logs, l) }

func (r *fakeChatbotRepo) Create(_ context.Context, l *chatbot.Log) error {
	if r.err != nil {
		return r.err
	}
	if l.ID == "" {
		l.ID = fmt.Sprintf("chat-%d", len(r.logs)+1)
	}
	l.CreatedAt = time.Now()
	r.logs = append(r.logs, *l)
	return nil
}

func (r *fakeChatbotRepo) RecentByPhone(ctx context.Context, phone string, limit int) ([]chatbot.Log, error) {
	all, err := r.ListByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all, nil
}

func (r *fakeChatbotRepo) ListByPhone(_ context.Context, phone string) ([]chatbot.Log, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := []chatbot.Log{}
	for _, l := range r.logs {
		if l.Phone == phone {
			out = append(out, l)
		}
	}
	return out, nil
}

func (r *fakeChatbotRepo) ListConversations(_ context.Context, f chatbot.Filter) ([]chatbot.Conversation, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	byPhone := map[string]*chatbot.Conversation{}
	phones := []string{}
	for _, l := range r.logs {
		if f.Phone != "" && l.Phone != f.Phone {
			continue
		}
		conv, ok := byPhone[l.Phone]
		if !ok {
			conv = &chatbot.Conversation{Phone: l.Phone, FirstChat: l.CreatedAt}
			byPhone[l.Phone] = conv
			phones = append(phones, l.Phone)
		}
		conv.MessageCount++
		conv.LastChat = l.CreatedAt
	}
	out := []chatbot.Conversation{}
	for _, p := range phones {
		out = append(out, *byPhone[p])
	}
	return out, len(out), nil
}
