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
