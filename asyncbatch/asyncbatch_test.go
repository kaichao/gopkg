package asyncbatch_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaichao/gopkg/asyncbatch"
)

func TestBatchProcessor(t *testing.T) {

	t.Run("AutoRun", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Add(-len(batch)) // 按实际处理的任务数递减
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		wg.Add(2) // 总任务数
		bp.Add("task1")
		bp.Add("task2")
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != 2 {
			t.Errorf("Expected 2 tasks processed, got %d", totalProcessed)
		}
		mu.Unlock()
	})

	t.Run("ContinuousProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](6),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Add(-len(batch)) // 关键修改点
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 9
		wg.Add(totalTasks) // 总任务数
		for i := 0; i < totalTasks; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, totalProcessed)
		}
		mu.Unlock()
	})

	t.Run("UnderfilledProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.3),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		wg.Add(1)
		bp.Add("task1")
		bp.Add("task2")
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		if len(batches) != 1 {
			t.Errorf("Expected 1 underfilled batch, got %d", len(batches))
		}
		if len(batches) > 0 && len(batches[0]) != 2 {
			t.Errorf("Expected batch size 2, got %d", len(batches[0]))
		}
		mu.Unlock()
	})

	t.Run("MultipleWorkers", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](5),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithNumWorkers[string](3),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				// 每个任务调用Done()
				wg.Add(-len(batch)) // 或使用 defer wg.Add(-len(batch))
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 20
		wg.Add(totalTasks) // 添加总任务数到WaitGroup
		for i := 0; i < totalTasks; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, time.Second) // 等待所有任务完成

		mu.Lock()
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, totalProcessed)
		}
		mu.Unlock()
	})

	t.Run("EmptyBatch", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		if len(batches) != 0 {
			t.Errorf("Expected no batches for empty input, got %d", len(batches))
		}
		mu.Unlock()
	})

	t.Run("ExtremeRatios", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithUpperRatio[string](1.0),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		wg.Add(1)
		for i := 0; i < 5; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		if len(batches) != 1 {
			t.Errorf("Expected 1 batch with high upperRatio, got %d", len(batches))
		}
		mu.Unlock()
	})

	t.Run("HighLoad", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](100),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithNumWorkers[string](4),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Add(-len(batch)) // 关键修改点
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 1000
		wg.Add(totalTasks) // 总任务数
		for i := 0; i < totalTasks; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, 2*time.Second)

		mu.Lock()
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, totalProcessed)
		}
		mu.Unlock()
	})

	t.Run("ShutdownBehavior", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}

		wg.Add(1)
		bp.Add("task1")
		waitWithTimeout(t, &wg, time.Second)
		bp.Shutdown()

		err = bp.Add("task2")
		if err == nil || err.Error() != "batch processor is closed" {
			t.Errorf("Expected error 'batch processor is closed', got %v", err)
		}
	})

	t.Run("GetterMethods", func(t *testing.T) {
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](50),
			asyncbatch.WithUpperRatio[string](0.7),
			asyncbatch.WithLowerRatio[string](0.2),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](500*time.Millisecond),
			asyncbatch.WithNumWorkers[string](2),
			asyncbatch.WithWorker[string](func([]string) {}),
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
	})

	t.Run("ChannelFull", func(t *testing.T) {
		const (
			capacity = 4
		)
		bp, _ := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](2),
			asyncbatch.WithNumWorkers[string](1),
			asyncbatch.WithWorker[string](func(_ []string) {}),
		)
		defer bp.Shutdown()

		// 填充通道
		for i := 0; i < capacity; i++ {
			if err := bp.Add(fmt.Sprintf("task%d", i)); err != nil {
				t.Fatal(err)
			}
		}

		// 验证通道满
		if err := bp.Add("overflow"); err == nil {
			t.Error("Expected channel full error")
		}

		// 处理任务恢复通道
		bp.Shutdown()
		if err := bp.Add("new_task"); err == nil {
			t.Error("Should reject after shutdown")
		}
	})

	t.Run("MaxWorkers", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](5),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](50*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](100*time.Millisecond),
			asyncbatch.WithNumWorkers[string](8),
			asyncbatch.WithWorker[string](func(batch []string) {
				if len(batch) == 0 {
					t.Error("Empty batch processed")
					return
				}
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Add(-len(batch))
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 20
		wg.Add(totalTasks)

		// 带缓冲的任务添加
		// 增加通道容量检查
		start := time.Now()
		for i := 0; i < totalTasks; {
			if time.Since(start) > 2*time.Second {
				t.Fatal("Timeout adding tasks")
			}
			if err := bp.Add(fmt.Sprintf("task%d", i)); err == nil {
				i++
			} else {
				time.Sleep(time.Microsecond * 100)
			}
		}

		waitWithTimeout(t, &wg, 2*time.Second)

		mu.Lock()
		defer mu.Unlock()

		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, totalProcessed)
		}
	})

	t.Run("LargeMaxSize", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](1000),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 500
		expectedBatches := (totalTasks + bp.MaxSize() - 1) / bp.MaxSize()
		wg.Add(expectedBatches)
		for i := 0; i < totalTasks; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		if len(batches) != expectedBatches {
			t.Errorf("Expected %d batch with 500 tasks, got %d", expectedBatches, len(batches))
		}
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, totalProcessed)
		}
		mu.Unlock()
	})

	t.Run("MultipleShutdown", func(t *testing.T) {
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func([]string) {}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}

		bp.Shutdown()
		bp.Shutdown() // Should not panic
	})

	t.Run("NoEmptyBatches", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d: %v", len(batch), batch)
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}

		totalTasks := 10
		expectedBatches := 1
		wg.Add(expectedBatches)
		for i := 0; i < totalTasks; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		time.Sleep(20 * time.Millisecond) // Ensure tasks are processed
		bp.Shutdown()
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		if len(batches) != expectedBatches {
			t.Errorf("Expected %d batch, got %d", expectedBatches, len(batches))
		}
		for i, b := range batches {
			if len(b) == 0 {
				t.Errorf("Found empty batch at index %d", i)
			}
		}
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, totalProcessed)
		}
		mu.Unlock()
	})

	t.Run("SmallBatchPrevention", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d: %v", len(batch), batch)
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 12
		expectedBatches := 2
		wg.Add(expectedBatches)
		for i := 0; i < 10; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		time.Sleep(20 * time.Millisecond) // Ensure first batch processes
		for i := 0; i < 2; i++ {
			bp.Add(fmt.Sprintf("task%d", i+10))
		}
		time.Sleep(20 * time.Millisecond) // Ensure second batch collects tasks
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		if len(batches) != expectedBatches {
			t.Errorf("Expected %d batches, got %d", expectedBatches, len(batches))
		}
		if len(batches) > 0 && len(batches[0]) != 10 {
			t.Errorf("Expected first batch size 10, got %d", len(batches[0]))
		}
		if len(batches) > 1 && len(batches[1]) != 2 {
			t.Errorf("Expected second batch size 2, got %d", len(batches[1]))
		}
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, totalProcessed)
		}
		mu.Unlock()
	})

	t.Run("ContinuousProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](6),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](100*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
			asyncbatch.WithWorker[string](func(batch []string) {
				t.Logf("Processing batch of size %d: %v", len(batch), batch)
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 9
		expectedBatches := 2 // 6 + 3
		wg.Add(expectedBatches)
		for i := 0; i < 6; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		time.Sleep(20 * time.Millisecond) // Ensure first batch processes
		for i := 0; i < 3; i++ {
			bp.Add(fmt.Sprintf("task%d", i+6))
		}
		time.Sleep(20 * time.Millisecond) // Ensure second batch collects tasks
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		if len(batches) != expectedBatches {
			t.Errorf("Expected %d batches, got %d", expectedBatches, len(batches))
		}
		if len(batches) > 0 && len(batches[0]) != 6 {
			t.Errorf("Expected first batch size 6, got %d", len(batches[0]))
		}
		if len(batches) > 1 && len(batches[1]) != 3 {
			t.Errorf("Expected second batch size 3, got %d", len(batches[1]))
		}
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, totalProcessed)
		}
		mu.Unlock()
	})

}

