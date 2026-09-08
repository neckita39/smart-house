# Веб-пульт умного дома Яндекса — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Локальное веб-приложение (один Go-бинарник со вшитым React-фронтом) для управления умным домом Яндекса с ноутбука: вкл/выкл, регуляторы, датчики, запуск сценариев и локальные макросы.

**Architecture:** Go-сервер на `127.0.0.1:8080` раздаёт собранный Vite-фронт из `go:embed` и проксирует запросы к `https://api.iot.yandex.net`, подставляя OAuth-токен. Токен получается один раз через ручную вставку кода подтверждения и хранится в `data/token.json` с авто-обновлением. Макросы — свой JSON-файл `data/macros.json`, выполняются одним запросом `devices/actions`.

**Tech Stack:** Go 1.26 (только стандартная библиотека: `net/http` с method/path-паттернами, `log/slog`, `embed`, `httptest`), React 19 + Vite 7 (JavaScript, без UI-библиотек), npm.

**Spec:** `docs/superpowers/specs/2026-09-08-smart-home-web-remote-design.md`

## Global Constraints

- Go: только стандартная библиотека, никаких сторонних модулей.
- Фронтенд: React + Vite, plain CSS, без тестов в первой версии.
- Сервер слушает только `127.0.0.1` (доступ с других устройств вне рамок).
- `.env`, `data/`, `web/node_modules/`, `web/dist/` — не в git.
- Секреты (`YANDEX_CLIENT_SECRET`) никогда не попадают в код, логи и коммиты.
- Каждый запрос к Яндексу логируется: метод, путь, HTTP-статус, `request_id`.
- Все сообщения об ошибках для пользователя — по-русски.
- Коммиты: сообщение по-русски в формате `тип: описание` (feat/fix/docs/chore/test); в конце сообщения строки
  `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` и
  `Claude-Session: https://claude.ai/code/session_01TNjwA7nbzkXDYNwyWA4yGW`.
- Перед каждым коммитом: `gofmt -l .` пустой, `go vet ./...` чистый, `go test ./...` зелёный.

## Что важно знать об API Яндекса

`GET /v1.0/user/info` (заголовок `Authorization: Bearer <token>`) возвращает:

```json
{
  "status": "ok", "request_id": "…",
  "rooms": [{"id": "r1", "name": "Гостиная", "household_id": "h1", "devices": ["d1"]}],
  "groups": [{"id": "g1", "name": "Свет", "type": "devices.types.light", "devices": ["d1"], "capabilities": []}],
  "devices": [{
    "id": "d1", "name": "Лампа", "type": "devices.types.light", "room": "r1", "groups": [],
    "capabilities": [
      {"type": "devices.capabilities.on_off", "retrievable": true, "parameters": {"split": false},
       "state": {"instance": "on", "value": true}, "last_updated": 1757325000.0},
      {"type": "devices.capabilities.range", "retrievable": true,
       "parameters": {"instance": "brightness", "unit": "unit.percent", "random_access": true,
                      "range": {"min": 1, "max": 100, "precision": 1}},
       "state": {"instance": "brightness", "value": 50}},
      {"type": "devices.capabilities.color_setting", "retrievable": true,
       "parameters": {"color_model": "hsv", "temperature_k": {"min": 2700, "max": 6500}},
       "state": {"instance": "temperature_k", "value": 4000}},
      {"type": "devices.capabilities.mode", "retrievable": true,
       "parameters": {"instance": "thermostat", "modes": [{"value": "heat"}, {"value": "cool"}]},
       "state": {"instance": "thermostat", "value": "heat"}}
    ],
    "properties": [
      {"type": "devices.properties.float", "retrievable": true,
       "parameters": {"instance": "temperature", "unit": "unit.temperature.celsius"},
       "state": {"instance": "temperature", "value": 22.5}, "last_updated": 1757325000.0}
    ]
  }],
  "scenarios": [{"id": "s1", "name": "Кино", "is_active": true}],
  "households": [{"id": "h1", "name": "Мой дом"}]
}
```

`POST /v1.0/devices/actions` принимает и возвращает:

```json
// запрос
{"devices": [{"id": "d1", "actions": [
  {"type": "devices.capabilities.on_off", "state": {"instance": "on", "value": true}}
]}]}
// ответ
{"status": "ok", "request_id": "…", "devices": [{"id": "d1", "capabilities": [
  {"type": "devices.capabilities.on_off",
   "state": {"instance": "on", "action_result": {"status": "DONE"}}}
]}]}
// action_result при ошибке:
{"status": "ERROR", "error_code": "DEVICE_UNREACHABLE", "error_message": "Устройство не отвечает"}
```

`POST /v1.0/scenarios/{id}/actions` — без тела, ответ `{"status":"ok","request_id":"…"}`.

Ошибки API: HTTP 4xx/5xx с телом `{"request_id":"…","status":"error","message":"…"}`.

OAuth: `POST https://oauth.yandex.ru/token` (form-urlencoded) с
`grant_type=authorization_code&code=…&client_id=…&client_secret=…` или
`grant_type=refresh_token&refresh_token=…&client_id=…&client_secret=…`.
Успех: `{"access_token":"…","refresh_token":"…","expires_in":31536000,"token_type":"bearer"}`.
Ошибка: HTTP 400 `{"error":"invalid_grant","error_description":"…"}`.
Страница получения кода: `https://oauth.yandex.ru/authorize?response_type=code&client_id=…`.

## Структура файлов

```
go.mod                              module smarthome, go 1.26
main.go                             сборка зависимостей, запуск HTTP-сервера
Makefile                            build / web / test / run
.env.example                        образец конфигурации
README.md                           как собрать и запустить
internal/config/config.go           Config + Load(".env"): .env + переменные окружения
internal/config/config_test.go
internal/yandex/oauth.go            OAuth: LoginURL, Exchange, Refresh; Token
internal/yandex/oauth_test.go
internal/yandex/client.go           Client: UserInfo, DeviceActions, RunScenario; APIError; retry по 401
internal/yandex/client_test.go
internal/auth/store.go              Store: token.json (0600, атомарная запись)
internal/auth/manager.go            Manager: Login, Token (авто-refresh), Refresh, Authorized; ErrNoToken
internal/auth/manager_test.go
internal/macros/store.go            Macro, Action, Store (CRUD в macros.json), ToActionsRequest
internal/macros/store_test.go
internal/catalog/catalog.go         Build(user/info) → компактный каталог устройств для Claude
internal/catalog/catalog_test.go
internal/server/catalog_handler.go  GET /api/catalog
.claude/skills/smart-home/SKILL.md  проектный skill: как Claude собирает и запускает сценарии
internal/server/server.go           Server, Handler(), writeJSON/writeRaw/writeError, интерфейсы HomeAPI и Auth
internal/server/auth_handlers.go    GET /api/auth/status, POST /api/auth/code
internal/server/home_handlers.go    GET /api/home, POST /api/devices/actions, POST /api/scenarios/{id}/run
internal/server/macro_handlers.go   CRUD /api/macros, POST /api/macros/{id}/run
internal/server/*_test.go
web/embed.go                        package web: //go:embed all:dist
web/package.json, vite.config.js, index.html
web/src/main.jsx                    точка входа React
web/src/api.js                      fetch-обёртки над /api/*
web/src/App.jsx                     проверка авторизации → Login | Home
web/src/Login.jsx                   форма ввода кода подтверждения
web/src/Home.jsx                    опрос /api/home, вкладки «Устройства» / «Сценарии»
web/src/useHome.js                  хук опроса раз в 10 с
web/src/Dashboard.jsx               комнаты → карточки устройств
web/src/DeviceCard.jsx              карточка: умения + свойства, отправка действий
web/src/Controls.jsx                OnOff, Toggle, Range, ColorSetting, Mode, PropertyReadout
web/src/labels.js                   русские подписи instance/unit, разбор action_result
web/src/color.js                    hex ↔ rgb-int / hsv
web/src/Scenarios.jsx               родные сценарии + список макросов
web/src/MacroEditor.jsx             создание/редактирование макроса
web/src/styles.css
```

Зависимости пакетов: `server` → `auth`, `macros`, `catalog`, `yandex`; `auth` → `yandex`; `macros` → `yandex`; `catalog` — ни от кого; `main` → все + `web`. Циклов нет.

---

### Task 1: Модуль Go и конфигурация

**Files:**
- Create: `go.mod`
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `config.Config{ClientID, ClientSecret, Port, DataDir string}`, `config.Load(envPath string) (Config, error)`.

- [ ] **Step 1: Создать go.mod**

```bash
cd /Volumes/projects/pet/smart-house && go mod init smarthome
```

Убедиться, что в `go.mod` строка `go 1.26` (или `go 1.26.2`) — нужен `net/http` роутер с паттернами методов.

- [ ] **Step 2: Написать падающий тест**

`internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"YANDEX_CLIENT_ID", "YANDEX_CLIENT_SECRET", "PORT", "DATA_DIR"} {
		t.Setenv(k, "")
	}
}

func TestLoadReadsDotEnvAndDefaults(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), ".env")
	content := "# комментарий\nYANDEX_CLIENT_ID=id1\nYANDEX_CLIENT_SECRET=\"sec1\"\n\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClientID != "id1" || cfg.ClientSecret != "sec1" {
		t.Errorf("creds = %q/%q, want id1/sec1", cfg.ClientID, cfg.ClientSecret)
	}
	if cfg.Port != "8080" || cfg.DataDir != "data" {
		t.Errorf("defaults = port %q, dataDir %q", cfg.Port, cfg.DataDir)
	}
}

func TestLoadEnvOverridesFile(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, []byte("YANDEX_CLIENT_ID=file\nYANDEX_CLIENT_SECRET=s\nPORT=1111\n"), 0o600)
	t.Setenv("YANDEX_CLIENT_ID", "env")
	t.Setenv("PORT", "9090")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClientID != "env" || cfg.Port != "9090" {
		t.Errorf("got clientID %q port %q, want env/9090", cfg.ClientID, cfg.Port)
	}
}

func TestLoadMissingFileIsFine(t *testing.T) {
	clearEnv(t)
	t.Setenv("YANDEX_CLIENT_ID", "id")
	t.Setenv("YANDEX_CLIENT_SECRET", "sec")

	cfg, err := Load(filepath.Join(t.TempDir(), "nope.env"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClientID != "id" {
		t.Errorf("ClientID = %q", cfg.ClientID)
	}
}

func TestLoadFailsWithoutCredentials(t *testing.T) {
	clearEnv(t)
	if _, err := Load(filepath.Join(t.TempDir(), "nope.env")); err == nil {
		t.Fatal("ожидалась ошибка без YANDEX_CLIENT_ID/SECRET")
	}
}

func TestLoadRejectsMalformedLine(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, []byte("YANDEX_CLIENT_ID id\n"), 0o600)
	if _, err := Load(path); err == nil {
		t.Fatal("ожидалась ошибка на строке без '='")
	}
}
```

- [ ] **Step 3: Убедиться, что тест падает**

Run: `go test ./internal/config/`
Expected: ошибка компиляции — `undefined: Load`.

- [ ] **Step 4: Реализация**

`internal/config/config.go`:

```go
// Package config читает настройки приложения из .env и переменных окружения.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ClientID     string
	ClientSecret string
	Port         string
	DataDir      string
}

// Load читает файл envPath (если он есть) и переменные окружения;
// окружение имеет приоритет над файлом.
func Load(envPath string) (Config, error) {
	fileVals, err := readDotEnv(envPath)
	if err != nil {
		return Config{}, err
	}
	get := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		if v := fileVals[key]; v != "" {
			return v
		}
		return def
	}
	cfg := Config{
		ClientID:     get("YANDEX_CLIENT_ID", ""),
		ClientSecret: get("YANDEX_CLIENT_SECRET", ""),
		Port:         get("PORT", "8080"),
		DataDir:      get("DATA_DIR", "data"),
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return Config{}, errors.New("нужны YANDEX_CLIENT_ID и YANDEX_CLIENT_SECRET (в .env или переменных окружения)")
	}
	return cfg, nil
}

func readDotEnv(path string) (map[string]string, error) {
	vals := map[string]string{}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return vals, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: ожидается строка вида KEY=VALUE", path, n)
		}
		vals[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(val), `"'`)
	}
	return vals, sc.Err()
}
```

- [ ] **Step 5: Проверить, что тесты проходят**

Run: `gofmt -l . && go vet ./... && go test ./internal/config/ -v`
Expected: 5 тестов PASS, gofmt ничего не выводит.

- [ ] **Step 6: Коммит**

```bash
git add go.mod internal/config
git commit -m "feat: конфигурация из .env и окружения"
```

### Task 2: Яндекс OAuth — обмен кода и обновление токена

**Files:**
- Create: `internal/yandex/oauth.go`
- Test: `internal/yandex/oauth_test.go`

**Interfaces:**
- Produces:
  - `yandex.Token{AccessToken, RefreshToken string; ExpiresAt time.Time}` (json-теги `access_token`, `refresh_token`, `expires_at`)
  - `yandex.OAuth{ClientID, ClientSecret, AuthorizeURL, TokenURL string; HTTP *http.Client; Now func() time.Time}`
  - `(*OAuth).LoginURL() string`, `(*OAuth).Exchange(ctx, code string) (Token, error)`, `(*OAuth).Refresh(ctx, refreshToken string) (Token, error)`
  - вспомогательная `truncate(b []byte) string` (используется в Task 4).

- [ ] **Step 1: Написать падающие тесты**

`internal/yandex/oauth_test.go`:

```go
package yandex

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)

