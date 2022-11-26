package zlog

import "fmt"

// Level is a logger Level
type Level int8

const (
	// LevelError is logger error level
	LevelError Level = iota
	// LevelWarn is logger warn level
	LevelWarn
	// LevelInfo is logger info level
	LevelInfo
	// LevelDebug is logger debug level
	LevelDebug
)

// Enabled compare whether th logging level is enabled.
func (l Level) Enabled(lv Level) bool {
	return lv <= l
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("LEVEL(%d)", l)
	}
}

func (l Level) StringShort() byte {
	switch l {
	case LevelDebug:
		return 'D'
	case LevelInfo:
		return 'I'
	case LevelWarn:
		return 'W'
	case LevelError:
		return 'E'
	default:
		if l > LevelDebug {
			return '+'
		} else if l < LevelError {
			return '-'
		}
	}
	return '?'
}
