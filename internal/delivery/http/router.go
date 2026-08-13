package httpdelivery

import (
	"database/sql"
	"net/http"
	"sync"
	"time"

	_ "github.com/irfan-ghzl/pintour-travel/docs"
	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	leadsvc "github.com/irfan-ghzl/pintour-travel/internal/application/lead"
	pkgsvc "github.com/irfan-ghzl/pintour-travel/internal/application/package"
	participantsvc "github.com/irfan-ghzl/pintour-travel/internal/application/participant"
	usersvc "github.com/irfan-ghzl/pintour-travel/internal/application/user"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/airport"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/chatbot"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/privacy"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// Services bundles all services needed to register routes.
type Services struct {
	Package        *pkgsvc.Service
	Lead           *leadsvc.Service
	Participant    *participantsvc.Service
	Invoice        *invoicesvc.Service
	User           *usersvc.UserService
	UserRepo       domainUser.Repository
	TourLeaderRepo domainUser.TourLeaderRepository
	Airport        airport.Repository
	Participants   participant.Repository
	Document       document.Repository
	// TODO(ocr-v2.0-F3): re-enable when GCP Vision billing active
	OCRRepo    document.OCRResultRepository
	OCR        *service.OCRService
	CountryReq document.CountryRequirementRepository
	PDF        *service.PDFService
	Fonnte     *service.FonnteService
	Email      *service.EmailService
	Storage    *service.StorageService
	NotifRepo  notification.Repository
	// Deletions stores the §25.5 erasure requests the portal accepts and the
	// admin queue works through.
	Deletions    privacy.Repository
	DB           *sql.DB
	Midtrans     *service.MidtransService
	Chatbot      *service.ChatbotService
	ChatbotRepo  chatbot.Repository
	ChatbotToken string
	AppURL       string
	// PortalURL is where participants log in — distinct from AppURL, which is
	// the admin-facing front end.
	PortalURL      string
	JWTSecret      string
	JWTExpiryHours int
	Production     bool
}

// RegisterRoutes mounts all API routes.
func RegisterRoutes(e *echo.Echo, svc Services) {
	// Runtime safeguards (§ketahanan runtime). They live here, not at the call
	// site, so every deployment gets them and no future entry point can forget.
	// Recovery goes on first, which puts every later e.Use — the caller's CORS
	// adapter included — inside it.
	e.Use(recoverPanics(), bodyLimit())
	// Reset tokens are held in memory between the request that issues one and
	// the request that redeems it; most are never redeemed, so they are evicted
	// on a timer the way the rate limiter evicts its per-IP buckets.
	resetTokens.startSweeper(resetTokenTTL)

	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "version": "2.0.0"})
	})

	pkgH := NewPackageHandler(svc.Package)
	leadH := NewLeadHandler(svc.Lead, svc.NotifRepo)
	paxH := NewParticipantHandler(svc.Participant, svc.PortalURL)
	invH := NewInvoiceHandler(svc.Invoice)
	userH := NewUserHandler(svc.User, svc.UserRepo, svc.Email, svc.AppURL, svc.Production, svc.JWTExpiryHours)
	airH := NewAirportHandler(svc.Airport, svc.Participants, svc.Package, svc.TourLeaderRepo, svc.Fonnte, svc.PDF)
	docH := NewDocumentHandler(svc.Document, svc.CountryReq, svc.Participants, svc.Airport, svc.Fonnte, svc.Email, svc.OCR, svc.OCRRepo)
	tlH := NewTourLeaderHandler(svc.TourLeaderRepo)
	portalH := NewPortalHandler(svc.Participant, svc.Invoice, svc.Package, svc.Document, svc.CountryReq, svc.TourLeaderRepo, svc.UserRepo, svc.PDF, svc.Email, svc.OCR, svc.Deletions, svc.JWTSecret)
	uploadH := NewUploadHandler(svc.Storage, svc.Document, svc.Invoice, svc.Participant)
	dashH := NewDashboardHandler(svc.DB)
	reportH := NewReportHandler(svc.DB)
	webhookH := NewWebhookHandler(svc.Midtrans, svc.Invoice)
	chatbotH := NewChatbotHandler(svc.Chatbot, svc.ChatbotRepo, svc.Lead, svc.ChatbotToken)
	privacyH := NewPrivacyHandler(svc.Deletions)

	api := e.Group("/api/v1")

	// Rate limiter umum endpoint publik (60 req/min per IP)
	rl := rateLimiter(60, time.Minute)
	// Rate limiter ketat untuk endpoint auth: 10 req/min per IP (cegah brute force §10.3)
	rlAuth := rateLimiter(10, time.Minute)

	// ── Public routes ──────────────────────────────────────────────────────────
	api.POST("/auth/login", userH.Login, rlAuth)
	api.POST("/auth/logout", userH.Logout)
	api.POST("/auth/forgot-password", userH.ForgotPassword, rlAuth)
	api.POST("/auth/reset-password", userH.ResetPassword, rlAuth)
	api.POST("/portal/login", portalH.PortalLogin, rlAuth)

	api.GET("/packages", pkgH.ListPackages, rl)
	api.GET("/packages/:slug", pkgH.GetPackageBySlug, rl)

	api.POST("/leads", leadH.CreateLead, rl)

	api.GET("/country-requirements/:country_code", docH.ListCountryRequirements)

	// ── Public webhooks (no JWT; verified by signature/token) ───────────────────
	rlWebhook := rateLimiter(100, time.Minute) // §keamanan: rate limit endpoint publik
	api.POST("/webhooks/midtrans", webhookH.HandleMidtrans, rlWebhook)
	api.POST("/webhooks/fonnte", chatbotH.HandleFonnteWebhook, rlWebhook) // v2.0 F2 chatbot inbound

	// ── Portal routes (participant JWT) ────────────────────────────────────────
	portalMW := PortalJWTMiddleware(svc.JWTSecret)
	portal := api.Group("/portal", portalMW)

	portal.GET("/me", portalH.PortalMe)
	portal.GET("/my-trips", portalH.PortalMyTrips)                                    // v2.0 F2 riwayat perjalanan
	portal.GET("/my-trips/:participant_id/invoice-pdf", portalH.PortalTripInvoicePDF) // v2.0 F2 download artefak lama
	portal.GET("/my-trips/:participant_id/itinerary", portalH.PortalTripItinerary)    // FR-PORTAL-10
	portal.GET("/consultation-prefill", portalH.PortalConsultationPrefill)            // FR-PORTAL-12
	portal.PUT("/profile", portalH.PortalUpdateProfile)                               // §15.4 portal-profile
	portal.GET("/my-data", portalH.PortalMyData)                                      // §25.5 Right to Access
	// §25.5 Right to Erasure. The PRD writes this as DELETE and the portal client
	// sends POST; both reach the same handler rather than one of them being wrong.
	portal.POST("/account-deletion-request", portalH.PortalRequestDeletion)
	portal.DELETE("/account-deletion-request", portalH.PortalRequestDeletion)
	portal.GET("/invoices", portalH.PortalInvoices)
	portal.GET("/invoices/:id/pdf", portalH.PortalInvoicePDF)
	portal.POST("/invoices/:id/proofs", portalH.PortalUploadProof)
	portal.POST("/invoices/:id/create-payment", portalH.PortalCreatePayment) // v2.0 F1 Midtrans
	portal.GET("/documents", portalH.PortalDocuments)
	portal.POST("/documents", portalH.PortalUploadDocument)
	portal.GET("/documents/requirements", portalH.PortalCountryRequirements)
	portal.GET("/itinerary", portalH.PortalItinerary)
	portal.GET("/briefing/pdf", portalH.PortalBriefingPDF)
	portal.GET("/batch-leader", portalH.PortalBatchLeader) // FR-BRIEF-02/03

	// Upload endpoints (Supabase Storage §16.2)
	portal.POST("/upload/document", uploadH.UploadDocument)
	portal.POST("/upload/payment-proof", uploadH.UploadPaymentProof)
	portal.GET("/signed-url", uploadH.SignedURLForPortal) // §19.2, own files only

	// ── Protected admin routes (staff JWT + per-role RBAC) ─────────────────────
	// Each group shares the /admin prefix but applies a different role allow-list
	// (RequireRole). The sidebar only hides links; this is what actually blocks an
	// out-of-scope role from reaching the data via the API directly.
	jwtMW := JWTMiddleware(svc.JWTSecret)

	// Base: any authenticated staff — shared dashboard + own profile.
	admin := api.Group("/admin", jwtMW)
	// Ops: super_admin + admin — operational management.
	ops := api.Group("/admin", jwtMW, RequireRole("super_admin", "admin"))
	// Sales: + konsultan — CRM leads & their own participants.
	sales := api.Group("/admin", jwtMW, RequireRole("super_admin", "admin", "konsultan"))
	// Airport: + tour_leader — departure handling.
	airportG := api.Group("/admin", jwtMW, RequireRole("super_admin", "admin", "tour_leader"))
	// Pickers: every staff role, because naming a departure is what each of them
	// does instead of typing its UUID. Ditulis lengkap alih-alih memakai grup
	// dasar `admin`, supaya peran yang ditambahkan nanti tidak ikut diberi akses
	// tanpa ada yang memutuskannya.
	pickers := api.Group("/admin", jwtMW,
		RequireRole("super_admin", "admin", "konsultan", "tour_leader"))
	// Super: super_admin only — user management.
	super := api.Group("/admin", jwtMW, RequireRole("super_admin"))

	admin.GET("/auth/me", userH.Me)
	admin.GET("/dashboard/stats", dashH.GetStats)
	admin.GET("/dashboard/analytics", dashH.GetAnalytics) // prompt §4.3
	// Signed URL so staff can open a private file (participant document / bank
	// transfer) in a private Supabase bucket; the raw path is unreadable without
	// a token (§19.2).
	//
	// §5.3 does not cover this endpoint, so its group is a decision rather than a
	// reading of the matrix: it goes on ops, alongside the document-review and
	// invoice routes that list the very files it unlocks. A signer must not hand
	// out more than the listings do, which is why konsultan and tour_leader — who
	// cannot list participant documents — cannot sign one either. Recorded in
	// .scratch/prd-alignment/issues/04-kontrol-akses.md, together with the §5.3
	// row proposed to make the matrix cover it.
	ops.GET("/signed-url", uploadH.SignedURLForStaff)
	ops.GET("/reports/export", reportH.Export) // prompt §4.1/§4.2

	// Erasure queue (§25.5). Ops-only: it exposes who asked to be forgotten and
	// carries out the anonymisation.
	ops.GET("/privacy/deletion-requests", privacyH.ListDeletionRequests)
	ops.POST("/privacy/deletion-requests/:id/process", privacyH.ProcessDeletionRequest)

	// Chatbot logs (v2.0 F2)
	ops.GET("/chatbot-logs", chatbotH.ListConversations)
	ops.GET("/chatbot-logs/:phone", chatbotH.GetConversation)
	ops.POST("/chatbot-logs/:phone/create-lead", chatbotH.CreateLeadFromChat)

	// Packages (CMS) — incl. batch & WA group link
	ops.GET("/packages", pkgH.AdminListPackages)
	ops.POST("/packages", pkgH.CreatePackage)
	ops.PUT("/packages/:id", pkgH.UpdatePackage)
	ops.DELETE("/packages/:id", pkgH.DeletePackage)
	ops.GET("/packages/:package_id/images", pkgH.ListImages)
	ops.POST("/packages/:package_id/images", pkgH.AddImage)
	ops.POST("/packages/:package_id/images/upload", uploadH.UploadPackageImage) // §16.2
	ops.DELETE("/packages/:package_id/images/:image_id", pkgH.DeleteImage)
	// Reading a package's departures sits on sales, not ops. A konsultan may
	// convert their own lead (sales.POST /participants/convert below), and the
	// conversion dialog cannot offer a departure it is forbidden to list — so
	// this route answered 403 to the one role whose daily work needs it, and the
	// dialog showed "gagal memuat" instead of a batch list.
	//
	// It discloses nothing new: the same rows already go to anonymous visitors
	// through GET /packages/{slug} in the public catalogue. Creating and editing
	// departures stays ops-only, which is where the actual authority is.
	sales.GET("/packages/:package_id/batches", pkgH.ListBatches)
	ops.POST("/packages/:package_id/batches", pkgH.CreateBatch)
	ops.PUT("/batches/:id", pkgH.UpdateBatch)

	// Departures across every package — the read behind every batch picker in the
	// admin UI. It sits on the airport group rather than ops because the airport
	// page needs it and a tour_leader works that page; §5.3 gains a row for it
	// (recorded in .scratch/admin-pickers/issues/01-airport-pemilih-batch.md).
	// It answers with what a picker shows and nothing a listing does not already
	// expose: departure date, package name, status, quota, and head count.
	//
	// Konsultan ikut, dan itu koreksi atas keputusan sebelumnya. Endpoint ini
	// mula-mula ditempatkan di grup airport dengan alasan satu-satunya tempat
	// konsultan memilih batch adalah dialog konversi, yang memakai daftar
	// per-paket. Alasan itu melewatkan filter keberangkatan di halaman Peserta —
	// halaman yang memang boleh dibuka konsultan — sehingga baginya filter itu
	// hanya menampilkan "Gagal memuat daftar keberangkatan". Bentuk cacat yang
	// sama persis dengan yang ditemukan pada rute batches per-paket sebelumnya.
	pickers.GET("/batches", pkgH.AdminListBatches)

	// Leads / CRM — konsultan included (handler scopes to own leads)
	sales.GET("/leads", leadH.ListLeads)
	sales.GET("/leads/:id", leadH.GetLead)
	sales.PATCH("/leads/:id/status", leadH.UpdateStatus)
	sales.PATCH("/leads/:id/assign", leadH.AssignLead)
	sales.POST("/leads/:id/notes", leadH.AddNote)

	// Participants — konsultan may convert & view their own (handler scopes list);
	// NIK and document management stay ops-only.
	sales.POST("/participants/convert", paxH.ConvertLead)
	sales.GET("/participants", paxH.ListParticipants)
	sales.GET("/participants/:id", paxH.GetParticipant)
	ops.PATCH("/participants/:id/nik", paxH.SetNIK) // v2.0 F6 — set NIK (mis. dari OCR)
	ops.GET("/participants/:participant_id/documents", docH.ListByParticipant)
	ops.POST("/participants/:participant_id/documents", docH.UploadDocument)

	// Invoices
	ops.GET("/invoices", invH.ListInvoices)
	ops.GET("/invoices/:id", invH.GetInvoice)
	ops.POST("/invoices", invH.CreateInvoice)
	ops.GET("/invoices/:id/pdf", invH.GetInvoicePDF)
	ops.POST("/invoices/:id/confirm", invH.ConfirmPayment)
	ops.POST("/invoices/:id/proofs", invH.UploadProof)
	ops.PATCH("/invoices/:id/proofs/:proof_id/review", invH.ReviewProof)

	// Documents review
	ops.GET("/documents", docH.ListAllDocuments) // global list + filter status/participant
	ops.PATCH("/documents/:id/review", docH.ReviewDocument)
	ops.GET("/documents/:id/ocr-result", docH.GetOCRResult) // v2.0 F6
	ops.DELETE("/documents/:id", docH.DeleteDocument)

	// Country document requirements (configurable per country)
	ops.GET("/country-requirements", docH.ListAllCountryRequirements)
	ops.POST("/country-requirements", docH.CreateCountryRequirement)
	ops.PUT("/country-requirements/:id", docH.UpdateCountryRequirement)
	ops.DELETE("/country-requirements/:id", docH.DeleteCountryRequirement)

	// Airport handling — tour_leader included
	airportG.GET("/airport/checklist", airH.ListChecklist)
	airportG.POST("/airport/checklist/init", airH.InitChecklist) // eksplisit, bukan efek samping pembacaan
	airportG.GET("/airport/report", airH.GetReport)
	airportG.GET("/airport/report/pdf", airH.GetReportPDF) // FR-AIR-06
	airportG.POST("/airport/confirm-departure", airH.ConfirmDeparture)
	airportG.PATCH("/airport/participants/:participant_id/baggage", airH.UpdateBaggage)
	airportG.PATCH("/airport/participants/:participant_id/ticket", airH.UpdateTicket)
	airportG.PATCH("/airport/participants/:participant_id/passport", airH.UpdatePassport)

	// Tour leader profiles
	ops.GET("/tour-leaders", tlH.List)
	ops.GET("/tour-leaders/:user_id", tlH.GetByUser)
	ops.PUT("/tour-leaders/:user_id", tlH.Upsert)

	// Users management (super_admin only — enforced by RequireRole + handler)
	super.GET("/users", userH.ListUsers)
	super.POST("/users", userH.CreateUser)
	super.PUT("/users/:id", userH.UpdateUser)
	super.PATCH("/users/:id/deactivate", userH.DeactivateUser)
	super.PATCH("/users/:id/reset-password", userH.ResetPasswordAdmin)
}

