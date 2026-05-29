# CLAUDE.md

## self Package

Runtime introspection utilities for goroutine, thread, and process identification.

### Functions
```go
func GetGoroutineID() int64       // Current goroutine ID (unofficial, via runtime.Stack)
func GetThreadID() int64          // OS thread ID (via syscall.SYS_GETTID)
func GetProcessID() int64         // Process ID (via os.Getpid)
func GetCurrentGoroutineStack() string  // Stack trace of current goroutine
func GetFunctionName(i interface{}, seps ...rune) string  // Function name with custom separators
```

### Usage Example
```go
import "github.com/kaichao/gopkg/self"

goid := self.GetGoroutineID()
tid := self.GetThreadID()
pid := self.GetProcessID()
name := self.GetFunctionName(myFunc, '.')
stack := self.GetCurrentGoroutineStack()
```

### Notes
- `GetGoroutineID()` parses `runtime.Stack()` output — unofficial, for debugging only
- `GetThreadID()` uses `SYS_GETTID` — Linux/macOS only
- `GetFunctionName()` splits on custom separators and returns the last segment
