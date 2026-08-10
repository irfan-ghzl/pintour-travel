// Package migrate applies the SQL migrations a database has not seen yet.
//
// The project used to have no runner at all. Migrations reached a database only
// through the Postgres image's init directory, which runs once — when the data
// volume is first created — so a deployment that already held data could never
// receive a new one. The only documented path forward wiped the database. That
// is how a live deployment ended up serving migration 007 while the code
// expected 009, and every feature added by the missing two failed at runtime.
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// ledgerDDL records which migrations ran. It is itself created outside the
// ledger, since there is nowhere to record it until it exists.
const ledgerDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT        PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

// SQLSTATE codes meaning "this object is already here" — the signature of a
// migration that ran before the ledger existed. Spelled out rather than pulled
// from a constants package so the dependency stays at pgconn alone. Indexes are
// relations in Postgres, so a duplicate index reports duplicate_table too.
var alreadyExists = map[string]bool{
	"42P07": true, // duplicate_table (covers indexes)
	"42701": true, // duplicate_column
	"42710": true, // duplicate_object (covers constraints)
}

// Run applies every migration in src the database has not recorded, in file
// order, and returns the names it applied.
//
// Each migration runs inside its own transaction: one that fails leaves neither
// half-changed schema nor a ledger row claiming it succeeded, so a failed run
// can be fixed and repeated. Running Run twice in a row is a no-op the second
// time.
func Run(ctx context.Context, db *sql.DB, src fs.FS) ([]string, error) {
	if _, err := db.ExecContext(ctx, ledgerDDL); err != nil {
		return nil, fmt.Errorf("create migration ledger: %w", err)
	}

	files, err := list(src)
	if err != nil {
		return nil, err
	}
	done, err := applied(ctx, db)
	if err != nil {
		return nil, err
	}

	// Whether the ledger is being filled for the first time on a database that
	// already has objects. Only then may an "already exists" failure be read as
	// evidence of an earlier run rather than as a broken migration.
	adopting, err := needsAdoption(ctx, db, done)
	if err != nil {
		return nil, err
	}

	var ran []string
	for _, name := range files {
		if done[name] {
			continue
		}
		body, err := fs.ReadFile(src, name)
		if err != nil {
			return ran, fmt.Errorf("read %s: %w", name, err)
		}
		switch err := apply(ctx, db, name, string(body)); {
		case err == nil:
			ran = append(ran, name)
		case adopting && isAlreadyExists(err):
			// The objects are already there, from the init-directory run that
			// predates this ledger. Record it and move on; re-running the file
			// would fail on the first CREATE and block every later migration.
			if err := record(ctx, db, name); err != nil {
				return ran, err
			}
			log.Printf("migrate: %s already present, adopted into ledger", name)
		default:
			return ran, fmt.Errorf("apply %s: %w", name, err)
		}
	}
	return ran, nil
}

// apply runs one migration and records it in the same transaction, so the file
// and the fact that it ran commit together or not at all.
func apply(ctx context.Context, db *sql.DB, name, body string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	if _, err := tx.ExecContext(ctx, body); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return err
	}
	return tx.Commit()
}

func record(ctx context.Context, db *sql.DB, name string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, name)
	if err != nil {
		return fmt.Errorf("record %s: %w", name, err)
	}
	return nil
}

// downSuffix marks the rollback counterpart of a migration: 008_x.sql is undone
// by 008_x.down.sql. Keeping the pair next to each other means a migration that
// can be rolled back says so by existing, and one that cannot says so by its
// absence — there is no separate list to keep in step.
const downSuffix = ".down.sql"

// list returns the forward migration file names in the order they must run.
// Sorting the names is the ordering: the files are numbered, and a new one is
// added by picking the next number. Rollback files are not migrations and are
// skipped here.
func list(src fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		if strings.HasSuffix(e.Name(), downSuffix) {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, errors.New("no migrations found")
	}
	return names, nil
}

// ErrNoRollback reports that the most recent migration ships no rollback, so
// undoing it is a decision someone has to make by hand rather than one this
// command can take.
var ErrNoRollback = errors.New("migration has no rollback file")

// Down undoes the most recently applied migration and returns its name.
//
// One step at a time, on purpose: each rollback here discards data (see the
// notes in the .down.sql files), so unwinding several is a sequence of separate
// decisions rather than one flag. The rollback and the ledger row are removed in
// the same transaction, so the database never claims a migration is applied
// after its objects are gone.
func Down(ctx context.Context, db *sql.DB, src fs.FS) (string, error) {
	var name string
	err := db.QueryRowContext(ctx,
		`SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("no migrations applied")
	}
	if err != nil {
		return "", fmt.Errorf("read migration ledger: %w", err)
	}

	body, err := fs.ReadFile(src, strings.TrimSuffix(name, ".sql")+downSuffix)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrNoRollback, name)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return "", fmt.Errorf("roll back %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM schema_migrations WHERE version = $1`, name); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return name, nil
}

func applied(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read migration ledger: %w", err)
	}
	defer rows.Close()

	done := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		done[v] = true
	}
	return done, rows.Err()
}

// needsAdoption reports whether this database predates the ledger: it holds
// application tables but has recorded no migrations. A fresh database has
// neither, and a database already under the ledger has both.
func needsAdoption(ctx context.Context, db *sql.DB, done map[string]bool) (bool, error) {
	if len(done) > 0 {
		return false, nil
	}
	var exists bool
	err := db.QueryRowContext(ctx,
		`SELECT to_regclass('public.users') IS NOT NULL`).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("probe existing schema: %w", err)
	}
	return exists, nil
}

func isAlreadyExists(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return alreadyExists[pgErr.Code]
	}
	return false
}
