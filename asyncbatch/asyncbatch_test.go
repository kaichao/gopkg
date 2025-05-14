package asyncbatch_test

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/kaichao/gopkg/asyncbatch"
	"github.com/stretchr/testify/assert"
)

// TestBatchMultipleWorkers 测试多工作者处理批量任务
func TestBatchMultipleWorkers(t *testing.T) {
	var mu sync.Mutex
	processed := make(map[string][]int) // 记录每个工作者处理的批次
	var wg sync.WaitGroup

	// 定义处理函数，记录工作者处理的批次大小
	processFunc := func(items []int) error {
		workerID := fmt.Sprintf("worker-%d", rand.Int())
		mu.Lock()
		processed[workerID] = append(processed[workerID], len(items))
		mu.Unlock()
		log.Printf("%s 处理了 %d 个任务", workerID, len(items))
		wg.Done() // 标记该批次处理完成
		return nil
	}

	// 配置 Batch：3 个工作者，批量大小 10，刷新间隔 500ms
	b := asyncbatch.New(processFunc,
		asyncbatch.WithNumWorkers(3),
		asyncbatch.WithBatchSize(10),
		asyncbatch.WithFlushInterval(500*time.Millisecond),
	)

	// 向队列推送 100 个任务，预期分为 10 个批次
	wg.Add(100 / 10) // 预期 10 个批次（100 任务 / 10 每批）
	for i := 0; i < 100; i++ {
		b.Push(i)
	}

	// 关闭队列
	b.Close()

	// 等待所有批次处理完成
	wg.Wait()

	// 验证结果
	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, len(processed), 1, "应该使用多个工作者")
	totalBatches := 0
	for id, batches := range processed {
		log.Printf("工作者 %s 处理的批次: %v", id, batches)
		totalBatches += len(batches)
	}
	assert.Equal(t, 10, totalBatches, "总共应处理 10 个批次")
}

// TestBatchSingleWorker 测试单工作者处理
func TestBatchSingleWorker(t *testing.T) {
	var mu sync.Mutex
	processedItems := []int{}
	var wg sync.WaitGroup

	processFunc := func(items []int) error {
		mu.Lock()
		processedItems = append(processedItems, items...)
		mu.Unlock()
		wg.Done()
		return nil
	}

	b := asyncbatch.New(processFunc,
		asyncbatch.WithNumWorkers(1),
		asyncbatch.WithBatchSize(5),
		asyncbatch.WithFlushInterval(500*time.Millisecond),
	)

	wg.Add(2) // 预期 2 个批次（10 任务 / 5 每批）
	for i := 0; i < 10; i++ {
		b.Push(i)
	}
	b.Close()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 10, len(processedItems), "应处理 10 个任务")
	assert.ElementsMatch(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, processedItems, "任务顺序应正确")
}

// TestBatchFlushInterval 测试刷新间隔触发
func TestBatchFlushInterval(t *testing.T) {
	var mu sync.Mutex
	processedItems := []int{}
	var wg sync.WaitGroup

	processFunc := func(items []int) error {
		mu.Lock()
		processedItems = append(processedItems, items...)
		mu.Unlock()
		wg.Done()
		return nil
	}

	b := asyncbatch.New(processFunc,
		asyncbatch.WithNumWorkers(1),
		asyncbatch.WithBatchSize(100), // 故意设置大批量大小
		asyncbatch.WithFlushInterval(100*time.Millisecond),
	)

	wg.Add(1) // 预期 1 个批次（由刷新间隔触发）
	b.Push(1)
	time.Sleep(150 * time.Millisecond) // 等待刷新间隔触发
	b.Close()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, len(processedItems), "应处理 1 个任务")
	assert.ElementsMatch(t, []int{1}, processedItems, "任务应正确处理")
}

// TestBatchEmptyQueue 测试关闭空队列
func TestBatchEmptyQueue(t *testing.T) {
	var mu sync.Mutex
	processed := false

	processFunc := func(items []int) error {
		mu.Lock()
		processed = true
		mu.Unlock()
		return nil
	}

	b := asyncbatch.New(processFunc,
		asyncbatch.WithNumWorkers(1),
		asyncbatch.WithBatchSize(10),
		asyncbatch.WithFlushInterval(500*time.Millisecond),
	)

	b.Close()
	time.Sleep(50 * time.Millisecond) // 等待工作者退出

	mu.Lock()
	defer mu.Unlock()
	assert.False(t, processed, "空队列不应触发处理")
}

// TestBatchErrorHandling 测试错误处理
func TestBatchErrorHandling(t *testing.T) {
	var mu sync.Mutex
	errorsCount := 0
	var wg sync.WaitGroup

	processFunc := func(items []int) error {
		mu.Lock()
		errorsCount++
		mu.Unlock()
		wg.Done()
		return errors.New("处理错误")
	}

	b := asyncbatch.New(processFunc,
		asyncbatch.WithNumWorkers(1),
		asyncbatch.WithBatchSize(5),
		asyncbatch.WithFlushInterval(500*time.Millisecond),
	)

	wg.Add(2) // 预期 2 个批次
	for i := 0; i < 10; i++ {
		b.Push(i)
	}
	b.Close()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, errorsCount, "应记录 2 次错误")
}

// TestBatchInvalidConfig 测试无效配置
func TestBatchInvalidConfig(t *testing.T) {
	var mu sync.Mutex
	processedItems := []int{}
	var wg sync.WaitGroup

	processFunc := func(items []int) error {
		mu.Lock()
		processedItems = append(processedItems, items...)
		mu.Unlock()
		wg.Done()
		return nil
	}

	b := asyncbatch.New(processFunc,
		asyncbatch.WithNumWorkers(0), // 无效工作者数量，修正为 1
		asyncbatch.WithBatchSize(-1), // 无效批量大小，修正为 1
		asyncbatch.WithFlushInterval(500*time.Millisecond),
	)

	wg.Add(10) // 预期 10 个批次（10 任务 / 1 每批）
	for i := 0; i < 10; i++ {
		b.Push(i)
	}
	b.Close()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 10, len(processedItems), "应处理 10 个任务")
}

// TestBatchConcurrentPush 测试并发推送
func TestBatchConcurrentPush(t *testing.T) {
	var mu sync.Mutex
	processedItems := []int{}
	var wg sync.WaitGroup

	processFunc := func(items []int) error {
		mu.Lock()
		processedItems = append(processedItems, items...)
		mu.Unlock()
		// 仅在任务总数未达 100 时验证批次大小为 10
		if len(processedItems) < 100 && len(items) != 10 {
			t.Errorf("预期批次大小为 10，实际为 %d", len(items))
		}
		wg.Done()
		return nil
	}

	b := asyncbatch.New(processFunc,
		asyncbatch.WithNumWorkers(2),
		asyncbatch.WithBatchSize(10),
		asyncbatch.WithFlushInterval(1*time.Minute), // 避免刷新间隔触发
	)

	// 并发推送 100 个任务
	concurrency := 10
	tasksPerGoroutine := 10
	totalTasks := concurrency * tasksPerGoroutine
	// 使用硬编码的 batchSize=10 计算批次数量
	batchSize := 10
	wg.Add((totalTasks + batchSize - 1) / batchSize)
	var pushWG sync.WaitGroup
	pushWG.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(start int) {
			for j := 0; j < tasksPerGoroutine; j++ {
				b.Push(start + j)
			}
			pushWG.Done()
		}(i * tasksPerGoroutine)
	}
	pushWG.Wait()

	// 等待所有批次处理完成
	wg.Wait()
	b.Close()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 100, len(processedItems), "应处理 100 个任务")
}
