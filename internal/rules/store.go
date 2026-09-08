package rules

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"smarthome/internal/snapshot"
)

// Store хранит правила в JSON-файле; сервер — единственный писатель.
type Store struct {
	path string

	mu    sync.Mutex
	rules []Rule
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &s.rules); err != nil {
		return nil, fmt.Errorf("файл правил %s: %w", path, err)
	}
	return s, nil
}

func (s *Store) List() []Rule {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Rule, len(s.rules))
	copy(out, s.rules)
	return out
}

func (s *Store) Get(id string) (Rule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i := s.index(id); i >= 0 {
		return s.rules[i], true
	}
	return Rule{}, false
}

func (s *Store) Create(r Rule, snap *snapshot.Snapshot, macroExists func(string) bool) (Rule, error) {
	if r.CooldownMinutes == 0 {
		r.CooldownMinutes = DefaultCooldownMinutes
	}
	if err := r.Validate(snap, macroExists); err != nil {
		return Rule{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r.ID = newID()
	s.rules = append(s.rules, r)
	if err := s.save(); err != nil {
		s.rules = s.rules[:len(s.rules)-1]
		return Rule{}, err
	}
	return r, nil
}

func (s *Store) Update(r Rule, snap *snapshot.Snapshot, macroExists func(string) bool) error {
	if r.CooldownMinutes == 0 {
		r.CooldownMinutes = DefaultCooldownMinutes
	}
	if err := r.Validate(snap, macroExists); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(r.ID)
	if i < 0 {
		return ErrNotFound
	}
	old := s.rules[i]
	s.rules[i] = r
	if err := s.save(); err != nil {
		s.rules[i] = old
		return err
	}
	return nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(id)
	if i < 0 {
		return ErrNotFound
	}
	old := s.rules
	s.rules = append(s.rules[:i:i], s.rules[i+1:]...)
	if err := s.save(); err != nil {
		s.rules = old
		return err
	}
	return nil
}

func (s *Store) index(id string) int {
	for i, r := range s.rules {
		if r.ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.rules, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand недоступен: " + err.Error())
	}
	return hex.EncodeToString(b)
}
