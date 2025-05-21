package pgbulk

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// InsertReturningID 插入数据并返回插入行的 ID
func InsertReturningID(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, returningColumn ...string) ([]int, error) {
	// 默认返回的列名为 "id"
	returning := "id"
	if len(returningColumn) > 0 {
		returning = returningColumn[0]
	}

	// 构建 VALUES 部分
	var valuePlaceholders []string
	for i := range data {
		var placeholders []string
		for j := 0; j < len(data[i]); j++ {
			placeholders = append(placeholders, fmt.Sprintf("$%d", i*len(data[i])+j+1))
		}
		valuePlaceholders = append(valuePlaceholders, "("+strings.Join(placeholders, ",")+")")
	}
	valuesClause := strings.Join(valuePlaceholders, ",")

	// 构建完整的 SQL 语句，包含 RETURNING 子句
	fullSQL := fmt.Sprintf("%s VALUES %s RETURNING %s", sqlTemplate, valuesClause, returning)

	// 准备参数
	var args []interface{}
	for _, row := range data {
		args = append(args, row...)
	}

	// 执行 SQL 语句并获取返回的 ID
	rows, err := conn.Query(context.Background(), fullSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("插入失败: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("扫描返回的 ID 失败: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取行失败: %w", err)
	}

	return ids, nil
}
