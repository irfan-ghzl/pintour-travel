package lead

import (
	"errors"
	"testing"
)

// §14.4 Lead.ChangeStatus(newStatus) error
func TestLeadChangeStatus(t *testing.T) {
	cases := map[string]struct {
		from    string
		to      string
		wantErr bool
	}{
		"baru ke dihubungi":       {"baru", "dihubungi", false},
		"dihubungi ke konsultasi": {"dihubungi", "konsultasi", false},
		"konsultasi ke deal":      {"konsultasi", "deal", false},
		"deal ke peserta":         {"deal", "peserta", false},
		"mundur ke baru":          {"deal", "baru", false},
		"status yang sama":        {"baru", "baru", false},
		"status tak dikenal":      {"baru", "expired", true},
		"status kosong":           {"baru", "", true},
		"beda kapitalisasi":       {"baru", "Deal", true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			l := &Lead{ID: "lead-1", Status: tc.from}
			err := l.ChangeStatus(tc.to)
			switch {
			case tc.wantErr && !errors.Is(err, ErrInvalidStatus):
				t.Fatalf("err = %v, want ErrInvalidStatus", err)
			case tc.wantErr:
				if l.Status != tc.from {
					t.Errorf("status berubah ke %q meski ditolak", l.Status)
				}
			case err != nil:
				t.Fatalf("ChangeStatus(%q): %v", tc.to, err)
			case l.Status != tc.to:
				t.Errorf("Status = %q, want %q", l.Status, tc.to)
			}
		})
	}
}

// §14.4 Lead.AssignTo(consultantID) error
func TestLeadAssignTo(t *testing.T) {
	t.Run("menugaskan konsultan", func(t *testing.T) {
		l := &Lead{ID: "lead-1"}
		if err := l.AssignTo("user-7"); err != nil {
			t.Fatalf("AssignTo: %v", err)
		}
		if l.AssignedTo == nil || *l.AssignedTo != "user-7" {
			t.Errorf("AssignedTo = %v, want user-7", l.AssignedTo)
		}
	})

	t.Run("menugaskan ulang menimpa yang lama", func(t *testing.T) {
		old := "user-1"
		l := &Lead{ID: "lead-1", AssignedTo: &old}
		if err := l.AssignTo("user-2"); err != nil {
			t.Fatalf("AssignTo: %v", err)
		}
		if *l.AssignedTo != "user-2" {
			t.Errorf("AssignedTo = %q, want user-2", *l.AssignedTo)
		}
	})

	t.Run("menolak konsultan kosong", func(t *testing.T) {
		l := &Lead{ID: "lead-1"}
		if err := l.AssignTo(""); err == nil {
			t.Error("AssignTo(\"\") = nil, want error")
		}
		if l.AssignedTo != nil {
			t.Errorf("AssignedTo diisi %v meski ditolak", *l.AssignedTo)
		}
	})
}

// §14.4 Lead.ConvertToParticipant() Participant
func TestLeadConvertToParticipant(t *testing.T) {
	t.Run("menyalin identitas lead ke peserta", func(t *testing.T) {
		portalID := "pu-1"
		l := &Lead{
			ID: "lead-1", Name: "Budi", Phone: "628111", Email: "budi@example.com",
			Status: "deal", PortalUserID: &portalID,
		}
		p, err := l.ConvertToParticipant("batch-1", "double")
		if err != nil {
			t.Fatalf("ConvertToParticipant: %v", err)
		}
		if p.LeadID == nil || *p.LeadID != "lead-1" {
			t.Errorf("LeadID = %v, want lead-1", p.LeadID)
		}
		if p.Name != "Budi" || p.Phone != "628111" || p.Email != "budi@example.com" {
			t.Errorf("identitas tidak tersalin: %+v", p)
		}
		if p.BatchID != "batch-1" || p.RoomType != "double" {
			t.Errorf("BatchID/RoomType = %q/%q, want batch-1/double", p.BatchID, p.RoomType)
		}
		if p.PortalUserID == nil || *p.PortalUserID != "pu-1" {
			t.Errorf("PortalUserID = %v, want pu-1", p.PortalUserID)
		}
		// A converted participant is not active until payment is confirmed.
		if p.IsActive {
			t.Error("peserta hasil konversi seharusnya belum aktif")
		}
	})

	t.Run("tanpa akun portal", func(t *testing.T) {
		l := &Lead{ID: "lead-1", Name: "Budi", Phone: "628111", Status: "deal"}
		p, err := l.ConvertToParticipant("batch-1", "single")
		if err != nil {
			t.Fatalf("ConvertToParticipant: %v", err)
		}
		if p.PortalUserID != nil {
			t.Errorf("PortalUserID = %v, want nil", *p.PortalUserID)
		}
	})

	t.Run("menolak tipe kamar tak dikenal", func(t *testing.T) {
		l := &Lead{ID: "lead-1", Status: "deal"}
		if _, err := l.ConvertToParticipant("batch-1", "presidential"); err == nil {
			t.Error("tipe kamar tak dikenal diterima")
		}
	})

	t.Run("menolak lead yang belum deal", func(t *testing.T) {
		for _, status := range []string{"baru", "dihubungi", "konsultasi", "tidak_deal", "peserta"} {
			l := &Lead{ID: "lead-1", Status: status}
			if _, err := l.ConvertToParticipant("batch-1", "double"); !errors.Is(err, ErrNotConvertible) {
				t.Errorf("status %q: err = %v, want ErrNotConvertible", status, err)
			}
		}
	})

	t.Run("menolak batch kosong", func(t *testing.T) {
		l := &Lead{ID: "lead-1", Status: "deal"}
		if _, err := l.ConvertToParticipant("", "double"); err == nil {
			t.Error("batch kosong diterima")
		}
	})
}
