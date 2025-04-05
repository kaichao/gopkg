package dbcache

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

// DBCache ...
type DBCache[T any] struct {
	db         *sql.DB
	cache      *cache.Cache
	sql        string
	defaultExp time.Duration
	loadFunc   func(...any) (T, error)
}

// New ...
func New[T any](
	db *sql.DB,
	sqlTemplate string,
	defaultExp, cleanupInterval time.Duration,
	loader func(...any) (T, error),
) *DBCache[T] {
	if loader == nil {
		loader = func(params ...any) (T, error) {
			var result T
			err := db.QueryRow(sqlTemplate, params...).Scan(&result)
			if err == sql.ErrNoRows {
				return result, nil
			}
			return result, err
		}
	}

	return &DBCache[T]{
		db:         db,
		cache:      cache.New(defaultExp, cleanupInterval),
		sql:        sqlTemplate,
		defaultExp: defaultExp,
		loadFunc:   loader,
	}
}

func (c *DBCache[T]) Get(params ...any) (T, error) {
	key := fmt.Sprintf("%v", params)

	if val, found := c.cache.Get(key); found {
		return val.(T), nil
	}

	result, err := c.loadFunc(params...)
	if err != nil {
		return result, err
	}

	c.cache.Set(key, result, c.defaultExp)
	return result, nil
}