func oauthServer(t *testing.T, status int, body string) (*httptest.Server, *url.Values) {
	t.Helper()
	got := &url.Values{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		*got = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func TestExchangeSendsFormAndParsesToken(t *testing.T) {
	srv, got := oauthServer(t, 200,
		`{"access_token":"acc","refresh_token":"ref","expires_in":3600,"token_type":"bearer"}`)
	o := &OAuth{ClientID: "cid", ClientSecret: "sec", TokenURL: srv.URL,
		Now: func() time.Time { return fixedNow }}

	tok, err := o.Exchange(context.Background(), "1234567")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	want := url.Values{
		"grant_type": {"authorization_code"}, "code": {"1234567"},
		"client_id": {"cid"}, "client_secret": {"sec"},
	}
	if got.Encode() != want.Encode() {
		t.Errorf("form = %v, want %v", *got, want)
	}
	if tok.AccessToken != "acc" || tok.RefreshToken != "ref" {
		t.Errorf("token = %+v", tok)
	}
	if !tok.ExpiresAt.Equal(fixedNow.Add(time.Hour)) {
		t.Errorf("ExpiresAt = %v, want %v", tok.ExpiresAt, fixedNow.Add(time.Hour))
	}
}

func TestRefreshUsesRefreshGrant(t *testing.T) {
	srv, got := oauthServer(t, 200,
		`{"access_token":"acc2","refresh_token":"ref2","expires_in":10,"token_type":"bearer"}`)
	o := &OAuth{ClientID: "cid", ClientSecret: "sec", TokenURL: srv.URL}

	tok, err := o.Refresh(context.Background(), "ref1")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if got.Get("grant_type") != "refresh_token" || got.Get("refresh_token") != "ref1" {
		t.Errorf("form = %v", *got)
	}
	if tok.AccessToken != "acc2" || tok.RefreshToken != "ref2" {
		t.Errorf("token = %+v", tok)
	}
}

func TestExchangeReturnsOAuthError(t *testing.T) {
	srv, _ := oauthServer(t, 400, `{"error":"invalid_grant","error_description":"Code has expired"}`)
	o := &OAuth{ClientID: "cid", ClientSecret: "sec", TokenURL: srv.URL}

	_, err := o.Exchange(context.Background(), "bad")
	if err == nil || !strings.Contains(err.Error(), "invalid_grant") || !strings.Contains(err.Error(), "Code has expired") {
		t.Fatalf("err = %v, want invalid_grant with description", err)
	}
}

func TestExchangeRejectsNonJSON(t *testing.T) {
	srv, _ := oauthServer(t, 502, `<html>Bad gateway</html>`)
	o := &OAuth{ClientID: "cid", ClientSecret: "sec", TokenURL: srv.URL}

	_, err := o.Exchange(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err = %v, want mention of HTTP 502", err)
	}
}

func TestLoginURL(t *testing.T) {
	o := &OAuth{ClientID: "cid"}
	want := "https://oauth.yandex.ru/authorize?client_id=cid&response_type=code"
	if got := o.LoginURL(); got != want {
		t.Errorf("LoginURL = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/yandex/`
Expected: ошибка компиляции — `undefined: OAuth`.

- [ ] **Step 3: Реализация**

`internal/yandex/oauth.go`:

```go
// Package yandex — клиенты Яндекс OAuth и API умного дома (api.iot.yandex.net).
package yandex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultAuthorizeURL = "https://oauth.yandex.ru/authorize"
	defaultTokenURL     = "https://oauth.yandex.ru/token"
)

type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// OAuth обменивает код подтверждения на токены и обновляет их.
// Пустые AuthorizeURL/TokenURL/HTTP/Now заменяются значениями по умолчанию.
type OAuth struct {
	ClientID     string
	ClientSecret string
	AuthorizeURL string
	TokenURL     string
	HTTP         *http.Client
	Now          func() time.Time
}

// LoginURL — страница Яндекса, где пользователь получает код подтверждения.
func (o *OAuth) LoginURL() string {
	base := o.AuthorizeURL
	if base == "" {
		base = defaultAuthorizeURL
	}
	q := url.Values{"response_type": {"code"}, "client_id": {o.ClientID}}
	return base + "?" + q.Encode()
}

func (o *OAuth) Exchange(ctx context.Context, code string) (Token, error) {
	return o.request(ctx, url.Values{"grant_type": {"authorization_code"}, "code": {code}})
}

func (o *OAuth) Refresh(ctx context.Context, refreshToken string) (Token, error) {
	return o.request(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}})
}

func (o *OAuth) request(ctx context.Context, form url.Values) (Token, error) {
	form.Set("client_id", o.ClientID)
	form.Set("client_secret", o.ClientSecret)

	tokenURL := o.TokenURL
	if tokenURL == "" {
		tokenURL = defaultTokenURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpc := o.HTTP
	if httpc == nil {
		httpc = http.DefaultClient
	}
	resp, err := httpc.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("oauth: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Token{}, fmt.Errorf("oauth: чтение ответа: %w", err)
	}

	var payload struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		ExpiresIn        int64  `json:"expires_in"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Token{}, fmt.Errorf("oauth: неожиданный ответ (HTTP %d): %s", resp.StatusCode, truncate(body))
	}
	if resp.StatusCode != http.StatusOK || payload.Error != "" {
		return Token{}, fmt.Errorf("oauth: %s: %s", payload.Error, payload.ErrorDescription)
	}

	now := time.Now
	if o.Now != nil {
		now = o.Now
	}
	return Token{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		ExpiresAt:    now().Add(time.Duration(payload.ExpiresIn) * time.Second),
	}, nil
}

func truncate(b []byte) string {
	const max = 200
	if len(b) > max {
		return string(b[:max]) + "…"
	}
	return string(b)
}
```

- [ ] **Step 4: Проверить, что тесты проходят**

Run: `gofmt -l . && go vet ./... && go test ./internal/yandex/ -v`
Expected: 5 тестов PASS.

- [ ] **Step 5: Коммит**

```bash
git add internal/yandex
git commit -m "feat: обмен и обновление OAuth-токена Яндекса"
```

---

### Task 3: Хранение токена и менеджер авторизации

**Files:**
- Create: `internal/auth/store.go`
- Create: `internal/auth/manager.go`
- Test: `internal/auth/manager_test.go`

**Interfaces:**
- Consumes: `yandex.Token`, `yandex.OAuth` (Task 2).
- Produces:
  - `auth.ErrNoToken` (ошибка «нет токена»)
  - `auth.Store{Path string}`: `Load() (yandex.Token, bool, error)`, `Save(yandex.Token) error`
  - `auth.NewManager(o *yandex.OAuth, s *Store) (*Manager, error)`
  - `(*Manager).Authorized() bool`, `LoginURL() string`, `Login(ctx, code string) error`,
    `Token(ctx) (string, error)`, `Refresh(ctx) (string, error)`.
  - `Manager` удовлетворяет интерфейсу `yandex.TokenSource` (Task 4) и `server.Auth` (Task 6).

- [ ] **Step 1: Написать падающие тесты**

`internal/auth/manager_test.go`:

```go
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
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/auth/`
Expected: ошибка компиляции — `undefined: Store`, `undefined: NewManager`.

- [ ] **Step 3: Реализация хранилища**

`internal/auth/store.go`:

```go
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
```

- [ ] **Step 4: Реализация менеджера**

`internal/auth/manager.go`:

```go
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
	has, tok := m.has, m.token
	m.mu.Unlock()
	if !has {
		return "", ErrNoToken
	}
	if m.now().Add(refreshBefore).Before(tok.ExpiresAt) {
		return tok.AccessToken, nil
	}
	return m.Refresh(ctx)
}

// Refresh принудительно обновляет токен (например, после 401 от API).
func (m *Manager) Refresh(ctx context.Context) (string, error) {
	m.mu.Lock()
	has, tok := m.has, m.token
	m.mu.Unlock()
	if !has {
		return "", ErrNoToken
	}
	fresh, err := m.oauth.Refresh(ctx, tok.RefreshToken)
	if err != nil {
		return "", err
	}
	if err := m.set(fresh); err != nil {
		return "", err
	}
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
```

- [ ] **Step 5: Проверить, что тесты проходят**

Run: `gofmt -l . && go vet ./... && go test ./internal/auth/ -v`
Expected: 6 тестов PASS.

- [ ] **Step 6: Коммит**

```bash
git add internal/auth
git commit -m "feat: хранение и авто-обновление OAuth-токена"
```

### Task 4: Клиент API умного дома

**Files:**
- Create: `internal/yandex/client.go`
- Test: `internal/yandex/client_test.go`

**Interfaces:**
- Consumes: `truncate` (Task 2).
- Produces:
  - `yandex.TokenSource` interface: `Token(ctx) (string, error)`, `Refresh(ctx) (string, error)`
  - `yandex.APIError{StatusCode int; Message, RequestID string}` с методом `Error()`
  - `yandex.ActionsRequest{Devices []DeviceActions}`, `DeviceActions{ID string; Actions []Action}`, `Action{Type string; State ActionState}`, `ActionState{Instance string; Value any}` (json-теги: `devices`, `id`, `actions`, `type`, `state`, `instance`, `value`)
  - `yandex.Client{BaseURL string; HTTP *http.Client; Tokens TokenSource; Log *slog.Logger}`
  - `(*Client).UserInfo(ctx) (json.RawMessage, error)`, `DeviceActions(ctx, ActionsRequest) (json.RawMessage, error)`, `RunScenario(ctx, id string) (json.RawMessage, error)`.

- [ ] **Step 1: Написать падающие тесты**

`internal/yandex/client_test.go`:

```go
package yandex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeTokens struct {
	token      string
	refreshed  int
	refreshErr error
}

func (f *fakeTokens) Token(context.Context) (string, error) { return f.token, nil }
func (f *fakeTokens) Refresh(context.Context) (string, error) {
	if f.refreshErr != nil {
		return "", f.refreshErr
	}
	f.refreshed++
	f.token = "fresh"
	return f.token, nil
}

type recorded struct {
	method, path, auth, body string
}

func apiServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (*Client, *recorded, *fakeTokens) {
	t.Helper()
	rec := &recorded{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*rec = recorded{method: r.Method, path: r.URL.Path, auth: r.Header.Get("Authorization"), body: string(body)}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	tokens := &fakeTokens{token: "tok"}
	return &Client{BaseURL: srv.URL, Tokens: tokens}, rec, tokens
}

func okJSON(body string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}
}

func TestUserInfoSendsBearerAndReturnsBody(t *testing.T) {
	c, rec, _ := apiServer(t, okJSON(`{"status":"ok","request_id":"r1","devices":[]}`))

	raw, err := c.UserInfo(context.Background())
	if err != nil {
		t.Fatalf("UserInfo: %v", err)
	}
	if rec.method != http.MethodGet || rec.path != "/v1.0/user/info" || rec.auth != "Bearer tok" {
		t.Errorf("request = %+v", *rec)
	}
	if string(raw) != `{"status":"ok","request_id":"r1","devices":[]}` {
		t.Errorf("raw = %s", raw)
	}
}

func TestDeviceActionsPostsJSON(t *testing.T) {
	c, rec, _ := apiServer(t, okJSON(`{"status":"ok","request_id":"r2","devices":[]}`))
	req := ActionsRequest{Devices: []DeviceActions{{
		ID: "d1",
		Actions: []Action{{Type: "devices.capabilities.on_off", State: ActionState{Instance: "on", Value: true}}},
	}}}

	if _, err := c.DeviceActions(context.Background(), req); err != nil {
		t.Fatalf("DeviceActions: %v", err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1.0/devices/actions" {
		t.Errorf("request = %+v", *rec)
	}
	var sent ActionsRequest
	if err := json.Unmarshal([]byte(rec.body), &sent); err != nil {
		t.Fatalf("body %q: %v", rec.body, err)
	}
	if len(sent.Devices) != 1 || sent.Devices[0].ID != "d1" || sent.Devices[0].Actions[0].State.Value != true {
		t.Errorf("sent = %+v", sent)
	}
}

func TestRunScenarioPath(t *testing.T) {
	c, rec, _ := apiServer(t, okJSON(`{"status":"ok","request_id":"r3"}`))
	if _, err := c.RunScenario(context.Background(), "sc 1"); err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1.0/scenarios/sc 1/actions" {
		t.Errorf("request = %+v", *rec)
	}
}

func TestNon200BecomesAPIError(t *testing.T) {
	c, _, _ := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"request_id":"r4","status":"error","message":"device not found"}`)
	})

	_, err := c.UserInfo(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.StatusCode != 404 || apiErr.Message != "device not found" || apiErr.RequestID != "r4" {
		t.Errorf("apiErr = %+v", *apiErr)
	}
}

func TestUnauthorizedTriggersRefreshAndRetry(t *testing.T) {
	c, rec, tokens := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fresh" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"request_id":"r5","status":"error","message":"unauthorized"}`)
			return
		}
		fmt.Fprint(w, `{"status":"ok","request_id":"r6"}`)
	})

	raw, err := c.UserInfo(context.Background())
	if err != nil {
		t.Fatalf("UserInfo: %v", err)
	}
	if tokens.refreshed != 1 || rec.auth != "Bearer fresh" {
		t.Errorf("refreshed=%d, last auth=%q", tokens.refreshed, rec.auth)
	}
	if string(raw) != `{"status":"ok","request_id":"r6"}` {
		t.Errorf("raw = %s", raw)
	}
}

func TestUnauthorizedWithFailedRefreshReturns401(t *testing.T) {
	c, _, tokens := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"request_id":"r7","status":"error","message":"unauthorized"}`)
	})
	tokens.refreshErr = errors.New("oauth: invalid_grant")

	_, err := c.UserInfo(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("err = %v, want APIError 401", err)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/yandex/`
Expected: ошибка компиляции — `undefined: Client`, `undefined: ActionsRequest`.

- [ ] **Step 3: Реализация**

`internal/yandex/client.go`:

```go
package yandex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const DefaultBaseURL = "https://api.iot.yandex.net"

// TokenSource выдаёт действующий access-токен и умеет обновить его после 401.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
	Refresh(ctx context.Context) (string, error)
}

// APIError — ответ API Яндекса со статусом, отличным от 200.
type APIError struct {
	StatusCode int
	Message    string
	RequestID  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("yandex api: HTTP %d: %s (request_id=%s)", e.StatusCode, e.Message, e.RequestID)
}

type ActionsRequest struct {
	Devices []DeviceActions `json:"devices"`
}

type DeviceActions struct {
	ID      string   `json:"id"`
	Actions []Action `json:"actions"`
}

type Action struct {
	Type  string      `json:"type"`
	State ActionState `json:"state"`
}

type ActionState struct {
	Instance string `json:"instance"`
	Value    any    `json:"value"`
}

// Client — клиент api.iot.yandex.net. Ответы отдаются как есть (json.RawMessage),
// чтобы фронтенд получал полную структуру Яндекса без потерь.
type Client struct {
	BaseURL string
	HTTP    *http.Client
	Tokens  TokenSource
	Log     *slog.Logger
}

func (c *Client) UserInfo(ctx context.Context) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, "/v1.0/user/info", nil)
}

func (c *Client) DeviceActions(ctx context.Context, req ActionsRequest) (json.RawMessage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, "/v1.0/devices/actions", body)
}

func (c *Client) RunScenario(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, "/v1.0/scenarios/"+url.PathEscape(id)+"/actions", nil)
}

// do выполняет запрос и при 401 один раз обновляет токен и повторяет его.
func (c *Client) do(ctx context.Context, method, path string, body []byte) (json.RawMessage, error) {
	token, err := c.Tokens.Token(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := c.send(ctx, method, path, body, token)
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized {
		token, rerr := c.Tokens.Refresh(ctx)
		if rerr != nil {
			c.logger().Warn("не удалось обновить токен", "err", rerr)
			return nil, err
		}
		raw, err = c.send(ctx, method, path, body, token)
	}
	return raw, err
}

func (c *Client) send(ctx context.Context, method, path string, body []byte, token string) (json.RawMessage, error) {
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	httpc := c.HTTP
	if httpc == nil {
		httpc = http.DefaultClient
	}
	start := time.Now()
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yandex api: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("yandex api: чтение ответа: %w", err)
	}

	var meta struct {
		RequestID string `json:"request_id"`
		Message   string `json:"message"`
	}
	_ = json.Unmarshal(raw, &meta)
	c.logger().Info("yandex api", "method", method, "path", path,
		"status", resp.StatusCode, "request_id", meta.RequestID, "duration", time.Since(start))

	if resp.StatusCode != http.StatusOK {
		msg := meta.Message
		if msg == "" {
			msg = truncate(raw)
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Message: msg, RequestID: meta.RequestID}
	}
	return json.RawMessage(raw), nil
}

func (c *Client) logger() *slog.Logger {
	if c.Log != nil {
		return c.Log
	}
	return slog.Default()
}
```

- [ ] **Step 4: Проверить, что тесты проходят**

Run: `gofmt -l . && go vet ./... && go test ./internal/yandex/ -v`
Expected: 11 тестов PASS (5 из Task 2 + 6 новых).

- [ ] **Step 5: Коммит**

```bash
git add internal/yandex
git commit -m "feat: клиент API умного дома с повтором после 401"
```

---

### Task 5: Хранилище макросов

**Files:**
- Create: `internal/macros/store.go`
- Test: `internal/macros/store_test.go`

**Interfaces:**
- Consumes: `yandex.ActionsRequest`, `yandex.DeviceActions`, `yandex.Action`, `yandex.ActionState` (Task 4).
- Produces:
  - `macros.ErrNotFound`
  - `macros.Action{DeviceID, Type, Instance string; Value any}` (json: `device_id`, `type`, `instance`, `value`)
  - `macros.Macro{ID, Name string; Actions []Action}` (json: `id`, `name`, `actions`), `(Macro).Validate() error`, `(Macro).ToActionsRequest() yandex.ActionsRequest`
  - `macros.NewStore(path string) (*Store, error)`; `(*Store).List() []Macro`, `Get(id) (Macro, bool)`, `Create(Macro) (Macro, error)`, `Update(Macro) error`, `Delete(id) error`.

- [ ] **Step 1: Написать падающие тесты**

`internal/macros/store_test.go`:

