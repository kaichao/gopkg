package pgbulk

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/errors"
)

// Update performs a bulk update using the provided SQL template, data, and ids.
// Returns: (error, [][]interface{}), where second parameter is failed record ids
func Update(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, ids [][]interface{}) ([][]interface{}, error) {
	if len(data) != len(ids) {
		return nil, errors.E("data and ids must have the same number of rows")
	}

	if len(data) == 0 {
		return nil, nil
	}

	// Modification: use timeout context to limit transaction time
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start transaction
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

	// Send batch operations through transaction
	br := tx.SendBatch(ctx, batch)

	failedIds := [][]interface{}{}

	// Modification: check each result to ensure capturing unique constraint errors
	for i := 0; i < batch.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			failedIds = append(failedIds, ids[i])
			// Immediately close batch operation to ensure resource release
			br.Close()
			return failedIds, errors.WrapE(err, "batch execution", "record-num", i)
		}
	}

	// Close batch operation
	if err := br.Close(); err != nil {
		return failedIds, errors.WrapE(err, "close batch")

	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return failedIds, errors.WrapE(err, "commit transaction")
	}

	return nil, nil
}
