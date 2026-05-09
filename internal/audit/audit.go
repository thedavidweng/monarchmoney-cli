package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/config"
)

// Record represents a single audit log entry.
type Record struct {
	Timestamp  time.Time `json:"timestamp"`
	Command    string    `json:"command"`
	ResourceID string    `json:"resource_id,omitempty"`
	DryRun     bool      `json:"dry_run"`
	Confirmed  bool      `json:"confirmed"`
	Profile    string    `json:"profile"`
	ArgsHash   string    `json:"args_hash,omitempty"`
	Result     string    `json:"result"`
	ErrorCode  string    `json:"error_code,omitempty"`
}

// Logger handles writing audit records.
type Logger struct {
	Dir string
}

// NewLogger returns a new Logger.
func NewLogger() *Logger {
	return &Logger{Dir: config.DefaultAuditDir()}
}

// Log writes a record to the daily audit log file.
func (l *Logger) Log(r *Record) error {
	if err := os.MkdirAll(l.Dir, 0700); err != nil {
		return err
	}

	r.Timestamp = time.Now().UTC()
	filename := r.Timestamp.Format("2006-01-02") + ".jsonl"
	path := filepath.Join(l.Dir, filename)

	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}

	return nil
}
