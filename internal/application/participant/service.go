package participant

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	domainDocument "github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	domainPortalUser "github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
	domainUoW "github.com/irfan-ghzl/pintour-travel/internal/domain/uow"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
	"golang.org/x/crypto/bcrypt"
)

// Service handles participant business logic.
type Service struct {
	participants domainParticipant.Repository
	leads        domainLead.Repository
	portalUsers  domainPortalUser.Repository
	// unit runs the writes that only make sense together as one all-or-nothing
	// unit. The repositories above stay for the single-row operations, which are
	// most of them.
	unit domainUoW.Runner
	// Collaborators for convert-time automation (prompt §1.1 / §1.2).
	batches     domainPkg.BatchRepository
	packages    domainPkg.Repository
	invoices    *invoicesvc.Service
	countryReqs domainDocument.CountryRequirementRepository
	fonnte      *service.FonnteService
}

func NewService(
	participants domainParticipant.Repository,
	leads domainLead.Repository,
	portalUsers domainPortalUser.Repository,
	unit domainUoW.Runner,
	batches domainPkg.BatchRepository,
	packages domainPkg.Repository,
	invoices *invoicesvc.Service,
	countryReqs domainDocument.CountryRequirementRepository,
	fonnte *service.FonnteService,
) *Service {
	return &Service{
		participants: participants,
		leads:        leads,
		portalUsers:  portalUsers,
		unit:         unit,
		batches:      batches,
		packages:     packages,
		invoices:     invoices,
		countryReqs:  countryReqs,
		fonnte:       fonnte,
	}
}

// ConvertResult is the outcome of converting a lead into a participant (v2.0 F1).
type ConvertResult struct {
	Participant *domainParticipant.Participant
	// TempPassword is the freshly generated portal password for a NEW account,
	// empty when an existing returning-customer account was reused.
	TempPassword string
	// ReusedAccount reports that the participant was linked to an existing
	// portal_user (same phone) instead of creating a new login (F1/F4).
	ReusedAccount bool
	// CredentialsSent reports that TempPassword was dispatched to the customer's
	// WhatsApp. False when no gateway is configured — the admin then has to
	// relay it, and needs to be told so rather than assured it was handled.
	CredentialsSent bool
}

// ConvertFromLead creates a participant from a converted lead and triggers the
// convert-time automation: auto-generate invoice (§1.1) and resolve the document
// checklist (§1.2). Automation steps are best-effort — they log on failure but
// do not roll back the participant, which is the primary action.
//
// Returning customers (v2.0 F1): if a portal_user already exists for the lead's
// phone, the new participant is linked to it and the existing password is kept —
// no new account/credential is issued. Otherwise a fresh portal_user is created.
//
// The portal identity, the participant, and the converted lead are written as
// one unit. Written separately, a failure partway left the identity behind
// holding a password hash that had reached nobody; the retry then read that
// orphan as a returning customer, issued no password, and told the admin the old
// one still worked. Since portal accounts have no reset flow, that locked the
// participant out for good — with no error anywhere to say so.
func (s *Service) ConvertFromLead(ctx context.Context, leadID, batchID, roomType, issuedBy, portalBaseURL string) (*ConvertResult, error) {
	// Refused here rather than at the column: the write that would fail is in the
	// middle of the unit, and "tipe kamar tidak dikenal" is a better answer than
	// a rolled-back conversion reporting a constraint violation.
	if !domainParticipant.IsValidRoomType(roomType) {
		return nil, fmt.Errorf("tipe kamar %q tidak dikenal (pilih: %s)",
			roomType, strings.Join(domainParticipant.RoomTypes, ", "))
	}

	l, err := s.leads.GetByID(ctx, leadID)
	if err != nil {
		return nil, fmt.Errorf("lead not found: %w", err)
	}
	if l.Status != "deal" {
		return nil, fmt.Errorf("lead status harus 'deal' untuk dikonversi")
	}

	res := &ConvertResult{}
	var identity portalIdentity

	if err := s.unit.Do(ctx, func(ctx context.Context, repos domainUoW.Repos) error {
		resolved, err := resolvePortalIdentity(ctx, repos.PortalUsers, l)
		if err != nil {
			return err
		}
		identity = resolved

		p := &domainParticipant.Participant{
			LeadID:         &leadID,
			BatchID:        batchID,
			Name:           l.Name,
			Phone:          l.Phone,
			Email:          l.Email,
			RoomType:       roomType,
			PortalPassword: identity.passwordHash, // kept in sync for backward-compat reads
			IsActive:       false,
		}
		if identity.id != "" {
			portalUserID := identity.id
			p.PortalUserID = &portalUserID
		}
		if err := repos.Participants.Create(ctx, p); err != nil {
			return err
		}
		// Recorded in the status trail as deal → peserta by whoever converted it
		// (FR-CRM-02), inside the same unit as the rest of the conversion.
		if err := repos.Leads.MarkConverted(ctx, leadID, issuedBy); err != nil {
			return err
		}
		res.Participant = p
		return nil
	}); err != nil {
		return nil, err
	}

	res.TempPassword = identity.tempPassword
	res.ReusedAccount = identity.reused

	// FR-PORTAL-01: the password reaches the customer without an admin retyping
	// it. Sent only after the unit commits, so no credential is ever announced
	// for a conversion that was rolled back.
	if identity.tempPassword != "" && s.fonnte.Enabled() {
		s.sendPortalCredentials(l, res.Participant.ID, identity.tempPassword, portalBaseURL)
		res.CredentialsSent = true
	}

	// Convert-time automation, all of it outside the unit and best-effort: an
	// invoice that fails to generate is worth a log line and a retry, not
	// throwing away the conversion that succeeded.
	//
	// §1.1 Auto-generate invoice.
	p := res.Participant
	if err := s.autoGenerateInvoice(ctx, l, p, batchID, roomType, issuedBy); err != nil {
		log.Printf("convert[%s]: auto-generate invoice failed: %v", p.ID, err)
	}
	// §1.2 Resolve document checklist. Schema has no 'belum_upload' placeholder
	// rows, so we resolve the required types dynamically and log them; the portal
	// surfaces the same list via country requirements.
	if n, err := s.autoResolveDocumentChecklist(ctx, batchID); err != nil {
		log.Printf("convert[%s]: resolve doc checklist failed: %v", p.ID, err)
	} else {
		log.Printf("convert[%s]: %d dokumen wajib teridentifikasi", p.ID, n)
	}

	return res, nil
}

