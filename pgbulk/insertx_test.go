package pgbulk_test

import (
	"context"
	"testing"

	"github.com/kaichao/gopkg/pgbulk"
)

func TestInsertReturningID(t *testing.T) {
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

	// Call InsertReturningID function
	ids, err := pgbulk.InsertReturningID(conn, sqlTemplate, data)
	if err != nil {
		t.Errorf("InsertReturningID failed: %v", err)
	}

	// Verify returned ID count
	if len(ids) != 2 {
		t.Errorf("Expected to return 2 IDs, got %d", len(ids))
	}

	// Verify each returned ID exists in database
	for _, id := range ids {
		var exists bool
		err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM test_table WHERE id = $1)", id).Scan(&exists)
		if err != nil || !exists {
			t.Errorf("ID %d not found in database", id)
		}
	}
}

func TestInsertReturningIDWithOnConflict(t *testing.T) {
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
	ids1, err := pgbulk.InsertReturningID(conn, sqlTemplate, data1)
	if err != nil {
		t.Fatalf("First insert failed: %v", err)
	}
	if len(ids1) != 2 {
		t.Fatalf("Expected first insert to return 2 IDs, got %d", len(ids1))
	}

	// Try to insert duplicate data with ON CONFLICT DO NOTHING
	data2 := [][]interface{}{
		{"alice@example.com", "Alice Updated"}, // duplicate email
		{"charlie@example.com", "Charlie"},     // new email
	}

	// Test ON CONFLICT DO NOTHING
	ids2, err := pgbulk.InsertReturningID(conn, sqlTemplate, data2, "id", "ON CONFLICT (email) DO NOTHING")
	if err != nil {
		t.Fatalf("Insert with ON CONFLICT DO NOTHING failed: %v", err)
	}

	// Should return only 1 ID (charlie@example.com)
	if len(ids2) != 1 {
		t.Errorf("Insert with ON CONFLICT DO NOTHING expected to return 1 ID, got %d", len(ids2))
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

	// Test custom return column and ON CONFLICT DO UPDATE
	ids3, err := pgbulk.InsertReturningID(conn, sqlTemplate, data3, "id", "ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name")
	if err != nil {
		t.Fatalf("Insert with ON CONFLICT DO UPDATE failed: %v", err)
	}

	// Should return 2 IDs (alice@example.com updated, david@example.com inserted)
	if len(ids3) != 2 {
		t.Errorf("Insert with ON CONFLICT DO UPDATE expected to return 2 IDs, got %d", len(ids3))
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
