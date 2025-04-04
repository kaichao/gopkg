package pgbulk

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// BulkInsertReturningID performs batch inserts and returns generated IDs (e.g., from a SERIAL or IDENTITY column).
// BulkInsertReturningID performs batch insert and returns generated IDs.
func BulkInsertReturningID(db *sql.DB, sqlTemplate string, data [][]interface{}) ([]int, error) {
	if len(data) == 0 {
		logrus.Warn("No data provided for batch insert.")
		return nil, nil
	}

	paramsPerRow := len(data[0])
	if paramsPerRow == 0 {
		logrus.Warn("Empty rows provided for batch insert.")
		return nil, nil
	}
	maxBatchSize := 65535 / paramsPerRow
	logrus.Infof("Calculated max batch size: %d rows per batch", maxBatchSize)

	var insertedIDs []int

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

		query := fmt.Sprintf(sqlTemplate, strings.Join(placeholders, ",")) + " RETURNING id"
		logrus.Infof("Executing SQL: %s", query)

		rows, err := db.Query(query, args...)
		if err != nil {
			logrus.Errorf("Batch insert returning ID error: %v", err)
			return nil, err
		}
		defer rows.Close()

		batchInsertedIDs := []int{}
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				logrus.Errorf("Scan error: %v", err)
				return nil, err
			}
			batchInsertedIDs = append(batchInsertedIDs, id)
		}
		if err := rows.Err(); err != nil {
			logrus.Errorf("Rows error: %v", err)
			return nil, err
		}

		insertedIDs = append(insertedIDs, batchInsertedIDs...)
		logrus.Infof("Batch insert completed for %d rows. Inserted IDs: %v", len(batchInsertedIDs), batchInsertedIDs)
	}

	logrus.Infof("Total inserted: %d rows.", len(insertedIDs))
	return insertedIDs, nil
}
