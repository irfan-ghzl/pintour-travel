package migrate_test

// These run against a real Postgres — the "second seam" the spec describes.
// Nothing here can be proved with a fake: what is under test is whether SQL the
// project ships actually applies, and whether a database that already holds
// objects is left intact.
//
// Set TEST_DATABASE_URL to a reachable server to run them; without it they skip,
// so a machine with no database still gets a green suite.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	projectdb "github.com/irfan-ghzl/pintour-travel/db"
	"github.com/irfan-ghzl/pintour-travel/internal/migrate"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// scratchDB creates an empty database for one test and drops it afterwards, so
// tests neither see each other's schema nor touch a developer's data.
func scratchDB(t *testing.T) *sql.DB {
	t.Helper()

	adminDSN := os.Getenv("TEST_DATABASE_URL")
	if adminDSN == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping database-backed migration tests")
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

	name := fmt.Sprintf("migtest_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		admin.Close()
		t.Fatalf("create scratch database: %v", err)
	}

	conn, err := sql.Open("pgx", replaceDBName(adminDSN, name))
	if err != nil {
		t.Fatalf("open scratch database: %v", err)
	}
	t.Cleanup(func() {
		conn.Close()
		if _, err := admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)"); err != nil {
			t.Logf("drop scratch database %s: %v", name, err)
		}
		admin.Close()
	})
	return conn
}

// replaceDBName swaps the database in a postgres URL, keeping credentials,
// host, and query string.
func replaceDBName(dsn, name string) string {
	slash := len(dsn)
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '/' {
			slash = i
			break
		}
	}
	tail := ""
	for i := slash; i < len(dsn); i++ {
		if dsn[i] == '?' {
			tail = dsn[i:]
			break
		}
	}
	return dsn[:slash+1] + name + tail
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRow(`SELECT to_regclass('public.' || $1) IS NOT NULL`, name).Scan(&exists); err != nil {
		t.Fatalf("probe %s: %v", name, err)
	}
	return exists
}

func TestRunAppliesEveryMigrationToAnEmptyDatabase(t *testing.T) {
	db := scratchDB(t)

	ran, err := migrate.Run(context.Background(), db, projectdb.Migrations())
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if len(ran) == 0 {
		t.Fatal("expected migrations to be applied to an empty database, got none")
	}

	// A few tables spread across the range, so the assertion fails loudly if the
	// runner stops partway rather than only checking the last file landed.
	for _, table := range []string{"users", "invoices", "portal_users", "lead_status_history", "invoice_gateway_orders"} {
		if !tableExists(t, db, table) {
			t.Errorf("table %q missing after migrating an empty database", table)
		}
	}
}

func TestRunIsANoOpTheSecondTime(t *testing.T) {
	db := scratchDB(t)
	ctx := context.Background()

	first, err := migrate.Run(ctx, db, projectdb.Migrations())
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := migrate.Run(ctx, db, projectdb.Migrations())
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("second run applied %v, want nothing (first run did %d)", second, len(first))
	}
}

func TestRunAdoptsASchemaThatPredatesTheLedger(t *testing.T) {
	db := scratchDB(t)
	ctx := context.Background()

	// Stand in for a volume built by the Postgres init directory: the objects are
	// there, but nothing recorded that they are.
	if _, err := migrate.Run(ctx, db, projectdb.Migrations()); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE schema_migrations`); err != nil {
		t.Fatalf("drop ledger: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (name, email, password, role) VALUES ('Adopsi','adopsi@test.local','x','admin')`); err != nil {
		t.Fatalf("insert row that must survive: %v", err)
	}

	if _, err := migrate.Run(ctx, db, projectdb.Migrations()); err != nil {
		t.Fatalf("adoption run: %v", err)
	}

	var rows int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE email='adopsi@test.local'`).Scan(&rows); err != nil {
		t.Fatalf("count surviving rows: %v", err)
	}
	if rows != 1 {
		t.Errorf("adoption lost data: found %d rows, want 1", rows)
	}

	var recorded int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&recorded); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if recorded == 0 {
		t.Error("adoption recorded nothing — the next run would try to apply everything again")
	}
}

func TestRunStopsAtAFailingMigrationWithoutRecordingIt(t *testing.T) {
	db := scratchDB(t)

	broken := fstest.MapFS{
		"001_ok.sql":     &fstest.MapFile{Data: []byte(`CREATE TABLE migrate_ok (id INT)`)},
		"002_broken.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE migrate_broken (id INT) ; SELECT bukan_fungsi()`)},
		"003_never.sql":  &fstest.MapFile{Data: []byte(`CREATE TABLE migrate_never (id INT)`)},
	}

	ran, err := migrate.Run(context.Background(), db, broken)
	if err == nil {
		t.Fatal("expected the broken migration to fail the run")
	}
	if len(ran) != 1 || ran[0] != "001_ok.sql" {
		t.Errorf("applied %v, want only 001_ok.sql before stopping", ran)
	}
	if tableExists(t, db, "migrate_broken") {
		t.Error("a failed migration left its table behind — it did not run in a transaction")
	}
	if tableExists(t, db, "migrate_never") {
		t.Error("a migration after the failing one ran anyway")
	}
}

// ─── Rollback ─────────────────────────────────────────────────────────────────

