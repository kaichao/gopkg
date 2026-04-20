// Advanced usage example for asyncbatch package
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kaichao/gopkg/asyncbatch"
)

func main() {
	// Example 1: Processing with metrics collection
	fmt.Println("=== Example 1: Batch Processing with Metrics ===")
	var totalProcessed atomic.Int32
	var batchCount atomic.Int32
	var mu sync.Mutex
	lastBatchTime := make(map[int]time.Time)

	bp1, err := asyncbatch.NewBatchProcessor[int](
		func(batch []int) {
			count := len(batch)
			totalProcessed.Add(int32(count))
			batchNum := batchCount.Add(1)

			mu.Lock()
			lastBatchTime[int(batchNum)] = time.Now()
			mu.Unlock()

			fmt.Printf("Batch %d: processed %d items (total: %d)\n",
				batchNum, count, totalProcessed.Load())
		},
		asyncbatch.WithMaxSize(100),
		asyncbatch.WithFixedWait(10*time.Millisecond),
		asyncbatch.WithUnderfilledWait(50*time.Millisecond),
		asyncbatch.WithNumWorkers(2),
	)
	if err != nil {
		fmt.Printf("Error creating BatchProcessor: %v\n", err)
		return
	}
	defer bp1.Shutdown()

	// Add tasks
	for i := 0; i < 500; i++ {
		bp1.Add(i)
	}

	time.Sleep(1 * time.Second)
	fmt.Printf("Total processed: %d in %d batches\n",
		totalProcessed.Load(), batchCount.Load())

	// Example 2: String processing with different configurations
	fmt.Println("\n=== Example 2: String Processing with Dynamic Configuration ===")

	bp2, err := asyncbatch.NewBatchProcessor[string](
		func(batch []string) {
			fmt.Printf("Processing strings: %v\n", batch[:min(3, len(batch))])
			if len(batch) > 3 {
				fmt.Printf("... and %d more\n", len(batch)-3)
			}
		},
		asyncbatch.WithMaxSize(5),      // Small batches
		asyncbatch.WithLowerRatio(0.8), // Wait for 80% full
		asyncbatch.WithFixedWait(20*time.Millisecond),
		asyncbatch.WithUnderfilledWait(100*time.Millisecond),
		asyncbatch.WithNumWorkers(1),
	)
	if err != nil {
		fmt.Printf("Error creating BatchProcessor: %v\n", err)
		return
	}
	defer bp2.Shutdown()

	// Simulate incoming data
	words := []string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"}
	for _, word := range words {
		bp2.Add(word)
		time.Sleep(15 * time.Millisecond) // Simulate variable arrival rate
	}

	time.Sleep(500 * time.Millisecond)

	// Example 3: Graceful shutdown with pending tasks
	fmt.Println("\n=== Example 3: Graceful Shutdown Example ===")

	var shutdownProcessed atomic.Int32
	bp3, err := asyncbatch.NewBatchProcessor[int](
		func(batch []int) {
			shutdownProcessed.Add(int32(len(batch)))
			fmt.Printf("Shutdown batch processed %d items (total: %d)\n",
				len(batch), shutdownProcessed.Load())
		},
		asyncbatch.WithMaxSize(10),
		asyncbatch.WithFixedWait(5*time.Millisecond),
	)
	if err != nil {
		fmt.Printf("Error creating BatchProcessor: %v\n", err)
		return
	}

	// Add tasks quickly
	for i := 0; i < 25; i++ {
		bp3.Add(i)
	}

	// Immediate shutdown while tasks are being added
	go func() {
		time.Sleep(10 * time.Millisecond)
		fmt.Println("Initiating graceful shutdown...")
		bp3.Shutdown()
		fmt.Println("Shutdown complete")
	}()

	// Try to add more tasks after shutdown
	time.Sleep(30 * time.Millisecond)
	if err := bp3.Add(999); err != nil {
		fmt.Printf("Expected error after shutdown: %v\n", err)
	}

	time.Sleep(200 * time.Millisecond)
	fmt.Printf("Final processed count after shutdown: %d\n", shutdownProcessed.Load())

	fmt.Println("\n=== All examples completed successfully ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
