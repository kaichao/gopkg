package asyncbatch

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"math/rand"
)

// Batch ...
type Batch[T any] struct {
	processFunc   func([]T) error
	q             chan T
	numWorkers    int
	batchSize     int
	flushInterval time.Duration
}

// Option 配置选项
type Option func(*Batch[any])

// WithNumWorkers 设置工作者数量
func WithNumWorkers(numWorkers int) Option {
	return func(b *Batch[any]) {
		if numWorkers < 1 {
			numWorkers = 1
		}
		b.numWorkers = numWorkers
	}
}

// WithBatchSize 设置批量大小
func WithBatchSize(size int) Option {
	return func(b *Batch[any]) {
		if size < 1 {
			size = 1
		}
		b.batchSize = size
	}
}

// WithFlushInterval 设置刷新间隔
func WithFlushInterval(d time.Duration) Option {
	return func(b *Batch[any]) {
		b.flushInterval = d
	}
}

const defaultChanSize = 1000

// New 创建 Batch 实例
func New[T any](processFunc func([]T) error, opts ...Option) *Batch[T] {
	b := &Batch[T]{
		processFunc: processFunc,
		q:           make(chan T, defaultChanSize),
		numWorkers:  1,
	}
	// 将 Batch[T] 转换为 Batch[any] 以应用 Option
	bAny := (*Batch[any])(unsafe.Pointer(b))
	for _, opt := range opts {
		opt(bAny)
	}
	for i := 0; i < b.numWorkers; i++ {
		go b.run()
	}
	return b
}

// run 工作者主循环
func (b *Batch[T]) run() {
	var batch []T
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()
	workerID := fmt.Sprintf("worker-%d", rand.Int())
	log.Printf("%s started", workerID)

	for {
		select {
		case item, ok := <-b.q:
			if !ok {
				if len(batch) > 0 {
					log.Printf("%s processing %d items", workerID, len(batch))
					_ = b.processFunc(batch)
				}
				log.Printf("%s stopped", workerID)
				return
			}
			batch = append(batch, item)
			if len(batch) >= b.batchSize {
				log.Printf("%s processing %d items", workerID, len(batch))
				_ = b.processFunc(batch)
				batch = nil
			}
		case <-ticker.C:
			if len(batch) > 0 {
				log.Printf("%s flushing %d items", workerID, len(batch))
				_ = b.processFunc(batch)
				batch = nil
			}
		}
	}
}

// Push 向队列添加任务
func (b *Batch[T]) Push(item T) {
	b.q <- item
}

// Close 关闭队列
func (b *Batch[T]) Close() {
	close(b.q)
}
