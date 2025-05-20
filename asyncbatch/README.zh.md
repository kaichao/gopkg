# AsyncBatch

[English](README.md) | 中文

[![Go Reference](https://pkg.go.dev/badge/github.com/kaichao/gopkg/asyncbatch.svg)](https://pkg.go.dev/github.com/kaichao/gopkg/asyncbatch)

`asyncbatch` 是一个支持异步批处理的 Go 包，基于泛型实现，支持动态流量控制，适用于高吞吐或低延迟的场景。

## 特性
- **泛型支持**：支持任意类型任务，类型安全。
- **灵活配置**：通过 `With...` 函数配置批处理参数。
- **动态等待**：根据任务数量和上次批次大小调整触发时机。
- **并行处理**：支持多个 Worker 并发处理批次。
- **优雅关闭**：处理剩余任务后安全退出。

## 安装
```bash
go get github.com/kaichao/gopkg/asyncbatch
```

## 使用方法

`asyncbatch` 包提供了一个 `BatchProcessor`，用于按批次处理类型为 `T` 的任务。你可以通过选项配置处理器，例如最大批次大小、上下限比例、固定和未满批次的等待时间，以及工作者数量。

### 示例

以下是如何使用 `BatchProcessor` 处理整数批次的示例：

```go
package main

import (
	"fmt"
	"time"

	"github.com/kaichao/gopkg/asyncbatch"
)

func main() {
	// 定义一个处理整数批次的工作函数
	worker := func(batch []int) {
		fmt.Printf("处理批次：%v\n", batch)
	}

	// 使用自定义选项创建新的 BatchProcessor
	bp, err := asyncbatch.NewBatchProcessor[int](
		worker,
		asyncbatch.WithMaxSize(1000),
		asyncbatch.WithUpperRatio(0.5),
		asyncbatch.WithLowerRatio(0.1),
		asyncbatch.WithFixedWait(5*time.Millisecond),
		asyncbatch.WithUnderfilledWait(20*time.Millisecond),
		asyncbatch.WithNumWorkers(4),
	)
	if err != nil {
		fmt.Printf("创建 BatchProcessor 失败：%v\n", err)
		return
	}
	defer bp.Shutdown()

	// 向处理器添加任务
	for i := 0; i < 1000; i++ {
		if err := bp.Add(i); err != nil {
			fmt.Printf("添加任务失败：%v\n", err)
			return
		}
	}

	// 等待处理完成
	time.Sleep(time.Second)
}
```

### 选项

- `WithMaxSize(size int)`：设置最大批次大小。
- `WithUpperRatio(ratio float64)`：设置连续处理的比率上限（当前实现未使用）。
- `WithLowerRatio(ratio float64)`：设置未满批次处理的比率下限。
- `WithFixedWait(duration time.Duration)`：设置初始任务检查的固定等待时间。
- `WithUnderfilledWait(duration time.Duration)`：设置未满批次的等待时间。
- `WithNumWorkers(numWorkers int)`：设置并行工作者的数量（1 到 8）。

## 参数设置原理

以下是 `BatchProcessor` 的配置参数作用、默认值和推荐值范围的说明，帮助用户根据实际场景优化配置。

- **maxSize（最大批次大小）**  
  决定每个批次最多包含的任务数量。较大的值提高吞吐量，但可能增加内存使用和处理延迟；较小的值适合低延迟场景。  

- **upperRatio（比率上限）**  
  理论上控制连续处理的比率上限，当 `len(batch)/maxSize >= upperRatio` 时触发批次处理。当前实现未使用此参数（代码中相关逻辑被注释）。  

- **lowerRatio（比率下限）**  
  控制未满批次处理的最小比率，当批次大小达到 `maxSize * lowerRatio` 且等待时间超过 `underfilledWait` 时触发处理。较低值减少延迟，但可能降低吞吐量。  

- **fixedWait（固定等待时间）**  
  初次检查任务的等待时间，控制批次收集的频率。较短的等待时间适合高吞吐场景，较长时间减少频繁处理，节省 CPU 资源。  

- **underfilledWait（未满批次等待时间）**  
  未满批次（未达 `maxSize` 但超过 `lowerRatio`）的等待时间，平衡延迟和吞吐量。较短值减少延迟，较长值增加批次填充率。  

- **numWorkers（工作者数量）**  
  并行工作者的数量，决定批次处理的并发度。较多工作者提高吞吐量，但增加 CPU 和内存开销，需与系统资源匹配。  

### 参数默认值与推荐范围

| 参数名                | 默认值         | 推荐值范围                     |
|-----------------------|----------------|-------------------------------|
| 最大批次大小 (maxSize) | 1000          | 100-10000（高吞吐：1000-10000，低延迟：100-500） |
| 比率上限 (upperRatio) | 0.5           | 0.5-0.8（当前未使用，建议忽略） |
| 比率下限 (lowerRatio) | 0.1           | 0.05-0.3（低延迟：0.05-0.1，高吞吐：0.2-0.3） |
| 固定等待时间 (fixedWait) | 5 毫秒       | 1ms-50ms（高吞吐：1ms-10ms，低频率：20ms-50ms） |
| 未满批次等待时间 (underfilledWait) | 20 毫秒 | 10ms-100ms（低延迟：10ms-20ms，高吞吐：50ms-100ms） |
| 工作者数量 (numWorkers) | 1           | 1-8（单核：1-2，多核高并发：4-8） |

**注意**：`upperRatio` 在当前实现中未使用，设置此参数不会影响行为，建议忽略。

## 注意事项

- `worker` 函数负责处理任务批次，必须在创建 `BatchProcessor` 时提供。
- 处理器根据配置的 `maxSize` 和 `lowerRatio` 自动处理任务批次，时间由 `fixedWait` 和 `underfilledWait` 控制。
- 使用 `Shutdown()` 优雅地停止处理器并处理剩余任务。
- 当前实现未使用 `upperRatio`，设置此参数不会影响行为。

## 依赖

- Go 1.18 或更高版本（由于支持泛型）。
