// Basic usage example for asyncbatch package
package main

import (
	"fmt"
	"time"

	"github.com/kaichao/gopkg/asyncbatch"
)

func main() {
	// Define a worker function to process a batch of integers
	worker := func(batch []int) {
		fmt.Printf("Processing batch: %v\n", batch)
	}

	// Create a new BatchProcessor with custom options
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
		fmt.Printf("Error creating BatchProcessor: %v\n", err)
		return
	}
	defer bp.Shutdown()

	// Add tasks to the processor
	for i := 0; i < 1000; i++ {
		if err := bp.Add(i); err != nil {
			fmt.Printf("Error adding task: %v\n", err)
			return
		}
	}

	// Wait for processing to complete
	time.Sleep(time.Second)
}
