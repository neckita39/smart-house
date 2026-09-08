package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"smarthome/internal/yandex"
)

var fixedNow = time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)

// fakeOAuth поднимает сервер токенов, считающий запросы по grant_type.
func fakeOAuth(t *testing.T, token string) (*yandex.OAuth, map[string]int) {
	t.Helper()
	calls := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		calls[r.PostForm.Get("grant_type")]++
		fmt.Fprintf(w, `{"access_token":%q,"refresh_token":"ref-new","expires_in":31536000}`, token)
	}))
	t.Cleanup(srv.Close)
	return &yandex.OAuth{ClientID: "cid", ClientSecret: "sec", TokenURL: srv.URL,
		Now: func() time.Time { return fixedNow }}, calls
}

func newTestManager(t *testing.T, o *yandex.OAuth) (*Manager, *Store) {
	t.Helper()
	store := &Store{Path: filepath.Join(t.TempDir(), "data", "token.json")}
	m, err := NewManager(o, store)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	m.now = func() time.Time { return fixedNow }
	return m, store
}

func TestTokenWithoutLoginReturnsErrNoToken(t *testing.T) {
	o, _ := fakeOAuth(t, "acc")
	m, _ := newTestManager(t, o)
	if m.Authorized() {
		t.Error("Authorized() = true до входа")
	}
	if _, err := m.Token(context.Background()); err != ErrNoToken {
		t.Errorf("Token err = %v, want ErrNoToken", err)
	}
}

func TestLoginExchangesAndPersists(t *testing.T) {
	o, calls := fakeOAuth(t, "acc")
	m, store := newTestManager(t, o)

	if err := m.Login(context.Background(), "1234567"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if calls["authorization_code"] != 1 {
		t.Errorf("authorization_code calls = %d", calls["authorization_code"])
	}
	if !m.Authorized() {
		t.Error("Authorized() = false после входа")
	}
	tok, err := m.Token(context.Background())
	if err != nil || tok != "acc" {
		t.Errorf("Token = %q, %v", tok, err)
	}

	saved, ok, err := store.Load()
	if err != nil || !ok || saved.AccessToken != "acc" || saved.RefreshToken != "ref-new" {
		t.Errorf("saved = %+v, ok=%v, err=%v", saved, ok, err)
	}
	info, err := os.Stat(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("token file perm = %o, want 600", perm)
	}
}

func TestNewManagerLoadsSavedToken(t *testing.T) {
	o, _ := fakeOAuth(t, "unused")
	store := &Store{Path: filepath.Join(t.TempDir(), "token.json")}
	if err := store.Save(yandex.Token{AccessToken: "saved", RefreshToken: "r", ExpiresAt: fixedNow.Add(365 * 24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(o, store)
	if err != nil {
		t.Fatal(err)
	}
	m.now = func() time.Time { return fixedNow }
	if tok, err := m.Token(context.Background()); err != nil || tok != "saved" {
		t.Errorf("Token = %q, %v", tok, err)
	}
}

func TestTokenRefreshesWhenExpiringSoon(t *testing.T) {
	o, calls := fakeOAuth(t, "fresh")
	m, store := newTestManager(t, o)
	if err := store.Save(yandex.Token{AccessToken: "stale", RefreshToken: "ref-old", ExpiresAt: fixedNow.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(o, store)
	if err != nil {
		t.Fatal(err)
	}
	m.now = func() time.Time { return fixedNow }

	tok, err := m.Token(context.Background())
	if err != nil || tok != "fresh" {
		t.Fatalf("Token = %q, %v; want fresh", tok, err)
	}
	if calls["refresh_token"] != 1 {
		t.Errorf("refresh calls = %d, want 1", calls["refresh_token"])
	}
	saved, _, _ := store.Load()
	if saved.AccessToken != "fresh" {
		t.Errorf("persisted token = %q, want fresh", saved.AccessToken)
	}
	// Повторный вызов не обновляет заново — новый токен свежий.
	m.Token(context.Background())
	if calls["refresh_token"] != 1 {
		t.Errorf("refresh calls after second Token = %d, want 1", calls["refresh_token"])
	}
}

func TestRefreshForcesNewToken(t *testing.T) {
	o, calls := fakeOAuth(t, "forced")
	m, store := newTestManager(t, o)
	store.Save(yandex.Token{AccessToken: "ok", RefreshToken: "r", ExpiresAt: fixedNow.Add(365 * 24 * time.Hour)})
	m, _ = NewManager(o, store)
	m.now = func() time.Time { return fixedNow }

	tok, err := m.Refresh(context.Background())
	if err != nil || tok != "forced" || calls["refresh_token"] != 1 {
		t.Errorf("Refresh = %q, %v, calls=%d", tok, err, calls["refresh_token"])
	}
}

func TestRefreshWithoutTokenReturnsErrNoToken(t *testing.T) {
	o, _ := fakeOAuth(t, "x")
	m, _ := newTestManager(t, o)
	if _, err := m.Refresh(context.Background()); err != ErrNoToken {
		t.Errorf("err = %v, want ErrNoToken", err)
	}
}
