package helper

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/AtlanCI/zlog"
)

const (
	colorRed = uint8(iota + 91)
	colorGreen
	colorYellow
	colorBlue

	ecsControl = "\x1b"
)

type textStdout struct {
	enableColors bool
	level        zlog.Level
	bufferPool   BytesBufferPool
	writer       io.Writer
}

func NewTextStdoutLogger() zlog.Logger {
	return &textStdout{
		level:        zlog.LevelDebug,
		bufferPool:   newBufferPool(16 << 10),
		writer:       os.Stdout,
		enableColors: true,
	}
}

// Log Time Level TraceID Caller Message
func (l *textStdout) Log(t time.Time, lv zlog.Level, tid string, c *zlog.Caller, format string, v ...interface{}) {
	buffer := l.bufferPool.Get()
	defer l.bufferPool.Put(buffer)
	buffer.Reset()

	y, m, d := t.Date()
	hh, mm, ss := t.Clock()

	var prefix, suffix string
	if l.enableColors {
		prefix, suffix = color(lv)
	}

	// x1b[?m
	buffer.WriteString(prefix)

	//[2021-03-17 19:25:50][1615980441.370]
	buffer.WriteString(fmt.Sprintf("[%04d-%02d-%02d %02d:%02d:%02d][%d.%d]", y, m, d, hh, mm, ss, t.Unix(), t.Nanosecond()/int(time.Millisecond)))

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

	// \x1b[0m
	buffer.WriteString(suffix)

	//\n
	if format[len(format)-1] != '\n' {
		buffer.WriteByte('\n')
	}

	_, err := buffer.WriteTo(l.writer)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "stdlog write error. err=%s\n", err)
	}
}

// SetLevel set globe log level
func (l *textStdout) SetLevel(lv zlog.Level) {
	l.level = lv
}

// GetLevel get globe log level
func (l *textStdout) GetLevel() zlog.Level {
	return l.level
}

// Close ...
func (l *textStdout) Close() {
	//do nothing
}

func color(level zlog.Level) (string, string) {
	switch level {
	case zlog.LevelInfo:
		return fmt.Sprintf("%s[%dm", ecsControl, colorGreen), ecsControl + "[0m"
	case zlog.LevelDebug:
		return fmt.Sprintf("%s[%dm", ecsControl, colorBlue), ecsControl + "[0m"
	case zlog.LevelError:
		return fmt.Sprintf("%s[%dm", ecsControl, colorRed), ecsControl + "[0m"
	case zlog.LevelWarn:
		return fmt.Sprintf("%s[%dm", ecsControl, colorYellow), ecsControl + "[0m"
	default:
		return fmt.Sprintf("%s[%dm", ecsControl, colorBlue), ecsControl + "[0m"
	}
}
