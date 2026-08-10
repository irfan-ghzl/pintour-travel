package document

import (
	"errors"
	"testing"
	"time"
)

// §14.4 Document.Approve(reviewerID) error
func TestDocumentApprove(t *testing.T) {
	at := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC)

	t.Run("menyetujui dokumen menunggu", func(t *testing.T) {
		d := &Document{ID: "doc-1", Status: "menunggu"}
		if err := d.Approve("user-1", at); err != nil {
			t.Fatalf("Approve: %v", err)
		}
		if d.Status != "disetujui" {
			t.Errorf("Status = %q, want disetujui", d.Status)
		}
		if d.ReviewedBy == nil || *d.ReviewedBy != "user-1" {
			t.Errorf("ReviewedBy = %v, want user-1", d.ReviewedBy)
		}
		if d.ReviewedAt == nil || !d.ReviewedAt.Equal(at) {
			t.Errorf("ReviewedAt = %v, want %v", d.ReviewedAt, at)
		}
	})

	t.Run("menyetujui dokumen yang sebelumnya ditolak menghapus alasannya", func(t *testing.T) {
		d := &Document{ID: "doc-1", Status: "ditolak", RejectionReason: "buram"}
		if err := d.Approve("user-1", at); err != nil {
			t.Fatalf("Approve: %v", err)
		}
		if d.RejectionReason != "" {
			t.Errorf("RejectionReason = %q, want kosong", d.RejectionReason)
		}
	})

	t.Run("menolak peninjau kosong", func(t *testing.T) {
		d := &Document{ID: "doc-1", Status: "menunggu"}
		if err := d.Approve("", at); err == nil {
			t.Error("Approve tanpa peninjau diterima")
		}
		if d.Status != "menunggu" {
			t.Errorf("Status berubah ke %q meski ditolak", d.Status)
		}
	})
}

// §14.4 Document.Reject(reviewerID, reason) error
func TestDocumentReject(t *testing.T) {
	at := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC)

	t.Run("menolak dokumen dengan alasan", func(t *testing.T) {
		d := &Document{ID: "doc-1", Status: "menunggu"}
		if err := d.Reject("user-1", "foto buram", at); err != nil {
			t.Fatalf("Reject: %v", err)
		}
		if d.Status != "ditolak" {
			t.Errorf("Status = %q, want ditolak", d.Status)
		}
		if d.RejectionReason != "foto buram" {
			t.Errorf("RejectionReason = %q, want 'foto buram'", d.RejectionReason)
		}
		if d.ReviewedBy == nil || *d.ReviewedBy != "user-1" {
			t.Errorf("ReviewedBy = %v, want user-1", d.ReviewedBy)
		}
		if d.ReviewedAt == nil || !d.ReviewedAt.Equal(at) {
			t.Errorf("ReviewedAt = %v, want %v", d.ReviewedAt, at)
		}
	})

	t.Run("alasan wajib diisi", func(t *testing.T) {
		for _, reason := range []string{"", "   ", "\t\n"} {
			d := &Document{ID: "doc-1", Status: "menunggu"}
			if err := d.Reject("user-1", reason, at); !errors.Is(err, ErrRejectionReasonRequired) {
				t.Errorf("Reject(%q): err = %v, want ErrRejectionReasonRequired", reason, err)
			}
			if d.Status != "menunggu" {
				t.Errorf("Status berubah ke %q meski alasan kosong", d.Status)
			}
		}
	})

	t.Run("menolak peninjau kosong", func(t *testing.T) {
		d := &Document{ID: "doc-1", Status: "menunggu"}
		if err := d.Reject("", "foto buram", at); err == nil {
			t.Error("Reject tanpa peninjau diterima")
		}
	})
}
