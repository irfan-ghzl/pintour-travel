// Package main — Pintour E-Catalog API server entry point.
//
//	@title          Pintour E-Catalog API
//	@version        2.0
//	@description    REST API untuk sistem E-Catalog Tour & Travel dengan WhatsApp Gateway
//	@host           localhost:8080
//	@BasePath       /api/v1
//	@securityDefinitions.apikey BearerAuth
//	@in             header
//	@name           Authorization
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	leadsvc "github.com/irfan-ghzl/pintour-travel/internal/application/lead"
	pkgsvc "github.com/irfan-ghzl/pintour-travel/internal/application/package"
	participantsvc "github.com/irfan-ghzl/pintour-travel/internal/application/participant"
	usersvc "github.com/irfan-ghzl/pintour-travel/internal/application/user"
	"github.com/irfan-ghzl/pintour-travel/internal/cache"
	"github.com/irfan-ghzl/pintour-travel/internal/config"
	httpdelivery "github.com/irfan-ghzl/pintour-travel/internal/delivery/http"
	"github.com/irfan-ghzl/pintour-travel/internal/infrastructure/postgres"
	"github.com/irfan-ghzl/pintour-travel/internal/scheduler"
	"github.com/irfan-ghzl/pintour-travel/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib" // §18: PostgreSQL driver via pgx/v5 (database/sql compat)
	"github.com/joho/godotenv"         // §18: load .env (joho/godotenv)
	"github.com/labstack/echo/v4"
	"github.com/rs/cors" // §19.4: rs/cors middleware
)

