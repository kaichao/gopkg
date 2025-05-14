# pgbulk

[English](README.md) | 中文

轻量级 PostgreSQL 批量操作 Go 工具包

## 功能特性

- **批量处理**: 自动将大数据集分块为最优批次
- **SQL模板**: 支持动态占位符的可复用模板
- **完整CRUD**: 支持 `INSERT`、`UPDATE` 及 `INSERT...RETURNING` 操作
- **PG兼容**: 严格遵循 PostgreSQL 参数限制

## 典型场景

- 从CSV/API批量导入百万级数据
- 高频指标/日志数据存储
- 需要返回ID的ETL数据管道
- 库存/订单系统的批量更新

## 安装
```bash
go get github.com/kaichao/gopkg/pgbulk
go get github.com/jackc/pgx/v5
```

## 快速开始

以下是使用 `pgbulk` 包的主要函数的示例。

### Copy

`Copy` 函数使用 PostgreSQL 的 `COPY` 命令批量插入数据。SQL 模板需提供为 `"INSERT INTO table_name (col1, col2)"`，内部会处理为实际的 `COPY` 命令。

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

`Insert` 函数用于插入单行或多行数据。SQL 模板需提供为 `"INSERT INTO table_name (col1, col2)"`，内部会处理为完整的插入语句。

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

`InsertReturningID` 函数插入数据并返回新插入记录的 ID。SQL 模板需提供为 `"INSERT INTO table_name (col1, col2)"`，内部会处理为包含 `RETURNING` 子句的插入语句。

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

`Update` 函数用于更新数据。SQL 模板需为完整的更新语句，例如 `"UPDATE t_task SET f1=$1, f2=$2, f3=$3, f4=$4, f5=$5 WHERE id=$6"`。

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

## API说明

### Copy

```go
func Copy(conn *pgx.Conn, sqlTemplate string, data [][]interface{}) (int64, error)
```

- **参数**:
  - `conn *pgx.Conn`: PostgreSQL 连接
  - `sqlTemplate string`: SQL 模板，例如 `"INSERT INTO table_name (col1, col2)"`，内部处理为 `COPY` 命令
  - `data [][]interface{}`: 待插入的数据，二维数组，每行对应一条记录

- **返回**:
  - `int64`: 成功插入的记录数
  - `error`: 错误信息

- **描述**:
  使用 PostgreSQL 的 `COPY` 命令批量插入数据，高效处理大量数据。SQL 模板 `"INSERT INTO table_name (col1, col2)"` 会被内部处理为 `COPY table_name (col1, col2) FROM STDIN`。

### Insert

```go
func Insert(conn *pgx.Conn, sqlTemplate string, data [][]interface{}) (int64, error)
```

- **参数**:
  - `conn *pgx.Conn`: PostgreSQL 连接
  - `sqlTemplate string`: SQL 模板，例如 `"INSERT INTO table_name (col1, col2)"`，内部处理为插入语句
  - `data [][]interface{}`: 待插入的数据，二维数组，每行对应一条记录

- **返回**:
  - `int64`: 成功插入的记录数
  - `error`: 错误信息

- **描述**:
  插入单行或多行数据。SQL 模板 `"INSERT INTO table_name (col1, col2)"` 会被内部处理为 `INSERT INTO table_name (col1, col2) VALUES ($1, $2)`。

### InsertReturningID

```go
func InsertReturningID(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, returningColumn ...string) ([]int64, error)
```

- **参数**:
  - `conn *pgx.Conn`: PostgreSQL 连接
  - `sqlTemplate string`: SQL 模板，例如 `"INSERT INTO table_name (col1, col2)"`，内部处理为包含 `RETURNING` 子句的插入语句
  - `data [][]interface{}`: 待插入的数据，二维数组，每行对应一条记录
  - `returningColumn ...string`: 返回的字段名称，例如 `"id"`

- **返回**:
  - `[]int64`: 新插入记录的 ID 数组
  - `error`: 错误信息

- **描述**:
  插入数据并返回新插入记录的 ID。SQL 模板 `"INSERT INTO table_name (col1, col2)"` 会被内部处理为 `INSERT INTO table_name (col1, col2) VALUES ($1, $2) RETURNING id`。

### Update

```go
func Update(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, ids [][]interface{}) ([][]interface{}, error)
```

- **参数**:
  - `conn *pgx.Conn`: PostgreSQL 连接
  - `sqlTemplate string`: SQL 模板，例如 `"UPDATE t_task SET f1=$1, f2=$2, f3=$3, f4=$4, f5=$5 WHERE id=$6"`
  - `data [][]interface{}`: 待更新的数据，二维数组，每行对应 SET 子句的值
  - `ids [][]interface{}`: WHERE 子句的值，二维数组，每行对应 WHERE 子句的值

- **返回**:
  - `[][]interface{}`: 更新失败的记录的 WHERE 值数组（即 WHERE 条件未匹配的记录）
  - `error`: 错误信息

- **描述**:
  更新数据，返回更新失败的记录的 WHERE 值。

## 单元测试，启动 PostgreSQL:
```sh
docker run -e POSTGRES_PASSWORD=secret -p 5432:5432 -d postgres:17.4
```
