package pgbulk

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Update performs a bulk update using the provided SQL template, data, and ids.
// 修改：返回值从 error 改为 (error, int)，返回成功更新的记录数
func Update(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, ids [][]interface{}) (error, int) {
	if len(data) != len(ids) {
		return fmt.Errorf("data and ids must have the same number of rows"), 0
	}

	if len(data) == 0 {
		return nil, 0
	}

	batch := &pgx.Batch{}
	for i := 0; i < len(data); i++ {
		// Combine data[i] and ids[i] for each record
		params := append(data[i], ids[i]...)
		batch.Queue(sqlTemplate, params...)
	}

	// Send the batch and get results
	br := conn.SendBatch(context.Background(), batch)
	defer br.Close()

	// 新增：统计成功更新的记录数
	rowsAffected := 0

	// Check each result
	for i := 0; i < batch.Len(); i++ {
		ct, err := br.Exec()
		if err != nil {
			// 修改：返回当前 rowsAffected
			return fmt.Errorf("batch execution failed at record %d: %v", i, err), rowsAffected
		}
		// 新增：累加受影响的行数
		rowsAffected += int(ct.RowsAffected())
	}
	// 修改：返回 nil 和 rowsAffected
	return nil, rowsAffected
}