// waitWithTimeout waits for WaitGroup with a timeout.
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

func TestInvalidParameters(t *testing.T) {
	t.Run("MissingWorker", func(t *testing.T) {
		_, err := asyncbatch.NewBatchProcessor[string]()
		if err == nil || !strings.Contains(err.Error(), "worker function is required") {
			t.Errorf("Expected worker required error, got: %v", err)
		}
	})

	t.Run("InvalidWaitTimes", func(t *testing.T) {
		_, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithFixedWait[string](500*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](100*time.Millisecond),
			asyncbatch.WithWorker[string](func([]string) {}),
		)
		if err == nil || !strings.Contains(err.Error(), "fixedWait must be less") {
			t.Errorf("Expected wait time error, got: %v", err)
		}
	})

	t.Run("TooManyWorkers", func(t *testing.T) {
		_, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithNumWorkers[string](9),
			asyncbatch.WithWorker[string](func([]string) {}),
			asyncbatch.WithMaxSize[string](100),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
		)
		if err == nil || !strings.Contains(err.Error(), "numWorkers must be between") {
			t.Errorf("Expected worker count error, got: %v", err)
		}
	})

	t.Run("InvalidRatios", func(t *testing.T) {
		_, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithUpperRatio[string](0.2),
			asyncbatch.WithLowerRatio[string](0.5),
			asyncbatch.WithWorker[string](func([]string) {}),
		)
		if err == nil || !strings.Contains(err.Error(), "upperRatio must be greater") {
			t.Errorf("Expected ratio error, got: %v", err)
		}
	})
}

