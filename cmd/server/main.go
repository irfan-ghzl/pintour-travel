// Package main is the entry point for the pintour-travel API server.
//
//	@title          Pintour Travel API
//	@version        1.0
//	@description    REST API for Pintour Tour & Travel Consultant System
//	@termsOfService http://pintour.example.com/terms
//
//	@contact.name  Pintour Support
//	@contact.email support@pintour.example.com
//
//	@license.name Apache 2.0
//	@license.url  http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@host     localhost:8080
//	@BasePath /
//
//	@securityDefinitions.apikey BearerAuth
//	@in                         header
//	@name                       Authorization
//	@description                Enter: Bearer <token>
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

	_ "github.com/irfan-ghzl/pintour-travel/docs"
	"github.com/irfan-ghzl/pintour-travel/internal/cache"
	"github.com/irfan-ghzl/pintour-travel/internal/config"
	"github.com/irfan-ghzl/pintour-travel/internal/handler"
	appmiddleware "github.com/irfan-ghzl/pintour-travel/internal/middleware"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func main() {
	cfg := config.Load()

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Printf("warning: database not reachable at startup: %v", err)
	}
	defer db.Close()

	// ── Redis ─────────────────────────────────────────────────────────────────
	var redisClient *cache.Client
	redisClient, err = cache.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Printf("warning: redis not reachable at startup: %v", err)
	}
	if redisClient != nil {
		defer redisClient.Close()
	}

	// ── Echo ──────────────────────────────────────────────────────────────────
	e := echo.New()
	e.HideBanner = true

	// Global middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "[${time_rfc3339}] ${method} ${uri} ${status} ${latency_human}\n",
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderAccept},
	}))

	// ── Routes ────────────────────────────────────────────────────────────────
	registerRoutes(e, db, cfg)

	// ── Start server ──────────────────────────────────────────────────────────
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 Server listening on %s  (env=%s)", addr, cfg.Server.Env)
		log.Printf("📚 Swagger docs: http://localhost:%s/swagger/index.html", cfg.Server.Port)
		if err := e.StartServer(srv); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
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

func registerRoutes(e *echo.Echo, db *sql.DB, cfg *config.Config) {
	// Swagger UI
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"version": "1.0.0",
		})
	})

	// Services
	userSvc := service.NewUserService(db, cfg.JWT.Secret, cfg.JWT.ExpirationHours)

	// Handlers
	tourH := handler.NewTourHandler(db)
	inquiryH := handler.NewInquiryHandler(db, cfg.Server.Env)
	quotationH := handler.NewQuotationHandler(db)
	userH := handler.NewUserHandler(userSvc)
	dashH := handler.NewDashboardHandler()

	api := e.Group("/api/v1")

	// ── Public routes ──────────────────────────────────────────────────────
	// Auth
	api.POST("/auth/login", userH.Login)

	// Packages (public read)
	api.GET("/packages", tourH.ListPackages)
	api.GET("/packages/:slug", tourH.GetPackage)

	// Destinations (public read)
	api.GET("/destinations", tourH.ListDestinations)

	// Testimonials (public read)
	api.GET("/testimonials", tourH.ListTestimonials)

	// Inquiry (public submit)
	api.POST("/inquiries", inquiryH.CreateInquiry)

	// ── Protected admin routes ──────────────────────────────────────────────
	jwtMW := appmiddleware.JWTMiddleware(cfg.JWT.Secret)

	admin := api.Group("/admin", jwtMW)

	// Auth
	admin.GET("/auth/me", userH.Me)

	// Dashboard
	admin.GET("/dashboard/stats", dashH.GetStats)

	// Packages (admin CRUD)
	admin.POST("/packages", tourH.CreatePackage)
	admin.PUT("/packages/:id", tourH.UpdatePackage)
	admin.DELETE("/packages/:id", tourH.DeletePackage)

	// Inquiries (admin)
	admin.GET("/inquiries", inquiryH.ListInquiries)
	admin.PATCH("/inquiries/:id/status", inquiryH.UpdateInquiryStatus)

	// Quotations (admin)
	admin.POST("/quotations", quotationH.CreateQuotation)
	admin.GET("/quotations", quotationH.ListQuotations)
	admin.GET("/quotations/:id", quotationH.GetQuotation)
}
