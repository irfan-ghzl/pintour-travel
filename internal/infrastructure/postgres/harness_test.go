package postgres

// Repository contract tests run against a real Postgres — the "second seam" the
// spec describes, and the one thing a fake repository can never stand in for.
//
// What lives here is everything the schema promises and Go cannot check: that a
// status change and its history row commit together, that soft-deleted rows stay
// out of every listing, that an invoice number is never reused, that a gateway
// notification is claimed exactly once. The HTTP seam proves the behaviour on
// fakes; this proves the SQL underneath it agrees.
//
// Set TEST_DATABASE_URL to a reachable server to run them; without it they skip,
// so a machine with no database still gets a green suite.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	projectdb "github.com/irfan-ghzl/pintour-travel/db"
	"github.com/irfan-ghzl/pintour-travel/internal/migrate"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// ─── Scratch database ─────────────────────────────────────────────────────────

// adminConn opens the server-level connection used to create and drop scratch
// databases, skipping the test when none is configured.
func adminConn(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping repository contract tests")
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		admin.Close()
		t.Skipf("TEST_DATABASE_URL not reachable (%v) — skipping", err)
	}
	return admin, dsn
}

// scratchDatabase creates an empty database and returns a connection to it. The
// schema is left to the caller so a migration test can choose how far to apply.
func scratchDatabase(t *testing.T) *sql.DB {
	t.Helper()
	admin, dsn := adminConn(t)

	name := fmt.Sprintf("repotest_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		admin.Close()
		t.Fatalf("create scratch database: %v", err)
	}

	conn, err := sql.Open("pgx", swapDatabase(dsn, name))
	if err != nil {
		t.Fatalf("open scratch database: %v", err)
	}
	t.Cleanup(func() {
		conn.Close()
		_, _ = admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")
		admin.Close()
	})
	return conn
}

// swapDatabase replaces the database name in a postgres URL, keeping
// credentials, host, and query string.
func swapDatabase(dsn, name string) string {
	slash := strings.LastIndex(dsn, "/")
	if slash < 0 {
		return dsn
	}
	tail := ""
	if q := strings.Index(dsn[slash:], "?"); q >= 0 {
		tail = dsn[slash+q:]
	}
	return dsn[:slash+1] + name + tail
}

// migratedDB is a scratch database with every migration applied — the schema a
// deployment actually runs on.
func migratedDB(t *testing.T) *sql.DB {
	t.Helper()
	db := scratchDatabase(t)
	if _, err := migrate.Run(context.Background(), db, projectdb.Migrations()); err != nil {
		t.Fatalf("migrate scratch database: %v", err)
	}
	return db
}

// ─── Bulk dummy data ──────────────────────────────────────────────────────────

// fixture is the dummy dataset a contract test works against. Ids are kept so a
// test can reach for a specific row without querying for it.
type fixture struct {
	AdminID      string
	ConsultantID string
	PackageID    string
	BatchID      string
	LeadIDs      []string
	PaxIDs       []string
	InvoiceIDs   []string
}

// seedScale is how many leads, participants, and invoices a fixture creates.
// Large enough that a missing index or an unbounded query shows up as a slow or
// wrong answer rather than passing by accident on three rows — the seed data the
// earlier round tested against had eight.
const seedScale = 300

