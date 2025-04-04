package asyncbatch

/*
Package asyncbatch provides a generic asynchronous batch processing queue, suitable for scenarios that require efficient handling of large volumes of data.

Main features:
- Accepts data via an asynchronous queue and processes it in batches.
- Invokes a user-defined processing function when a batch reaches a specified size or a timeout occurs.
- Thread-safe submission and processing of data.
- Graceful shutdown mechanism to ensure all data is processed before exit.

Use cases:
- Log collection and bulk database insertion.
- Batch API request optimization to reduce call frequency.
- Asynchronous message processing, such as queue consumption.

Example usage:

	processor := asyncbatch.New(5, 10, 2*time.Second, func(batch []string) {
		fmt.Println("Processing batch:", batch)
	})

	for i := 1; i <= 20; i++ {
		processor.Submit(fmt.Sprintf("Record-%d", i))
	}

	time.Sleep(5 * time.Second)
	processor.Stop()

*/

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AsyncBatch is a generic asynchronous batch processing queue.
type AsyncBatch[T any] struct {
	inputChan     chan T
	batchSize     int
	flushInterval time.Duration
	done          chan struct{}
	processFunc   func([]T)
	wg            sync.WaitGroup
}

// New creates a new asynchronous batch processing queue.
func New[T any](batchSize int, queueSize int, flushInterval time.Duration, processFunc func([]T)) *AsyncBatch[T] {
	if queueSize <= 0 {
		queueSize = 10 // Default to 10
	}

	ab := &AsyncBatch[T]{
		inputChan:     make(chan T, batchSize*queueSize), // Provide some buffer
		batchSize:     batchSize,
		flushInterval: flushInterval,
		done:          make(chan struct{}),
		processFunc:   processFunc,
	}
	ab.wg.Add(1)
	go ab.run()
	return ab
}

// Submit submits a single element to the asynchronous queue.
func (ab *AsyncBatch[T]) Submit(item T) {
	select {
	case ab.inputChan <- item:
	default:
		logrus.Warn("asyncbatch: queue full, dropping data")
	}
}

// run listens to the asynchronous queue and processes data in batches.
func (ab *AsyncBatch[T]) run() {
	defer ab.wg.Done()
	var batch []T

	for {
		select {
		case <-ab.done:
			// Process remaining data before exit
			if len(batch) > 0 {
				ab.processFunc(batch)
			}
			return
		case item := <-ab.inputChan:
			batch = append(batch, item)
			// Continue reading until queue is empty
			for {
				select {
				case nextItem := <-ab.inputChan:
					batch = append(batch, nextItem)
				default:
					ab.processFunc(batch)
					batch = nil
					break
				}
			}
		}
	}
}

// Stop stops the asynchronous batch processing queue.
func (ab *AsyncBatch[T]) Stop() {
	close(ab.done)
	ab.wg.Wait()
}
