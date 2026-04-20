package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/kaichao/gopkg/dbcache"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	// Example 1: Basic database query caching
	db, err := sql.Open("postgres", "user=postgres password=secret dbname=mydb sslmode=disable")
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		return
	}
	defer db.Close()

	// Create a simple cache for user names
	nameCache := dbcache.New[string](
		db,
		"SELECT name FROM users WHERE id = $1",
		5*time.Minute,  // Cache items expire after 5 minutes
		10*time.Minute, // Cleanup interval for expired items
		nil,            // Use default SQL loader
	)

	// First call queries the database
	name1, err := nameCache.Get(123)
	if err != nil {
		fmt.Printf("Error getting name: %v\n", err)
		return
	}
	fmt.Printf("First call (database query): %s\n", name1)

	// Second call uses cache
	name2, err := nameCache.Get(123)
	if err != nil {
		fmt.Printf("Error getting name: %v\n", err)
		return
	}
	fmt.Printf("Second call (cache hit): %s\n", name2)

	// Example 2: Numeric data caching
	ageCache := dbcache.New[int](
		db,
		"SELECT age FROM users WHERE id = $1",
		10*time.Minute,
		30*time.Minute,
		nil,
	)

	age, err := ageCache.Get(456)
	if err != nil {
		fmt.Printf("Error getting age: %v\n", err)
		return
	}
	fmt.Printf("User age: %d\n", age)

	fmt.Println("Basic examples completed successfully!")
}
