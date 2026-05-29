# CLAUDE.md

## dbcache Package

Generic database caching layer with SQL template support and automatic cache population.

### Core Type
```go
type Cache[T any] struct { ... }
func New[T any](db *sql.DB, query string, expiration, cleanup time.Duration, loader Loader[T]) *Cache[T]
```

### Methods
- `Get(params ...interface{}) (T, error)` — Returns cached value or loads from DB/custom loader

### Usage Example
```go
emailCache := dbcache.New[string](
    db,
    "SELECT email FROM users WHERE id = $1",
    5*time.Minute,   // Cache expiration
    10*time.Minute,  // Cleanup interval
    nil,             // Use default SQL loader
)

email, err := emailCache.Get(123)
```

### Notes
- Cache keys generated via `fmt.Sprintf("%v", params)`
- Errors returned as-is from DB operations, no special wrapping
- Requires Go 1.18+ (generics)
