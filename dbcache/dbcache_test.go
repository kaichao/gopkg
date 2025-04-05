package dbcache_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/kaichao/gopkg/dbcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func TestDBCache_Integration(t *testing.T) {
	// Setup test DB
	db, err := sql.Open("postgres", "user=postgres password=secret dbname=postgres sslmode=disable")
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`CREATE TEMP TABLE users (id INT PRIMARY KEY, name TEXT)`)
	require.NoError(t, err)

	// Insert test data
	_, err = db.Exec(`INSERT INTO users VALUES (1, 'Alice'), (2, 'Bob')`)
	require.NoError(t, err)

	// Initialize cache
	cache := dbcache.New[string](
		db,
		"SELECT name FROM users WHERE id = $1",
		time.Minute, 2*time.Minute, nil,
	)

	// Test cache miss
	name, err := cache.Get(1)
	require.NoError(t, err)
	assert.Equal(t, "Alice", name)

	// Test cache hit
	name, err = cache.Get(1)
	require.NoError(t, err)
	assert.Equal(t, "Alice", name)
}
