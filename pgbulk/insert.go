package pgbulk

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/errors"
)

// Insert 使用提供的 SQL 模板和数据将数据插入数据库
// 参数说明：
//   - onConflict: 可变长参数
//   - 如果提供0个参数：不使用ON CONFLICT子句
//   - 如果提供1个参数：第一个参数为ON CONFLICT子句
func Insert(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, onConflict ...string) error {
	// 检查是否有ON CONFLICT子句
	conflictClause := ""
	if len(onConflict) > 0 {
		conflictClause = onConflict[0]
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

	// 构建完整的 SQL 语句
	var fullSQL string
	if conflictClause != "" {
		fullSQL = fmt.Sprintf("%s VALUES %s %s", sqlTemplate, valuesClause, conflictClause)
	} else {
		fullSQL = fmt.Sprintf("%s VALUES %s", sqlTemplate, valuesClause)
	}

	// 准备参数
	var args []interface{}
	for _, row := range data {
		args = append(args, row...)
	}

	// 执行 SQL 语句
	_, err := conn.Exec(context.Background(), fullSQL, args...)
	return errors.WrapE(err, "pgx insert", "sql-template", sqlTemplate)
}
