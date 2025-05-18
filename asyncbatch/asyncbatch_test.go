package asyncbatch_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kaichao/gopkg/asyncbatch"
)

func TestBatchProcessor(t *testing.T) {
	t.Run("CreateWithOptions", func(t *testing.T) {
		worker := func(batch []string) {}
		bp := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithMaxWait[string](500*time.Millisecond),
			asyncbatch.WithEmptyWait[string](200*time.Millisecond),
			asyncbatch.WithPartialWait[string](300*time.Millisecond),
			asyncbatch.WithWorker[string](worker),
		)
		if bp.MaxSize() != 10 {
			t.Errorf("expected maxSize 10, got %d", bp.MaxSize())
		}
		if bp.MaxWait() != 500*time.Millisecond {
			t.Errorf("expected maxWait 500ms, got %v", bp.MaxWait())
		}
		if bp.EmptyWait() != 200*time.Millisecond {
			t.Errorf("expected emptyWait 200ms, got %v", bp.EmptyWait())
		}
		if bp.PartialWait() != 300*time.Millisecond {
			t.Errorf("expected partialWait 300ms, got %v", bp.PartialWait())
		}
		if bp.Worker() == nil {
			t.Error("expected worker to be set, got nil")
		}
	})

	t.Run("AddAndProcessBatch", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		worker := func(batch []string) {
			mu.Lock()
			batches = append(batches, batch)
			mu.Unlock()
		}
		bp := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](3),
			asyncbatch.WithMaxWait[string](time.Second),
			asyncbatch.WithWorker[string](worker),
		)
		go bp.Run()

		if err := bp.Add("task1"); err != nil {
			t.Errorf("Add failed: %v", err)
		}
		if err := bp.Add("task2"); err != nil {
			t.Errorf("Add failed: %v", err)
		}
		if err := bp.Add("task3"); err != nil {
			t.Errorf("Add failed: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
		bp.Shutdown()

		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("expected 1 batch, got %d", len(batches))
		}
		if len(batches) > 0 && len(batches[0]) != 3 {
			t.Errorf("expected batch size 3, got %d", len(batches[0]))
		}
	})

	t.Run("TimeoutProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		worker := func(batch []string) {
			mu.Lock()
			batches = append(batches, batch)
			mu.Unlock()
		}
		bp := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithMaxWait[string](time.Second),
			asyncbatch.WithPartialWait[string](100*time.Millisecond),
			asyncbatch.WithWorker[string](worker),
		)
		go bp.Run()

		bp.Add("task1")
		bp.Add("task2")

		time.Sleep(200 * time.Millisecond)
		bp.Shutdown()

		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("expected 1 batch, got %d", len(batches))
		}
		if len(batches) > 0 && len(batches[0]) != 2 {
			t.Errorf("expected batch size 2, got %d", len(batches[0]))
		}
	})

	t.Run("EmptyWaitProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var called bool
		worker := func(batch []string) {
			mu.Lock()
			called = true
			mu.Unlock()
		}
		bp := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithEmptyWait[string](100*time.Millisecond),
			asyncbatch.WithWorker[string](worker),
		)
		go bp.Run()

		time.Sleep(150 * time.Millisecond)
		bp.Shutdown()

		mu.Lock()
		defer mu.Unlock()
		if called {
			t.Error("worker should not be called with empty batch")
		}
	})

	t.Run("PartialWaitProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		worker := func(batch []string) {
			mu.Lock()
			batches = append(batches, batch)
			mu.Unlock()
		}
		bp := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithPartialWait[string](100*time.Millisecond),
			asyncbatch.WithWorker[string](worker),
		)
		go bp.Run()

		bp.Add("task1")
		bp.Add("task2")

		time.Sleep(150 * time.Millisecond)
		bp.Shutdown()

		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("expected 1 batch, got %d", len(batches))
		}
		if len(batches) > 0 && len(batches[0]) != 2 {
			t.Errorf("expected batch size 2, got %d", len(batches[0]))
		}
	})

	t.Run("AddToClosedProcessor", func(t *testing.T) {
		bp := asyncbatch.NewBatchProcessor[string]()
		bp.Shutdown()
		err := bp.Add("task1")
		if !errors.Is(err, asyncbatch.ErrBatchProcessorClosed) {
			t.Errorf("expected ErrBatchProcessorClosed, got %v", err)
		}
	})

	t.Run("ShutdownWithPendingTasks", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		worker := func(batch []string) {
			mu.Lock()
			batches = append(batches, batch)
			mu.Unlock()
		}
		bp := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](10),
			asyncbatch.WithMaxWait[string](time.Second),
			asyncbatch.WithWorker[string](worker),
		)
		go bp.Run()

		bp.Add("task1")
		bp.Add("task2")
		bp.Shutdown()

		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("expected 1 batch, got %d", len(batches))
		}
		if len(batches) > 0 && len(batches[0]) != 2 {
			t.Errorf("expected batch size 2, got %d", len(batches[0]))
		}
	})

	t.Run("GenericTypeSafety", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]int
		worker := func(batch []int) {
			mu.Lock()
			batches = append(batches, batch)
			mu.Unlock()
		}
		bp := asyncbatch.NewBatchProcessor[int](
			asyncbatch.WithMaxSize[int](2),
			asyncbatch.WithWorker[int](worker),
		)
		go bp.Run()

		bp.Add(1)
		bp.Add(2)
		time.Sleep(100 * time.Millisecond)
		bp.Shutdown()

		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("expected 1 batch, got %d", len(batches))
		}
		if len(batches) > 0 && (len(batches[0]) != 2 || batches[0][0] != 1 || batches[0][1] != 2) {
			t.Errorf("expected batch [1,2], got %v", batches[0])
		}
	})

	t.Run("FullBatchImmediateProcessing", func(t *testing.T) {
		var mu sync.Mutex
		var batches [][]string
		worker := func(batch []string) {
			mu.Lock()
			batches = append(batches, batch)
			mu.Unlock()
		}
		bp := asyncbatch.NewBatchProcessor[string](
			asyncbatch.WithMaxSize[string](2),
			asyncbatch.WithMaxWait[string](500*time.Millisecond),
			asyncbatch.WithWorker[string](worker),
		)
		go bp.Run()

		start := time.Now()
		bp.Add("task1")
		bp.Add("task2")
		time.Sleep(50 * time.Millisecond)
		bp.Shutdown()

		mu.Lock()
		defer mu.Unlock()
		if len(batches) != 1 {
			t.Errorf("expected 1 batch, got %d", len(batches))
		}
		if len(batches) > 0 && len(batches[0]) != 2 {
			t.Errorf("expected batch size 2, got %d", len(batches[0]))
		}
		if duration := time.Since(start); duration > 100*time.Millisecond {
			t.Errorf("full batch processing took too long: %v, expected immediate", duration)
		}
	})
}
