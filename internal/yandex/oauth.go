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

// OAuthError — отказ сервера oauth.yandex.ru (invalid_grant, invalid_client …).
type OAuthError struct {
	Status      int
	Code        string
	Description string
}

func (e *OAuthError) Error() string {
	return fmt.Sprintf("oauth: HTTP %d %s: %s", e.Status, e.Code, e.Description)
}

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
		code, desc := payload.Error, payload.ErrorDescription
		if code == "" {
			code, desc = "http_error", truncate(body)
		}
		return Token{}, &OAuthError{Status: resp.StatusCode, Code: code, Description: desc}
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
