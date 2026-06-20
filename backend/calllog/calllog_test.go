package calllog

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordAndQuery(t *testing.T) {
	dir := t.TempDir()
	ls, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()

	now := time.Now().Truncate(time.Millisecond)
	r := CallRecord{
		RequestID:       "req_1",
		Timestamp:       now,
		Endpoint:        "/api/tts",
		Model:           "mimo-v2.5-tts",
		Voice:           "冰糖",
		UpstreamID:      "mk_1",
		UpstreamLabel:   "主Key",
		UpstreamBaseURL: "https://api.xiaomimimo.com/v1",
		CallerKeyID:     "ek_1",
		Success:         true,
		HTTPStatus:      200,
		DurationMs:      520,
		AudioBytes:      1024,
		InputChars:      50,
	}
	if err := ls.Record(r); err != nil {
		t.Fatal(err)
	}

	// Query all
	qr, err := ls.Query(QueryOpts{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if qr.Total != 1 {
		t.Fatalf("expected total=1, got %d", qr.Total)
	}
	if len(qr.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(qr.Logs))
	}
	if qr.Logs[0].RequestID != "req_1" {
		t.Errorf("expected requestID=req_1, got %s", qr.Logs[0].RequestID)
	}

	// Filter by endpoint
	qr, err = ls.Query(QueryOpts{Limit: 10, Endpoint: "/api/tts"})
	if err != nil {
		t.Fatal(err)
	}
	if qr.Total != 1 {
		t.Fatalf("expected total=1 for /api/tts, got %d", qr.Total)
	}

	// Filter by endpoint (no match)
	qr, err = ls.Query(QueryOpts{Limit: 10, Endpoint: "/v1/audio/speech"})
	if err != nil {
		t.Fatal(err)
	}
	if qr.Total != 0 {
		t.Fatalf("expected total=0 for /v1/audio/speech, got %d", qr.Total)
	}

	// Filter by success
	success := true
	qr, err = ls.Query(QueryOpts{Limit: 10, Success: &success})
	if err != nil {
		t.Fatal(err)
	}
	if qr.Total != 1 {
		t.Fatalf("expected total=1 for success=true, got %d", qr.Total)
	}
	fail := false
	qr, err = ls.Query(QueryOpts{Limit: 10, Success: &fail})
	if err != nil {
		t.Fatal(err)
	}
	if qr.Total != 0 {
		t.Fatalf("expected total=0 for success=false, got %d", qr.Total)
	}
}

func TestRecordFailure(t *testing.T) {
	dir := t.TempDir()
	ls, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()

	r := CallRecord{
		RequestID:     "req_2",
		Timestamp:     time.Now(),
		Endpoint:      "/v1/audio/speech",
		Model:         "mimo-v2.5-tts-voiceclone",
		Voice:         "clone_1",
		UpstreamID:    "up_1",
		UpstreamLabel: "上游A",
		Success:       false,
		HTTPStatus:    502,
		DurationMs:    3000,
		Error:         "MiMo API 返回 502: {\"error\":\"...\"}",
	}
	if err := ls.Record(r); err != nil {
		t.Fatal(err)
	}

	qr, err := ls.Query(QueryOpts{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if qr.Total != 1 {
		t.Fatalf("expected total=1, got %d", qr.Total)
	}
	if qr.Logs[0].Success {
		t.Error("expected success=false")
	}
	if qr.Logs[0].Error == "" {
		t.Error("expected error message")
	}
}

func TestStats(t *testing.T) {
	dir := t.TempDir()
	ls, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()

	// Write two records
	for i := 0; i < 2; i++ {
		r := CallRecord{
			RequestID:       "req_s",
			Timestamp:       time.Now(),
			Endpoint:        "/api/tts",
			Model:           "mimo-v2.5-tts",
			Voice:           "冰糖",
			UpstreamID:      "mk_1",
			UpstreamLabel:   "主Key",
			UpstreamBaseURL: "https://api.xiaomimimo.com/v1",
			Success:         i == 0,
			DurationMs:      500,
			AudioBytes:      2048,
			InputChars:      100,
		}
		if err := ls.Record(r); err != nil {
			t.Fatal(err)
		}
	}

	stats, err := ls.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalCalls != 2 {
		t.Fatalf("expected totalCalls=2, got %d", stats.TotalCalls)
	}
	if stats.SuccessCalls != 1 {
		t.Fatalf("expected successCalls=1, got %d", stats.SuccessCalls)
	}
	if stats.FailedCalls != 1 {
		t.Fatalf("expected failedCalls=1, got %d", stats.FailedCalls)
	}
	if stats.TotalAudioBytes != 4096 {
		t.Fatalf("expected totalAudioBytes=4096, got %d", stats.TotalAudioBytes)
	}
	if stats.AvgDurationMs != 500 {
		t.Fatalf("expected avgDurationMs=500, got %d", stats.AvgDurationMs)
	}
}

func TestPurge(t *testing.T) {
	dir := t.TempDir()
	ls, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()

	// Create an old file manually
	oldDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	oldPath := filepath.Join(dir, "call-logs-"+oldDate+".jsonl")
	if err := os.WriteFile(oldPath, []byte(`{"requestId":"old"}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	// Record something today (so today's file exists)
	ls.Record(CallRecord{RequestID: "new", Timestamp: time.Now(), Endpoint: "/api/tts"})

	// Purge anything older than 7 days
	deleted, err := ls.PurgeBefore(time.Now().AddDate(0, 0, -7))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	// Verify old file is gone
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("expected old file to be deleted")
	}
}

func TestQueryOffsetLimit(t *testing.T) {
	dir := t.TempDir()
	ls, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()

	for i := 0; i < 5; i++ {
		ls.Record(CallRecord{
			RequestID: fmt.Sprintf("req_%d", i), //nolint:govet
			Timestamp: time.Now(),
			Endpoint:  "/api/tts",
			Success:   true,
		})
	}

	qr, err := ls.Query(QueryOpts{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if qr.Total != 5 {
		t.Fatalf("expected total=5, got %d", qr.Total)
	}
	if len(qr.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(qr.Logs))
	}

	qr, err = ls.Query(QueryOpts{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(qr.Logs) != 1 {
		t.Fatalf("expected 1 log (offset past most), got %d", len(qr.Logs))
	}
}
