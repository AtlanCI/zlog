package zlog

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
)

var (
	_enableCaller = false

	// cache caller info
	_callerCache sync.Map

	_loggers []Logger

	_stainingTags    = sync.Map{}
	_stainingTagsNum = atomic.NewUint64(0)
)

func SetCallerEnable(enable bool) {
	_enableCaller = enable
}

func AddStainingUsers(tags ...string) {
	for _, u := range tags {
		_, loaded := _stainingTags.LoadOrStore(u, nil)
		if loaded {
			_stainingTagsNum.Inc()
		}
	}
}

func DelStainingUsers(tags ...string) {
	for _, u := range tags {
		_, loaded := _stainingTags.LoadAndDelete(u)
		if loaded {
			_stainingTagsNum.Dec()
		}
	}
}

func GetStainingUsers() []string {
	tags := make([]string, 0, 128)
	_stainingTags.Range(func(key, value interface{}) bool {
		tags = append(tags, key.(string))
		return true
	})
	return tags
}

func AddLoggers(l ...Logger) {
	_loggers = append(_loggers, l...)
}

func Close() {
	for _, l := range _loggers {
		l.Close()
	}
}

func LevelEnabled(lv Level) bool {
	for i := 0; i < len(_loggers); i++ {
		if _loggers[i].GetLevel().Enabled(lv) {
			return true
		}
	}
	return false
}

func Errorf(ctx context.Context, format string, v ...interface{}) {
	var caller *Caller
	if _enableCaller {
		caller = zlogGetCaller(1)
	}

	doLog(ctx, caller, LevelError, format, v...)
}

func Warnf(ctx context.Context, format string, v ...interface{}) {
	var caller *Caller
	if _enableCaller {
		caller = zlogGetCaller(1)
	}
	doLog(ctx, caller, LevelWarn, format, v...)
}

func Infof(ctx context.Context, format string, v ...interface{}) {
	var caller *Caller
	if _enableCaller {
		caller = zlogGetCaller(1)
	}
	doLog(ctx, caller, LevelInfo, format, v...)
}

func Debugf(ctx context.Context, format string, v ...interface{}) {
	var caller *Caller
	if _enableCaller {
		caller = zlogGetCaller(1)
	}
	doLog(ctx, caller, LevelDebug, format, v...)
}

func Logf(ctx context.Context, caller *Caller, lv Level, format string, v ...interface{}) {
	doLog(ctx, caller, lv, format, v...)
}

func doLog(ctx context.Context, caller *Caller, lv Level, format string, v ...interface{}) {
	var isStaining bool
	if ctx != context.Background() && _stainingTagsNum.Load() > 0 {
		//TODO: get uid and test if contain in _stainingTags
		uid := "todo"

		_, isStaining = _stainingTags.Load(uid)
	}

	if !isStaining && !LevelEnabled(lv) {
		// ignore log
		return
	}

	now := time.Now()
	var tid string

	if ctx != context.Background() {
		spanCtx := trace.SpanContextFromContext(ctx)
		if spanCtx.IsValid() && spanCtx.IsSampled() {
			tid = spanCtx.TraceID().String()
		}
	}

	for i := 0; i < len(_loggers); i++ {
		if _loggers[i].GetLevel().Enabled(lv) {
			_loggers[i].Log(now, lv, tid, caller, format, v...)
		}
	}
}
func zlogGetCaller(skip int) (caller *Caller) {
	rpc := [1]uintptr{}

	// skip +2 : +2 cause Callers()=>caller()

	n := runtime.Callers(skip+2, rpc[:])
	if n < 1 {
		return
	}

	pc := rpc[0]

	if item, ok := _callerCache.Load(pc); ok {
		caller = item.(*Caller)
	} else {
		var frame runtime.Frame
		tmpprpc := []uintptr{pc}
		frame, _ = runtime.CallersFrames(tmpprpc).Next()
		if frame.PC == 0 {
			frame.Line = 1
			frame.File = "unknown"
		}

		caller = &Caller{
			File: trimFileName(frame.File),
			Line: frame.Line,
		}

		_callerCache.Store(pc, caller)
	}
	return
}

func trimFileName(name string) string {
	const prefix = "/builds/server/"

	i := strings.Index(name, prefix) + len(prefix)
	if i >= len(prefix) && i < len(name) /* BCE */ {
		name = name[i:]
	}
	return name
}
