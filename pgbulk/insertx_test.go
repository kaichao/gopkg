package pgbulk_test

import (
	"database/sql"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kaichao/gopkg/pgbulk"
	"github.com/stretchr/testify/assert"
)

func TestMockInsertReturningID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlTemplate := "INSERT INTO test_table (col1, col2) VALUES %s"
	data := [][]interface{}{
		{1, "a"},
		{2, "b"},
		{3, "c"},
	}

	// 模拟实际生成的 SQL 语句
	actualSQL := "INSERT INTO test_table (col1, col2) VALUES ($1,$2),($3,$4),($5,$6) RETURNING id"

	// 用 QuoteMeta 来确保正则安全匹配实际 SQL
	expectedSQL := regexp.QuoteMeta(actualSQL)

	rows := sqlmock.NewRows([]string{"id"}).
		AddRow(101).
		AddRow(102).
		AddRow(103)

	mock.ExpectQuery(expectedSQL).
		WithArgs(1, "a", 2, "b", 3, "c").
		WillReturnRows(rows)

	ids, err := pgbulk.InsertReturningID(db, sqlTemplate, data)
	assert.NoError(t, err)
	assert.Equal(t, []int{101, 102, 103}, ids)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func ExampleInsertReturningID() {
	db, err := sql.Open("postgres", "user=postgres password=secret dbname=postgres sslmode=disable")
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}
	defer db.Close()

	// 创建测试表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        name TEXT,
        age INT
    )`)
	if err != nil {
		fmt.Println("Failed to create table:", err)
		return
	}

	// 准备批量插入的数据
	data := [][]interface{}{
		{"Alice", 25},
		{"Bob", 30},
		{"Charlie", 35},
	}

	// 插入并获取 ID
	ids, err := pgbulk.InsertReturningID(db, "INSERT INTO users (name, age)", data)
	if err != nil {
		fmt.Println("Bulk insert failed:", err)
		return
	}

	// 打印插入的 ID
	fmt.Println("Inserted IDs:", ids)

	// Output:
	// Inserted IDs: [1 2 3]
}
