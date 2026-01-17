package pgbulk_test

import (
	"context"
	"testing"

	"github.com/kaichao/gopkg/pgbulk"
)

func TestInsertReturningID(t *testing.T) {
	conn := getTestConn(t)
	ctx := context.Background()

	// 创建测试表
	cleanup := setupTestTable(t, conn, "test_table", `
		CREATE TABLE test_table (
			id SERIAL PRIMARY KEY,
			col1 TEXT,
			col2 TEXT
		)
	`)
	defer cleanup()

	// 准备测试数据
	sqlTemplate := "INSERT INTO test_table (col1, col2)"
	data := [][]interface{}{
		{"value1", "value2"},
		{"value3", "value4"},
	}

	// 调用 InsertReturningID 函数
	ids, err := pgbulk.InsertReturningID(conn, sqlTemplate, data)
	if err != nil {
		t.Errorf("InsertReturningID 失败: %v", err)
	}

	// 验证返回的ID数量
	if len(ids) != 2 {
		t.Errorf("预期返回 2 个 ID，实际得到 %d 个", len(ids))
	}

	// 验证每个返回的ID是否存在于数据库中
	for _, id := range ids {
		var exists bool
		err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM test_table WHERE id = $1)", id).Scan(&exists)
		if err != nil || !exists {
			t.Errorf("ID %d 在数据库中未找到", id)
		}
	}
}

func TestInsertReturningIDWithOnConflict(t *testing.T) {
	conn := getTestConn(t)
	ctx := context.Background()

	// 创建测试表，包含唯一约束
	cleanup := setupTestTable(t, conn, "test_table_conflict", `
		CREATE TABLE test_table_conflict (
			id SERIAL PRIMARY KEY,
			email TEXT UNIQUE,
			name TEXT
		)
	`)
	defer cleanup()

	// 首先插入一些数据
	sqlTemplate := "INSERT INTO test_table_conflict (email, name)"
	data1 := [][]interface{}{
		{"alice@example.com", "Alice"},
		{"bob@example.com", "Bob"},
	}

	// 插入第一批数据
	ids1, err := pgbulk.InsertReturningID(conn, sqlTemplate, data1)
	if err != nil {
		t.Fatalf("第一次插入失败: %v", err)
	}
	if len(ids1) != 2 {
		t.Fatalf("预期第一次插入返回 2 个 ID，实际得到 %d 个", len(ids1))
	}

	// 尝试插入重复数据，使用 ON CONFLICT DO NOTHING
	data2 := [][]interface{}{
		{"alice@example.com", "Alice Updated"}, // 重复的email
		{"charlie@example.com", "Charlie"},     // 新的email
	}

	// 测试 ON CONFLICT DO NOTHING
	ids2, err := pgbulk.InsertReturningID(conn, sqlTemplate, data2, "id", "ON CONFLICT (email) DO NOTHING")
	if err != nil {
		t.Fatalf("带ON CONFLICT DO NOTHING的插入失败: %v", err)
	}

	// 应该只返回1个ID (charlie@example.com)
	if len(ids2) != 1 {
		t.Errorf("带ON CONFLICT DO NOTHING插入预期返回 1 个 ID，实际得到 %d 个", len(ids2))
	}

	// 验证 alice@example.com 没有被更新
	var aliceName string
	err = conn.QueryRow(ctx, "SELECT name FROM test_table_conflict WHERE email = 'alice@example.com'").Scan(&aliceName)
	if err != nil {
		t.Fatalf("查询alice数据失败: %v", err)
	}
	if aliceName != "Alice" {
		t.Errorf("预期alice的名字为 'Alice'，实际为 '%s'", aliceName)
	}

	// 测试 ON CONFLICT DO UPDATE
	data3 := [][]interface{}{
		{"alice@example.com", "Alice Updated Again"},
		{"david@example.com", "David"},
	}

	// 测试自定义返回列和 ON CONFLICT DO UPDATE
	ids3, err := pgbulk.InsertReturningID(conn, sqlTemplate, data3, "id", "ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name")
	if err != nil {
		t.Fatalf("带ON CONFLICT DO UPDATE的插入失败: %v", err)
	}

	// 应该返回2个ID (alice@example.com 更新, david@example.com 插入)
	if len(ids3) != 2 {
		t.Errorf("带ON CONFLICT DO UPDATE插入预期返回 2 个 ID，实际得到 %d 个", len(ids3))
	}

	// 验证 alice@example.com 被更新了
	err = conn.QueryRow(ctx, "SELECT name FROM test_table_conflict WHERE email = 'alice@example.com'").Scan(&aliceName)
	if err != nil {
		t.Fatalf("查询更新后的alice数据失败: %v", err)
	}
	if aliceName != "Alice Updated Again" {
		t.Errorf("预期alice更新后的名字为 'Alice Updated Again'，实际为 '%s'", aliceName)
	}
}
