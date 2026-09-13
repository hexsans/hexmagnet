package logging

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/servercfg"
)

const (
	defaultBaseName = "hexmagnet"
	timeFormat      = "2006-01-02"
	bytesPerMB      = 1024 * 1024
)

func newFileRotator(
	config servercfg.FileRotatorConfig,
) *fileRotator {
	var maxSizeBytes int64
	if config.MaxSizeMB > 0 {
		maxSizeBytes = int64(config.MaxSizeMB) * bytesPerMB
	}

	return &fileRotator{
		path:         config.Path,
		baseName:     defaultBaseName,
		maxBackups:   config.MaxBackups,
		maxSizeBytes: maxSizeBytes,
	}
}

type fileRotator struct {
	lock         sync.Mutex
	path         string
	pathCreated  bool
	baseName     string
	maxBackups   int
	maxSizeBytes int64
	fileDate     string
	filePath     string
	file         *fileRotatorFile
	closed       bool
}

func (r *fileRotator) Write(output []byte) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.closed {
		return len(output), nil
	}

	if !r.pathCreated {
		err := os.MkdirAll(r.path, 0o755)
		if err != nil {
			return 0, err
		}

		r.pathCreated = true
	}

	if err := r.checkRotate(len(output)); err != nil {
		return 0, err
	}

	return r.file.Write(output)
}

// Sync flushes buffered log data to disk without closing the active file, so
// the rotator stays writable after a Sync.
func (r *fileRotator) Sync() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.file == nil {
		return nil
	}

	return r.file.Flush()
}

func (r *fileRotator) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.closed = true
	if r.file == nil {
		return nil
	}

	return r.file.Close()
}

func (r *fileRotator) checkRotate(nextLen int) error {
	if !r.shouldRotate(nextLen) {
		return nil
	}

	return r.rotate()
}

func (r *fileRotator) shouldRotate(nextLen int) bool {
	if r.file == nil {
		return true
	}

	if time.Now().Format(timeFormat) != r.fileDate {
		return true
	}

	// Rotate before the pending write so file sizes stay close to the
	// configured cap. Oversized single entries still get their own file.
	return r.maxSizeBytes > 0 &&
		r.file.size > 0 &&
		r.file.size+int64(nextLen) > r.maxSizeBytes
}

func (r *fileRotator) rotate() error {
	if r.file != nil {
		err := r.file.Close()
		r.file = nil

		if err != nil {
			return err
		}
	}

	now := time.Now()
	sameDay := r.fileDate == now.Format(timeFormat)

	filePath, err := r.newFilePath(now, sameDay && r.maxSizeBytes > 0)
	if err != nil {
		return err
	}

	fp, err := newFileRotatorFile(filePath)
	if err != nil {
		return err
	}

	r.file = fp
	r.filePath = filePath
	r.fileDate = now.Format(timeFormat)

	return r.pruneBackups(now)
}

// newFilePath returns the active log file path. The first file of a day uses
// the date-only name; same-day rotations caused by the size cap append a
// sequence number so files stay sortable and unique.
func (r *fileRotator) newFilePath(now time.Time, sequenced bool) (string, error) {
	date := now.Format(timeFormat)

	if !sequenced {
		return r.resolveActiveFilePath(date)
	}

	seq, err := r.nextSequence(date)
	if err != nil {
		return "", err
	}

	return path.Join(r.path, fmt.Sprintf("%s.%s.%d.log", r.baseName, date, seq)), nil
}

// resolveActiveFilePath returns the file to append to when the rotator opens
// for the first time or the day changes: the newest existing file for the
// date if it is still within the size cap, otherwise the next sequence.
// Without this, every restart would start from the date-only file and rotate
// away from it, leaving a trail of near-empty files.
func (r *fileRotator) resolveActiveFilePath(date string) (string, error) {
	entries, err := os.ReadDir(r.path)
	if err != nil {
		return "", err
	}

	newest := ""
	newestSeq := 0
	newestSize := int64(0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileDate, seq, ok := parseBackupName(r.baseName, entry.Name())
		if !ok || fileDate != date {
			continue
		}

		if newest != "" && seq <= newestSeq {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		newest = entry.Name()
		newestSeq = seq
		newestSize = info.Size()
	}

	if newest == "" {
		return path.Join(r.path, fmt.Sprintf("%s.%s.log", r.baseName, date)), nil
	}

	if r.maxSizeBytes > 0 && newestSize >= r.maxSizeBytes {
		return path.Join(r.path, fmt.Sprintf("%s.%s.%d.log", r.baseName, date, newestSeq+1)), nil
	}

	return path.Join(r.path, newest), nil
}

func (r *fileRotator) nextSequence(date string) (int, error) {
	entries, err := os.ReadDir(r.path)
	if err != nil {
		return 0, err
	}

	maxSeq := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileDate, seq, ok := parseBackupName(r.baseName, entry.Name())
		if !ok || fileDate != date {
			continue
		}

		if seq > maxSeq {
			maxSeq = seq
		}
	}

	return maxSeq + 1, nil
}

// pruneBackups keeps the newest maxBackups rotated files and never removes
// the file currently being written. A maxBackups of 0 disables pruning.
func (r *fileRotator) pruneBackups(_ time.Time) error {
	if r.maxBackups <= 0 {
		return nil
	}

	entries, err := os.ReadDir(r.path)
	if err != nil {
		return err
	}

	activeName := path.Base(r.filePath)

	var backups []backupFile

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if name == activeName {
			continue
		}

		date, seq, ok := parseBackupName(r.baseName, name)
		if !ok {
			continue
		}

		backups = append(backups, backupFile{name: name, date: date, seq: seq})
	}

	if len(backups) <= r.maxBackups {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		if backups[i].date != backups[j].date {
			return backups[i].date < backups[j].date
		}

		return backups[i].seq < backups[j].seq
	})

	for _, backup := range backups[:len(backups)-r.maxBackups] {
		// make a best effort to remove the file
		_ = os.Remove(path.Join(r.path, backup.name))
	}

	return nil
}

type backupFile struct {
	name string
	date string
	seq  int
}

// parseBackupName accepts both hexmagnet.YYYY-MM-DD.log and
// hexmagnet.YYYY-MM-DD.N.log rotated file names.
func parseBackupName(baseName, name string) (string, int, bool) {
	if !strings.HasPrefix(name, baseName+".") || !strings.HasSuffix(name, ".log") {
		return "", 0, false
	}

	rest := name[len(baseName)+1 : len(name)-len(".log")]

	date := rest
	seq := 0

	if idx := strings.LastIndex(rest, "."); idx >= 0 {
		parsed, err := strconv.Atoi(rest[idx+1:])
		if err != nil || parsed <= 0 {
			return "", 0, false
		}

		date = rest[:idx]
		seq = parsed
	}

	if _, err := time.Parse(timeFormat, date); err != nil {
		return "", 0, false
	}

	return date, seq, true
}

func newFileRotatorFile(path string) (*fileRotatorFile, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		_ = f.Close()

		return nil, err
	}

	return &fileRotatorFile{
		writer: bufio.NewWriterSize(f, 1000),
		file:   f,
		size:   info.Size(),
	}, nil
}

type fileRotatorFile struct {
	writer *bufio.Writer
	file   *os.File
	size   int64
}

func (f *fileRotatorFile) Write(p []byte) (int, error) {
	n, err := f.writer.Write(p)
	f.size += int64(n)

	return n, err
}

func (f *fileRotatorFile) Flush() error {
	return f.writer.Flush()
}

func (f *fileRotatorFile) Close() error {
	if err := f.Flush(); err != nil {
		return err
	}

	return f.file.Close()
}
