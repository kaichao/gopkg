# CLAUDE.md

## logger Package

Structured logging with error tracing, async output, log rotation, and sensitive data filtering.

### Logger Struct (Recommended)

```go
cfg := &logger.Config{
    Level:        "info",
    Format:       "json",
    Output:       "stdout",
    FilePath:     "app.log",
    MaxSize:      100,     // MB
    MaxAge:       7,       // days
    MaxBackups:   5,
    AsyncEnabled: false,
    BufferSize:   1000,
}
log, err := logger.NewLogger(cfg)
defer log.Close()
```

Constructor: `NewOrMust(cfg *Config) *Logger` — simpler, panics on error.

### Error Logging Functions

- **LogError(err, entry)** — Auto: DEBUG/TRACE → detailed, INFO+ → simple. Override with `LOG_ERROR_VERBOSE` env var.
- **LogTracedError(err, entry)** — Detailed chain with full context. Inner errors at Debug level.
- **SimpleLog(err, entry)** — Production-safe, filters sensitive data.

### Configuration via Env Vars
| Variable | Default | Description |
|----------|---------|-------------|
| LOG_LEVEL | info | trace/debug/info/warn/error/fatal |
| LOG_FORMAT | json | text, json |
| LOG_OUTPUT | stdout | stdout, stderr, file |
| LOG_FILE_PATH | app.log | File path |
| LOG_ASYNC_ENABLED | false | Enable async |
| LOG_ERROR_VERBOSE | (auto) | true=detailed, false=simple |

### Usage Examples
```go
log, _ := logger.NewLogger(&logger.Config{Level: "info"})
defer log.Close()

log.Info("started")
log.WithField("user_id", 123).Info("login")

err := errors.New("db error")
logger.LogError(err, logrus.NewEntry(log))
```
