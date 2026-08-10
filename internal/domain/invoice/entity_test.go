package invoice

import (
	"errors"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
)

// §14.4 Invoice.ConfirmPayment(confirmedBy) error
func TestInvoiceConfirmPayment(t *testing.T) {
	at := time.Date(2026, 8, 9, 10, 30, 0, 0, time.UTC)

	t.Run("menandai lunas beserta pelakunya", func(t *testing.T) {
		inv := &Invoice{ID: "inv-1", Status: "menunggu_bayar"}
		if err := inv.ConfirmPayment("user-1", at); err != nil {
			t.Fatalf("ConfirmPayment: %v", err)
		}
		if inv.Status != "lunas" {
			t.Errorf("Status = %q, want lunas", inv.Status)
		}
		if inv.ConfirmedBy == nil || *inv.ConfirmedBy != "user-1" {
			t.Errorf("ConfirmedBy = %v, want user-1", inv.ConfirmedBy)
		}
		if inv.ConfirmedAt == nil || !inv.ConfirmedAt.Equal(at) {
			t.Errorf("ConfirmedAt = %v, want %v", inv.ConfirmedAt, at)
		}
	})

	t.Run("invoice yang sudah lunas ditolak sebagai sudah selesai", func(t *testing.T) {
		earlier := at.Add(-time.Hour)
		by := "user-1"
		inv := &Invoice{ID: "inv-1", Status: "lunas", ConfirmedBy: &by, ConfirmedAt: &earlier}
		err := inv.ConfirmPayment("user-2", at)
		if !errors.Is(err, ErrAlreadySettled) {
			t.Fatalf("err = %v, want ErrAlreadySettled", err)
		}
		if *inv.ConfirmedBy != "user-1" || !inv.ConfirmedAt.Equal(earlier) {
			t.Error("konfirmasi pertama tertimpa oleh konfirmasi kedua")
		}
	})

	t.Run("menolak pelaku kosong", func(t *testing.T) {
		inv := &Invoice{ID: "inv-1", Status: "menunggu_bayar"}
		if err := inv.ConfirmPayment("", at); err == nil {
			t.Error("ConfirmPayment tanpa pelaku diterima")
		}
		if inv.Status == "lunas" {
			t.Error("status berubah lunas meski konfirmasi ditolak")
		}
	})
}

// §14.4 Invoice.IsOverdue() bool
func TestInvoiceIsOverdue(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	today := calendar.Of(now)

	cases := map[string]struct {
		status string
		due    calendar.Date
		want   bool
	}{
		"diterbitkan dan lewat tempo":      {"diterbitkan", today.AddDays(-1), true},
		"menunggu bayar dan lewat tempo":   {"menunggu_bayar", today.AddDays(-3), true},
		"dibayar sebagian dan lewat tempo": {"dibayar", today.AddDays(-3), true},
		"lunas meski lewat tempo":          {"lunas", today.AddDays(-30), false},
		"belum jatuh tempo":                {"menunggu_bayar", today.AddDays(1), false},
		// A due date is a whole day: an invoice due today still has today to be
		// paid, at any hour of it.
		"jatuh tempo hari ini":        {"menunggu_bayar", today, false},
		"menunggu konfirmasi gateway": {"menunggu_konfirmasi_gateway", today.AddDays(-1), false},
		"tanpa tanggal jatuh tempo":   {"menunggu_bayar", calendar.Date{}, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inv := Invoice{Status: tc.status, DueDate: tc.due}
			for _, at := range []time.Time{now, now.Add(-11 * time.Hour), now.Add(11 * time.Hour)} {
				if got := inv.IsOverdueAt(at); got != tc.want {
					t.Errorf("IsOverdueAt(%v) status=%q due=%v = %v, want %v",
						at.Format(time.Kitchen), tc.status, tc.due, got, tc.want)
				}
			}
		})
	}
}

func TestInvoiceIsOverdueSkipsDeleted(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	deleted := now.AddDate(0, 0, -1)
	inv := Invoice{
		Status:    "menunggu_bayar",
		DueDate:   calendar.Of(now).AddDays(-5),
		DeletedAt: &deleted,
	}
	if inv.IsOverdueAt(now) {
		t.Error("invoice terhapus lunak dilaporkan jatuh tempo")
	}
}

func TestInvoiceRemainingBalance(t *testing.T) {
	cases := map[string]struct {
		amount float64
		paid   float64
		want   float64
	}{
		"belum dibayar sama sekali": {5_000_000, 0, 5_000_000},
		"dibayar sebagian":          {5_000_000, 2_000_000, 3_000_000},
		"dibayar pas":               {5_000_000, 5_000_000, 0},
		"lebih bayar":               {5_000_000, 6_000_000, 0},
		"nominal nol":               {0, 0, 0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inv := Invoice{Amount: tc.amount}
			if got := inv.RemainingBalance(tc.paid); got != tc.want {
				t.Errorf("RemainingBalance(%v) amount=%v = %v, want %v", tc.paid, tc.amount, got, tc.want)
			}
		})
	}
}

func TestInvoiceIsFullyPaid(t *testing.T) {
	cases := map[string]struct {
		amount float64
		paid   float64
		want   bool
	}{
		"kurang satu rupiah": {5_000_000, 4_999_999, false},
		"pas":                {5_000_000, 5_000_000, true},
		"lebih":              {5_000_000, 5_000_001, true},
		"belum bayar":        {5_000_000, 0, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inv := Invoice{Amount: tc.amount}
			if got := inv.IsFullyPaid(tc.paid); got != tc.want {
				t.Errorf("IsFullyPaid(%v) amount=%v = %v, want %v", tc.paid, tc.amount, got, tc.want)
			}
		})
	}
}
