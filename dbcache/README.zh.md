# DBCache

[English](README.md) | 中文

Go 通用数据库缓存层，支持基于 SQL 模板的数据加载与自动缓存填充。

## 功能特性

- **SQL 模板化**: 支持 $1, $2 等参数化查询占位符
- **自动缓存**: 透明化的缓存穿透处理
- **类型安全**: 通过泛型支持任意数据类型
- **缓存控制**: 可配置的过期时间与清理策略

## 典型场景

- 高频访问的数据库记录（用户/商品信息）
- 复杂查询结果缓存
- 限流API响应缓存
- 配置信息存储优化

## 安装
```sh
go get github.com/kaichao/gopkg/dbcache
```

## 快速开始
```go
// Initialize cache for user emails
emailCache := dbcache.New[string](
    db, // *sql.DB connection
    "SELECT email FROM users WHERE id = $1",
    5*time.Minute, // Cache expiration
    10*time.Minute, // Cleanup interval
    nil, // Use default loader
)

// Get user email (cached)
email, _ := emailCache.Get(123)
```

## API参考

### New
```go
func New[T any](
    db *sql.DB,
    sqlTemplate string,
    defaultExp, cleanupInterval time.Duration,
    loader func(...any) (T, error),
) *DBCache[T]
```

### Get

```go
// Custom hash generator
loader := func(params ...any) (string, error) {
    id := params[0].(int)
    return fmt.Sprintf("hash-%d", id), nil
}

hashCache := dbcache.NewDBCache[string](
    nil, // db not required
    "", 
    0, 0,
    loader,
)
```
