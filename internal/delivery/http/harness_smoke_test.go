package httpdelivery

// Checks on the harness's own affordances. If these break, every test built on
// the seam is reading something other than what it thinks it is.

import (
	"errors"
	"net/http"
	"testing"
	"time"

	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
)

// Seeded state must be what the request sees, and what the request writes must
// be readable back off the fake. This one seeds its own single lead rather than
// calling seedBaseline, so the count below means something.
func TestHarness_SeededStateFlowsThroughTheRequest(t *testing.T) {
	h := newHarness(t)
	consultant := "user-konsultan"
	h.Leads.Seed(domainLead.Lead{
		ID: "lead-1", Name: "Prospek Satu", Phone: "628111111111",
		PackageID: "package-1", Status: "baru", AssignedTo: &consultant,
		CreatedAt: time.Now(),
	})

	var listed struct {
		Data []domainLead.Lead `json:"data"`
	}
	h.as("admin").GET("/api/v1/admin/leads").expectCode(http.StatusOK).decode(&listed)
	if len(listed.Data) != 1 || listed.Data[0].Name != "Prospek Satu" {
		t.Fatalf("listed leads = %+v, want the one seeded lead", listed.Data)
	}

	h.as("admin").PATCH("/api/v1/admin/leads/lead-1/status",
		map[string]string{"status": "dihubungi"}).expectCode(http.StatusOK)

	if len(h.Leads.StatusChanges) != 1 {
		t.Fatalf("recorded %d status changes, want 1", len(h.Leads.StatusChanges))
	}
	if got := h.Leads.StatusChanges[0].To; got != "dihubungi" {
		t.Errorf("new status = %q, want dihubungi", got)
	}
}

// Fail lets a test drive the handler paths that only a broken database reaches —
// the difference between "not found" and "try again later".
func TestHarness_FailDrivesRepositoryErrorPaths(t *testing.T) {
	h := newHarness(t)
	h.Invoices.Fail(errors.New("connection refused"))

	h.as("admin").GET("/api/v1/admin/invoices").expectCode(http.StatusInternalServerError)
}
