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
