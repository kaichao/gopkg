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

To use the `pgbulk` package, first install it along with its dependencies:

```sh
go get github.com/kaichao/gopkg/pgbulk
go get github.com/jackc/pgx/v5
```

## Quick Start

Below are examples demonstrating the main functions of the `pgbulk` package.

### Copy

The `Copy` function uses PostgreSQL's `COPY` command to perform bulk data insertion. The SQL template should be provided as `"INSERT INTO table_name (col1, col2)"`, and it will be internally processed into an actual `COPY` command.

```go
package main

import (
	"context"
	"fmt"
	"github.com/kaichao/gopkg/pgbulk"
	"github.com/jackc/pgx/v5"
)

func main() {
	conn, err := pgx.Connect(context.Background(), "postgres://user:password@localhost/dbname")
	if err != nil {
		panic(err)
	}
	defer conn.Close(context.Background())

	sqlTemplate := "INSERT INTO table_name (col1, col2)"
	data := [][]interface{}{
		{"value1", "value2"},
		{"value3", "value4"},
	}
	affected, err := pgbulk.Copy(conn, sqlTemplate, data)
	if err != nil {
		panic(err)
	}
	fmt.Println("Affected rows:", affected)
}
```

### Insert

The `Insert` function is used to insert single or multiple rows of data. The SQL template should be provided as `"INSERT INTO table_name (col1, col2)"`, and it will be internally processed into a complete insert statement.

```go
sqlTemplate := "INSERT INTO table_name (col1, col2)"
data := [][]interface{}{
	{"value1", "value2"},
	{"value3", "value4"},
}
affected, err := pgbulk.Insert(conn, sqlTemplate, data)
if err != nil {
	panic(err)
}
fmt.Println("Affected rows:", affected)
```

### InsertReturningID

The `InsertReturningID` function inserts data and returns the IDs of the newly inserted records. The SQL template should be provided as `"INSERT INTO table_name (col1, col2)"`, and it will be internally processed into an insert statement with a `RETURNING` clause.

```go
sqlTemplate := "INSERT INTO table_name (col1, col2)"
data := [][]interface{}{
	{"value1", "value2"},
	{"value3", "value4"},
}
ids, err := pgbulk.InsertReturningID(conn, sqlTemplate, data, "id")
if err != nil {
	panic(err)
}
fmt.Println("Inserted IDs:", ids)
```

### Update

The `Update` function is used to update data. The SQL template must be a complete update statement, such as `"UPDATE t_task SET f1=$1, f2=$2, f3=$3, f4=$4, f5=$5 WHERE id=$6"`.

```go
sqlTemplate := "UPDATE t_task SET f1=$1, f2=$2, f3=$3, f4=$4, f5=$5 WHERE id=$6"
data := [][]interface{}{
	{"newf1", "newf2", "newf3", "newf4", "newf5"},
	{"newf6", "newf7", "newf8", "newf9", "newf10"},
}
ids := [][]interface{}{
	{1},
	{2},
}
failedIDs, err := pgbulk.Update(conn, sqlTemplate, data, ids)
if err != nil {
	panic(err)
}
fmt.Println("Failed update IDs:", failedIDs)
```

## API Documentation

### Copy

```go
func Copy(conn *pgx.Conn, sqlTemplate string, data [][]interface{}) (int64, error)
```

- **Parameters**:
  - `conn *pgx.Conn`: PostgreSQL connection
  - `sqlTemplate string`: SQL template, e.g., `"INSERT INTO table_name (col1, col2)"`, internally processed into a `COPY` command
  - `data [][]interface{}`: Data to be inserted, a 2D array where each row corresponds to a record

- **Returns**:
  - `int64`: Number of successfully inserted records
  - `error`: Error information

- **Description**:
  Uses PostgreSQL's `COPY` command to perform bulk data insertion, efficiently handling large datasets. The SQL template `"INSERT INTO table_name (col1, col2)"` is internally processed into `COPY table_name (col1, col2) FROM STDIN`.

### Insert

```go
func Insert(conn *pgx.Conn, sqlTemplate string, data [][]interface{}) (int64, error)
```

- **Parameters**:
  - `conn *pgx.Conn`: PostgreSQL connection
  - `sqlTemplate string`: SQL template, e.g., `"INSERT INTO table_name (col1, col2)"`, internally processed into an insert statement
  - `data [][]interface{}`: Data to be inserted, a 2D array where each row corresponds to a record

- **Returns**:
  - `int64`: Number of successfully inserted records
  - `error`: Error information

- **Description**:
  Inserts single or multiple rows of data. The SQL template `"INSERT INTO table_name (col1, col2)"` is internally processed into `INSERT INTO table_name (col1, col2) VALUES ($1, $2)`.

### InsertReturningID

```go
func InsertReturningID(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, returningColumn ...string) ([]int64, error)
```

- **Parameters**:
  - `conn *pgx.Conn`: PostgreSQL connection
  - `sqlTemplate string`: SQL template, e.g., `"INSERT INTO table_name (col1, col2)"`, internally processed into an insert statement with a `RETURNING` clause
  - `data [][]interface{}`: Data to be inserted, a 2D array where each row corresponds to a record
  - `returningColumn ...string`: Name of the column to return, e.g., `"id"`

- **Returns**:
  - `[]int64`: Array of IDs for the newly inserted records
  - `error`: Error information

- **Description**:
  Inserts data and returns the IDs of the newly inserted records. The SQL template `"INSERT INTO table_name (col1, col2)"` is internally processed into `INSERT INTO table_name (col1, col2) VALUES ($1, $2) RETURNING id`.

### Update

```go
func Update(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, ids [][]interface{}) ([][]interface{}, error)
```

- **Parameters**:
  - `conn *pgx.Conn`: PostgreSQL connection
  - `sqlTemplate string`: SQL template, e.g., `"UPDATE t_task SET f1=$1, f2=$2, f3=$3, f4=$4, f5=$5 WHERE id=$6"`
  - `data [][]interface{}`: Data to be updated, a 2D array where each row corresponds to the values for the SET clause
  - `ids [][]interface{}`: Values for the WHERE clause, a 2D array where each row corresponds to the values for the WHERE clause

- **Returns**:
  - `[][]interface{}`: Array of WHERE clause values for records that failed to update (i.e., records where the WHERE condition did not match)
  - `error`: Error information

- **Description**:
  Updates data and returns the WHERE clause values for records that failed to update.