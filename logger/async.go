package logger

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kaichao/gopkg/errors"
	"github.com/sirupsen/logrus"
)

// AsyncWriter implements asynchronous log writer
type AsyncWriter struct {
	underlying logrus.Formatter
	out        io.Writer
	ch         chan *logrus.Entry
	wg         sync.WaitGroup
	quit       chan struct{}
	started    uint32 // 0 = not started, 1 = started (English comment already)
	queueSize  int
	batchSize  int
}

// NewAsyncWriter creates async writer
func NewAsyncWriter(out io.Writer, bufferSize, batchSize int) *AsyncWriter {
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	if batchSize <= 0 {
		batchSize = 100
	}

	return &AsyncWriter{
		out:       out,
		ch:        make(chan *logrus.Entry, bufferSize),
		quit:      make(chan struct{}),
		queueSize: bufferSize,
		batchSize: batchSize,
	}
}

// SetFormatter sets log format
func (w *AsyncWriter) SetFormatter(formatter logrus.Formatter) {
	w.underlying = formatter
}

// WriteEntry asynchronously writes log entry
func (w *AsyncWriter) WriteEntry(entry *logrus.Entry) error {
	// if not started, write synchronously
	if atomic.LoadUint32(&w.started) == 0 {
		w.syncWrite(entry)
		return nil
	}

	// async write, degrade to sync if channel is full
	select {
	case w.ch <- entry:
		return nil
	default:
		// channel full, degrade to sync write
		w.syncWrite(entry)
		return nil
	}
}

// Write implements io.Writer interface for compatibility with logrus.Logger.SetOutput
func (w *AsyncWriter) Write(p []byte) (n int, err error) {
	// for raw byte writes, write synchronously directly
	if w.underlying == nil {
		return len(p), nil
	}

	// create a temporary entry to route through WriteEntry
	entry := newLogEntry(w.underlying, string(p))
	w.WriteEntry(entry)
	return len(p), nil
}

// syncWrite synchronously writes
func (w *AsyncWriter) syncWrite(entry *logrus.Entry) error {
	if w.underlying == nil {
		return nil
	}

	serialized, err := w.underlying.Format(entry)
	if err != nil {
		return err
	}

	_, err = w.out.Write(serialized)
	return err
}

// Start starts async writer goroutine
func (w *AsyncWriter) Start() {
	if !atomic.CompareAndSwapUint32(&w.started, 0, 1) {
		return // already started
	}

	w.wg.Add(1)
	go w.run()
}

// run is the main async write loop
func (w *AsyncWriter) run() {
	defer w.wg.Done()

	var batch []*logrus.Entry
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case entry := <-w.ch:
			batch = append(batch, entry)
			if len(batch) >= w.batchSize {
				w.flush(batch)
				batch = batch[:0] // reset batch
			}
		case <-ticker.C:
			// periodic flush to avoid long waits
			if len(batch) > 0 {
				w.flush(batch)
				batch = batch[:0]
			}
		case <-w.quit:
			// flush remaining logs before exit
			if len(batch) > 0 {
				w.flush(batch)
			}

			// drain all remaining entries in channel
			for {
				select {
				case entry := <-w.ch:
					w.syncWrite(entry)
				default:
					return
				}
			}
		}
	}
}

// flush batch flushes logs
func (w *AsyncWriter) flush(entries []*logrus.Entry) {
	if w.underlying == nil || len(entries) == 0 {
		return
	}

	for _, entry := range entries {
		serialized, err := w.underlying.Format(entry)
		if err != nil {
			continue // skip logs that fail formatting
		}

		w.out.Write(serialized)
	}
}

// Stop stops async writer
func (w *AsyncWriter) Stop() {
	if atomic.LoadUint32(&w.started) == 0 {
		return // not started
	}

	close(w.quit)
	w.wg.Wait()
}

// Wait waits for all logs to be written
func (w *AsyncWriter) Wait(timeout time.Duration) error {
	if atomic.LoadUint32(&w.started) == 0 {
		return nil
	}

	// create a timeout channel
	timeoutCh := time.After(timeout)
	done := make(chan struct{})

	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-timeoutCh:
		return errors.New("wait timeout")
	}
}

// Buffered returns current buffered log count
func (w *AsyncWriter) Buffered() int {
	return len(w.ch)
}

// Cap returns channel capacity
func (w *AsyncWriter) Cap() int {
	return cap(w.ch)
}

// newLogEntry creates a logrus.Entry with the given formatter and message
func newLogEntry(formatter logrus.Formatter, msg string) *logrus.Entry {
	return &logrus.Entry{
		Logger: &logrus.Logger{
			Formatter: formatter,
		},
		Message: msg,
		Level:   logrus.InfoLevel,
	}
}

// Close closes async writer
func (w *AsyncWriter) Close() error {
	w.Stop()
	return nil
}

// WriteString implements io.StringWriter interface
func (w *AsyncWriter) WriteString(s string) (int, error) {
	entry := newLogEntry(w.underlying, s)
	w.WriteEntry(entry)
	return len(s), nil
}

// Sync syncs logs to disk
func (w *AsyncWriter) Sync() error {
	if syncer, ok := w.out.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}

// AsyncWriterAdapter adapts AsyncWriter to io.Writer interface
type AsyncWriterAdapter struct {
	*AsyncWriter
}

// Write implements io.Writer interface for compatibility with logrus.Logger.SetOutput
func (a *AsyncWriterAdapter) Write(p []byte) (n int, err error) {
	if a.AsyncWriter == nil || a.AsyncWriter.underlying == nil {
		return len(p), nil
	}

	entry := newLogEntry(a.AsyncWriter.underlying, string(p))
	a.AsyncWriter.WriteEntry(entry)
	return len(p), nil
}

// Sync syncs logs to disk
func (a *AsyncWriterAdapter) Sync() error {
	if a.AsyncWriter == nil {
		return nil
	}
	return a.AsyncWriter.Sync()
}