```go
package macros

import (
	"errors"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data", "macros.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s, path
}

func sample() Macro {
	return Macro{Name: "Кино", Actions: []Action{
		{DeviceID: "lamp", Type: "devices.capabilities.on_off", Instance: "on", Value: false},
		{DeviceID: "led", Type: "devices.capabilities.on_off", Instance: "on", Value: true},
		{DeviceID: "led", Type: "devices.capabilities.range", Instance: "brightness", Value: 30},
	}}
}

func TestCreateAssignsIDAndPersists(t *testing.T) {
	s, path := newTestStore(t)

	created, err := s.Create(sample())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("ID не назначен")
	}
	if got := s.List(); len(got) != 1 || got[0].ID != created.ID {
		t.Errorf("List = %+v", got)
	}

	reloaded, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := reloaded.Get(created.ID)
	if !ok || m.Name != "Кино" || len(m.Actions) != 3 {
		t.Errorf("после перезагрузки: ok=%v, macro=%+v", ok, m)
	}
}

func TestListOnEmptyStoreIsEmptySlice(t *testing.T) {
	s, _ := newTestStore(t)
	if got := s.List(); got == nil || len(got) != 0 {
		t.Errorf("List = %#v, want пустой (не nil) срез", got)
	}
}

func TestCreateValidates(t *testing.T) {
	s, _ := newTestStore(t)
	cases := map[string]Macro{
		"без имени":     {Name: " ", Actions: sample().Actions},
		"без действий":  {Name: "x"},
		"без device_id": {Name: "x", Actions: []Action{{Type: "t", Instance: "i"}}},
	}
	for name, m := range cases {
		if _, err := s.Create(m); err == nil {
			t.Errorf("%s: ожидалась ошибка валидации", name)
		}
	}
}

func TestUpdateReplacesAndPersists(t *testing.T) {
	s, path := newTestStore(t)
	created, _ := s.Create(sample())

	created.Name = "Кино 2"
	created.Actions = created.Actions[:1]
	if err := s.Update(created); err != nil {
		t.Fatalf("Update: %v", err)
	}
	reloaded, _ := NewStore(path)
	m, _ := reloaded.Get(created.ID)
	if m.Name != "Кино 2" || len(m.Actions) != 1 {
		t.Errorf("после Update: %+v", m)
	}
}

func TestUpdateUnknownReturnsErrNotFound(t *testing.T) {
	s, _ := newTestStore(t)
	m := sample()
	m.ID = "nope"
	if err := s.Update(m); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestDeleteRemoves(t *testing.T) {
	s, path := newTestStore(t)
	created, _ := s.Create(sample())

	if err := s.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get(created.ID); ok {
		t.Error("макрос остался после Delete")
	}
	if err := s.Delete(created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("повторный Delete err = %v, want ErrNotFound", err)
	}
	reloaded, _ := NewStore(path)
	if len(reloaded.List()) != 0 {
		t.Error("удаление не сохранилось на диск")
	}
}

func TestToActionsRequestGroupsByDevice(t *testing.T) {
	req := sample().ToActionsRequest()
	if len(req.Devices) != 2 {
		t.Fatalf("devices = %d, want 2 (lamp, led)", len(req.Devices))
	}
	if req.Devices[0].ID != "lamp" || len(req.Devices[0].Actions) != 1 {
		t.Errorf("devices[0] = %+v", req.Devices[0])
	}
	led := req.Devices[1]
	if led.ID != "led" || len(led.Actions) != 2 {
		t.Fatalf("devices[1] = %+v", led)
	}
	if led.Actions[1].Type != "devices.capabilities.range" || led.Actions[1].State.Instance != "brightness" || led.Actions[1].State.Value != 30 {
		t.Errorf("led.Actions[1] = %+v", led.Actions[1])
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/macros/`
Expected: ошибка компиляции — `undefined: NewStore`.

- [ ] **Step 3: Реализация**

`internal/macros/store.go`:

```go
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
```

> Замечание к `TestToActionsRequestGroupsByDevice`: значение `30` в `sample()` — `int`, и сравнение `Value != 30` работает, потому что макрос не проходил через JSON. После JSON-раунда числа становятся `float64` — это нормально, Яндексу отправляется JSON.

- [ ] **Step 4: Проверить, что тесты проходят**

Run: `gofmt -l . && go vet ./... && go test ./internal/macros/ -v`
Expected: 7 тестов PASS.

- [ ] **Step 5: Коммит**

```bash
git add internal/macros
git commit -m "feat: хранилище локальных макросов"
```

### Task 6: HTTP-сервер — авторизация, дом, устройства, сценарии

**Files:**
- Create: `internal/server/server.go`
- Create: `internal/server/auth_handlers.go`
- Create: `internal/server/home_handlers.go`
- Test: `internal/server/server_test.go`

**Interfaces:**
- Consumes: `auth.ErrNoToken` (Task 3), `yandex.APIError`, `yandex.ActionsRequest` (Task 4), `macros.Store` (Task 5).
- Produces:
  - `server.HomeAPI` interface: `UserInfo(ctx) (json.RawMessage, error)`, `DeviceActions(ctx, yandex.ActionsRequest) (json.RawMessage, error)`, `RunScenario(ctx, id string) (json.RawMessage, error)` — его реализует `*yandex.Client`.
  - `server.Auth` interface: `Authorized() bool`, `LoginURL() string`, `Login(ctx, code string) error` — его реализует `*auth.Manager`.
  - `server.Server{Auth Auth; Home HomeAPI; Macros *macros.Store; UI fs.FS; Log *slog.Logger}`, `(*Server).Handler() http.Handler`.
  - Вспомогательные (используются в Task 7): `writeJSON(w, status, v)`, `writeRaw(w, raw)`, `(*Server).writeError(w, err)`, тип `errorBody{Error, Message, RequestID string}`.
  - В тестах: `fakeHome`, `fakeAuth`, `newTestServer(t, home, auth)` — переиспользуются в Task 7.

HTTP-контракт (JSON, все ошибки — `{"error": "<код>", "message": "<по-русски>", "request_id": "…"?}`):

| Запрос | Ответ |
|---|---|
| `GET /api/auth/status` | 200 `{"authorized": bool, "login_url": "https://oauth.yandex.ru/authorize?…"}` |
| `POST /api/auth/code` `{"code":"…"}` | 200 `{"authorized": true}`; 400 `bad_request` / `login_failed` |
| `GET /api/home` | 200 — тело `user/info` как есть; 401 `unauthorized`; 502 `yandex_error` |
| `POST /api/devices/actions` (формат Яндекса) | 200 — ответ Яндекса как есть; 400 / 401 / 502 |
| `POST /api/scenarios/{id}/run` | 200 — ответ Яндекса как есть; 401 / 502 |

- [ ] **Step 1: Написать падающие тесты**

`internal/server/server_test.go`:

```go
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"smarthome/internal/auth"
	"smarthome/internal/macros"
	"smarthome/internal/yandex"
)

type fakeHome struct {
	userInfo    json.RawMessage
	actionsResp json.RawMessage
	err         error
	gotActions  *yandex.ActionsRequest
	gotScenario string
}

func (f *fakeHome) UserInfo(context.Context) (json.RawMessage, error) {
	return f.userInfo, f.err
}

func (f *fakeHome) DeviceActions(_ context.Context, req yandex.ActionsRequest) (json.RawMessage, error) {
	f.gotActions = &req
	return f.actionsResp, f.err
}

func (f *fakeHome) RunScenario(_ context.Context, id string) (json.RawMessage, error) {
	f.gotScenario = id
	return json.RawMessage(`{"status":"ok","request_id":"rs"}`), f.err
}

type fakeAuth struct {
	authorized bool
	loginErr   error
	gotCode    string
}

func (f *fakeAuth) Authorized() bool { return f.authorized }
func (f *fakeAuth) LoginURL() string { return "https://oauth.yandex.ru/authorize?client_id=cid&response_type=code" }
func (f *fakeAuth) Login(_ context.Context, code string) error {
	f.gotCode = code
	if f.loginErr != nil {
		return f.loginErr
	}
	f.authorized = true
	return nil
}

func newTestServer(t *testing.T, home *fakeHome, a *fakeAuth) *httptest.Server {
	t.Helper()
	store, err := macros.NewStore(filepath.Join(t.TempDir(), "macros.json"))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer((&Server{Auth: a, Home: home, Macros: store}).Handler())
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, method, url, body string) (int, string) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, url, rdr)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func decode(t *testing.T, body string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), v); err != nil {
		t.Fatalf("json %q: %v", body, err)
	}
}

func TestAuthStatus(t *testing.T) {
	srv := newTestServer(t, &fakeHome{}, &fakeAuth{authorized: false})
	code, body := do(t, "GET", srv.URL+"/api/auth/status", "")
	if code != 200 {
		t.Fatalf("status %d: %s", code, body)
	}
	var got struct {
		Authorized bool   `json:"authorized"`
		LoginURL   string `json:"login_url"`
	}
	decode(t, body, &got)
	if got.Authorized || !strings.HasPrefix(got.LoginURL, "https://oauth.yandex.ru/authorize?") {
		t.Errorf("got %+v", got)
	}
}

func TestAuthCodeLogsIn(t *testing.T) {
	a := &fakeAuth{}
	srv := newTestServer(t, &fakeHome{}, a)
	code, body := do(t, "POST", srv.URL+"/api/auth/code", `{"code":" 1234567 "}`)
	if code != 200 || a.gotCode != "1234567" {
		t.Errorf("status %d, gotCode %q, body %s", code, a.gotCode, body)
	}
}

func TestAuthCodeRejectsEmptyAndFailed(t *testing.T) {
	a := &fakeAuth{loginErr: errors.New("oauth: invalid_grant: Code has expired")}
	srv := newTestServer(t, &fakeHome{}, a)

	if code, _ := do(t, "POST", srv.URL+"/api/auth/code", `{"code":""}`); code != 400 {
		t.Errorf("empty code: status %d, want 400", code)
	}
	code, body := do(t, "POST", srv.URL+"/api/auth/code", `{"code":"bad"}`)
	if code != 400 || !strings.Contains(body, "Code has expired") {
		t.Errorf("failed login: status %d body %s", code, body)
	}
}

func TestHomeProxiesUserInfo(t *testing.T) {
	raw := `{"status":"ok","request_id":"r1","rooms":[],"devices":[]}`
	srv := newTestServer(t, &fakeHome{userInfo: json.RawMessage(raw)}, &fakeAuth{authorized: true})
	code, body := do(t, "GET", srv.URL+"/api/home", "")
	if code != 200 || body != raw {
		t.Errorf("status %d body %s", code, body)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"нет токена", auth.ErrNoToken, 401},
		{"401 от Яндекса", &yandex.APIError{StatusCode: 401, Message: "unauthorized", RequestID: "r"}, 401},
		{"ошибка Яндекса", &yandex.APIError{StatusCode: 404, Message: "not found", RequestID: "r"}, 502},
		{"прочее", errors.New("boom"), 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer(t, &fakeHome{err: tc.err}, &fakeAuth{authorized: true})
			code, body := do(t, "GET", srv.URL+"/api/home", "")
			if code != tc.want {
				t.Errorf("status %d, want %d (%s)", code, tc.want, body)
			}
			var eb errorBody
			decode(t, body, &eb)
			if eb.Error == "" || eb.Message == "" {
				t.Errorf("error body = %+v", eb)
			}
		})
	}
}

func TestDeviceActionsForwardsRequest(t *testing.T) {
	home := &fakeHome{actionsResp: json.RawMessage(`{"status":"ok","request_id":"r2","devices":[]}`)}
	srv := newTestServer(t, home, &fakeAuth{authorized: true})
	body := `{"devices":[{"id":"d1","actions":[{"type":"devices.capabilities.on_off","state":{"instance":"on","value":true}}]}]}`

	code, resp := do(t, "POST", srv.URL+"/api/devices/actions", body)
	if code != 200 || !strings.Contains(resp, `"request_id":"r2"`) {
		t.Fatalf("status %d body %s", code, resp)
	}
	if home.gotActions == nil || home.gotActions.Devices[0].ID != "d1" || home.gotActions.Devices[0].Actions[0].State.Value != true {
		t.Errorf("forwarded = %+v", home.gotActions)
	}
}

func TestDeviceActionsRejectsBadBody(t *testing.T) {
	srv := newTestServer(t, &fakeHome{}, &fakeAuth{authorized: true})
	for _, body := range []string{`not json`, `{"devices":[]}`} {
		if code, _ := do(t, "POST", srv.URL+"/api/devices/actions", body); code != 400 {
			t.Errorf("body %q: status %d, want 400", body, code)
		}
	}
}

func TestRunScenarioUsesPathID(t *testing.T) {
	home := &fakeHome{}
	srv := newTestServer(t, home, &fakeAuth{authorized: true})
	code, _ := do(t, "POST", srv.URL+"/api/scenarios/sc-1/run", "")
	if code != 200 || home.gotScenario != "sc-1" {
		t.Errorf("status %d, scenario %q", code, home.gotScenario)
	}
}

func TestUnknownAPIRouteIs404(t *testing.T) {
	srv := newTestServer(t, &fakeHome{}, &fakeAuth{})
	if code, _ := do(t, "GET", srv.URL+"/api/nope", ""); code != 404 {
		t.Errorf("status %d, want 404", code)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/server/`
Expected: ошибка компиляции — `undefined: Server`.

- [ ] **Step 3: Реализация ядра сервера**

`internal/server/server.go`:

```go
// Package server — HTTP API для фронтенда и раздача собранного UI.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"

	"smarthome/internal/auth"
	"smarthome/internal/macros"
	"smarthome/internal/yandex"
)

// HomeAPI — то, что серверу нужно от клиента Яндекса; реализуется *yandex.Client.
type HomeAPI interface {
	UserInfo(ctx context.Context) (json.RawMessage, error)
	DeviceActions(ctx context.Context, req yandex.ActionsRequest) (json.RawMessage, error)
	RunScenario(ctx context.Context, id string) (json.RawMessage, error)
}

// Auth — то, что серверу нужно от менеджера токенов; реализуется *auth.Manager.
type Auth interface {
	Authorized() bool
	LoginURL() string
	Login(ctx context.Context, code string) error
}

type Server struct {
	Auth   Auth
	Home   HomeAPI
	Macros *macros.Store
	UI     fs.FS // корень собранного фронтенда; nil — UI не раздаётся
	Log    *slog.Logger
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/status", s.authStatus)
	mux.HandleFunc("POST /api/auth/code", s.authCode)
	mux.HandleFunc("GET /api/home", s.home)
	mux.HandleFunc("POST /api/devices/actions", s.deviceActions)
	mux.HandleFunc("POST /api/scenarios/{id}/run", s.runScenario)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, errorBody{Error: "not_found", Message: "нет такого метода API"})
	})
	if s.UI != nil {
		mux.Handle("/", http.FileServerFS(s.UI))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "UI не собран: выполните `cd web && npm run build` и пересоберите бинарник", http.StatusServiceUnavailable)
		})
	}
	return mux
}

type errorBody struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeRaw отдаёт ответ Яндекса как есть.
func writeRaw(w http.ResponseWriter, raw json.RawMessage) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

// writeError: нет токена или 401 от Яндекса → 401; прочие ошибки API Яндекса → 502; остальное → 500.
func (s *Server) writeError(w http.ResponseWriter, err error) {
	var apiErr *yandex.APIError
	switch {
	case errors.Is(err, auth.ErrNoToken):
		writeJSON(w, http.StatusUnauthorized, errorBody{Error: "unauthorized", Message: err.Error()})
	case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized:
		writeJSON(w, http.StatusUnauthorized, errorBody{Error: "unauthorized",
			Message: "токен Яндекса недействителен, войдите заново", RequestID: apiErr.RequestID})
	case errors.As(err, &apiErr):
		writeJSON(w, http.StatusBadGateway, errorBody{Error: "yandex_error",
			Message: "Яндекс ответил ошибкой: " + apiErr.Message, RequestID: apiErr.RequestID})
	default:
		s.logger().Error("внутренняя ошибка", "err", err)
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "internal", Message: err.Error()})
	}
}

func (s *Server) logger() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}
```

- [ ] **Step 4: Обработчики авторизации**

`internal/server/auth_handlers.go`:

