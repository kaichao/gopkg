# AsyncBatch

[English](README.md) | 中文

Go 通用异步批处理工具，支持可配置的刷新策略

## 功能特性

- **自动批处理**: 达到批次大小或时间间隔时自动处理
- **泛型支持**: 通过 Go 泛型支持任意数据类型
- **线程安全**: 支持并发提交和处理
- **优雅关闭**: 确保所有待处理项完成处理
- **多处理后端**: 加速计算的响应速度

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
    // 定义处理函数
    processFunc := func(items []string) error {
        fmt.Printf("正在处理批次：%v", items)
        return nil
    }

    // 创建 Batch，3 个 worker，批量 5 条，2 秒刷新
    batch := asyncbatch.New[string](
        processFunc,
        asyncbatch.WithNumWorkers(3),
        asyncbatch.WithBatchSize(5),
        asyncbatch.WithFlushInterval(2*time.Second),
    )

    // 入队任务
    for i := 0; i < 12; i++ {
        batch.Push(fmt.Sprintf("item-%d", i))
    }

    // 等待一段时间以触发刷新
    time.Sleep(5 * time.Second)

    // 关闭，处理剩余任务并停止 worker
    batch.Close()
```

### API 说明

```go
type Batch[T any] struct {
    // Push(item) 添加一个任务到队列
    Push(item T)

    // Close() 关闭队列，处理剩余任务，并停止所有 worker
    Close()
}
```

#### 构造函数

```go
func New[T any](
    processFunc func([]T) error,
    opts ...Option,
) *Batch[T]
```

- `processFunc`
  处理批量任务的函数，接收 `[]T` 并返回错误。错误会被内部忽略。

- `opts`
  可选配置项（见下文）。

#### 可选配置

- `WithNumWorkers(n int)`
  并发 worker 数量。默认：`1`。最小：`1`。

- `WithBatchSize(size int)`
  每批处理的最大任务数。默认：`1`。最小：`1`。

- `WithFlushInterval(d time.Duration)`
  最长等待时间后刷新非空批次。默认：`0`（禁用定时刷新，仅在满批或 Close 时触发）。

#### 默认参数

- **通道容量**：`100`
- **Worker 数量**：`1`
- **批量大小**：`1`
- **刷新间隔**：`0`（仅满批或 Close 时处理）

## 日志

Worker 会输出启动、批次处理、定时刷新和停止的日志：

```
worker-12345 started
worker-12345 processing 5 items
worker-12345 flushing 2 items
worker-12345 stopped
```

日志使用标准 `log.Printf`。

