package pgbulk

import (
	"database/sql"
	"fmt"
	"strings"
)

// BulkInsertReturningID performs batch inserts and returns generated IDs (e.g., from a SERIAL or IDENTITY column).
func BulkInsertReturningID(db *sql.DB, sqlTemplate string, data [][]interface{}, returningColumn ...string) ([]int, error) {
	if len(data) == 0 {
		return nil, nil
	}

	paramsPerRow := len(data[0])
	if paramsPerRow == 0 {
		return nil, nil
	}

	const maxParamLimit = 65535
	maxBatchSize := maxParamLimit / paramsPerRow
	if maxBatchSize == 0 {
		maxBatchSize = 1
	}

	// 默认返回列为 "id"，如果指定则使用用户提供的列
	retCol := "id"
	if len(returningColumn) > 0 && returningColumn[0] != "" {
		retCol = returningColumn[0]
	}

	var insertedIDs []int

	// 修改 3: 添加事务支持，确保原子性
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback() // 如果未提交，则回滚

	for start := 0; start < len(data); start += maxBatchSize {
		end := start + maxBatchSize
		if end > len(data) {
			end = len(data)
		}
		batch := data[start:end]

		var placeholders []string
		var args []interface{}
		argIdx := 1
		for _, row := range batch {
			valuePlaceholders := make([]string, len(row))
			for i := range row {
				valuePlaceholders[i] = fmt.Sprintf("$%d", argIdx)
				argIdx++
			}
			placeholders = append(placeholders, fmt.Sprintf("(%s)", strings.Join(valuePlaceholders, ",")))
			args = append(args, row...)
		}

		// 构造查询，包含 RETURNING 子句
		query := fmt.Sprintf("%s VALUES %s RETURNING %s", sqlTemplate, strings.Join(placeholders, ","), retCol)

		rows, err := tx.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("batch insert returning ID error: %v", err)
		}
		defer rows.Close()

		batchInsertedIDs := make([]int, 0, len(batch))
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				return nil, fmt.Errorf("scan error: %v", err)
			}
			batchInsertedIDs = append(batchInsertedIDs, id)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("rows error: %v", err)
		}

		// 修改 4: 验证返回的 ID 数量与批次大小匹配
		if len(batchInsertedIDs) != len(batch) {
			return nil, fmt.Errorf("expected %d IDs, got %d", len(batch), len(batchInsertedIDs))
		}

		insertedIDs = append(insertedIDs, batchInsertedIDs...)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return insertedIDs, nil
}
