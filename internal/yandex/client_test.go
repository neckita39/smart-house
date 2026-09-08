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
		ID:      "d1",
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
