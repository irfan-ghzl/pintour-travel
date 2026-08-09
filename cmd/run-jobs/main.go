// Package main — dev helper to run the automation scheduler jobs (prompt §2)
// on demand, without waiting for the cron schedule.
//
// Usage:
//
//	go run ./cmd/run-jobs                    # run all 4 jobs
//	go run ./cmd/run-jobs -job stale-leads   # run one job
//	go run ./cmd/run-jobs -job expire-invoices
//
// Jobs: stale-leads | expire-leads | expire-invoices | batch-quota | all
//
// Notes:
//   - WA sends only happen when FONNTE_API_TOKEN is set; otherwise the job logic
//     still runs (and you can inspect DB effects + logs).
//   - Email sends only happen when RESEND_API_KEY is set.
package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/config"
	"github.com/irfan-ghzl/pintour-travel/internal/infrastructure/postgres"
	"github.com/irfan-ghzl/pintour-travel/internal/scheduler"
	"github.com/irfan-ghzl/pintour-travel/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	job := flag.String("job", "all", "stale-leads|expire-leads|expire-invoices|batch-quota|all")
	flag.Parse()

	cfg := config.Load()

	db, err := sql.Open("pgx", cfg.Database.DSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	notifRepo := postgres.NewNotificationRepo(db)
	paxRepo := postgres.NewParticipantRepo(db)
	userRepo := postgres.NewUserRepo(db)
	fonnte := service.NewFonnteService(cfg.Fonnte.APIToken, notifRepo)
	email := service.NewEmailService(cfg.Email.ResendAPIKey, cfg.Email.FromAddress)

	sched, err := scheduler.New(paxRepo, userRepo, fonnte, email, db)
	if err != nil {
		log.Fatalf("scheduler init: %v", err)
	}

	if cfg.Fonnte.APIToken == "" {
		log.Println("⚠️  FONNTE_API_TOKEN kosong — WA tidak dikirim (logika job tetap jalan)")
	}
	if cfg.Email.ResendAPIKey == "" {
		log.Println("⚠️  RESEND_API_KEY kosong — email tidak dikirim")
	}

	log.Printf("▶️  Menjalankan job: %s", *job)
	if err := sched.RunJobNow(*job); err != nil {
		log.Fatalf("job error: %v", err)
	}
	log.Println("✅ Selesai. Cek log di atas + tabel wa_notifications untuk hasilnya.")
}
