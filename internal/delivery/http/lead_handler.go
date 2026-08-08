package httpdelivery

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	leadsvc "github.com/irfan-ghzl/pintour-travel/internal/application/lead"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
)

type LeadHandler struct {
	svc       *leadsvc.Service
	notifRepo notification.Repository
}

func NewLeadHandler(svc *leadsvc.Service, notifRepo notification.Repository) *LeadHandler {
	return &LeadHandler{svc: svc, notifRepo: notifRepo}
}

// CreateLead godoc
// @Summary Submit form leads dari landing page
// @Tags leads
// @Router /leads [post]
func (h *LeadHandler) CreateLead(c echo.Context) error {
	var l domainLead.Lead
	if err := bindJSON(c, &l); err != nil {
		return invalidPayload(c, err, "nama, nomor WA, dan paket harus diisi")
	}
	// The tag accepts 08…, 62…, and +62…; everything downstream expects the 62…
	// form, so normalise once here.
	l.Phone = normalizePhone(l.Phone)
	// The column defaults to 1, but the repository writes this field explicitly,
	// so an omitted pax would reach Postgres as 0 and break its CHECK.
	if l.Pax == 0 {
		l.Pax = 1
	}

	// F4: returning-customer status is decided server-side by phone, never trusted
	// from the client (the prefilled portal form only carries name/phone/email).
	l.PortalUserID = nil
	l.IsReturning = false

	if err := h.svc.CreateLead(c.Request().Context(), &l); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(map[string]interface{}{
		"id":      l.ID,
		"message": "Terima kasih! Tim konsultan kami akan segera menghubungi Anda.",
	}))
}

// ListLeads godoc
// @Summary      Daftar leads (admin/konsultan)
// @Tags         leads
// @Security     BearerAuth
// @Produce      json
// @Param        status query string false "Filter status"
// @Param        assigned_to query string false "Konsultan ID"
// @Param        package_id query string false "Paket ID"
// @Param        search query string false "Cari nama atau WA"
// @Param        date_from query string false "Dari tanggal (YYYY-MM-DD)"
// @Param        date_to query string false "Sampai tanggal (YYYY-MM-DD)"
// @Param        page query int false "Halaman" default(1)
// @Success      200 {object} map[string]interface{}
// @Router       /admin/leads [get]
func (h *LeadHandler) ListLeads(c echo.Context) error {
	f := domainLead.Filter{
		Page:    queryInt(c, "page", 1),
		PerPage: queryPageSize(c, "per_page", 20),
	}
	if v := c.QueryParam("status"); v != "" { f.Status = &v }
	if v := c.QueryParam("assigned_to"); v != "" { f.AssignedTo = &v }
	if v := c.QueryParam("package_id"); v != "" { f.PackageID = &v }
	if v := c.QueryParam("search"); v != "" { f.Search = &v }
	// FR-CRM-05: filter rentang tanggal masuk
	if v := c.QueryParam("date_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			f.DateFrom = &t
		}
	}
	if v := c.QueryParam("date_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(24 * time.Hour)
			f.DateTo = &end
		}
	}

	// Konsultan hanya lihat leads milik sendiri
	role := claimRole(c)
	if role == "konsultan" {
		uid := claimUserID(c)
		f.AssignedTo = &uid
	}

	leads, total, err := h.svc.ListLeads(c.Request().Context(), f)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, pageResponse(leads, total, f.Page, f.PerPage))
}

// GetLead godoc
// @Summary      Detail lead + activity log + WA log
// @Tags         leads
// @Security     BearerAuth
// @Param        id path string true "Lead ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/leads/{id} [get]
func (h *LeadHandler) GetLead(c echo.Context) error {
	l, err := h.svc.GetLead(c.Request().Context(), c.Param("id"))
	if err != nil {
		return notFound(c, "leads tidak ditemukan")
	}
	notes, _ := h.svc.ListNotes(c.Request().Context(), l.ID)

	// FR-CRM-03: include WA notification history linked to this lead
	var waLog []notification.WANotification
	if h.notifRepo != nil {
		waLog, _ = h.notifRepo.ListByReference(c.Request().Context(), l.ID, "lead")
	}

	// F4: returning customer — surface previous tours in the lead detail panel.
	var previousTrips interface{}
	if l.IsReturning {
		if trips, err := h.svc.PreviousTrips(c.Request().Context(), l); err == nil {
			previousTrips = trips
		}
	}

	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"lead":           l,
		"activity_log":   notes,
		"wa_log":         waLog,
		"previous_trips": previousTrips,
	}))
}

// UpdateStatus godoc
// @Summary      Ubah status lead (auto-record activity)
// @Tags         leads
// @Security     BearerAuth
// @Param        id path string true "Lead ID"
// @Param        body body map[string]string true "{status: ...}"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/leads/{id}/status [patch]
func (h *LeadHandler) UpdateStatus(c echo.Context) error {
	var body struct {
		Status string `json:"status" validate:"required,lead_status"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "status harus diisi")
	}

	changedBy := claimUserID(c)
	if err := h.svc.UpdateStatus(c.Request().Context(), c.Param("id"), body.Status, changedBy); err != nil {
		if errors.Is(err, domainLead.ErrInvalidStatus) {
			return badRequest(c, err.Error())
		}
		return serverErr(c, err)
	}

	// Auto-log status change as a note
	note := &domainLead.Note{
		LeadID:    c.Param("id"),
		UserID:    changedBy,
		Note:      fmt.Sprintf("[SISTEM] Status diubah menjadi '%s' pada %s", body.Status, time.Now().Format("02 Jan 2006 15:04")),
	}
	_ = h.svc.AddNote(c.Request().Context(), note)

	return c.JSON(http.StatusOK, ok(map[string]string{"status": body.Status}))
}

// AssignLead godoc
// @Summary      Assign lead ke konsultan (manual)
// @Tags         leads
// @Security     BearerAuth
// @Param        id path string true "Lead ID"
// @Accept       json
// @Success      200 {object} map[string]interface{}
// @Router       /admin/leads/{id}/assign [patch]
func (h *LeadHandler) AssignLead(c echo.Context) error {
	var body struct {
		ConsultantID string `json:"consultant_id" validate:"required"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "consultant_id harus diisi")
	}

	changedBy := claimUserID(c)
	if err := h.svc.AssignLead(c.Request().Context(), c.Param("id"), body.ConsultantID); err != nil {
		return serverErr(c, err)
	}

	// Auto-log assignment
	note := &domainLead.Note{
		LeadID: c.Param("id"),
		UserID: changedBy,
		Note:   fmt.Sprintf("[SISTEM] Leads di-assign pada %s", time.Now().Format("02 Jan 2006 15:04")),
	}
	_ = h.svc.AddNote(c.Request().Context(), note)

	return c.JSON(http.StatusOK, ok(nil))
}

// AddNote godoc
// @Summary      Tambah catatan konsultan ke lead (FR-CRM-04)
// @Tags         leads
// @Security     BearerAuth
// @Param        id path string true "Lead ID"
// @Accept       json
// @Success      201 {object} map[string]interface{}
// @Router       /admin/leads/{id}/notes [post]
func (h *LeadHandler) AddNote(c echo.Context) error {
	var note domainLead.Note
	if err := bindJSON(c, &note); err != nil {
		return invalidPayload(c, err, "catatan tidak boleh kosong")
	}
	note.LeadID = c.Param("id")
	note.UserID = claimUserID(c)
	if err := h.svc.AddNote(c.Request().Context(), &note); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(note))
}
