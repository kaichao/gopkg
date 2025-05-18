package asyncbatch

import (
	"errors"
	"time"
)

// Package-level error constants
var (
	ErrBatchProcessorClosed = errors.New("batch processor is closed")
	ErrTaskChannelFull      = errors.New("task channel is full")
)

// Option defines the configuration function type
type Option[T any] func(*BatchProcessor[T])

// BatchProcessor is a generic batch processor
type BatchProcessor[T any] struct {
	maxSize     int           // Maximum batch size
	maxWait     time.Duration // Maximum wait time for full batch
	emptyWait   time.Duration // Wait time for empty batch
	partialWait time.Duration // Wait time for partial batch
	worker      func([]T)     // Batch processing function
	tasks       chan T        // Task channel
	closed      bool          // Whether the processor is closed
	stop        chan struct{} // Stop signal
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor[T any](opts ...Option[T]) *BatchProcessor[T] {
	bp := &BatchProcessor[T]{
		maxSize:     100,         // Default max batch size
		maxWait:     time.Second, // Default full batch wait time
		emptyWait:   time.Second, // Default empty batch wait time
		partialWait: time.Second, // Default partial batch wait time
		tasks:       make(chan T, 100),
		stop:        make(chan struct{}),
	}
	for _, opt := range opts {
		opt(bp)
	}
	return bp
}

// WithMaxSize sets the maximum batch size
func WithMaxSize[T any](size int) Option[T] {
	return func(bp *BatchProcessor[T]) {
		if size > 0 {
			bp.maxSize = size
		}
	}
}

// WithMaxWait sets the maximum wait time for a full batch
func WithMaxWait[T any](duration time.Duration) Option[T] {
	return func(bp *BatchProcessor[T]) {
		if duration > 0 {
			bp.maxWait = duration
		}
	}
}

// WithEmptyWait sets the wait time for an empty batch
func WithEmptyWait[T any](duration time.Duration) Option[T] {
	return func(bp *BatchProcessor[T]) {
		if duration > 0 {
			bp.emptyWait = duration
		}
	}
}

// WithPartialWait sets the wait time for a partial batch
func WithPartialWait[T any](duration time.Duration) Option[T] {
	return func(bp *BatchProcessor[T]) {
		if duration > 0 {
			bp.partialWait = duration
		}
	}
}

// WithWorker sets the batch processing function
func WithWorker[T any](worker func([]T)) Option[T] {
	return func(bp *BatchProcessor[T]) {
		bp.worker = worker
	}
}

// MaxSize returns the maximum batch size
func (bp *BatchProcessor[T]) MaxSize() int {
	return bp.maxSize
}

// MaxWait returns the maximum wait time for a full batch
func (bp *BatchProcessor[T]) MaxWait() time.Duration {
	return bp.maxWait
}

// EmptyWait returns the wait time for an empty batch
func (bp *BatchProcessor[T]) EmptyWait() time.Duration {
	return bp.emptyWait
}

// PartialWait returns the wait time for a partial batch
func (bp *BatchProcessor[T]) PartialWait() time.Duration {
	return bp.partialWait
}

// Worker returns the batch processing function
func (bp *BatchProcessor[T]) Worker() func([]T) {
	return bp.worker
}

// Add adds a task to the batch processor
func (bp *BatchProcessor[T]) Add(task T) error {
	if bp.closed {
		return ErrBatchProcessorClosed
	}
	select {
	case bp.tasks <- task:
		return nil
	default:
		return ErrTaskChannelFull
	}
}

// Shutdown closes the batch processor
func (bp *BatchProcessor[T]) Shutdown() {
	if !bp.closed {
		bp.closed = true
		close(bp.stop)
	}
}

// Run starts the batch processor
func (bp *BatchProcessor[T]) Run() {
	if bp.worker == nil {
		return
	}

	batch := make([]T, 0, bp.maxSize)
	var timer *time.Timer
	var waitTime time.Duration

	for {
		// Select wait time based on batch state
		if len(batch) == 0 {
			waitTime = bp.emptyWait
		} else if len(batch) < bp.maxSize {
			waitTime = bp.partialWait
		} else {
			waitTime = bp.maxWait
		}

		if timer == nil {
			timer = time.NewTimer(waitTime)
			defer timer.Stop()
		} else {
			timer.Reset(waitTime)
		}

		select {
		case task, ok := <-bp.tasks:
			if !ok {
				if len(batch) > 0 {
					bp.worker(batch)
				}
				return
			}
			batch = append(batch, task)
			if len(batch) >= bp.maxSize {
				// Process full batch immediately
				bp.worker(batch)
				batch = make([]T, 0, bp.maxSize)
				timer.Reset(bp.emptyWait)
			}
		case <-timer.C:
			// Timeout: process non-empty batch
			if len(batch) > 0 {
				bp.worker(batch)
				batch = make([]T, 0, bp.maxSize)
			}
			timer.Reset(bp.emptyWait)
		case <-bp.stop:
			// Drain tasks channel before exiting
			close(bp.tasks)
			for task := range bp.tasks {
				batch = append(batch, task)
				if len(batch) >= bp.maxSize {
					bp.worker(batch)
					batch = make([]T, 0, bp.maxSize)
				}
			}
			if len(batch) > 0 {
				bp.worker(batch)
			}
			return
		}
	}
}
