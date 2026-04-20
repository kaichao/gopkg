// Package pgbulk provides lightweight PostgreSQL bulk operations for Go with performance optimizations.
//
// Features:
// - Batch Processing: Automatically chunks large datasets into optimal batches
// - SQL Templates: Reusable templates with dynamic placeholders
// - Full CRUD Support: INSERT, UPDATE, and INSERT...RETURNING operations
// - PG-Compatible: Respects PostgreSQL's parameter limits
//
// Use Cases:
// - Bulk importing millions of records from CSV/APIs
// - High-frequency metrics/logging data storage
// - ETL pipelines requiring INSERT RETURNING id
// - Batch updates for inventory/order systems
//
// Basic Usage:
//
//	import (
//	    "context"
//	    "fmt"
//	    "github.com/jackc/pgx/v5"
//	    "github.com/kaichao/gopkg/pgbulk"
//	)
//
//	// Connect to PostgreSQL
//	conn, err := pgx.Connect(context.Background(), "postgres://user:password@localhost/dbname")
//	if err != nil {
//	    panic(err)
//	}
//	defer conn.Close(context.Background())
//
//	// Bulk insert using COPY command
//	sqlTemplate := "INSERT INTO table_name (col1, col2)"
//	data := [][]interface{}{
//	    {"value1", "value2"},
//	    {"value3", "value4"},
//	}
//	affected, err := pgbulk.Copy(conn, sqlTemplate, data)
//	if err != nil {
//	    panic(err)
//	}
//	fmt.Println("Affected rows:", affected)
//
//	// Insert with ON CONFLICT clause
//	err = pgbulk.Insert(conn, "INSERT INTO users (email, name)", [][]interface{}{
//	    {"alice@example.com", "Alice"},
//	    {"bob@example.com", "Bob"},
//	}, "ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name")
//	if err != nil {
//	    panic(err)
//	}
//
//	// Insert returning IDs
//	ids, err := pgbulk.InsertReturningID(conn, "INSERT INTO products (name, price)", [][]interface{}{
//	    {"Product A", 99.99},
//	    {"Product B", 149.99},
//	})
//	if err != nil {
//	    panic(err)
//	}
//	fmt.Println("Inserted IDs:", ids)
//
//	// Bulk update
//	sqlTemplate := "UPDATE orders SET status = $1 WHERE id = $2"
//	data := [][]interface{}{
//	    {"shipped"},
//	    {"processing"},
//	}
//	ids := [][]interface{}{
//	    {1001},
//	    {1002},
//	}
//	failedIDs, err := pgbulk.Update(conn, sqlTemplate, data, ids)
//	if err != nil {
//	    panic(err)
//	}
//	fmt.Println("Failed update IDs:", failedIDs)
//
// Available Functions:
//
//	// Copy performs a batch insert using PostgreSQL's COPY command
//	func Copy(conn *pgx.Conn, sqlTemplate string, data [][]interface{}) (int, error)
//
//	// Insert inserts data into database using provided SQL template and data
//	func Insert(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, onConflict ...string) error
//
//	// InsertReturningID inserts data and returns IDs of inserted rows
//	func InsertReturningID(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, returningColumnAndOnConflict ...string) ([]int, error)
//
//	// Update performs a bulk update using the provided SQL template, data, and ids
//	func Update(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, ids [][]interface{}) ([][]interface{}, error)
//
// Dependencies:
// - github.com/jackc/pgx/v5
// - github.com/kaichao/gopkg/errors
// - github.com/sirupsen/logrus (for trace logging in Copy function)
//
// Error Handling:
// All functions use github.com/kaichao/gopkg/errors for enhanced error tracing and context.
//
// For more detailed examples, see the examples/ directory.
package pgbulk
