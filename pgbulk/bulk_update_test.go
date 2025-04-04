package pgbulk_test

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"github.com/kaichao/gopkg/pgbulk"
)

func TestBulkUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlTemplate := "UPDATE users SET name = ?, age = ? WHERE id IN (?)"
	ids := []int{1, 2}
	data := [][]interface{}{
		{"Alice", 30},
		{"Bob", 25},
	}

	// 实际执行生成的 SQL（已知准确）
	// 	actualSQL := `
	// UPDATE users SET
	//   name = CASE WHEN id = $1 THEN $2 WHEN id = $3 THEN $4 ELSE name END,
	//   age = CASE WHEN id = $5 THEN $6 WHEN id = $7 THEN $8 ELSE age END
	// WHERE id IN ($9, $10)`

	// 去掉换行 + 正则转义，防止匹配失败
	cleanSQL := regexp.QuoteMeta(`
UPDATE users SET name = CASE WHEN id = $1 THEN $2 WHEN id = $3 THEN $4 ELSE name END, age = CASE WHEN id = $5 THEN $6 WHEN id = $7 THEN $8 ELSE age END WHERE id IN ($9, $10)`)

	mock.ExpectExec(cleanSQL).
		WithArgs(1, "Alice", 2, "Bob", 1, 30, 2, 25, 1, 2).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err = pgbulk.BulkUpdate(db, sqlTemplate, ids, data)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
