package asyncbatch

import (
	"math"
	"sync"
	"time"
	"unsafe"

	"github.com/kaichao/gopkg/errors"
)

// BatchProcessor is a generic batch processor for asynchronous task processing.
type BatchProcessor[T any] struct {
	maxSize         int
	upperRatio      float64
	lowerRatio      float64
	fixedWait       time.Duration
	underfilledWait time.Duration
	numWorkers      int
	worker          func([]T)
	tasks           chan T
	closed          bool
	stop            chan struct{}
	wg              sync.WaitGroup
	closeOnce       sync.Once
}

// Option configures BatchProcessor.
type Option func(*BatchProcessor[any])

// WithMaxSize sets the maximum batch size.
func WithMaxSize(size int) Option {
	return func(bp *BatchProcessor[any]) {
		if size > 0 {
			bp.maxSize = size
		}
	}
}

// WithUpperRatio sets the upper ratio for continuous processing.
func WithUpperRatio(ratio float64) Option {
	return func(bp *BatchProcessor[any]) {
		if ratio > 0 && ratio <= 1 {
			bp.upperRatio = ratio
		}
	}
}

// WithLowerRatio sets the lower ratio for underfilled waiting.
func WithLowerRatio(ratio float64) Option {
	return func(bp *BatchProcessor[any]) {
		if ratio > 0 && ratio <= 1 {
			bp.lowerRatio = ratio
		}
	}
}

// WithFixedWait sets the fixed wait time for initial task check.
func WithFixedWait(duration time.Duration) Option {
	return func(bp *BatchProcessor[any]) {
		if duration > 0 {
			bp.fixedWait = duration
		}
	}
}

// WithUnderfilledWait sets the wait time for underfilled batches.
func WithUnderfilledWait(duration time.Duration) Option {
	return func(bp *BatchProcessor[any]) {
		if duration > 0 {
			bp.underfilledWait = duration
		}
	}
}

// WithNumWorkers sets the number of parallel workers (max 8).
func WithNumWorkers(n int) Option {
	return func(bp *BatchProcessor[any]) {
		if n > 0 {
			// Ensure assignment to correct field
			bp.numWorkers = n
		}
	}
}

// NewBatchProcessor creates and starts a batch processor with the given options.
func NewBatchProcessor[T any](
	worker func([]T),
	opts ...Option,
) (*BatchProcessor[T], error) {
	bp := &BatchProcessor[T]{
		worker:          worker,
		maxSize:         1000,
		upperRatio:      0.5,
		lowerRatio:      0.1,
		fixedWait:       5 * time.Millisecond,
		underfilledWait: 20 * time.Millisecond,
		numWorkers:      1,
		stop:            make(chan struct{}),
	}

	// Type conversion to adapt Option
	anyBP := (*BatchProcessor[any])(unsafe.Pointer(bp))
	for _, opt := range opts {
		opt(anyBP)
	}

	// Keep original validation logic
	if bp.worker == nil {
		return nil, errors.E("worker function is required")
	}
	if bp.numWorkers < 1 || bp.numWorkers > 8 {
		return nil, errors.E("numWorkers must be between 1 and 8")
	}
	if bp.upperRatio <= 0 || bp.upperRatio > 1 {
		return nil, errors.E("upperRatio must be between 0 and 1")
	}
	if bp.lowerRatio <= 0 || bp.lowerRatio > 1 {
		return nil, errors.E("lowerRatio must be between 0 and 1")
	}
	if bp.upperRatio < bp.lowerRatio {
		return nil, errors.E("upperRatio must be greater than or equal to lowerRatio")
	}
	if bp.fixedWait >= bp.underfilledWait {
		return nil, errors.E("fixedWait must be less than underfilledWait")
	}

	bufferSize := bp.maxSize * bp.numWorkers * 2
	if bufferSize < bp.maxSize*2 {
		bufferSize = bp.maxSize * 2
	}
	bp.tasks = make(chan T, bufferSize)

	bp.wg.Add(bp.numWorkers)
	for i := 0; i < bp.numWorkers; i++ {
		go func() {
			defer bp.wg.Done()
			bp.run()
		}()
	}

	return bp, nil
}

