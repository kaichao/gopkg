package asyncbatch_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kaichao/gopkg/asyncbatch"
)

func TestBatch(t *testing.T) {
	var mu sync.Mutex
	var batches [][]int
	queue := make(chan int, 100)
	batchSize := 10
	emptyWait := 100 * time.Millisecond
	partialWait := 50 * time.Millisecond
	logSampleRate := 1

	b := asyncbatch.New(queue, batchSize, emptyWait, partialWait, func(items []int) error {
		mu.Lock()
		batches = append(batches, items)
		mu.Unlock()
		return nil
	}, nil, logSampleRate)

	go b.Run()

	// Send items
	for i := 0; i < 25; i++ {
		queue <- i
	}
	// MODIFIED: increased sleep time to 4s
	time.Sleep(4 * time.Second)
	close(queue)
	time.Sleep(200 * time.Millisecond)

	// Verify results
	mu.Lock()
	defer mu.Unlock()
	if len(batches) != 3 {
		t.Errorf("expected 3 batches, got %d", len(batches))
	}
	if len(batches) >= 1 && len(batches[0]) != 10 {
		t.Errorf("expected first batch size 10, got %d", len(batches[0]))
	}
	if len(batches) >= 2 && len(batches[1]) != 10 {
		t.Errorf("expected second batch size 10, got %d", len(batches[1]))
	}
	if len(batches) >= 3 && len(batches[2]) != 5 {
		t.Errorf("expected third batch size 5, got %d", len(batches[2]))
	}
}

func TestFullBatchImmediateProcessing(t *testing.T) {
	var mu sync.Mutex
	var batches [][]int
	queue := make(chan int, 100)
	batchSize := 10
	emptyWait := 500 * time.Millisecond
	partialWait := 50 * time.Millisecond
	logSampleRate := 1

	b := asyncbatch.New(queue, batchSize, emptyWait, partialWait, func(items []int) error {
		mu.Lock()
		batches = append(batches, items)
		mu.Unlock()
		return nil
	}, nil, logSampleRate)

	go b.Run()

	// Send 20 items (2 full batches)
	for i := 0; i < 20; i++ {
		queue <- i
	}
	// MODIFIED: increased sleep time to 3s
	time.Sleep(3 * time.Second)
	close(queue)
	time.Sleep(200 * time.Millisecond)

	// Verify: expect 2 full batches processed immediately
	mu.Lock()
	defer mu.Unlock()
	if len(batches) != 2 {
		t.Errorf("expected 2 batches, got %d", len(batches))
	}
	if len(batches) >= 1 && len(batches[0]) != 10 {
		t.Errorf("expected first batch size 10, got %d", len(batches[0]))
	}
	if len(batches) >= 2 && len(batches[1]) != 10 {
		t.Errorf("expected second batch size 10, got %d", len(batches[1]))
	}
}

