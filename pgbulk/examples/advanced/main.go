// Advanced pgbulk usage examples
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/pgbulk"
)

func main() {
	fmt.Println("=== Advanced pgbulk Examples ===")

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgres://user:password@localhost/dbname")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	fmt.Println("\n1. Large Batch Processing with Error Handling:")
	largeBatchExample(conn)

	fmt.Println("\n2. Complex Update with Multiple Conditions:")
	complexUpdateExample(conn)

	fmt.Println("\n3. Bulk Operations with Transaction Control:")
	transactionExample(conn)

	fmt.Println("\n4. Insert with ON CONFLICT and Returning ID:")
	upsertReturningExample(conn)

	fmt.Println("\nAll advanced examples completed.")
}

func largeBatchExample(conn *pgx.Conn) {
	// Simulate large dataset
	var data [][]interface{}
	for i := 0; i < 100; i++ {
		data = append(data, []interface{}{fmt.Sprintf("User%d", i), fmt.Sprintf("user%d@example.com", i)})
	}

	sqlTemplate := "INSERT INTO users (name, email)"
	affected, err := pgbulk.Copy(conn, sqlTemplate, data)
	if err != nil {
		log.Printf("Large batch Copy error: %v", err)
		return
	}
	fmt.Printf("  Copied %d rows in large batch\n", affected)
}

func complexUpdateExample(conn *pgx.Conn) {
	// Example from update_example_test.go
	sqlTemplate := "UPDATE employees SET name = $1, salary = $2 WHERE id = $3 AND department = $4"

	data := [][]interface{}{
		{"Alice Updated", 75000},
		{"Bob Updated", 85000},
		{"Charlie Updated", 90000},
	}

	ids := [][]interface{}{
		{1, "Engineering"},
		{2, "Engineering"},
		{3, "Sales"},
	}

	failedIDs, err := pgbulk.Update(conn, sqlTemplate, data, ids)
	if err != nil {
		log.Printf("Complex update error: %v", err)
		return
	}

	if len(failedIDs) > 0 {
		fmt.Printf("  Complex update completed with %d failed IDs: %v\n", len(failedIDs), failedIDs)
	} else {
		fmt.Println("  Complex update completed successfully")
	}
}

func transactionExample(conn *pgx.Conn) {
	ctx := context.Background()

	// Start transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	// Insert data within transaction
	sqlTemplate := "INSERT INTO audit_log (action, details)"
	data := [][]interface{}{
		{"login", "User logged in"},
		{"logout", "User logged out"},
	}

	// Note: pgbulk functions work with *pgx.Conn, not *pgx.Tx
	// For transaction support, you'd need to use the connection
	// In practice, you'd pass the transaction's connection
	err = pgbulk.Insert(conn, sqlTemplate, data)
	if err != nil {
		log.Printf("Transaction insert error: %v", err)
		return
	}

	// Update within same transaction
	updateSQL := "UPDATE users SET last_login = NOW() WHERE id = $1"
	updateData := [][]interface{}{{1}, {2}}
	updateIDs := [][]interface{}{{1001}, {1002}}

	failedIDs, err := pgbulk.Update(conn, updateSQL, updateData, updateIDs)
	if err != nil {
		log.Printf("Transaction update error: %v", err)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return
	}

	fmt.Printf("  Transaction completed. Failed updates: %v\n", failedIDs)
}

func upsertReturningExample(conn *pgx.Conn) {
	sqlTemplate := "INSERT INTO products (sku, name, price)"
	data := [][]interface{}{
		{"SKU001", "Product A", 99.99},
		{"SKU002", "Product B", 149.99},
	}

	// Insert with ON CONFLICT clause and return IDs
	onConflict := "ON CONFLICT (sku) DO UPDATE SET name = EXCLUDED.name, price = EXCLUDED.price"
	ids, err := pgbulk.InsertReturningID(conn, sqlTemplate, data, "id", onConflict)
	if err != nil {
		log.Printf("Upsert returning error: %v", err)
		return
	}

	fmt.Printf("  Upsert completed. Affected IDs: %v\n", ids)

	// Test with duplicate SKUs (should update existing)
	duplicateData := [][]interface{}{
		{"SKU001", "Product A Updated", 109.99},
		{"SKU003", "Product C", 199.99},
	}

	ids, err = pgbulk.InsertReturningID(conn, sqlTemplate, duplicateData, "id", onConflict)
	if err != nil {
		log.Printf("Duplicate upsert error: %v", err)
		return
	}

	fmt.Printf("  Duplicate upsert completed. Affected IDs: %v\n", ids)
}