func main() {
	// §18: load env from .env if present (joho/godotenv)
	_ = godotenv.Load()

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// ── Database via pgx/v5 stdlib ────────────────────────────────────────────
	// PRD §18: jackc/pgx/v5 sebagai PostgreSQL driver.
	db, err := sql.Open("pgx", cfg.Database.DSN)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Printf("warning: database not reachable at startup: %v", err)
	}
	defer db.Close()

	// ── Redis (optional) ──────────────────────────────────────────────────────
	var redisClient *cache.Client
	if cfg.Redis.Addr != "" {
		redisClient, err = cache.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			log.Printf("warning: redis not reachable: %v", err)
		}
		if redisClient != nil {
			defer redisClient.Close()
		}
	}

	// ── Repositories ──────────────────────────────────────────────────────────
	pkgRepo := postgres.NewPackageRepo(db)
	imgRepo := postgres.NewPackageImageRepo(db)
	batchRepo := postgres.NewPackageBatchRepo(db)
	leadRepo := postgres.NewLeadRepo(db)
	noteRepo := postgres.NewLeadNoteRepo(db)
	paxRepo := postgres.NewParticipantRepo(db)
	portalUserRepo := postgres.NewPortalUserRepo(db) // v2.0 F1 — central portal identity
	invRepo := postgres.NewInvoiceRepo(db)
	proofRepo := postgres.NewPaymentProofRepo(db)
	docRepo := postgres.NewDocumentRepo(db)
	ocrRepo := postgres.NewOCRResultRepo(db) // v2.0 F6 — OCR results
	countryReqRepo := postgres.NewCountryRequirementRepo(db)
	airportRepo := postgres.NewAirportRepo(db)
	notifRepo := postgres.NewNotificationRepo(db)
	userRepo := postgres.NewUserRepo(db)
	tourLeaderRepo := postgres.NewTourLeaderRepo(db)
	chatbotRepo := postgres.NewChatbotRepo(db)
	// Runs the writes that only make sense together — currently lead conversion —
	// as one transaction. It sits beside the repositories rather than replacing
	// them: every single-row operation still goes through the ones above.
	unitOfWork := postgres.NewUnitOfWork(db)

	// ── Services ──────────────────────────────────────────────────────────────
	fonnteSvc := service.NewFonnteService(cfg.Fonnte.APIToken, notifRepo)
	emailSvc := service.NewEmailService(cfg.Email.ResendAPIKey, cfg.Email.FromAddress)
	midtransSvc := service.NewMidtransService(cfg.Midtrans.ServerKey, cfg.Midtrans.ClientKey, cfg.Midtrans.Env)
	if !midtransSvc.Enabled() {
		log.Println("Midtrans not configured — payment gateway endpoints will return error until MIDTRANS_SERVER_KEY is set")
	}
	pdfSvc := service.NewPDFService()
	storageSvc := service.NewStorageService(cfg.Supabase.URL, cfg.Supabase.ServiceKey)
	if !storageSvc.Enabled() {
		log.Println("Supabase Storage not configured — uploads will fall back to manual URL input")
	}

	packageService := pkgsvc.NewService(pkgRepo, imgRepo, batchRepo)
	leadService := leadsvc.NewService(leadRepo, noteRepo, userRepo, portalUserRepo, paxRepo, fonnteSvc, emailSvc)
	// invoiceService is constructed before participantService because the convert
	// flow (§1.1) reuses it to auto-generate invoices.
	invoiceService := invoicesvc.NewService(invRepo, proofRepo, paxRepo, fonnteSvc, pdfSvc, emailSvc, midtransSvc)
	participantService := participantsvc.NewService(paxRepo, leadRepo, portalUserRepo,
		unitOfWork, batchRepo, pkgRepo, invoiceService, countryReqRepo, fonnteSvc)
	userService := usersvc.NewUserService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpirationHours)
	chatbotSvc := service.NewChatbotService(chatbotRepo, db, pkgRepo, fonnteSvc,
		cfg.Chatbot.GeminiKey, cfg.Chatbot.Model, cfg.Chatbot.MaxHistory, cfg.Chatbot.Active)
	if !chatbotSvc.Active() {
		log.Println("Chatbot inactive — set GEMINI_API_KEY and CHATBOT_ACTIVE=true to enable")
	}
	ocrSvc := service.NewOCRService(cfg.OCR.Engine, cfg.OCR.GoogleVisionKey, cfg.OCR.TesseractURL, cfg.OCR.ConfidenceThreshold, ocrRepo, paxRepo, storageSvc)
	if !ocrSvc.Enabled() {
		log.Println("OCR disabled — set OCR_ENGINE=tesseract_local (self-hosted) atau google_vision untuk mengaktifkan")
	} else {
		log.Printf("OCR aktif — engine=%s", cfg.OCR.Engine)
	}

	// ── Scheduler — gocron (§18) ─────────────────────────────────────────────
	sched, err := scheduler.New(paxRepo, userRepo, fonnteSvc, emailSvc, db)
	if err != nil {
		log.Fatalf("scheduler init: %v", err)
	}
	sched.Start()
	defer sched.Stop()
	if cfg.Fonnte.APIToken == "" {
		log.Println("FONNTE_API_TOKEN not set — WA jobs will skip sends, retention cleanup tetap jalan")
	}

	// ── Echo ──────────────────────────────────────────────────────────────────
	e := echo.New()
	e.HideBanner = true
	httpdelivery.RegisterValidator(e) // §18 go-playground/validator

	// CORS via rs/cors (PRD §19.4) — allow credentials untuk httpOnly cookie (§19.1).
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   parseOrigins(cfg.Server.PortalBaseURL, cfg.Email.AppURL),
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderAccept, "X-Portal-Token"},
		AllowCredentials: true,
		MaxAge:           600,
	})

	// ── Routes ────────────────────────────────────────────────────────────────
	httpdelivery.RegisterRoutes(e, httpdelivery.Services{
		Package:        packageService,
		Lead:           leadService,
		Participant:    participantService,
		Invoice:        invoiceService,
		User:           userService,
		UserRepo:       userRepo,
		TourLeaderRepo: tourLeaderRepo,
		Airport:        airportRepo,
		Participants:   paxRepo,
		Document:       docRepo,
		OCRRepo:        ocrRepo, // v2.0 F6
		OCR:            ocrSvc,  // v2.0 F6
		CountryReq:     countryReqRepo,
		PDF:            pdfSvc,
		Fonnte:         fonnteSvc,
		Email:          emailSvc,
		Storage:        storageSvc,
		NotifRepo:      notifRepo,
		DB:             db,
		Midtrans:       midtransSvc,
		Chatbot:        chatbotSvc,
		ChatbotRepo:    chatbotRepo,
		ChatbotToken:   cfg.Chatbot.WebhookToken,
		AppURL:         cfg.Email.AppURL,
		PortalURL:      cfg.Server.PortalBaseURL,
		JWTSecret:      cfg.JWT.Secret,
		JWTExpiryHours: cfg.JWT.ExpirationHours,
		Production:     cfg.Server.Env == "production",
	})

	// After RegisterRoutes on purpose: Echo runs global middleware in the order
	// it was added, so registering CORS last puts it inside the panic recovery
	// RegisterRoutes installs rather than in front of it.
	e.Use(echoCORSAdapter(corsHandler))

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// The one goroutine deliberately left outside safe.Go: it is not background
	// work but the service itself. Recovering a panic here would leave the
	// process alive with nothing listening, which hides the failure instead of
	// surviving it.
	go func() {
		log.Printf("🚀 Pintour API v2.0 — listening on %s (env=%s, driver=pgx/v5)", addr, cfg.Server.Env)
		log.Printf("📚 Swagger: http://localhost:%s/swagger/index.html", cfg.Server.Port)
		if err := e.StartServer(srv); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	log.Println("Server stopped gracefully")
}

// parseOrigins menerima 1-2 URL & balikan sebagai []string untuk CORS.
// Empty atau "*" fallback ke wildcard untuk dev.
func parseOrigins(urls ...string) []string {
	out := []string{}
	for _, u := range urls {
		if u != "" && u != "*" {
			out = append(out, u)
		}
	}
	if len(out) == 0 {
		return []string{"http://localhost:3000", "http://localhost:5173"}
	}
	return out
}

// echoCORSAdapter bridges rs/cors middleware ke Echo MiddlewareFunc.
func echoCORSAdapter(c *cors.Cors) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			handler := c.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx.SetRequest(r)
				ctx.Response().Writer = w
				if err := next(ctx); err != nil {
					ctx.Error(err)
				}
			}))
			handler.ServeHTTP(ctx.Response().Writer, ctx.Request())
			return nil
		}
	}
}
