package scheduler

// The payment-reminder query runs against a real Postgres here, and nowhere
// else. That is deliberate: the defect it guards against was not a logic error a
// fake could catch, but a parameter the driver refused to encode — the query
// failed before it reached the server, on every run, for weeks, while the job
// reported nothing amiss.
//
// Set TEST_DATABASE_URL to a reachable server to run these; without it they
// skip.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	projectdb "github.com/irfan-ghzl/pintour-travel/db"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
	"github.com/irfan-ghzl/pintour-travel/internal/migrate"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// migratedDB hands back an empty database with the project schema applied.
func migratedDB(t *testing.T) *sql.DB {
	t.Helper()

	adminDSN := os.Getenv("TEST_DATABASE_URL")
	if adminDSN == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping database-backed reminder tests")
	}
	admin, err := sql.Open("pgx", adminDSN)
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		admin.Close()
		t.Skipf("TEST_DATABASE_URL not reachable (%v) — skipping", err)
	}

	name := fmt.Sprintf("remindertest_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		admin.Close()
		t.Fatalf("create scratch database: %v", err)
	}

	dsn := adminDSN
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '/' {
			tail := ""
			for j := i; j < len(dsn); j++ {
				if dsn[j] == '?' {
					tail = dsn[j:]
					break
				}
			}
			dsn = dsn[:i+1] + name + tail
			break
		}
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open scratch database: %v", err)
	}
	if _, err := migrate.Run(context.Background(), conn, projectdb.Migrations()); err != nil {
		t.Fatalf("migrate scratch database: %v", err)
	}
	t.Cleanup(func() {
		conn.Close()
		_, _ = admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")
		admin.Close()
	})
	return conn
}

// seedUnpaidInvoice creates the smallest graph the reminder query walks —
// user → package → batch → participant → invoice — with the invoice issued
// ageDays ago, and returns its id and number.
func seedUnpaidInvoice(t *testing.T, db *sql.DB, ageDays int, phone string) (string, string) {
	t.Helper()
	ctx := context.Background()

	var userID string
	must(t, db.QueryRowContext(ctx, `
		INSERT INTO users (name, email, password, role)
		VALUES ('Admin Uji', $1, 'x', 'admin') RETURNING id`,
		fmt.Sprintf("admin%d@uji.local", time.Now().UnixNano())).Scan(&userID))

	var pkgID string
	must(t, db.QueryRowContext(ctx, `
		INSERT INTO packages (name, slug, destination, duration_days, base_price, created_by)
		VALUES ('Paket Uji', $1, 'Jepang', 5, 1000000, $2) RETURNING id`,
		fmt.Sprintf("paket-uji-%d", time.Now().UnixNano()), userID).Scan(&pkgID))

	var batchID string
	must(t, db.QueryRowContext(ctx, `
		INSERT INTO package_batches (package_id, departure_date, return_date, quota,
			price_single, price_double, price_triple, status)
		VALUES ($1, CURRENT_DATE + 30, CURRENT_DATE + 35, 20, 3000000, 2500000, 2000000, 'tersedia')
		RETURNING id`, pkgID).Scan(&batchID))

	var partID string
	must(t, db.QueryRowContext(ctx, `
		INSERT INTO participants (batch_id, name, phone, email, room_type, portal_password, is_active)
		VALUES ($1, 'Peserta Uji', $2, 'peserta@uji.local', 'double', 'x', true)
		RETURNING id`, batchID, phone).Scan(&partID))

	number := fmt.Sprintf("INV-UJI-%d", time.Now().UnixNano()%100000)
	var invID string
	must(t, db.QueryRowContext(ctx, `
		INSERT INTO invoices (invoice_number, participant_id, batch_id, amount, due_date,
			status, issued_by, created_at, updated_at)
		VALUES ($1, $2, $3, 2500000, CURRENT_DATE + 7, 'menunggu_bayar', $4,
			NOW() - make_interval(days => $5), NOW())
		RETURNING id`, number, partID, batchID, userID, ageDays).Scan(&invID))

	return invID, number
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestUnpaidInvoiceRemindersRunsForEveryAge(t *testing.T) {
	db := migratedDB(t)
	s := &Scheduler{db: db}

	// FR-INV-05 asks for three. All three must reach the server; the defect was
	// that none of them did.
	for _, days := range notification.PaymentReminderDays {
		if _, err := s.unpaidInvoiceReminders(context.Background(), days); err != nil {
			t.Errorf("H+%d query failed: %v", days, err)
		}
	}
}

func TestUnpaidInvoiceRemindersFindsAnInvoiceOfThatAge(t *testing.T) {
	db := migratedDB(t)
	s := &Scheduler{db: db}

	invID, number := seedUnpaidInvoice(t, db, 3, "628111222333")

	due, err := s.unpaidInvoiceReminders(context.Background(), 3)
	if err != nil {
		t.Fatalf("H+3 query: %v", err)
	}
	var found *unpaidInvoiceReminder
	for i := range due {
		if due[i].invoiceID == invID {
			found = &due[i]
		}
	}
	if found == nil {
		t.Fatalf("invoice issued 3 days ago was not selected for the H+3 reminder")
	}
	if found.invoiceNumber != number {
		t.Errorf("invoice number = %q, want %q — the message must name the invoice", found.invoiceNumber, number)
	}
	if found.phone != "628111222333" {
		t.Errorf("phone = %q, want the participant's", found.phone)
	}

	// An invoice of a different age is not chased by this reminder.
	other, err := s.unpaidInvoiceReminders(context.Background(), 1)
	if err != nil {
		t.Fatalf("H+1 query: %v", err)
	}
	for _, d := range other {
		if d.invoiceID == invID {
			t.Error("a 3-day-old invoice was also selected for the H+1 reminder")
		}
	}
}

func TestUnpaidInvoiceRemindersSkipsOneAlreadySentToday(t *testing.T) {
	db := migratedDB(t)
	s := &Scheduler{db: db}
	ctx := context.Background()

	invID, _ := seedUnpaidInvoice(t, db, 6, "628444555666")

	// The marker the sender writes: same invoice, same template, today.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO wa_notifications (recipient_phone, recipient_name, message_type,
			message_content, reference_id, reference_type, status)
		VALUES ('628444555666', 'Peserta Uji', $1, 'sudah dikirim', $2, 'invoice', 'sent')`,
		notification.PaymentReminderType(6), invID); err != nil {
		t.Fatalf("insert marker: %v", err)
	}

	due, err := s.unpaidInvoiceReminders(ctx, 6)
	if err != nil {
		t.Fatalf("H+6 query: %v", err)
	}
	for _, d := range due {
		if d.invoiceID == invID {
			t.Error("invoice was selected again despite today's reminder already being logged")
		}
	}
}

func TestPaymentReminderTypesAreDistinctPerAge(t *testing.T) {
	seen := map[string]int{}
	for _, days := range notification.PaymentReminderDays {
		seen[notification.PaymentReminderType(days)]++
	}
	if len(seen) != len(notification.PaymentReminderDays) {
		t.Errorf("reminder ages share a template: %v — the notification log cannot tell them apart", seen)
	}
}