```go
package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"authorized": s.Auth.Authorized(),
		"login_url":  s.Auth.LoginURL(),
	})
}

func (s *Server) authCode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Code) == "" {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: "введите код подтверждения"})
		return
	}
	if err := s.Auth.Login(r.Context(), strings.TrimSpace(body.Code)); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "login_failed", Message: "не удалось войти: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"authorized": true})
}
```

- [ ] **Step 5: Обработчики дома, устройств, сценариев**

`internal/server/home_handlers.go`:

```go
package server

import (
	"encoding/json"
	"net/http"

	"smarthome/internal/yandex"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	raw, err := s.Home.UserInfo(r.Context())
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeRaw(w, raw)
}

func (s *Server) deviceActions(w http.ResponseWriter, r *http.Request) {
	var req yandex.ActionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Devices) == 0 {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request",
			Message: "ожидается {\"devices\":[{\"id\":…,\"actions\":[…]}]}"})
		return
	}
	raw, err := s.Home.DeviceActions(r.Context(), req)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeRaw(w, raw)
}

func (s *Server) runScenario(w http.ResponseWriter, r *http.Request) {
	raw, err := s.Home.RunScenario(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeRaw(w, raw)
}
```

- [ ] **Step 6: Проверить, что тесты проходят**

Run: `gofmt -l . && go vet ./... && go test ./internal/server/ -v`
Expected: 9 тестов PASS (включая 4 подтеста TestErrorMapping).

- [ ] **Step 7: Коммит**

```bash
git add internal/server
git commit -m "feat: HTTP API — авторизация, дом, устройства, сценарии"
```

---

### Task 7: HTTP-обработчики макросов

**Files:**
- Create: `internal/server/macro_handlers.go`
- Modify: `internal/server/server.go` (добавить маршруты в `Handler()`)
- Test: `internal/server/macro_handlers_test.go`

**Interfaces:**
- Consumes: `macros.Store`, `macros.Macro`, `macros.ErrNotFound` (Task 5); `writeJSON`, `writeRaw`, `writeError`, `errorBody`, тестовые `fakeHome`, `fakeAuth`, `newTestServer`, `do`, `decode` (Task 6).

HTTP-контракт:

| Запрос | Ответ |
|---|---|
| `GET /api/macros` | 200 `[{"id","name","actions":[{"device_id","type","instance","value"}]}]` |
| `POST /api/macros` `{name, actions}` | 201 макрос с `id`; 400 `bad_request` при невалидном теле |
| `PUT /api/macros/{id}` `{name, actions}` | 200 макрос; 404 `not_found`; 400 |
| `DELETE /api/macros/{id}` | 204; 404 |
| `POST /api/macros/{id}/run` | 200 — ответ Яндекса на `devices/actions` как есть; 404; 401/502 |

- [ ] **Step 1: Написать падающие тесты**

`internal/server/macro_handlers_test.go`:

```go
package server

import (
	"encoding/json"
	"testing"

	"smarthome/internal/macros"
)

const macroJSON = `{"name":"Кино","actions":[
  {"device_id":"lamp","type":"devices.capabilities.on_off","instance":"on","value":false},
  {"device_id":"led","type":"devices.capabilities.on_off","instance":"on","value":true},
  {"device_id":"led","type":"devices.capabilities.range","instance":"brightness","value":30}
]}`

func TestMacroCRUD(t *testing.T) {
	srv := newTestServer(t, &fakeHome{}, &fakeAuth{authorized: true})

	code, body := do(t, "GET", srv.URL+"/api/macros", "")
	if code != 200 || body != "[]\n" {
		t.Fatalf("empty list: %d %q", code, body)
	}

	code, body = do(t, "POST", srv.URL+"/api/macros", macroJSON)
	if code != 201 {
		t.Fatalf("create: %d %s", code, body)
	}
	var created macros.Macro
	decode(t, body, &created)
	if created.ID == "" || created.Name != "Кино" || len(created.Actions) != 3 {
		t.Fatalf("created = %+v", created)
	}

	code, body = do(t, "PUT", srv.URL+"/api/macros/"+created.ID, `{"name":"Кино 2","actions":[{"device_id":"lamp","type":"devices.capabilities.on_off","instance":"on","value":true}]}`)
	if code != 200 {
		t.Fatalf("update: %d %s", code, body)
	}
	var updated macros.Macro
	decode(t, body, &updated)
	if updated.ID != created.ID || updated.Name != "Кино 2" || len(updated.Actions) != 1 {
		t.Errorf("updated = %+v", updated)
	}

	code, body = do(t, "GET", srv.URL+"/api/macros", "")
	var list []macros.Macro
	decode(t, body, &list)
	if code != 200 || len(list) != 1 || list[0].Name != "Кино 2" {
		t.Errorf("list = %+v", list)
	}

	if code, _ = do(t, "DELETE", srv.URL+"/api/macros/"+created.ID, ""); code != 204 {
		t.Errorf("delete: %d", code)
	}
	if code, _ = do(t, "DELETE", srv.URL+"/api/macros/"+created.ID, ""); code != 404 {
		t.Errorf("second delete: %d, want 404", code)
	}
}

func TestMacroValidationAndNotFound(t *testing.T) {
	srv := newTestServer(t, &fakeHome{}, &fakeAuth{authorized: true})
	if code, _ := do(t, "POST", srv.URL+"/api/macros", `{"name":"","actions":[]}`); code != 400 {
		t.Errorf("invalid create: %d, want 400", code)
	}
	if code, _ := do(t, "POST", srv.URL+"/api/macros", `garbage`); code != 400 {
		t.Errorf("garbage create: %d, want 400", code)
	}
	if code, _ := do(t, "PUT", srv.URL+"/api/macros/nope", macroJSON); code != 404 {
		t.Errorf("update unknown: %d, want 404", code)
	}
	if code, _ := do(t, "POST", srv.URL+"/api/macros/nope/run", ""); code != 404 {
		t.Errorf("run unknown: %d, want 404", code)
	}
}

func TestRunMacroSendsGroupedActions(t *testing.T) {
	home := &fakeHome{actionsResp: json.RawMessage(`{"status":"ok","request_id":"rm","devices":[]}`)}
	srv := newTestServer(t, home, &fakeAuth{authorized: true})
	_, body := do(t, "POST", srv.URL+"/api/macros", macroJSON)
	var created macros.Macro
	decode(t, body, &created)

	code, resp := do(t, "POST", srv.URL+"/api/macros/"+created.ID+"/run", "")
	if code != 200 || resp != `{"status":"ok","request_id":"rm","devices":[]}` {
		t.Fatalf("run: %d %s", code, resp)
	}
	if home.gotActions == nil || len(home.gotActions.Devices) != 2 {
		t.Fatalf("gotActions = %+v", home.gotActions)
	}
	led := home.gotActions.Devices[1]
	if led.ID != "led" || len(led.Actions) != 2 || led.Actions[1].State.Value != float64(30) {
		t.Errorf("led = %+v", led)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/server/`
Expected: FAIL — `TestMacroCRUD` получает 404 вместо 200 (маршрутов нет).

- [ ] **Step 3: Реализация обработчиков**

`internal/server/macro_handlers.go`:

```go
package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"smarthome/internal/macros"
)

func (s *Server) listMacros(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Macros.List())
}

func (s *Server) createMacro(w http.ResponseWriter, r *http.Request) {
	var m macros.Macro
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: "некорректный JSON макроса"})
		return
	}
	created, err := s.Macros.Create(m)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateMacro(w http.ResponseWriter, r *http.Request) {
	var m macros.Macro
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: "некорректный JSON макроса"})
		return
	}
	m.ID = r.PathValue("id")
	if err := s.Macros.Update(m); err != nil {
		s.writeMacroError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) deleteMacro(w http.ResponseWriter, r *http.Request) {
	if err := s.Macros.Delete(r.PathValue("id")); err != nil {
		s.writeMacroError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) runMacro(w http.ResponseWriter, r *http.Request) {
	m, ok := s.Macros.Get(r.PathValue("id"))
	if !ok {
		s.writeMacroError(w, macros.ErrNotFound)
		return
	}
	raw, err := s.Home.DeviceActions(r.Context(), m.ToActionsRequest())
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeRaw(w, raw)
}

func (s *Server) writeMacroError(w http.ResponseWriter, err error) {
	if errors.Is(err, macros.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorBody{Error: "not_found", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
}
```

- [ ] **Step 4: Добавить маршруты**

В `internal/server/server.go`, в `Handler()` после строки с `POST /api/scenarios/{id}/run` добавить:

```go
	mux.HandleFunc("GET /api/macros", s.listMacros)
	mux.HandleFunc("POST /api/macros", s.createMacro)
	mux.HandleFunc("PUT /api/macros/{id}", s.updateMacro)
	mux.HandleFunc("DELETE /api/macros/{id}", s.deleteMacro)
	mux.HandleFunc("POST /api/macros/{id}/run", s.runMacro)
```

- [ ] **Step 5: Проверить, что тесты проходят**

Run: `gofmt -l . && go vet ./... && go test ./... `
Expected: все пакеты `ok`.

- [ ] **Step 6: Коммит**

```bash
git add internal/server
git commit -m "feat: HTTP API макросов"
```

### Task 8: Точка входа, встраивание UI, сборка

**Files:**
- Create: `main.go`
- Create: `web/embed.go`
- Create: `Makefile`
- Create: `.env.example`
- Create: `README.md`
- Modify: `.gitignore`

**Interfaces:**
- Consumes: `config.Load` (Task 1), `yandex.OAuth`, `yandex.Client` (Tasks 2, 4), `auth.NewManager`, `auth.Store` (Task 3), `macros.NewStore` (Task 5), `server.Server` (Task 6).
- Produces: `web.Dist embed.FS` (содержимое `web/dist`), бинарник `smart-house`.

Особенность `go:embed`: каталог `web/dist` должен существовать и быть непустым на момент `go build`/`go test`, иначе компиляция падает с «no matching files found». Каталог не хранится в git — перед сборкой Go либо собирают фронтенд (`make web`), либо создают заглушку (шаг 6).

- [ ] **Step 1: Встраивание фронтенда**

`web/embed.go`:

```go
// Package web содержит собранный Vite-фронтенд (каталог dist), вшиваемый в бинарник.
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
```

- [ ] **Step 2: Точка входа**

`main.go`:

```go
// Веб-пульт умного дома Яндекса: локальный сервер, раздающий UI и проксирующий API Яндекса.
package main

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"smarthome/internal/auth"
	"smarthome/internal/config"
	"smarthome/internal/macros"
	"smarthome/internal/server"
	"smarthome/internal/yandex"
	"smarthome/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("завершение с ошибкой", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load(".env")
	if err != nil {
		return err
	}

	oauth := &yandex.OAuth{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret}
	tokens, err := auth.NewManager(oauth, &auth.Store{Path: filepath.Join(cfg.DataDir, "token.json")})
	if err != nil {
		return err
	}
	macroStore, err := macros.NewStore(filepath.Join(cfg.DataDir, "macros.json"))
	if err != nil {
		return err
	}

	ui, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		return err
	}
	if _, err := fs.Stat(ui, "index.html"); err != nil {
		logger.Warn("фронтенд не собран (нет web/dist/index.html) — выполните `make web` и пересоберите")
		ui = nil
	}

	srv := &server.Server{
		Auth:   tokens,
		Home:   &yandex.Client{Tokens: tokens, Log: logger},
		Macros: macroStore,
		UI:     ui,
		Log:    logger,
	}

	addr := "127.0.0.1:" + cfg.Port
	logger.Info("сервер запущен", "url", "http://localhost:"+cfg.Port, "authorized", tokens.Authorized())
	return http.ListenAndServe(addr, srv.Handler())
}
```

- [ ] **Step 3: Makefile**

`Makefile` (отступы — табуляция):

```make
.PHONY: build web test run clean

# Собрать фронтенд и бинарник.
build: web
	go build -o smart-house .

# Собрать фронтенд в web/dist (нужен Node).
web:
	cd web && npm install && npm run build

test:
	gofmt -l . && go vet ./... && go test ./...

run:
	go run .

clean:
	rm -rf smart-house web/dist
```

- [ ] **Step 4: Образец конфигурации и README**

`.env.example`:

```
# Приложение на https://oauth.yandex.ru (права iot:view, iot:control)
YANDEX_CLIENT_ID=
YANDEX_CLIENT_SECRET=
# Необязательно
PORT=8080
DATA_DIR=data
```

`README.md`:

```markdown
# Веб-пульт умного дома Яндекса

Локальное приложение для управления устройствами умного дома Яндекса с ноутбука:
один Go-бинарник со встроенным веб-интерфейсом.

## Требования

- Go 1.26+
- Node 20+ (только для сборки фронтенда)
- OAuth-приложение на https://oauth.yandex.ru с правами `iot:view` и `iot:control`

## Запуск

1. Скопировать `.env.example` в `.env` и вписать `YANDEX_CLIENT_ID` / `YANDEX_CLIENT_SECRET`.
2. `make build` — соберёт фронтенд и бинарник `smart-house`.
3. `./smart-house` и открыть http://localhost:8080.
4. При первом запуске нажать «Открыть страницу Яндекса», разрешить доступ,
   скопировать показанный код и вставить в форму. Токен сохранится в `data/token.json`.

## Разработка

- `make test` — форматирование, vet и тесты Go.
- Фронтенд с горячей перезагрузкой: `go run .` в одном терминале,
  `cd web && npm run dev` в другом (Vite проксирует `/api` на :8080).

## Ограничения

- Сервер слушает только `127.0.0.1`.
- API Яндекса не умеет создавать сценарии — вместо этого есть локальные макросы
  (`data/macros.json`), выполняемые через `devices/actions`.
```

- [ ] **Step 5: Обновить .gitignore**

Заменить содержимое `.gitignore` на:

```
.env
data/
web/node_modules/
web/dist/
smart-house
```

- [ ] **Step 6: Заглушка dist и проверка сборки**

```bash
mkdir -p web/dist && echo '<!doctype html><title>UI не собран</title><p>Выполните make web' > web/dist/index.html
gofmt -l . && go vet ./... && go test ./... && go build -o smart-house .
```

Expected: всё чисто, бинарник `smart-house` создан.

- [ ] **Step 7: Дымовой запуск без учётных данных**

```bash
YANDEX_CLIENT_ID=x YANDEX_CLIENT_SECRET=y PORT=18080 DATA_DIR=/tmp/smart-house-smoke ./smart-house & PID=$!
sleep 1
curl -s localhost:18080/api/auth/status
curl -s -o /dev/null -w '%{http_code}\n' localhost:18080/
kill $PID
```

Expected: `{"authorized":false,"login_url":"https://oauth.yandex.ru/authorize?client_id=x&response_type=code"}` и `200` (заглушка index.html).

- [ ] **Step 8: Коммит**

```bash
git add main.go web/embed.go Makefile .env.example README.md .gitignore
git commit -m "feat: точка входа, встраивание UI, сборка"
```

---

### Task 9: Фронтенд — каркас Vite и вход через Яндекс

**Files:**
- Create: `web/package.json`
- Create: `web/vite.config.js`
- Create: `web/index.html`
- Create: `web/src/main.jsx`
- Create: `web/src/api.js`
- Create: `web/src/App.jsx`
- Create: `web/src/Login.jsx`
- Create: `web/src/Home.jsx` (заглушка, полная версия в Task 10)
- Create: `web/src/styles.css`

**Interfaces:**
- Consumes: HTTP-контракт `/api/auth/*` (Task 6).
- Produces: модуль `api` (`web/src/api.js`) с функциями `authStatus()`, `login(code)`, `home()`, `deviceActions(devices)`, `runScenario(id)`, `macros()`, `createMacro(m)`, `updateMacro(m)`, `deleteMacro(id)`, `runMacro(id)` — все возвращают Promise с распарсенным JSON и бросают `Error` с полями `status` (HTTP-код) и `code` (поле `error` из тела). Компонент `<Home onUnauthorized />`.

- [ ] **Step 1: package.json и конфиг Vite**

`web/package.json`:

```json
{
  "name": "smart-house-web",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^19.1.0",
    "react-dom": "^19.1.0"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^5.0.0",
    "vite": "^7.1.0"
  }
}
```

