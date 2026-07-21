package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareMigrationDirExcludesManualSeed(t *testing.T) {
	sourceDir := t.TempDir()
	for _, name := range []string{"999_seed_data.up.sql", "1010_plans.up.sql"} {
		if err := os.WriteFile(filepath.Join(sourceDir, name), []byte("-- migration"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	migrationDir, cleanup, err := prepareMigrationDir("up", sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if _, err := os.Stat(filepath.Join(migrationDir, "999_seed_data.up.sql")); !os.IsNotExist(err) {
		t.Fatalf("manual seed migration should be excluded, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(migrationDir, "1010_plans.up.sql")); err != nil {
		t.Fatalf("runtime migration should be copied: %v", err)
	}
}
