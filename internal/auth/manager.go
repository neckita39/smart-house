package auth

import (
	"context"
	"errors"
	"sync"
	"time"

	"smarthome/internal/yandex"
)

var ErrNoToken = errors.New("нет токена Яндекса: сначала войдите")

// refreshBefore — за сколько до истечения обновлять access-токен.
const refreshBefore = 24 * time.Hour

type Manager struct {
	oauth *yandex.OAuth
	store *Store
	now   func() time.Time

	mu    sync.Mutex
	token yandex.Token
	has   bool
}

func NewManager(o *yandex.OAuth, s *Store) (*Manager, error) {
	tok, ok, err := s.Load()
	if err != nil {
		return nil, err
	}
	return &Manager{oauth: o, store: s, now: time.Now, token: tok, has: ok}, nil
}

func (m *Manager) LoginURL() string { return m.oauth.LoginURL() }

func (m *Manager) Authorized() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.has
}

// Login меняет код подтверждения на токены и сохраняет их.
func (m *Manager) Login(ctx context.Context, code string) error {
	tok, err := m.oauth.Exchange(ctx, code)
	if err != nil {
		return err
	}
	return m.set(tok)
}

// Token возвращает действующий access-токен, обновляя его, если срок почти вышел.
func (m *Manager) Token(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.has {
		return "", ErrNoToken
	}
	if m.now().Add(refreshBefore).Before(m.token.ExpiresAt) {
		return m.token.AccessToken, nil
	}
	return m.refreshLocked(ctx)
}

// Refresh принудительно обновляет токен (например, после 401 от API).
func (m *Manager) Refresh(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.has {
		return "", ErrNoToken
	}
	return m.refreshLocked(ctx)
}

// refreshLocked обновляет и сохраняет токен; вызывать под m.mu — так параллельные
// вызовы выстраиваются в очередь и не обновляют один и тот же refresh-токен дважды.
func (m *Manager) refreshLocked(ctx context.Context) (string, error) {
	fresh, err := m.oauth.Refresh(ctx, m.token.RefreshToken)
	if err != nil {
		return "", err
	}
	if err := m.store.Save(fresh); err != nil {
		return "", err
	}
	m.token = fresh
	return fresh.AccessToken, nil
}

func (m *Manager) set(tok yandex.Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.store.Save(tok); err != nil {
		return err
	}
	m.token, m.has = tok, true
	return nil
}
