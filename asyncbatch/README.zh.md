# AsyncBatch

[English](README.md) | 中文

Go 通用异步批处理工具，支持可配置的刷新策略

## 功能特性

- **自动批处理**: 达到批次大小或时间间隔时自动处理
- **线程安全**: 支持并发提交和处理
- **优雅关闭**: 确保所有待处理项完成处理
- **泛型支持**: 通过 Go 泛型支持任意数据类型

## 典型场景

- 数据库批量操作(插入/更新)
- API 请求批量聚合
- 日志/事件处理管道
- 队列消费者实现

## 安装
```sh
 go get github.com/kaichao/gopkg/asyncbatch
```

## 快速开始
```go
processor := asyncbatch.New[string](
    5,              // Batch size 
    10,             // Queue size
    2*time.Second, // Flush interval
    func(batch []string) {
        fmt.Println("Processing:", batch)
    },
)

// Submit items
for i := 0; i < 20; i++ {
    processor.Submit(fmt.Sprintf("item-%d", i))
}

// Shutdown cleanly
processor.Stop()
```

## API参考
### `New`
```go
func New[T any](batchSize int, queueSize int, flushInterval time.Duration, processFunc func([]T)) *AsyncBatch[T]
```

### `Submit`
```go
func (ab *AsyncBatch[T]) Submit(item T)
```

### `Stop`
```go
func (ab *AsyncBatch[T]) Stop()
```