// portalIdentity is the login a converted participant will use: either the
// account this phone already has, or a new one together with the password that
// has to reach the customer.
type portalIdentity struct {
	id           string
	passwordHash string
	// tempPassword is the plaintext, held only long enough to send it. Empty
	// when an existing account was reused — there is no new password to send,
	// and the old one cannot be recovered from its hash.
	tempPassword string
	reused       bool
}

// resolvePortalIdentity reuses the portal account registered against the lead's
// phone, or creates one. It takes the repository as an argument rather than
// reading the service's, so it runs against whichever one the unit of work
// handed it.
func resolvePortalIdentity(ctx context.Context, repo domainPortalUser.Repository, l *domainLead.Lead) (portalIdentity, error) {
	// No repository means no way to record a login. Refusing is the point: the
	// alternative is a participant created with no portal identity at all, which
	// is the silent half-state the unit around this exists to rule out.
	if repo == nil {
		return portalIdentity{}, fmt.Errorf("portal user repository tidak tersedia")
	}

	existing, err := lookupPortalUser(ctx, repo, l.Phone)
	if err != nil {
		return portalIdentity{}, fmt.Errorf("lookup portal user: %w", err)
	}
	if existing != nil {
		// Returning customer (F1/FR-CRM-08) — reuse the account and its password.
		return portalIdentity{id: existing.ID, passwordHash: existing.PasswordHash, reused: true}, nil
	}

	rawPassword := generatePassword(8)
	hashed, err := bcrypt.GenerateFromPassword([]byte(rawPassword), 12) // §13.2 cost factor 12
	if err != nil {
		return portalIdentity{}, err
	}
	pu := &domainPortalUser.PortalUser{
		Phone:        l.Phone,
		PasswordHash: string(hashed),
		Name:         l.Name,
		Email:        l.Email,
	}
	if err := repo.Create(ctx, pu); err != nil {
		return portalIdentity{}, fmt.Errorf("create portal user: %w", err)
	}
	return portalIdentity{id: pu.ID, passwordHash: string(hashed), tempPassword: rawPassword}, nil
}

// sendPortalCredentials delivers the temporary password over WhatsApp.
//
// Async and best-effort, matching how the invoice is sent: the send retries for
// up to a minute and a half on a slow gateway, which is not something the admin
// clicking "convert" should wait through. A failure leaves the password in the
// convert response, which is why it is still returned to the admin.
func (s *Service) sendPortalCredentials(l *domainLead.Lead, participantID, password, portalBaseURL string) {
	name, phone := l.Name, l.Phone
	safe.Go("kirim kredensial portal", func() {
		if err := s.fonnte.SendPortalCredentials(context.Background(),
			phone, name, password, portalBaseURL+"/portal", participantID); err != nil {
			log.Printf("convert[%s]: kirim kredensial portal gagal: %v", participantID, err)
		}
	})
}

// lookupPortalUser returns the portal identity registered against a phone, or
// (nil, nil) when there is none.
//
// Only "no such row" counts as none. A read that failed for any other reason is
// returned as an error, because the caller's next move on a nil is to create an
// account: a transient failure answered with nil would turn a returning customer
// into a new one, and the only thing standing between that and a duplicate login
// is the unique index on the phone column.
func lookupPortalUser(ctx context.Context, repo domainPortalUser.Repository, phone string) (*domainPortalUser.PortalUser, error) {
	if repo == nil {
		return nil, nil
	}
	pu, err := repo.GetByPhone(ctx, phone)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	}
	return pu, nil
}