// seedBulk fills a migrated database with a realistic spread: every lead status,
// every invoice status, soft-deleted rows among the live ones, and participants
// both linked and unlinked to a portal identity.
//
// The mix matters more than the volume. A filter that ignores deleted_at passes
// against data that has none; a listing that never paginates passes against ten
// rows. Both defects were found in this project, and both are invisible to a
// fixture that only contains happy rows.
func seedBulk(t *testing.T, db *sql.DB) fixture {
	t.Helper()
	ctx := context.Background()
	var f fixture
	stamp := time.Now().UnixNano()

	must(t, db.QueryRowContext(ctx, `
		INSERT INTO users (name, email, password, role, phone)
		VALUES ('Admin Uji', $1, 'x', 'super_admin', '628100000001') RETURNING id`,
		fmt.Sprintf("admin%d@uji.local", stamp)).Scan(&f.AdminID))

	must(t, db.QueryRowContext(ctx, `
		INSERT INTO users (name, email, password, role, phone)
		VALUES ('Konsultan Uji', $1, 'x', 'konsultan', '628100000002') RETURNING id`,
		fmt.Sprintf("konsultan%d@uji.local", stamp)).Scan(&f.ConsultantID))

	must(t, db.QueryRowContext(ctx, `
		INSERT INTO packages (name, slug, destination, duration_days, base_price, created_by)
		VALUES ('Umroh Uji', $1, 'Arab Saudi', 9, 25000000, $2) RETURNING id`,
		fmt.Sprintf("umroh-uji-%d", stamp), f.AdminID).Scan(&f.PackageID))

	must(t, db.QueryRowContext(ctx, `
		INSERT INTO package_batches (package_id, departure_date, return_date, quota,
			price_single, price_double, price_triple, status)
		VALUES ($1, CURRENT_DATE + 45, CURRENT_DATE + 54, 400, 30000000, 25000000, 22000000, 'tersedia')
		RETURNING id`, f.PackageID).Scan(&f.BatchID))

	statuses := []string{"baru", "dihubungi", "konsultasi", "deal", "tidak_deal", "peserta"}
	invStatuses := []string{"diterbitkan", "menunggu_bayar", "dibayar", "lunas"}

	for i := 0; i < seedScale; i++ {
		phone := fmt.Sprintf("6288%08d", i)

		var leadID string
		// Every sixth lead is soft-deleted, so a listing that forgets deleted_at
		// returns a number no assertion expects.
		deleted := i%6 == 5
		must(t, db.QueryRowContext(ctx, `
			INSERT INTO leads (name, phone, email, package_id, batch_id, pax, source, status,
				assigned_to, consent_given, created_at, updated_at, deleted_at)
			VALUES ($1, $2, $3, $4, $5, 1, 'organic', $6, $7, true,
				NOW() - make_interval(days => $8), NOW(),
				CASE WHEN $9 THEN NOW() ELSE NULL END)
			RETURNING id`,
			fmt.Sprintf("Prospek %03d", i), phone, fmt.Sprintf("prospek%03d@uji.local", i),
			f.PackageID, f.BatchID, statuses[i%len(statuses)], f.ConsultantID,
			i%30, deleted).Scan(&leadID))
		f.LeadIDs = append(f.LeadIDs, leadID)

		// Two thirds of the leads became participants; half of those carry a
		// portal identity, the rest stand in for rows predating it.
		if i%3 == 2 {
			continue
		}
		var portalUserID any
		if i%2 == 0 {
			var puID string
			must(t, db.QueryRowContext(ctx, `
				INSERT INTO portal_users (phone, password_hash, name, email)
				VALUES ($1, 'x', $2, $3)
				ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name
				RETURNING id`, phone, fmt.Sprintf("Peserta %03d", i),
				fmt.Sprintf("peserta%03d@uji.local", i)).Scan(&puID))
			portalUserID = puID
		}

		var paxID string
		must(t, db.QueryRowContext(ctx, `
			INSERT INTO participants (lead_id, portal_user_id, batch_id, name, phone, email,
				room_type, portal_password, is_active, deleted_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'double', 'x', true,
				CASE WHEN $7 THEN NOW() ELSE NULL END)
			RETURNING id`, leadID, portalUserID, f.BatchID,
			fmt.Sprintf("Peserta %03d", i), phone, fmt.Sprintf("peserta%03d@uji.local", i),
			i%10 == 9).Scan(&paxID))
		f.PaxIDs = append(f.PaxIDs, paxID)

		var invID string
		must(t, db.QueryRowContext(ctx, `
			INSERT INTO invoices (invoice_number, participant_id, batch_id, amount, due_date,
				status, issued_by, created_at, updated_at, deleted_at)
			VALUES ($1, $2, $3, 25000000, CURRENT_DATE + 7, $4, $5,
				NOW() - make_interval(days => $6), NOW(),
				CASE WHEN $7 THEN NOW() ELSE NULL END)
			RETURNING id`,
			fmt.Sprintf("INV-UJI-%06d", i), paxID, f.BatchID,
			invStatuses[i%len(invStatuses)], f.AdminID, i%10, i%8 == 7).Scan(&invID))
		f.InvoiceIDs = append(f.InvoiceIDs, invID)

		must(t, exec(db, ctx, `
			INSERT INTO documents (participant_id, document_type, file_path, file_name, status)
			VALUES ($1, 'passport', $2, 'paspor.jpg', $3)`,
			paxID, paxID+"/paspor.jpg",
			[]string{"menunggu", "disetujui", "ditolak"}[i%3]))
	}
	return f
}

func exec(db *sql.DB, ctx context.Context, query string, args ...any) error {
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}
