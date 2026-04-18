package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kaichao/gopkg/errors"
)

// RotatedWriter 实现带轮转的日志写入器
type RotatedWriter struct {
	mu           sync.Mutex
	file         *os.File
	filePath     string
	maxSize      int           // MB
	maxAge       int           // days
	maxBackups   int           // count
	currentSize  int64         // 当前文件大小
	currentName  string        // 当前文件名（带时间戳）
	rotationTime time.Duration // 轮转间隔
	lastRotation time.Time     // 上次轮转时间
}

// NewRotatedWriter 创建新的轮转写入器
func NewRotatedWriter(filePath string, maxSize, maxAge, maxBackups int) (*RotatedWriter, error) {
	if maxSize <= 0 {
		maxSize = 100 // 默认100MB
	}
	if maxAge <= 0 {
		maxAge = 7 // 默认7天
	}
	if maxBackups <= 0 {
		maxBackups = 5 // 默认5个备份
	}

	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.WrapE(err, "create log directory", "directory", dir)
	}

	// 生成当前日志文件名
	currentName := fmt.Sprintf("%s.%s", filepath.Base(filePath), time.Now().Format("2006-01-02"))
	fullPath := filepath.Join(dir, currentName)

	// 打开或创建日志文件
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, errors.WrapE(err, "open log file", "filename", fullPath)
	}

	// 获取当前文件大小
	stat, err := file.Stat()
	var currentSize int64
	if err != nil {
		currentSize = 0
	} else {
		currentSize = stat.Size()
	}

	// 清理旧日志文件
	go func() {
		cleanupOldLogs(dir, filepath.Base(filePath), maxAge, maxBackups)
	}()

	return &RotatedWriter{
		file:         file,
		filePath:     filePath,
		maxSize:      maxSize,
		maxAge:       maxAge,
		maxBackups:   maxBackups,
		currentSize:  currentSize,
		currentName:  currentName,
		rotationTime: time.Hour * 24, // 每天轮转一次
		lastRotation: time.Now(),
	}, nil
}

// Write 写入数据到日志文件
func (w *RotatedWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查是否需要轮转
	if err := w.checkRotation(len(p)); err != nil {
		return 0, err
	}

	// 写入数据
	n, err = w.file.Write(p)
	if err == nil {
		w.currentSize += int64(n)
	}

	return n, err
}

// checkRotation 检查是否需要轮转
func (w *RotatedWriter) checkRotation(additionalSize int) error {
	now := time.Now()
	needRotation := false

	// 检查文件大小
	sizeLimit := int64(w.maxSize) * 1024 * 1024
	if w.currentSize+int64(additionalSize) > sizeLimit {
		needRotation = true
	}

	// 检查时间（每天轮转）
	if now.Sub(w.lastRotation) >= w.rotationTime {
		needRotation = true
	}

	if needRotation {
		return w.rotate()
	}

	return nil
}

// rotate 执行日志轮转
func (w *RotatedWriter) rotate() error {
	// 关闭当前文件
	if err := w.file.Close(); err != nil {
		return err
	}

	// 生成新的文件名（使用时间戳）
	now := time.Now()
	oldName := w.currentName
	newName := fmt.Sprintf("%s.%s", filepath.Base(w.filePath), now.Format("2006-01-02"))
	oldFullPath := filepath.Join(filepath.Dir(w.filePath), oldName)
	newFullPath := filepath.Join(filepath.Dir(w.filePath), newName)

	// 如果旧文件存在且不是当前文件，进行轮转
	if oldName != newName {
		if _, err := os.Stat(oldFullPath); err == nil {
			// 对旧文件进行压缩或重命名
			backupName := fmt.Sprintf("%s.%d", oldName, now.Unix())
			backupPath := filepath.Join(filepath.Dir(w.filePath), backupName)
			if err := os.Rename(oldFullPath, backupPath); err != nil {
				// 如果重命名失败，继续
			}
		}
	}

	// 创建新文件
	file, err := os.OpenFile(newFullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	w.file = file
	w.currentName = newName
	w.currentSize = 0
	w.lastRotation = now

	// 清理旧日志文件
	go func() {
		w.cleanupOldLogs()
	}()

	return nil
}

// cleanupOldLogs 清理旧日志文件
func (w *RotatedWriter) cleanupOldLogs() {
	dir := filepath.Dir(w.filePath)
	prefix := filepath.Base(w.filePath)

	cleanupOldLogs(dir, prefix, w.maxAge, w.maxBackups)
}

// // cleanupOldLogs 清理指定目录下的旧日志
func cleanupOldLogs(dir, prefix string, maxAge, maxBackups int) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var logFiles []struct {
		name    string
		modTime time.Time
	}

	for _, file := range files {
		if !file.IsDir() && file.Name() != prefix && strings.HasPrefix(file.Name(), prefix) {
			if info, err := file.Info(); err == nil {
				logFiles = append(logFiles, struct {
					name    string
					modTime time.Time
				}{file.Name(), info.ModTime()})
			}
		}
	}

	// 按时间排序（最新的在前）
	for i := 0; i < len(logFiles); i++ {
		for j := i + 1; j < len(logFiles); j++ {
			if logFiles[i].modTime.Before(logFiles[j].modTime) {
				logFiles[i], logFiles[j] = logFiles[j], logFiles[i]
			}
		}
	}

	// 删除超过数量的旧文件
	if len(logFiles) > maxBackups {
		for i := 0; i < len(logFiles)-maxBackups; i++ {
			filePath := filepath.Join(dir, logFiles[i].name)
			os.Remove(filePath)
		}
	}

	// 删除超过天数的旧文件
	cutoffTime := time.Now().AddDate(0, 0, -maxAge)
	for _, file := range logFiles {
		if file.modTime.Before(cutoffTime) {
			filePath := filepath.Join(dir, file.name)
			os.Remove(filePath)
		}
	}
}

// Close 关闭日志文件
func (w *RotatedWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Close()
}

// Sync 同步文件内容到磁盘
func (w *RotatedWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Sync()
}

// Stat 获取当前文件状态
func (w *RotatedWriter) Stat() (os.FileInfo, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Stat()
}
