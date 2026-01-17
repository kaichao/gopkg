package pgbulk_test

import (
	"context"
	"testing"

	"github.com/kaichao/gopkg/pgbulk"
)

func TestInsert(t *testing.T) {
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

	// 调用 Insert 函数
	err := pgbulk.Insert(conn, sqlTemplate, data)
	if err != nil {
		t.Errorf("插入失败: %v", err)
	}

	// 验证插入结果
	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM test_table").Scan(&count)
	if err != nil {
		t.Errorf("查询行数失败: %v", err)
	}
	if count != 2 {
		t.Errorf("预期插入 2 行，实际得到 %d 行", count)
	}
}

func TestInsertWithOnConflict(t *testing.T) {
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
	err := pgbulk.Insert(conn, sqlTemplate, data1)
	if err != nil {
		t.Fatalf("第一次插入失败: %v", err)
	}

	// 尝试插入重复数据，使用 ON CONFLICT DO NOTHING
	data2 := [][]interface{}{
		{"alice@example.com", "Alice Updated"}, // 重复的email
		{"charlie@example.com", "Charlie"},     // 新的email
	}

	// 测试 ON CONFLICT DO NOTHING
	err = pgbulk.Insert(conn, sqlTemplate, data2, "ON CONFLICT (email) DO NOTHING")
	if err != nil {
		t.Fatalf("带ON CONFLICT DO NOTHING的插入失败: %v", err)
	}

	// 验证数据
	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM test_table_conflict").Scan(&count)
	if err != nil {
		t.Fatalf("查询行数失败: %v", err)
	}
	// 应该有3行：alice, bob, charlie (alice没有重复插入)
	if count != 3 {
		t.Errorf("预期有 3 行数据，实际得到 %d 行", count)
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

	// 测试 ON CONFLICT DO UPDATE
	err = pgbulk.Insert(conn, sqlTemplate, data3, "ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name")
	if err != nil {
		t.Fatalf("带ON CONFLICT DO UPDATE的插入失败: %v", err)
	}

	// 验证数据
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM test_table_conflict").Scan(&count)
	if err != nil {
		t.Fatalf("查询行数失败: %v", err)
	}
	// 应该有4行：alice(更新), bob, charlie, david(新增)
	if count != 4 {
		t.Errorf("预期有 4 行数据，实际得到 %d 行", count)
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
