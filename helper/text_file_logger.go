package helper

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/AtlanCI/zlog"
)

var (
	_forceDebugLevelBefore time.Time
	_forceEnableQuickFlag  = false
)

func init() {
	_forceDebugLevelBefore = time.Now().Add(time.Second * 15)
	_forceEnableQuickFlag = true
}

type textFile struct {
	level zlog.Level
	rfile *rotatedFile
}

func NewTextFileLogger(fileName string) (zlog.Logger, error) {
	f, err := newRotatedFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("newRotatedFile(): %w", err)
	}

	return &textFile{
		level: zlog.LevelDebug,
		rfile: f,
	}, nil
}

// Log Time Level TraceID Caller Message
func (l *textFile) Log(t time.Time, lv zlog.Level, tid string, c *zlog.Caller, format string, v ...interface{}) {
	buffer := l.rfile.GetBuffer()
	buffer.Reset()

	y, m, d := t.Date()
	hh, mm, ss := t.Clock()
	//[2021-03-17 19:25:50][1615980441.370]
	buffer.WriteString(fmt.Sprintf("[%04d-%02d-%02d %02d:%02d:%02d][%d.%03d]", y, m, d, hh, mm, ss, t.Unix(), t.Nanosecond()/int(time.Millisecond)))

	//[tid]
	if len(tid) > 0 {
		buffer.WriteByte('[')
		buffer.WriteString(tid)
		buffer.WriteByte(']')
	}

	//[main.go:78]
	if c != nil {
		buffer.WriteByte('[')
		buffer.WriteString(c.File)
		buffer.WriteByte(':')
		buffer.WriteString(strconv.Itoa(c.Line))
		buffer.WriteByte(']')
	}

	//[E]
	buffer.WriteByte('[')
	buffer.WriteByte(lv.StringShort())
	buffer.WriteByte(']')

	buffer.WriteByte(' ')

	//msg
	buffer.WriteString(fmt.Sprintf(format, v...))

	//\n
	if format[len(format)-1] != '\n' {
		buffer.WriteByte('\n')
	}

	l.rfile.PutBuffer(buffer)
}

// SetLevel set globe log level
func (l *textFile) SetLevel(lv zlog.Level) {
	l.level = lv
}

// GetLevel get globe log level
func (l *textFile) GetLevel() zlog.Level {
	// _forceDebugLevelBefore Return to log level debug before expiration
	// Prevent certain logs from being ignored on startup ex: wrap sarama
	if _forceEnableQuickFlag {
		if time.Now().Before(_forceDebugLevelBefore) {
			return zlog.LevelDebug
		} else {
			_forceEnableQuickFlag = false
		}
	}

	return l.level
}

// Close file safely
func (l *textFile) Close() {
	l.rfile.Close()
}

// ==============rotatedFile=============

type ByteSize int64

const (
	B  ByteSize = 1
	KB          = B << 10
	MB          = B << 20
	GB          = B << 30
	TB          = B << 40
	PB          = B << 50
	EB          = B << 60
	//ZB          = B << 70
)

type rotatedFile struct {
	rawFileName string

	checkTimer *time.Ticker

	rotatedTime time.Duration
	rotatedSize ByteSize
	bufferPool  BytesBufferPool

	currFileCreate  time.Time
	currentFile     *os.File
	currentFileLock sync.RWMutex

	queue   chan *bytes.Buffer
	done    chan struct{}
	writeWg sync.WaitGroup
}

func newRotatedFile(fileName string) (*rotatedFile, error) {
	const (
		checkInterval = time.Second * 30
		rotatedTime   = time.Hour * 24
		rotatedSize   = MB * 300
		queueSize     = 128
		writeRoutine  = 1
	)

	rf := &rotatedFile{
		rawFileName: fileName,
		checkTimer:  time.NewTicker(checkInterval),
		rotatedTime: rotatedTime,
		rotatedSize: rotatedSize,
		bufferPool:  newBufferPool(16 << 10),
		queue:       make(chan *bytes.Buffer, writeRoutine*queueSize),
		done:        make(chan struct{}),
		writeWg:     sync.WaitGroup{},
	}

	// run file rotated goroutine
	err := rf.rotatedFile()
	if err != nil {
		return nil, fmt.Errorf("rotatedFile(): %w", err)
	}

	go rf.rotatedRun()

	for i := 0; i < writeRoutine; i++ {
		rf.writeWg.Add(1)
		go func() {
			defer rf.writeWg.Done()
			rf.writeRun()
		}()
	}

	return rf, nil
}

func (r *rotatedFile) GetBuffer() *bytes.Buffer {
	return r.bufferPool.Get()
}

func (r *rotatedFile) PutBuffer(buf *bytes.Buffer) {
	select {
	case r.queue <- buf:
	default:
	}
}

func (r *rotatedFile) Close() {
	close(r.done)

	r.checkTimer.Stop()
	r.writeWg.Wait()
	r.currentFileLock.Lock()
	_ = r.currentFile.Close()
	r.currentFileLock.Unlock()
}

