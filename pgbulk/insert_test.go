package pgbulk_test

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kaichao/gopkg/pgbulk"
	"github.com/sirupsen/logrus"
)

// TestInsert tests the BulkInsert function under various scenarios
func TestInsert(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
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
		expectedError error
		setupMock     func(*testing.T)
	}{
		{
			name:          "Empty data",
			sqlTemplate:   "INSERT INTO table_name (col1, col2)",
			data:          [][]interface{}{},
			expectedError: fmt.Errorf("data is empty"),
			setupMock:     func(t *testing.T) {},
		},
		{
			name:        "Single batch insert",
			sqlTemplate: "INSERT INTO table_name (col1, col2)",
			data: [][]interface{}{
				{1, "test1"},
				{2, "test2"},
			},
			expectedError: nil,
			setupMock: func(t *testing.T) {
				mock.ExpectExec(`INSERT INTO table_name \(col1, col2\) VALUES \(\$1,\$2\),\(\$3,\$4\)`).
					WithArgs(1, "test1", 2, "test2").
					WillReturnResult(sqlmock.NewResult(1, 2))
			},
		},
		{
			name:        "Multiple batch insert",
			sqlTemplate: "INSERT INTO table_name (col1, col2)",
			data: [][]interface{}{
				{1, "test1"},
				{2, "test2"},
				{3, "test3"},
				{4, "test4"},
			},
			expectedError: nil,
			setupMock: func(t *testing.T) {
				// Expect single batch with all 4 rows since maxBatchSize (65535/2 = 32767) is large
				mock.ExpectExec(`INSERT INTO table_name \(col1, col2\) VALUES \(\$1,\$2\),\(\$3,\$4\),\(\$5,\$6\),\(\$7,\$8\)`).
					WithArgs(1, "test1", 2, "test2", 3, "test3", 4, "test4").
					WillReturnResult(sqlmock.NewResult(1, 4))
			},
		},
		{
			name:        "Database error",
			sqlTemplate: "INSERT INTO table_name (col1, col2)",
			data: [][]interface{}{
				{1, "test1"},
			},
			expectedError: errors.New("database error"),
			setupMock: func(t *testing.T) {
				mock.ExpectExec(`INSERT INTO table_name \(col1, col2\) VALUES \(\$1,\$2\)`).
					WithArgs(1, "test1").
					WillReturnError(errors.New("database error"))
			},
		},
		{
			name:        "Invalid SQL template",
			sqlTemplate: "",
			data: [][]interface{}{
				{1, "test1"},
			},
			expectedError: errors.New("pq: syntax error at or near \"VALUES\""),
			setupMock: func(t *testing.T) {
				mock.ExpectExec(`VALUES \(\$1,\$2\)`).
					WithArgs(1, "test1").
					WillReturnError(errors.New("pq: syntax error at or near \"VALUES\""))
			},
		},
		{
			name:        "Forced small batch insert",
			sqlTemplate: "INSERT INTO table_name (col1, col2)",
			data: [][]interface{}{
				{1, "test1"},
				{2, "test2"},
				{3, "test3"},
			},
			expectedError: nil,
			setupMock: func(t *testing.T) {
				// Expect single batch with all 3 rows since maxBatchSize (65535/2 = 32767) is large
				// Note: To test actual batching, Insert would need a configurable maxBatchSize
				mock.ExpectExec(`INSERT INTO table_name \(col1, col2\) VALUES \(\$1,\$2\),\(\$3,\$4\),\(\$5,\$6\)`).
					WithArgs(1, "test1", 2, "test2", 3, "test3").
					WillReturnResult(sqlmock.NewResult(1, 3))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(t)
			err := pgbulk.Insert(db, tt.sqlTemplate, tt.data)
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("Expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled mock expectations: %v", err)
			}
		})
	}
}