// autoGenerateInvoice computes amount = room price × pax and creates the invoice
// (which itself generates the PDF + sends the INVOICE_SENT WA asynchronously).
func (s *Service) autoGenerateInvoice(ctx context.Context, l *domainLead.Lead, p *domainParticipant.Participant, batchID, roomType, issuedBy string) error {
	if s.invoices == nil || s.batches == nil {
		return nil
	}
	batch, err := s.batches.GetByID(ctx, batchID)
	if err != nil {
		return fmt.Errorf("get batch: %w", err)
	}
	price := batch.PriceDouble
	switch strings.ToLower(roomType) {
	case "single":
		price = batch.PriceSingle
	case "triple":
		price = batch.PriceTriple
	}
	pax := l.Pax
	if pax < 1 {
		pax = 1
	}
	inv := &domainInvoice.Invoice{
		ParticipantID: p.ID,
		BatchID:       batchID,
		Amount:        price * float64(pax),
		DueDate:       time.Now().AddDate(0, 0, 7),
		IssuedBy:      issuedBy,
		Notes:         "Invoice otomatis dibuat saat konversi leads.",
	}
	return s.invoices.Create(ctx, inv)
}

// autoResolveDocumentChecklist returns the count of required document types for
// the participant's destination country (§1.2).
func (s *Service) autoResolveDocumentChecklist(ctx context.Context, batchID string) (int, error) {
	if s.batches == nil || s.packages == nil || s.countryReqs == nil {
		return 0, nil
	}
	batch, err := s.batches.GetByID(ctx, batchID)
	if err != nil {
		return 0, err
	}
	pkg, err := s.packages.GetByID(ctx, batch.PackageID)
	if err != nil {
		return 0, err
	}
	reqs, err := s.countryReqs.List(ctx, service.DestinationToCountryCode(pkg.Destination))
	if err != nil {
		return 0, err
	}
	return len(reqs), nil
}

func (s *Service) GetParticipant(ctx context.Context, id string) (*domainParticipant.Participant, error) {
	return s.participants.GetByID(ctx, id)
}

func (s *Service) ListParticipants(ctx context.Context, f domainParticipant.Filter) ([]domainParticipant.Participant, int, error) {
	return s.participants.List(ctx, f)
}

func (s *Service) Activate(ctx context.Context, id string) error {
	return s.participants.Activate(ctx, id)
}

// SetNIK updates the participant NIK (v2.0 F3 — applied from an OCR result).
func (s *Service) SetNIK(ctx context.Context, id, nik string) error {
	return s.participants.SetNIK(ctx, id, nik)
}

// UpdateProfile updates participant name and email (§15.4 portal-profile).
func (s *Service) UpdateProfile(ctx context.Context, p *domainParticipant.Participant) error {
	return s.participants.Update(ctx, p)
}

// PortalLogin authenticates a customer by phone + password against the central
// portal identity (v2.0 F1). It returns the participant used as the default tour
// context — the active tour with the latest departure. Legacy accounts without a
// portal_user row fall back to the per-participant password.
func (s *Service) PortalLogin(ctx context.Context, phone, password string) (*domainParticipant.Participant, error) {
	pu, err := lookupPortalUser(ctx, s.portalUsers, phone)
	if err == nil && pu != nil {
		if bcrypt.CompareHashAndPassword([]byte(pu.PasswordHash), []byte(password)) != nil {
			return nil, fmt.Errorf("nomor WA atau password salah")
		}
		trips, err := s.participants.ListByPortalUser(ctx, pu.ID, phone)
		if err != nil {
			return nil, fmt.Errorf("peserta tidak ditemukan")
		}
		// ListByPortalUser is ordered newest-departure-first; pick the first
		// active tour as the default portal context. PortalUserID is set
		// explicitly because the participant scan does not project it.
		for i := range trips {
			if trips[i].IsActive {
				puID := pu.ID
				trips[i].PortalUserID = &puID
				return &trips[i], nil
			}
		}
		return nil, fmt.Errorf("portal belum aktif, menunggu konfirmasi pembayaran")
	}

	// Fallback: legacy participant-keyed login (pre-F1 accounts).
	p, err := s.participants.GetByPhone(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("peserta tidak ditemukan")
	}
	if !p.IsActive {
		return nil, fmt.Errorf("portal belum aktif, menunggu konfirmasi pembayaran")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(p.PortalPassword), []byte(password)); err != nil {
		return nil, fmt.Errorf("nomor WA atau password salah")
	}
	return p, nil
}

// GetPortalUser returns the central portal identity by ID (v2.0 F1).
func (s *Service) GetPortalUser(ctx context.Context, id string) (*domainPortalUser.PortalUser, error) {
	if s.portalUsers == nil {
		return nil, fmt.Errorf("portal user repository tidak tersedia")
	}
	return s.portalUsers.GetByID(ctx, id)
}

// ListTrips returns every tour (active + history) for a portal identity (F2).
func (s *Service) ListTrips(ctx context.Context, portalUserID, phone string) ([]domainParticipant.Participant, error) {
	return s.participants.ListByPortalUser(ctx, portalUserID, phone)
}

func generatePassword(n int) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"
	result := make([]byte, n)
	for i := range result {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[idx.Int64()]
	}
	return string(result)
}
