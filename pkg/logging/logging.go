// logging 日志系统
package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

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
		return "UNKNOWN"
	}
}

func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Logger struct {
	level  Level
	format Format
	out    io.Writer
}

var defaultLogger = &Logger{
	level:  LevelInfo,
	format: FormatText,
	out:    os.Stderr,
}

func NewLogger(level Level, format Format, output io.Writer) *Logger {
	return &Logger{level: level, format: format, out: output}
}

func SetDefaultLogger(l *Logger) { defaultLogger = l }

func (l *Logger) log(level Level, msg string, args ...any) {
	if level < l.level {
		return
	}
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.output(level, msg)
}

func (l *Logger) output(level Level, msg string) {
	if l.format == FormatJSON {
		entry := map[string]any{
			"time":  time.Now().Format(time.RFC3339),
			"level": level.String(),
			"msg":   msg,
		}
		json.NewEncoder(l.out).Encode(entry)
		return
	}
	fmt.Fprintf(l.out, "[%s] %s %s\n", time.Now().Format("15:04:05"), level.String(), msg)
}

func (l *Logger) Debug(msg string, args ...any) { l.log(LevelDebug, msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.log(LevelInfo, msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.log(LevelWarn, msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.log(LevelError, msg, args...) }

func Debug(msg string, args ...any) { defaultLogger.Debug(msg, args...) }
func Info(msg string, args ...any)  { defaultLogger.Info(msg, args...) }
func Warn(msg string, args ...any)  { defaultLogger.Warn(msg, args...) }
func Error(msg string, args ...any) { defaultLogger.Error(msg, args...) }
