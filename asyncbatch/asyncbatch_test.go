package asyncbatch_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaichao/gopkg/asyncbatch"
)

// Helper function to wait with timeout
func waitWithTimeout(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("Test timed out waiting for WaitGroup")
	}
}

// addTasks adds tasks to BatchProcessor with retry mechanism.
func addTasks[T any](t *testing.T, bp *asyncbatch.BatchProcessor[T], tasks []T, timeout time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for i, task := range tasks {
		for {
			select {
			case <-ctx.Done():
				t.Fatalf("Timeout adding task %d", i)
			default:
				if err := bp.Add(task); err == nil {
					break
				}
				time.Sleep(100 * time.Microsecond)
			}
		}
	}
}

// verifyBatches checks batch sizes and total processed tasks.
func verifyBatches[T any](t *testing.T, batches [][]T, expectedBatches int, expectedTasks int, expectedSizes []int) {
	t.Helper()
	if expectedBatches > 0 && len(batches) > expectedBatches {
		t.Errorf("Expected up to %d batches, got %d", expectedBatches, len(batches))
	}
	totalProcessed := 0
	for _, b := range batches {
		if len(b) == 0 {
			t.Error("Found empty batch")
		}
		totalProcessed += len(b)
	}
	if totalProcessed != expectedTasks {
		t.Errorf("Expected %d tasks processed, got %d", expectedTasks, totalProcessed)
	}
	if len(expectedSizes) > 0 && len(batches) == len(expectedSizes) {
		for i, size := range expectedSizes {
			if i < len(batches) && len(batches[i]) != size {
				t.Errorf("Expected batch %d size %d, got %d", i, size, len(batches[i]))
			}
		}
	}
}

