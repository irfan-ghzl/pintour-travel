package migrate_test

// Migration 007 and 009 do not just change the shape of the schema — they move
// data. 007 derives one portal identity per phone from every participant that
// exists; 009 lifts each invoice's gateway order onto its own table. Both scan
// whole tables, and both had only ever run against eight seeded rows.
//
// These tests run them against a dataset with the shapes that make a backfill
// wrong rather than slow: repeated phones, soft-deleted rows, rows the backfill
// must skip. Volume alone proves nothing; volume plus those shapes does.

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	projectdb "github.com/irfan-ghzl/pintour-travel/db"
	"github.com/irfan-ghzl/pintour-travel/internal/migrate"
)

// upTo returns the migrations numbered up to and including prefix, so a test can
// stand a database at the version that existed before the one under test.
func upTo(t *testing.T, prefix string) fs.FS {
	t.Helper()
	src := projectdb.Migrations()
	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	subset := fstest.MapFS{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") || e.Name() > prefix+"_zzz" {
			continue
		}
		body, err := fs.ReadFile(src, e.Name())
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		subset[e.Name()] = &fstest.MapFile{Data: body}
	}
	if len(subset) == 0 {
		t.Fatalf("no migrations at or below %s", prefix)
	}
	return subset
}

// backfillScale is how many participants the backfill has to walk. Large enough
// that a per-row mistake shows up as a wrong count rather than hiding among a
// handful of rows.
const backfillScale = 500

// seedParticipantsForBackfill fills the pre-007 schema with participants whose
// shapes decide whether the backfill is correct:
//   - every fifth phone is shared by two participants (a returning customer who
//     booked twice before portal identities existed), so DISTINCT ON has to pick
//     one and the UPDATE has to link both;
//   - every ninth participant is soft-deleted, and the backfill must not mint an
//     identity for one.
func seedParticipantsForBackfill(t *testing.T, db *sql.DB) (distinctPhones, softDeleted int) {
	t.Helper()
	ctx := context.Background()
	stamp := time.Now().UnixNano()

	var adminID, pkgID, batchID string
	mustExec(t, db.QueryRowContext(ctx, `
		INSERT INTO users (name, email, password, role)
		VALUES ('Admin', $1, 'x', 'super_admin') RETURNING id`,
		fmt.Sprintf("admin%d@uji.local", stamp)).Scan(&adminID))
	mustExec(t, db.QueryRowContext(ctx, `
		INSERT INTO packages (name, slug, destination, duration_days, base_price, created_by)
		VALUES ('Paket', $1, 'Jepang', 5, 1000000, $2) RETURNING id`,
		fmt.Sprintf("paket-%d", stamp), adminID).Scan(&pkgID))
	mustExec(t, db.QueryRowContext(ctx, `
		INSERT INTO package_batches (package_id, departure_date, return_date, quota,
			price_single, price_double, price_triple, status)
		VALUES ($1, CURRENT_DATE + 30, CURRENT_DATE + 35, 900, 1, 1, 1, 'tersedia')
		RETURNING id`, pkgID).Scan(&batchID))

	live := map[string]bool{}
	for i := 0; i < backfillScale; i++ {
		// Every fifth index reuses the previous phone.
		phoneIdx := i
		if i%5 == 4 {
			phoneIdx = i - 1
		}
		phone := fmt.Sprintf("6289%08d", phoneIdx)
		deleted := i%9 == 8

		mustExec(t, db.QueryRowContext(ctx, `
			INSERT INTO participants (batch_id, name, phone, email, room_type,
				portal_password, is_active, created_at, deleted_at)
			VALUES ($1, $2, $3, $4, 'double', $5, true,
				NOW() - make_interval(days => $6),
				CASE WHEN $7 THEN NOW() ELSE NULL END)
			RETURNING id`, batchID, fmt.Sprintf("Peserta %03d", i), phone,
			fmt.Sprintf("p%03d@uji.local", i), fmt.Sprintf("hash-%03d", i),
			i%40, deleted).Scan(new(string)))

		if deleted {
			softDeleted++
			continue
		}
		live[phone] = true
	}
	return len(live), softDeleted
}

