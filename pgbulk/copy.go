package pgbulk

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	pgx "github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
)

// Copy performs a batch insert into PostgreSQL using pgx's CopyFrom
func Copy(conn *pgx.Conn, sqlTemplate string, data [][]interface{}) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	re := regexp.MustCompile(`INSERT\s+INTO\s+(\w+)\s*\(([^)]*)\)`)
	matches := re.FindStringSubmatch(sqlTemplate)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid sqlTemplate format")
	}
	tableName := matches[1]
	columns := strings.Split(matches[2], ",")
	for i := range columns {
		columns[i] = strings.TrimSpace(columns[i])
	}

	copyCount, err := conn.CopyFrom(
		context.Background(),
		pgx.Identifier{tableName},
		columns,
		pgx.CopyFromRows(data),
	)
	if err != nil {
		logrus.Errorf("COPY execution error: %v", err)
		return 0, err
	}

	logrus.Infof("Total copied: %d rows.", copyCount)
	return int(copyCount), nil
}

/*
// Copy performs a batch insert into PostgreSQL using the COPY command and returns the number of inserted rows.
func Copy1(db *sql.DB, sqlTemplate string, data [][]interface{}) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	// Extract table name and columns from sqlTemplate
	// Expected format: "INSERT INTO table_name (col1, col2, ...)" or "INSERT INTO table_name ()"
	re := regexp.MustCompile(`INSERT\s+INTO\s+(\w+)\s*\(([^)]*)\)`)
	matches := re.FindStringSubmatch(sqlTemplate)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid sqlTemplate format")
	}
	tableName := matches[1]
	columns := strings.TrimSpace(matches[2])
	if columns == "" {
		return 0, fmt.Errorf("no columns specified in sqlTemplate")
	}
	columnList := strings.Split(columns, ",")
	for i := range columnList {
		columnList[i] = strings.TrimSpace(columnList[i])
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		logrus.Errorf("Failed to begin transaction: %v", err)
		return 0, err
	}

	// Prepare COPY command
	copyStmt := fmt.Sprintf("COPY %s (%s) FROM STDIN WITH (FORMAT CSV)", tableName, columns)
	stmt, err := tx.Prepare(copyStmt)
	if err != nil {
		tx.Rollback()
		logrus.Errorf("Failed to prepare COPY statement: %v", err)
		return 0, err
	}
	defer stmt.Close()

	// Create buffer for CSV data
	var buffer bytes.Buffer
	for _, row := range data {
		if len(row) != len(columnList) {
			tx.Rollback()
			return 0, fmt.Errorf("row length %d does not match column count %d", len(row), len(columnList))
		}
		for i, val := range row {
			if i > 0 {
				buffer.WriteString(",")
			}
			switch v := val.(type) {
			case nil:
				// Write empty for NULL
			case string:
				// Escape quotes and wrap in quotes
				escaped := strings.ReplaceAll(v, `"`, `""`)
				buffer.WriteString(`"`)
				buffer.WriteString(escaped)
				buffer.WriteString(`"`)
			case int, int32, int64, float64, bool:
				fmt.Fprintf(&buffer, "%v", v)
			case time.Time:
				// Format timestamp as YYYY-MM-DD HH:MM:SS.000000
				formatted := v.Format("2006-01-02 15:04:05.999999")
				buffer.WriteString(`"`)
				buffer.WriteString(formatted)
				buffer.WriteString(`"`)
			case []time.Time:
				// Format timestamp array as {"YYYY-MM-DD HH:MM:SS.000000","YYYY-MM-DD HH:MM:SS.000000"}
				if len(v) == 0 {
					buffer.WriteString(`"{}"`)
				} else {
					var elements []string
					for _, t := range v {
						formatted := t.Format("2006-01-02 15:04:05.999999")
						escaped := strings.ReplaceAll(formatted, `"`, `""`)
						elements = append(elements, `"`+escaped+`"`)
					}
					arrayStr := "{" + strings.Join(elements, ",") + "}"
					buffer.WriteString(`"`)
					buffer.WriteString(strings.ReplaceAll(arrayStr, `"`, `""`))
					buffer.WriteString(`"`)
				}
			case sql.NullString:
				if v.Valid {
					escaped := strings.ReplaceAll(v.String, `"`, `""`)
					buffer.WriteString(`"`)
					buffer.WriteString(escaped)
					buffer.WriteString(`"`)
				}
			case sql.NullInt64:
				if v.Valid {
					fmt.Fprintf(&buffer, "%d", v.Int64)
				}
			case sql.NullFloat64:
				if v.Valid {
					fmt.Fprintf(&buffer, "%f", v.Float64)
				}
			case sql.NullBool:
				if v.Valid {
					fmt.Fprintf(&buffer, "%t", v.Bool)
				}
			case sql.NullTime:
				if v.Valid {
					formatted := v.Time.Format("2006-01-02 15:04:05.999999")
					buffer.WriteString(`"`)
					buffer.WriteString(formatted)
					buffer.WriteString(`"`)
				}
			case map[string]interface{}, map[string]string, []interface{}:
				// Serialize jsonb as JSON string
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					tx.Rollback()
					return 0, fmt.Errorf("failed to marshal jsonb: %v", err)
				}
				escaped := strings.ReplaceAll(string(jsonBytes), `"`, `""`)
				buffer.WriteString(`"`)
				buffer.WriteString(escaped)
				buffer.WriteString(`"`)
			default:
				tx.Rollback()
				return 0, fmt.Errorf("unsupported data type: %T", v)
			}
		}
		buffer.WriteString("\n")
	}

	// Execute COPY
	result, err := stmt.Exec(buffer.String())
	if err != nil {
		tx.Rollback()
		logrus.Errorf("COPY execution error: %v", err)
		return 0, err
	}

	// Get number of inserted rows
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		logrus.Errorf("Failed to get rows affected: %v", err)
		return 0, err
	}
	if rowsAffected != int64(len(data)) {
		tx.Rollback()
		return 0, fmt.Errorf("expected to insert %d rows, but inserted %d", len(data), rowsAffected)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		logrus.Errorf("Failed to commit transaction: %v", err)
		return 0, err
	}

	logrus.Infof("Total copied: %d rows.", rowsAffected)
	return int(rowsAffected), nil
}
*/
