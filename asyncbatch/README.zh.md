# AsyncBatch

[English](README.md) | 中文

`asyncbatch` 是一个支持异步批处理的 Go 包，基于泛型实现，允许用户配置批处理大小、等待时间及处理逻辑，适用于高吞吐或低延迟的场景。

## 功能特性

- **泛型支持**：支持任意类型任务，类型安全。
- **灵活配置**：通过 `With...` 函数配置批处理参数。
- **动态等待**：根据批次状态（空、部分、满）调整等待时间。
- **优雅关闭**：支持处理剩余任务后安全退出。

## 典型场景

- 数据库批量操作（插入/更新）
- API 请求批量聚合
- 日志/事件处理管道
- 队列消费者实现

## 安装

```sh
go get github.com/kaichao/gopkg/asyncbatch
```

## 快速开始

```go
package main

import (
    "fmt"
    "time"
    "github.com/kaichao/gopkg/asyncbatch"
)

func main() {
    // 创建批处理器
    bp := asyncbatch.NewBatchProcessor[string](
        asyncbatch.WithMaxSize[string](3),
        asyncbatch.WithMaxWait[string](time.Second),
        asyncbatch.WithEmptyWait[string](200*time.Millisecond),
        asyncbatch.WithPartialWait[string](100*time.Millisecond),
        asyncbatch.WithWorker[string](func(batch []string) {
            fmt.Printf("处理批次: %v\n", batch)
        }),
    )

    // 启动批处理
    go bp.Run()

    // 添加任务
    bp.Add("task1")
    bp.Add("task2")
    bp.Add("task3")

    // 等待处理
    time.Sleep(200 * time.Millisecond)

    // 关闭
    bp.Shutdown()
}
```

## API
### 类型
- `BatchProcessor[T any]`：批处理器，基于泛型，支持任意类型任务。
- `Option[T any]`：配置函数类型，用于设置参数。

### 函数
- **`NewBatchProcessor[T any](opts ...Option[T]) *BatchProcessor[T]`**  
  创建批处理器，接受配置选项。默认：最大批次 100，等待时间 1 秒。
- **`WithMaxSize[T any](size int) Option[T]`**  
  设置最大批次大小（>0 有效）。
- **`WithMaxWait[T any](duration time.Duration) Option[T]`**  
  设置满批次最大等待时间（>0 有效）。满批次通常立即处理，`maxWait` 限制连续满批次的最大延迟。
- **`WithEmptyWait[T any](duration time.Duration) Option[T]`**  
  设置空批次等待时间（>0 有效）。空批次无任务时等待新任务。
- **`WithPartialWait[T any](duration time.Duration) Option[T]`**  
  设置部分批次等待时间（>0 有效）。未满批次等待更多任务或超时处理。
- **`WithWorker[T any](worker func([]T)) Option[T]`**  
  设置批处理函数，处理一批任务。
- **`(*BatchProcessor[T]) Add(task T) error`**  
  添加任务到批处理器。返回错误若批处理器已关闭或通道满。
- **`(*BatchProcessor[T]) Run()`**  
  启动批处理器，异步处理任务。
- **`(*BatchProcessor[T]) Shutdown()`**  
  关闭批处理器，处理剩余任务后退出。
- **`(*BatchProcessor[T]) MaxSize() int`**  
  返回最大批次大小。
- **`(*BatchProcessor[T]) MaxWait() time.Duration`**  
  返回满批次最大等待时间。
- **`(*BatchProcessor[T]) EmptyWait() time.Duration`**  
  返回空批次等待时间。
- **`(*BatchProcessor[T]) PartialWait() time.Duration`**  
  返回部分批次等待时间。
- **`(*BatchProcessor[T]) Worker() func([]T)`**  
  返回批处理函数。

## 等待时间原理
`asyncbatch` 通过三种等待时间动态控制批处理触发，优化吞吐量和延迟：

- **`maxWait`**：满批次（任务数 ≥ `maxSize`）的最大等待时间。  
  - **作用**：满批次立即处理，`maxWait` 限制高吞吐场景下连续满批次的最大延迟，避免任务堆积。  
  - **示例**：`maxSize=3`, `maxWait=1s`，3 个任务立即处理，若任务持续到来，每隔最多 1 秒处理一次。
- **`emptyWait`**：空批次（无任务）的等待时间。  
  - **作用**：空闲时等待新任务，不触发处理，降低资源消耗。  
  - **示例**：`emptyWait=200ms`，无任务时每 200ms 检查一次新任务。
- **`partialWait`**：部分批次（任务数 < `maxSize`）的等待时间。  
  - **作用**：未满批次等待更多任务或超时处理，适合低吞吐场景。  
  - **示例**：`partialWait=100ms`，2 个任务等待 100ms 后处理，若新任务加入则重置等待。

**原理**：`Run()` 根据批次状态（空、部分、满）选择等待时间，动态调整定时器。满批次优先处理，部分批次平衡延迟与吞吐，空批次节省资源。关闭时确保剩余任务处理完成。
