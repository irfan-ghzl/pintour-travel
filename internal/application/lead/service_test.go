package lead

import (
	"context"
	"errors"
	"testing"
	"time"

	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

// ─── Mock Repositories ────────────────────────────────────────────────────────

type mockLeadRepo struct {
	leads     []domainLead.Lead
	counts    map[string]int // consultantID → active leads
	createErr error
}

func (m *mockLeadRepo) Create(_ context.Context, l *domainLead.Lead) error {
	if m.createErr != nil {
		return m.createErr
	}
	l.ID = "generated-id"
	m.leads = append(m.leads, *l)
	return nil
}
func (m *mockLeadRepo) Update(_ context.Context, _ *domainLead.Lead) error { return nil }
func (m *mockLeadRepo) GetByID(_ context.Context, id string) (*domainLead.Lead, error) {
	for i := range m.leads {
		if m.leads[i].ID == id {
			return &m.leads[i], nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockLeadRepo) List(_ context.Context, _ domainLead.Filter) ([]domainLead.Lead, int, error) {
	return m.leads, len(m.leads), nil
}
func (m *mockLeadRepo) UpdateStatus(_ context.Context, _, _, _ string) error { return nil }
func (m *mockLeadRepo) AssignTo(_ context.Context, _, _ string) error        { return nil }
func (m *mockLeadRepo) MarkConverted(_ context.Context, _ string) error      { return nil }
func (m *mockLeadRepo) CountActiveByConsultant(_ context.Context, cid string) (int, error) {
	return m.counts[cid], nil
}

type mockNoteRepo struct{ notes []domainLead.Note }

func (m *mockNoteRepo) Create(_ context.Context, n *domainLead.Note) error {
	m.notes = append(m.notes, *n)
	return nil
}
func (m *mockNoteRepo) ListByLead(_ context.Context, _ string) ([]domainLead.Note, error) {
	return m.notes, nil
}

type mockUserRepo struct {
	consultants []domainUser.User
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*domainUser.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetByID(_ context.Context, id string) (*domainUser.User, error) {
	for i := range m.consultants {
		if m.consultants[i].ID == id {
			return &m.consultants[i], nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockUserRepo) Create(_ context.Context, _ *domainUser.User) error      { return nil }
func (m *mockUserRepo) Update(_ context.Context, _ *domainUser.User) error      { return nil }
func (m *mockUserRepo) Deactivate(_ context.Context, _ string) error            { return nil }
func (m *mockUserRepo) ListByRole(_ context.Context, _ string) ([]domainUser.User, error) {
	return m.consultants, nil
}
func (m *mockUserRepo) ListKonsultan(_ context.Context) ([]domainUser.User, error) {
	return m.consultants, nil
}
func (m *mockUserRepo) CountActiveleadsByConsultant(_ context.Context, _ string) (int, error) {
	return 0, nil
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestCreateLeadAutoAssignsToConsultantWithFewestLeads(t *testing.T) {
	leadRepo := &mockLeadRepo{
		counts: map[string]int{
			"c1": 10,
			"c2": 3, // <-- least busy
			"c3": 7,
		},
	}
	users := &mockUserRepo{consultants: []domainUser.User{
		{ID: "c1", Name: "Konsultan A", Phone: "628111"},
		{ID: "c2", Name: "Konsultan B", Phone: "628222"},
		{ID: "c3", Name: "Konsultan C", Phone: "628333"},
	}}
	svc := NewService(leadRepo, &mockNoteRepo{}, users, nil) // nil fonnte = no-op

	l := &domainLead.Lead{
		Name: "Budi", Phone: "628123456789",
		PackageID: "pkg1", Pax: 2,
	}
	err := svc.CreateLead(context.Background(), l)
	if err != nil {
		t.Fatalf("CreateLead error: %v", err)
	}

	if l.Status != "baru" {
		t.Errorf("status = %q, want 'baru'", l.Status)
	}
	if l.Source != "organic" {
		t.Errorf("source = %q, want 'organic' (default)", l.Source)
	}
	if l.AssignedTo == nil {
		t.Fatal("AssignedTo must be set")
	}
	if *l.AssignedTo != "c2" {
		t.Errorf("AssignedTo = %q, want 'c2' (least busy)", *l.AssignedTo)
	}

	// Give async goroutine time to complete (fonnte is nil so it's instant)
	time.Sleep(10 * time.Millisecond)
}

func TestCreateLeadNoConsultantsLeavesUnassigned(t *testing.T) {
	svc := NewService(
		&mockLeadRepo{counts: map[string]int{}},
		&mockNoteRepo{},
		&mockUserRepo{consultants: []domainUser.User{}}, // no consultants
		nil,
	)
	l := &domainLead.Lead{Name: "Y", Phone: "628", PackageID: "p1"}
	if err := svc.CreateLead(context.Background(), l); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.AssignedTo != nil {
		t.Errorf("expected nil AssignedTo, got %v", *l.AssignedTo)
	}
}

func TestCreateLeadPreservesExistingSource(t *testing.T) {
	svc := NewService(
		&mockLeadRepo{},
		&mockNoteRepo{},
		&mockUserRepo{},
		nil,
	)
	l := &domainLead.Lead{Name: "X", Phone: "628", PackageID: "p1", Source: "meta_ads"}
	_ = svc.CreateLead(context.Background(), l)
	if l.Source != "meta_ads" {
		t.Errorf("source overwritten: got %q", l.Source)
	}
}

func TestCreateLeadCreateErrorIsReturned(t *testing.T) {
	svc := NewService(
		&mockLeadRepo{createErr: errors.New("db down")},
		&mockNoteRepo{},
		&mockUserRepo{},
		nil,
	)
	err := svc.CreateLead(context.Background(), &domainLead.Lead{Name: "X", Phone: "628", PackageID: "p1"})
	if err == nil {
		t.Fatal("expected error on create failure")
	}
}

func TestCoalesce(t *testing.T) {
	if coalesce("", "x") != "x" {
		t.Error("empty should default")
	}
	if coalesce("a", "x") != "a" {
		t.Error("non-empty should pass through")
	}
}

func TestServiceQueries(t *testing.T) {
	leadRepo := &mockLeadRepo{
		leads: []domainLead.Lead{
			{ID: "l1", Name: "Budi"},
			{ID: "l2", Name: "Siti"},
		},
	}
	svc := NewService(leadRepo, &mockNoteRepo{notes: []domainLead.Note{
		{ID: "n1", LeadID: "l1", Note: "hello"},
	}}, &mockUserRepo{}, nil)

	got, err := svc.GetLead(context.Background(), "l1")
	if err != nil || got.Name != "Budi" {
		t.Errorf("GetLead l1 failed: %v", err)
	}

	leads, total, err := svc.ListLeads(context.Background(), domainLead.Filter{})
	if err != nil || total != 2 || len(leads) != 2 {
		t.Errorf("ListLeads got %d/%d", len(leads), total)
	}

	notes, err := svc.ListNotes(context.Background(), "l1")
	if err != nil || len(notes) != 1 {
		t.Errorf("ListNotes got %d items", len(notes))
	}

	if err := svc.UpdateStatus(context.Background(), "l1", "deal", "user1"); err != nil {
		t.Error("UpdateStatus failed")
	}
	if err := svc.AssignLead(context.Background(), "l1", "c1"); err != nil {
		t.Error("AssignLead failed")
	}
	if err := svc.AddNote(context.Background(), &domainLead.Note{LeadID: "l1", Note: "test"}); err != nil {
		t.Error("AddNote failed")
	}
}
