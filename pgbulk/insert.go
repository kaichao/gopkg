package pgbulk

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/kaichao/gopkg/errors"
)

// Insert inserts data into database using provided SQL template and data
// Parameters:
//   - onConflict: variadic parameter
//   - If 0 parameters provided: no ON CONFLICT clause used
//   - If 1 parameter provided: first parameter is ON CONFLICT clause
func Insert(conn *pgx.Conn, sqlTemplate string, data [][]interface{}, onConflict ...string) error {
	// Check for ON CONFLICT clause
	conflictClause := ""
	if len(onConflict) > 0 {
		conflictClause = onConflict[0]
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
	if conflictClause != "" {
		fullSQL = fmt.Sprintf("%s VALUES %s %s", sqlTemplate, valuesClause, conflictClause)
	} else {
		fullSQL = fmt.Sprintf("%s VALUES %s", sqlTemplate, valuesClause)
	}

	// Prepare parameters
	var args []interface{}
	for _, row := range data {
		args = append(args, row...)
	}

	// Execute SQL statement
	_, err := conn.Exec(context.Background(), fullSQL, args...)
	return errors.WrapE(err, "pgx insert", "sql-template", sqlTemplate)
}