func (r *rotatedFile) rotatedRun() {
	last := time.Now()
	for {
		select {
		case now := <-r.checkTimer.C:
			if now.Sub(last) >= r.rotatedTime {
				// need rotate for file create time
				err := r.rotatedFile()
				if err != nil {
					fmt.Printf("rotatedFile rotatedRun rotatedFile(time) failed. err=%+v\n", err)
				}
				last = now
			} else {
				if info, err := r.currentFile.Stat(); err == nil && info.Size() >= int64(r.rotatedSize) {
					// need rotate for file size
					err := r.rotatedFile()
					if err != nil {
						fmt.Printf("rotatedFile rotatedRun rotatedFile(size) failed. err=%+v\n", err)
					}
					last = now
				}
			}
		case <-r.done:
			return
		}
	}
}

func (r *rotatedFile) rotatedFile() error {
	err := os.MkdirAll(dir(r.rawFileName), os.ModePerm)
	if err != nil {
		return fmt.Errorf("zlog: create log dir: %s: err=%w", path.Dir(r.rawFileName), err)
	}

	//var fileName string
	//if strings.HasPrefix(r.rawFileName, "./") {
	//	fileName = r.rawFileName[2:]
	//}

	prefix, extension := splitPath(r.rawFileName)

	now := time.Now()

	// example: /var/mysql_20060102_150405
	baseName := fmt.Sprintf("%s_%s", prefix, now.Format("20060102_150405"))

	var newFile *os.File

	var newName string

	for i := 0; true; i++ {
		newName = baseName
		if i != 0 {
			newName = newName + fmt.Sprintf("_%02d", i) // _01
		}

		newName += extension //.log

		newFile, err = os.OpenFile(newName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			if !os.IsExist(err) {
				return fmt.Errorf("zlog: create log file: %s: err=%w", newName, err)
			}
			continue
		}
		break
	}

	var old *os.File
	r.currentFileLock.Lock()
	old = r.currentFile
	r.currentFile = newFile
	r.currentFileLock.Unlock()

	// create link for log file
	tmpLinkName := prefix + `_symlink`
	if err = os.Symlink(newName, tmpLinkName); err != nil {
		zlog.Errorf(context.Background(), "zlog: create new symlink: %s to %s: err=%=v", newName, tmpLinkName, err)
	} else {
		if err = os.Rename(tmpLinkName, prefix); err != nil {
			zlog.Errorf(context.Background(), "zlog: rename symlink: %s to %s: err=%=v", tmpLinkName, prefix, err)
		}
	}

	if old != nil {
		_, _ = old.WriteString(fmt.Sprintf("==============Rotate to %s==============", newName))
		_ = old.Close()
	}
	return nil
}

func (r *rotatedFile) writeRun() {
WaitExit:
	for {
		select {
		case <-r.done:
			break WaitExit
		case buf := <-r.queue:
			r.write(buf)
		}
	}

	// Try to ensure that the log output is complete
	atr := time.NewTicker(time.Millisecond * 400)
	defer atr.Stop()
	for goon := true; goon; {
		tr := time.NewTicker(time.Millisecond * 50)
		select {
		case buf := <-r.queue:
			r.write(buf)
		case <-tr.C:
			goon = false
		case <-atr.C:
			goon = false
		}
		tr.Stop()
	}
}

func (r *rotatedFile) write(buf *bytes.Buffer) {
	r.currentFileLock.RLock()
	_, err := buf.WriteTo(r.currentFile)
	if err != nil {
		fmt.Printf("zlog: failed to write file. fileName=%s,err=%+v", r.currentFile.Name(), err)
	}
	r.currentFileLock.RUnlock()

	r.bufferPool.Put(buf)
}

type BytesBufferPool interface {
	Get() *bytes.Buffer
	Put(*bytes.Buffer)
}

func newBufferPool(baseByte int) BytesBufferPool {
	return &bytesBufferPool{
		pool: sync.Pool{
			New: syncPoolNewFunc(baseByte),
		},
	}
}

func syncPoolNewFunc(baseByte int) func() interface{} {
	if baseByte <= 0 {
		baseByte = 16 << 10
	}
	return func() interface{} {
		// 16K buffer in memory
		return bytes.NewBuffer(make([]byte, 0, baseByte))
	}
}

type bytesBufferPool struct {
	pool sync.Pool
}

func (p *bytesBufferPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

func (p *bytesBufferPool) Put(x *bytes.Buffer) {
	if x == nil {
		return
	}
	p.pool.Put(x)
}

// SplitPath split extension and remaining prefix
func splitPath(fullPath string) (prefix, extension string) {
	fullPath, _ = filepath.Abs(fullPath)

	for i := len(fullPath) - 1; i >= 0 && !os.IsPathSeparator(fullPath[i]); i-- {
		if fullPath[i] == '.' {
			return fullPath[:i], fullPath[i:]
		}
	}
	return fullPath, ""
}

// Dir shortcut for filepath.Dir
func dir(fullPath string) string {
	return filepath.Dir(fullPath)
}
