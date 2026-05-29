# CLAUDE.md

## pgbulk Package

Lightweight PostgreSQL bulk operations for high-performance data operations.

### Key Functions
```go
func Copy(conn *pgx.Conn, sql string, rows [][]interface{}) error
func Insert(conn *pgx.Conn, sql string, rows [][]interface{}, onConflict ...string) error
func InsertReturningID(conn *pgx.Conn, sql string, rows [][]interface{}) ([]int64, error)
func Update(conn *pgx.Conn, sql string, rows [][]interface{}) error
```

All functions return enhanced traced errors via `gopkg/errors`.

### Usage Example
```go
conn, _ := pgx.Connect(ctx, "postgres://user:pass@localhost/dbname")

// COPY
pgbulk.Copy(conn, "INSERT INTO users (name, email)", [][]interface{}{
    {"Alice", "alice@example.com"},
})

// Insert with ON CONFLICT
pgbulk.Insert(conn, "INSERT INTO users (email, name)", rows,
    "ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name")

// Insert returning IDs
ids, _ := pgbulk.InsertReturningID(conn, "INSERT INTO products (name, price)", [][]interface{}{
    {"Product A", 99.99},
})
```
