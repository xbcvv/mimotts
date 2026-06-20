package calllog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// CallRecord represents a single model invocation log entry.
type CallRecord struct {
	RequestID       string    `json:"requestId"`
	Timestamp       time.Time `json:"timestamp"`
	Endpoint        string    `json:"endpoint"`                 // /api/tts or /v1/audio/speech
	Model           string    `json:"model"`                    // actual upstream model after voice resolution
	Voice           string    `json:"voice"`                    // actual upstream voice/preset after voice resolution
	VoiceKind       string    `json:"voiceKind,omitempty"`      // preset/design/clone when project voice is used
	ProjectVoiceID  string    `json:"projectVoiceId,omitempty"` // original project voice id, if any
	RequestedModel  string    `json:"requestedModel,omitempty"` // model requested by caller before resolution
	RequestedVoice  string    `json:"requestedVoice,omitempty"` // voice requested by caller before resolution
	UpstreamID      string    `json:"upstreamId"`               // channel/key ID used
	UpstreamLabel   string    `json:"upstreamLabel"`
	UpstreamBaseURL string    `json:"upstreamBaseUrl"`
	CallerKeyID     string    `json:"callerKeyId"` // external key ID (not secret)
	Success         bool      `json:"success"`
	HTTPStatus      int       `json:"httpStatus"`
	DurationMs      int64     `json:"durationMs"`
	AudioBytes      int       `json:"audioBytes"`
	InputChars      int       `json:"inputChars"`
	Error           string    `json:"error,omitempty"` // sanitized/truncated error
}

// CallLogStats represents aggregate statistics.
type CallLogStats struct {
	TotalCalls      int            `json:"totalCalls"`
	SuccessCalls    int            `json:"successCalls"`
	FailedCalls     int            `json:"failedCalls"`
	TotalAudioBytes int64          `json:"totalAudioBytes"`
	TotalInputChars int64          `json:"totalInputChars"`
	AvgDurationMs   int64          `json:"avgDurationMs"`
	LastCallAt      *time.Time     `json:"lastCallAt,omitempty"`
	ByEndpoint      map[string]int `json:"byEndpoint"`
	ByUpstream      map[string]int `json:"byUpstream"`
	ByModel         map[string]int `json:"byModel"`
}

// LogStore manages call log persistence via JSONL.
type LogStore struct {
	mu   sync.Mutex
	dir  string
	file *os.File
	buf  *bufio.Writer
}

// New creates a new LogStore. Data is stored as append-only JSONL files
// in dir, organized by day (call-logs-2026-06-12.jsonl).
func New(dir string) (*LogStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("calllog: mkdir: %w", err)
	}
	ls := &LogStore{dir: dir}
	if err := ls.openToday(); err != nil {
		return nil, err
	}
	return ls, nil
}

func todayPath(dir string) string {
	return filepath.Join(dir, "call-logs-"+time.Now().Format("2006-01-02")+".jsonl")
}

func (ls *LogStore) openToday() error {
	p := todayPath(ls.dir)
	// If the current file matches today, reuse it
	if ls.file != nil && ls.file.Name() == p {
		return nil
	}
	if ls.file != nil {
		ls.buf.Flush()
		ls.file.Close()
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("calllog: open: %w", err)
	}
	ls.file = f
	ls.buf = bufio.NewWriter(f)
	return nil
}

// Record writes a CallRecord to today's JSONL file.
func (ls *LogStore) Record(r CallRecord) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if err := ls.openToday(); err != nil {
		return err
	}
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := ls.buf.Write(append(line, '\n')); err != nil {
		return err
	}
	return ls.buf.Flush()
}

// Close flushes and closes the current file.
func (ls *LogStore) Close() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.buf != nil {
		ls.buf.Flush()
	}
	if ls.file != nil {
		return ls.file.Close()
	}
	return nil
}

// QueryResult is returned by Query.
type QueryResult struct {
	Logs  []CallRecord `json:"logs"`
	Total int          `json:"total"`
}

// QueryOpts for listing call logs.
type QueryOpts struct {
	Limit    int
	Offset   int
	Endpoint string // filter by endpoint
	Success  *bool  // filter by success
}

