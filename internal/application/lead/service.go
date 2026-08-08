package lead

import (
	"context"
	"fmt"
	"os"

	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	domainPortalUser "github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// Service handles lead business logic.
type Service struct {
	leads        domainLead.Repository
	notes        domainLead.NoteRepository
	users        domainUser.Repository
	portalUsers  domainPortalUser.Repository
	participants domainParticipant.Repository
	fonnte       *service.FonnteService
	email        *service.EmailService
}

func NewService(
	leads domainLead.Repository,
	notes domainLead.NoteRepository,
	users domainUser.Repository,
	portalUsers domainPortalUser.Repository,
	participants domainParticipant.Repository,
	fonnte *service.FonnteService,
	email *service.EmailService,
) *Service {
	return &Service{
		leads:        leads,
		notes:        notes,
		users:        users,
		portalUsers:  portalUsers,
		participants: participants,
		fonnte:       fonnte,
		email:        email,
	}
}

// CreateLead creates a new lead, auto-assigns to consultant, and sends WA notifications.
func (s *Service) CreateLead(ctx context.Context, l *domainLead.Lead) error {
	l.Status = "baru"
	l.Source = coalesce(l.Source, "organic")

	// F4: flag returning customers so the CRM can show a "Pelanggan Lama" badge.
	// Trust an explicit portal_user_id from the portal-prefilled form, otherwise
	// detect by phone.
	if l.PortalUserID == nil && s.portalUsers != nil {
		if pu, err := s.portalUsers.GetByPhone(ctx, l.Phone); err == nil && pu != nil {
			l.PortalUserID = &pu.ID
		}
	}
	l.IsReturning = l.PortalUserID != nil

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
	safe.Go("notifikasi lead baru", func() {
		bgCtx := context.Background()
		// Fetch full lead to get PackageName (Create only returns id/timestamps)
		full, err := s.leads.GetByID(bgCtx, l.ID)
		if err != nil {
			full = l // fallback to partial data
		}
		_ = s.fonnte.SendLeadsWelcome(bgCtx, full.Phone, full.Name, full.PackageName, full.ID)

		if full.AssignedTo != nil {
			consultant, err := s.users.GetByID(bgCtx, *full.AssignedTo)
			if err == nil && consultant.Phone != "" {
				_ = s.fonnte.SendLeadsNotifSales(bgCtx,
					consultant.Phone, consultant.Name,
					full.Name, full.Phone, full.PackageName, full.ID)
			}
		}

		// §3.3 notify all admins of the new lead via email.
		if s.email != nil {
			appURL := os.Getenv("APP_URL")
			if appURL == "" {
				appURL = "http://localhost:5173"
			}
			if admins, err := s.users.ListByRole(bgCtx, "admin"); err == nil {
				for _, a := range admins {
					if a.Email == "" {
						continue
					}
					_ = s.email.SendEmailAdminNewLeads(bgCtx, a.Email,
						l.Name, l.Phone, l.Email, l.PackageName, appURL+"/admin/leads")
				}
			}
		}
	})

	return nil
}

func (s *Service) GetLead(ctx context.Context, id string) (*domainLead.Lead, error) {
	return s.leads.GetByID(ctx, id)
}

// PreviousTrips returns a returning customer's past tours for the CRM lead-detail
// panel (F4). Returns nil when the phone has no portal account / prior tour.
func (s *Service) PreviousTrips(ctx context.Context, l *domainLead.Lead) ([]domainParticipant.Participant, error) {
	if s.participants == nil {
		return nil, nil
	}
	puID := ""
	if l.PortalUserID != nil {
		puID = *l.PortalUserID
	}
	return s.participants.ListByPortalUser(ctx, puID, l.Phone)
}

func (s *Service) ListLeads(ctx context.Context, f domainLead.Filter) ([]domainLead.Lead, int, error) {
	return s.leads.List(ctx, f)
}

func (s *Service) UpdateStatus(ctx context.Context, id, status, changedBy string) error {
	if !domainLead.IsValidStatus(status) {
		return domainLead.ErrInvalidStatus
	}
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
