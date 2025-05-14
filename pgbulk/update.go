package pgbulk

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Update performs a bulk update using the provided SQL template, data, and ids.
// 返回值：(error, [][]interface{})，第二个参数为失败记录的 ids
func Update(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, ids [][]interface{}) ([][]interface{}, error) {
	if len(data) != len(ids) {
		return nil, fmt.Errorf("data and ids must have the same number of rows")
	}

	if len(data) == 0 {
		return nil, nil
	}

	// 修改：使用带超时的上下文，限制事务时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}
	for i := 0; i < len(data); i++ {
		params := append(data[i], ids[i]...)
		batch.Queue(sqlTemplate, params...)
	}

	// 通过事务发送批量操作
	br := tx.SendBatch(ctx, batch)

	failedIds := [][]interface{}{}

	// 修改：检查每个结果，确保捕获唯一约束错误
	for i := 0; i < batch.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			failedIds = append(failedIds, ids[i])
			// 立即关闭批量操作，确保资源释放
			br.Close()
			return failedIds, fmt.Errorf("batch execution failed for record %d: %v", i, err)
		}
	}

	// 关闭批量操作
	if err := br.Close(); err != nil {
		return failedIds, fmt.Errorf("failed to close batch: %v", err)
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return failedIds, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil, nil
}