// ── Simple in-memory rate limiter ─────────────────────────────────────────────
// Per-instance buckets agar antar endpoint group tidak saling kunci.

type ipCounter struct {
	count  int
	window time.Time
}

// rateLimiter returns Echo middleware that limits a single IP to maxReq requests
// per rolling window. Each call returns an independent middleware with its own
// in-memory bucket map (so /auth/login dan /leads punya counter terpisah).
func rateLimiter(maxReq int, window time.Duration) echo.MiddlewareFunc {
	var mu sync.Mutex
	buckets := map[string]*ipCounter{}

	// Periodically evict expired buckets so the map does not grow unbounded
	// as new IPs arrive over the lifetime of the process.
	safe.Every("penyapu bucket rate limiter", window, func() {
		now := time.Now()
		mu.Lock()
		defer mu.Unlock()
		for ip, b := range buckets {
			if now.After(b.window) {
				delete(buckets, ip)
			}
		}
	})

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			now := time.Now()
			mu.Lock()
			b, ok := buckets[ip]
			if !ok || now.After(b.window) {
				buckets[ip] = &ipCounter{count: 1, window: now.Add(window)}
				mu.Unlock()
				return next(c)
			}
			b.count++
			over := b.count > maxReq
			mu.Unlock()
			if over {
				return c.JSON(http.StatusTooManyRequests,
					errResponse("RATE_LIMIT", "Terlalu banyak permintaan, coba lagi dalam 1 menit"))
			}
			return next(c)
		}
	}
}
