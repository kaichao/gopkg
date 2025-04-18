# DBCache

[中文](README.zh.md) | English

Generic database caching layer for Go, providing SQL-template based data loading with automatic cache population.

## Features

- **SQL Templating**: Parameterized query support with $1, $2 placeholders
- **Automatic Caching**: Transparent cache population on miss
- **Type Safety**: Generics support for any data type
- **Cache Control**: Configurable expiration and cleanup

## Use Cases

- Frequently accessed database records (users/products)
- Expensive query result caching
- Rate-limited API response caching
- Configuration storage optimization

## Installation
```sh
go get github.com/kaichao/gopkg/dbcache
```

## Quick Start
```go
// Initialize cache for user emails
emailCache := dbcache.New[string](
    db, // *sql.DB connection
    "SELECT email FROM users WHERE id = $1",
    5*time.Minute, // Cache expiration
    10*time.Minute, // Cleanup interval
    nil, // Use default loader
)

// Get user email (cached)
email, _ := emailCache.Get(123)
```

## API Reference
### New
Creates a new database cache instance.
```go
func NewDBCache[T any](
    db *sql.DB,
    sqlTemplate string,
    defaultExp, cleanupInterval time.Duration,
    loader func(...any) (T, error),
) *DBCache[T]
```

### Get
Retrieves cached value or executes SQL query.
```go
// Custom hash generator
loader := func(params ...any) (string, error) {
    id := params[0].(int)
    return fmt.Sprintf("hash-%d", id), nil
}

hashCache := dbcache.NewDBCache[string](
    nil, // db not required
    "", 
    0, 0,
    loader,
)
```
