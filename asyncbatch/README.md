# AsyncBatch

AsyncBatch is a Go package that provides a generic asynchronous batch processing queue. It is designed for scenarios requiring efficient handling of large volumes of data, reducing processing overhead through batch execution.

## Features
- Accepts data asynchronously and processes it in batches.
- Invokes a user-defined function when a batch reaches a specified size or a timeout occurs.
- Thread-safe data submission and processing.
- Graceful shutdown mechanism to ensure all data is processed before exit.

## Use Cases
- Log collection and bulk database insertion.
- Batch API request optimization to reduce call frequency.
- Asynchronous message processing, such as queue consumption.

## Installation
```sh
 go get github.com/kaichao/gopkg/asyncbatch
```

## Usage Example
```go
package main

import (
	"fmt"
	"time"
	"github.com/kaichao/gopkg/asyncbatch"
)

func main() {
	processor := asyncbatch.New(5, 10, 2*time.Second, func(batch []string) {
		fmt.Println("Processing batch:", batch)
	})

	for i := 1; i <= 20; i++ {
		processor.Submit(fmt.Sprintf("Record-%d", i))
	}

	time.Sleep(5 * time.Second)
	processor.Stop()
}
```

## API

### `New`
Creates a new asynchronous batch processing queue.
```go
func New[T any](batchSize int, queueSize int, flushInterval time.Duration, processFunc func([]T)) *AsyncBatch[T]
```

### `Submit`
Submits a single element to the asynchronous queue.
```go
func (ab *AsyncBatch[T]) Submit(item T)
```

### `Stop`
Stops the asynchronous batch processing queue, ensuring all data is processed before exit.
```go
func (ab *AsyncBatch[T]) Stop()
```

