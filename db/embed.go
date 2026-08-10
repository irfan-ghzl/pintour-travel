// Package db carries the SQL migrations into the binary.
//
// They are embedded rather than read from disk because the runtime image copies
// only the compiled server (see Dockerfile): a deployment has no db/ directory
// to read, so a migration that lives only on disk can never reach the database
// it is meant to change.
package db

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrations returns the migration files rooted so that each entry is the bare
// file name — "001_init.sql" rather than "migrations/001_init.sql".
func Migrations() fs.FS {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		// Unreachable: the embed directive above fixes the directory at compile
		// time, so it either exists in the binary or the build failed.
		panic(err)
	}
	return sub
}
