# pgbulk

[中文](README.zh.md) | English

Lightweight Go package for high-performance PostgreSQL bulk operations.

## Features

- **Batch Processing**: Automatically chunks large datasets into optimal batches
- **SQL Templates**: Reusable templates with dynamic placeholders
- **Full CRUD Support**: `INSERT`, `UPDATE`, and `INSERT...RETURNING` operations
- **PG-Compatible**: Respects PostgreSQL's parameter limits

## Use Cases

- Bulk importing millions of records from CSV/APIs
- High-frequency metrics/logging data storage
- ETL pipelines requiring `INSERT RETURNING id`
- Batch updates for inventory/order systems

## Installation

```bash
go get github.com/kaichao/gopkg/pgbulk
```

## Quick Start

### 1. Start PostgreSQL:
```sh
docker run -e POSTGRES_PASSWORD=secret -p 5432:5432 -d postgres:17.4
```

### 2. Basic usage:
```go

```
