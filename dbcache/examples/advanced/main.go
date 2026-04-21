package main

import (
	"fmt"
	"time"

	"github.com/kaichao/gopkg/dbcache"
	"github.com/kaichao/gopkg/errors"
)

// Product ...
type Product struct {
	ID    int
	Name  string
	Price float64
}

func main() {
	fmt.Println("Advanced DBCache Examples")
	fmt.Println("=========================")

	// Example 1: Custom loader with complex data type
	fmt.Println("\n1. Custom loader with struct type:")
	productLoader := func(params ...any) (Product, error) {
		if len(params) != 1 {
			return Product{}, errors.E("expected exactly one parameter: product ID")
		}
		id, ok := params[0].(int)
		if !ok {
			return Product{}, errors.E("parameter must be an integer")
		}

		// Simulate database query
		// In real application, this would query a database
		if id == 999 {
			return Product{}, errors.E(404, "product not found")
		}

		return Product{
			ID:    id,
			Name:  fmt.Sprintf("Product %d", id),
			Price: float64(id) * 10.0,
		}, nil
	}

	productCache := dbcache.New(
		nil, // No database connection needed for custom loader
		"",  // No SQL template needed
		time.Hour,
		2*time.Hour,
		productLoader,
	)

	// Cache miss - loads via custom loader
	product, err := productCache.Get(123)
	if err != nil {
		fmt.Printf("Error getting product: %v\n", err)
	} else {
		fmt.Printf("Product loaded: %+v\n", product)
	}

	// Cache hit - returns cached value
	product2, err := productCache.Get(123)
	if err != nil {
		fmt.Printf("Error getting product: %v\n", err)
	} else {
		fmt.Printf("Product from cache: %+v\n", product2)
	}

	// Error handling example
	_, err = productCache.Get(999)
	if err != nil {
		fmt.Printf("Expected error for product 999: %v\n", err)
	}

	// Example 2: Multi-parameter caching
	fmt.Println("\n2. Multi-parameter caching:")
	multiParamLoader := func(params ...any) (string, error) {
		if len(params) != 2 {
			return "", errors.E("expected 2 parameters: category and subcategory")
		}
		category, ok1 := params[0].(string)
		subcategory, ok2 := params[1].(string)
		if !ok1 || !ok2 {
			return "", errors.E("parameters must be strings")
		}
		return fmt.Sprintf("Settings for %s/%s", category, subcategory), nil
	}

	settingsCache := dbcache.New(
		nil, // No database needed
		"",
		30*time.Minute,
		time.Hour,
		multiParamLoader,
	)

	// Cache with multiple parameters
	settings, err := settingsCache.Get("user", "preferences")
	if err != nil {
		fmt.Printf("Error getting settings: %v\n", err)
	} else {
		fmt.Printf("Settings: %s\n", settings)
	}

	// Example 3: Dynamic expiration based on data
	fmt.Println("\n3. Dynamic expiration example:")
	configLoader := func(params ...any) (map[string]string, error) {
		configName, ok := params[0].(string)
		if !ok {
			return nil, errors.E("parameter must be a string")
		}

		// Different expiration for different config types
		configs := map[string]map[string]string{
			"app": {
				"version": "1.0.0",
				"env":     "production",
			},
			"db": {
				"host": "localhost",
				"port": "5432",
			},
		}

		config, exists := configs[configName]
		if !exists {
			return nil, errors.E(404, "configuration not found")
		}

		return config, nil
	}

	// Note: For truly dynamic expiration, you'd need to extend DBCache
	// This example uses fixed expiration
	configCache := dbcache.New(
		nil,
		"",
		5*time.Minute, // Shorter expiration for configuration
		10*time.Minute,
		configLoader,
	)

	appConfig, err := configCache.Get("app")
	if err != nil {
		fmt.Printf("Error getting app config: %v\n", err)
	} else {
		fmt.Printf("App config: %+v\n", appConfig)
	}

	dbConfig, err := configCache.Get("db")
	if err != nil {
		fmt.Printf("Error getting db config: %v\n", err)
	} else {
		fmt.Printf("DB config: %+v\n", dbConfig)
	}

	fmt.Println("\nAdvanced examples completed successfully!")
}
