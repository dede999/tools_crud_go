package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"

	"github.com/dede999/tools_crud_go/migrations"
)

// RunMigrations executes all SQL migration files in lexical order using the embedded migrations.FS
func RunMigrations(ctx context.Context, db *sql.DB) error {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to read embedded migrations: %w", err)
	}

	// Sort migrations in deterministic lexical order (e.g. 001_..., 002_...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "migrations.go" {
			continue
		}

		content, err := migrations.FS.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}
