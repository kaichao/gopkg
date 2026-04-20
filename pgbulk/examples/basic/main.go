// Basic pgbulk usage examples
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/pgbulk"
)

func main() {
	fmt.Println("=== Basic pgbulk Examples ===")

	// Connect to PostgreSQL (replace with your actual connection string)
	// For demonstration, we'll create a mock example
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgres://user:password@localhost/dbname")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	fmt.Println("\n1. Bulk Insert using COPY command:")
	copyExample(conn)

	fmt.Println("\n2. Regular Insert:")
	insertExample(conn)

	fmt.Println("\n3. Insert Returning IDs:")
	insertReturningExample(conn)

	fmt.Println("\n4. Bulk Update:")
	updateExample(conn)

	fmt.Println("\nAll examples completed.")
}

func copyExample(conn *pgx.Conn) {
	sqlTemplate := "INSERT INTO users (name, email)"
	data := [][]interface{}{
		{"Alice", "alice@example.com"},
		{"Bob", "bob@example.com"},
	}

	affected, err := pgbulk.Copy(conn, sqlTemplate, data)
	if err != nil {
		log.Printf("Copy error: %v", err)
		return
	}
	fmt.Printf("  Copied %d rows\n", affected)
}

func insertExample(conn *pgx.Conn) {
	sqlTemplate := "INSERT INTO products (name, price)"
	data := [][]interface{}{
		{"Product A", 99.99},
		{"Product B", 149.99},
	}

	err := pgbulk.Insert(conn, sqlTemplate, data)
	if err != nil {
		log.Printf("Insert error: %v", err)
		return
	}
	fmt.Println("  Insert completed")

	// With ON CONFLICT clause
	conflictClause := "ON CONFLICT (name) DO UPDATE SET price = EXCLUDED.price"
	err = pgbulk.Insert(conn, "INSERT INTO products (name, price)", data, conflictClause)
	if err != nil {
		log.Printf("Insert with conflict clause error: %v", err)
		return
	}
	fmt.Println("  Insert with ON CONFLICT completed")
}

func insertReturningExample(conn *pgx.Conn) {
	sqlTemplate := "INSERT INTO orders (product_id, quantity)"
	data := [][]interface{}{
		{1, 2},
		{2, 5},
	}

	ids, err := pgbulk.InsertReturningID(conn, sqlTemplate, data)
	if err != nil {
		log.Printf("InsertReturningID error: %v", err)
		return
	}
	fmt.Printf("  Inserted IDs: %v\n", ids)

	// With custom returning column
	ids, err = pgbulk.InsertReturningID(conn, sqlTemplate, data, "order_id")
	if err != nil {
		log.Printf("InsertReturningID with custom column error: %v", err)
		return
	}
	fmt.Printf("  Inserted IDs with custom column: %v\n", ids)
}

func updateExample(conn *pgx.Conn) {
	sqlTemplate := "UPDATE products SET price = $1 WHERE id = $2"
	data := [][]interface{}{
		{109.99}, // new price for product 1
		{159.99}, // new price for product 2
	}
	ids := [][]interface{}{
		{1}, // product ID 1
		{2}, // product ID 2
	}

	failedIDs, err := pgbulk.Update(conn, sqlTemplate, data, ids)
	if err != nil {
		log.Printf("Update error: %v", err)
		return
	}
	fmt.Printf("  Update completed. Failed IDs: %v\n", failedIDs)
}
