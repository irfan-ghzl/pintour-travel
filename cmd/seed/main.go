// Package main — Seed default admin user dengan bcrypt cost 12 (PRD §13.2).
//
// Usage:
//
//	go run ./cmd/seed                          # pakai default admin@pintour.com / admin123
//	go run ./cmd/seed -email=x@y.com -pass=abc # custom
//	go run ./cmd/seed -role=super_admin        # custom role
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	_ = godotenv.Load()

	var (
		email    = flag.String("email", "admin@pintour.com", "Email user")
		password = flag.String("pass", "admin123", "Password plain text")
		name     = flag.String("name", "Admin Pintour", "Nama user")
		role     = flag.String("role", "super_admin", "Role (super_admin/admin/konsultan/tour_leader)")
		phone    = flag.String("phone", "628000000000", "Nomor WA")
	)
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://pintour:pintour_pass@localhost:5432/pintour_db?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v\nDB URL: %s", err, dsn)
	}

	// Hash password dengan bcrypt cost 12 (PRD §13.2)
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), 12)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	var existingID string
	err = db.QueryRow(`SELECT id FROM users WHERE email=$1`, *email).Scan(&existingID)

	if err == nil {
		// Sudah ada — update password
		_, err = db.Exec(`
			UPDATE users SET password=$1, name=$2, role=$3, phone=$4, is_active=true, updated_at=NOW()
			WHERE email=$5`, string(hash), *name, *role, *phone, *email)
		if err != nil {
			log.Fatalf("update user: %v", err)
		}
		fmt.Printf("✅ User exists — password updated.\n")
	} else if err == sql.ErrNoRows {
		// Insert baru
		_, err = db.Exec(`
			INSERT INTO users (id, name, email, password, role, phone, is_active, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, true, NOW(), NOW())`,
			*name, *email, string(hash), *role, *phone)
		if err != nil {
			log.Fatalf("insert user: %v", err)
		}
		fmt.Printf("✅ User baru dibuat.\n")
	} else {
		log.Fatalf("query user: %v", err)
	}

	fmt.Println("─────────────────────────────────────")
	fmt.Printf("  Email    : %s\n", *email)
	fmt.Printf("  Password : %s\n", *password)
	fmt.Printf("  Role     : %s\n", *role)
	fmt.Printf("  Name     : %s\n", *name)
	fmt.Println("─────────────────────────────────────")
	fmt.Println("→ Login di http://localhost:3000/login")
}
