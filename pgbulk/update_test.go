package pgbulk_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/pgbulk"
)

// setupTable 创建测试表并插入初始数据
// 修改：将 ctx 放在第一个参数，t 放在第二个参数
func setupTable(ctx context.Context, t *testing.T, conn *pgx.Conn) {
	_, err := conn.Exec(ctx, `
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

	_, err = conn.Exec(ctx, `
		INSERT INTO test_table (name, age, dept) VALUES
			('Alice', 25, 'HR'),
			('Bob', 30, 'IT'),
			('Charlie', 28, 'HR');
	`)
	if err != nil {
		t.Fatalf("Failed to insert initial data: %v", err)
	}
}

// cleanupTable 删除测试表
// 修改：将 ctx 放在第一个参数，t 放在第二个参数
func cleanupTable(ctx context.Context, t *testing.T, conn *pgx.Conn) {
	_, err := conn.Exec(ctx, "DROP TABLE IF EXISTS test_table")
	if err != nil {
		t.Fatalf("Failed to drop test table: %v", err)
	}
}

// verifyData 验证数据库中的数据
// 修改：将 ctx 放在第一个参数，t 放在第二个参数
func verifyData(ctx context.Context, t *testing.T, conn *pgx.Conn, expected []struct {
	id   int
	name string
	age  int
	dept string
}) {
	rows, err := conn.Query(ctx, "SELECT id, name, age, dept FROM test_table ORDER BY id")
	if err != nil {
		t.Fatalf("Failed to query updated data: %v", err)
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		var id, age int
		var name, dept string
		err := rows.Scan(&id, &name, &age, &dept)
		if err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		if i >= len(expected) {
			t.Errorf("Unexpected row: got %v", struct {
				id   int
				name string
				age  int
				dept string
			}{id, name, age, dept})
			continue
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
	if i != len(expected) {
		t.Errorf("Expected %d rows, got %d", len(expected), i)
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgres://postgres:secret@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	sqlTemplate := "UPDATE test_table SET name = $1, age = $2 WHERE id = $3 AND dept = $4"

	t.Run("Successful Batch Update", func(t *testing.T) {
		// 修改：调整 setupTable 和 cleanupTable 的参数顺序为 (ctx, t, conn)
		setupTable(ctx, t, conn)
		defer cleanupTable(ctx, t, conn)

		// 定义 data 和 ids
		data := [][]interface{}{
			{"Alice Updated", 26},
			{"Bob Updated", 31},
		}
		ids := [][]interface{}{
			{1, "HR"},
			{2, "IT"},
		}

		err, rowsAffected := pgbulk.Update(conn, sqlTemplate, data, ids)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if rowsAffected != 2 {
			t.Errorf("Expected %d rows affected, got %d", 2, rowsAffected)
		}

		// 修改：调整 verifyData 的参数顺序为 (ctx, t, conn, expected)
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
		verifyData(ctx, t, conn, expected)
	})

	t.Run("Mismatched Parameter Rows", func(t *testing.T) {
		// 修改：调整 setupTable 和 cleanupTable 的参数顺序
		setupTable(ctx, t, conn)
		defer cleanupTable(ctx, t, conn)

		// 定义 data 和 ids，行数不匹配
		data := [][]interface{}{
			{"Alice Updated", 26},
		}
		ids := [][]interface{}{
			{1, "HR"},
			{2, "IT"},
		}

		err, rowsAffected := pgbulk.Update(conn, sqlTemplate, data, ids)
		if err == nil {
			t.Error("Expected error due to mismatched parameter rows, got nil")
		}
		if rowsAffected != 0 {
			t.Errorf("Expected 0 rows affected, got %d", rowsAffected)
		}
	})

	t.Run("Invalid SQL Template", func(t *testing.T) {
		// 修改：调整 setupTable 和 cleanupTable 的参数顺序
		setupTable(ctx, t, conn)
		defer cleanupTable(ctx, t, conn)

		// 使用无效的 SQL 模板
		invalidSQL := "UPDATE test_table SET name = $1, age = $2 WHERE nonexistent_column = $3"
		data := [][]interface{}{
			{"Alice Updated", 26},
		}
		ids := [][]interface{}{
			{1},
		}

		err, rowsAffected := pgbulk.Update(conn, invalidSQL, data, ids)
		if err == nil {
			t.Error("Expected error due to invalid SQL template, got nil")
		}
		if rowsAffected != 0 {
			t.Errorf("Expected 0 rows affected, got %d", rowsAffected)
		}
	})

	t.Run("Empty Data", func(t *testing.T) {
		// 修改：调整 setupTable 和 cleanupTable 的参数顺序
		setupTable(ctx, t, conn)
		defer cleanupTable(ctx, t, conn)

		// 定义空的 data 和 ids
		data := [][]interface{}{}
		ids := [][]interface{}{}

		err, rowsAffected := pgbulk.Update(conn, sqlTemplate, data, ids)
		if err != nil {
			t.Errorf("Expected no error for empty data, got %v", err)
		}
		if rowsAffected != 0 {
			t.Errorf("Expected 0 rows affected, got %d", rowsAffected)
		}

		// 修改：调整 verifyData 的参数顺序
		expected := []struct {
			id   int
			name string
			age  int
			dept string
		}{
			{1, "Alice", 25, "HR"},
			{2, "Bob", 30, "IT"},
			{3, "Charlie", 28, "HR"},
		}
		verifyData(ctx, t, conn, expected)
	})

	t.Run("Partial Update", func(t *testing.T) {
		// 修改：调整 setupTable 和 cleanupTable 的参数顺序
		setupTable(ctx, t, conn)
		defer cleanupTable(ctx, t, conn)

		// 定义 data 和 ids，只更新部分行
		data := [][]interface{}{
			{"Charlie Updated", 29},
		}
		ids := [][]interface{}{
			{3, "HR"},
		}

		err, rowsAffected := pgbulk.Update(conn, sqlTemplate, data, ids)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if rowsAffected != 1 {
			t.Errorf("Expected 1 row affected, got %d", rowsAffected)
		}

		// 修改：调整 verifyData 的参数顺序
		expected := []struct {
			id   int
			name string
			age  int
			dept string
		}{
			{1, "Alice", 25, "HR"},
			{2, "Bob", 30, "IT"},
			{3, "Charlie Updated", 29, "HR"},
		}
		verifyData(ctx, t, conn, expected)
	})

	t.Run("No Rows Affected", func(t *testing.T) {
		// 修改：调整 setupTable 和 cleanupTable 的参数顺序
		setupTable(ctx, t, conn)
		defer cleanupTable(ctx, t, conn)

		// 定义 data 和 ids，匹配不存在的条件
		data := [][]interface{}{
			{"Nonexistent", 99},
		}
		ids := [][]interface{}{
			{999, "Nonexistent"},
		}

		err, rowsAffected := pgbulk.Update(conn, sqlTemplate, data, ids)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if rowsAffected != 0 {
			t.Errorf("Expected 0 rows affected, got %d", rowsAffected)
		}

		// 修改：调整 verifyData 的参数顺序
		expected := []struct {
			id   int
			name string
			age  int
			dept string
		}{
			{1, "Alice", 25, "HR"},
			{2, "Bob", 30, "IT"},
			{3, "Charlie", 28, "HR"},
		}
		verifyData(ctx, t, conn, expected)
	})
}

func TestUpdateExample(t *testing.T) {
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
	err, _ = pgbulk.Update(conn, sqlTemplate, data, ids)
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
