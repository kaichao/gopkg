package pgbulk_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/pgbulk"
	"github.com/stretchr/testify/assert"
)

// 用docker启动本地postgresql，docker run -e POSTGRES_PASSWORD=secret -p 5432:5432 -d postgres:17.4
func ExampleCopy() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgres://postgres:secret@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), `
		DROP TABLE IF EXISTS test_bulk;
		CREATE TABLE test_bulk (
			id SERIAL PRIMARY KEY,
			name TEXT,
			age INT
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	data := [][]interface{}{
		{"Alice", 30},
		{"Bob", 25},
		{"Charlie", 35},
	}

	sqlTemplate := "INSERT INTO test_bulk (name, age)"
	inserted, err := pgbulk.Copy(conn, sqlTemplate, data)
	if err != nil {
		log.Fatalf("Copy failed: %v", err)
	}

	fmt.Printf("Inserted %d rows.\n", inserted)

	// Output:
	// Inserted 3 rows.
}

func TestCopy_RealPostgres(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, "postgres://postgres:secret@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	defer conn.Close(ctx)

	// Create test table
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_copy (
			id SERIAL PRIMARY KEY,
			str TEXT,
			num INT,
			active BOOLEAN,
			ts TIMESTAMP,
			tsarr TIMESTAMP[],
			js JSONB,
			nullstr TEXT
		)`)
	assert.NoError(t, err)

	// Clean table
	_, err = conn.Exec(ctx, `DELETE FROM test_copy`)
	assert.NoError(t, err)

	// Sample data
	now := time.Date(2025, 5, 6, 15, 4, 5, 123456000, time.UTC)
	jsonMap := map[string]interface{}{"name": "example", "age": 30}

	data := [][]interface{}{
		{
			"hello",          // str
			42,               // num
			true,             // active
			now,              // ts
			[]time.Time{now}, // tsarr
			jsonMap,          // js
			sql.NullString{String: "nullable", Valid: true}, // nullstr
		},
		{
			"world",                              // str
			99,                                   // num
			false,                                // active
			now.Add(time.Hour),                   // ts
			[]time.Time{now, now.Add(time.Hour)}, // tsarr
			jsonMap,                              // js
			sql.NullString{Valid: false},         // nullstr = NULL
		},
	}

	// Execute COPY
	count, err := pgbulk.Copy(conn, "INSERT INTO test_copy (str, num, active, ts, tsarr, js, nullstr)", data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), count)

	// Validate data
	rows, err := conn.Query(ctx, `SELECT str, num, active, ts, tsarr, js, nullstr FROM test_copy ORDER BY id`)
	assert.NoError(t, err)
	defer rows.Close()

	var readCount int
	for rows.Next() {
		var (
			str     string
			num     int
			active  bool
			ts      time.Time
			tsarr   []time.Time
			js      []byte
			nullstr sql.NullString
		)
		err := rows.Scan(&str, &num, &active, &ts, &tsarr, &js, &nullstr)
		assert.NoError(t, err)

		readCount++
	}

	assert.Equal(t, len(data), readCount)
}
