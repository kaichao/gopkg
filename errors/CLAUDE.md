# CLAUDE.md

## errors Package

Enhanced error handling for Go with tracing, context, and error codes.

### Core Types

#### TracedError
```go
type TracedError struct {
    Message   string         // Error message
    Code      int            // Error code (default 1)
    Location  string         // File:line:function
    Timestamp time.Time      // When it happened
    Context   map[string]any // Context information
    // cause is private, accessed via Unwrap()
}
```

#### UsageError
Signals incorrect CLI usage (e.g., missing flags). Callers should display usage/help.
```go
type UsageError struct { msg string }
func NewUsage(msg string) *UsageError
```

### Key Functions

**Creation:**
- `New(msg string, args ...int) *TracedError` — code optional, default 1
- `E(args ...any) error` — flexible: `E("msg")`, `E(code, "msg")`, `E("msg", "k", v)`, `E(code, "msg", "k", v)`
- `NewUsage(msg string) *UsageError` — signals usage error

**Wrapping:**
- `Wrap(err error, msg string, skip ...int) *TracedError`
- `WrapE(err error, args ...any) error` — flexible like `E()`

**Inspection:**
- `GetCode(err error) int` — 0 if nil, -1 if not *TracedError
- `Cause(err error) error` — root cause of error chain
- `Is(err, target error) bool` — wraps stdlib `errors.Is`
- `As(err error, target any) bool` — wraps stdlib `errors.As`
- `Unwrap(err error) error` — wraps stdlib `errors.Unwrap`

**Other:**
- `Must(err error)` — panics if err != nil
- `MustValue[T any](value T, err error) T` — returns value or panics

### TracedError Methods
- `WithContext(key string, value any) *TracedError` — chainable
- `Error() string` — message only
- `Format(f fmt.State, verb rune)` — `%v` message, `%+v` full details, `%#v` Go-syntax
- `Detailed() string` — full detail string
- `Unwrap() error` — for stdlib errors.Is/As
- `GetFullChain() []*TracedError` — full error chain
- `Cause() *TracedError` — underlying TracedError cause
- `Is(target error) bool` / `As(target any) bool`

### Default Error Code
Default code is **1** (not 0). The `E()` function uses `parseEArgs()` which defaults to 1.

### Usage Examples
```go
// Creation
err := errors.New("file not found")
err := errors.New("db error", 1001)
err := errors.E(404, "user not found", "user_id", 123)

// Wrapping
return errors.WrapE(err, "database query failed", "query", query)

// Inspection
code := errors.GetCode(err)
root := errors.Cause(err)
```
