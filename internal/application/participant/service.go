package participant

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"golang.org/x/crypto/bcrypt"
)

// Service handles participant business logic.
type Service struct {
	participants domainParticipant.Repository
	leads        domainLead.Repository
}

func NewService(participants domainParticipant.Repository, leads domainLead.Repository) *Service {
	return &Service{participants: participants, leads: leads}
}

// ConvertFromLead creates a participant from a converted lead.
func (s *Service) ConvertFromLead(ctx context.Context, leadID, batchID, roomType string) (*domainParticipant.Participant, string, error) {
	l, err := s.leads.GetByID(ctx, leadID)
	if err != nil {
		return nil, "", fmt.Errorf("lead not found: %w", err)
	}
	if l.Status != "deal" {
		return nil, "", fmt.Errorf("lead status harus 'deal' untuk dikonversi")
	}

	rawPassword := generatePassword(8)
	hashed, err := bcrypt.GenerateFromPassword([]byte(rawPassword), 12) // §13.2 cost factor 12
	if err != nil {
		return nil, "", err
	}

	p := &domainParticipant.Participant{
		LeadID:         &leadID,
		BatchID:        batchID,
		Name:           l.Name,
		Phone:          l.Phone,
		Email:          l.Email,
		RoomType:       roomType,
		PortalPassword: string(hashed),
		IsActive:       false,
	}

	if err := s.participants.Create(ctx, p); err != nil {
		return nil, "", err
	}

	if err := s.leads.MarkConverted(ctx, leadID); err != nil {
		return nil, "", err
	}

	return p, rawPassword, nil
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

// UpdateProfile updates participant name and email (§15.4 portal-profile).
func (s *Service) UpdateProfile(ctx context.Context, p *domainParticipant.Participant) error {
	return s.participants.Update(ctx, p)
}

// PortalLogin authenticates a participant by phone + password.
func (s *Service) PortalLogin(ctx context.Context, phone, password string) (*domainParticipant.Participant, error) {
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

func generatePassword(n int) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"
	result := make([]byte, n)
	for i := range result {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[idx.Int64()]
	}
	return string(result)
}