`web/vite.config.js`:

```js
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: { '/api': 'http://localhost:8080' },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
```

Установить зависимости: `cd web && npm install` (создаст `package-lock.json` — его коммитим).

- [ ] **Step 2: index.html и точка входа**

`web/index.html`:

```html
<!doctype html>
<html lang="ru">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Умный дом</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.jsx"></script>
  </body>
</html>
```

`web/src/main.jsx`:

```jsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './styles.css'

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
```

- [ ] **Step 3: Обёртка над API**

`web/src/api.js`:

```js
async function request(path, options = {}) {
  const res = await fetch(path, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
  })
  const text = await res.text()
  let data = null
  try {
    data = text ? JSON.parse(text) : null
  } catch {
    data = null
  }
  if (!res.ok) {
    const err = new Error(data?.message || `Ошибка HTTP ${res.status}`)
    err.status = res.status
    err.code = data?.error
    throw err
  }
  return data
}

const json = (method, body) => ({ method, body: JSON.stringify(body) })

export const api = {
  authStatus: () => request('/api/auth/status'),
  login: (code) => request('/api/auth/code', json('POST', { code })),
  home: () => request('/api/home'),
  deviceActions: (devices) => request('/api/devices/actions', json('POST', { devices })),
  runScenario: (id) => request(`/api/scenarios/${encodeURIComponent(id)}/run`, { method: 'POST' }),
  macros: () => request('/api/macros'),
  createMacro: (m) => request('/api/macros', json('POST', m)),
  updateMacro: (m) => request(`/api/macros/${encodeURIComponent(m.id)}`, json('PUT', m)),
  deleteMacro: (id) => request(`/api/macros/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  runMacro: (id) => request(`/api/macros/${encodeURIComponent(id)}/run`, { method: 'POST' }),
}
```

- [ ] **Step 4: App, Login и заглушка Home**

`web/src/App.jsx`:

```jsx
import { useCallback, useEffect, useState } from 'react'
import { api } from './api'
import Login from './Login'
import Home from './Home'

export default function App() {
  const [auth, setAuth] = useState(null) // null — ещё не проверяли

  const refreshAuth = useCallback(() => {
    api
      .authStatus()
      .then(setAuth)
      .catch((e) => setAuth({ authorized: false, login_url: '', error: e.message }))
  }, [])

  useEffect(() => {
    refreshAuth()
  }, [refreshAuth])

  if (auth === null) return <p className="muted center">Загрузка…</p>
  if (!auth.authorized) return <Login loginUrl={auth.login_url} error={auth.error} onLoggedIn={refreshAuth} />
  return <Home onUnauthorized={refreshAuth} />
}
```

`web/src/Login.jsx`:

```jsx
import { useState } from 'react'
import { api } from './api'

export default function Login({ loginUrl, error: initialError, onLoggedIn }) {
  const [code, setCode] = useState('')
  const [error, setError] = useState(initialError || '')
  const [busy, setBusy] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await api.login(code.trim())
      onLoggedIn()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login card">
      <h1>Вход через Яндекс</h1>
      <ol>
        <li>
          <a href={loginUrl} target="_blank" rel="noreferrer">
            Открыть страницу Яндекса
          </a>{' '}
          и разрешить доступ.
        </li>
        <li>Скопировать показанный код подтверждения и вставить его сюда.</li>
      </ol>
      <form onSubmit={submit} className="row">
        <input
          value={code}
          onChange={(e) => setCode(e.target.value)}
          placeholder="Код подтверждения"
          autoFocus
        />
        <button type="submit" disabled={busy || !code.trim()}>
          {busy ? 'Проверяем…' : 'Войти'}
        </button>
      </form>
      {error && <p className="error">{error}</p>}
    </div>
  )
}
```

`web/src/Home.jsx` (временная заглушка):

```jsx
export default function Home() {
  return <p className="center">Авторизация прошла. Устройства появятся в следующем шаге.</p>
}
```

- [ ] **Step 5: Стили**

`web/src/styles.css`:

```css
:root {
  --bg: #f4f5f7;
  --card: #ffffff;
  --text: #1c1e21;
  --muted: #6b7280;
  --accent: #2563eb;
  --danger: #dc2626;
  --ok: #16a34a;
  --border: #e5e7eb;
  --radius: 12px;
}

* { box-sizing: border-box; }

body {
  margin: 0;
  font: 15px/1.45 -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--bg);
  color: var(--text);
}

h1, h2, h3 { margin: 0 0 12px; }
h1 { font-size: 22px; }
h2 { font-size: 18px; }
h3 { font-size: 15px; }

.center { text-align: center; padding: 40px 16px; }
.muted { color: var(--muted); }
.error { color: var(--danger); margin: 8px 0 0; }
.ok { color: var(--ok); }

.card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px;
}

.login { max-width: 460px; margin: 60px auto; }
.login ol { padding-left: 20px; }

.row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.row.between { justify-content: space-between; }

input, select, button {
  font: inherit;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--text);
}
input[type='range'] { padding: 0; flex: 1; min-width: 120px; }
input[type='color'] { padding: 2px; width: 44px; height: 34px; }
input[type='checkbox'] { width: 20px; height: 20px; accent-color: var(--accent); }

