package pgbulk_test

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kaichao/gopkg/pgbulk"
	"github.com/stretchr/testify/assert"
)

func TestBulkInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	data := [][]interface{}{
		{"Alice", 30},
		{"Bob", 25},
		{"Charlie", 28},
	}

	sqlTemplate := "INSERT INTO test_bulk_insert (name, age) VALUES %s"

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO test_bulk_insert (name, age) VALUES")).
		WithArgs("Alice", 30, "Bob", 25, "Charlie", 28).
		WillReturnResult(sqlmock.NewResult(0, int64(len(data))))
	err = pgbulk.BulkInsert(db, sqlTemplate, data)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
