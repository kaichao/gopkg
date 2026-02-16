package pgbulk

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/errors"
)

// InsertReturningID 插入数据并返回插入行的 ID
// 参数说明：
//   - returningColumnAndOnConflict: 可变长参数
//   - 如果提供0个参数：使用默认返回列 "id"
//   - 如果提供1个参数：第一个参数为返回列名
//   - 如果提供2个参数：第一个参数为返回列名，第二个参数为ON CONFLICT子句
func InsertReturningID(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, returningColumnAndOnConflict ...string) ([]int, error) {
	// 默认返回的列名为 "id"
	returning := "id"
	if len(returningColumnAndOnConflict) > 0 {
		returning = returningColumnAndOnConflict[0]
	}

	// 检查是否有ON CONFLICT子句
	onConflict := ""
	if len(returningColumnAndOnConflict) > 1 {
		onConflict = returningColumnAndOnConflict[1]
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
	if onConflict != "" {
		fullSQL = fmt.Sprintf("%s VALUES %s %s RETURNING %s", sqlTemplate, valuesClause, onConflict, returning)
	} else {
		fullSQL = fmt.Sprintf("%s VALUES %s RETURNING %s", sqlTemplate, valuesClause, returning)
	}

	// 准备参数
	var args []interface{}
	for _, row := range data {
		args = append(args, row...)
	}

	// 执行 SQL 语句并获取返回的 ID
	rows, err := conn.Query(context.Background(), fullSQL, args...)
	if err != nil {
		return nil, errors.WrapE(err, "insert", "full-sql", fullSQL)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, errors.WrapE(err, " rows.Scan()")
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.WrapE(err, " rows.Next()")
	}

	return ids, nil
}
