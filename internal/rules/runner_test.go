package rules

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smarthome/internal/macros"
	"smarthome/internal/yandex"
)

type fakeHome struct {
	actions   *yandex.ActionsRequest
	resp      string
	err       error
	scenarios []string
}

func (f *fakeHome) DeviceActions(_ context.Context, req yandex.ActionsRequest) (json.RawMessage, error) {
	f.actions = &req
	if f.err != nil {
		return nil, f.err
	}
	return json.RawMessage(f.resp), nil
}

func (f *fakeHome) RunScenario(_ context.Context, id string) (json.RawMessage, error) {
	f.scenarios = append(f.scenarios, id)
	return json.RawMessage(`{"status":"ok","request_id":"s"}`), f.err
}

const okResp = `{"status":"ok","request_id":"r","devices":[{"id":"blinds","capabilities":[{"type":"devices.capabilities.range","state":{"instance":"open","action_result":{"status":"DONE"}}}]}]}`
const failResp = `{"status":"ok","request_id":"r","devices":[{"id":"blinds","capabilities":[{"type":"devices.capabilities.range","state":{"instance":"open","action_result":{"status":"ERROR","error_code":"DEVICE_UNREACHABLE","error_message":"Устройство не отвечает"}}}]}]}`

func TestExecuteGroupsDeviceActionsAndMacros(t *testing.T) {
	ms, _ := macros.NewStore(filepath.Join(t.TempDir(), "macros.json"))
	m, _ := ms.Create(macros.Macro{Name: "Свет", Actions: []macros.Action{
		{DeviceID: "fan", Type: "devices.capabilities.on_off", Instance: "on", Value: true},
		{DeviceID: "blinds", Type: "devices.capabilities.on_off", Instance: "on", Value: true},
	}})
	home := &fakeHome{resp: okResp}
	r := &Runner{Home: home, Macros: ms}
	rule := Rule{ID: "r1", Name: "Кабинет жарко", Then: []Action{
		{DeviceID: "blinds", Type: "devices.capabilities.range", Instance: "open", Value: 40},
		{MacroID: m.ID},
		{ScenarioID: "s1"},
	}}

	details, err := r.Execute(context.Background(), rule)
	if err != nil {
		t.Fatalf("Execute: %v (%s)", err, details)
	}
	if home.actions == nil || len(home.actions.Devices) != 2 {
		t.Fatalf("devices in request = %+v, want 2 (blinds, fan)", home.actions)
	}
	var blinds yandex.DeviceActions
	for _, d := range home.actions.Devices {
		if d.ID == "blinds" {
			blinds = d
		}
	}
	if len(blinds.Actions) != 2 {
		t.Errorf("blinds actions = %+v, want range+on_off в одном запросе", blinds.Actions)
	}
	if len(home.scenarios) != 1 || home.scenarios[0] != "s1" {
		t.Errorf("scenarios = %v", home.scenarios)
	}
	if !strings.Contains(details, "устройств: 2") || !strings.Contains(details, "сценариев: 1") {
		t.Errorf("details = %q", details)
	}
}

func TestExecuteReportsActionResultErrors(t *testing.T) {
	home := &fakeHome{resp: failResp}
	r := &Runner{Home: home}
	rule := Rule{ID: "r1", Name: "x", Then: []Action{{DeviceID: "blinds", Type: "devices.capabilities.range", Instance: "open", Value: 40}}}
	details, err := r.Execute(context.Background(), rule)
	if err == nil || !strings.Contains(details, "blinds") || !strings.Contains(details, "не отвечает") {
		t.Errorf("err=%v details=%q", err, details)
	}
}

func TestExecuteTransportError(t *testing.T) {
	r := &Runner{Home: &fakeHome{err: errors.New("yandex api: HTTP 500")}}
	rule := Rule{ID: "r1", Name: "x", Then: []Action{{DeviceID: "blinds", Type: "devices.capabilities.range", Instance: "open", Value: 40}}}
	if _, err := r.Execute(context.Background(), rule); err == nil {
		t.Fatal("ожидалась ошибка")
	}
}

func TestEventLogAppendsAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "events.jsonl")
	l, err := NewEventLog(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 4; i++ {
		if err := l.Append(Event{Time: time.Date(2026, 9, 8, 12, i, 0, 0, time.Local), RuleID: "r", RuleName: "Правило", OK: i%2 == 0, Details: "d"}); err != nil {
			t.Fatal(err)
		}
	}
	got := l.List(10)
	if len(got) != 3 || got[0].Time.Minute() != 4 || got[2].Time.Minute() != 2 {
		t.Errorf("List = %+v (ожидались 3 последних, новые сверху)", got)
	}
	if got := l.List(1); len(got) != 1 || got[0].Time.Minute() != 4 {
		t.Errorf("List(1) = %+v", got)
	}
	data, _ := os.ReadFile(path)
	if lines := strings.Count(strings.TrimSpace(string(data)), "\n") + 1; lines != 4 {
		t.Errorf("в файле %d строк, want 4 (файл не усекается)", lines)
	}
	re, err := NewEventLog(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	if got := re.List(10); len(got) != 3 || got[0].Time.Minute() != 4 {
		t.Errorf("после перезагрузки List = %+v", got)
	}
}
