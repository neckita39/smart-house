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
