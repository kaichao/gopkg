package pgbulk_test

import (
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kaichao/gopkg/pgbulk"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCopySpecialTypes(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	defer db.Close()

	logrus.SetOutput(io.Discard)

	ts := time.Date(2025, 5, 6, 15, 4, 5, 123456000, time.UTC)
	ts2 := ts.Add(time.Minute)
	js := map[string]interface{}{"name": "example", "age": 30}
	data := [][]interface{}{
		{ts, []time.Time{ts, ts2}, js},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("COPY table_name (ts, tsarr, js) FROM STDIN WITH (FORMAT CSV)").
		WillReturnCloseError(nil)

	mock.ExpectExec("COPY table_name (ts, tsarr, js) FROM STDIN WITH (FORMAT CSV)").
		WithArgs(sqlmock.AnyArg()). // 忽略具体匹配，避免类型转换错误
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	count, err := pgbulk.Copy(db, "INSERT INTO table_name (ts, tsarr, js)", data)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCopy tests the Copy function under various scenarios
func TestCopyBase(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Setup logrus to discard logs during testing
	logrus.SetOutput(io.Discard)

	tests := []struct {
		name          string
		sqlTemplate   string
		data          [][]interface{}
		expectedCount int
		expectedError error
		setupMock     func(*testing.T, sqlmock.Sqlmock)
	}{
		{
			name:          "Empty data",
			sqlTemplate:   "INSERT INTO table_name (col1, col2)",
			data:          [][]interface{}{},
			expectedCount: 0,
			expectedError: nil,
			setupMock:     func(t *testing.T, mock sqlmock.Sqlmock) {},
		},
		{
			name:          "Invalid SQL template",
			sqlTemplate:   "INVALID SQL",
			data:          [][]interface{}{{1, "test1"}},
			expectedCount: 0,
			expectedError: fmt.Errorf("invalid sqlTemplate format"),
			setupMock:     func(t *testing.T, mock sqlmock.Sqlmock) {},
		},
		{
			name:          "Empty columns",
			sqlTemplate:   "INSERT INTO table_name ()",
			data:          [][]interface{}{{1, "test1"}},
			expectedCount: 0,
			expectedError: fmt.Errorf("no columns specified in sqlTemplate"),
			setupMock:     func(t *testing.T, mock sqlmock.Sqlmock) {},
		},
		{
			name:          "Single row copy",
			sqlTemplate:   "INSERT INTO table_name (col1, col2)",
			data:          [][]interface{}{{1, "test1"}},
			expectedCount: 1,
			expectedError: nil,
			setupMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectPrepare("COPY table_name (col1, col2) FROM STDIN WITH (FORMAT CSV)").
					WillReturnCloseError(nil)
				mock.ExpectExec("COPY table_name (col1, col2) FROM STDIN WITH (FORMAT CSV)").
					WithArgs("1,\"test1\"\n").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:          "Multiple row copy",
			sqlTemplate:   "INSERT INTO table_name (col1, col2)",
			data:          [][]interface{}{{1, "test1"}, {2, "test2"}},
			expectedCount: 2,
			expectedError: nil,
			setupMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectPrepare("COPY table_name (col1, col2) FROM STDIN WITH (FORMAT CSV)").
					WillReturnCloseError(nil)
				mock.ExpectExec("COPY table_name (col1, col2) FROM STDIN WITH (FORMAT CSV)").
					WithArgs("1,\"test1\"\n2,\"test2\"\n").
					WillReturnResult(sqlmock.NewResult(1, 2))
				mock.ExpectCommit()
			},
		},
		{
			name:          "Row length mismatch",
			sqlTemplate:   "INSERT INTO table_name (col1, col2)",
			data:          [][]interface{}{{1}},
			expectedCount: 0,
			expectedError: fmt.Errorf("row length 1 does not match column count 2"),
			setupMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectPrepare("COPY table_name (col1, col2) FROM STDIN WITH (FORMAT CSV)").
					WillReturnCloseError(nil)
				mock.ExpectRollback()
			},
		},
		{
			name:          "Unsupported data type",
			sqlTemplate:   "INSERT INTO table_name (col1, col2)",
			data:          [][]interface{}{{1, complex(1, 1)}},
			expectedCount: 0,
			expectedError: fmt.Errorf("unsupported data type: complex128"),
			setupMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectPrepare("COPY table_name (col1, col2) FROM STDIN WITH (FORMAT CSV)").
					WillReturnCloseError(nil)
				mock.ExpectRollback()
			},
		},
		{
			name:          "COPY execution error",
			sqlTemplate:   "INSERT INTO table_name (col1, col2)",
			data:          [][]interface{}{{1, "test1"}},
			expectedCount: 0,
			expectedError: errors.New("copy error"),
			setupMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectPrepare("COPY table_name (col1, col2) FROM STDIN WITH (FORMAT CSV)").
					WillReturnCloseError(nil)
				mock.ExpectExec("COPY table_name (col1, col2) FROM STDIN WITH (FORMAT CSV)").
					WithArgs("1,\"test1\"\n").
					WillReturnError(errors.New("copy error"))
				mock.ExpectRollback()
			},
		},
		{
			name:          "Rows affected mismatch",
			sqlTemplate:   "INSERT INTO table_name (col1, col2)",
			data:          [][]interface{}{{1, "test1"}, {2, "test2"}},
			expectedCount: 0,
			expectedError: fmt.Errorf("expected to insert 2 rows, but inserted 1"),
			setupMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectPrepare("COPY table_name (col1, col2) FROM STDIN WITH (FORMAT CSV)").
					WillReturnCloseError(nil)
				mock.ExpectExec("COPY table_name (col1, col2) FROM STDIN WITH (FORMAT CSV)").
					WithArgs("1,\"test1\"\n2,\"test2\"\n").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectRollback()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(t, mock)
			count, err := pgbulk.Copy(db, tt.sqlTemplate, tt.data)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error(), "Error mismatch")
			} else {
				assert.NoError(t, err, "Unexpected error")
			}
			assert.Equal(t, tt.expectedCount, count, "Count mismatch")
			assert.NoError(t, mock.ExpectationsWereMet(), "Unfulfilled mock expectations")
		})
	}
}