// The two migrations that carry schema for live features ship a rollback, so a
// deploy that goes wrong has a way back that does not start with restoring a
// backup. Each undoes exactly one step, because each discards something.
func TestDownUndoesTheMostRecentMigration(t *testing.T) {
	db := scratchDB(t)
	ctx := context.Background()

	if _, err := migrate.Run(ctx, db, projectdb.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !tableExists(t, db, "gateway_notifications") {
		t.Fatal("fixture is not at the version under test")
	}

	newest := latestMigration(t)
	name, err := migrate.Down(ctx, db, projectdb.Migrations())
	if err != nil {
		t.Fatalf("roll back: %v", err)
	}
	if name != newest {
		t.Errorf("rolled back %q, want the most recent migration %q", name, newest)
	}
	if recorded := count(t, db,
		`SELECT COUNT(*) FROM schema_migrations WHERE version = $1`, newest); recorded != 0 {
		t.Error("the ledger still claims the rolled-back migration is applied")
	}
	// Everything the earlier migrations built is still standing.
	for _, table := range []string{"users", "invoices", "lead_status_history"} {
		if !tableExists(t, db, table) {
			t.Errorf("rollback took %q with it — it belongs to an earlier migration", table)
		}
	}
}

// Rolling back and applying again must land on the same schema, or the rollback
// is a one-way door dressed up as a way back.
func TestDownThenUpRestoresTheSchema(t *testing.T) {
	db := scratchDB(t)
	ctx := context.Background()

	if _, err := migrate.Run(ctx, db, projectdb.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	before := schemaFingerprint(t, db)

	if _, err := migrate.Down(ctx, db, projectdb.Migrations()); err != nil {
		t.Fatalf("roll back: %v", err)
	}
	ran, err := migrate.Run(ctx, db, projectdb.Migrations())
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if len(ran) != 1 {
		t.Errorf("re-applied %v, want exactly the rolled-back migration", ran)
	}
	if after := schemaFingerprint(t, db); after != before {
		t.Error("schema after down-then-up differs from before the rollback")
	}
}

// Two steps back reaches the second rollback file, and the ledger keeps up.
func TestDownTwiceReachesTheMigrationBefore(t *testing.T) {
	db := scratchDB(t)
	ctx := context.Background()

	if _, err := migrate.Run(ctx, db, projectdb.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Unwind every migration that ships a rollback, one step at a time, and stop
	// at the first that does not. Written this way so adding a migration does not
	// break the test — what is under test is the sequence, not a version number.
	var undone []string
	for {
		name, err := migrate.Down(ctx, db, projectdb.Migrations())
		if errors.Is(err, migrate.ErrNoRollback) {
			break
		}
		if err != nil {
			t.Fatalf("roll back: %v", err)
		}
		undone = append(undone, name)
	}
	if len(undone) < 2 {
		t.Fatalf("only %v could be rolled back; the test needs at least two to prove the sequence", undone)
	}
	// Descending order: each step undoes the newest still applied.
	for i := 1; i < len(undone); i++ {
		if undone[i] >= undone[i-1] {
			t.Errorf("rolled back %q after %q — steps must go newest first", undone[i], undone[i-1])
		}
	}
	if tableExists(t, db, "lead_status_history") {
		t.Error("the audit-trail migration was rolled back but its table is still there")
	}
	for _, name := range undone {
		if count(t, db, `SELECT COUNT(*) FROM schema_migrations WHERE version = $1`, name) != 0 {
			t.Errorf("ledger still claims %q is applied", name)
		}
	}
}

// A migration with no rollback file says so rather than doing something
// approximate. Most of the early migrations have none: rolling back to an empty
// schema is a restore, not a migration.
func TestDownRefusesAMigrationWithoutARollback(t *testing.T) {
	db := scratchDB(t)
	ctx := context.Background()

	only := fstest.MapFS{
		"001_only.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE migrate_only (id INT)`)},
	}
	if _, err := migrate.Run(ctx, db, only); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := migrate.Down(ctx, db, only); !errors.Is(err, migrate.ErrNoRollback) {
		t.Errorf("err = %v, want ErrNoRollback", err)
	}
	if !tableExists(t, db, "migrate_only") {
		t.Error("a refused rollback changed the schema anyway")
	}
}

// Rollback files must not be mistaken for migrations to apply — that would drop
// the table the migration beside them just created.
func TestRunIgnoresRollbackFiles(t *testing.T) {
	db := scratchDB(t)

	ran, err := migrate.Run(context.Background(), db, projectdb.Migrations())
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, name := range ran {
		if strings.HasSuffix(name, ".down.sql") {
			t.Errorf("applied %q as a migration", name)
		}
	}
	if !tableExists(t, db, "lead_status_history") {
		t.Error("a rollback file ran during a forward migration")
	}
}

// schemaFingerprint is every table and column in the public schema, which is
// what "the same schema" has to mean for a rollback to be reversible.
func schemaFingerprint(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`
		SELECT table_name || '.' || column_name || ':' || data_type
		FROM information_schema.columns
		WHERE table_schema = 'public'
		ORDER BY 1`)
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	defer rows.Close()
	var sb strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("fingerprint scan: %v", err)
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// latestMigration is the highest-numbered forward migration the project ships —
// the one a rollback undoes first.
func latestMigration(t *testing.T) string {
	t.Helper()
	entries, err := fs.ReadDir(projectdb.Migrations(), ".")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	newest := ""
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, ".down.sql") {
			continue
		}
		if name > newest {
			newest = name
		}
	}
	if newest == "" {
		t.Fatal("no forward migrations found")
	}
	return newest
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}
