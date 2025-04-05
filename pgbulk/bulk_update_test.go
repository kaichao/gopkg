package pgbulk_test

import (
	"database/sql"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"github.com/kaichao/gopkg/pgbulk"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func TestMockBulkUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlTemplate := "UPDATE users SET name = ?, age = ? WHERE id IN (?)"
	ids := []int{1, 2}
	data := [][]interface{}{
		{"Alice", 30},
		{"Bob", 25},
	}

	// 实际执行生成的 SQL（已知准确）
	// 	actualSQL := `
	// UPDATE users SET
	//   name = CASE WHEN id = $1 THEN $2 WHEN id = $3 THEN $4 ELSE name END,
	//   age = CASE WHEN id = $5 THEN $6 WHEN id = $7 THEN $8 ELSE age END
	// WHERE id IN ($9, $10)`

	// 去掉换行 + 正则转义，防止匹配失败
	cleanSQL := regexp.QuoteMeta(`
UPDATE users SET name = CASE WHEN id = $1 THEN $2 WHEN id = $3 THEN $4 ELSE name END, age = CASE WHEN id = $5 THEN $6 WHEN id = $7 THEN $8 ELSE age END WHERE id IN ($9, $10)`)

	mock.ExpectExec(cleanSQL).
		WithArgs(1, "Alice", 2, "Bob", 1, 30, 2, 25, 1, 2).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err = pgbulk.BulkUpdate(db, sqlTemplate, ids, data)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ExampleBulkUpdate demonstrates how to use BulkUpdate to update multiple rows in a PostgreSQL table.
func ExampleBulkUpdate() {
	// 假设已有一个 PostgreSQL 数据库连接
	db, err := sql.Open("postgres", "user=postgres password=mysecretpassword dbname=test sslmode=disable")
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

	// 插入初始数据
	_, err = db.Exec(`INSERT INTO users (name, age) VALUES 
		('Alice', 25), 
		('Bob', 30), 
		('Charlie', 35)`)
	if err != nil {
		fmt.Println("Failed to insert initial data:", err)
		return
	}

	// 定义要更新的数据
	ids := []int{1, 2, 3}
	data := [][]interface{}{
		{"Alice Updated", 26},
		{"Bob Updated", 31},
		{"Charlie Updated", 36},
	}

	// SQL 模板
	sqlTemplate := "UPDATE users SET name = ?, age = ? WHERE id = ?"

	// 执行批量更新
	err = pgbulk.BulkUpdate(db, sqlTemplate, ids, data)
	if err != nil {
		fmt.Println("Bulk update failed:", err)
		return
	}

	// 验证更新结果
	rows, err := db.Query("SELECT id, name, age FROM users ORDER BY id")
	if err != nil {
		fmt.Println("Query failed:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		var age int
		err = rows.Scan(&id, &name, &age)
		if err != nil {
			fmt.Println("Scan failed:", err)
			return
		}
		fmt.Printf("ID: %d, Name: %s, Age: %d\n", id, name, age)
	}

	// 输出:
	// ID: 1, Name: Alice Updated, Age: 26
	// ID: 2, Name: Bob Updated, Age: 31
	// ID: 3, Name: Charlie Updated, Age: 36
}

func TestBulkUpdate(t *testing.T) {
	db, err := sql.Open("postgres", "user=postgres password=secret dbname=postgres sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 验证数据库连接
	err = db.Ping()
	if err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}

	// 创建测试表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name TEXT,
			age INT
		)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 插入初始数据
	_, err = db.Exec(`INSERT INTO users (name, age) VALUES ('TestUser', 20)`)
	if err != nil {
		t.Fatalf("Failed to insert initial data: %v", err)
	}

	// 准备更新数据
	ids := []int{1}
	data := [][]interface{}{{"Test Updated", 21}}
	sqlTemplate := "UPDATE users SET name = ?, age = ? WHERE id = ?"

	// 执行批量更新
	err = pgbulk.BulkUpdate(db, sqlTemplate, ids, data)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 验证更新结果
	var name string
	var age int
	err = db.QueryRow("SELECT name, age FROM users WHERE id = 1").Scan(&name, &age)
	if err != nil {
		t.Errorf("Failed to query updated row: %v", err)
	}
	if name != "Test Updated" || age != 21 {
		t.Errorf("Expected name='Test Updated' and age=21, got name='%s' and age=%d", name, age)
	}

	// 清理测试环境
	_, err = db.Exec("DROP TABLE users")
	if err != nil {
		t.Errorf("Failed to drop table: %v", err)
	}
}
