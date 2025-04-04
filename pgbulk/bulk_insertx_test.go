package pgbulk_test

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kaichao/gopkg/pgbulk"
	"github.com/stretchr/testify/assert"
)

func TestBulkInsertReturningID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlTemplate := "INSERT INTO test_table (col1, col2) VALUES %s"
	data := [][]interface{}{
		{1, "a"},
		{2, "b"},
		{3, "c"},
	}

	// 模拟实际生成的 SQL 语句
	actualSQL := "INSERT INTO test_table (col1, col2) VALUES ($1,$2),($3,$4),($5,$6) RETURNING id"

	// 用 QuoteMeta 来确保正则安全匹配实际 SQL
	expectedSQL := regexp.QuoteMeta(actualSQL)

	rows := sqlmock.NewRows([]string{"id"}).
		AddRow(101).
		AddRow(102).
		AddRow(103)

	mock.ExpectQuery(expectedSQL).
		WithArgs(1, "a", 2, "b", 3, "c").
		WillReturnRows(rows)

	ids, err := pgbulk.BulkInsertReturningID(db, sqlTemplate, data)
	assert.NoError(t, err)
	assert.Equal(t, []int{101, 102, 103}, ids)

	assert.NoError(t, mock.ExpectationsWereMet())
}
