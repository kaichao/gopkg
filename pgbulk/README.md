# pgbulk

Lightweight Go package for high-performance PostgreSQL bulk operations.

## Features

- **Batch Processing**: Automatically chunks large datasets into optimal batches
- **SQL Templates**: Reusable templates with dynamic placeholders  
- **Full CRUD Support**: `INSERT`, `UPDATE`, and `INSERT...RETURNING` operations
- **PG-Compatible**: Respects PostgreSQL's parameter limits
- **Error Handling**: Enhanced error tracing with `github.com/kaichao/gopkg/errors`

## Installation

```sh
go get github.com/kaichao/gopkg/pgbulk
go get github.com/jackc/pgx/v5
```

## Quick Start

```go
import (
    "context"
    "github.com/jackc/pgx/v5"
    "github.com/kaichao/gopkg/pgbulk"
)

func main() {
    conn, _ := pgx.Connect(context.Background(), "postgres://user:password@localhost/dbname")
    defer conn.Close(context.Background())

    // Bulk insert using COPY
    pgbulk.Copy(conn, "INSERT INTO users (name, email)", [][]interface{}{
        {"Alice", "alice@example.com"},
        {"Bob", "bob@example.com"},
    })

    // Insert with ON CONFLICT clause
    pgbulk.Insert(conn, "INSERT INTO users (email, name)", [][]interface{}{
        {"alice@example.com", "Alice"},
        {"bob@example.com", "Bob"},
    }, "ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name")

    // Insert returning IDs
    ids, _ := pgbulk.InsertReturningID(conn, "INSERT INTO products (name, price)", [][]interface{}{
        {"Product A", 99.99},
        {"Product B", 149.99},
    })
    fmt.Println("Inserted IDs:", ids)
}
```

For comprehensive examples, see the [examples directory](examples/).

## API Reference

- **Copy**: Bulk insert using PostgreSQL's COPY command
- **Insert**: Insert data with optional ON CONFLICT clause  
- **InsertReturningID**: Insert data and return IDs of inserted rows
- **Update**: Bulk update with error tracking

Detailed API documentation: [package documentation](doc.go)

## Dependencies

- [github.com/jackc/pgx/v5](https://github.com/jackc/pgx) - PostgreSQL driver
- [github.com/kaichao/gopkg/errors](https://github.com/kaichao/gopkg/errors) - Error handling

## License

MIT License
