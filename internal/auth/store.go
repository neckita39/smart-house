// Package auth хранит OAuth-токен Яндекса и следит за его свежестью.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"smarthome/internal/yandex"
)

// Store — токен в JSON-файле; пишется атомарно с правами 0600.
type Store struct {
	Path string
}

// Load возвращает ok=false, если файла ещё нет.
func (s *Store) Load() (yandex.Token, bool, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return yandex.Token{}, false, nil
	}
	if err != nil {
		return yandex.Token{}, false, err
	}
	var t yandex.Token
	if err := json.Unmarshal(data, &t); err != nil {
		return yandex.Token{}, false, fmt.Errorf("файл токена %s: %w", s.Path, err)
	}
	return t, true, nil
}

func (s *Store) Save(t yandex.Token) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
