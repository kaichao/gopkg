# gopkg

[中文](README.zh.md) | English

`gopkg` is a Go utility library powering [scalebox](https://github.com/kaichao/scalebox). Key sub-packages:

## Modules

### 1. `asyncbatch`
Generic batch processor for asynchronous task processing with dynamic flow control, type safety, and parallel processing.

### 2. `pgbulk`
Lightweight PostgreSQL bulk operations (COPY, INSERT, UPDATE) with batch processing, SQL templates, and enhanced error handling.

### 3. `dbcache`
Database caching layer via [go-cache](https://github.com/patrickmn/go-cache), supporting SQL template-based loading.

### 4. `exec`
Cross-environment command executor (local/SSH) with stdout/stderr capture.

### 5. `errors`
Enhanced error handling with tracing, context, error codes, and standard `errors` package compatibility.

### 6. `logger`
Structured logging for traced errors with sensitive data filtering, async output, log rotation, and production-safe logging. Supports both detailed error chains (development) and filtered safe logs (production).

### 7. `param`
Unified command line parameter management for Go with Cobra, supporting multiple data types, environment variables, dynamic defaults, and validation.


## License

MIT License
