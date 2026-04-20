// Package asyncbatch provides a generic batch processor for asynchronous task processing.
//
// Design Goals:
// 1. Generic support for tasks of any type with type safety
// 2. Flexible configuration via With... functions
// 3. Dynamic waiting based on task count and previous batch size
// 4. Parallel processing with multiple workers
// 5. Graceful shutdown with remaining task processing
//
// Core Features:
// - Generic Type Support: Handles tasks of any type with compile-time type safety
// - Flexible Configuration: Configures batch processing parameters via With... functions
// - Dynamic Batching: Adjusts batch triggering based on task count and ratios
// - Parallel Processing: Supports multiple workers for concurrent batch processing
// - Graceful Shutdown: Safely processes remaining tasks before exiting
// - Configurable Parameters: Max batch size, wait times, ratios, and worker count
//
// Usage Examples:
//
//	import (
//	    "fmt"
//	    "time"
//	    "github.com/kaichao/gopkg/asyncbatch"
//	)
//
//	func main() {
//	    // Define a worker function to process batches
//	    worker := func(batch []int) {
//	        fmt.Printf("Processing batch: %v\n", batch)
//	    }
//
//	    // Create a new BatchProcessor with custom options
//	    bp, err := asyncbatch.NewBatchProcessor[int](
//	        worker,
//	        asyncbatch.WithMaxSize(1000),
//	        asyncbatch.WithUpperRatio(0.5),
//	        asyncbatch.WithLowerRatio(0.1),
//	        asyncbatch.WithFixedWait(5*time.Millisecond),
//	        asyncbatch.WithUnderfilledWait(20*time.Millisecond),
//	        asyncbatch.WithNumWorkers(4),
//	    )
//	    if err != nil {
//	        fmt.Printf("Error creating BatchProcessor: %v\n", err)
//	        return
//	    }
//	    defer bp.Shutdown()
//
//	    // Add tasks to the processor
//	    for i := 0; i < 1000; i++ {
//	        if err := bp.Add(i); err != nil {
//	            fmt.Printf("Error adding task: %v\n", err)
//	            return
//	        }
//	    }
//
//	    // Wait for processing to complete
//	    time.Sleep(time.Second)
//	}
//
// Configuration Parameters:
//
//   - maxSize (Maximum Batch Size): Maximum tasks per batch. Larger values increase throughput
//     but may increase memory usage and latency. Smaller values are suitable for low-latency scenarios.
//
//   - lowerRatio (Lower Ratio): Minimum ratio for underfilled batch processing. Triggers a batch
//     when batch size reaches maxSize * lowerRatio and wait time exceeds underfilledWait.
//
//   - fixedWait (Fixed Wait Time): Initial wait time for task collection. Controls frequency of
//     batch collection. Shorter times for high-throughput, longer times to save CPU.
//
//   - underfilledWait (Underfilled Wait Time): Wait time for underfilled batches (below maxSize
//     but above lowerRatio). Balances latency and throughput.
//
//   - numWorkers (Number of Workers): Parallel workers for batch processing. More workers increase
//     throughput but add CPU/memory overhead.
//
// Note: The upperRatio parameter is currently unused in the implementation and setting it
// has no effect. It is recommended to ignore this parameter.
//
// Available Functions:
//
//	NewBatchProcessor[T any](worker func([]T), opts ...Option) (*BatchProcessor[T], error)
//	(bp *BatchProcessor[T]) Add(task T) error
//	(bp *BatchProcessor[T]) Shutdown()
//	(bp *BatchProcessor[T]) TasksCap() int
//
// Getter Methods:
//
//	(bp *BatchProcessor[T]) MaxSize() int
//	(bp *BatchProcessor[T]) UpperRatio() float64
//	(bp *BatchProcessor[T]) LowerRatio() float64
//	(bp *BatchProcessor[T]) FixedWait() time.Duration
//	(bp *BatchProcessor[T]) UnderfilledWait() time.Duration
//	(bp *BatchProcessor[T]) NumWorkers() int
//	(bp *BatchProcessor[T]) Worker() func([]T)
//
// Available Options:
//
//	WithMaxSize(size int) Option                 // Set maximum batch size
//	WithUpperRatio(ratio float64) Option         // Set upper ratio (currently unused)
//	WithLowerRatio(ratio float64) Option         // Set lower ratio for underfilled batches
//	WithFixedWait(duration time.Duration) Option // Set fixed wait time
//	WithUnderfilledWait(duration time.Duration) Option // Set underfilled wait time
//	WithNumWorkers(numWorkers int) Option        // Set number of parallel workers (1-8)
//
// Parameter Defaults and Recommended Ranges:
//
//	Parameter           Default  Recommended Range
//	MaxSize             1000     100-10000 (High throughput: 1000-10000, Low latency: 100-500)
//	UpperRatio          0.5      0.5-0.8 (Currently unused)
//	LowerRatio          0.1      0.05-0.3 (Low latency: 0.05-0.1, High throughput: 0.2-0.3)
//	FixedWait           5 ms     1ms-50ms (High throughput: 1ms-10ms, Low frequency: 20ms-50ms)
//	UnderfilledWait     20 ms    10ms-100ms (Low latency: 10ms-20ms, High throughput: 50ms-100ms)
//	NumWorkers          1        1-8 (Single-core: 1-2, Multi-core high concurrency: 4-8)
//
// Dependencies:
//
// - Go 1.18 or higher (generic type support required)
package asyncbatch
