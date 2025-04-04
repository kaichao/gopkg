package asyncbatch_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kaichao/gopkg/asyncbatch"
)

// Unit test
func TestAsyncBatch(t *testing.T) {
	var mu sync.Mutex
	var results [][]string

	processFunc := func(batch []string) {
		mu.Lock()
		results = append(results, batch)
		mu.Unlock()
	}

	batchSize := 5
	queueSize := 10
	flushInterval := 2 * time.Second
	processor := asyncbatch.New(batchSize, queueSize, flushInterval, processFunc)

	// Submit 12 items to the asynchronous queue
	for i := 1; i <= 12; i++ {
		processor.Submit(fmt.Sprintf("Record-%d", i))
		time.Sleep(200 * time.Millisecond) // Simulate interval between submissions
	}

	// Wait for all batches to be processed
	time.Sleep(5 * time.Second)
	processor.Stop()

	mu.Lock()
	defer mu.Unlock()

	// Verify number of processed batches
	expectedBatches := (12 + batchSize - 1) / batchSize // Expected number of batches
	if len(results) != expectedBatches {
		t.Errorf("Expected %d batches, but got %d", expectedBatches, len(results))
	}
}