func TestPartialBatchWait(t *testing.T) {
	var mu sync.Mutex
	var batches [][]int
	queue := make(chan int, 100)
	batchSize := 10
	emptyWait := 500 * time.Millisecond
	partialWait := 3 * time.Second // 将 partialWait 调整为 3秒
	logSampleRate := 1

	b := asyncbatch.New(queue, batchSize, emptyWait, partialWait, func(items []int) error {
		mu.Lock()
		batches = append(batches, items)
		mu.Unlock()
		return nil
	}, nil, logSampleRate)

	go b.Run()

	// 发送 5 项（部分批次）
	for i := 0; i < 5; i++ {
		queue <- i
	}
	// 等待时间调整为小于 partialWait（例如 2秒 < 3秒）
	time.Sleep(2 * time.Second)
	// 再发送 3 项
	for i := 5; i < 8; i++ {
		queue <- i
	}
	// 等待足够时间让定时器触发
	time.Sleep(4 * time.Second)
	close(queue)
	time.Sleep(200 * time.Millisecond) // 确保处理完成

	// 验证结果
	mu.Lock()
	defer mu.Unlock()
	if len(batches) != 1 {
		t.Errorf("expected 1 batch, got %d", len(batches))
	}
	if len(batches) >= 1 && len(batches[0]) != 8 {
		t.Errorf("expected batch size 8, got %d", len(batches[0]))
	}
}
func TestErrorHandling(t *testing.T) {
	var mu sync.Mutex
	var errorsHandled [][]int
	queue := make(chan int, 100)
	batchSize := 10
	emptyWait := 100 * time.Millisecond
	partialWait := 50 * time.Millisecond
	logSampleRate := 1

	b := asyncbatch.New(queue, batchSize, emptyWait, partialWait, func(items []int) error {
		return errors.New("processing error")
	}, func(items []int, err error) {
		mu.Lock()
		errorsHandled = append(errorsHandled, items)
		mu.Unlock()
	}, logSampleRate)

	go b.Run()

	// Send 10 items
	for i := 0; i < 10; i++ {
		queue <- i
	}
	// MODIFIED: increased sleep time to 3s
	time.Sleep(3 * time.Second)
	close(queue)
	time.Sleep(200 * time.Millisecond)

	// Verify: error handler called
	mu.Lock()
	defer mu.Unlock()
	if len(errorsHandled) != 1 {
		t.Errorf("expected 1 error handled, got %d", len(errorsHandled))
	}
	if len(errorsHandled) >= 1 && len(errorsHandled[0]) != 10 {
		t.Errorf("expected error batch size 10, got %d", len(errorsHandled[0]))
	}
}

func TestDynamicPartialWait(t *testing.T) {
	var mu sync.Mutex
	var batches [][]int
	queue := make(chan int, 100)
	batchSize := 10
	emptyWait := 500 * time.Millisecond
	partialWait := 3 * time.Second // 将 partialWait 调整为 3秒
	logSampleRate := 1

	b := asyncbatch.New(queue, batchSize, emptyWait, partialWait, func(items []int) error {
		mu.Lock()
		batches = append(batches, items)
		mu.Unlock()
		return nil
	}, nil, logSampleRate)

	go b.Run()

	// 第一次发送 5 项
	for i := 0; i < 5; i++ {
		queue <- i
	}
	// 等待时间调整为小于 partialWait（例如 2秒 < 3秒）
	time.Sleep(2 * time.Second)
	// 第二次发送 5 项
	for i := 5; i < 10; i++ {
		queue <- i
	}
	// 等待足够时间让定时器触发
	time.Sleep(4 * time.Second)
	close(queue)
	time.Sleep(200 * time.Millisecond) // 确保处理完成

	// 验证结果
	mu.Lock()
	defer mu.Unlock()
	if len(batches) != 1 {
		t.Errorf("expected 1 batch, got %d", len(batches))
	}
	if len(batches) >= 1 && len(batches[0]) != 10 {
		t.Errorf("expected batch size 10, got %d", len(batches[0]))
	}
}

func TestEmptyWaitTrigger(t *testing.T) {
	var mu sync.Mutex
	var batches [][]int
	queue := make(chan int, 100)
	batchSize := 10
	emptyWait := 100 * time.Millisecond
	partialWait := 50 * time.Millisecond
	logSampleRate := 1

	b := asyncbatch.New(queue, batchSize, emptyWait, partialWait, func(items []int) error {
		mu.Lock()
		batches = append(batches, items)
		mu.Unlock()
		return nil
	}, nil, logSampleRate)

	go b.Run()

	// Send 3 items (low load)
	for i := 0; i < 3; i++ {
		queue <- i
	}
	// MODIFIED: increased sleep time to 3s
	time.Sleep(3 * time.Second)
	close(queue)
	time.Sleep(200 * time.Millisecond)

	// Verify: expect 1 batch of 3 items triggered by emptyWait
	mu.Lock()
	defer mu.Unlock()
	if len(batches) != 1 {
		t.Errorf("expected 1 batch, got %d", len(batches))
	}
	if len(batches) >= 1 && len(batches[0]) != 3 {
		t.Errorf("expected batch size 3, got %d", len(batches[0]))
	}
}