func TestBatchProcessor(t *testing.T) {
	t.Run("AutoRun", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		// 创建 BatchProcessor，设置最大批次大小为 10
		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d: %v", len(batch), batch)
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done() // 每个批次处理后调用 Done
			},
			asyncbatch.WithMaxSize(10),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		// 预期一个批次
		wg.Add(1)

		// 添加两个唯一任务
		if err := bp.Add("task1"); err != nil {
			t.Fatalf("Failed to add task1: %v", err)
		}
		if err := bp.Add("task2"); err != nil {
			t.Fatalf("Failed to add task2: %v", err)
		}

		// 等待批次处理完成
		waitWithTimeout(t, &wg, time.Second)

		// 验证结果
		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("Expected 1 batch, got %d", len(batches))
		}
		if len(batches[0]) != 2 {
			t.Errorf("Expected batch size 2, got %d", len(batches[0]))
		}
		if batches[0][0] != "task1" || batches[0][1] != "task2" {
			t.Errorf("Expected tasks [task1, task2], got %v", batches[0])
		}
	})

	t.Run("ContinuousProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		// Initialize BatchProcessor with maxSize=6 and upperRatio=0.5
		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d: %v", len(batch), batch)
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done() // Called once per batch
			},
			asyncbatch.WithMaxSize(6),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 9
		wg.Add(3) // Expect 3 batches of 3 tasks each

		// Add 9 unique tasks
		for i := 0; i < totalTasks; i++ {
			task := fmt.Sprintf("task%d", i)
			if err := bp.Add(task); err != nil {
				t.Fatalf("Failed to add %s: %v", task, err)
			}
		}

		// Wait for processing to complete with a timeout
		waitWithTimeout(t, &wg, time.Second)

		// Validate results
		mu.Lock()
		defer mu.Unlock()
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, totalProcessed)
		}
		if len(batches) != 3 {
			t.Errorf("Expected 3 batches, got %d", len(batches))
		}
	})

	t.Run("UnderfilledProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		// Initialize BatchProcessor with maxSize=10
		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d: %v", len(batch), batch)
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			},
			asyncbatch.WithMaxSize(10),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.3),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		// Expect 1 underfilled batch
		wg.Add(1)

		// Add 2 unique tasks
		if err := bp.Add("task1"); err != nil {
			t.Fatalf("Failed to add task1: %v", err)
		}
		if err := bp.Add("task2"); err != nil {
			t.Fatalf("Failed to add task2: %v", err)
		}

		// Wait for processing with timeout
		waitWithTimeout(t, &wg, time.Second)

		// Validate results
		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("Expected 1 batch, got %d", len(batches))
		}
		if len(batches[0]) != 2 {
			t.Errorf("Expected batch size 2, got %d", len(batches[0]))
		}
		if batches[0][0] != "task1" || batches[0][1] != "task2" {
			t.Errorf("Expected tasks [task1, task2], got %v", batches[0])
		}
	})

	t.Run("EmptyBatch", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string

		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d: %v", len(batch), batch)
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
			},
			asyncbatch.WithMaxSize(10),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 0 {
			t.Errorf("Expected no batches for empty input, got %d", len(batches))
		}
	})

	t.Run("ExtremeRatios", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			},
			asyncbatch.WithMaxSize(10),
			asyncbatch.WithUpperRatio(1.0),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		wg.Add(1) // 预期 1 个批次
		for i := 0; i < 5; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("Expected 1 batch, got %d", len(batches))
		}
	})

	t.Run("ShutdownBehavior", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			},
			asyncbatch.WithMaxSize(10),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}

		wg.Add(1) // 预期 1 个批次
		bp.Add("task1")
		waitWithTimeout(t, &wg, time.Second)
		bp.Shutdown()

		if err := bp.Add("task2"); err == nil || err.Error() != "batch processor is closed" {
			t.Errorf("Expected error 'batch processor is closed', got %v", err)
		}
	})

	t.Run("UpperRatioTrigger", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			},
			asyncbatch.WithMaxSize(10),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		wg.Add(1) // 预期 1 个批次
		for i := 0; i < 5; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 || len(batches[0]) != 5 {
			t.Errorf("Expected 1 batch of size 5, got %d batches", len(batches))
		}
	})

	t.Run("NoEmptyBatches", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			},
			asyncbatch.WithMaxSize(10),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		wg.Add(2) // 接受 1 或 2 个批次
		for i := 0; i < 10; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		defer mu.Unlock()
		if len(batches) > 2 {
			t.Errorf("Expected up to 2 batches, got %d", len(batches))
		}
	})

}

func TestInvalidParameters(t *testing.T) {
	t.Run("MissingWorker", func(t *testing.T) {
		_, err := asyncbatch.NewBatchProcessor[string](nil)
		if err == nil || !strings.Contains(err.Error(), "worker function is required") {
			t.Errorf("Expected worker required error, got: %v", err)
		}
	})

	t.Run("InvalidWaitTimes", func(t *testing.T) {
		_, err := asyncbatch.NewBatchProcessor[string](
			func([]string) {},
			asyncbatch.WithFixedWait(500*time.Millisecond),
			asyncbatch.WithUnderfilledWait(100*time.Millisecond),
		)
		if err == nil || !strings.Contains(err.Error(), "fixedWait must be less") {
			t.Errorf("Expected wait time error, got: %v", err)
		}
	})

	t.Run("TooManyWorkers", func(t *testing.T) {
		_, err := asyncbatch.NewBatchProcessor[string](
			func([]string) {},
			asyncbatch.WithNumWorkers(9),
			asyncbatch.WithMaxSize(100),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
		)
		if err == nil || !strings.Contains(err.Error(), "numWorkers must be between") {
			t.Errorf("Expected worker count error, got: %v", err)
		}
	})

	t.Run("InvalidRatios", func(t *testing.T) {
		_, err := asyncbatch.NewBatchProcessor[string](
			func([]string) {},
			asyncbatch.WithUpperRatio(0.2),
			asyncbatch.WithLowerRatio(0.5),
		)
		if err == nil || !strings.Contains(err.Error(), "upperRatio must be greater") {
			t.Errorf("Expected ratio error, got: %v", err)
		}
	})
}

