# AsyncBatch

[English](README.md) | 中文

`asyncbatch` 是一个支持异步批处理的 Go 包，基于泛型实现，允许用户配置批处理大小、等待时间及处理逻辑，适用于高吞吐或低延迟的场景。

## 特性
- **泛型支持**：支持任意类型任务，类型安全。
- **灵活配置**：通过 `With...` 函数配置批处理参数。
- **动态等待**：根据任务数量和上次批次大小调整触发时机。
- **并行处理**：支持多个 Worker 并发处理批次。
- **自动启动**：创建批处理器后自动运行，无需手动启动。
- **优雅关闭**：处理剩余任务后安全退出。

## 安装
```bash
go get github.com/kaichao/gopkg/asyncbatch
```

## 使用示例
```go
package main

import (
    "fmt"
    "time"
    "github.com/kaichao/gopkg/asyncbatch"
)

func main() {
    // 创建批处理器（自动启动）
    bp := asyncbatch.NewBatchProcessor[string](
        asyncbatch.WithMaxSize[string](100),
        asyncbatch.WithUpperRatio[string](0.5),
        asyncbatch.WithLowerRatio[string](0.1),
        asyncbatch.WithFixedWait[string](100*time.Millisecond),
        asyncbatch.WithUnderfilledWait[string](500*time.Millisecond),
        asyncbatch.WithNumWorkers[string](2),
        asyncbatch.WithWorker[string](func(batch []string) {
            fmt.Printf("处理批次: %v\n", batch)
        }),
    )

    // 添加任务
    for i := 0; i < 50; i++ {
        bp.Add(fmt.Sprintf("task%d", i))
    }

    // 等待处理
    time.Sleep(600 * time.Millisecond)

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
  创建并自动启动批处理器，接受配置选项。默认：最大批次 100，上限比例 0.5，下限比例 0.1，固定等待时间 100ms，未满等待时间 500ms，Worker 数量 1。
- **`WithMaxSize[T any](size int) Option[T]`**  
  设置最大批次大小（>0 有效）。
- **`WithUpperRatio[T any](ratio float64) Option[T]`**  
  设置上限比例（>0 有效）。上次批次任务数超过 `maxSize * ratio` 时立即处理。
- **`WithLowerRatio[T any](ratio float64) Option[T]`**  
  设置下限比例（>0 有效）。固定等待后任务数低于 `maxSize * ratio` 时继续等待。
- **`WithFixedWait[T any](duration time.Duration) Option[T]`**  
  设置固定等待时间（>0 有效）。首次检查任务数量的等待时间。
- **`WithUnderfilledWait[T any](duration time.Duration) Option[T]`**  
  设置未满等待时间（>0 有效）。任务数低于下限比例时的额外等待时间。
- **`WithNumWorkers[T any](numWorkers int) Option[T]`**  
  设置并行 Worker（goroutine）数量（>0 且 ≤8 有效）。每个 Worker 独立处理批次，提高吞吐量。默认 1。
- **`WithWorker[T any](worker func([]T)) Option[T]`**  
  设置批处理函数，处理所有批次。
- **`(*BatchProcessor[T]) Add(task T) error`**  
  添加任务到批处理器。返回错误若批处理器已关闭或通道满。
- **`(*BatchProcessor[T]) Shutdown()`**  
  关闭批处理器，处理剩余任务后退出。
- **`(*BatchProcessor[T]) MaxSize() int`**  
  返回最大批次大小。
- **`(*BatchProcessor[T]) UpperRatio() float64`**  
  返回上限比例。
- **`(*BatchProcessor[T]) LowerRatio() float64`**  
  返回下限比例。
- **`(*BatchProcessor[T]) FixedWait() time.Duration`**  
  返回固定等待时间。
- **`(*BatchProcessor[T]) UnderfilledWait() time.Duration`**  
  返回未满等待时间。
- **`(*BatchProcessor[T]) NumWorkers() int`**  
  返回并行 Worker 数量。
- **`(*BatchProcessor[T]) Worker() func([]T)`**  
  返回批处理函数。

## 等待时间原理
`asyncbatch` 通过上限比例、下限比例、固定等待时间和未满等待时间动态控制批处理触发，优化吞吐量和延迟。所有批次由单一 `worker` 函数处理，创建批处理器后自动启动，多个 Worker（goroutine）可并行运行，共享任务通道。

- **上限比例（`upperRatio`）**：上次批次任务数超过 `maxSize * upperRatio` 时立即处理。  
  - **作用**：高负载时连续处理，避免任务积压，提高吞吐量。  
  - **示例**：`maxSize=100`, `upperRatio=0.5`，上次处理 50 个任务，立即触发新批次。
- **下限比例（`lowerRatio`）**：固定等待时间后，任务数低于 `maxSize * lowerRatio` 时继续等待。  
  - **作用**：低负载时等待更多任务，优化批次效率。  
  - **示例**：`maxSize=100`, `lowerRatio=0.1`，任务数 < 10，继续等待。
- **固定等待时间（`fixedWait`）**：首次检查任务数量的等待时间。  
  - **作用**：快速响应任务到达，降低初始延迟。  
  - **示例**：`fixedWait=100ms`，100ms 后检查任务数。
- **未满等待时间（`underfilledWait`）**：任务数低于下限比例时的额外等待时间。  
  - **作用**：允许凑更多任务，若超时则处理，防止积压。  
  - **示例**：`underfilledWait=500ms`，任务数 5 个，等待 500ms 后处理。

**触发逻辑**：
1. 若上次批次任务数 ≥ `maxSize * upperRatio`，立即处理当前批次（若有任务）。
2. 否则，等待 `fixedWait`：
   - 若任务数 ≥ `maxSize * lowerRatio`，立即处理。
   - 若任务数 < `maxSize * lowerRatio`，等待 `underfilledWait`，超时后处理（若有任务）。
3. 空批次（0 任务）继续等待 `fixedWait`，避免空转。

**并行处理**：
- 多个 Worker（通过 `WithNumWorkers()` 配置，最大 8）并行运行上述逻辑，从共享任务通道获取任务。
- 每个 Worker 独立维护批次和定时器，调用同一 `worker` 函数。

**典型配置与调整**：
- **典型值**：
  - `upperRatio`：0.5，适合高负载连续处理。
  - `lowerRatio`：0.1，适合低负载等待。
  - `fixedWait`：100ms，快速检查任务。
  - `underfilledWait`：500ms，适中等待低任务批次。
  - `numWorkers`：1（默认），建议 2-4，最大 8。
- **约束**：
  - `fixedWait` < `underfilledWait`，确保两阶段等待有效。
  - `upperRatio` 和 `lowerRatio` > 0，无上限（建议 `upperRatio` ≥ `lowerRatio`）。
- **场景化配置**：
  - **高吞吐场景**（如数据库批量写入）：`upperRatio=0.5`, `lowerRatio=0.1`, `fixedWait=100ms`, `underfilledWait=500ms`, `numWorkers=4`。
  - **低延迟场景**（如实时日志处理）：`upperRatio=0.3`, `lowerRatio=0.05`, `fixedWait=50ms`, `underfilledWait=200ms`, `numWorkers=2`。
  - **通用场景**（如队列消费者）：`upperRatio=0.5`, `lowerRatio=0.1`, `fixedWait=100ms`, `underfilledWait=500ms`, `numWorkers=2`。
- **调整建议**：
  - 高负载：增大 `upperRatio`（如 0.7），增加 `numWorkers`（如 4）。
  - 低延迟：减小 `fixedWait`（如 50ms）和 `underfilledWait`（如 200ms）。
  - 调试比例时，建议 `upperRatio` ≥ `lowerRatio`，避免逻辑冲突。

**原理**：`NewBatchProcessor` 自动启动多个 Worker，每个 Worker 根据任务数量和上次批次大小选择触发时机，动态调整定时器。关闭时通过 `Shutdown()` 确保剩余任务处理完成。
