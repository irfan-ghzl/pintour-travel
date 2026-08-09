package httpdelivery

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	pkgsvc "github.com/irfan-ghzl/pintour-travel/internal/application/package"
	participantsvc "github.com/irfan-ghzl/pintour-travel/internal/application/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/config"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/privacy"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/irfan-ghzl/pintour-travel/internal/format"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// PortalHandler handles participant portal endpoints.
type PortalHandler struct {
	participants *participantsvc.Service
	invoices     *invoicesvc.Service
	packages     *pkgsvc.Service
	docs         document.Repository
	countryReqs  document.CountryRequirementRepository
	tourLeaders  domainUser.TourLeaderRepository
	users        domainUser.Repository
	pdf          *service.PDFService
	email        *service.EmailService
	// TODO(ocr-v2.0-F3): re-enable when GCP Vision billing active
	ocr       *service.OCRService
	deletions privacy.Repository
	jwtSecret string
}

func NewPortalHandler(
	participants *participantsvc.Service,
	invoices *invoicesvc.Service,
	packages *pkgsvc.Service,
	docs document.Repository,
	countryReqs document.CountryRequirementRepository,
	tourLeaders domainUser.TourLeaderRepository,
	users domainUser.Repository,
	pdf *service.PDFService,
	email *service.EmailService,
	ocr *service.OCRService,
	deletions privacy.Repository,
	jwtSecret string,
) *PortalHandler {
	return &PortalHandler{
		participants: participants,
		invoices:     invoices,
		packages:     packages,
		docs:         docs,
		countryReqs:  countryReqs,
		tourLeaders:  tourLeaders,
		users:        users,
		pdf:          pdf,
		email:        email,
		ocr:          ocr,
		deletions:    deletions,
		jwtSecret:    jwtSecret,
	}
}

// notifyAdmins sends an email to every admin user (best-effort, async-safe).
func (h *PortalHandler) notifyAdmins(ctx context.Context, send func(adminEmail string)) {
	if h.email == nil || h.users == nil {
		return
	}
	for _, a := range domainUser.ListAdmins(ctx, h.users) {
		if a.Email != "" {
			send(a.Email)
		}
	}
}

