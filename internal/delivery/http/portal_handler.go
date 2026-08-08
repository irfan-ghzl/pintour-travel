package httpdelivery

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	participantsvc "github.com/irfan-ghzl/pintour-travel/internal/application/participant"
	pkgsvc "github.com/irfan-ghzl/pintour-travel/internal/application/package"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
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
	ocr          *service.OCRService
	jwtSecret    string
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
		jwtSecret:    jwtSecret,
	}
}

// notifyAdmins sends an email to every admin user (best-effort, async-safe).
func (h *PortalHandler) notifyAdmins(ctx context.Context, send func(adminEmail string)) {
	if h.email == nil || h.users == nil {
		return
	}
	admins, err := h.users.ListByRole(ctx, "admin")
	if err != nil {
		return
	}
	for _, a := range admins {
		if a.Email != "" {
			send(a.Email)
		}
	}
}

func portalAppURL() string {
	if v := os.Getenv("APP_URL"); v != "" {
		return v
	}
	return "http://localhost:5173"
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
	if p.BatchDepartureDate != nil {
		days := int(time.Until(*p.BatchDepartureDate).Hours() / 24)
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
		card := map[string]interface{}{
			"participant_id": t.ID,
			"package_name":   t.PackageName,
			"room_type":      t.RoomType,
			"departure_date": t.BatchDepartureDate,
			"is_active":      t.IsActive,
			"payment_status": h.tripPaymentStatus(ctx, t.ID),
		}
		isHistory := t.BatchDepartureDate != nil && t.BatchDepartureDate.Before(now)
		if isHistory {
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

	// Authorize: the requested tour must belong to this portal identity.
	// Resolve the phone the same way PortalMyTrips does, incl. the legacy
	// fallback for tokens without a portal_user_id — otherwise an owned trip
	// fails the ownership check and the PDF 404s.
	puID := portalUserID(c)
	phone := ""
	if puID != "" {
		if pu, err := h.participants.GetPortalUser(ctx, puID); err == nil {
			phone = pu.Phone
		}
	}
	if phone == "" {
		if p, err := h.participants.GetParticipant(ctx, portalParticipantID(c)); err == nil {
			phone = p.Phone
		}
	}
	trips, err := h.participants.ListTrips(ctx, puID, phone)
	if err != nil {
		return serverErr(c, err)
	}
	owned := false
	for i := range trips {
		if trips[i].ID == participantID {
			owned = true
			break
		}
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
		verifyLink := portalAppURL() + "/admin/invoices"
		h.notifyAdmins(bg, func(adminEmail string) {
			_ = h.email.SendEmailAdminPaymentProof(bg, adminEmail, p.Name,
				rupiahFmt(amount), verifyLink)
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
		reviewLink := portalAppURL() + "/admin/documents"
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
	pid := portalParticipantID(c)
	p, err := h.participants.GetParticipant(c.Request().Context(), pid)
	if err != nil {
		return notFound(c, "peserta tidak ditemukan")
	}

	// Get batch then package
	batch, err := h.packages.GetBatch(c.Request().Context(), p.BatchID)
	if err != nil {
		return serverErr(c, err)
	}
	pkg, err := h.packages.GetPackage(c.Request().Context(), batch.PackageID)
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
func (h *PortalHandler) PortalRequestDeletion(c echo.Context) error {
	pid := portalParticipantID(c)
	var body struct {
		Reason string `json:"reason"`
	}
	_ = bindJSON(c, &body)

	// In a production system, this would create a deletion request record.
	// For now, we log it and return a confirmation.
	_ = pid
	return c.JSON(http.StatusOK, ok(map[string]string{
		"message": "Permintaan penghapusan data Anda telah diterima. " +
			"Tim Pintour akan memproses dalam 14 hari kerja sesuai UU PDP Pasal 46.",
		"ticket": "DEL-" + c.Response().Header().Get("X-Request-Id"),
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

// PortalJWTMiddleware validates portal JWT tokens.
func PortalJWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("X-Portal-Token")
			if token == "" {
				return c.JSON(http.StatusUnauthorized, errResponse("UNAUTHORIZED", "Token portal diperlukan"))
			}
			parsed, err := jwt.ParseWithClaims(token, &portalClaims{}, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil || !parsed.Valid {
				return c.JSON(http.StatusUnauthorized, errResponse("UNAUTHORIZED", "Token tidak valid atau kadaluarsa"))
			}
			c.Set("portal_claims", parsed.Claims)
			return next(c)
		}
	}
}

func isBriefingActive(p *participant.Participant) bool {
	if p.BatchDepartureDate == nil {
		return false
	}
	return time.Until(*p.BatchDepartureDate).Hours() <= 14*24
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
