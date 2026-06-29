// Package testutil provides test utilities for database and integration testing.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestDB provides isolated database connections for tests.
type TestDB struct {
	db   *sql.DB
	name string
}

// SetupTestDB creates a new test database connection and clears all tables.
func SetupTestDB(t *testing.T) *TestDB {
	dsn := "user=postgres password=dev host=localhost port=6432 dbname=ucp sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("test database ping failed: %v", err)
	}

	testdb := &TestDB{db: db, name: t.Name()}
	testdb.Clean(t)
	return testdb
}

// TeardownTestDB closes the test database connection.
func (t *TestDB) TeardownTestDB() error {
	if t.db != nil {
		return t.db.Close()
	}
	return nil
}

// Clean removes all records from tables in dependency order.
// This ensures foreign key constraints don't prevent cleanup.
func (t *TestDB) Clean(tb testing.TB) {
	tables := []string{
		"delivery_queue",
		"federation_bundle_log",
		"federation_connections",
		"key_shares",
		"mls_groups",
		"message_attachments",
		"attachments",
		"messages",
		"sessions",
		"key_packages",
		"identities",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, table := range tables {
		if _, err := t.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			tb.Logf("warning: failed to clean table %s: %v", table, err)
		}
	}
}

// DB returns the underlying *sql.DB connection.
func (t *TestDB) DB() *sql.DB {
	return t.db
}

// QueryRow executes a query and returns a single row.
func (t *TestDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.db.QueryRowContext(ctx, query, args...)
}

// Exec executes a query without returning rows.
func (t *TestDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.db.ExecContext(ctx, query, args...)
}

// Query executes a query returning multiple rows.
func (t *TestDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.db.QueryContext(ctx, query, args...)
}
