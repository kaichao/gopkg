# AsyncBatch

[![Go Reference](https://pkg.go.dev/badge/github.com/kaichao/gopkg/asyncbatch.svg)](https://pkg.go.dev/github.com/kaichao/gopkg/asyncbatch)

`asyncbatch` provides a generic batch processor for asynchronous task processing with dynamic flow control and parallel execution.

## Features

- **Generic Support**: Type-safe processing for any task type
- **Flexible Configuration**: Configure parameters via `With...` functions
- **Dynamic Batching**: Adjusts batch triggering based on task count and timing
- **Parallel Processing**: Multiple workers for concurrent batch processing
- **Graceful Shutdown**: Safely processes remaining tasks before exiting

## Installation

```bash
go get github.com/kaichao/gopkg/asyncbatch
```

## Quick Start

```go
package main

import (
	"fmt"
	"time"
	"github.com/kaichao/gopkg/asyncbatch"
)

func main() {
	bp, _ := asyncbatch.NewBatchProcessor[int](
		func(batch []int) {
			fmt.Printf("Processing batch: %v\n", batch)
		},
		asyncbatch.WithMaxSize(100),
		asyncbatch.WithNumWorkers(2),
	)
	defer bp.Shutdown()

	for i := 0; i < 500; i++ {
		bp.Add(i)
	}
	time.Sleep(time.Second)
}
```

## Documentation

For complete documentation including configuration parameters, available options, and usage examples, see:

- [Package Documentation](https://pkg.go.dev/github.com/kaichao/gopkg/asyncbatch)
- [doc.go](./doc.go) - Detailed API reference with examples
- [examples/basic/main.go](./examples/basic/main.go) - Basic usage example
- [examples/advanced/main.go](./examples/advanced/main.go) - Advanced scenarios

## Configuration

Key configuration parameters:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `maxSize` | 1000 | Maximum tasks per batch |
| `lowerRatio` | 0.1 | Minimum ratio for underfilled batches |
| `fixedWait` | 5ms | Wait time for initial task checks |
| `underfilledWait` | 20ms | Wait time for underfilled batches |
| `numWorkers` | 1 | Number of parallel workers (1-8) |

**Note**: The `upperRatio` parameter is currently unused and setting it has no effect.

## Examples

See the [examples](./examples/) directory for complete working examples.

## License

MIT License