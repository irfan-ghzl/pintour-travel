package lead

import (
	"context"
	"fmt"

	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// Service handles lead business logic.
type Service struct {
	leads    domainLead.Repository
	notes    domainLead.NoteRepository
	users    domainUser.Repository
	fonnte   *service.FonnteService
}

func NewService(
	leads domainLead.Repository,
	notes domainLead.NoteRepository,
	users domainUser.Repository,
	fonnte *service.FonnteService,
) *Service {
	return &Service{leads: leads, notes: notes, users: users, fonnte: fonnte}
}

// CreateLead creates a new lead, auto-assigns to consultant, and sends WA notifications.
func (s *Service) CreateLead(ctx context.Context, l *domainLead.Lead) error {
	l.Status = "baru"
	l.Source = coalesce(l.Source, "organic")

	// Auto-assign to consultant with fewest active leads
	consultants, err := s.users.ListKonsultan(ctx)
	if err == nil && len(consultants) > 0 {
		assigned := consultants[0]
		minLoad, _ := s.leads.CountActiveByConsultant(ctx, assigned.ID)
		for _, c := range consultants[1:] {
			n, _ := s.leads.CountActiveByConsultant(ctx, c.ID)
			if n < minLoad {
				minLoad = n
				assigned = c
			}
		}
		l.AssignedTo = &assigned.ID
	}

	if err := s.leads.Create(ctx, l); err != nil {
		return fmt.Errorf("create lead: %w", err)
	}

	// Async WA notifications (best-effort, don't block response)
	go func() {
		bgCtx := context.Background()
		_ = s.fonnte.SendLeadsWelcome(bgCtx, l.Phone, l.Name, l.PackageName, l.ID)

		if l.AssignedTo != nil {
			consultant, err := s.users.GetByID(bgCtx, *l.AssignedTo)
			if err == nil && consultant.Phone != "" {
				_ = s.fonnte.SendLeadsNotifSales(bgCtx,
					consultant.Phone, consultant.Name,
					l.Name, l.Phone, l.PackageName, l.ID)
			}
		}
	}()

	return nil
}

func (s *Service) GetLead(ctx context.Context, id string) (*domainLead.Lead, error) {
	return s.leads.GetByID(ctx, id)
}

func (s *Service) ListLeads(ctx context.Context, f domainLead.Filter) ([]domainLead.Lead, int, error) {
	return s.leads.List(ctx, f)
}

func (s *Service) UpdateStatus(ctx context.Context, id, status, changedBy string) error {
	return s.leads.UpdateStatus(ctx, id, status, changedBy)
}

func (s *Service) AssignLead(ctx context.Context, leadID, consultantID string) error {
	return s.leads.AssignTo(ctx, leadID, consultantID)
}

func (s *Service) AddNote(ctx context.Context, n *domainLead.Note) error {
	return s.notes.Create(ctx, n)
}

func (s *Service) ListNotes(ctx context.Context, leadID string) ([]domainLead.Note, error) {
	return s.notes.ListByLead(ctx, leadID)
}

func coalesce(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
