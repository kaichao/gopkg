package pgbulk_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// testDBURL is the connection string for test database
const testDBURL = "postgres://scalebox:secret@10.0.6.100/scalebox?sslmode=disable"

// getTestConn returns a test database connection
// If connection fails, test will fail directly
func getTestConn(t *testing.T) *pgx.Conn {
	t.Helper()

	conn, err := pgx.Connect(context.Background(), testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Register cleanup function
	t.Cleanup(func() {
		conn.Close(context.Background())
	})

	return conn
}

// setupTestTable creates test table, returns cleanup function
func setupTestTable(t *testing.T, conn *pgx.Conn, tableName, schema string) func() {
	t.Helper()

	// Drop table if exists
	dropSQL := "DROP TABLE IF EXISTS " + tableName
	_, err := conn.Exec(context.Background(), dropSQL)
	if err != nil {
		t.Fatalf("Failed to drop table %s: %v", tableName, err)
	}

	// Create table
	_, err = conn.Exec(context.Background(), schema)
	if err != nil {
		t.Fatalf("Failed to create table %s: %v", tableName, err)
	}

	// Return cleanup function
	return func() {
		_, err := conn.Exec(context.Background(), dropSQL)
		if err != nil {
			t.Logf("Warning: Failed to cleanup table %s: %v", tableName, err)
		}
	}
}