button { cursor: pointer; }
button.primary { background: var(--accent); color: #fff; border-color: var(--accent); }
button.danger { color: var(--danger); }
button.active { background: var(--accent); color: #fff; border-color: var(--accent); }
button:disabled { opacity: 0.5; cursor: default; }

.app { max-width: 1100px; margin: 0 auto; padding: 16px; }
.app > header {
  display: flex; justify-content: space-between; align-items: center;
  gap: 12px; flex-wrap: wrap; margin-bottom: 16px;
}
nav { display: flex; gap: 8px; }

.room { margin-bottom: 24px; }
.devices {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.device .type { font-size: 12px; color: var(--muted); }
.device .control { display: flex; align-items: center; gap: 8px; margin-top: 10px; }
.device .control label { min-width: 110px; color: var(--muted); font-size: 13px; }
.device .value { min-width: 48px; text-align: right; font-variant-numeric: tabular-nums; }
.device .props { margin-top: 10px; display: flex; flex-wrap: wrap; gap: 6px 14px; font-size: 14px; }

.list { display: grid; gap: 8px; }
.list .item { display: flex; justify-content: space-between; align-items: center; gap: 8px; }

.editor .action { display: grid; grid-template-columns: 1fr 1fr 1fr auto; gap: 8px; margin-bottom: 8px; }
@media (max-width: 700px) { .editor .action { grid-template-columns: 1fr; } }
```

- [ ] **Step 6: Собрать и проверить**

```bash
cd web && npm run build && cd .. && go build -o smart-house . && \
YANDEX_CLIENT_ID=x YANDEX_CLIENT_SECRET=y PORT=18080 DATA_DIR=/tmp/smart-house-smoke ./smart-house & PID=$!
sleep 1
curl -s localhost:18080/ | grep -o '<title>[^<]*</title>'
kill $PID
```

Expected: `npm run build` без ошибок, в HTML — `<title>Умный дом</title>`. Открыть http://localhost:18080 в браузере: форма «Вход через Яндекс» со ссылкой на oauth.yandex.ru и полем для кода; ввод произвольного кода показывает сообщение об ошибке от сервера (с фейковыми учётными данными вход невозможен — это ожидаемо).

- [ ] **Step 7: Коммит**

```bash
git add web/package.json web/package-lock.json web/vite.config.js web/index.html web/src
git commit -m "feat: каркас фронтенда и форма входа через Яндекс"
```

### Task 10: Фронтенд — дашборд устройств

**Files:**
- Create: `web/src/labels.js`
- Create: `web/src/color.js`
- Create: `web/src/useHome.js`
- Create: `web/src/Dashboard.jsx`
- Create: `web/src/DeviceCard.jsx`
- Create: `web/src/Controls.jsx`
- Modify: `web/src/Home.jsx` (заменить заглушку)

**Interfaces:**
- Consumes: `api.home`, `api.deviceActions` (Task 9); структура `user/info` и `devices/actions` (см. раздел «Что важно знать об API Яндекса»).
- Produces (используются в Task 11): `labels.js` — `label(instance)`, `unit(u)`, `modeLabel(m)`, `eventLabel(e)`, `typeLabel(type)`, `actionErrors(resp) → string[]`; `color.js` — `hexToRgbInt`, `rgbIntToHex`, `hexToHsv`, `hsvToHex`; хук `useHome(onUnauthorized) → {home, error, reload}`; `<Home>` рендерит вкладку «Сценарии» через компонент `Scenarios` (в этой задаче — заглушка внутри `Home.jsx`).

- [ ] **Step 1: Подписи и разбор результата действий**

`web/src/labels.js`:

```js
const INSTANCE = {
  on: 'Питание', brightness: 'Яркость', temperature: 'Температура', humidity: 'Влажность',
  temperature_k: 'Цвет. температура', hsv: 'Цвет', rgb: 'Цвет', scene: 'Сцена',
  volume: 'Громкость', channel: 'Канал', open: 'Открытие', mute: 'Без звука', backlight: 'Подсветка',
  battery_level: 'Батарея', co2_level: 'CO₂', illumination: 'Освещённость', power: 'Мощность',
  voltage: 'Напряжение', amperage: 'Ток', pressure: 'Давление', water_level: 'Уровень воды',
  motion: 'Движение', water_leak: 'Протечка', button: 'Кнопка', vibration: 'Вибрация',
  smoke: 'Дым', gas: 'Газ', thermostat: 'Режим', fan_speed: 'Скорость', work_speed: 'Скорость',
  program: 'Программа', cleanup_mode: 'Режим уборки', swing: 'Поворот', heat: 'Обогрев',
  'pm2.5_density': 'PM2.5', pm10_density: 'PM10', tvoc: 'ЛОС', meter: 'Счётчик',
  food_level: 'Корм', controls_locked: 'Блокировка', ionization: 'Ионизация', keep_warm: 'Подогрев',
  oscillation: 'Вращение', pause: 'Пауза',
}

const UNIT = {
  'unit.percent': '%', 'unit.temperature.celsius': '°C', 'unit.temperature.kelvin': 'K',
  'unit.ppm': 'ppm', 'unit.watt': 'Вт', 'unit.volt': 'В', 'unit.ampere': 'А',
  'unit.kilowatt_hour': 'кВт·ч', 'unit.pressure.mmhg': 'мм рт. ст.', 'unit.pressure.pascal': 'Па',
  'unit.pressure.atm': 'атм', 'unit.pressure.bar': 'бар', 'unit.lux': 'лк',
  'unit.density.mcg_m3': 'мкг/м³', 'unit.cubic_meter': 'м³', 'unit.gigacalorie': 'Гкал',
}

const EVENT = {
  opened: 'открыто', closed: 'закрыто', detected: 'обнаружено', not_detected: 'нет',
  high: 'высокий', low: 'низкий', normal: 'норма', click: 'нажатие', double_click: 'двойное нажатие',
  long_press: 'долгое нажатие', dry: 'сухо', leak: 'протечка', tilt: 'наклон', fall: 'падение',
  vibration: 'вибрация', empty: 'пусто', full: 'полно',
}

const MODE = {
  auto: 'авто', heat: 'обогрев', cool: 'охлаждение', dry: 'осушение', fan_only: 'вентилятор',
  eco: 'эко', low: 'низкая', medium: 'средняя', high: 'высокая', turbo: 'турбо', quiet: 'тихий',
  normal: 'обычный', max: 'макс', min: 'мин', fast: 'быстро', slow: 'медленно', express: 'экспресс',
  horizontal: 'горизонтально', vertical: 'вертикально', stationary: 'без вращения',
}

const TYPE = {
  light: 'Лампа', socket: 'Розетка', switch: 'Выключатель', thermostat: 'Термостат',
  'thermostat.ac': 'Кондиционер', 'media_device.tv': 'Телевизор', 'media_device.tv_box': 'ТВ-приставка',
  'media_device.receiver': 'Ресивер', humidifier: 'Увлажнитель', purifier: 'Очиститель',
  vacuum_cleaner: 'Пылесос', 'cooking.kettle': 'Чайник', 'cooking.coffee_maker': 'Кофеварка',
  washing_machine: 'Стиральная машина', dishwasher: 'Посудомойка', 'openable.curtain': 'Шторы',
  openable: 'Дверь/окно', sensor: 'Датчик', 'sensor.climate': 'Климат', 'sensor.motion': 'Движение',
  'sensor.open': 'Открытие', 'sensor.button': 'Кнопка', 'sensor.water_leak': 'Протечка',
  'sensor.smoke': 'Дым', 'sensor.gas': 'Газ', 'sensor.vibration': 'Вибрация', 'sensor.illumination': 'Освещённость',
  camera: 'Камера', fan: 'Вентилятор', iron: 'Утюг', 'pet_feeder': 'Кормушка', other: 'Устройство',
}

export const label = (instance) => INSTANCE[instance] || instance || ''
export const unit = (u) => (u && UNIT[u]) || ''
export const eventLabel = (e) => EVENT[e] || e
export const modeLabel = (m) => MODE[m] || m

// typeLabel: 'devices.types.sensor.climate' → 'Климат'; неизвестное — последний сегмент.
export function typeLabel(type = '') {
  const short = type.replace(/^devices\.types\./, '')
  return TYPE[short] || short.split('.').pop()
}

// actionErrors собирает ошибки из ответа devices/actions в человекочитаемый список.
export function actionErrors(resp) {
  const out = []
  for (const d of resp?.devices || []) {
    for (const c of d.capabilities || []) {
      const r = c.state?.action_result
      if (r && r.status !== 'DONE') {
        out.push(`${label(c.state?.instance)}: ${r.error_message || errorText(r.error_code)}`)
      }
    }
  }
  return out
}

function errorText(code) {
  return {
    DEVICE_UNREACHABLE: 'устройство не отвечает',
    DEVICE_BUSY: 'устройство занято',
    DEVICE_NOT_FOUND: 'устройство не найдено',
    INTERNAL_ERROR: 'внутренняя ошибка Яндекса',
    INVALID_ACTION: 'недопустимое действие',
    INVALID_VALUE: 'недопустимое значение',
    NOT_SUPPORTED_IN_CURRENT_MODE: 'недоступно в текущем режиме',
  }[code] || code || 'неизвестная ошибка'
}
```

- [ ] **Step 2: Преобразование цветов**

Яндекс принимает цвет либо как целое `rgb` (0xRRGGBB), либо как объект `hsv` с `h` 0–360, `s` 0–100, `v` 0–100.

`web/src/color.js`:

```js
export const hexToRgbInt = (hex) => parseInt(hex.slice(1), 16)

export const rgbIntToHex = (n) => '#' + (Number(n) & 0xffffff).toString(16).padStart(6, '0')

export function hexToHsv(hex) {
  const n = hexToRgbInt(hex)
  const r = ((n >> 16) & 255) / 255
  const g = ((n >> 8) & 255) / 255
  const b = (n & 255) / 255
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  const d = max - min
  let h = 0
  if (d) {
    if (max === r) h = ((g - b) / d) % 6
    else if (max === g) h = (b - r) / d + 2
    else h = (r - g) / d + 4
    h = Math.round(h * 60)
    if (h < 0) h += 360
  }
  return { h, s: Math.round(max ? (d / max) * 100 : 0), v: Math.round(max * 100) }
}

export function hsvToHex({ h = 0, s = 0, v = 0 } = {}) {
  const S = s / 100
  const V = v / 100
  const c = V * S
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = V - c
  let rgb
  if (h < 60) rgb = [c, x, 0]
  else if (h < 120) rgb = [x, c, 0]
  else if (h < 180) rgb = [0, c, x]
  else if (h < 240) rgb = [0, x, c]
  else if (h < 300) rgb = [x, 0, c]
  else rgb = [c, 0, x]
  const [r, g, b] = rgb.map((q) => Math.round((q + m) * 255))
  return rgbIntToHex((r << 16) | (g << 8) | b)
}
```

- [ ] **Step 3: Хук опроса и Home**

`web/src/useHome.js`:

```js
import { useCallback, useEffect, useState } from 'react'
import { api } from './api'

const POLL_MS = 10_000

// useHome держит актуальное состояние дома: загружает сразу и опрашивает раз в 10 с
// (пока вкладка видима). 401 отдаёт наверх — там покажут форму входа.
export function useHome(onUnauthorized) {
  const [home, setHome] = useState(null)
  const [error, setError] = useState('')

  const reload = useCallback(async () => {
    try {
      setHome(await api.home())
      setError('')
    } catch (e) {
      if (e.status === 401) onUnauthorized()
      else setError(e.message)
    }
  }, [onUnauthorized])

  useEffect(() => {
    reload()
    const timer = setInterval(() => {
      if (!document.hidden) reload()
    }, POLL_MS)
    return () => clearInterval(timer)
  }, [reload])

  return { home, error, reload }
}
```

`web/src/Home.jsx` (полная замена заглушки из Task 9):

```jsx
import { useState } from 'react'
import { useHome } from './useHome'
import Dashboard from './Dashboard'

// Scenarios появится в Task 11; до тех пор — заглушка.
function Scenarios() {
  return <p className="muted center">Сценарии — в следующем шаге.</p>
}

export default function Home({ onUnauthorized }) {
  const [tab, setTab] = useState('devices')
  const { home, error, reload } = useHome(onUnauthorized)

  return (
    <div className="app">
      <header>
        <h1>Умный дом</h1>
        <nav>
          <button className={tab === 'devices' ? 'active' : ''} onClick={() => setTab('devices')}>
            Устройства
          </button>
          <button className={tab === 'scenarios' ? 'active' : ''} onClick={() => setTab('scenarios')}>
            Сценарии
          </button>
          <button onClick={reload} title="Обновить">
            ↻
          </button>
        </nav>
      </header>
      {error && <p className="error">{error}</p>}
      {!home ? (
        <p className="muted center">Загружаем устройства…</p>
      ) : tab === 'devices' ? (
        <Dashboard home={home} reload={reload} onUnauthorized={onUnauthorized} />
      ) : (
        <Scenarios home={home} onUnauthorized={onUnauthorized} />
      )}
    </div>
  )
}
```

- [ ] **Step 4: Дашборд и карточка устройства**

`web/src/Dashboard.jsx`:

```jsx
import DeviceCard from './DeviceCard'

// groupByRoom раскладывает устройства по комнатам; без комнаты — в «Без комнаты».
export function groupByRoom(home) {
  const byId = new Map((home.devices || []).map((d) => [d.id, d]))
  const rooms = (home.rooms || []).map((r) => ({
    id: r.id,
    name: r.name,
    devices: (r.devices || []).map((id) => byId.get(id)).filter(Boolean),
  }))
  const placed = new Set(rooms.flatMap((r) => r.devices.map((d) => d.id)))
  const rest = (home.devices || []).filter((d) => !placed.has(d.id))
  if (rest.length) rooms.push({ id: '_none', name: 'Без комнаты', devices: rest })
  return rooms.filter((r) => r.devices.length)
}

export default function Dashboard({ home, reload, onUnauthorized }) {
  const rooms = groupByRoom(home)
  if (!rooms.length) {
    return <p className="muted center">Устройств нет. Добавьте их в приложении «Дом с Алисой».</p>
  }
  return rooms.map((room) => (
    <section className="room" key={room.id}>
      <h2>{room.name}</h2>
      <div className="devices">
        {room.devices.map((d) => (
          <DeviceCard key={d.id} device={d} reload={reload} onUnauthorized={onUnauthorized} />
        ))}
      </div>
    </section>
  ))
}
```

`web/src/DeviceCard.jsx`:

```jsx
import { useState } from 'react'
import { api } from './api'
import { actionErrors, typeLabel } from './labels'
import { Capability, PropertyReadout } from './Controls'

export default function DeviceCard({ device, reload, onUnauthorized }) {
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  // send отправляет одно действие и после ответа перечитывает состояние дома,
  // чтобы карточка показала фактическое состояние устройства.
  async function send(type, instance, value) {
    setBusy(true)
    setError('')
    try {
      const resp = await api.deviceActions([{ id: device.id, actions: [{ type, state: { instance, value } }] }])
      const errors = actionErrors(resp)
      if (errors.length) setError(errors.join('; '))
    } catch (e) {
      if (e.status === 401) onUnauthorized()
      setError(e.message)
    } finally {
      setBusy(false)
      reload()
    }
  }

  const caps = device.capabilities || []
  const props = device.properties || []

  return (
    <div className="card device">
      <div className="row between">
        <strong>{device.name}</strong>
        <span className="type">{typeLabel(device.type)}</span>
      </div>
      {caps.map((cap, i) => (
        <Capability key={i} cap={cap} busy={busy} onChange={send} />
      ))}
      {props.length > 0 && (
        <div className="props">
          {props.map((p, i) => (
            <PropertyReadout key={i} prop={p} />
          ))}
        </div>
      )}
      {error && <p className="error">{error}</p>}
    </div>
  )
}
```

- [ ] **Step 5: Элементы управления**

`web/src/Controls.jsx`:

```jsx
import { useEffect, useRef, useState } from 'react'
import { eventLabel, label, modeLabel, unit } from './labels'
import { hexToHsv, hexToRgbInt, hsvToHex, rgbIntToHex } from './color'

// useSynced — локальное значение, которое подхватывает новое значение с сервера
// (оптимистичное обновление: показываем сразу, сервер подтвердит при следующем опросе).
function useSynced(value) {
  const [local, setLocal] = useState(value)
  useEffect(() => setLocal(value), [value])
  return [local, setLocal]
}

// useDebounced вызывает fn через delay мс после последнего вызова (для color picker,
// который сыплет события при каждом движении курсора).
function useDebounced(fn, delay = 400) {
  const timer = useRef(null)
  useEffect(() => () => clearTimeout(timer.current), [])
  return (...args) => {
    clearTimeout(timer.current)
    timer.current = setTimeout(() => fn(...args), delay)
  }
}

// Capability выбирает контрол по типу умения. onChange(type, instance, value).
export function Capability({ cap, busy, onChange }) {
  const kind = cap.type.split('.').pop()
  const p = cap.parameters || {}
  const state = cap.state || {}
  switch (kind) {
    case 'on_off':
      return <Switch text="Питание" checked={!!state.value} disabled={busy} onChange={(v) => onChange(cap.type, 'on', v)} />
    case 'toggle':
      return <Switch text={label(p.instance)} checked={!!state.value} disabled={busy} onChange={(v) => onChange(cap.type, p.instance, v)} />
    case 'range':
      return <Range cap={cap} busy={busy} onChange={onChange} />
    case 'mode':
      return <Mode cap={cap} busy={busy} onChange={onChange} />
    case 'color_setting':
      return <ColorSetting cap={cap} busy={busy} onChange={onChange} />
    default:
      return null
  }
}

function Switch({ text, checked, disabled, onChange }) {
  const [local, setLocal] = useSynced(checked)
  return (
    <div className="control">
      <label>{text}</label>
      <input
        type="checkbox"
        checked={local}
        disabled={disabled}
        onChange={(e) => {
          setLocal(e.target.checked)
          onChange(e.target.checked)
        }}
      />
      <span className="value">{local ? 'вкл' : 'выкл'}</span>
    </div>
  )
}

function Slider({ text, min, max, step, value, suffix, disabled, onCommit }) {
  const [local, setLocal] = useSynced(value)
  const commit = () => {
    if (local !== value) onCommit(Number(local))
  }
  return (
    <div className="control">
      <label>{text}</label>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={local}
        disabled={disabled}
        onChange={(e) => setLocal(Number(e.target.value))}
        onPointerUp={commit}
        onKeyUp={commit}
      />
      <span className="value">
        {local}
        {suffix}
      </span>
    </div>
  )
}

function Range({ cap, busy, onChange }) {
  const p = cap.parameters || {}
  const r = p.range || { min: 0, max: 100, precision: 1 }
  return (
    <Slider
      text={label(p.instance)}
      min={r.min}
      max={r.max}
      step={r.precision || 1}
      value={cap.state?.value ?? r.min}
      suffix={unit(p.unit)}
      disabled={busy}
      onCommit={(v) => onChange(cap.type, p.instance, v)}
    />
  )
}

function Mode({ cap, busy, onChange }) {
  const p = cap.parameters || {}
  return (
    <div className="control">
      <label>{label(p.instance)}</label>
      <select value={cap.state?.value ?? ''} disabled={busy} onChange={(e) => onChange(cap.type, p.instance, e.target.value)}>
        <option value="" disabled>
          —
        </option>
        {(p.modes || []).map((m) => (
          <option key={m.value} value={m.value}>
            {modeLabel(m.value)}
          </option>
        ))}
      </select>
    </div>
  )
}

function ColorSetting({ cap, busy, onChange }) {
  const p = cap.parameters || {}
  const state = cap.state || {}
  const model = p.color_model // 'hsv' | 'rgb' | undefined
  const current = model && state.instance === model ? state.value : null
  const hex = current == null ? '#ffffff' : model === 'rgb' ? rgbIntToHex(current) : hsvToHex(current)
  const [localHex, setLocalHex] = useSynced(hex)
  const commitColor = useDebounced((h) => onChange(cap.type, model, model === 'rgb' ? hexToRgbInt(h) : hexToHsv(h)))

  return (
    <>
      {p.temperature_k && (
        <Slider
          text="Цвет. температура"
          min={p.temperature_k.min}
          max={p.temperature_k.max}
          step={100}
          value={state.instance === 'temperature_k' ? state.value : p.temperature_k.min}
          suffix=" K"
          disabled={busy}
          onCommit={(v) => onChange(cap.type, 'temperature_k', v)}
        />
      )}
      {model && (
        <div className="control">
          <label>Цвет</label>
          <input
            type="color"
            value={localHex}
            disabled={busy}
            onChange={(e) => {
              setLocalHex(e.target.value)
              commitColor(e.target.value)
            }}
          />
        </div>
      )}
      {p.color_scene?.scenes?.length > 0 && (
        <div className="control">
          <label>Сцена</label>
          <select
            value={state.instance === 'scene' ? state.value : ''}
            disabled={busy}
            onChange={(e) => onChange(cap.type, 'scene', e.target.value)}
          >
            <option value="">—</option>
            {p.color_scene.scenes.map((s) => (
              <option key={s.id} value={s.id}>
                {s.id}
              </option>
            ))}
          </select>
        </div>
      )}
    </>
  )
}

// PropertyReadout — показание датчика: число с единицей или событие.
export function PropertyReadout({ prop }) {
  const p = prop.parameters || {}
  const v = prop.state?.value
  const kind = prop.type.split('.').pop() // 'float' | 'event'
  let text
  if (v == null) text = '—'
  else if (kind === 'event') text = eventLabel(v)
  else text = `${typeof v === 'number' ? Math.round(v * 10) / 10 : v} ${unit(p.unit)}`.trim()
  return (
    <span title={p.instance}>
      <span className="muted">{label(p.instance)}: </span>
      {text}
    </span>
  )
}
```

- [ ] **Step 6: Собрать и проверить**

```bash
cd web && npm run build && cd .. && gofmt -l . && go vet ./... && go test ./... && go build -o smart-house .
```

Expected: сборка без ошибок и предупреждений Vite про неиспользуемые импорты.

Проверка вживую возможна только с реальным токеном — она вынесена в Task 13. Здесь достаточно: `npm run build` зелёный, `go build` зелёный.

- [ ] **Step 7: Коммит**

```bash
git add web/src
git commit -m "feat: дашборд устройств — комнаты, умения, датчики"
```

---

### Task 11: Фронтенд — сценарии и макросы

**Files:**
- Create: `web/src/Scenarios.jsx`
- Create: `web/src/MacroEditor.jsx`
- Modify: `web/src/Home.jsx` (убрать заглушку `Scenarios`, импортировать компонент)

**Interfaces:**
- Consumes: `api.runScenario`, `api.macros`, `api.createMacro`, `api.updateMacro`, `api.deleteMacro`, `api.runMacro` (Task 9); `actionErrors`, `label`, `modeLabel` (Task 10); `color.js` (Task 10); контракт `/api/macros` (Task 7).
- Produces: `<Scenarios home onUnauthorized />`, `<MacroEditor home macro onSaved onCancel />`, `capabilityOptions(device)`.

Формат макроса на фронте совпадает с бэкендом: `{id?, name, actions: [{device_id, type, instance, value}]}`.

- [ ] **Step 1: Список сценариев и макросов**

`web/src/Scenarios.jsx`:

```jsx
import { useCallback, useEffect, useState } from 'react'
import { api } from './api'
import { actionErrors } from './labels'
import MacroEditor from './MacroEditor'

export default function Scenarios({ home, onUnauthorized }) {
  const [macros, setMacros] = useState([])
  const [editing, setEditing] = useState(null) // null | {} для нового | существующий макрос
  const [status, setStatus] = useState({}) // id → текст результата запуска
  const [error, setError] = useState('')

  const loadMacros = useCallback(
    () =>
      api
        .macros()
        .then(setMacros)
        .catch((e) => setError(e.message)),
    [],
  )
  useEffect(() => {
    loadMacros()
  }, [loadMacros])

  // run запускает сценарий/макрос и пишет результат рядом с кнопкой.
  function run(id, promise) {
    setStatus((s) => ({ ...s, [id]: 'Запускаем…' }))
    promise
      .then((resp) => {
        const errs = actionErrors(resp)
        setStatus((s) => ({ ...s, [id]: errs.length ? 'Ошибка: ' + errs.join('; ') : 'Выполнено ✓' }))
      })
      .catch((e) => {
        if (e.status === 401) onUnauthorized()
        setStatus((s) => ({ ...s, [id]: 'Ошибка: ' + e.message }))
      })
  }

  function remove(id) {
    api
      .deleteMacro(id)
      .then(loadMacros)
      .catch((e) => setError(e.message))
  }

  const scenarios = home.scenarios || []

  return (
    <>
      <section className="room">
        <h2>Сценарии Яндекса</h2>
        {!scenarios.length ? (
          <p className="muted">Сценариев нет — их создают в приложении «Дом с Алисой».</p>
        ) : (
          <div className="list">
            {scenarios.map((s) => (
              <div className="card item" key={s.id}>
                <span>{s.name}</span>
                <span className="row">
                  <span className="muted">{status[s.id]}</span>
                  <button className="primary" onClick={() => run(s.id, api.runScenario(s.id))}>
                    Запустить
                  </button>
                </span>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="room">
        <div className="row between">
          <h2>Мои макросы</h2>
          {!editing && <button onClick={() => setEditing({})}>+ Новый макрос</button>}
        </div>
        {error && <p className="error">{error}</p>}
        {editing && (
          <MacroEditor
            home={home}
            macro={editing}
            onSaved={() => {
              setEditing(null)
              loadMacros()
            }}
            onCancel={() => setEditing(null)}
          />
        )}
        {!macros.length && !editing && <p className="muted">Макросов пока нет.</p>}
        <div className="list">
          {macros.map((m) => (
            <div className="card item" key={m.id}>
              <span>
                <strong>{m.name}</strong> <span className="muted">· действий: {m.actions.length}</span>
              </span>
              <span className="row">
                <span className="muted">{status[m.id]}</span>
                <button className="primary" onClick={() => run(m.id, api.runMacro(m.id))}>
                  Запустить
                </button>
                <button onClick={() => setEditing(m)}>Изменить</button>
                <button className="danger" onClick={() => remove(m.id)}>
                  Удалить
                </button>
              </span>
            </div>
          ))}
        </div>
      </section>
    </>
  )
}
```

- [ ] **Step 2: Редактор макроса**

`web/src/MacroEditor.jsx`:

```jsx
import { useState } from 'react'
import { api } from './api'
import { label, modeLabel } from './labels'
import { hexToHsv, hexToRgbInt, hsvToHex, rgbIntToHex } from './color'

// capabilityOptions — управляемые умения устройства в виде вариантов для редактора:
// [{key, type, instance, kind: 'bool'|'number'|'mode'|'color', params}].
export function capabilityOptions(device) {
  const out = []
  for (const cap of device.capabilities || []) {
    const kind = cap.type.split('.').pop()
    const p = cap.parameters || {}
    if (kind === 'on_off') out.push({ type: cap.type, instance: 'on', kind: 'bool', params: p })
    else if (kind === 'toggle') out.push({ type: cap.type, instance: p.instance, kind: 'bool', params: p })
    else if (kind === 'range') out.push({ type: cap.type, instance: p.instance, kind: 'number', params: p })
    else if (kind === 'mode') out.push({ type: cap.type, instance: p.instance, kind: 'mode', params: p })
    else if (kind === 'color_setting') {
      if (p.temperature_k) {
        const { min, max } = p.temperature_k
        out.push({ type: cap.type, instance: 'temperature_k', kind: 'number', params: { range: { min, max, precision: 100 } } })
      }
      if (p.color_model) out.push({ type: cap.type, instance: p.color_model, kind: 'color', params: p })
    }
  }
  return out.map((o) => ({ ...o, key: o.type + ':' + o.instance }))
}

function defaultValue(opt) {
  switch (opt.kind) {
    case 'bool':
      return true
    case 'number':
      return opt.params.range?.min ?? 0
    case 'mode':
      return opt.params.modes?.[0]?.value ?? ''
    case 'color':
      return opt.instance === 'rgb' ? 0xffffff : { h: 0, s: 0, v: 100 }
    default:
      return null
  }
}

export default function MacroEditor({ home, macro, onSaved, onCancel }) {
  const [name, setName] = useState(macro.name || '')
  const [actions, setActions] = useState(macro.actions || [])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const devices = (home.devices || []).filter((d) => capabilityOptions(d).length)

  function addAction() {
    const d = devices[0]
    if (!d) return
    const opt = capabilityOptions(d)[0]
    setActions((a) => [...a, { device_id: d.id, type: opt.type, instance: opt.instance, value: defaultValue(opt) }])
  }
  const update = (i, patch) => setActions((a) => a.map((x, j) => (j === i ? { ...x, ...patch } : x)))
  const remove = (i) => setActions((a) => a.filter((_, j) => j !== i))

  async function save(e) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const body = { name: name.trim(), actions }
      if (macro.id) await api.updateMacro({ ...body, id: macro.id })
      else await api.createMacro(body)
      onSaved()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <form className="card editor" onSubmit={save}>
      <h3>{macro.id ? 'Изменить макрос' : 'Новый макрос'}</h3>
      <div className="control">
        <label>Название</label>
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Например, Кино" autoFocus />
      </div>
      {actions.map((a, i) => (
        <ActionRow key={i} action={a} devices={devices} onChange={(patch) => update(i, patch)} onRemove={() => remove(i)} />
      ))}
      <div className="row">
        <button type="button" onClick={addAction} disabled={!devices.length}>
          + Действие
        </button>
        <button type="submit" className="primary" disabled={busy || !name.trim() || !actions.length}>
          Сохранить
        </button>
        <button type="button" onClick={onCancel}>
          Отмена
        </button>
      </div>
      {!devices.length && <p className="muted">Нет устройств, которыми можно управлять.</p>}
      {error && <p className="error">{error}</p>}
    </form>
  )
}

function ActionRow({ action, devices, onChange, onRemove }) {
  const device = devices.find((d) => d.id === action.device_id) || devices[0]
  const options = capabilityOptions(device)
  const opt = options.find((o) => o.type === action.type && o.instance === action.instance) || options[0]

  function pickDevice(id) {
    const o = capabilityOptions(devices.find((d) => d.id === id))[0]
    onChange({ device_id: id, type: o.type, instance: o.instance, value: defaultValue(o) })
  }
  function pickCapability(key) {
    const o = options.find((x) => x.key === key)
    onChange({ type: o.type, instance: o.instance, value: defaultValue(o) })
  }

  return (
    <div className="action">
      <select value={device.id} onChange={(e) => pickDevice(e.target.value)}>
        {devices.map((d) => (
          <option key={d.id} value={d.id}>
            {d.name}
          </option>
        ))}
      </select>
      <select value={opt.key} onChange={(e) => pickCapability(e.target.value)}>
        {options.map((o) => (
          <option key={o.key} value={o.key}>
            {label(o.instance)}
          </option>
        ))}
      </select>
      <ValueInput opt={opt} value={action.value} onChange={(value) => onChange({ value })} />
      <button type="button" className="danger" onClick={onRemove} title="Убрать действие">
        ✕
      </button>
    </div>
  )
}

function ValueInput({ opt, value, onChange }) {
  switch (opt.kind) {
    case 'bool':
      return (
        <select value={value ? '1' : '0'} onChange={(e) => onChange(e.target.value === '1')}>
          <option value="1">включить</option>
          <option value="0">выключить</option>
        </select>
      )
    case 'number': {
      const r = opt.params.range || {}
      return (
        <input
          type="number"
          min={r.min}
          max={r.max}
          step={r.precision || 1}
          value={value ?? ''}
          onChange={(e) => onChange(Number(e.target.value))}
        />
      )
    }
    case 'mode':
      return (
        <select value={value ?? ''} onChange={(e) => onChange(e.target.value)}>
          {(opt.params.modes || []).map((m) => (
            <option key={m.value} value={m.value}>
              {modeLabel(m.value)}
            </option>
          ))}
        </select>
      )
    case 'color': {
      const hex = opt.instance === 'rgb' ? rgbIntToHex(value ?? 0xffffff) : hsvToHex(value || { h: 0, s: 0, v: 100 })
      return (
        <input
          type="color"
          value={hex}
          onChange={(e) => onChange(opt.instance === 'rgb' ? hexToRgbInt(e.target.value) : hexToHsv(e.target.value))}
        />
      )
    }
    default:
      return null
  }
}
```

- [ ] **Step 3: Подключить Scenarios в Home**

В `web/src/Home.jsx` удалить локальную заглушку `function Scenarios() {…}` и добавить импорт:

```jsx
import Scenarios from './Scenarios'
```

- [ ] **Step 4: Собрать и проверить**

```bash
cd web && npm run build && cd .. && go build -o smart-house .
```

Expected: сборка без ошибок. Затем без реального токена можно проверить только API макросов:

```bash
YANDEX_CLIENT_ID=x YANDEX_CLIENT_SECRET=y PORT=18080 DATA_DIR=/tmp/smart-house-smoke ./smart-house & PID=$!
sleep 1
curl -s -X POST localhost:18080/api/macros -d '{"name":"Тест","actions":[{"device_id":"d1","type":"devices.capabilities.on_off","instance":"on","value":true}]}'
curl -s localhost:18080/api/macros
kill $PID
```

Expected: первый запрос — макрос с `id`, второй — список из одного макроса.

- [ ] **Step 5: Коммит**

```bash
git add web/src
git commit -m "feat: вкладка сценариев — запуск сценариев Яндекса и локальные макросы"
```

---

### Task 12: Каталог для Claude и проектный skill

**Files:**
- Create: `internal/catalog/catalog.go`
- Test: `internal/catalog/catalog_test.go`
- Create: `internal/server/catalog_handler.go`
- Modify: `internal/server/server.go` (маршрут `GET /api/catalog`)
- Modify: `internal/server/server_test.go` (тест каталога)
- Create: `.claude/skills/smart-home/SKILL.md`
- Modify: `README.md` (раздел «Сценарии через Claude»)

**Interfaces:**
- Consumes: `HomeAPI.UserInfo` (Task 6), `writeJSON`, `writeError`, тестовые `fakeHome`, `fakeAuth`, `newTestServer`, `do`, `decode` (Task 6).
- Produces: `catalog.Build(raw json.RawMessage) (Catalog, error)`; `catalog.Catalog{Devices []Device; Scenarios []Scenario}`, `Device{ID, Name, Room, Type string; Capabilities []Capability; Properties []Property}`, `Capability{Type, Instance, Kind string; Min, Max, Step *float64; Unit string; Modes []string; Value any}`, `Property{Instance, Unit string; Value any}`, `Scenario{ID, Name string}`; endpoint `GET /api/catalog`.

Формат ответа `GET /api/catalog` (то, что читает Claude):

```json
{
  "devices": [{
    "id": "d1", "name": "Лампа", "room": "Гостиная", "type": "devices.types.light",
    "capabilities": [
      {"type": "devices.capabilities.on_off", "instance": "on", "kind": "bool", "value": true},
      {"type": "devices.capabilities.range", "instance": "brightness", "kind": "number",
       "min": 1, "max": 100, "step": 1, "unit": "unit.percent", "value": 50},
      {"type": "devices.capabilities.color_setting", "instance": "temperature_k", "kind": "number",
       "min": 2700, "max": 6500, "step": 100, "value": 4000},
      {"type": "devices.capabilities.color_setting", "instance": "hsv", "kind": "color"},
      {"type": "devices.capabilities.mode", "instance": "thermostat", "kind": "mode",
       "modes": ["heat", "cool"], "value": "heat"}
    ],
    "properties": [{"instance": "temperature", "unit": "unit.temperature.celsius", "value": 22.5}]
  }],
  "scenarios": [{"id": "s1", "name": "Кино"}]
}
```

- [ ] **Step 1: Написать падающий тест каталога**

`internal/catalog/catalog_test.go`:

```go
package catalog

import (
	"encoding/json"
	"testing"
)

const sampleUserInfo = `{
  "status": "ok", "request_id": "r",
  "rooms": [{"id": "r1", "name": "Гостиная", "household_id": "h1", "devices": ["d1"]}],
  "devices": [{
    "id": "d1", "name": "Лампа", "type": "devices.types.light", "room": "r1",
    "capabilities": [
      {"type": "devices.capabilities.on_off", "retrievable": true, "parameters": {"split": false},
       "state": {"instance": "on", "value": true}, "last_updated": 1757325000.0},
      {"type": "devices.capabilities.range", "retrievable": true,
       "parameters": {"instance": "brightness", "unit": "unit.percent", "random_access": true,
                      "range": {"min": 1, "max": 100, "precision": 1}},
       "state": {"instance": "brightness", "value": 50}},
      {"type": "devices.capabilities.color_setting", "retrievable": true,
       "parameters": {"color_model": "hsv", "temperature_k": {"min": 2700, "max": 6500}},
       "state": {"instance": "temperature_k", "value": 4000}},
      {"type": "devices.capabilities.mode", "retrievable": true,
       "parameters": {"instance": "thermostat", "modes": [{"value": "heat"}, {"value": "cool"}]},
       "state": {"instance": "thermostat", "value": "heat"}}
    ],
    "properties": [
      {"type": "devices.properties.float", "retrievable": true,
       "parameters": {"instance": "temperature", "unit": "unit.temperature.celsius"},
       "state": {"instance": "temperature", "value": 22.5}}
    ]
  }, {
    "id": "d2", "name": "Датчик", "type": "devices.types.sensor", "room": "",
    "capabilities": [], "properties": []
  }],
  "scenarios": [{"id": "s1", "name": "Кино", "is_active": true}],
  "households": [{"id": "h1", "name": "Мой дом"}]
}`

func TestBuildCompactsUserInfo(t *testing.T) {
	cat, err := Build(json.RawMessage(sampleUserInfo))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(cat.Devices) != 2 || len(cat.Scenarios) != 1 {
		t.Fatalf("devices=%d scenarios=%d", len(cat.Devices), len(cat.Scenarios))
	}
	d := cat.Devices[0]
	if d.ID != "d1" || d.Name != "Лампа" || d.Room != "Гостиная" || d.Type != "devices.types.light" {
		t.Errorf("device = %+v", d)
	}
	if len(d.Capabilities) != 5 {
		t.Fatalf("capabilities = %d, want 5 (on, brightness, temperature_k, hsv, thermostat)", len(d.Capabilities))
	}
	kinds := map[string]string{}
	for _, c := range d.Capabilities {
		kinds[c.Instance] = c.Kind
	}
	want := map[string]string{"on": "bool", "brightness": "number", "temperature_k": "number", "hsv": "color", "thermostat": "mode"}
	for inst, kind := range want {
		if kinds[inst] != kind {
			t.Errorf("instance %s kind = %q, want %q", inst, kinds[inst], kind)
		}
	}
	br := d.Capabilities[1]
	if br.Min == nil || *br.Min != 1 || br.Max == nil || *br.Max != 100 || br.Step == nil || *br.Step != 1 || br.Unit != "unit.percent" || br.Value != float64(50) {
		t.Errorf("brightness = %+v", br)
	}
	tk := d.Capabilities[2]
	if tk.Min == nil || *tk.Min != 2700 || tk.Step == nil || *tk.Step != 100 || tk.Value != float64(4000) {
		t.Errorf("temperature_k = %+v", tk)
	}
	if hsv := d.Capabilities[3]; hsv.Value != nil {
		t.Errorf("hsv value = %v, want nil (state относится к temperature_k)", hsv.Value)
	}
	mode := d.Capabilities[4]
	if len(mode.Modes) != 2 || mode.Modes[0] != "heat" || mode.Value != "heat" {
		t.Errorf("mode = %+v", mode)
	}
	if len(d.Properties) != 1 || d.Properties[0].Instance != "temperature" || d.Properties[0].Value != float64(22.5) {
		t.Errorf("properties = %+v", d.Properties)
	}
	if cat.Devices[1].Room != "" || cat.Devices[1].Capabilities == nil || cat.Devices[1].Properties == nil {
		t.Errorf("device without room = %+v (срезы должны быть пустыми, не nil)", cat.Devices[1])
	}
	if cat.Scenarios[0].ID != "s1" || cat.Scenarios[0].Name != "Кино" {
		t.Errorf("scenarios = %+v", cat.Scenarios)
	}
}

func TestBuildRejectsBadJSON(t *testing.T) {
	if _, err := Build(json.RawMessage(`nope`)); err == nil {
		t.Fatal("ожидалась ошибка")
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/catalog/`
Expected: ошибка компиляции — `undefined: Build`.

- [ ] **Step 3: Реализация каталога**

`internal/catalog/catalog.go`:

```go
// Package catalog сжимает ответ user/info Яндекса до того, что нужно Claude,
// чтобы собирать макросы: устройства, их управляемые умения и показания.
package catalog

import (
	"encoding/json"
	"strings"
)

type Capability struct {
	Type     string   `json:"type"`
	Instance string   `json:"instance"`
	Kind     string   `json:"kind"` // bool | number | mode | color
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Step     *float64 `json:"step,omitempty"`
	Unit     string   `json:"unit,omitempty"`
	Modes    []string `json:"modes,omitempty"`
	Value    any      `json:"value,omitempty"`
}

type Property struct {
	Instance string `json:"instance"`
	Unit     string `json:"unit,omitempty"`
	Value    any    `json:"value"`
}

type Device struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Room         string       `json:"room"`
	Type         string       `json:"type"`
	Capabilities []Capability `json:"capabilities"`
	Properties   []Property   `json:"properties"`
}

type Scenario struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Catalog struct {
	Devices   []Device   `json:"devices"`
	Scenarios []Scenario `json:"scenarios"`
}

type rawState struct {
	Instance string `json:"instance"`
	Value    any    `json:"value"`
}

type rawCapability struct {
	Type       string `json:"type"`
	Parameters struct {
		Instance string `json:"instance"`
		Unit     string `json:"unit"`
		Range    *struct {
			Min       float64 `json:"min"`
			Max       float64 `json:"max"`
			Precision float64 `json:"precision"`
		} `json:"range"`
		Modes []struct {
			Value string `json:"value"`
		} `json:"modes"`
		ColorModel   string `json:"color_model"`
		TemperatureK *struct {
			Min float64 `json:"min"`
			Max float64 `json:"max"`
		} `json:"temperature_k"`
	} `json:"parameters"`
	State *rawState `json:"state"`
}

type rawProperty struct {
	Parameters struct {
		Instance string `json:"instance"`
		Unit     string `json:"unit"`
	} `json:"parameters"`
	State *rawState `json:"state"`
}

type rawUserInfo struct {
	Rooms []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"rooms"`
	Devices []struct {
		ID           string          `json:"id"`
		Name         string          `json:"name"`
		Type         string          `json:"type"`
		Room         string          `json:"room"`
		Capabilities []rawCapability `json:"capabilities"`
		Properties   []rawProperty   `json:"properties"`
	} `json:"devices"`
	Scenarios []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"scenarios"`
}

func Build(raw json.RawMessage) (Catalog, error) {
	var info rawUserInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return Catalog{}, err
	}
	rooms := map[string]string{}
	for _, r := range info.Rooms {
		rooms[r.ID] = r.Name
	}

	cat := Catalog{Devices: []Device{}, Scenarios: []Scenario{}}
	for _, d := range info.Devices {
		dev := Device{ID: d.ID, Name: d.Name, Room: rooms[d.Room], Type: d.Type,
			Capabilities: []Capability{}, Properties: []Property{}}
		for _, c := range d.Capabilities {
			dev.Capabilities = append(dev.Capabilities, convert(c)...)
		}
		for _, p := range d.Properties {
			prop := Property{Instance: p.Parameters.Instance, Unit: p.Parameters.Unit}
			if p.State != nil {
				prop.Value = p.State.Value
			}
			dev.Properties = append(dev.Properties, prop)
		}
		cat.Devices = append(cat.Devices, dev)
	}
	for _, s := range info.Scenarios {
		cat.Scenarios = append(cat.Scenarios, Scenario{ID: s.ID, Name: s.Name})
	}
	return cat, nil
}

// convert разворачивает одно умение Яндекса в одну или несколько записей каталога
// (color_setting может дать и temperature_k, и hsv/rgb).
func convert(c rawCapability) []Capability {
	kind := c.Type[strings.LastIndex(c.Type, ".")+1:]
	p := c.Parameters
	valueOf := func(instance string) any {
		if c.State != nil && c.State.Instance == instance {
			return c.State.Value
		}
		return nil
	}
	switch kind {
	case "on_off":
		return []Capability{{Type: c.Type, Instance: "on", Kind: "bool", Value: valueOf("on")}}
	case "toggle":
		return []Capability{{Type: c.Type, Instance: p.Instance, Kind: "bool", Value: valueOf(p.Instance)}}
	case "range":
		out := Capability{Type: c.Type, Instance: p.Instance, Kind: "number", Unit: p.Unit, Value: valueOf(p.Instance)}
		if p.Range != nil {
			out.Min, out.Max, out.Step = &p.Range.Min, &p.Range.Max, &p.Range.Precision
		}
		return []Capability{out}
	case "mode":
		modes := make([]string, 0, len(p.Modes))
		for _, m := range p.Modes {
			modes = append(modes, m.Value)
		}
		return []Capability{{Type: c.Type, Instance: p.Instance, Kind: "mode", Modes: modes, Value: valueOf(p.Instance)}}
	case "color_setting":
		var out []Capability
		if p.TemperatureK != nil {
			step := 100.0
			out = append(out, Capability{Type: c.Type, Instance: "temperature_k", Kind: "number",
				Min: &p.TemperatureK.Min, Max: &p.TemperatureK.Max, Step: &step, Value: valueOf("temperature_k")})
		}
		if p.ColorModel != "" {
			out = append(out, Capability{Type: c.Type, Instance: p.ColorModel, Kind: "color", Value: valueOf(p.ColorModel)})
		}
		return out
	}
	return nil
}
```

- [ ] **Step 4: Проверить тест каталога**

Run: `gofmt -l . && go vet ./... && go test ./internal/catalog/ -v`
Expected: 2 теста PASS.

- [ ] **Step 5: Тест и обработчик endpoint**

Добавить в `internal/server/server_test.go`:

```go
func TestCatalogCompactsHome(t *testing.T) {
	raw := `{"status":"ok","request_id":"r","rooms":[{"id":"r1","name":"Кухня","devices":["d1"]}],
	  "devices":[{"id":"d1","name":"Розетка","type":"devices.types.socket","room":"r1",
	    "capabilities":[{"type":"devices.capabilities.on_off","parameters":{},"state":{"instance":"on","value":false}}],
	    "properties":[]}],
	  "scenarios":[]}`
	srv := newTestServer(t, &fakeHome{userInfo: json.RawMessage(raw)}, &fakeAuth{authorized: true})
	code, body := do(t, "GET", srv.URL+"/api/catalog", "")
	if code != 200 {
		t.Fatalf("status %d: %s", code, body)
	}
	var cat struct {
		Devices []struct {
			Room         string `json:"room"`
			Capabilities []struct {
				Instance string `json:"instance"`
				Kind     string `json:"kind"`
				Value    any    `json:"value"`
			} `json:"capabilities"`
		} `json:"devices"`
	}
	decode(t, body, &cat)
	if len(cat.Devices) != 1 || cat.Devices[0].Room != "Кухня" {
		t.Fatalf("catalog = %s", body)
	}
	c := cat.Devices[0].Capabilities
	if len(c) != 1 || c[0].Instance != "on" || c[0].Kind != "bool" || c[0].Value != false {
		t.Errorf("capabilities = %+v", c)
	}
}
```

`internal/server/catalog_handler.go`:

```go
package server

import (
	"net/http"

	"smarthome/internal/catalog"
)

func (s *Server) catalog(w http.ResponseWriter, r *http.Request) {
	raw, err := s.Home.UserInfo(r.Context())
	if err != nil {
		s.writeError(w, err)
		return
	}
	cat, err := catalog.Build(raw)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cat)
}
```

В `internal/server/server.go`, в `Handler()` после строки `GET /api/home` добавить:

```go
	mux.HandleFunc("GET /api/catalog", s.catalog)
```

Run: `gofmt -l . && go vet ./... && go test ./...`
Expected: все пакеты `ok`.

- [ ] **Step 6: Проектный skill**

`.claude/skills/smart-home/SKILL.md`:

````markdown
---
name: smart-home
description: Управление умным домом Яндекса и сборка сценариев (макросов) из терминала. Использовать, когда пользователь просит включить/выключить/настроить устройство, показать устройства или показания датчиков, создать/изменить/удалить сценарий или макрос («сделай сценарий кино», «когда ухожу — выключай всё»), запустить сценарий или макрос.
---

# Умный дом: управление и сценарии

Сервер приложения работает на http://localhost:8080 и сам хранит OAuth-токен.
Все операции — через его API (`curl`). Напрямую к Яндексу не ходить, ключи из `.env` не читать.

## 1. Убедиться, что сервер запущен и авторизован

```bash
curl -sf localhost:8080/api/auth/status
```

- Соединение отклонено → запустить сервер в фоне: `./smart-house` (если бинарника нет — сначала `make build`), подождать 2 секунды, повторить проверку.
- `"authorized":false` → попросить пользователя открыть http://localhost:8080 и войти через Яндекс; до входа дальше не продолжать.

## 2. Узнать, какие устройства есть

```bash
curl -s localhost:8080/api/catalog
```

Ответ: `devices[]` — `id`, `name`, `room`, `type`, `capabilities[]` (что можно менять), `properties[]` (показания датчиков); `scenarios[]` — родные сценарии Яндекса `{id, name}`.

Каждое умение: `type`, `instance`, `kind` и ограничения:
- `kind: bool` → значение `true`/`false` (`instance` обычно `on`);
- `kind: number` → число в пределах `min..max` с шагом `step` (`unit` — единица; `temperature_k` — кельвины);
- `kind: mode` → одна из строк `modes`;
- `kind: color` → `instance` `rgb`: целое `0xRRGGBB` (например, 16711680 — красный); `instance` `hsv`: объект `{"h": 0–360, "s": 0–100, "v": 0–100}`.

Использовать только `id`, `type` и `instance` из каталога. Если подходящего устройства или умения нет — сказать об этом, ничего не выдумывать. Если название неоднозначно (три «лампы») — спросить пользователя, какую именно.

## 3. Создать или изменить сценарий (макрос)

Макрос — `{"name": "...", "actions": [{"device_id", "type", "instance", "value"}, ...]}`; одно действие на пару «устройство + умение». JSON писать во временный файл в scratchpad, не в проект.

Перед сохранением показать пользователю список «устройство → что сделаем» и сохранить:

```bash
# новый
curl -s -X POST localhost:8080/api/macros -H 'Content-Type: application/json' -d @macro.json
# правка существующего (id — из GET /api/macros)
curl -s -X PUT localhost:8080/api/macros/<id> -H 'Content-Type: application/json' -d @macro.json
# удалить
curl -s -X DELETE localhost:8080/api/macros/<id>
```

Ответ 400 — в `message` причина (нет имени, нет действий, нет device_id). После сохранения предложить сразу выполнить макрос и проверить результат.

## 4. Выполнить

```bash
curl -s -X POST localhost:8080/api/macros/<id>/run        # макрос (id или найти по имени в GET /api/macros)
curl -s -X POST localhost:8080/api/scenarios/<id>/run     # родной сценарий Яндекса (id из каталога)
```

Разовая команда без макроса («включи свет на кухне»):

```bash
curl -s -X POST localhost:8080/api/devices/actions -H 'Content-Type: application/json' \
  -d '{"devices":[{"id":"<device_id>","actions":[{"type":"devices.capabilities.on_off","state":{"instance":"on","value":true}}]}]}'
```

В ответе Яндекса для каждого устройства `capabilities[].state.action_result.status`: `DONE` — сработало; иначе `error_code`/`error_message` — сообщить пользователю, какое устройство не сработало и почему (`DEVICE_UNREACHABLE` — не отвечает, обычно нет питания или сети).

HTTP 401 от сервера — токен протух: попросить пользователя войти заново через http://localhost:8080.

## Как отвечать

Коротко: что сделано, какие устройства задействованы и с какими значениями, что не сработало. Макросы, созданные здесь, сразу видны во вкладке «Сценарии» веб-интерфейса.
````

- [ ] **Step 7: README**

Добавить в `README.md` перед разделом «Разработка»:

```markdown
## Сценарии через Claude

Главный способ создавать сценарии — попросить Claude Code в этом проекте:
«сделай сценарий „кино“: выключи верхний свет, подсветку на 30 %». Skill
`.claude/skills/smart-home` читает каталог устройств (`GET /api/catalog`),
собирает макрос из реальных устройств, сохраняет его через API и по просьбе
запускает. Сервер должен быть запущен (Claude запустит его сам, если нужно).
```

- [ ] **Step 8: Дымовая проверка каталога без токена**

```bash
go build -o smart-house . && \
YANDEX_CLIENT_ID=x YANDEX_CLIENT_SECRET=y PORT=18080 DATA_DIR=/tmp/smart-house-smoke ./smart-house & PID=$!
sleep 1
curl -s -w '\n%{http_code}\n' localhost:18080/api/catalog
kill $PID
```

Expected: `{"error":"unauthorized",…}` и `401` — маршрут есть, без токена честно отвечает 401.

- [ ] **Step 9: Коммит**

```bash
git add internal/catalog internal/server .claude/skills/smart-home/SKILL.md README.md
git commit -m "feat: каталог устройств и skill для сборки сценариев через Claude"
```

---

### Task 13: Приёмка с реальным аккаунтом

Выполняется владельцем проекта (или основным агентом с браузером): нужны настоящие `YANDEX_CLIENT_ID`/`SECRET` и вход в Яндекс. Субагенту эту задачу не делегировать.

**Files:**
- Create: `.env` (не коммитить — в `.gitignore`)

- [ ] **Step 1: Настроить и собрать**

```bash
cp .env.example .env   # вписать реальные YANDEX_CLIENT_ID и YANDEX_CLIENT_SECRET
make build && ./smart-house
```

Expected: в логе `сервер запущен url=http://localhost:8080 authorized=false`.

- [ ] **Step 2: Войти**

Открыть http://localhost:8080 → «Открыть страницу Яндекса» → «Разрешить» → скопировать код → вставить → «Войти».
Expected: появляется дашборд; файл `data/token.json` создан с правами `-rw-------`; в логе запись `yandex api method=GET path=/v1.0/user/info status=200 request_id=…`.

- [ ] **Step 3: Проверить устройства**

- Комнаты и устройства соответствуют приложению «Дом с Алисой».
- Тумблер лампы/розетки переключает устройство; состояние подтверждается после опроса.
- Слайдер яркости/температуры применяется при отпускании.
- Показания датчиков отображаются с единицами и обновляются не реже чем раз в 10 с.
- Выключить устройство физически/из сети (если возможно) → при попытке управлять появляется ошибка вида «Питание: устройство не отвечает».

- [ ] **Step 4: Проверить сценарии и макросы**

- Вкладка «Сценарии»: запуск родного сценария показывает «Выполнено ✓».
- «+ Новый макрос» → название, 2 действия на разных устройствах → «Сохранить» → макрос в списке, `data/macros.json` обновился.
- «Запустить» макрос → устройства изменили состояние, «Выполнено ✓».
- «Изменить» → сменить действие → сохранить; «Удалить» → макрос исчез.

- [ ] **Step 5: Проверить повторный запуск и протухший токен**

- Перезапустить `./smart-house` → вход не требуется (`authorized=true` в логе).
- Испортить `access_token` в `data/token.json` (оставив `refresh_token`) → перезапустить → дашборд грузится, в логе первый запрос 401, затем 200 (токен обновился).
- Испортить и `refresh_token` → перезапустить → показывается форма входа.

- [ ] **Step 6: Проверить сборку сценария через Claude**

В Claude Code в этом проекте попросить: «сделай сценарий „тест“: выключи <устройство A>, включи <устройство B>» (подставить реальные названия). Ожидается: Claude через skill `smart-home` прочитает `GET /api/catalog`, покажет, что именно сделает, сохранит макрос (`POST /api/macros`) и предложит запустить. Макрос появляется во вкладке «Сценарии» UI; запуск меняет состояние устройств. Затем: «включи <устройство A>» — выполняется разовой командой без создания макроса.

- [ ] **Step 7: Зафиксировать результат**

Если замечания есть — завести их как отдельные bounded-задачи. Если всё прошло — коммитить нечего (`.env` и `data/` вне git), задача закрыта.

---

## Самопроверка плана

**Покрытие спеки:**
- Архитектура (один бинарник, embed, прокси, localhost) — Tasks 6, 8.
- Авторизация через код (`POST /api/auth/code`, `GET /api/auth/status`, хранение и refresh, 401 → форма входа) — Tasks 2, 3, 6, 9; сценарий проверки — Task 12.
- Бэкенд: `/api/home`, `/api/devices/actions`, `/api/scenarios/{id}/run` — Task 6; CRUD и запуск макросов — Tasks 5, 7; логирование `request_id` — Task 4; конфиг из `.env` — Task 1.
- Фронтенд: дашборд по комнатам, on_off/range/color/mode, датчики, опрос 10 с, оптимистичные тумблеры — Task 10; сценарии + макросы — Task 11.
- Сценарии через Claude: `GET /api/catalog` и skill `.claude/skills/smart-home` — Task 12; проверка — Task 13, шаг 6.
- Ошибки: человекочитаемые сообщения, разбор `action_result` по устройствам, 401 → вход — Tasks 6, 10, 11.
- Тесты Go: клиент через httptest (успех/ошибка/401), хранилище макросов, обновление токена — Tasks 3, 4, 5, 6, 7. Фронт без тестов — по спеке.
- Вне рамок (группы, история, доступ извне) — не реализуется, в UI не упоминается.

**Согласованность имён:** `yandex.TokenSource` = `{Token, Refresh}` — `auth.Manager` реализует оба; `server.Auth` = `{Authorized, LoginURL, Login}` — `auth.Manager` реализует; `server.HomeAPI` = `{UserInfo, DeviceActions, RunScenario}` — `yandex.Client` реализует; макрос на фронте и бэке — одни и те же поля `device_id/type/instance/value`; `api.js` пути совпадают с маршрутами `Handler()`.

**Известные упрощения (осознанные):** цвета `hsv`/`rgb` конвертируются через `<input type="color">` без отдельного выбора яркости; макрос, ссылающийся на удалённое устройство, показывается с первым устройством в списке до правки пользователем; опрос не приостанавливается при ошибках сети (просто показывает сообщение).
