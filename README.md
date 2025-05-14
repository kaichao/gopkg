# gopkg

[中文](README.zh.md) | English

`gopkg` is a Go utility library powering [scalebox](https://github.com/kaichao/scalebox). Key sub-packages:

## Modules

### 1. `asyncbatch`
Asynchronous batch processor triggered by size threshold or timeout.

### 2. `pgbulk`
PostgreSQL bulk operations (insert/update with ID returning) with performance optimizations.

### 3. `dbcache`
Database caching layer via [go-cache](https://github.com/patrickmn/go-cache), supporting SQL template-based loading.

### 4. `exec`
Cross-environment command executor (local/SSH) with stdout/stderr capture.


## License

MIT License
