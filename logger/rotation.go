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

// RotatedWriter implements rotating log writer
type RotatedWriter struct {
	mu           sync.Mutex
	file         *os.File
	filePath     string
	maxSize      int           // MB
	maxAge       int           // days
	maxBackups   int           // count
	currentSize  int64         // current file size
	currentName  string        // current filename (with timestamp)
	rotationTime time.Duration // rotation interval
	lastRotation time.Time     // last rotation time
}

// NewRotatedWriter creates new rotated writer
func NewRotatedWriter(filePath string, maxSize, maxAge, maxBackups int) (*RotatedWriter, error) {
	if maxSize <= 0 {
		maxSize = 100 // default 100MB
	}
	if maxAge <= 0 {
		maxAge = 7 // default 7 days
	}
	if maxBackups <= 0 {
		maxBackups = 5 // default 5 backups
	}

	// ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.WrapE(err, "create log directory", "directory", dir)
	}

	// generate current log filename
	currentName := fmt.Sprintf("%s.%s", filepath.Base(filePath), time.Now().Format("2006-01-02"))
	fullPath := filepath.Join(dir, currentName)

	// open or create log file
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, errors.WrapE(err, "open log file", "filename", fullPath)
	}

	// get current file size
	stat, err := file.Stat()
	var currentSize int64
	if err != nil {
		currentSize = 0
	} else {
		currentSize = stat.Size()
	}

	// clean up old log files
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
		rotationTime: time.Hour * 24, // rotate once per day
		lastRotation: time.Now(),
	}, nil
}

// Write writes data to log file
func (w *RotatedWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// check if rotation is needed
	if err := w.checkRotation(len(p)); err != nil {
		return 0, err
	}

	// write data
	n, err = w.file.Write(p)
	if err == nil {
		w.currentSize += int64(n)
	}

	return n, err
}

// checkRotation checks if rotation is needed
func (w *RotatedWriter) checkRotation(additionalSize int) error {
	now := time.Now()
	needRotation := false

	// check file size
	sizeLimit := int64(w.maxSize) * 1024 * 1024
	if w.currentSize+int64(additionalSize) > sizeLimit {
		needRotation = true
	}

	// check time (rotate once per day)
	if now.Sub(w.lastRotation) >= w.rotationTime {
		needRotation = true
	}

	if needRotation {
		return w.rotate()
	}

	return nil
}

// rotate performs log rotation
func (w *RotatedWriter) rotate() error {
	// close current file
	if err := w.file.Close(); err != nil {
		return err
	}

	// generate new filename (with timestamp)
	now := time.Now()
	oldName := w.currentName
	newName := fmt.Sprintf("%s.%s", filepath.Base(w.filePath), now.Format("2006-01-02"))
	oldFullPath := filepath.Join(filepath.Dir(w.filePath), oldName)
	newFullPath := filepath.Join(filepath.Dir(w.filePath), newName)

	// if old file exists and is not current file, perform rotation
	if oldName != newName {
		if _, err := os.Stat(oldFullPath); err == nil {
			// compress or rename old file
			backupName := fmt.Sprintf("%s.%d", oldName, now.Unix())
			backupPath := filepath.Join(filepath.Dir(w.filePath), backupName)
			if err := os.Rename(oldFullPath, backupPath); err != nil {
				// if rename fails, continue
			}
		}
	}

	// create new file
	file, err := os.OpenFile(newFullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	w.file = file
	w.currentName = newName
	w.currentSize = 0
	w.lastRotation = now

	// clean up old log files
	go func() {
		w.cleanupOldLogs()
	}()

	return nil
}

// cleanupOldLogs cleans up old log files
func (w *RotatedWriter) cleanupOldLogs() {
	dir := filepath.Dir(w.filePath)
	prefix := filepath.Base(w.filePath)

	cleanupOldLogs(dir, prefix, w.maxAge, w.maxBackups)
}

// cleanupOldLogs cleans up old log files in specified directory
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

	// sort by time (newest first)
	for i := 0; i < len(logFiles); i++ {
		for j := i + 1; j < len(logFiles); j++ {
			if logFiles[i].modTime.Before(logFiles[j].modTime) {
				logFiles[i], logFiles[j] = logFiles[j], logFiles[i]
			}
		}
	}

	// delete old files exceeding count limit
	if len(logFiles) > maxBackups {
		for i := 0; i < len(logFiles)-maxBackups; i++ {
			filePath := filepath.Join(dir, logFiles[i].name)
			os.Remove(filePath)
		}
	}

	// delete old files exceeding age limit
	cutoffTime := time.Now().AddDate(0, 0, -maxAge)
	for _, file := range logFiles {
		if file.modTime.Before(cutoffTime) {
			filePath := filepath.Join(dir, file.name)
			os.Remove(filePath)
		}
	}
}

// Close closes log file
func (w *RotatedWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Close()
}

// Sync syncs file content to disk
func (w *RotatedWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Sync()
}

// Stat gets current file status
func (w *RotatedWriter) Stat() (os.FileInfo, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Stat()
}
