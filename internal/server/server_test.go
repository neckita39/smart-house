package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
func (f *fakeAuth) LoginURL() string {
	return "https://oauth.yandex.ru/authorize?client_id=cid&response_type=code"
}
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
		{"мёртвый токен (обёрнутый ErrNoToken)", fmt.Errorf("%w: oauth: invalid_grant", auth.ErrNoToken), 401},
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

func TestUnknownAPIRouteIs404(t *testing.T) {
	srv := newTestServer(t, &fakeHome{}, &fakeAuth{})
	if code, _ := do(t, "GET", srv.URL+"/api/nope", ""); code != 404 {
		t.Errorf("status %d, want 404", code)
	}
}

func TestLocalOnlyRejectsForeignOriginAndHost(t *testing.T) {
	srv := newTestServer(t, &fakeHome{}, &fakeAuth{authorized: true})

	get := func(origin, host string) (int, string) {
		req, err := http.NewRequest("GET", srv.URL+"/api/auth/status", nil)
		if err != nil {
			t.Fatal(err)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if host != "" {
			req.Host = host
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	if code, body := get("https://evil.example", ""); code != 403 {
		t.Errorf("чужой Origin: status %d, want 403 (body %s)", code, body)
	}
	if code, body := get("", ""); code != 200 {
		t.Errorf("без Origin: status %d, want 200 (body %s)", code, body)
	}
	if code, body := get("http://localhost:5173", ""); code != 200 {
		t.Errorf("Origin с dev-сервера Vite: status %d, want 200 (body %s)", code, body)
	}
	if code, body := get("", "evil.example"); code != 403 {
		t.Errorf("чужой Host: status %d, want 403 (body %s)", code, body)
	}
}
