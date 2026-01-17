package pgbulk_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// testDBURL 是测试数据库的连接字符串
const testDBURL = "postgres://scalebox:secret@10.0.6.100/scalebox?sslmode=disable"

// getTestConn 返回一个测试数据库连接
// 如果连接失败，测试会直接失败
func getTestConn(t *testing.T) *pgx.Conn {
	t.Helper()

	conn, err := pgx.Connect(context.Background(), testDBURL)
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}

	// 注册清理函数
	t.Cleanup(func() {
		conn.Close(context.Background())
	})

	return conn
}

// setupTestTable 创建测试表，返回清理函数
func setupTestTable(t *testing.T, conn *pgx.Conn, tableName, schema string) func() {
	t.Helper()

	// 删除表（如果存在）
	dropSQL := "DROP TABLE IF EXISTS " + tableName
	_, err := conn.Exec(context.Background(), dropSQL)
	if err != nil {
		t.Fatalf("删除表 %s 失败: %v", tableName, err)
	}

	// 创建表
	_, err = conn.Exec(context.Background(), schema)
	if err != nil {
		t.Fatalf("创建表 %s 失败: %v", tableName, err)
	}

	// 返回清理函数
	return func() {
		_, err := conn.Exec(context.Background(), dropSQL)
		if err != nil {
			t.Logf("警告: 清理表 %s 失败: %v", tableName, err)
		}
	}
}
