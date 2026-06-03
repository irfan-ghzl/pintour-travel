package httpdelivery

import (
	"net/http"
	"sync"
	"time"

	_ "github.com/irfan-ghzl/pintour-travel/docs"
	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	leadsvc "github.com/irfan-ghzl/pintour-travel/internal/application/lead"
	participantsvc "github.com/irfan-ghzl/pintour-travel/internal/application/participant"
	pkgsvc "github.com/irfan-ghzl/pintour-travel/internal/application/package"
	usersvc "github.com/irfan-ghzl/pintour-travel/internal/application/user"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/airport"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
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
	CountryReq     document.CountryRequirementRepository
	PDF            *service.PDFService
	Fonnte         *service.FonnteService
	Email          *service.EmailService
	Storage        *service.StorageService
	NotifRepo      notification.Repository
	AppURL         string
	JWTSecret      string
	JWTExpiryHours int
	Production     bool
}

// RegisterRoutes mounts all API routes.
func RegisterRoutes(e *echo.Echo, svc Services) {
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "version": "2.0.0"})
	})

	pkgH := NewPackageHandler(svc.Package)
	leadH := NewLeadHandler(svc.Lead, svc.NotifRepo)
	paxH := NewParticipantHandler(svc.Participant)
	invH := NewInvoiceHandler(svc.Invoice)
	userH := NewUserHandler(svc.User, svc.UserRepo, svc.Email, svc.AppURL, svc.Production, svc.JWTExpiryHours)
	airH := NewAirportHandler(svc.Airport, svc.Participants, svc.Fonnte, svc.PDF)
	docH := NewDocumentHandler(svc.Document, svc.CountryReq, svc.Participants, svc.Fonnte)
	tlH := NewTourLeaderHandler(svc.TourLeaderRepo)
	portalH := NewPortalHandler(svc.Participant, svc.Invoice, svc.Package, svc.Document, svc.CountryReq, svc.TourLeaderRepo, svc.PDF, svc.JWTSecret)
	uploadH := NewUploadHandler(svc.Storage)
	dashH := NewDashboardHandler()

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

	// ── Portal routes (participant JWT) ────────────────────────────────────────
	portalMW := PortalJWTMiddleware(svc.JWTSecret)
	portal := api.Group("/portal", portalMW)

	portal.GET("/me", portalH.PortalMe)
	portal.PUT("/profile", portalH.PortalUpdateProfile)         // §15.4 portal-profile
	portal.GET("/my-data", portalH.PortalMyData)               // §25.5 Right to Access
	portal.POST("/account-deletion-request", portalH.PortalRequestDeletion) // §25.5 Right to Erasure
	portal.GET("/invoices", portalH.PortalInvoices)
	portal.GET("/invoices/:id/pdf", portalH.PortalInvoicePDF)
	portal.POST("/invoices/:id/proofs", portalH.PortalUploadProof)
	portal.GET("/documents", portalH.PortalDocuments)
	portal.POST("/documents", portalH.PortalUploadDocument)
	portal.GET("/documents/requirements", portalH.PortalCountryRequirements)
	portal.GET("/itinerary", portalH.PortalItinerary)
	portal.GET("/briefing/pdf", portalH.PortalBriefingPDF)
	portal.GET("/batch-leader", portalH.PortalBatchLeader) // FR-BRIEF-02/03

	// Upload endpoints (Supabase Storage §16.2)
	portal.POST("/upload/document", uploadH.UploadDocument)
	portal.POST("/upload/payment-proof", uploadH.UploadPaymentProof)
	portal.GET("/signed-url", uploadH.SignedURL) // §19.2 signed URL

	// ── Protected admin routes (staff JWT) ────────────────────────────────────
	jwtMW := JWTMiddleware(svc.JWTSecret)
	admin := api.Group("/admin", jwtMW)

	admin.GET("/auth/me", userH.Me)
	admin.GET("/dashboard/stats", dashH.GetStats)

	// Packages (CMS)
	admin.GET("/packages", pkgH.AdminListPackages)
	admin.POST("/packages", pkgH.CreatePackage)
	admin.PUT("/packages/:id", pkgH.UpdatePackage)
	admin.DELETE("/packages/:id", pkgH.DeletePackage)
	admin.GET("/packages/:package_id/images", pkgH.ListImages)
	admin.POST("/packages/:package_id/images", pkgH.AddImage)
	admin.POST("/packages/:package_id/images/upload", uploadH.UploadPackageImage) // §16.2
	admin.DELETE("/packages/:package_id/images/:image_id", pkgH.DeleteImage)
	admin.GET("/packages/:package_id/batches", pkgH.ListBatches)
	admin.POST("/packages/:package_id/batches", pkgH.CreateBatch)
	admin.PUT("/batches/:id", pkgH.UpdateBatch)

	// Leads / CRM
	admin.GET("/leads", leadH.ListLeads)
	admin.GET("/leads/:id", leadH.GetLead)
	admin.PATCH("/leads/:id/status", leadH.UpdateStatus)
	admin.PATCH("/leads/:id/assign", leadH.AssignLead)
	admin.POST("/leads/:id/notes", leadH.AddNote)

	// Participants
	admin.POST("/participants/convert", paxH.ConvertLead)
	admin.GET("/participants", paxH.ListParticipants)
	admin.GET("/participants/:id", paxH.GetParticipant)
	admin.GET("/participants/:participant_id/documents", docH.ListByParticipant)
	admin.POST("/participants/:participant_id/documents", docH.UploadDocument)

	// Invoices
	admin.GET("/invoices", invH.ListInvoices)
	admin.GET("/invoices/:id", invH.GetInvoice)
	admin.POST("/invoices", invH.CreateInvoice)
	admin.GET("/invoices/:id/pdf", invH.GetInvoicePDF)
	admin.POST("/invoices/:id/confirm", invH.ConfirmPayment)
	admin.POST("/invoices/:id/proofs", invH.UploadProof)
	admin.PATCH("/invoices/:id/proofs/:proof_id/review", invH.ReviewProof)

	// Documents review
	admin.PATCH("/documents/:id/review", docH.ReviewDocument)
	admin.DELETE("/documents/:id", docH.DeleteDocument)

	// Country document requirements (configurable per country)
	admin.GET("/country-requirements", docH.ListAllCountryRequirements)
	admin.POST("/country-requirements", docH.CreateCountryRequirement)
	admin.PUT("/country-requirements/:id", docH.UpdateCountryRequirement)
	admin.DELETE("/country-requirements/:id", docH.DeleteCountryRequirement)

	// Airport handling
	admin.GET("/airport/checklist", airH.ListChecklist)
	admin.GET("/airport/report", airH.GetReport)
	admin.GET("/airport/report/pdf", airH.GetReportPDF) // FR-AIR-06
	admin.POST("/airport/confirm-departure", airH.ConfirmDeparture)
	admin.PATCH("/airport/participants/:participant_id/baggage", airH.UpdateBaggage)
	admin.PATCH("/airport/participants/:participant_id/ticket", airH.UpdateTicket)
	admin.PATCH("/airport/participants/:participant_id/passport", airH.UpdatePassport)

	// Tour leader profiles
	admin.GET("/tour-leaders", tlH.List)
	admin.GET("/tour-leaders/:user_id", tlH.GetByUser)
	admin.PUT("/tour-leaders/:user_id", tlH.Upsert)

	// Users management (super_admin only — checked in handler)
	admin.GET("/users", userH.ListUsers)
	admin.POST("/users", userH.CreateUser)
	admin.PUT("/users/:id", userH.UpdateUser)
	admin.PATCH("/users/:id/deactivate", userH.DeactivateUser)
	admin.PATCH("/users/:id/reset-password", userH.ResetPasswordAdmin)
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
