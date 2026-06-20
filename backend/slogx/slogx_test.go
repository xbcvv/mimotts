package slogx

import (
	"os"
	"strings"
	"testing"
)

func TestLogLevel(t *testing.T) {
	var buf strings.Builder
	l := New(&buf)
	l.level = DebugLevel
	l.format = "json"

	l.DebugMsg("test debug")
	l.InfoMsg("test info")

	if !strings.Contains(buf.String(), `"level":"debug"`) {
		t.Error("expected debug level in output")
	}
	if !strings.Contains(buf.String(), `"level":"info"`) {
		t.Error("expected info level in output")
	}
}

func TestLogLevelFilter(t *testing.T) {
	var buf strings.Builder
	l := New(&buf)
	l.level = WarnLevel
	l.format = "json"

	l.InfoMsg("should be filtered")
	l.WarnMsg("should appear")

	if strings.Contains(buf.String(), "should be filtered") {
		t.Error("info should be filtered when level=warn")
	}
	if !strings.Contains(buf.String(), "should appear") {
		t.Error("warn should appear")
	}
}

func TestTextFormat(t *testing.T) {
	var buf strings.Builder
	l := New(&buf)
	l.level = DebugLevel
	l.format = "text"

	l.InfoMsg("hello", map[string]any{"key": "val"})
	output := buf.String()
	if !strings.Contains(output, "[info]") {
		t.Errorf("expected [info] in text format, got: %q", output)
	}
}

func TestSanitizeErr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bearer token stripped",
			input: "MiMo API 返回 401: Bearer sk-abc123def456 token invalid",
			want:  "MiMo API 返回 401: Bearer **** token invalid",
		},
		{
			name:  "mtts token stripped",
			input: "error with mtts_abc123xyz token",
			want:  "error with mtts_**** token",
		},
		{
			name:  "sk- prefix stripped",
			input: "key sk-abc123def is invalid",
			want:  "key sk-**** is invalid",
		},
		{
			name:  "truncation",
			input: strings.Repeat("x", 300),
			want:  strings.Repeat("x", 200) + "...",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeErr(tc.input, 200)
			if got != tc.want {
				t.Errorf("SanitizeErr(%q) = %q, want %q", tc.input[:min(50, len(tc.input))], got[:min(50, len(got))], tc.want[:min(50, len(tc.want))])
			}
		})
	}
}

func TestMaskURL(t *testing.T) {
	got := MaskURL("https://api.example.com/v1?secret=abc")
	if got != "https://api.example.com/v1?..." {
		t.Errorf("MaskURL = %q, want %q", got, "https://api.example.com/v1?...")
	}
	got = MaskURL("https://api.example.com/v1")
	if got != "https://api.example.com/v1" {
		t.Errorf("MaskURL = %q, want %q", got, "https://api.example.com/v1")
	}
}

func TestEnvConfig(t *testing.T) {
	os.Setenv("LOG_LEVEL", "error")
	defer os.Unsetenv("LOG_LEVEL")
	l := New(os.Stderr)
	if l.level != ErrorLevel {
		t.Errorf("expected level=Error, got %v", l.level)
	}
}