// Add adds a task to the processor.
func (bp *BatchProcessor[T]) Add(task T) error {
	if bp.closed {
		return errors.E("batch processor is closed")
	}
	select {
	case bp.tasks <- task:
		return nil
	default:
		return errors.E("task channel is full")
	}
}

// Shutdown stops the processor and processes remaining tasks.
func (bp *BatchProcessor[T]) Shutdown() {
	bp.closeOnce.Do(func() {
		bp.closed = true
		close(bp.stop)
		bp.wg.Wait() // Wait for all workers to stop

		// Process remaining tasks separately, not involving WaitGroup
		close(bp.tasks)
		remaining := make([]T, 0, len(bp.tasks))
		for task := range bp.tasks {
			remaining = append(remaining, task)
		}
		if len(remaining) > 0 {
			bp.worker(remaining)
		}
	})
}

func (bp *BatchProcessor[T]) TasksCap() int {
	return cap(bp.tasks)
}

// run is the internal worker loop for processing batches.
func (bp *BatchProcessor[T]) run() {
	batch := make([]T, 0, bp.maxSize)
	var timer *time.Timer
	lowerThreshold := int(math.Max(1, math.Floor(float64(bp.maxSize)*bp.lowerRatio)))

	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		// First check for stop signal
		select {
		case <-bp.stop:
			bp.flushBatch(batch)
			return
		default:
		}

		// Check thresholds first
		if shouldFlush := len(batch) >= bp.maxSize || len(batch) >= int(float64(bp.maxSize)*bp.upperRatio); shouldFlush {
			bp.flushBatch(batch)
			batch, timer = bp.resetBatchAndTimer(batch, timer)
			continue
		}

		// Initialize timer
		timer = bp.initTimer(timer)

		select {
		case task, ok := <-bp.tasks:
			if !ok {
				bp.flushBatch(batch)
				return
			}
			batch = append(batch, task)

		case <-timer.C:
			batch, timer = bp.handleTimerExpired(batch, timer, lowerThreshold)
		}
	}
}

// Helper function 1: Process batch submission
func (bp *BatchProcessor[T]) flushBatch(batch []T) {
	if len(batch) > 0 {
		bp.worker(batch)
	}
}

// Helper function 2: Reset batch and timer
func (bp *BatchProcessor[T]) resetBatchAndTimer(batch []T, timer *time.Timer) ([]T, *time.Timer) {
	if timer != nil {
		timer.Stop()
	}
	return make([]T, 0, bp.maxSize), nil
}

// Helper function 3: Initialize timer
func (bp *BatchProcessor[T]) initTimer(timer *time.Timer) *time.Timer {
	if timer == nil {
		return time.NewTimer(bp.fixedWait)
	}
	timer.Reset(bp.fixedWait)
	return timer
}

// Helper function 4: Handle timer expiration
func (bp *BatchProcessor[T]) handleTimerExpired(batch []T, timer *time.Timer, lowerThreshold int) ([]T, *time.Timer) {
	if len(batch) >= lowerThreshold {
		bp.flushBatch(batch)
		return bp.resetBatchAndTimer(batch, timer)
	}

	// Start secondary waiting
	timer.Reset(bp.underfilledWait)
	select {
	case task, ok := <-bp.tasks:
		if !ok {
			bp.flushBatch(batch)
			return batch, timer
		}
		return append(batch, task), timer

	case <-timer.C:
		bp.flushBatch(batch)
		return bp.resetBatchAndTimer(batch, timer)

	case <-bp.stop:
		bp.flushBatch(batch)
		return batch, timer
	}
}

// Getter methods
func (bp *BatchProcessor[T]) MaxSize() int                   { return bp.maxSize }
func (bp *BatchProcessor[T]) UpperRatio() float64            { return bp.upperRatio }
func (bp *BatchProcessor[T]) LowerRatio() float64            { return bp.lowerRatio }
func (bp *BatchProcessor[T]) FixedWait() time.Duration       { return bp.fixedWait }
func (bp *BatchProcessor[T]) UnderfilledWait() time.Duration { return bp.underfilledWait }
func (bp *BatchProcessor[T]) NumWorkers() int                { return bp.numWorkers }
func (bp *BatchProcessor[T]) Worker() func([]T)              { return bp.worker }
