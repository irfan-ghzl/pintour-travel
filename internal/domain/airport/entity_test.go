package airport

import (
	"testing"
	"time"
)

// §14.4 AirportChecklist.IsComplete() bool
func TestChecklistIsComplete(t *testing.T) {
	cases := map[string]struct {
		baggage  bool
		ticket   bool
		passport bool
		want     bool
	}{
		"ketiganya selesai":     {true, true, true, true},
		"bagasi belum":          {false, true, true, false},
		"tiket belum":           {true, false, true, false},
		"paspor belum":          {true, true, false, false},
		"baru bagasi":           {true, false, false, false},
		"belum satu pun":        {false, false, false, false},
		"tiket dan paspor saja": {false, true, true, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := Checklist{
				BaggageChecked:    tc.baggage,
				TicketDistributed: tc.ticket,
				PassportReturned:  tc.passport,
			}
			if got := c.IsComplete(); got != tc.want {
				t.Errorf("IsComplete() = %v, want %v", got, tc.want)
			}
		})
	}
}

// §14.4 AirportChecklist.MarkBaggage(by) / MarkTicket(by) / MarkPassport(by)
func TestChecklistMarkSteps(t *testing.T) {
	at := time.Date(2026, 8, 9, 4, 30, 0, 0, time.UTC)

	steps := map[string]struct {
		mark  func(*Checklist, string, time.Time)
		done  func(Checklist) bool
		stamp func(Checklist) *time.Time
	}{
		"bagasi": {
			(*Checklist).MarkBaggage,
			func(c Checklist) bool { return c.BaggageChecked },
			func(c Checklist) *time.Time { return c.BaggageCheckedAt },
		},
		"tiket": {
			(*Checklist).MarkTicket,
			func(c Checklist) bool { return c.TicketDistributed },
			func(c Checklist) *time.Time { return c.TicketDistributedAt },
		},
		"paspor": {
			(*Checklist).MarkPassport,
			func(c Checklist) bool { return c.PassportReturned },
			func(c Checklist) *time.Time { return c.PassportReturnedAt },
		},
	}

	for name, step := range steps {
		t.Run(name+" ditandai beserta waktu dan petugasnya", func(t *testing.T) {
			c := &Checklist{ID: "cl-1"}
			step.mark(c, "user-9", at)
			if !step.done(*c) {
				t.Error("langkah tidak ditandai selesai")
			}
			if got := step.stamp(*c); got == nil || !got.Equal(at) {
				t.Errorf("waktu = %v, want %v", got, at)
			}
			if c.HandledBy == nil || *c.HandledBy != "user-9" {
				t.Errorf("HandledBy = %v, want user-9", c.HandledBy)
			}
			if !c.UpdatedAt.Equal(at) {
				t.Errorf("UpdatedAt = %v, want %v", c.UpdatedAt, at)
			}
		})

		t.Run(name+" ditandai ulang mempertahankan waktu pertama", func(t *testing.T) {
			c := &Checklist{ID: "cl-1"}
			step.mark(c, "user-9", at)
			later := at.Add(time.Hour)
			step.mark(c, "user-8", later)
			if got := step.stamp(*c); got == nil || !got.Equal(at) {
				t.Errorf("waktu = %v, want tetap %v", got, at)
			}
			if c.HandledBy == nil || *c.HandledBy != "user-8" {
				t.Errorf("HandledBy = %v, want user-8 (petugas terakhir)", c.HandledBy)
			}
		})
	}

	t.Run("tiga langkah membuat checklist lengkap", func(t *testing.T) {
		c := &Checklist{ID: "cl-1"}
		c.MarkBaggage("user-9", at)
		c.MarkTicket("user-9", at)
		if c.IsComplete() {
			t.Error("IsComplete() true padahal paspor belum")
		}
		c.MarkPassport("user-9", at)
		if !c.IsComplete() {
			t.Error("IsComplete() false setelah ketiga langkah")
		}
	})
}