func TestTinyRatios(t *testing.T) {
	var mu sync.Mutex
	var batches [][]string
	var wg sync.WaitGroup

	// 配置 BatchProcessor，upperRatio=0.001 导致每个批次上限为 1
	bp, err := asyncbatch.NewBatchProcessor[string](
		func(batch []string) {
			t.Logf("Processing micro batch of size %d: %v", len(batch), batch)
			mu.Lock()
			batches = append(batches, batch)
			mu.Unlock()
			wg.Done() // 每个批次处理后调用一次 wg.Done()
		},
		asyncbatch.WithMaxSize(1000),
		asyncbatch.WithUpperRatio(0.001), // upperThreshold = 1
		asyncbatch.WithLowerRatio(0.0005),
		asyncbatch.WithFixedWait(1*time.Millisecond),
		asyncbatch.WithUnderfilledWait(2*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewBatchProcessor failed: %v", err)
	}
	defer bp.Shutdown()

	totalTasks := 50
	wg.Add(totalTasks) // 预期 50 个批次，每个批次 1 个任务

	// 生成 50 个唯一任务
	for i := 0; i < totalTasks; i++ {
		task := fmt.Sprintf("task%d", i)
		for {
			if err := bp.Add(task); err == nil {
				break
			}
			time.Sleep(100 * time.Microsecond)
		}
	}

	// 等待所有批次处理完成
	waitWithTimeout(t, &wg, 3*time.Second)

	// 验证批次数量和任务唯一性
	mu.Lock()
	defer mu.Unlock()
	if len(batches) != totalTasks {
		t.Errorf("Expected %d batches, got %d", totalTasks, len(batches))
	}
	for i, batch := range batches {
		if len(batch) != 1 {
			t.Errorf("Batch %d: expected size 1, got %d", i, len(batch))
		}
		if batch[0] != fmt.Sprintf("task%d", i) {
			t.Errorf("Batch %d: expected task%d, got %s", i, i, batch[0])
		}
	}
}

func TestMultipleShutdown(t *testing.T) {
	bp, err := asyncbatch.NewBatchProcessor[string](
		func([]string) {},
		asyncbatch.WithMaxSize(10),
		asyncbatch.WithUpperRatio(0.5),
		asyncbatch.WithLowerRatio(0.1),
		asyncbatch.WithFixedWait(100*time.Millisecond),
		asyncbatch.WithUnderfilledWait(200*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewBatchProcessor failed: %v", err)
	}

	bp.Shutdown()
	bp.Shutdown() // Should not panic
}

func TestGetterMethods(t *testing.T) {
	bp, err := asyncbatch.NewBatchProcessor[string](
		func([]string) {},
		asyncbatch.WithMaxSize(50),
		asyncbatch.WithUpperRatio(0.7),
		asyncbatch.WithLowerRatio(0.2),
		asyncbatch.WithFixedWait(100*time.Millisecond),
		asyncbatch.WithUnderfilledWait(500*time.Millisecond),
		asyncbatch.WithNumWorkers(2),
	)
	if err != nil {
		t.Fatalf("NewBatchProcessor failed: %v", err)
	}
	defer bp.Shutdown()

	if bp.MaxSize() != 50 {
		t.Errorf("Expected MaxSize 50, got %d", bp.MaxSize())
	}
	if bp.UpperRatio() != 0.7 {
		t.Errorf("Expected UpperRatio 0.7, got %f", bp.UpperRatio())
	}
	if bp.LowerRatio() != 0.2 {
		t.Errorf("Expected LowerRatio 0.2, got %f", bp.LowerRatio())
	}
	if bp.FixedWait() != 100*time.Millisecond {
		t.Errorf("Expected FixedWait 100ms, got %v", bp.FixedWait())
	}
	if bp.UnderfilledWait() != 500*time.Millisecond {
		t.Errorf("Expected UnderfilledWait 500ms, got %v", bp.UnderfilledWait())
	}
	if bp.NumWorkers() != 2 {
		t.Errorf("Expected NumWorkers 2, got %d", bp.NumWorkers())
	}
	if bp.Worker() == nil {
		t.Error("Expected non-nil Worker")
	}
}

func BenchmarkBatchProcessor(b *testing.B) {
	var mu sync.Mutex
	var batchSizes []int

	bp, _ := asyncbatch.NewBatchProcessor[string](
		func(batch []string) {
			mu.Lock()
			batchSizes = append(batchSizes, len(batch))
			mu.Unlock()
		},
		asyncbatch.WithMaxSize(100),
		asyncbatch.WithUpperRatio(0.5),
		asyncbatch.WithLowerRatio(0.1),
		asyncbatch.WithFixedWait(100*time.Millisecond),
		asyncbatch.WithUnderfilledWait(200*time.Millisecond),
		asyncbatch.WithNumWorkers(4),
	)
	defer bp.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for {
			if err := bp.Add("task"); err == nil {
				break
			}
			time.Sleep(10 * time.Microsecond)
		}
	}
	b.StopTimer()

	mu.Lock()
	defer mu.Unlock()
	totalTasks := 0
	for _, size := range batchSizes {
		totalTasks += size
	}
	b.Logf("Processed %d tasks in %d batches, avg batch size: %.2f",
		totalTasks, len(batchSizes), float64(totalTasks)/float64(len(batchSizes)))
}

// 测试并发添加任务的正确性和完整性
func TestConcurrentAdd(t *testing.T) {
	const numTasks = 5000
	var mu sync.Mutex
	processed := make(map[int]struct{})
	var wg sync.WaitGroup

	bp, _ := asyncbatch.NewBatchProcessor[int](
		func(batch []int) {
			mu.Lock()
			defer mu.Unlock()
			for _, v := range batch {
				processed[v] = struct{}{}
			}
			wg.Add(-len(batch)) // 每个任务完成时减少计数
		},
		asyncbatch.WithMaxSize(100),
		asyncbatch.WithNumWorkers(8),
	)
	defer bp.Shutdown()

	wg.Add(numTasks)

	// 启动多个生产者
	for i := 0; i < 5; i++ {
		go func(start int) {
			for j := 0; j < numTasks/5; j++ {
				task := start*1000 + j
				for {
					if err := bp.Add(task); err == nil {
						break
					}
					time.Sleep(time.Microsecond)
				}
			}
		}(i)
	}

	waitWithTimeout(t, &wg, 5*time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(processed) != numTasks {
		t.Errorf("Processed %d tasks, expected %d", len(processed), numTasks)
	}
}

// 测试通道满时的处理逻辑
func TestFullChannelBehavior(t *testing.T) {
	bp, _ := asyncbatch.NewBatchProcessor[string](
		func([]string) {},
		asyncbatch.WithMaxSize(10),
		asyncbatch.WithNumWorkers(1),
	)
	defer bp.Shutdown()

	// 使用新添加的方法获取容量
	maxCapacity := bp.TasksCap()

	// 填充通道
	for i := 0; i < maxCapacity; i++ {
		bp.Add(fmt.Sprintf("task%d", i))
	}

	// 测试通道满的情况
	err := bp.Add("overflow")
	if err == nil || !strings.Contains(err.Error(), "task channel is full") {
		t.Errorf("Expected channel full error, got: %v", err)
	}

	// 使用回调函数触发 Done
	var wg sync.WaitGroup
	wg.Add(1)

	// 创建新的 processor 来避免修改私有字段
	newBp, _ := asyncbatch.NewBatchProcessor[string](
		func([]string) { wg.Done() },
		asyncbatch.WithMaxSize(10),
	)
	defer newBp.Shutdown()

	newBp.Add("recovery")
	waitWithTimeout(t, &wg, time.Second)
}

// 验证关闭时剩余任务的完整处理
func TestGracefulShutdown(t *testing.T) {
	const pendingTasks = 25
	var processed int
	var mu sync.Mutex
	var wg sync.WaitGroup

	bp, _ := asyncbatch.NewBatchProcessor[int](
		func(batch []int) {
			mu.Lock()
			processed += len(batch)
			mu.Unlock()
			// 每个批次处理完成后减少计数
			wg.Add(-len(batch))
			time.Sleep(5 * time.Millisecond) // 保留小量延迟模拟处理
		},
		asyncbatch.WithMaxSize(10),
		asyncbatch.WithNumWorkers(2), // 增加 worker 数量
	)

	wg.Add(pendingTasks)
	for i := 0; i < pendingTasks; i++ {
		bp.Add(i)
	}

	// 保证足够时间让所有任务入队
	time.Sleep(50 * time.Millisecond)
	bp.Shutdown()

	// 使用带超时的等待机制
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for shutdown")
		default:
			mu.Lock()
			if processed == pendingTasks {
				mu.Unlock()
				return
			}
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// 验证默认参数配置
func TestDefaultConfiguration(t *testing.T) {
	bp, err := asyncbatch.NewBatchProcessor[string](func([]string) {})
	if err != nil {
		t.Fatal(err)
	}
	defer bp.Shutdown()

	if bp.MaxSize() != 1000 {
		t.Errorf("Default maxSize: expected 1000, got %d", bp.MaxSize())
	}
	if bp.NumWorkers() != 1 {
		t.Errorf("Default numWorkers: expected 1, got %d", bp.NumWorkers())
	}
	if bp.UpperRatio() != 0.5 {
		t.Errorf("Default upperRatio: expected 0.5, got %f", bp.UpperRatio())
	}
}

// 测试边界条件下的批次分割
// asyncbatch_test.go
func TestEdgeCaseBatchSplitting(t *testing.T) {
	testCases := []struct {
		maxSize       int
		upperRatio    float64
		addCount      int
		expectBatches int
	}{
		{100, 0.5, 50, 1}, // 触发 upperRatio
		{100, 0.5, 51, 2}, // 分两批处理：50（upperRatio） + 1（超时）
		{100, 0.5, 49, 1}, // 不触发任何阈值
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("size%d-add%d", tc.maxSize, tc.addCount), func(t *testing.T) {
			var (
				batchCount int32
				processed  int32
				wg         sync.WaitGroup
			)

			bp, _ := asyncbatch.NewBatchProcessor[int](
				func(batch []int) {
					count := len(batch)
					atomic.AddInt32(&batchCount, 1)
					atomic.AddInt32(&processed, int32(count))
					wg.Add(-count) // 按实际处理数量减少计数
				},
				asyncbatch.WithMaxSize(tc.maxSize),
				asyncbatch.WithUpperRatio(tc.upperRatio),
				asyncbatch.WithFixedWait(50*time.Millisecond),
				asyncbatch.WithUnderfilledWait(100*time.Millisecond),
				asyncbatch.WithNumWorkers(1), // 单worker确保顺序
			)
			defer bp.Shutdown()

			wg.Add(tc.addCount)
			for i := 0; i < tc.addCount; i++ {
				bp.Add(i)
			}

			// 精确等待处理完成
			assertWait(t, &wg, 2*time.Second)

			if batchCount != int32(tc.expectBatches) {
				t.Errorf("Expected %d batches, got %d", tc.expectBatches, batchCount)
			}
			if processed != int32(tc.addCount) {
				t.Errorf("Expected %d processed, got %d", tc.addCount, processed)
			}
		})
	}
}

// 专用等待函数
func assertWait(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("Test timed out")
	}
}
