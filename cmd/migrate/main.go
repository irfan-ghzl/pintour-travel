// Command migrate applies pending SQL migrations to the configured database.
//
// The server runs the same code at startup; this exists so an operator can run
// it on its own — before a deploy, or to see what a database is missing without
// starting the service.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/irfan-ghzl/pintour-travel/db"
	"github.com/irfan-ghzl/pintour-travel/internal/config"
	"github.com/irfan-ghzl/pintour-travel/internal/migrate"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	down := flag.Bool("down", false,
		"Batalkan satu migrasi terakhir (butuh berkas .down.sql; membuang data — baca berkasnya dulu)")
	flag.Parse()

	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = config.Load().Database.DSN
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		log.Fatalf("database not reachable: %v", err)
	}

	if *down {
		name, err := migrate.Down(ctx, conn, db.Migrations())
		if err != nil {
			if errors.Is(err, migrate.ErrNoRollback) {
				log.Fatalf("migrate: %v — migrasi ini tidak menyertakan berkas rollback, "+
					"jadi pembatalannya harus diputuskan manual", err)
			}
			log.Fatalf("migrate: %v", err)
		}
		log.Printf("migrate: %s dibatalkan", name)
		return
	}

	ran, err := migrate.Run(ctx, conn, db.Migrations())
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if len(ran) == 0 {
		log.Println("migrate: basis data sudah mutakhir")
		return
	}
	for _, name := range ran {
		log.Printf("migrate: diterapkan %s", name)
	}
	log.Printf("migrate: %d migrasi diterapkan", len(ran))
}
