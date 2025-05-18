package asyncbatch

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// Batch ...
type Batch[T any] struct {
	q             chan T
	batchSize     int
	emptyWait     time.Duration
	partialWait   time.Duration
	processFunc   func([]T) error
	errorHandler  func([]T, error)
	logSampleRate int
}

// New ...
func New[T any](queue chan T, batchSize int, emptyWait, partialWait time.Duration, process func([]T) error, errorHandler func([]T, error), logSampleRate int) *Batch[T] {
	if errorHandler == nil {
		errorHandler = func(items []T, err error) {
			log.Printf("error processing %d items: %v", len(items), err)
		}
	}
	if logSampleRate <= 0 {
		logSampleRate = 1
	}
	return &Batch[T]{
		q:             queue,
		batchSize:     batchSize,
		emptyWait:     emptyWait,
		partialWait:   partialWait,
		processFunc:   process,
		errorHandler:  errorHandler,
		logSampleRate: logSampleRate,
	}
}

func (b *Batch[T]) Run() {
	var batch []T
	timer := time.NewTimer(b.emptyWait)
	workerID := fmt.Sprintf("worker-%d", rand.Int())
	if b.logSampleRate == 1 {
		log.Printf("%s started", workerID)
	}
	logCounter := 0

	defer func() {
		if len(batch) > 0 {
			logCounter++
			if logCounter%b.logSampleRate == 0 {
				log.Printf("%s flushing %d items", workerID, len(batch))
			}
			if err := b.processFunc(batch); err != nil {
				b.errorHandler(batch, err)
			}
		}
		if b.logSampleRate == 1 {
			log.Printf("%s stopped", workerID)
		}
		timer.Stop()
	}()

	for {
		select {
		case item, ok := <-b.q:
			if !ok {
				// 队列关闭，处理剩余项
				if len(batch) > 0 {
					logCounter++
					if logCounter%b.logSampleRate == 0 {
						log.Printf("%s flushing %d items", workerID, len(batch))
					}
					if err := b.processFunc(batch); err != nil {
						b.errorHandler(batch, err)
					}
				}
				return
			}

			// 添加到批次
			batch = append(batch, item)

			// 关键修改点：每次添加新项时，如果未满批次，重置定时器为 partialWait
			if len(batch) < b.batchSize {
				// 停止旧定时器并重置为 partialWait
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(b.partialWait)
			} else {
				// 批次已满，立即处理
				logCounter++
				if logCounter%b.logSampleRate == 0 {
					log.Printf("%s flushing %d items", workerID, len(batch))
				}
				if err := b.processFunc(batch); err != nil {
					b.errorHandler(batch, err)
				}
				batch = batch[:0]
				// 处理完满批次后，重置定时器为 emptyWait
				timer.Reset(b.emptyWait)
			}

		case <-timer.C:
			// 定时器触发，处理当前批次（可能为空）
			if len(batch) > 0 {
				logCounter++
				if logCounter%b.logSampleRate == 0 {
					log.Printf("%s flushing %d items", workerID, len(batch))
				}
				if err := b.processFunc(batch); err != nil {
					b.errorHandler(batch, err)
				}
				batch = batch[:0]
			}
			// 重置定时器为 emptyWait 等待新批次
			timer.Reset(b.emptyWait)
		}
	}
}
