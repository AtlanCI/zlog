package zlog

import (
	"io"
	"time"
)

type writerFunc func(p []byte) (n int, err error)

func (f writerFunc) Write(p []byte) (n int, err error) {
	return f(p)
}

func WrapWriter(lv Level, tag string) io.Writer {
	return writerFunc(func(p []byte) (n int, err error) {
		if !LevelEnabled(lv) {
			// Ignore
			return len(p), nil
		}

		s := string(p)
		now := time.Now()

		for i := 0; i < len(_loggers); i++ {
			if _loggers[i].GetLevel().Enabled(lv) {
				_loggers[i].Log(now, lv, tag, nil, s)
			}
		}

		return len(p), nil
	})
}
