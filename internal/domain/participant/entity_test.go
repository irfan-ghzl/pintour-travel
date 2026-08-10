package participant

import (
	"testing"
	"time"
)

// §14.4 Participant.ActivatePortal() string
func TestParticipantActivatePortal(t *testing.T) {
	t.Run("mengaktifkan dan mengembalikan identitas login", func(t *testing.T) {
		p := &Participant{ID: "p-1", Phone: "628111", IsActive: false}
		login := p.ActivatePortal()
		if !p.IsActive {
			t.Error("IsActive = false setelah ActivatePortal")
		}
		if login != "628111" {
			t.Errorf("login = %q, want 628111", login)
		}
	})

	t.Run("memanggil dua kali tetap aktif", func(t *testing.T) {
		p := &Participant{ID: "p-1", Phone: "628111"}
		p.ActivatePortal()
		if login := p.ActivatePortal(); login != "628111" || !p.IsActive {
			t.Errorf("pemanggilan kedua = %q, aktif=%v", login, p.IsActive)
		}
	})

	t.Run("peserta tanpa nomor tetap aktif", func(t *testing.T) {
		p := &Participant{ID: "p-1"}
		if login := p.ActivatePortal(); login != "" {
			t.Errorf("login = %q, want kosong", login)
		}
		if !p.IsActive {
			t.Error("IsActive = false padahal aktivasi tidak bergantung pada nomor")
		}
	})
}

// §14.4 Participant.DaysUntilDeparture() int
func TestParticipantDaysUntilDeparture(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	at := func(d time.Time) *time.Time { return &d }

	cases := map[string]struct {
		departure *time.Time
		want      int
	}{
		"dua minggu lagi":         {at(now.AddDate(0, 0, 14)), 14},
		"besok":                   {at(now.AddDate(0, 0, 1)), 1},
		"hari keberangkatan":      {at(now), 0},
		"sudah berangkat kemarin": {at(now.AddDate(0, 0, -1)), -1},
		"belum punya batch":       {nil, 0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := Participant{BatchDepartureDate: tc.departure}
			// The number a participant sees does not depend on what time of day
			// they open the portal: the same trip reads the same all day.
			for _, at := range []time.Time{now, now.Add(-11 * time.Hour), now.Add(11 * time.Hour)} {
				if got := p.DaysUntilDepartureFrom(at); got != tc.want {
					t.Errorf("DaysUntilDepartureFrom(%v) = %d, want %d",
						at.Format(time.Kitchen), got, tc.want)
				}
			}
		})
	}
}

// HasDeparture separates "no batch date" from "zero days left" — the two cases
// DaysUntilDeparture cannot tell apart on its own.
func TestParticipantHasDeparture(t *testing.T) {
	d := time.Now()
	if (&Participant{}).HasDeparture() {
		t.Error("peserta tanpa tanggal keberangkatan dilaporkan punya")
	}
	if !(&Participant{BatchDepartureDate: &d}).HasDeparture() {
		t.Error("peserta dengan tanggal keberangkatan dilaporkan tidak punya")
	}
}
