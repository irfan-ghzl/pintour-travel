package pkg

import (
	"errors"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
)

// §14.4 Package.Activate() / Package.Deactivate()
func TestPackageActivateDeactivate(t *testing.T) {
	cases := map[string]struct {
		start bool
		apply func(*Package)
		want  bool
	}{
		"aktifkan paket nonaktif":    {false, (*Package).Activate, true},
		"aktifkan paket sudah aktif": {true, (*Package).Activate, true},
		"nonaktifkan paket aktif":    {true, (*Package).Deactivate, false},
		"nonaktifkan paket nonaktif": {false, (*Package).Deactivate, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := &Package{ID: "pkg-1", IsActive: tc.start}
			tc.apply(p)
			if p.IsActive != tc.want {
				t.Errorf("IsActive = %v, want %v", p.IsActive, tc.want)
			}
		})
	}
}

// §14.4 Package.AddBatch(batch) error
func TestPackageAddBatch(t *testing.T) {
	t.Run("menautkan batch tanpa package_id", func(t *testing.T) {
		p := &Package{ID: "pkg-1"}
		b := &PackageBatch{Quota: 20}
		if err := p.AddBatch(b); err != nil {
			t.Fatalf("AddBatch: %v", err)
		}
		if b.PackageID != "pkg-1" {
			t.Errorf("PackageID = %q, want pkg-1", b.PackageID)
		}
	})

	t.Run("membiarkan package_id yang sudah cocok", func(t *testing.T) {
		p := &Package{ID: "pkg-1"}
		b := &PackageBatch{PackageID: "pkg-1", Quota: 20}
		if err := p.AddBatch(b); err != nil {
			t.Fatalf("AddBatch: %v", err)
		}
		if b.PackageID != "pkg-1" {
			t.Errorf("PackageID = %q, want pkg-1", b.PackageID)
		}
	})

	t.Run("menolak batch milik paket lain", func(t *testing.T) {
		p := &Package{ID: "pkg-1"}
		b := &PackageBatch{PackageID: "pkg-2", Quota: 20}
		err := p.AddBatch(b)
		if !errors.Is(err, ErrBatchPackageMismatch) {
			t.Errorf("err = %v, want ErrBatchPackageMismatch", err)
		}
		if b.PackageID != "pkg-2" {
			t.Errorf("PackageID diubah menjadi %q pada penolakan", b.PackageID)
		}
	})

	t.Run("menolak batch nil", func(t *testing.T) {
		p := &Package{ID: "pkg-1"}
		if err := p.AddBatch(nil); err == nil {
			t.Error("AddBatch(nil) = nil, want error")
		}
	})

	t.Run("menolak paket tanpa id", func(t *testing.T) {
		p := &Package{}
		if err := p.AddBatch(&PackageBatch{}); err == nil {
			t.Error("AddBatch pada paket tanpa id = nil, want error")
		}
	})
}

// §14.4 PackageBatch.HasAvailableSeats() bool
//
// The diagram's method takes no argument; the filled count is not a column
// (§14.4 note) so it is passed in by whoever counted it.
func TestPackageBatchHasAvailableSeats(t *testing.T) {
	cases := map[string]struct {
		quota  int
		sold   int
		status string
		want   bool
	}{
		"masih ada sisa":            {20, 5, "tersedia", true},
		"tersisa satu kursi":        {20, 19, "tersedia", true},
		"kuota persis habis":        {20, 20, "tersedia", false},
		"terjual melebihi kuota":    {20, 21, "tersedia", false},
		"belum ada yang terjual":    {20, 0, "tersedia", true},
		"kuota nol":                 {0, 0, "tersedia", false},
		"kuota negatif":             {-5, 0, "tersedia", false},
		"batch ditandai penuh":      {20, 5, "penuh", false},
		"batch ditutup":             {20, 5, "ditutup", false},
		"status kosong dianggap ok": {20, 5, "", true},
		"terjual negatif":           {20, -3, "tersedia", true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := PackageBatch{Quota: tc.quota, Status: tc.status}
			if got := b.HasAvailableSeats(tc.sold); got != tc.want {
				t.Errorf("HasAvailableSeats(%d) quota=%d status=%q = %v, want %v",
					tc.sold, tc.quota, tc.status, got, tc.want)
			}
		})
	}
}

func TestPackageBatchSeatsRemaining(t *testing.T) {
	cases := map[string]struct {
		quota int
		sold  int
		want  int
	}{
		"sisa normal":            {20, 5, 15},
		"habis":                  {20, 20, 0},
		"terjual melebihi kuota": {20, 25, 0},
		"kuota nol":              {0, 0, 0},
		"kuota negatif":          {-5, 0, 0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := PackageBatch{Quota: tc.quota}
			if got := b.SeatsRemaining(tc.sold); got != tc.want {
				t.Errorf("SeatsRemaining(%d) quota=%d = %d, want %d", tc.sold, tc.quota, got, tc.want)
			}
		})
	}
}

// §14.4 PackageBatch.DaysUntilDeparture() int
func TestPackageBatchDaysUntilDeparture(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	today := calendar.Of(now)

	cases := map[string]struct {
		departure calendar.Date
		want      int
	}{
		"tepat tujuh hari lagi": {today.AddDays(7), 7},
		"besok":                 {today.AddDays(1), 1},
		"hari ini juga":         {today, 0},
		"kemarin":               {today.AddDays(-1), -1},
		"sudah lama berangkat":  {today.AddDays(-30), -30},
		"belum ditetapkan":      {calendar.Date{}, 0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := PackageBatch{DepartureDate: tc.departure}
			// The answer counts whole calendar days, so it does not depend on
			// what time of day it is asked — the number a participant sees does
			// not tick over at an arbitrary hour.
			for _, at := range []time.Time{now, now.Add(-11 * time.Hour), now.Add(11 * time.Hour)} {
				if got := b.DaysUntilDepartureFrom(at); got != tc.want {
					t.Errorf("DaysUntilDepartureFrom(%v) = %d, want %d",
						at.Format(time.Kitchen), got, tc.want)
				}
			}
		})
	}
}
