package pgbulk

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Update performs a bulk update using the provided SQL template, data, and ids.
func Update(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, ids [][]interface{}) error {
	if len(data) != len(ids) {
		return fmt.Errorf("data and ids must have the same number of rows")
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

	// Check each result
	for i := 0; i < batch.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("batch execution failed at record %d: %v", i, err)
		}
	}
	return nil
}
