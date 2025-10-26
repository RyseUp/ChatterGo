package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/RyseUp/ChatterGo/config"
	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/RyseUp/ChatterGo/pkg/database"
	"gorm.io/gorm"
)

type migration struct {
	filename string
	upSQL    string
	downSQL  string
}

func main() {
	dir := flag.String("dir", "migration", "directory containing .sql migration files")
	down := flag.Bool("down", false, "apply down migrations instead of up")
	steps := flag.Int("steps", 0, "number of steps to migrate (0 = all)")
	newName := flag.String("new", "", "create a new timestamped migration file with this snake_case name")
	auto := flag.Bool("automigrate", false, "run GORM AutoMigrate for all models")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fatalf("failed to load config: %v", err)
	}

	db, err := database.Open(cfg.Database.GetDatabaseDSN())
	if err != nil {
		fatalf("failed to connect database: %v", err)
	}

	if err := ensureMigrationsTable(db); err != nil {
		fatalf("failed to ensure schema_migrations: %v", err)
	}

	if *newName != "" {
		path, err := createNewMigration(*dir, *newName)
		if err != nil {
			fatalf("failed to create migration: %v", err)
		}
		fmt.Println("created:", path)
		return
	}

	if *auto {
		fmt.Println("running AutoMigrate for registered models...")
		if err := db.AutoMigrate(models.All()...); err != nil {
			fatalf("automigrate failed: %v", err)
		}
		fmt.Println("automigrate complete")
		return
	}

	files, err := filepath.Glob(filepath.Join(*dir, "*.sql"))
	if err != nil {
		fatalf("failed to list migration files: %v", err)
	}
	sort.Strings(files)

	if len(files) == 0 {
		fmt.Println("no migration files found")
		return
	}

	if *down {
		if err := migrateDown(db, files, *steps); err != nil {
			fatalf("down migration failed: %v", err)
		}
		return
	}

	if err := migrateUp(db, files, *steps); err != nil {
		fatalf("up migration failed: %v", err)
	}
}

func fatalf(format string, a ...any) {
	fmt.Printf(format+"\n", a...)
	os.Exit(1)
}

func ensureMigrationsTable(db *gorm.DB) error {
	stmt := `CREATE TABLE IF NOT EXISTS schema_migrations (
        id SERIAL PRIMARY KEY,
        filename TEXT UNIQUE NOT NULL,
        applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )`
	return db.Exec(stmt).Error
}

func loadApplied(db *gorm.DB) (map[string]bool, []string, error) {
	type row struct{ Filename string }
	var rows []row
	if err := db.Raw("SELECT filename FROM schema_migrations ORDER BY applied_at").Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	m := make(map[string]bool, len(rows))
	order := make([]string, 0, len(rows))
	for _, r := range rows {
		m[r.Filename] = true
		order = append(order, r.Filename)
	}
	return m, order, nil
}

func parseMigrationFile(path string) (*migration, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // up to 1MB lines

	const upMarker = "-- +migrate Up"
	const downMarker = "-- +migrate Down"
	var (
		inUp    bool
		inDown  bool
		upBuf   strings.Builder
		downBuf strings.Builder
	)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, upMarker) {
			inUp = true
			inDown = false
			continue
		}
		if strings.HasPrefix(line, downMarker) {
			inDown = true
			inUp = false
			continue
		}
		if inUp {
			upBuf.WriteString(line)
			upBuf.WriteString("\n")
		} else if inDown {
			downBuf.WriteString(line)
			downBuf.WriteString("\n")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	m := &migration{
		filename: filepath.Base(path),
		upSQL:    strings.TrimSpace(upBuf.String()),
		downSQL:  strings.TrimSpace(downBuf.String()),
	}
	if m.upSQL == "" {
		return nil, errors.New("missing Up section")
	}
	return m, nil
}

func execStatements(db *gorm.DB, sql string) error {
	// Execute possibly multi-statement SQL; fall back to naive split by ';'
	if err := db.Exec(sql).Error; err == nil {
		return nil
	}
	// naive split if driver disallows multi statements
	stmts := strings.Split(sql, ";")
	for _, s := range stmts {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

func migrateUp(db *gorm.DB, files []string, steps int) error {
	applied, _, err := loadApplied(db)
	if err != nil {
		return err
	}

	count := 0
	for _, path := range files {
		name := filepath.Base(path)
		if applied[name] {
			continue
		}
		m, err := parseMigrationFile(path)
		if err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}
		fmt.Printf("Applying %s ...\n", name)
		start := time.Now()
		if err := execStatements(db, m.upSQL); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if err := db.Exec("INSERT INTO schema_migrations (filename) VALUES (?)", name).Error; err != nil {
			return fmt.Errorf("track %s: %w", name, err)
		}
		fmt.Printf("Applied %s in %s\n", name, time.Since(start).Round(time.Millisecond))
		count++
		if steps > 0 && count >= steps {
			break
		}
	}
	if count == 0 {
		fmt.Println("no pending migrations")
	}
	return nil
}

func migrateDown(db *gorm.DB, files []string, steps int) error {
	applied, order, err := loadApplied(db)
	if err != nil {
		return err
	}
	if len(order) == 0 {
		fmt.Println("no applied migrations")
		return nil
	}
	// Build a map from filename to path for lookup
	pathByName := make(map[string]string, len(files))
	for _, p := range files {
		pathByName[filepath.Base(p)] = p
	}

	count := 0
	// Iterate in reverse application order
	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		if !applied[name] {
			continue
		}
		path, ok := pathByName[name]
		if !ok {
			return fmt.Errorf("migration file missing for %s", name)
		}
		m, err := parseMigrationFile(path)
		if err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}
		if strings.TrimSpace(m.downSQL) == "" {
			return fmt.Errorf("missing Down section for %s", name)
		}
		fmt.Printf("Reverting %s ...\n", name)
		start := time.Now()
		if err := execStatements(db, m.downSQL); err != nil {
			return fmt.Errorf("revert %s: %w", name, err)
		}
		if err := db.Exec("DELETE FROM schema_migrations WHERE filename = ?", name).Error; err != nil {
			return fmt.Errorf("untrack %s: %w", name, err)
		}
		fmt.Printf("Reverted %s in %s\n", name, time.Since(start).Round(time.Millisecond))
		count++
		if steps > 0 && count >= steps {
			break
		}
	}
	if count == 0 {
		fmt.Println("nothing to revert")
	}
	return nil
}

func createNewMigration(dir, name string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	now := time.Now().UTC()
	ts := now.Format("20060102150405")
	base := fmt.Sprintf("%s_%s.sql", ts, sanitizeName(name))
	path := filepath.Join(dir, base)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("file already exists: %s", path)
	}
	created := now.Format("2006-01-02 15:04:05")
	tmpl := fmt.Sprintf(`-- Migration: %s
-- Created: %s UTC

-- +migrate Up
-- Write your SQL here

-- +migrate Down
-- Write the rollback SQL here
`, name, created)
	if err := os.WriteFile(path, []byte(tmpl), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	// Replace spaces with underscores and drop disallowed chars
	b := strings.Builder{}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "migration"
	}
	return out
}
