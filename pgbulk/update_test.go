package pgbulk_test

import (
	"context"
	"testing"

	"github.com/kaichao/gopkg/pgbulk"

	"github.com/jackc/pgx/v5"
)

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgres://postgres:secret@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// 创建测试表
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_table (
			id SERIAL PRIMARY KEY,
			name TEXT,
			age INT,
			dept TEXT
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// 插入初始数据
	_, err = conn.Exec(ctx, `
		INSERT INTO test_table (name, age, dept) VALUES
			('Alice', 25, 'HR'),
			('Bob', 30, 'IT'),
			('Charlie', 28, 'HR');
	`)
	if err != nil {
		t.Fatalf("Failed to insert initial data: %v", err)
	}

	// 定义 SQL 模板
	sqlTemplate := "UPDATE test_table SET name = $1, age = $2 WHERE id = $3 AND dept = $4"

	// 定义 data 和 ids
	data := [][]interface{}{
		{"Alice Updated", 26},
		{"Bob Updated", 31},
	}
	ids := [][]interface{}{
		{1, "HR"},
		{2, "IT"},
	}

	// 执行更新
	err = pgbulk.Update(conn, sqlTemplate, data, ids)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 验证更新后的数据
	rows, err := conn.Query(ctx, "SELECT id, name, age, dept FROM test_table ORDER BY id")
	if err != nil {
		t.Fatalf("Failed to query updated data: %v", err)
	}
	defer rows.Close()

	expected := []struct {
		id   int
		name string
		age  int
		dept string
	}{
		{1, "Alice Updated", 26, "HR"},
		{2, "Bob Updated", 31, "IT"},
		{3, "Charlie", 28, "HR"},
	}

	i := 0
	for rows.Next() {
		var id, age int
		var name, dept string
		err := rows.Scan(&id, &name, &age, &dept)
		if err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		if id != expected[i].id || name != expected[i].name || age != expected[i].age || dept != expected[i].dept {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], struct {
				id   int
				name string
				age  int
				dept string
			}{id, name, age, dept})
		}
		i++
	}

	// 清理测试表
	_, err = conn.Exec(ctx, "DROP TABLE test_table")
	if err != nil {
		t.Fatalf("Failed to drop test table: %v", err)
	}
}
