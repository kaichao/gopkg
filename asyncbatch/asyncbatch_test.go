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

func TestBatchProcessor(t *testing.T) {
	t.Run("AutoRun", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Add(-len(batch))
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

		wg.Add(2)
		bp.Add("task1")
		bp.Add("task2")
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		defer mu.Unlock()
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != 2 {
			t.Errorf("Expected 2 tasks processed, got %d", totalProcessed)
		}
	})

	t.Run("ContinuousProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Add(-len(batch))
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
		wg.Add(totalTasks)
		for i := 0; i < totalTasks; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, time.Second)

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

	t.Run("UnderfilledProcessing", func(t *testing.T) {
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
			asyncbatch.WithLowerRatio(0.3),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
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
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("Expected 1 underfilled batch, got %d", len(batches))
		}
		if len(batches) > 0 && len(batches[0]) != 2 {
			t.Errorf("Expected batch size 2, got %d", len(batches[0]))
		}
	})

	t.Run("MultipleWorkers", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Add(-len(batch))
			},
			asyncbatch.WithMaxSize(5),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
			asyncbatch.WithNumWorkers(3),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 20
		wg.Add(totalTasks)
		for i := 0; i < totalTasks; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, time.Second)

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

	t.Run("EmptyBatch", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string

		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
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

		wg.Add(1)
		for i := 0; i < 5; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("Expected 1 batch with high upperRatio, got %d", len(batches))
		}
	})

	t.Run("HighLoad", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Add(-len(batch))
			},
			asyncbatch.WithMaxSize(100),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
			asyncbatch.WithNumWorkers(4),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 1000
		wg.Add(totalTasks)
		for i := 0; i < totalTasks; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
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
	})

	t.Run("ChannelFull", func(t *testing.T) {
		const capacity = 4
		bp, _ := asyncbatch.NewBatchProcessor[string](
			func(_ []string) {},
			asyncbatch.WithMaxSize(2),
			asyncbatch.WithNumWorkers(1),
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
			func(batch []string) {
				if len(batch) == 0 {
					t.Error("Empty batch processed")
					return
				}
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Add(-len(batch))
			},
			asyncbatch.WithMaxSize(5),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(50*time.Millisecond),
			asyncbatch.WithUnderfilledWait(100*time.Millisecond),
			asyncbatch.WithNumWorkers(8),
		)
		if err != nil {
			t.Fatalf("NewBatchProcessor failed: %v", err)
		}
		defer bp.Shutdown()

		totalTasks := 20
		wg.Add(totalTasks)

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
			func(batch []string) {
				t.Logf("Processing batch of size %d", len(batch))
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
			},
			asyncbatch.WithMaxSize(1000),
			asyncbatch.WithUpperRatio(0.5),
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
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
		defer mu.Unlock()
		if len(batches) != expectedBatches {
			t.Errorf("Expected %d batches, got %d", expectedBatches, len(batches))
		}
		totalProcessed := 0
		for _, b := range batches {
			totalProcessed += len(b)
		}
		if totalProcessed != totalTasks {
			t.Errorf("Expected %d tasks processed, got %d", totalTasks, totalProcessed)
		}
	})

	t.Run("MultipleShutdown", func(t *testing.T) {
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
	})

	t.Run("NoEmptyBatches", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

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
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
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
		time.Sleep(20 * time.Millisecond)
		bp.Shutdown()
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		defer mu.Unlock()
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
	})

	t.Run("SmallBatchPrevention", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

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
			asyncbatch.WithLowerRatio(0.1),
			asyncbatch.WithFixedWait(100*time.Millisecond),
			asyncbatch.WithUnderfilledWait(200*time.Millisecond),
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
		time.Sleep(20 * time.Millisecond)
		for i := 0; i < 2; i++ {
			bp.Add(fmt.Sprintf("task%d", i+10))
		}
		time.Sleep(20 * time.Millisecond)
		waitWithTimeout(t, &wg, time.Second)

		mu.Lock()
		defer mu.Unlock()
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
	})

	t.Run("ContinuousProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		var wg sync.WaitGroup

		// 修改点：worker作为第一个参数，移除所有选项的泛型参数
		bp, err := asyncbatch.NewBatchProcessor[string](
			func(batch []string) {
				t.Logf("Processing batch of size %d: %v", len(batch), batch)
				mu.Lock()
				batches = append(batches, batch)
				mu.Unlock()
				wg.Done()
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
		expectedBatches := 2 // 6 + 3
		wg.Add(expectedBatches)

		// 添加任务逻辑保持不变
		for i := 0; i < 6; i++ {
			bp.Add(fmt.Sprintf("task%d", i))
		}
		time.Sleep(20 * time.Millisecond)
		for i := 0; i < 3; i++ {
			bp.Add(fmt.Sprintf("task%d", i+6))
		}
		time.Sleep(20 * time.Millisecond)
		waitWithTimeout(t, &wg, time.Second)

		// 断言逻辑保持不变
		mu.Lock()
		defer mu.Unlock()
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

	// 修改点：worker作为第一个参数，选项参数移除泛型
	bp, err := asyncbatch.NewBatchProcessor[string](
		func(batch []string) {
			if len(batch) == 0 {
				return
			}
			t.Logf("Processing micro batch of size %d", len(batch))
			mu.Lock()
			batches = append(batches, batch)
			mu.Unlock()
			wg.Add(-len(batch))
		},
		asyncbatch.WithMaxSize(maxSize),
		asyncbatch.WithUpperRatio(0.001),
		asyncbatch.WithLowerRatio(0.0005),
		asyncbatch.WithFixedWait(1*time.Millisecond),
		asyncbatch.WithUnderfilledWait(2*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Initialization failed: %v", err)
	}
	defer bp.Shutdown()

	wg.Add(totalTasks)

	// 任务添加逻辑保持不变
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

	// 结果验证逻辑保持不变
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
	bp, _ := asyncbatch.NewBatchProcessor[string](
		func([]string) {},
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
		bp.Add("task")
	}
}

func TestConcurrentAdd(t *testing.T) {
	var (
		mu             sync.Mutex
		processedTasks = make(map[string]struct{})
		wg             sync.WaitGroup
	)

	bp, _ := asyncbatch.NewBatchProcessor[string](
		func(batch []string) {
			mu.Lock()
			defer mu.Unlock()
			for _, task := range batch {
				processedTasks[task] = struct{}{}
			}
		},
		asyncbatch.WithMaxSize(10),
		asyncbatch.WithNumWorkers(8), // 增加工作线程数
		asyncbatch.WithFixedWait(5*time.Millisecond),
		asyncbatch.WithUnderfilledWait(20*time.Millisecond),
	)
	defer bp.Shutdown()

	totalTasks := 100
	wg.Add(totalTasks)

	// 使用带缓冲的channel控制并发量
	sem := make(chan struct{}, 100)
	for i := 0; i < totalTasks; i++ {
		sem <- struct{}{}
		go func(i int) {
			defer func() {
				<-sem
				wg.Done()
			}()
			for { // 重试机制
				if err := bp.Add(fmt.Sprintf("task%d", i)); err == nil {
					return
				}
				time.Sleep(time.Microsecond * 100)
			}
		}(i)
	}

	// 等待所有添加操作完成
	wg.Wait()

	// 等待处理完成
	bp.Shutdown()

	mu.Lock()
	defer mu.Unlock()
	if len(processedTasks) != totalTasks {
		t.Errorf("Expected %d tasks processed, got %d", totalTasks, len(processedTasks))
	}
}

func TestStressTest(t *testing.T) {
	var (
		processed int64
		wg        sync.WaitGroup
	)

	// 确保使用正确的初始化参数
	bp, err := asyncbatch.NewBatchProcessor[string](
		func(batch []string) {
			atomic.AddInt64(&processed, int64(len(batch)))
		},
		asyncbatch.WithMaxSize(500),
		asyncbatch.WithNumWorkers(16),
		asyncbatch.WithFixedWait(1*time.Millisecond),
		asyncbatch.WithUnderfilledWait(5*time.Millisecond),
		asyncbatch.WithUpperRatio(0.8),
		asyncbatch.WithLowerRatio(0.2),
	)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer bp.Shutdown()

	totalTasks := 10000
	wg.Add(totalTasks)

	// 使用可控的并发写入
	concurrency := 100
	sem := make(chan struct{}, concurrency)

	for i := 0; i < totalTasks; i++ {
		sem <- struct{}{}
		go func(i int) {
			defer func() {
				<-sem
				wg.Done()
			}()
			for {
				if err := bp.Add(fmt.Sprintf("task%d", i)); err == nil {
					return
				}
				time.Sleep(10 * time.Microsecond)
			}
		}(i)
	}

	wg.Wait()     // 等待所有添加完成
	bp.Shutdown() // 确保处理完成

	if atomic.LoadInt64(&processed) != int64(totalTasks) {
		t.Errorf("Processed %d/%d tasks", processed, totalTasks)
	}
}