func mustExec(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

// The backfill must produce exactly one identity per live phone, link every
// participant that has one, and leave soft-deleted rows alone.
func TestPortalUserBackfillAtScale(t *testing.T) {
	db := scratchDB(t)
	ctx := context.Background()

	if _, err := migrate.Run(ctx, db, upTo(t, "006")); err != nil {
		t.Fatalf("migrate to 006: %v", err)
	}
	wantIdentities, softDeleted := seedParticipantsForBackfill(t, db)
	if softDeleted == 0 {
		t.Fatal("fixture has no soft-deleted participants — the skip would be untested")
	}

	started := time.Now()
	if _, err := migrate.Run(ctx, db, projectdb.Migrations()); err != nil {
		t.Fatalf("migrate the rest: %v", err)
	}
	t.Logf("backfill over %d participants took %s", backfillScale, time.Since(started).Round(time.Millisecond))

	var identities int
	if err := db.QueryRow(`SELECT COUNT(*) FROM portal_users`).Scan(&identities); err != nil {
		t.Fatalf("count identities: %v", err)
	}
	if identities != wantIdentities {
		t.Errorf("portal_users = %d, want %d (one per live phone) — a shared phone produced more than one account",
			identities, wantIdentities)
	}

	var unlinked int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM participants
		WHERE deleted_at IS NULL AND portal_user_id IS NULL`).Scan(&unlinked); err != nil {
		t.Fatalf("count unlinked: %v", err)
	}
	if unlinked != 0 {
		t.Errorf("%d live participants were left without a portal identity — those customers cannot log in", unlinked)
	}

	// Two bookings on one phone must land on the same identity: that is what
	// makes the trip history of a returning customer whole.
	var sharedPhones int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT phone FROM participants WHERE deleted_at IS NULL
			GROUP BY phone HAVING COUNT(*) > 1
		) s`).Scan(&sharedPhones); err != nil {
		t.Fatalf("count shared phones: %v", err)
	}
	if sharedPhones == 0 {
		t.Fatal("fixture produced no shared phones — the collapse would be untested")
	}
	var split int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT phone FROM participants WHERE deleted_at IS NULL
			GROUP BY phone HAVING COUNT(DISTINCT portal_user_id) > 1
		) s`).Scan(&split); err != nil {
		t.Fatalf("count split identities: %v", err)
	}
	if split != 0 {
		t.Errorf("%d phones map to more than one identity — a returning customer's history is split in two", split)
	}
}

// The identity keeps the most recent participant's credentials, which is what
// decides whose password still works after the migration.
func TestPortalUserBackfillKeepsTheNewestCredential(t *testing.T) {
	db := scratchDB(t)
	ctx := context.Background()

	if _, err := migrate.Run(ctx, db, upTo(t, "006")); err != nil {
		t.Fatalf("migrate to 006: %v", err)
	}
	seedParticipantsForBackfill(t, db)
	if _, err := migrate.Run(ctx, db, projectdb.Migrations()); err != nil {
		t.Fatalf("migrate the rest: %v", err)
	}

	var mismatched int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM portal_users pu
		WHERE pu.password_hash <> (
			SELECT p.portal_password FROM participants p
			WHERE p.phone = pu.phone AND p.deleted_at IS NULL
			ORDER BY p.created_at DESC LIMIT 1
		)`).Scan(&mismatched); err != nil {
		t.Fatalf("compare credentials: %v", err)
	}
	if mismatched != 0 {
		t.Errorf("%d identities did not take the newest participant's password — those customers are locked out",
			mismatched)
	}
}

// 009 lifts the gateway order off the invoice onto its own table. An invoice
// that never had one must not gain an empty row.
func TestGatewayOrderBackfillMovesOnlyRealSessions(t *testing.T) {
	db := scratchDB(t)
	ctx := context.Background()

	if _, err := migrate.Run(ctx, db, upTo(t, "008")); err != nil {
		t.Fatalf("migrate to 008: %v", err)
	}

	stamp := time.Now().UnixNano()
	var adminID, pkgID, batchID string
	mustExec(t, db.QueryRowContext(ctx, `
		INSERT INTO users (name, email, password, role)
		VALUES ('Admin', $1, 'x', 'super_admin') RETURNING id`,
		fmt.Sprintf("admin%d@uji.local", stamp)).Scan(&adminID))
	mustExec(t, db.QueryRowContext(ctx, `
		INSERT INTO packages (name, slug, destination, duration_days, base_price, created_by)
		VALUES ('Paket', $1, 'Jepang', 5, 1000000, $2) RETURNING id`,
		fmt.Sprintf("paket-%d", stamp), adminID).Scan(&pkgID))
	mustExec(t, db.QueryRowContext(ctx, `
		INSERT INTO package_batches (package_id, departure_date, return_date, quota,
			price_single, price_double, price_triple, status)
		VALUES ($1, CURRENT_DATE + 30, CURRENT_DATE + 35, 900, 1, 1, 1, 'tersedia')
		RETURNING id`, pkgID).Scan(&batchID))

	withOrder := 0
	for i := 0; i < 200; i++ {
		var paxID string
		mustExec(t, db.QueryRowContext(ctx, `
			INSERT INTO participants (batch_id, name, phone, email, room_type,
				portal_password, is_active)
			VALUES ($1, $2, $3, 'p@uji.local', 'double', 'x', true) RETURNING id`,
			batchID, fmt.Sprintf("Peserta %03d", i), fmt.Sprintf("6287%08d", i)).Scan(&paxID))

		// Only every third invoice ever opened a payment session.
		var order any
		if i%3 == 0 {
			order = fmt.Sprintf("ORDER-%06d", i)
			withOrder++
		}
		mustExec(t, db.QueryRowContext(ctx, `
			INSERT INTO invoices (invoice_number, participant_id, batch_id, amount, due_date,
				status, issued_by, midtrans_order_id, snap_token)
			VALUES ($1, $2, $3, 1000000, CURRENT_DATE + 7, 'menunggu_bayar', $4, $5, 'tok')
			RETURNING id`, fmt.Sprintf("INV-%06d", i), paxID, batchID, adminID, order).Scan(new(string)))
	}

	if _, err := migrate.Run(ctx, db, projectdb.Migrations()); err != nil {
		t.Fatalf("migrate the rest: %v", err)
	}

	var moved int
	if err := db.QueryRow(`SELECT COUNT(*) FROM invoice_gateway_orders`).Scan(&moved); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if moved != withOrder {
		t.Errorf("invoice_gateway_orders = %d, want %d — only invoices with a session should have moved",
			moved, withOrder)
	}

	var orphaned int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM invoice_gateway_orders o
		LEFT JOIN invoices i ON i.id = o.invoice_id
		WHERE i.id IS NULL OR o.order_id IS NULL OR o.order_id = ''`).Scan(&orphaned); err != nil {
		t.Fatalf("check orders: %v", err)
	}
	if orphaned != 0 {
		t.Errorf("%d moved sessions point at nothing — a notification for them could not be resolved", orphaned)
	}
}
