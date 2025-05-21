package pgbulk

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Insert 使用提供的 SQL 模板和数据将数据插入数据库
func Insert(conn *pgx.Conn, sqlTemplate string, data [][]interface{}) error {
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

	// 构建完整的 SQL 语句
	fullSQL := fmt.Sprintf("%s VALUES %s", sqlTemplate, valuesClause)

	// 准备参数
	var args []interface{}
	for _, row := range data {
		args = append(args, row...)
	}

	// 执行 SQL 语句
	_, err := conn.Exec(context.Background(), fullSQL, args...)
	if err != nil {
		return fmt.Errorf("插入失败: %w", err)
	}

	return nil
}
