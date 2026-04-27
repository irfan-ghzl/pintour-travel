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

	bookingsvc "github.com/irfan-ghzl/pintour-travel/internal/application/booking"
	inquirysvc "github.com/irfan-ghzl/pintour-travel/internal/application/inquiry"
	quotationsvc "github.com/irfan-ghzl/pintour-travel/internal/application/quotation"
	toursvc "github.com/irfan-ghzl/pintour-travel/internal/application/tour"
	usersvc "github.com/irfan-ghzl/pintour-travel/internal/application/user"
	"github.com/irfan-ghzl/pintour-travel/internal/cache"
	"github.com/irfan-ghzl/pintour-travel/internal/config"
	httpdelivery "github.com/irfan-ghzl/pintour-travel/internal/delivery/http"
	"github.com/irfan-ghzl/pintour-travel/internal/infrastructure/postgres"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
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
	tourRepo := postgres.NewTourRepo(db)
	bookingRepo := postgres.NewBookingRepo(db)
	inquiryRepo := postgres.NewInquiryRepo(db)
	quotationRepo := postgres.NewQuotationRepo(db)
	userRepo := postgres.NewUserRepo(db)

	httpdelivery.RegisterRoutes(e, httpdelivery.Services{
		Tour:      toursvc.NewTourService(tourRepo),
		Booking:   bookingsvc.NewBookingService(bookingRepo),
		Inquiry:   inquirysvc.NewInquiryService(inquiryRepo, cfg.Server.ConsultantPhone),
		Quotation: quotationsvc.NewQuotationService(quotationRepo),
		User:      usersvc.NewUserService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpirationHours),
		JWTSecret: cfg.JWT.Secret,
	})

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
