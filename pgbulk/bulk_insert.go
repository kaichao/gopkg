package pgbulk

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// BulkInsert performs a regular batch insert into PostgreSQL.
func BulkInsert(db *sql.DB, sqlTemplate string, data [][]interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("data is empty")
	}

	paramsPerRow := len(data[0])
	maxBatchSize := 65535 / paramsPerRow
	logrus.Infof("Calculated max batch size: %d rows per batch", maxBatchSize)

	for start := 0; start < len(data); start += maxBatchSize {
		end := start + maxBatchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[start:end]
		var placeholders []string
		var args []interface{}

		for i, row := range batch {
			placeholders = append(placeholders, fmt.Sprintf("(%s)", strings.Join(makePlaceholders(len(row), i*len(row)), ",")))
			args = append(args, row...)
		}

		// query := fmt.Sprintf(sqlTemplate, strings.Join(placeholders, ","))
		query := fmt.Sprintf("%s VALUES %s", sqlTemplate, strings.Join(placeholders, ","))
		if _, err := db.Exec(query, args...); err != nil {
			logrus.Errorf("Batch insert execution error: %v", err)
			return err
		}
		logrus.Infof("Batch insert completed for %d rows.", len(batch))
	}

	logrus.Infof("Total inserted: %d rows.", len(data))
	return nil
}

// makePlaceholders generates placeholders like $1, $2, ..., $N
func makePlaceholders(count, offset int) []string {
	placeholders := make([]string, count)
	for i := 0; i < count; i++ {
		placeholders[i] = fmt.Sprintf("$%d", i+1+offset)
	}
	return placeholders
}
