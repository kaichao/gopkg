package logger

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kaichao/gopkg/errors"
	"github.com/sirupsen/logrus"
)

// AsyncWriter 实现异步日志写入器
type AsyncWriter struct {
	underlying logrus.Formatter
	out        io.Writer
	ch         chan *logrus.Entry
	wg         sync.WaitGroup
	quit       chan struct{}
	started    uint32 // 0 = not started, 1 = started
	queueSize  int
	batchSize  int
}

// NewAsyncWriter 创建异步写入器
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

// SetFormatter 设置日志格式
func (w *AsyncWriter) SetFormatter(formatter logrus.Formatter) {
	w.underlying = formatter
}

// WriteEntry 异步写入日志条目
func (w *AsyncWriter) WriteEntry(entry *logrus.Entry) error {
	// 如果未启动，直接写入
	if atomic.LoadUint32(&w.started) == 0 {
		w.syncWrite(entry)
		return nil
	}

	// 异步写入，如果通道已满则降级为同步
	select {
	case w.ch <- entry:
		return nil
	default:
		// 通道已满，降级为同步写入
		w.syncWrite(entry)
		return nil
	}
}

// Write 实现 io.Writer 接口，用于兼容 logrus.Logger.SetOutput
func (w *AsyncWriter) Write(p []byte) (n int, err error) {
	// 对于原始字节写入，直接同步写入
	if w.underlying == nil {
		return len(p), nil
	}

	// 创建一个临时 entry 来格式化输出
	entry := &logrus.Entry{
		Logger: &logrus.Logger{
			Formatter: w.underlying,
		},
		Message: string(p),
		Level:   logrus.InfoLevel,
	}
	w.syncWrite(entry)
	return len(p), nil
}

// syncWrite 同步写入
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

// Start 启动异步写入协程
func (w *AsyncWriter) Start() {
	if !atomic.CompareAndSwapUint32(&w.started, 0, 1) {
		return // 已经启动
	}

	w.wg.Add(1)
	go w.run()
}

// run 异步写入主循环
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
				batch = batch[:0] // 重置 batch
			}
		case <-ticker.C:
			// 定时刷新，避免长时间等待
			if len(batch) > 0 {
				w.flush(batch)
				batch = batch[:0]
			}
		case <-w.quit:
			// 退出前刷新剩余日志
			if len(batch) > 0 {
				w.flush(batch)
			}

			// 清空通道中的所有剩余条目
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

// flush 批量刷新日志
func (w *AsyncWriter) flush(entries []*logrus.Entry) {
	if w.underlying == nil || len(entries) == 0 {
		return
	}

	for _, entry := range entries {
		serialized, err := w.underlying.Format(entry)
		if err != nil {
			continue // 跳过格式化失败的日志
		}

		w.out.Write(serialized)
	}
}

// Stop 停止异步写入
func (w *AsyncWriter) Stop() {
	if atomic.LoadUint32(&w.started) == 0 {
		return // 未启动
	}

	close(w.quit)
	w.wg.Wait()
}

// Wait 等待所有日志写入完成
func (w *AsyncWriter) Wait(timeout time.Duration) error {
	if atomic.LoadUint32(&w.started) == 0 {
		return nil
	}

	// 创建一个超时通道
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

// Buffered 返回当前缓冲的日志数量
func (w *AsyncWriter) Buffered() int {
	return len(w.ch)
}

// Cap 返回通道容量
func (w *AsyncWriter) Cap() int {
	return cap(w.ch)
}

// Close 关闭异步写入器
func (w *AsyncWriter) Close() error {
	w.Stop()
	return nil
}

// WriteString 实现 io.StringWriter 接口
func (w *AsyncWriter) WriteString(s string) (int, error) {
	entry := &logrus.Entry{
		Logger: &logrus.Logger{
			Formatter: w.underlying,
		},
		Message: s,
		Level:   logrus.InfoLevel,
	}
	w.WriteEntry(entry)
	return len(s), nil
}

// Sync 同步日志到磁盘
func (w *AsyncWriter) Sync() error {
	if syncer, ok := w.out.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}

// AsyncWriterAdapter 适配 AsyncWriter 到 io.Writer 接口
type AsyncWriterAdapter struct {
	*AsyncWriter
}

// Write 实现 io.Writer 接口，用于兼容 logrus.Logger.SetOutput
func (a *AsyncWriterAdapter) Write(p []byte) (n int, err error) {
	if a.AsyncWriter == nil || a.AsyncWriter.underlying == nil {
		return len(p), nil
	}

	// 创建一个临时 entry 来格式化输出
	entry := &logrus.Entry{
		Logger: &logrus.Logger{
			Formatter: a.AsyncWriter.underlying,
		},
		Message: string(p),
		Level:   logrus.InfoLevel,
	}
	a.AsyncWriter.WriteEntry(entry)
	return len(p), nil
}

// Sync 同步日志到磁盘
func (a *AsyncWriterAdapter) Sync() error {
	if a.AsyncWriter == nil {
		return nil
	}
	return a.AsyncWriter.Sync()
}
