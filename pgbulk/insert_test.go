package pgbulk_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/pgbulk"
)

func TestInsert(t *testing.T) {
	// 连接数据库
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:secret@localhost/postgres")
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}
	defer conn.Close(context.Background())

	// 创建测试表
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

	// 准备测试数据
	sqlTemplate := "INSERT INTO test_table (col1, col2)"
	data := [][]interface{}{
		{"value1", "value2"},
		{"value3", "value4"},
	}

	// 调用 Insert 函数
	err = pgbulk.Insert(conn, sqlTemplate, data)
	if err != nil {
		t.Errorf("插入失败: %v", err)
	}

	// 验证插入结果
	var count int
	err = conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM test_table").Scan(&count)
	if err != nil {
		t.Errorf("查询行数失败: %v", err)
	}
	if count != 2 {
		t.Errorf("预期插入 2 行，实际得到 %d 行", count)
	}

	// 清理测试数据
	_, err = conn.Exec(context.Background(), "DROP TABLE test_table")
	if err != nil {
		t.Errorf("删除测试表失败: %v", err)
	}
}
