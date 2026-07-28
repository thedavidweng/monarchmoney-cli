package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/config"
)

type Record struct {
	Timestamp  time.Time `json:"timestamp"`
	Command    string    `json:"command"`
	ResourceID string    `json:"resource_id,omitempty"`
	DryRun     bool      `json:"dry_run"`
	Confirmed  bool      `json:"confirmed"`
	Profile    string    `json:"profile"`
	Result     string    `json:"result"`
	ErrorCode  string    `json:"error_code,omitempty"`
}

type Logger struct {
	Dir string
}

func NewLogger() *Logger {
	return &Logger{Dir: config.DefaultAuditDir()}
}

func (l *Logger) Log(r *Record) (err error) {
	if err := os.MkdirAll(l.Dir, 0o700); err != nil {
		return err
	}

	r.Timestamp = time.Now().UTC()
	filename := r.Timestamp.Format("2006-01-02") + ".jsonl"
	path := filepath.Join(l.Dir, filename)

	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if _, wErr := f.Write(append(data, '\n')); wErr != nil {
		return wErr
	}

	return nil
}

func (l *Logger) Cleanup(olderThanDays int) (int, error) {
	entries, err := os.ReadDir(l.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read audit directory: %w", err)
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".jsonl" {
			continue
		}
		dateStr := name[:len(name)-6] // strip ".jsonl"
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if fileDate.Before(cutoff) {
			path := filepath.Join(l.Dir, name)
			if err := os.Remove(path); err != nil {
				return removed, fmt.Errorf("failed to remove %s: %w", path, err)
			}
			removed++
		}
	}
	return removed, nil
}
