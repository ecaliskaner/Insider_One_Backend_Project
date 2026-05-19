package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the sql.DB connection
type DB struct {
	Conn *sql.DB
}

// DBTX is the subset of database/sql used by repositories.
// Both *sql.DB and *sql.Tx implement it, which keeps repository code reusable
// for normal reads and transactional writes.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// NewDB creates a new database connection and initializes the schema
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{Conn: conn}

	log.Println("Database connection established")
	return db, nil
}

// RunMigrations runs database migrations using golang-migrate
func (db *DB) RunMigrations() error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	dbDriver, err := sqlite3.WithInstance(db.Conn, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations up: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.Conn.Close()
}
