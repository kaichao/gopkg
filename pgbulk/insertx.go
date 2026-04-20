package pgbulk

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/errors"
)

// InsertReturningID inserts data and returns IDs of inserted rows
// Parameters:
//   - returningColumnAndOnConflict: variadic parameter
//   - If 0 parameters provided: uses default returning column "id"
//   - If 1 parameter provided: first parameter is returning column name
//   - If 2 parameters provided: first parameter is returning column name, second is ON CONFLICT clause
func InsertReturningID(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, returningColumnAndOnConflict ...string) ([]int, error) {
	// Default returning column is "id"
	returning := "id"
	if len(returningColumnAndOnConflict) > 0 {
		returning = returningColumnAndOnConflict[0]
	}

	// Check for ON CONFLICT clause
	onConflict := ""
	if len(returningColumnAndOnConflict) > 1 {
		onConflict = returningColumnAndOnConflict[1]
	}

	// Build VALUES part
	var valuePlaceholders []string
	for i := range data {
		var placeholders []string
		for j := 0; j < len(data[i]); j++ {
			placeholders = append(placeholders, fmt.Sprintf("$%d", i*len(data[i])+j+1))
		}
		valuePlaceholders = append(valuePlaceholders, "("+strings.Join(placeholders, ",")+")")
	}
	valuesClause := strings.Join(valuePlaceholders, ",")

	// Build complete SQL statement
	var fullSQL string
	if onConflict != "" {
		fullSQL = fmt.Sprintf("%s VALUES %s %s RETURNING %s", sqlTemplate, valuesClause, onConflict, returning)
	} else {
		fullSQL = fmt.Sprintf("%s VALUES %s RETURNING %s", sqlTemplate, valuesClause, returning)
	}

	// Prepare parameters
	var args []interface{}
	for _, row := range data {
		args = append(args, row...)
	}

	// Execute SQL statement and retrieve returned IDs
	rows, err := conn.Query(context.Background(), fullSQL, args...)
	if err != nil {
		return nil, errors.WrapE(err, "insert", "full-sql", fullSQL)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, errors.WrapE(err, "rows.Scan()")
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.WrapE(err, "rows.Next()")
	}

	return ids, nil
}
