// Package macros — локальные «сценарии»: именованные наборы действий над устройствами.
package macros

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"smarthome/internal/yandex"
)

var ErrNotFound = errors.New("макрос не найден")

type Action struct {
	DeviceID string `json:"device_id"`
	Type     string `json:"type"`
	Instance string `json:"instance"`
	Value    any    `json:"value"`
}

type Macro struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Actions []Action `json:"actions"`
}

func (m Macro) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return errors.New("у макроса должно быть имя")
	}
	if len(m.Actions) == 0 {
		return errors.New("макрос должен содержать хотя бы одно действие")
	}
	for i, a := range m.Actions {
		if a.DeviceID == "" || a.Type == "" || a.Instance == "" {
			return fmt.Errorf("действие %d: нужны device_id, type и instance", i+1)
		}
	}
	return nil
}

// ToActionsRequest группирует действия по устройствам в формат devices/actions Яндекса,
// сохраняя порядок первого появления устройства.
func (m Macro) ToActionsRequest() yandex.ActionsRequest {
	var req yandex.ActionsRequest
	index := map[string]int{}
	for _, a := range m.Actions {
		i, ok := index[a.DeviceID]
		if !ok {
			i = len(req.Devices)
			index[a.DeviceID] = i
			req.Devices = append(req.Devices, yandex.DeviceActions{ID: a.DeviceID})
		}
		req.Devices[i].Actions = append(req.Devices[i].Actions, yandex.Action{
			Type:  a.Type,
			State: yandex.ActionState{Instance: a.Instance, Value: a.Value},
		})
	}
	return req
}

// Store хранит макросы в JSON-файле; все операции потокобезопасны.
type Store struct {
	path string

	mu     sync.Mutex
	macros []Macro
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
	if err := json.Unmarshal(data, &s.macros); err != nil {
		return nil, fmt.Errorf("файл макросов %s: %w", path, err)
	}
	return s, nil
}

func (s *Store) List() []Macro {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Macro, len(s.macros))
	copy(out, s.macros)
	return out
}

func (s *Store) Get(id string) (Macro, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i := s.index(id); i >= 0 {
		return s.macros[i], true
	}
	return Macro{}, false
}

func (s *Store) Create(m Macro) (Macro, error) {
	if err := m.Validate(); err != nil {
		return Macro{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m.ID = newID()
	s.macros = append(s.macros, m)
	if err := s.save(); err != nil {
		s.macros = s.macros[:len(s.macros)-1]
		return Macro{}, err
	}
	return m, nil
}

func (s *Store) Update(m Macro) error {
	if err := m.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(m.ID)
	if i < 0 {
		return ErrNotFound
	}
	old := s.macros[i]
	s.macros[i] = m
	if err := s.save(); err != nil {
		s.macros[i] = old
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
	old := s.macros
	s.macros = append(s.macros[:i:i], s.macros[i+1:]...)
	if err := s.save(); err != nil {
		s.macros = old
		return err
	}
	return nil
}

// index ищет макрос по id; вызывать под s.mu.
func (s *Store) index(id string) int {
	for i, m := range s.macros {
		if m.ID == id {
			return i
		}
	}
	return -1
}

// save пишет файл атомарно; вызывать под s.mu.
func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.macros, "", "  ")
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