func TestTinyRatios(t *testing.T) {
	const (
		maxSize    = 1000
		totalTasks = 50
	)

	var (
		mu          sync.Mutex
		batches     [][]string
		wg          sync.WaitGroup
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	)
	defer cancel()

	bp, err := asyncbatch.NewBatchProcessor[string](
		asyncbatch.WithMaxSize[string](maxSize),
		asyncbatch.WithUpperRatio[string](0.001),
		asyncbatch.WithLowerRatio[string](0.0005),
		asyncbatch.WithFixedWait[string](1*time.Millisecond),
		asyncbatch.WithUnderfilledWait[string](2*time.Millisecond),
		asyncbatch.WithWorker[string](func(batch []string) {
			if len(batch) == 0 {
				return
			}
			t.Logf("Processing micro batch of size %d", len(batch))
			mu.Lock()
			batches = append(batches, batch)
			mu.Unlock()
			wg.Add(-len(batch))
		}),
	)
	if err != nil {
		t.Fatalf("Initialization failed: %v", err)
	}
	defer bp.Shutdown()

	wg.Add(totalTasks)

	for i := 0; i < totalTasks; {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout adding tasks")
		default:
			if err := bp.Add(fmt.Sprintf("task%d", i)); err == nil {
				i++
			} else {
				time.Sleep(time.Microsecond * 100)
			}
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("Test timed out")
	}

	mu.Lock()
	defer mu.Unlock()

	totalProcessed := 0
	for _, b := range batches {
		totalProcessed += len(b)
		if len(b) == 0 {
			t.Error("Empty batch detected")
		}
	}

	if totalProcessed != totalTasks {
		t.Errorf("Processed %d tasks, expected %d", totalProcessed, totalTasks)
	}
}

// BenchmarkBatchProcessor measures throughput.
func BenchmarkBatchProcessor(b *testing.B) {
	bp, err := asyncbatch.NewBatchProcessor[string](
		asyncbatch.WithMaxSize[string](100),
		asyncbatch.WithUpperRatio[string](0.5),
		asyncbatch.WithLowerRatio[string](0.1),
		asyncbatch.WithFixedWait[string](100*time.Millisecond),
		asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
		asyncbatch.WithNumWorkers[string](4),
		asyncbatch.WithWorker[string](func([]string) {}),
	)
	if err != nil {
		b.Fatalf("NewBatchProcessor failed: %v", err)
	}
	defer bp.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.Add("task")
	}
}

func TestConcurrentAdd(t *testing.T) {
	t.Run("ConcurrentAdd", func(t *testing.T) {
		var mu sync.Mutex
		processedTasks := make(map[string]bool)
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](5*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](20*time.Millisecond),
			asyncbatch.WithNumWorkers[string](4), // Increased to 4
			asyncbatch.WithWorker[string](func(batch []string) {
				mu.Lock()
				for _, task := range batch {
					processedTasks[task] = true
				}
				mu.Unlock()
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 100
		tasks := make([]string, totalTasks)
		for i := 0; i < totalTasks; i++ {
			tasks[i] = fmt.Sprintf("task%d", i)
		}

		var addWg sync.WaitGroup
		for i := 0; i < 10; i++ {
			addWg.Add(1)
			go func(start int) {
				defer addWg.Done()
				for j := 0; j < 10; j++ {
					bp.Add(tasks[start+j])
				}
			}(i * 10)
		}
		addWg.Wait()

		time.Sleep(5 * time.Second) // Extended to 5s

		mu.Lock()
		if len(processedTasks) != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, len(processedTasks))
		}
		mu.Unlock()
	})
}

func TestStressTest(t *testing.T) {
	t.Run("StressTest", func(t *testing.T) {
		var mu sync.Mutex
		processedTasks := make(map[string]bool)
		bp, err := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](100),
			asyncbatch.WithUpperRatio[string](0.5),
			asyncbatch.WithLowerRatio[string](0.1),
			asyncbatch.WithFixedWait[string](5*time.Millisecond),
			asyncbatch.WithUnderfilledWait[string](20*time.Millisecond),
			asyncbatch.WithNumWorkers[string](8), // 增加工作者数量
			asyncbatch.WithWorker[string](func(batch []string) {
				mu.Lock()
				for _, task := range batch {
					processedTasks[task] = true
				}
				mu.Unlock()
			}),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 10000
		tasks := make([]string, totalTasks)
		for i := 0; i < totalTasks; i++ {
			tasks[i] = fmt.Sprintf("task%d", i)
		}

		// 添加任务
		for _, task := range tasks {
			bp.Add(task)
		}

		// 等待处理完成，延长超时时间
		time.Sleep(30 * time.Second)

		// 检查结果
		mu.Lock()
		if len(processedTasks) != totalTasks {
			t.Errorf("预期处理 %d 个任务，实际处理 %d 个", totalTasks, len(processedTasks))
		}
		mu.Unlock()
	})
}
