// Package server — HTTP API для фронтенда и раздача собранного UI.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"

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
	mux.HandleFunc("GET /api/catalog", s.catalog)
	mux.HandleFunc("POST /api/devices/actions", s.deviceActions)
	mux.HandleFunc("POST /api/scenarios/{id}/run", s.runScenario)
	mux.HandleFunc("GET /api/macros", s.listMacros)
	mux.HandleFunc("POST /api/macros", s.createMacro)
	mux.HandleFunc("PUT /api/macros/{id}", s.updateMacro)
	mux.HandleFunc("DELETE /api/macros/{id}", s.deleteMacro)
	mux.HandleFunc("POST /api/macros/{id}/run", s.runMacro)
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
	return localOnlyAPI(mux)
}

// localOnlyAPI защищает /api/ от чужих сайтов в браузере владельца: без preflight
// (простой fetch с mode:'no-cors') запрос всё равно уйдёт, но не пройдёт проверку
// Host/Origin. Запросы без Origin (curl, skill) и раздача UI не затрагиваются.
func localOnlyAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if !isLocalHost(r.Host) {
			forbidden(w)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || !isLocalHost(u.Host) {
				forbidden(w)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isLocalHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func forbidden(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, errorBody{Error: "forbidden",
		Message: "запросы к API принимаются только с этого компьютера"})
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
