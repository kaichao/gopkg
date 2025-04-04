# pgbulk

`pgbulk` is a lightweight Go package for efficient bulk operations in PostgreSQL, including `INSERT`, `UPDATE`, and `INSERT ... RETURNING` patterns.

## Features

- Efficiently performs batch inserts, respecting PostgreSQL's parameter limits.
- Automatically chunks large datasets into manageable batches.
- Supports reusable SQL templates with dynamic placeholder generation.

## Use Cases

- Bulk importing millions of rows into a PostgreSQL database.
- Inserting logs, metrics, or analytical data in high-throughput systems.
- Scenarios requiring fast `INSERT RETURNING id` operations.

## Installation

```bash
go get github.com/kaichao/gopkg/pgbulk
```