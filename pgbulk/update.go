package pgbulk

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/errors"
)

// Update performs a bulk update using the provided SQL template, data, and ids.
// 返回值：(error, [][]interface{})，第二个参数为失败记录的 ids
func Update(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, ids [][]interface{}) ([][]interface{}, error) {
	if len(data) != len(ids) {
		return nil, errors.E("data and ids must have the same number of rows")
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
		return nil, errors.WrapE(err, "start transaction")
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
			return failedIds, errors.WrapE(err, "batch execution", "record-num", i)
		}
	}

	// 关闭批量操作
	if err := br.Close(); err != nil {
		return failedIds, errors.WrapE(err, "close batch")

	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return failedIds, errors.WrapE(err, "commit transaction")
	}

	return nil, nil
}
