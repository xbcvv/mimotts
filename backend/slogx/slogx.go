// Package slogx provides a lightweight structured logger that outputs JSON lines.
// It respects LOG_LEVEL (debug/info/warn/error) and LOG_FORMAT (json/text) env vars.
// No external dependencies — no CGO, no SQLite, no PostgreSQL.
package slogx

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	}
	return "unknown"
}

// Logger writes structured log entries.
type Logger struct {
	mu     sync.Mutex
	w      interface{ Write([]byte) (int, error) }
	level  Level
	format string // "json" or "text"
}

var defaultLogger *Logger

func init() {
	defaultLogger = New(os.Stderr)
}

// New creates a Logger writing to w, configured via LOG_LEVEL and LOG_FORMAT env vars.
func New(w interface{ Write([]byte) (int, error) }) *Logger {
	l := &Logger{w: w, level: InfoLevel, format: "json"}
	if v := strings.ToLower(os.Getenv("LOG_LEVEL")); v != "" {
		switch v {
		case "debug":
			l.level = DebugLevel
		case "info":
			l.level = InfoLevel
		case "warn":
			l.level = WarnLevel
		case "error":
			l.level = ErrorLevel
		}
	}
	if v := strings.ToLower(os.Getenv("LOG_FORMAT")); v == "text" {
		l.format = "text"
	}
	return l
}

// L returns the package-level default Logger.
func L() *Logger { return defaultLogger }

// SetOutput redirects the default logger.
func SetOutput(w interface{ Write([]byte) (int, error) }) { defaultLogger = New(w) }

type entry struct {
	Level   string         `json:"level"`
	Time    string         `json:"time"`
	Message string         `json:"msg"`
	Fields  map[string]any `json:"fields,omitempty"`
}

func (l *Logger) log(level Level, msg string, fields map[string]any) {
	if level < l.level {
		return
	}
	e := entry{
		Level:   level.String(),
		Time:    time.Now().Format(time.RFC3339Nano),
		Message: msg,
		Fields:  fields,
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.format == "text" {
		// Simple text format for dev
		fmt_ := fmt.Sprintf("[%s] %s %v\n", e.Level, e.Message, e.Fields)
		l.w.Write([]byte(fmt_))
		return
	}
	line, _ := json.Marshal(e)
	l.w.Write(append(line, '\n'))
}

func (l *Logger) DebugMsg(msg string, fields ...map[string]any) {
	f := merge(fields)
	l.log(DebugLevel, msg, f)
}
func (l *Logger) InfoMsg(msg string, fields ...map[string]any) {
	f := merge(fields)
	l.log(InfoLevel, msg, f)
}
func (l *Logger) WarnMsg(msg string, fields ...map[string]any) {
	f := merge(fields)
	l.log(WarnLevel, msg, f)
}
func (l *Logger) ErrorMsg(msg string, fields ...map[string]any) {
	f := merge(fields)
	l.log(ErrorLevel, msg, f)
}

// Package-level convenience functions
func Debug(msg string, fields ...map[string]any) { defaultLogger.DebugMsg(msg, fields...) }
func Info(msg string, fields ...map[string]any)  { defaultLogger.InfoMsg(msg, fields...) }
func Warn(msg string, fields ...map[string]any)  { defaultLogger.WarnMsg(msg, fields...) }
func Error(msg string, fields ...map[string]any) { defaultLogger.ErrorMsg(msg, fields...) }

func merge(fields []map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	m := make(map[string]any, len(fields[0]))
	for _, f := range fields {
		for k, v := range f {
			m[k] = v
		}
	}
	return m
}

// SanitizeErr truncates and sanitizes error messages for safe logging.
// Removes any Authorization/bearer token patterns and truncates to maxLen.
func SanitizeErr(errMsg string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 200
	}
	s := strings.TrimSpace(errMsg)
	// Remove common secret patterns
	s = replaceBearer(s)
	s = replaceTokenPrefix(s, "mtts_", "mtts_****")
	s = replaceTokenPrefix(s, "sk-", "sk-****")
	if len(s) > maxLen {
		s = s[:maxLen] + "..."
	}
	return s
}

// MaskURL removes the API key query parameter from URLs.
func MaskURL(rawURL string) string {
	if idx := strings.Index(rawURL, "?"); idx >= 0 {
		return rawURL[:idx] + "?..."
	}
	return rawURL
}

func replaceBearer(s string) string {
	idx := strings.Index(s, "Bearer ")
	if idx < 0 {
		return s
	}
	after := s[idx+7:]
	end := strings.IndexAny(after, " \t\n\r,;:\"'")
	if end < 0 {
		end = len(after)
	}
	return s[:idx] + "Bearer ****" + after[end:]
}

func replaceTokenPrefix(s, prefix, repl string) string {
	idx := strings.Index(s, prefix)
	if idx < 0 {
		return s
	}
	after := s[idx:]
	end := 0
	for end < len(after) && (after[end] >= 'a' && after[end] <= 'z' || after[end] >= 'A' && after[end] <= 'Z' || after[end] >= '0' && after[end] <= '9' || after[end] == '_' || after[end] == '-') {
		end++
	}
	return s[:idx] + repl + after[end:]
}