// PortalLogin godoc
// @Summary      Login peserta ke portal (FR-PORTAL-01)
// @Tags         portal
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /portal/login [post]
func (h *PortalHandler) PortalLogin(c echo.Context) error {
	var body struct {
		Phone    string `json:"phone" validate:"required,phone_id"`
		Password string `json:"password" validate:"required"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "nomor WA dan password harus diisi")
	}
	// Normalize phone
	phone := normalizePhone(body.Phone)

	p, err := h.participants.PortalLogin(c.Request().Context(), phone, body.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errResponse("UNAUTHORIZED", err.Error()))
	}

	token, err := h.generatePortalToken(p)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"token":          token,
		"participant_id": p.ID,
		"name":           p.Name,
		"package_name":   p.PackageName,
	}))
}

// PortalMe godoc
// @Summary      Info peserta saat ini + countdown + briefing active (FR-PORTAL-02)
// @Tags         portal
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /portal/me [get]
func (h *PortalHandler) PortalMe(c echo.Context) error {
	pid := portalParticipantID(c)
	p, err := h.participants.GetParticipant(c.Request().Context(), pid)
	if err != nil {
		return notFound(c, "peserta tidak ditemukan")
	}
	var countdown *int
	if p.HasDeparture() {
		days := p.DaysUntilDeparture() // §14.4 Participant.DaysUntilDeparture
		countdown = &days
	}
	resp := map[string]interface{}{
		"participant":     p,
		"days_to_depart":  countdown,
		"briefing_active": isBriefingActive(p),
	}
	// FR-PORTAL-11: warn when the OCR-read passport expires within 6 months.
	if h.ocr != nil {
		if expiry, soon := h.ocr.PassportExpiry(c.Request().Context(), pid); expiry != "" {
			resp["passport_expiry"] = expiry
			resp["passport_expiring_soon"] = soon
		}
	}
	return c.JSON(http.StatusOK, ok(resp))
}

// PortalMyTrips godoc
// @Summary      Riwayat perjalanan peserta — tour aktif + lampau (v2.0 F2)
// @Tags         portal
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /portal/my-trips [get]
func (h *PortalHandler) PortalMyTrips(c echo.Context) error {
	ctx := c.Request().Context()
	puID := portalUserID(c)

	// Resolve the phone fallback so legacy (unlinked) tours are still included.
	phone := ""
	if puID != "" {
		if pu, err := h.participants.GetPortalUser(ctx, puID); err == nil {
			phone = pu.Phone
		}
	}
	if phone == "" {
		// Legacy token without portal_user_id — derive from the participant.
		if p, err := h.participants.GetParticipant(ctx, portalParticipantID(c)); err == nil {
			phone = p.Phone
		}
	}

	trips, err := h.participants.ListTrips(ctx, puID, phone)
	if err != nil {
		return serverErr(c, err)
	}

	now := time.Now()
	active := []map[string]interface{}{}
	history := []map[string]interface{}{}
	for i := range trips {
		t := &trips[i]
		paymentStatus := h.tripPaymentStatus(ctx, t.ID)
		card := map[string]interface{}{
			"participant_id": t.ID,
			"package_name":   t.PackageName,
			"room_type":      t.RoomType,
			"departure_date": t.BatchDepartureDate,
			"is_active":      t.IsActive,
			"payment_status": paymentStatus,
		}
		isHistory := t.BatchDepartureDate != nil && t.BatchDepartureDate.Before(now)
		if isHistory {
			card["completion_status"] = tripCompletionStatus(paymentStatus)
			history = append(history, card)
		} else {
			active = append(active, card)
		}
	}
	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"active":  active,
		"history": history,
	}))
}

// Completion badges for a past trip (FR-PORTAL-09).
const (
	tripCompleted = "selesai"
	tripCancelled = "dibatalkan"
)

// tripCompletionStatus decides which badge a past trip carries.
//
// FR-PORTAL-09 asks for "Selesai" and "Dibatalkan", and the schema has no notion
// of a cancelled trip. Rather than add a status column nothing in the system
// would ever set — no PRD flow cancels a participant, and no admin screen asks
// to — the badge is derived from what the data already knows: a departure date
// that has passed on a trip that was never paid in full is a trip that did not
// happen. Recorded as a decision in the ticket.
func tripCompletionStatus(paymentStatus string) string {
	if paymentStatus == "lunas" {
		return tripCompleted
	}
	return tripCancelled
}

// tripPaymentStatus summarizes the invoice state of one tour: "lunas" when any
// invoice is settled, otherwise the latest invoice status, or "-" when none.
func (h *PortalHandler) tripPaymentStatus(ctx context.Context, participantID string) string {
	invs, err := h.invoices.GetInvoicesByParticipant(ctx, participantID)
	if err != nil || len(invs) == 0 {
		return "-"
	}
	for _, inv := range invs {
		if inv.Status == "lunas" {
			return "lunas"
		}
	}
	return invs[0].Status
}

// PortalTripInvoicePDF downloads the invoice PDF of any tour owned by the logged-in
// portal user, including past trips (v2.0 F2 — download artefak lama).
func (h *PortalHandler) PortalTripInvoicePDF(c echo.Context) error {
	ctx := c.Request().Context()
	participantID := c.Param("participant_id")

	owned, err := portalOwnsParticipant(c, h.participants, participantID)
	if err != nil {
		return serverErr(c, err)
	}
	if !owned {
		return notFound(c, "perjalanan tidak ditemukan")
	}

	invs, err := h.invoices.GetInvoicesByParticipant(ctx, participantID)
	if err != nil || len(invs) == 0 {
		return notFound(c, "invoice tidak ditemukan")
	}
	pdfBytes, err := h.invoices.GeneratePDFForParticipant(ctx, invs[0].ID, participantID)
	if err != nil {
		return notFound(c, "invoice tidak ditemukan")
	}
	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=invoice.pdf")
	return c.Blob(http.StatusOK, "application/pdf", pdfBytes)
}

// PortalTripItinerary returns the itinerary of any tour owned by the logged-in
// portal identity, past ones included (FR-PORTAL-10).
//
// The ownership check is the same one the invoice download uses, and it is here
// rather than in a shared wrapper because it has to run before anything about
// the trip is read — an unowned trip must not even reveal which package it was.
func (h *PortalHandler) PortalTripItinerary(c echo.Context) error {
	ctx := c.Request().Context()
	participantID := c.Param("participant_id")

	owned, err := portalOwnsParticipant(c, h.participants, participantID)
	if err != nil {
		return serverErr(c, err)
	}
	if !owned {
		return notFound(c, "perjalanan tidak ditemukan")
	}

	p, err := h.participants.GetParticipant(ctx, participantID)
	if err != nil {
		return notFound(c, "perjalanan tidak ditemukan")
	}
	return h.itineraryOf(c, p)
}

// PortalConsultationPrefill returns the participant's own details for the public
// consultation form (FR-PORTAL-12).
//
// It is an endpoint rather than something the browser remembers because the
// values have to be the ones the system holds now — a name corrected by an admin
// last week should arrive corrected, and a room type read from localStorage
// could be two tours out of date. Returning-customer status is deliberately NOT
// part of this: the lead endpoint decides that from the phone number itself, so
// nothing a client sends can claim it.
func (h *PortalHandler) PortalConsultationPrefill(c echo.Context) error {
	ctx := c.Request().Context()

	p, err := h.participants.GetParticipant(ctx, portalParticipantID(c))
	if err != nil {
		return notFound(c, "peserta tidak ditemukan")
	}
	// The most recent tour holds the freshest room-type preference; the token's
	// own participant is the fallback when there is only one.
	latest := p
	if trips, err := portalTrips(c, h.participants); err == nil && len(trips) > 0 {
		latest = &trips[0] // ListTrips is newest-departure-first
	}

	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"name":      p.Name,
		"phone":     p.Phone,
		"email":     p.Email,
		"room_type": latest.RoomType,
	}))
}

// PortalInvoices returns invoices for the logged-in participant.
func (h *PortalHandler) PortalInvoices(c echo.Context) error {
	pid := portalParticipantID(c)
	invs, err := h.invoices.GetInvoicesByParticipant(c.Request().Context(), pid)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(invs))
}

// PortalInvoicePDF returns the PDF bytes for a specific invoice.
func (h *PortalHandler) PortalInvoicePDF(c echo.Context) error {
	pid := portalParticipantID(c)
	pdfBytes, err := h.invoices.GeneratePDFForParticipant(c.Request().Context(), c.Param("id"), pid)
	if err != nil {
		return notFound(c, "invoice tidak ditemukan")
	}
	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=invoice.pdf")
	return c.Blob(http.StatusOK, "application/pdf", pdfBytes)
}

// PortalUploadProof lets participant upload payment proof.
func (h *PortalHandler) PortalUploadProof(c echo.Context) error {
	var pp invoice.PaymentProof
	if err := bindJSON(c, &pp); err != nil {
		return invalidPayload(c, err, "format tidak valid")
	}
	pp.InvoiceID = c.Param("id")
	pid := portalParticipantID(c)
	if !pathBelongsToParticipant(pp.FilePath, pid) {
		return badRequest(c, "file_path bukan berkas Anda — unggah lewat /portal/upload/payment-proof")
	}
	if err := h.invoices.UploadProofForParticipant(c.Request().Context(), &pp, pid); err != nil {
		if errors.Is(err, invoicesvc.ErrNotOwned) {
			return notFound(c, "invoice tidak ditemukan")
		}
		return serverErr(c, err)
	}
	// §3.3 notify admins a new payment proof needs verification.
	amount := pp.AmountClaimed
	safe.Go("notifikasi admin bukti bayar baru", func() {
		bg := context.Background()
		p, err := h.participants.GetParticipant(bg, pid)
		if err != nil {
			return
		}
		verifyLink := config.AppURL() + "/admin/invoices"
		h.notifyAdmins(bg, func(adminEmail string) {
			_ = h.email.SendEmailAdminPaymentProof(bg, adminEmail, p.Name,
				format.Rupiah(amount), verifyLink)
		})
	})
	return c.JSON(http.StatusCreated, ok(pp))
}

// PortalCreatePayment creates a Midtrans Snap transaction for an invoice (v2.0 F1).
//
//	@Summary  Buat transaksi pembayaran Midtrans untuk invoice peserta
//	@Tags     portal
//	@Param    id path string true "Invoice ID"
//	@Success  200 {object} map[string]interface{}
//	@Router   /portal/invoices/{id}/create-payment [post]
func (h *PortalHandler) PortalCreatePayment(c echo.Context) error {
	pid := portalParticipantID(c)
	token, clientKey, err := h.invoices.CreatePaymentForParticipant(c.Request().Context(), c.Param("id"), pid)
	if err != nil {
		switch {
		case errors.Is(err, invoicesvc.ErrNotOwned):
			return notFound(c, "invoice tidak ditemukan")
		case errors.Is(err, invoicesvc.ErrInvoiceAlreadyPaid):
			return c.JSON(http.StatusConflict, errResponse("INVOICE_PAID", "invoice sudah lunas"))
		default:
			return serverErr(c, err)
		}
	}
	return c.JSON(http.StatusOK, ok(map[string]string{
		"snap_token": token,
		"client_key": clientKey,
	}))
}

// PortalDocuments lists documents for the logged-in participant.
func (h *PortalHandler) PortalDocuments(c echo.Context) error {
	pid := portalParticipantID(c)
	docs, err := h.docs.ListByParticipant(c.Request().Context(), pid)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(docs))
}

// PortalUploadDocument lets participant upload a document.
func (h *PortalHandler) PortalUploadDocument(c echo.Context) error {
	pid := portalParticipantID(c)
	var d document.Document
	if err := bindJSON(c, &d); err != nil {
		return invalidPayload(c, err, "format tidak valid")
	}
	d.ParticipantID = pid
	if !pathBelongsToParticipant(d.FilePath, pid) {
		return badRequest(c, "file_path bukan berkas Anda — unggah lewat /portal/upload/document")
	}
	if err := h.docs.Create(c.Request().Context(), &d); err != nil {
		return serverErr(c, err)
	}
	// v2.0 F6 — async OCR via self-hosted Tesseract (best-effort).
	if h.ocr != nil && h.ocr.Enabled() && d.FilePath != "" {
		safe.Go("OCR dokumen portal", func() {
			h.ocr.ProcessDocument(context.Background(), d.ID, d.ParticipantID, d.FilePath, d.DocumentType)
		})
	}
	// §3.3 notify admins a new document needs review.
	docType := d.DocumentType
	safe.Go("notifikasi admin dokumen baru", func() {
		bg := context.Background()
		p, err := h.participants.GetParticipant(bg, pid)
		if err != nil {
			return
		}
		reviewLink := config.AppURL() + "/admin/documents"
		h.notifyAdmins(bg, func(adminEmail string) {
			_ = h.email.SendEmailAdminDocUploaded(bg, adminEmail, p.Name, docType, reviewLink)
		})
	})
	return c.JSON(http.StatusCreated, ok(d))
}

// PortalCountryRequirements returns document requirements for participant's destination (AC-PORTAL-02).
// Resolves country_code automatically: participant → batch → package → destination → country code lookup.
func (h *PortalHandler) PortalCountryRequirements(c echo.Context) error {
	pid := portalParticipantID(c)
	p, err := h.participants.GetParticipant(c.Request().Context(), pid)
	if err != nil {
		return notFound(c, "peserta tidak ditemukan")
	}

	// Allow override via query for testing
	countryCode := c.QueryParam("country_code")
	if countryCode == "" {
		// Auto-resolve from participant → batch → package destination
		batch, err := h.packages.GetBatch(c.Request().Context(), p.BatchID)
		if err == nil {
			pkg, err := h.packages.GetPackage(c.Request().Context(), batch.PackageID)
			if err == nil {
				countryCode = destinationToCountryCode(pkg.Destination)
			}
		}
	}

	reqs, err := h.countryReqs.List(c.Request().Context(), countryCode)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(reqs))
}

// PortalItinerary returns the itinerary from the participant's package.
func (h *PortalHandler) PortalItinerary(c echo.Context) error {
	p, err := h.participants.GetParticipant(c.Request().Context(), portalParticipantID(c))
	if err != nil {
		return notFound(c, "peserta tidak ditemukan")
	}
	return h.itineraryOf(c, p)
}

// itineraryOf renders one tour's itinerary. Shared by the current tour and by
// the archive download, so a past trip's itinerary is the same document the
// participant saw while it was upcoming.
func (h *PortalHandler) itineraryOf(c echo.Context, p *participant.Participant) error {
	ctx := c.Request().Context()
	batch, err := h.packages.GetBatch(ctx, p.BatchID)
	if err != nil {
		return serverErr(c, err)
	}
	pkg, err := h.packages.GetPackage(ctx, batch.PackageID)
	if err != nil {
		return serverErr(c, err)
	}

	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"package_name":   pkg.Name,
		"destination":    pkg.Destination,
		"duration_days":  pkg.DurationDays,
		"departure_date": p.BatchDepartureDate,
		"return_date":    batch.ReturnDate,
		"itinerary":      pkg.Itinerary,
		"requirements":   pkg.Requirements,
	}))
}

// PortalBatchLeader returns the tour leader + WA group link for the participant's batch.
func (h *PortalHandler) PortalBatchLeader(c echo.Context) error {
	pid := portalParticipantID(c)
	p, err := h.participants.GetParticipant(c.Request().Context(), pid)
	if err != nil {
		return notFound(c, "peserta tidak ditemukan")
	}
	batch, err := h.packages.GetBatch(c.Request().Context(), p.BatchID)
	if err != nil {
		return c.JSON(http.StatusOK, ok(nil))
	}

	result := map[string]interface{}{
		"wa_group_link": batch.WaGroupLink,
	}
	if batch.TourLeaderID != nil {
		tl, err := h.tourLeaders.GetByUserID(c.Request().Context(), *batch.TourLeaderID)
		if err == nil {
			result["tour_leader"] = tl
		}
	}
	return c.JSON(http.StatusOK, ok(result))
}

// PortalBriefingPDF generates and returns the briefing PDF.
func (h *PortalHandler) PortalBriefingPDF(c echo.Context) error {
	pid := portalParticipantID(c)
	p, err := h.participants.GetParticipant(c.Request().Context(), pid)
	if err != nil {
		return notFound(c, "peserta tidak ditemukan")
	}
	if !isBriefingActive(p) {
		return c.JSON(http.StatusForbidden, errResponse("NOT_YET", "Briefing belum aktif (tersedia H-14 sebelum keberangkatan)"))
	}

	depDate := "—"
	if p.BatchDepartureDate != nil {
		depDate = p.BatchDepartureDate.Format("02 January 2006")
	}

	data := service.BriefingData{
		ParticipantName: p.Name,
		PackageName:     p.PackageName,
		DepartureDate:   depDate,
		TourLeaderName:  "Tour Leader Pintour",
		TourLeaderPhone: "(hubungi admin untuk detail)",
		TourLeaderBio:   "Tour leader berpengalaman yang akan menemani perjalanan Anda.",
	}

	pdfBytes, err := h.pdf.GenerateBriefing(data)
	if err != nil {
		return serverErr(c, err)
	}
	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=briefing.pdf")
	return c.Blob(http.StatusOK, "application/pdf", pdfBytes)
}

// PortalUpdateProfile lets participant update their own data (§15.4 portal-profile).
func (h *PortalHandler) PortalUpdateProfile(c echo.Context) error {
	pid := portalParticipantID(c)
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email" validate:"omitempty,email"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "format tidak valid")
	}
	p, err := h.participants.GetParticipant(c.Request().Context(), pid)
	if err != nil {
		return notFound(c, "peserta tidak ditemukan")
	}
	if body.Name != "" {
		p.Name = body.Name
	}
	if body.Email != "" {
		p.Email = body.Email
	}
	if err := h.participants.UpdateProfile(c.Request().Context(), p); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"id": p.ID, "name": p.Name, "email": p.Email, "phone": p.Phone,
	}))
}

// PortalMyData godoc
// @Summary      Unduh semua data pribadi (UU PDP §25.5 Right to Access)
// @Tags         portal
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /portal/my-data [get]
func (h *PortalHandler) PortalMyData(c echo.Context) error {
	pid := portalParticipantID(c)
	p, err := h.participants.GetParticipant(c.Request().Context(), pid)
	if err != nil {
		return notFound(c, "peserta tidak ditemukan")
	}
	invs, _ := h.invoices.GetInvoicesByParticipant(c.Request().Context(), pid)
	docs, _ := h.docs.ListByParticipant(c.Request().Context(), pid)
	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"participant": map[string]interface{}{
			"id":    p.ID,
			"name":  p.Name,
			"phone": p.Phone,
			"email": p.Email,
		},
		"invoices":  invs,
		"documents": docs,
		"note":      "Data ini adalah semua informasi pribadi yang tersimpan di sistem Pintour.",
	}))
}

// PortalRequestDeletion submits an account deletion request (§25.5 Right to Erasure).
//
// The answer this gives is a legal commitment — 14 working days under UU PDP
// Pasal 46 — so it may only be given once the request is somewhere an admin will
// find it. It used to be given after storing nothing at all: the participant was
// told their data would be erased, and no one ever learned they had asked.
//
// Asking twice returns the request already open rather than filing a second one.
// Pressing the button again is what someone does when they are not sure the
// first press worked; it should not put the same person in the queue twice.
func (h *PortalHandler) PortalRequestDeletion(c echo.Context) error {
	pid := portalParticipantID(c)
	var body struct {
		Reason string `json:"reason" validate:"omitempty,max=1000"`
	}
	// The reason is optional, and the PRD spells this endpoint as DELETE — which
	// carries no body at all. An absent body is therefore a valid request, not a
	// malformed one; anything present still has to parse and validate.
	if err := bindJSON(c, &body); err != nil && !errors.Is(err, io.EOF) {
		return invalidPayload(c, err, "format tidak valid")
	}
	if h.deletions == nil {
		return serverErr(c, errDeletionsUnavailable)
	}

	req := &privacy.DeletionRequest{ParticipantID: pid, Reason: body.Reason}
	if err := h.deletions.Create(c.Request().Context(), req); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(map[string]any{
		"message": "Permintaan penghapusan data Anda telah diterima. " +
			"Tim Pintour akan memproses dalam 14 hari kerja sesuai UU PDP Pasal 46.",
		"ticket":       "DEL-" + req.ID,
		"status":       req.Status,
		"requested_at": req.RequestedAt,
	}))
}

// ─── Token ────────────────────────────────────────────────────────────────────

type portalClaims struct {
	ParticipantID string `json:"participant_id"`
	PortalUserID  string `json:"portal_user_id,omitempty"` // v2.0 F1 — central portal identity
	jwt.RegisteredClaims
}

func (h *PortalHandler) generatePortalToken(p *participant.Participant) (string, error) {
	claims := portalClaims{
		ParticipantID: p.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	if p.PortalUserID != nil {
		claims.PortalUserID = *p.PortalUserID
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(h.jwtSecret))
}

func portalParticipantID(c echo.Context) string {
	if claims, ok := c.Get("portal_claims").(*portalClaims); ok {
		return claims.ParticipantID
	}
	return ""
}

func portalUserID(c echo.Context) string {
	if claims, ok := c.Get("portal_claims").(*portalClaims); ok {
		return claims.PortalUserID
	}
	return ""
}

// portalOwnsParticipant reports whether the tour identified by participantID
// belongs to the portal identity on this request's token — the question behind
// every "is this mine?" the portal has to answer, whether the thing asked for is
// an invoice PDF or a signed URL to a passport scan.
//
// A returning customer's account spans several tours (v3.0), so ownership is not
// just the participant on the token: it is every tour the portal user owns. The
// phone fallback covers a token issued before portal identities existed —
// without it a customer's own history would read as somebody else's.
//
// Note what that makes "mine": the lookup matches on portal user OR phone, so
// two participants booked on one WhatsApp number are one identity here. That is
// the system's own notion of a portal account — PortalLogin authenticates by
// phone, so a shared number is a shared login — and this is the same rule
// PortalMyTrips lists by, not a wider one introduced for files.
func portalOwnsParticipant(
	c echo.Context,
	participants *participantsvc.Service,
	participantID string,
) (bool, error) {
	if participantID == "" {
		return false, nil
	}
	// The tour the caller logged in as is theirs by definition, and answering it
	// without a lookup keeps the common case working even when the rest is not
	// resolvable.
	if participantID == portalParticipantID(c) {
		return true, nil
	}

	trips, err := portalTrips(c, participants)
	if err != nil {
		return false, err
	}
	for i := range trips {
		if trips[i].ID == participantID {
			return true, nil
		}
	}
	return false, nil
}

// portalTrips lists every tour the caller's portal identity owns, newest
// departure first. The phone fallback covers a token issued before portal
// identities existed; without it a customer's own history reads as somebody
// else's.
func portalTrips(c echo.Context, participants *participantsvc.Service) ([]participant.Participant, error) {
	ctx := c.Request().Context()
	puID := portalUserID(c)

	phone := ""
	if puID != "" {
		if pu, err := participants.GetPortalUser(ctx, puID); err == nil {
			phone = pu.Phone
		}
	}
	if phone == "" {
		if p, err := participants.GetParticipant(ctx, portalParticipantID(c)); err == nil {
			phone = p.Phone
		}
	}
	if puID == "" && phone == "" {
		return nil, nil
	}
	return participants.ListTrips(ctx, puID, phone)
}

// PortalJWTMiddleware validates portal JWT tokens.
func PortalJWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("X-Portal-Token")
			if token == "" {
				return c.JSON(http.StatusUnauthorized, errResponse("UNAUTHORIZED", "Token portal diperlukan"))
			}
			parsed, err := jwt.ParseWithClaims(token, &portalClaims{}, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !parsed.Valid {
				return c.JSON(http.StatusUnauthorized, errResponse("UNAUTHORIZED", "Token tidak valid atau kadaluarsa"))
			}

			// Staff tokens are signed with the same secret and carry user_id and
			// role instead of participant_id, so they parse cleanly into these
			// claims with the participant left empty. Refusing that here is the
			// mirror of the staff middleware refusing a token with no user_id or
			// role, and completes the §19.1 separation in both directions.
			//
			// It also refuses a token that identifies nobody at all: without this
			// every portal handler below would run with an empty participant id
			// and report "no rows" where it means "not authenticated".
			claims, ok := parsed.Claims.(*portalClaims)
			if !ok || claims.ParticipantID == "" {
				return c.JSON(http.StatusUnauthorized,
					errResponse("UNAUTHORIZED", "Token bukan untuk akses portal peserta"))
			}

			c.Set("portal_claims", claims)
			return next(c)
		}
	}
}

// briefingWindowDays is how close to departure the briefing materials open
// (§5.8): H-14.
const briefingWindowDays = 14

// isBriefingActive reports whether the briefing is open to this participant.
//
// The comparison is on whole days, and inclusive, which is what §5.8's "H-14"
// says and what the portal already tells the participant: the countdown page
// stops showing "available in N days" once the count reaches fourteen. The
// previous form compared hours, so a participant whose countdown read 14 was
// shown an open briefing and given a closed one.
func isBriefingActive(p *participant.Participant) bool {
	return p.HasDeparture() && p.DaysUntilDeparture() <= briefingWindowDays
}

// destinationToCountryCode delegates to service.DestinationToCountryCode (shared
// with the application layer so the convert flow can resolve doc requirements).
func destinationToCountryCode(destination string) string {
	return service.DestinationToCountryCode(destination)
}

func normalizePhone(phone string) string {
	if len(phone) == 0 {
		return phone
	}
	if phone[0] == '+' {
		phone = phone[1:]
	} else if len(phone) > 1 && phone[0] == '0' {
		phone = "62" + phone[1:]
	}
	return phone
}

// errDeletionsUnavailable means the erasure store was not wired in. It is a
// server fault, never the caller's: answering "diterima" without one is the
// defect this endpoint was built to remove.
var errDeletionsUnavailable = errors.New("deletion request repository not configured")
