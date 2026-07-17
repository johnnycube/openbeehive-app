// Package sqlstore implements storage.Store for PostgreSQL, MySQL and SQLite.
//
// A single codebase serves all three dialects:
//   - placeholders are written generically as "?" and translated by sqlx.Rebind
//     into the driver's bindvar style ($1 for Postgres, ? for MySQL/SQLite)
//     .
//   - The schema uses only portable types (TEXT, INTEGER, REAL, TIMESTAMP),
//     IDs are string UUIDs - no dialect-specific AUTO_INCREMENT/SERIAL.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/johnnycube/openbeehive-app/server/internal/config"
	"github.com/johnnycube/openbeehive-app/server/internal/storage"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	db     *sqlx.DB
	driver config.DBDriver
}

func Open(cfg *config.Config) (*Store, error) {
	driverName := map[config.DBDriver]string{
		config.DriverPostgres: "pgx",
		config.DriverMySQL:    "mysql",
		config.DriverSQLite:   "sqlite",
	}[cfg.DB.Driver]
	if driverName == "" {
		return nil, fmt.Errorf("unknown driver %q", cfg.DB.Driver)
	}

	db, err := sqlx.Open(driverName, cfg.DB.DSN)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}
	// SQLite tolerates only one writer.
	if cfg.DB.Driver == config.DriverSQLite {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(25)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	return &Store{db: db, driver: cfg.DB.Driver}, nil
}

func (s *Store) Apiaries() storage.ApiaryRepo { return &apiaryRepo{s} }
func (s *Store) Hives() storage.HiveRepo       { return &hiveRepo{s} }
func (s *Store) Users() storage.UserRepo       { return &userRepo{s} }
func (s *Store) Orgs() storage.OrgRepo         { return &orgRepo{s} }
func (s *Store) Members() storage.MemberRepo   { return &memberRepo{s} }
func (s *Store) Invites() storage.InviteRepo   { return &inviteRepo{s} }
func (s *Store) Close() error                    { return s.db.Close() }

// DB exposes the sqlx handle (used by the sync service).
func (s *Store) DB() *sqlx.DB { return s.db }

// Migrate runs all embedded .sql files exactly once.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMP)`); err != nil {
		return err
	}
	entries, _ := migrationsFS.ReadDir("migrations")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var dummy string
		err := s.db.GetContext(ctx, &dummy,
			s.db.Rebind(`SELECT version FROM schema_migrations WHERE version = ?`), name)
		if err == nil {
			continue // already applied
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		raw, _ := migrationsFS.ReadFile("migrations/" + name)
		// run statements individually - not every driver executes
		// multiple statements in one Exec (e.g. MySQL without multiStatements).
		for _, stmt := range splitStatements(string(raw)) {
			if _, err := s.db.ExecContext(ctx, s.translate(stmt)); err != nil {
				return fmt.Errorf("migration %s: %w", name, err)
			}
		}
		if _, err := s.db.ExecContext(ctx,
			s.db.Rebind(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`),
			name, time.Now().UTC()); err != nil {
			return err
		}
	}
	return nil
}

// translate minimally adapts the portable migration statements to dialects.
// The base schema is deliberately portable; only a few tokens differ.
func (s *Store) translate(stmt string) string {
	switch s.driver {
	case config.DriverPostgres:
		// BOOLEAN is native; nothing to do
		return stmt
	case config.DriverMySQL:
		r := strings.NewReplacer(
			"TEXT PRIMARY KEY", "VARCHAR(64) PRIMARY KEY",
			"BOOLEAN", "TINYINT(1)",
		)
		return r.Replace(stmt)
	case config.DriverSQLite:
		// SQLite treats BOOLEAN as an alias and TIMESTAMP as TEXT/NUMERIC - ok.
		return stmt
	}
	return stmt
}

// helper: Rebind + Exec
func (s *Store) exec(ctx context.Context, q string, args ...any) error {
	_, err := s.db.ExecContext(ctx, s.db.Rebind(q), args...)
	return err
}

// splitStatements splits a .sql file into individual statements (on ";"),
// strips line comments (--) and empty fragments. Comments are stripped
// BEFORE splitting so a ";" inside a comment does not break a statement.
func splitStatements(sql string) []string {
	var clean strings.Builder
	for _, line := range strings.Split(sql, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		clean.WriteString(line)
		clean.WriteString("\n")
	}
	var out []string
	for _, part := range strings.Split(clean.String(), ";") {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}
