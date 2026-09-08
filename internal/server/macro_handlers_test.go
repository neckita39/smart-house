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
