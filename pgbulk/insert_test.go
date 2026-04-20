package pgbulk_test

import (
	"context"
	"testing"

	"github.com/kaichao/gopkg/pgbulk"
)

func TestInsert(t *testing.T) {
	conn := getTestConn(t)
	ctx := context.Background()

	// Create test table
	cleanup := setupTestTable(t, conn, "test_table", `
		CREATE TABLE test_table (
			id SERIAL PRIMARY KEY,
			col1 TEXT,
			col2 TEXT
		)
	`)
	defer cleanup()

	// Prepare test data
	sqlTemplate := "INSERT INTO test_table (col1, col2)"
	data := [][]interface{}{
		{"value1", "value2"},
		{"value3", "value4"},
	}

	// Call Insert function
	err := pgbulk.Insert(conn, sqlTemplate, data)
	if err != nil {
		t.Errorf("Insert failed: %v", err)
	}

	// Verify insertion result
	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM test_table").Scan(&count)
	if err != nil {
		t.Errorf("Query row count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected to insert 2 rows, got %d rows", count)
	}
}

func TestInsertWithOnConflict(t *testing.T) {
	conn := getTestConn(t)
	ctx := context.Background()

	// Create test table with unique constraint
	cleanup := setupTestTable(t, conn, "test_table_conflict", `
		CREATE TABLE test_table_conflict (
			id SERIAL PRIMARY KEY,
			email TEXT UNIQUE,
			name TEXT
		)
	`)
	defer cleanup()

	// First insert some data
	sqlTemplate := "INSERT INTO test_table_conflict (email, name)"
	data1 := [][]interface{}{
		{"alice@example.com", "Alice"},
		{"bob@example.com", "Bob"},
	}

	// Insert first batch of data
	err := pgbulk.Insert(conn, sqlTemplate, data1)
	if err != nil {
		t.Fatalf("First insert failed: %v", err)
	}

	// Try to insert duplicate data with ON CONFLICT DO NOTHING
	data2 := [][]interface{}{
		{"alice@example.com", "Alice Updated"}, // duplicate email
		{"charlie@example.com", "Charlie"},     // new email
	}

	// Test ON CONFLICT DO NOTHING
	err = pgbulk.Insert(conn, sqlTemplate, data2, "ON CONFLICT (email) DO NOTHING")
	if err != nil {
		t.Fatalf("Insert with ON CONFLICT DO NOTHING failed: %v", err)
	}

	// Verify data
	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM test_table_conflict").Scan(&count)
	if err != nil {
		t.Fatalf("Query row count failed: %v", err)
	}
	// Should have 3 rows: alice, bob, charlie (alice not duplicated)
	if count != 3 {
		t.Errorf("Expected 3 rows of data, got %d rows", count)
	}

	// Verify alice@example.com was not updated
	var aliceName string
	err = conn.QueryRow(ctx, "SELECT name FROM test_table_conflict WHERE email = 'alice@example.com'").Scan(&aliceName)
	if err != nil {
		t.Fatalf("Query alice data failed: %v", err)
	}
	if aliceName != "Alice" {
		t.Errorf("Expected alice's name to be 'Alice', got '%s'", aliceName)
	}

	// Test ON CONFLICT DO UPDATE
	data3 := [][]interface{}{
		{"alice@example.com", "Alice Updated Again"},
		{"david@example.com", "David"},
	}

	// Test ON CONFLICT DO UPDATE
	err = pgbulk.Insert(conn, sqlTemplate, data3, "ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name")
	if err != nil {
		t.Fatalf("Insert with ON CONFLICT DO UPDATE failed: %v", err)
	}

	// Verify data
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM test_table_conflict").Scan(&count)
	if err != nil {
		t.Fatalf("Query row count failed: %v", err)
	}
	// Should have 4 rows: alice(updated), bob, charlie, david(new)
	if count != 4 {
		t.Errorf("Expected 4 rows of data, got %d rows", count)
	}

	// Verify alice@example.com was updated
	err = conn.QueryRow(ctx, "SELECT name FROM test_table_conflict WHERE email = 'alice@example.com'").Scan(&aliceName)
	if err != nil {
		t.Fatalf("Query updated alice data failed: %v", err)
	}
	if aliceName != "Alice Updated Again" {
		t.Errorf("Expected alice's updated name to be 'Alice Updated Again', got '%s'", aliceName)
	}
}
