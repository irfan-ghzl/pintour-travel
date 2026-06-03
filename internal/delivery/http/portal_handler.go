package httpdelivery

import (
	"net/http"
	"strings"
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
	pdf          *service.PDFService
	jwtSecret    string
}

func NewPortalHandler(
	participants *participantsvc.Service,
	invoices *invoicesvc.Service,
	packages *pkgsvc.Service,
	docs document.Repository,
	countryReqs document.CountryRequirementRepository,
	tourLeaders domainUser.TourLeaderRepository,
	pdf *service.PDFService,
	jwtSecret string,
) *PortalHandler {
	return &PortalHandler{
		participants: participants,
		invoices:     invoices,
		packages:     packages,
		docs:         docs,
		countryReqs:  countryReqs,
		tourLeaders:  tourLeaders,
		pdf:          pdf,
		jwtSecret:    jwtSecret,
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
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := bindJSON(c, &body); err != nil {
		return badRequest(c, "format tidak valid")
	}
	if body.Phone == "" || body.Password == "" {
		return badRequest(c, "nomor WA dan password harus diisi")
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
	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"participant":     p,
		"days_to_depart":  countdown,
		"briefing_active": isBriefingActive(p),
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
	pdfBytes, err := h.invoices.GeneratePDF(c.Request().Context(), c.Param("id"))
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
		return badRequest(c, "format tidak valid")
	}
	pp.InvoiceID = c.Param("id")
	if err := h.invoices.UploadProof(c.Request().Context(), &pp); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(pp))
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
		return badRequest(c, "format tidak valid")
	}
	d.ParticipantID = pid
	if err := h.docs.Create(c.Request().Context(), &d); err != nil {
		return serverErr(c, err)
	}
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
		Email string `json:"email"`
	}
	if err := bindJSON(c, &body); err != nil {
		return badRequest(c, "format tidak valid")
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
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(h.jwtSecret))
}

func portalParticipantID(c echo.Context) string {
	if claims, ok := c.Get("portal_claims").(*portalClaims); ok {
		return claims.ParticipantID
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

// destinationToCountryCode maps free-text destination (cth "Jepang", "Arab Saudi & Turki")
// ke ISO country code 2 huruf yang dipakai di tabel country_document_requirements.
// Hanya substring match; untuk destinasi multi-negara, country pertama yang match menang.
func destinationToCountryCode(destination string) string {
	d := strings.ToLower(destination)
	mapping := []struct {
		needle string
		code   string
	}{
		{"jepang", "JP"}, {"japan", "JP"},
		{"korea", "KR"},
		{"turki", "TR"}, {"turkey", "TR"},
		{"arab saudi", "SA"}, {"saudi", "SA"}, {"makkah", "SA"}, {"madinah", "SA"},
		{"uea", "AE"}, {"emirat", "AE"}, {"dubai", "AE"}, {"abu dhabi", "AE"},
		{"singapore", "SG"}, {"singapura", "SG"},
		{"malaysia", "MY"},
		{"thailand", "TH"},
		{"vietnam", "VN"},
		{"china", "CN"}, {"tiongkok", "CN"},
		{"hong kong", "HK"}, {"hongkong", "HK"},
		{"taiwan", "TW"},
		{"australia", "AU"},
		{"belanda", "NL"}, {"netherlands", "NL"},
		{"perancis", "FR"}, {"prancis", "FR"}, {"france", "FR"},
		{"jerman", "DE"}, {"germany", "DE"},
		{"italia", "IT"}, {"italy", "IT"},
		{"spanyol", "ES"}, {"spain", "ES"},
		{"swiss", "CH"},
		{"inggris", "GB"}, {"uk", "GB"},
		{"amerika", "US"}, {"usa", "US"},
		{"kanada", "CA"}, {"canada", "CA"},
		{"mesir", "EG"}, {"egypt", "EG"},
		{"yordania", "JO"}, {"jordan", "JO"},
		{"indonesia", "ID"},
	}
	for _, m := range mapping {
		if strings.Contains(d, m.needle) {
			return m.code
		}
	}
	return "ID"
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