// Query reads call logs from JSONL files, sorted newest-first.
func (ls *LogStore) Query(opts QueryOpts) (QueryResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 500 {
		opts.Limit = 500
	}

	ls.mu.Lock()
	ls.buf.Flush()
	ls.mu.Unlock()

	var all []CallRecord
	entries, err := os.ReadDir(ls.dir)
	if err != nil {
		return QueryResult{}, err
	}
	// Read files newest-first (sort by name descending — dates go backwards)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() > entries[j].Name()
	})
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "call-logs-") || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		recs, err := readJSONL(filepath.Join(ls.dir, e.Name()))
		if err != nil {
			continue // skip corrupt files
		}
		all = append(all, recs...)
	}

	// Filter
	filtered := make([]CallRecord, 0, len(all))
	for _, r := range all {
		if opts.Endpoint != "" && r.Endpoint != opts.Endpoint {
			continue
		}
		if opts.Success != nil && r.Success != *opts.Success {
			continue
		}
		filtered = append(filtered, r)
	}

	// Sort newest-first across all files and within each file.
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	total := len(filtered)
	// Apply offset/limit
	start := opts.Offset
	if start > total {
		start = total
	}
	end := start + opts.Limit
	if end > total {
		end = total
	}

	return QueryResult{
		Logs:  filtered[start:end],
		Total: total,
	}, nil
}

// Stats computes aggregate statistics over all call log files.
func (ls *LogStore) Stats() (CallLogStats, error) {
	ls.mu.Lock()
	ls.buf.Flush()
	ls.mu.Unlock()

	var stats CallLogStats
	stats.ByEndpoint = make(map[string]int)
	stats.ByUpstream = make(map[string]int)
	stats.ByModel = make(map[string]int)

	entries, err := os.ReadDir(ls.dir)
	if err != nil {
		return stats, err
	}
	var totalDur int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "call-logs-") || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		recs, err := readJSONL(filepath.Join(ls.dir, e.Name()))
		if err != nil {
			continue
		}
		for _, r := range recs {
			stats.TotalCalls++
			if r.Success {
				stats.SuccessCalls++
			} else {
				stats.FailedCalls++
			}
			stats.TotalAudioBytes += int64(r.AudioBytes)
			stats.TotalInputChars += int64(r.InputChars)
			totalDur += r.DurationMs
			stats.ByEndpoint[r.Endpoint]++
			stats.ByUpstream[r.UpstreamID]++
			stats.ByModel[r.Model]++
			if stats.LastCallAt == nil || r.Timestamp.After(*stats.LastCallAt) {
				t := r.Timestamp
				stats.LastCallAt = &t
			}
		}
	}
	if stats.TotalCalls > 0 {
		stats.AvgDurationMs = totalDur / int64(stats.TotalCalls)
	}
	return stats, nil
}

// PurgeBefore deletes JSONL files older than the given time (by filename date).
func (ls *LogStore) PurgeBefore(before time.Time) (int, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.buf != nil {
		ls.buf.Flush()
	}

	dateStr := before.Format("2006-01-02")
	entries, err := os.ReadDir(ls.dir)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "call-logs-") || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		// Extract date from filename: call-logs-YYYY-MM-DD.jsonl
		name := e.Name()
		datePart := strings.TrimPrefix(name, "call-logs-")
		datePart = strings.TrimSuffix(datePart, ".jsonl")
		if datePart < dateStr {
			p := filepath.Join(ls.dir, name)
			// Don't delete the currently open file
			if ls.file != nil && ls.file.Name() == p {
				continue
			}
			if err := os.Remove(p); err == nil {
				deleted++
			}
		}
	}
	return deleted, nil
}

func readJSONL(path string) ([]CallRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var recs []CallRecord
	dec := json.NewDecoder(f)
	for {
		var r CallRecord
		if err := dec.Decode(&r); err != nil {
			if err == io.EOF {
				break
			}
			// Skip malformed lines
			continue
		}
		recs = append(recs, r)
	}
	return recs, nil
}
