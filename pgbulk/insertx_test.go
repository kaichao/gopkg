package pgbulk_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/pgbulk"
)

func TestInsertReturningID(t *testing.T) {
	// Connect to the database
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:secret@localhost/postgres")
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}
	defer conn.Close(context.Background())

	// Create a test table
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS test_table (
			id SERIAL PRIMARY KEY,
			col1 TEXT,
			col2 TEXT
		)
	`)
	if err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}

	// Prepare test data
	sqlTemplate := "INSERT INTO test_table (col1, col2)"
	data := [][]interface{}{
		{"value1", "value2"},
		{"value3", "value4"},
	}

	// Call InsertReturningID function
	ids, err := pgbulk.InsertReturningID(conn, sqlTemplate, data)
	if err != nil {
		t.Errorf("InsertReturningID 失败: %v", err)
	}

	// Verify the number of returned IDs
	if len(ids) != 2 {
		t.Errorf("预期返回 2 个 ID，实际得到 %d 个", len(ids))
	}

	// Verify if each returned ID exists in the database
	for _, id := range ids {
		var exists bool
		err = conn.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM test_table WHERE id = $1)", id).Scan(&exists)
		if err != nil || !exists {
			t.Errorf("ID %d 在数据库中未找到", id)
		}
	}

	// Clean up test data
	_, err = conn.Exec(context.Background(), "DROP TABLE test_table")
	if err != nil {
		t.Errorf("删除测试表失败: %v", err)
	}
}
