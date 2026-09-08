package rules

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Event struct {
	Time     time.Time `json:"time"`
	RuleID   string    `json:"rule_id"`
	RuleName string    `json:"rule_name"`
	OK       bool      `json:"ok"`
	Details  string    `json:"details"`
}

// EventLog дописывает события в JSONL-файл и держит последние keep в памяти.
type EventLog struct {
	path string
	keep int

	mu   sync.Mutex
	ring []Event // старые → новые
}

func NewEventLog(path string, keep int) (*EventLog, error) {
	l := &EventLog{path: path, keep: keep}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var e Event
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			l.push(e)
		}
	}
	return l, sc.Err()
}

func (l *EventLog) push(e Event) {
	l.ring = append(l.ring, e)
	if len(l.ring) > l.keep {
		l.ring = l.ring[len(l.ring)-l.keep:]
	}
}

func (l *EventLog) Append(e Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	l.push(e)
	return nil
}

// List возвращает до limit последних событий, новые сверху.
func (l *EventLog) List(limit int) []Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(l.ring)
	if limit > 0 && limit < n {
		n = limit
	}
	out := make([]Event, 0, n)
	for i := len(l.ring) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, l.ring[i])
	}
	return out
}
