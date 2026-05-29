# self

Runtime introspection utilities for Go: goroutine, thread, and process identification.

## Features

- Goroutine ID retrieval (for debugging)
- OS thread ID retrieval
- Process ID retrieval
- Current goroutine stack trace
- Function name extraction with custom separators

## Installation

```bash
go get github.com/kaichao/gopkg/self
```

## Quick Start

```go
import "github.com/kaichao/gopkg/self"

goid := self.GetGoroutineID()   // Current goroutine ID
tid := self.GetThreadID()       // OS thread ID
pid := self.GetProcessID()      // Process ID
name := self.GetFunctionName(myFunc, '.')  // Function name (last segment)
stack := self.GetCurrentGoroutineStack()   // Stack trace
```

## API Reference

### GetGoroutineID
```go
func GetGoroutineID() int64
```
Returns the current goroutine's ID by parsing `runtime.Stack()` output. **Unofficial method** — for debugging only. Returns -1 on failure.

### GetThreadID
```go
func GetThreadID() int64
```
Returns the current OS thread ID via `syscall.SYS_GETTID`. Linux/macOS only.

### GetProcessID
```go
func GetProcessID() int64
```
Returns the current process ID via `os.Getpid()`.

### GetCurrentGoroutineStack
```go
func GetCurrentGoroutineStack() string
```
Returns the stack trace of the current goroutine. Uses auto-growing buffer (starts at 4KB, doubles until it fits).

### GetFunctionName
```go
func GetFunctionName(i interface{}, seps ...rune) string
```
Returns the function name of `i`, split by custom separators, returning the last segment. Default separators: `.`, `/`.

## Notes

- `GetGoroutineID()` relies on parsing undocumented runtime output and may break across Go versions
- `GetThreadID()` uses platform-specific syscalls — not portable to all platforms
- These functions are intended for diagnostics and debugging, not production logic

## License

MIT License
