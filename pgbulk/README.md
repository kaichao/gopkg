# pgbulk

Lightweight Go package for high-performance PostgreSQL bulk operations.

## Features

- **Batch Processing**: Automatically chunks large datasets into optimal batches
- **SQL Templates**: Reusable templates with dynamic placeholders  
- **Full CRUD Support**: `INSERT`, `UPDATE`, and `INSERT...RETURNING` operations
- **PG-Compatible**: Respects PostgreSQL's parameter limits
- **Error Handling**: Enhanced error tracing with `github.com/kaichao/gopkg/errors`

## Use Cases

- Bulk importing millions of records from CSV/APIs
- High-frequency metrics/logging data storage
- ETL pipelines requiring `INSERT RETURNING id`
- Batch updates for inventory/order systems

## Installation

To use the `pgbulk` package, first install it along with its dependencies:

```sh
go get github.com/kaichao/gopkg/pgbulk
go get github.com/jackc/pgx/v5
```

## Quick Start

For comprehensive examples, see the [examples directory](examples/).

Basic usage:

```go
import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5"
    "github.com/kaichao/gopkg/pgbulk"
)

func main() {
    conn, err := pgx.Connect(context.Background(), "postgres://user:password@localhost/dbname")
    if err != nil {
        panic(err)
    }
    defer conn.Close(context.Background())

    // Bulk insert using COPY
    affected, err := pgbulk.Copy(conn, "INSERT INTO users (name, email)", [][]interface{}{
        {"Alice", "alice@example.com"},
        {"Bob", "bob@example.com"},
    })
    if err != nil {
        panic(err)
    }
    fmt.Println("Affected rows:", affected)
}
```

## Package Overview

### Functions

- **Copy**: Perform bulk insert using PostgreSQL's COPY command
- **Insert**: Insert data with optional ON CONFLICT clause  
- **InsertReturningID**: Insert data and return IDs of inserted rows
- **Update**: Perform bulk update with error tracking

For complete API documentation and examples, see:
- [package documentation](doc.go) - Detailed API reference
- [examples/basic](examples/basic/) - Basic usage examples  
- [examples/advanced](examples/advanced/) - Advanced scenarios

## Dependencies

- [github.com/jackc/pgx/v5](https://github.com/jackc/pgx) - PostgreSQL driver
- [github.com/kaichao/gopkg/errors](https://github.com/kaichao/gopkg/errors) - Error handling
- [github.com/sirupsen/logrus](https://github.com/sirupsen/logrus) - Trace logging (optional)

## Error Handling

All functions use `github.com/kaichao/gopkg/errors` for enhanced error tracing with context. Errors include stack traces and structured context information for easier debugging.

## Performance Notes

- **COPY**: Most efficient for large bulk inserts (thousands to millions of rows)
- **Insert/InsertReturningID**: Better for smaller batches with complex SQL
- **Update**: Uses pgx batching for efficient bulk updates
- All operations respect PostgreSQL's parameter limits (65,535 parameters per statement)

## License

MIT License

## See Also

- [gopkg](https://github.com/kaichao/gopkg) - Parent repository containing other utility packages