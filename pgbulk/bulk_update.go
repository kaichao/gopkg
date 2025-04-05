package pgbulk

import (
	"database/sql"
	"fmt"
	"strings"
)

// BulkUpdate performs a batch update into PostgreSQL.
func BulkUpdate(db *sql.DB, sqlTemplate string, ids []int, data [][]interface{}) error {
	if len(ids) == 0 || len(data) == 0 || len(data) != len(ids) {
		return fmt.Errorf("invalid input: ids and data must be non-empty and of the same length")
	}

	const maxParamLimit = 65535 // 硬编码为 PostgreSQL 默认限制
	paramsPerRow := len(data[0]) + 1
	maxBatchSize := maxParamLimit / paramsPerRow
	if maxBatchSize == 0 {
		maxBatchSize = 1
	}

	tableName, columnNames, err := parseSQLTemplate(sqlTemplate, len(data[0]))
	if err != nil {
		return fmt.Errorf("failed to parse sqlTemplate: %v", err)
	}

	for start := 0; start < len(ids); start += maxBatchSize {
		end := start + maxBatchSize
		if end > len(ids) {
			end = len(ids)
		}

		batchIDs := ids[start:end]
		batchData := data[start:end]
		batchSize := len(batchIDs)

		var queryBuilder strings.Builder
		queryBuilder.WriteString(fmt.Sprintf("UPDATE %s SET ", tableName))
		var args []interface{}
		paramIdx := 1

		for colIdx, colName := range columnNames {
			queryBuilder.WriteString(fmt.Sprintf("%s = CASE ", colName))
			for i := 0; i < batchSize; i++ {
				queryBuilder.WriteString(fmt.Sprintf("WHEN id = $%d THEN $%d ", paramIdx, paramIdx+1))
				args = append(args, batchIDs[i])
				args = append(args, batchData[i][colIdx])
				paramIdx += 2
			}
			queryBuilder.WriteString(fmt.Sprintf("ELSE %s END", colName))
			if colIdx < len(columnNames)-1 {
				queryBuilder.WriteString(", ")
			}
		}

		queryBuilder.WriteString(" WHERE id IN (")
		for i := 0; i < batchSize; i++ {
			if i > 0 {
				queryBuilder.WriteString(", ")
			}
			queryBuilder.WriteString(fmt.Sprintf("$%d", paramIdx))
			args = append(args, batchIDs[i])
			paramIdx++
		}
		queryBuilder.WriteString(")")

		query := queryBuilder.String()
		result, err := db.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("batch update execution error: %v", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("batch update error: %v", err)
		}
		if int(rowsAffected) != batchSize {
			return fmt.Errorf("expected %d rows affected, got %d", batchSize, rowsAffected)
		}
	}

	return nil
}

func parseSQLTemplate(sqlTemplate string, expectedColumnCount int) (string, []string, error) {
	parts := strings.SplitN(strings.TrimSpace(sqlTemplate), "SET", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid sqlTemplate format: missing SET clause")
	}

	updateParts := strings.Fields(parts[0])
	if len(updateParts) < 2 || strings.ToUpper(updateParts[0]) != "UPDATE" {
		return "", nil, fmt.Errorf("invalid sqlTemplate format: must start with UPDATE")
	}
	tableName := updateParts[1]

	setWhereParts := strings.SplitN(parts[1], "WHERE", 2)
	if len(setWhereParts) != 2 {
		return "", nil, fmt.Errorf("invalid sqlTemplate format: missing WHERE clause")
	}
	setClause := strings.TrimSpace(setWhereParts[0])
	assignments := strings.Split(setClause, ",")
	var columnNames []string
	for _, assignment := range assignments {
		assignment = strings.TrimSpace(assignment)
		if assignment == "" {
			continue
		}
		colParts := strings.SplitN(assignment, "=", 2)
		if len(colParts) != 2 {
			return "", nil, fmt.Errorf("invalid assignment in SET clause: %s", assignment)
		}
		columnNames = append(columnNames, strings.TrimSpace(colParts[0]))
	}

	if len(columnNames) != expectedColumnCount {
		return "", nil, fmt.Errorf("parsed column count (%d) does not match expected (%d)", len(columnNames), expectedColumnCount)
	}

	return tableName, columnNames, nil
}
