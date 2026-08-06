package logging

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/servercfg"
)

const defaultBaseName = "hexmagnet"

func newFileRotator(
	config servercfg.FileRotatorConfig,
) *fileRotator {
	return &fileRotator{
		path:       config.Path,
		baseName:   defaultBaseName,
		maxBackups: config.MaxBackups,
	}
}

type fileRotator struct {
	lock        sync.Mutex
	path        string
	pathCreated bool
	baseName    string
	maxBackups  int
	fileDate    string
	file        *fileRotatorFile
	closed      bool
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

	if err := r.checkRotate(); err != nil {
		return 0, err
	}

	return r.file.Write(output)
}

func (r *fileRotator) Sync() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.file == nil {
		return nil
	}

	return r.file.Close()
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

func (r *fileRotator) checkRotate() error {
	if !r.shouldRotate() {
		return nil
	}

	return r.rotate()
}

func (r *fileRotator) shouldRotate() bool {
	if r.file == nil {
		return true
	}

	return time.Now().Format(timeFormat) != r.fileDate
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

	fp, err := newFileRotatorFile(r.newFilePath(now))
	if err != nil {
		return err
	}

	r.file = fp
	r.fileDate = now.Format(timeFormat)

	return r.pruneBackups(now)
}

const timeFormat = "2006-01-02"

func (r *fileRotator) newFilePath(now time.Time) string {
	return path.Join(r.path, fmt.Sprintf("%s.%s.log", r.baseName, now.Format(timeFormat)))
}

func (r *fileRotator) pruneBackups(now time.Time) error {
	files, err := os.ReadDir(r.path)
	if err != nil {
		return err
	}

	var backupFiles []string

	strNow := now.Format(timeFormat)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasPrefix(name, r.baseName+".") || !strings.HasSuffix(name, ".log") {
			continue
		}

		strDate := name[len(r.baseName)+1 : len(name)-4]

		_, parseErr := time.Parse(timeFormat, strDate)
		if parseErr != nil {
			continue
		}

		if strDate >= strNow {
			continue
		}

		backupFiles = append(backupFiles, name)
	}

	if len(backupFiles) <= r.maxBackups {
		return nil
	}

	for _, name := range backupFiles[:len(backupFiles)-r.maxBackups] {
		// make a best effort to remove the file
		_ = os.Remove(path.Join(r.path, name))
	}

	return nil
}

func newFileRotatorFile(path string) (*fileRotatorFile, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	return &fileRotatorFile{
		writer: bufio.NewWriterSize(f, 1000),
		file:   f,
	}, nil
}

type fileRotatorFile struct {
	writer *bufio.Writer
	file   *os.File
}

func (f *fileRotatorFile) Write(p []byte) (int, error) {
	return f.writer.Write(p)
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
