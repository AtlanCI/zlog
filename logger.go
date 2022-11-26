package zlog

import (
	"log"
	"time"
)

type Caller struct {
	Line int
	File string
}

// Logger is a log record
type Logger interface {
	// Log need to format and output: Time Level TraceID Caller Message(format,v...)
	//! Do not filter level, just log.
	Log(t time.Time, lv Level, tid string, c *Caller, format string, v ...interface{})

	// SetLevel set logger level
	SetLevel(Level)

	// GetLevel use by log to filter
	GetLevel() Level

	// Close file or other output resource
	Close()
}

// HijackStdlog hijack standard log
// Hijack standard library log.Print
// not fmt.Println...
// example: [2022-11-26 19:08:24][1669460913.111][D] [stdlog]the is example log
func HijackStdlog() {
	log.Default().SetFlags(0) //关闭所有日志头
	log.SetOutput(WrapWriter(LevelDebug, "stdlog"))
}
