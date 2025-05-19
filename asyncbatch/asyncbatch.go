package asyncbatch

import (
	"errors"
	"math"
	"sync"
	"time"
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
type Option[T any] func(*BatchProcessor[T])

// WithMaxSize sets the maximum batch size.
func WithMaxSize[T any](size int) Option[T] {
	return func(bp *BatchProcessor[T]) {
		if size > 0 {
			bp.maxSize = size
		}
	}
}

// WithUpperRatio sets the upper ratio for continuous processing.
func WithUpperRatio[T any](ratio float64) Option[T] {
	return func(bp *BatchProcessor[T]) {
		if ratio > 0 && ratio <= 1 {
			bp.upperRatio = ratio
		}
	}
}

// WithLowerRatio sets the lower ratio for underfilled waiting.
func WithLowerRatio[T any](ratio float64) Option[T] {
	return func(bp *BatchProcessor[T]) {
		if ratio > 0 && ratio <= 1 {
			bp.lowerRatio = ratio
		}
	}
}

// WithFixedWait sets the fixed wait time for initial task check.
func WithFixedWait[T any](duration time.Duration) Option[T] {
	return func(bp *BatchProcessor[T]) {
		if duration > 0 {
			bp.fixedWait = duration
		}
	}
}

// WithUnderfilledWait sets the wait time for underfilled batches.
func WithUnderfilledWait[T any](duration time.Duration) Option[T] {
	return func(bp *BatchProcessor[T]) {
		if duration > 0 {
			bp.underfilledWait = duration
		}
	}
}

// WithNumWorkers sets the number of parallel workers (max 8).
func WithNumWorkers[T any](numWorkers int) Option[T] {
	return func(bp *BatchProcessor[T]) {
		bp.numWorkers = numWorkers
	}
}

// WithWorker sets the batch processing function.
func WithWorker[T any](worker func([]T)) Option[T] {
	return func(bp *BatchProcessor[T]) {
		bp.worker = worker
	}
}

// NewBatchProcessor creates and starts a batch processor with the given options.
func NewBatchProcessor[T any](opts ...Option[T]) (*BatchProcessor[T], error) {
	bp := &BatchProcessor[T]{
		maxSize:         100,
		upperRatio:      0.5,
		lowerRatio:      0.1,
		fixedWait:       5 * time.Millisecond,  // Reduced for faster processing
		underfilledWait: 20 * time.Millisecond, // Reduced for faster processing
		numWorkers:      1,
		stop:            make(chan struct{}),
	}

	for _, opt := range opts {
		opt(bp)
	}

	if bp.worker == nil {
		return nil, errors.New("worker function is required")
	}
	if bp.numWorkers < 1 || bp.numWorkers > 8 {
		return nil, errors.New("numWorkers must be between 1 and 8")
	}
	if bp.upperRatio <= 0 || bp.upperRatio > 1 {
		return nil, errors.New("upperRatio must be between 0 and 1")
	}
	if bp.lowerRatio <= 0 || bp.lowerRatio > 1 {
		return nil, errors.New("lowerRatio must be between 0 and 1")
	}
	if bp.upperRatio < bp.lowerRatio {
		return nil, errors.New("upperRatio must be greater than or equal to lowerRatio")
	}
	if bp.fixedWait >= bp.underfilledWait {
		return nil, errors.New("fixedWait must be less than underfilledWait")
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
		return errors.New("batch processor is closed")
	}
	select {
	case bp.tasks <- task:
		return nil
	default:
		return errors.New("task channel is full")
	}
}

// Shutdown stops the processor and processes remaining tasks.
func (bp *BatchProcessor[T]) Shutdown() {
	if !bp.closed {
		bp.closed = true
		close(bp.stop)  // 先关闭 stop 通道
		bp.wg.Wait()    // 等待所有工作者完成
		close(bp.tasks) // 最后关闭任务通道
	}
}

// run is the internal worker loop for processing batches.
func (bp *BatchProcessor[T]) run() {
	batch := make([]T, 0, bp.maxSize)
	var timer *time.Timer
	// upperThreshold := int(math.Max(1, math.Ceil(float64(bp.maxSize)*bp.upperRatio)))
	lowerThreshold := int(math.Max(1, math.Floor(float64(bp.maxSize)*bp.lowerRatio)))

	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		if len(batch) >= bp.maxSize {
			bp.worker(batch)
			batch = make([]T, 0, bp.maxSize)
			if timer != nil {
				timer.Stop()
				timer = nil
			}
			continue
		}

		if timer == nil {
			timer = time.NewTimer(bp.fixedWait)
		} else {
			timer.Reset(bp.fixedWait)
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
		case <-timer.C:
			if len(batch) >= lowerThreshold {
				bp.worker(batch)
				batch = make([]T, 0, bp.maxSize)
				timer.Stop()
				timer = nil
				continue
			}
			timer.Reset(bp.underfilledWait)
			select {
			case task, ok := <-bp.tasks:
				if !ok {
					if len(batch) > 0 {
						bp.worker(batch)
					}
					return
				}
				batch = append(batch, task)
			case <-timer.C:
				if len(batch) > 0 {
					bp.worker(batch)
					batch = make([]T, 0, bp.maxSize)
					timer.Stop()
					timer = nil
				}
			case <-bp.stop:
				if len(batch) > 0 {
					bp.worker(batch)
				}
				return
			}
		case <-bp.stop:
			if len(batch) > 0 {
				bp.worker(batch)
			}
			return
		}
	}
}

// Getter methods
func (bp *BatchProcessor[T]) MaxSize() int {
	return bp.maxSize
}

func (bp *BatchProcessor[T]) UpperRatio() float64 {
	return bp.upperRatio
}

func (bp *BatchProcessor[T]) LowerRatio() float64 {
	return bp.lowerRatio
}

func (bp *BatchProcessor[T]) FixedWait() time.Duration {
	return bp.fixedWait
}

func (bp *BatchProcessor[T]) UnderfilledWait() time.Duration {
	return bp.underfilledWait
}

func (bp *BatchProcessor[T]) NumWorkers() int {
	return bp.numWorkers
}

func (bp *BatchProcessor[T]) Worker() func([]T) {
	return bp.worker
}
