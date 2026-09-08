package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smarthome/internal/macros"
	"smarthome/internal/rules"
	"smarthome/internal/snapshot"
)

type fakeSnapshots struct {
	snap  snapshot.Snapshot
	has   bool
	kicks int
}

func (f *fakeSnapshots) Latest() (snapshot.Snapshot, bool) { return f.snap, f.has }
func (f *fakeSnapshots) Refresh(context.Context) (snapshot.Snapshot, error) {
	f.has = true
	return f.snap, nil
}
func (f *fakeSnapshots) Kick() { f.kicks++ }

func newRulesServer(t *testing.T, home *fakeHome) (*httptest.Server, *fakeSnapshots, *rules.Store) {
	t.Helper()
	snap, err := snapshot.Parse(json.RawMessage(sampleUserInfo), time.Now().Add(-5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	fs := &fakeSnapshots{snap: snap, has: true}
	dir := t.TempDir()
	ms, _ := macros.NewStore(filepath.Join(dir, "macros.json"))
	rs, _ := rules.NewStore(filepath.Join(dir, "rules.json"))
	ev, _ := rules.NewEventLog(filepath.Join(dir, "events.jsonl"), 500)
	srv := httptest.NewServer((&Server{Auth: &fakeAuth{authorized: true}, Home: home, Macros: ms,
		Snapshots: fs, Rules: rs, Runner: &rules.Runner{Home: home, Macros: ms}, Events: ev, Engine: rules.NewEngine(nil)}).Handler())
	t.Cleanup(srv.Close)
	return srv, fs, rs
}

const ruleJSON = `{"name":"Кабинет жарко","when":[
  {"device_id":"sensor-office","property":"temperature","op":">","value":25},
  {"device_id":"blinds","capability":"open","op":">","value":50},
  {"time":{"after":"00:00","before":"23:59"}}],
 "then":[{"device_id":"blinds","type":"devices.capabilities.range","instance":"open","value":40}]}`

func TestHomeServedFromSnapshot(t *testing.T) {
	home := &fakeHome{userInfo: json.RawMessage(`{"status":"ok","devices":[]}`)}
	srv, _, _ := newRulesServer(t, home)
	req, _ := http.NewRequest("GET", srv.URL+"/api/home", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.Contains(string(body), `"sensor-office"`) {
		t.Fatalf("home должен отдаваться из снимка: %d %s", resp.StatusCode, body[:60])
	}
	if age := resp.Header.Get("X-Snapshot-Age"); age == "" {
		t.Error("нет заголовка X-Snapshot-Age")
	}
	code, body2 := do(t, "GET", srv.URL+"/api/catalog", "")
	if code != 200 || !strings.Contains(body2, `"Кабинет"`) {
		t.Errorf("catalog из снимка: %d %s", code, body2[:80])
	}
}

func TestDeviceActionsKickPoller(t *testing.T) {
	home := &fakeHome{actionsResp: json.RawMessage(`{"status":"ok","devices":[]}`)}
	srv, fs, _ := newRulesServer(t, home)
	do(t, "POST", srv.URL+"/api/devices/actions", `{"devices":[{"id":"fan","actions":[{"type":"devices.capabilities.on_off","state":{"instance":"on","value":true}}]}]}`)
	if fs.kicks != 1 {
		t.Errorf("kicks = %d, want 1", fs.kicks)
	}
}

func TestRulesCRUDCheckRunEvents(t *testing.T) {
	home := &fakeHome{actionsResp: json.RawMessage(`{"status":"ok","request_id":"r","devices":[]}`)}
	srv, _, _ := newRulesServer(t, home)

	code, body := do(t, "POST", srv.URL+"/api/rules", ruleJSON)
	if code != 201 {
		t.Fatalf("create: %d %s", code, body)
	}
	var created rules.Rule
	decode(t, body, &created)
	if created.ID == "" || !created.Enabled || created.CooldownMinutes != 30 {
		t.Errorf("created = %+v", created)
	}

	code, body = do(t, "GET", srv.URL+"/api/rules/"+created.ID+"/check", "")
	var chk struct {
		OK         bool               `json:"ok"`
		Conditions []rules.CondResult `json:"conditions"`
	}
	decode(t, body, &chk)
	if code != 200 || !chk.OK || len(chk.Conditions) != 3 || chk.Conditions[0].Current != 26.3 {
		t.Errorf("check: %d %+v", code, chk)
	}

	code, body = do(t, "POST", srv.URL+"/api/rules/"+created.ID+"/run", "")
	if code != 200 || !strings.Contains(body, `"ok":true`) || home.gotActions == nil {
		t.Errorf("run: %d %s", code, body)
	}
	code, body = do(t, "GET", srv.URL+"/api/events?limit=10", "")
	var events []rules.Event
	decode(t, body, &events)
	if code != 200 || len(events) != 1 || events[0].RuleID != created.ID || !strings.Contains(events[0].Details, "вручную") {
		t.Errorf("events: %d %+v", code, events)
	}

	created.Enabled = false
	created.CooldownMinutes = 0 // не переопределяем — сервер должен вернуть сохранённый дефолт (30)
	upd, _ := json.Marshal(created)
	if code, body = do(t, "PUT", srv.URL+"/api/rules/"+created.ID, string(upd)); code != 200 || !strings.Contains(body, `"enabled":false`) {
		t.Errorf("update: %d %s", code, body)
	}
	var updated rules.Rule
	decode(t, body, &updated)
	if updated.CooldownMinutes != 30 {
		t.Errorf("update должен отвечать сохранённым правилом (cooldown 30 по умолчанию), получили cooldown_minutes=%d в %s", updated.CooldownMinutes, body)
	}
	code, body = do(t, "GET", srv.URL+"/api/rules", "")
	if code != 200 || !strings.Contains(body, `"last_fired"`) {
		t.Errorf("list должен содержать last_fired после run: %d %s", code, body)
	}
	if code, _ = do(t, "DELETE", srv.URL+"/api/rules/"+created.ID, ""); code != 204 {
		t.Errorf("delete: %d", code)
	}
	if code, _ = do(t, "GET", srv.URL+"/api/rules/"+created.ID+"/check", ""); code != 404 {
		t.Errorf("check unknown: %d", code)
	}
}

func TestCreateRuleWithoutSnapshotSkipsDeviceChecks(t *testing.T) {
	srv, fs, _ := newRulesServer(t, &fakeHome{})
	fs.has = false // ещё не было ни одного опроса дома (или токен мёртв)
	code, body := do(t, "POST", srv.URL+"/api/rules", ruleJSON)
	if code != 201 {
		t.Errorf("create без снимка должен пройти (проверки устройств пропускаются): %d %s", code, body)
	}
}

func TestRulesValidationErrors(t *testing.T) {
	srv, _, _ := newRulesServer(t, &fakeHome{})
	code, body := do(t, "POST", srv.URL+"/api/rules", `{"name":"x","when":[{"device_id":"nope","property":"t","op":">","value":1}],"then":[{"device_id":"blinds","type":"devices.capabilities.range","instance":"open","value":40}]}`)
	if code != 400 || !strings.Contains(body, "устройство") {
		t.Errorf("validation: %d %s", code, body)
	}
	if code, _ := do(t, "POST", srv.URL+"/api/rules", `garbage`); code != 400 {
		t.Errorf("garbage: %d", code)
	}
}
