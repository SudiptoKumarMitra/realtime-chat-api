package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations reads .sql files from the migrations directory
// and executes them in order. Uses CREATE IF NOT EXISTS, so safe
// to run on every startup.
func RunMigrations(db *sql.DB) error {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("no migrations directory found, skipping")
			return nil
		}
		return fmt.Errorf("read migrations directory: %w", err)
	}

	// Sort by filename to ensure order (001_, 002_, etc.).
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := filepath.Join("migrations", entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("execute migration %s: %w", entry.Name(), err)
		}

		log.Printf("migration applied: %s", entry.Name())
	}

	return nil
}
