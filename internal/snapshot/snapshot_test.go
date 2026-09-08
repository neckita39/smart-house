package snapshot

import (
	"encoding/json"
	"testing"
	"time"
)

const sampleUserInfo = `{"status":"ok","request_id":"r",
 "rooms":[{"id":"r-office","name":"Кабинет","devices":["sensor-office","blinds"]},
          {"id":"r-bedroom","name":"Спальня","devices":["fan","purifier"]}],
 "devices":[
  {"id":"sensor-office","name":"Датчик климата","type":"devices.types.sensor.climate","room":"r-office",
   "capabilities":[],
   "properties":[{"type":"devices.properties.float","parameters":{"instance":"temperature","unit":"unit.temperature.celsius"},"state":{"instance":"temperature","value":26.3}},
                 {"type":"devices.properties.float","parameters":{"instance":"humidity","unit":"unit.percent"},"state":{"instance":"humidity","value":48.9}}]},
  {"id":"blinds","name":"Жалюзи в кабинете","type":"devices.types.openable.curtain","room":"r-office",
   "capabilities":[{"type":"devices.capabilities.range","parameters":{"instance":"open","unit":"unit.percent","range":{"min":0,"max":100,"precision":1}},"state":{"instance":"open","value":100}},
                   {"type":"devices.capabilities.on_off","parameters":{},"state":{"instance":"on","value":false}}],
   "properties":[]},
  {"id":"fan","name":"Вентилятор","type":"devices.types.ventilation.fan","room":"r-bedroom",
   "capabilities":[{"type":"devices.capabilities.on_off","parameters":{},"state":{"instance":"on","value":false}},
                   {"type":"devices.capabilities.mode","parameters":{"instance":"fan_speed","modes":[{"value":"low"},{"value":"medium"},{"value":"high"}]},"state":{"instance":"fan_speed","value":"low"}}],
   "properties":[]},
  {"id":"purifier","name":"Очиститель воздуха","type":"devices.types.purifier","room":"r-bedroom",
   "capabilities":[{"type":"devices.capabilities.on_off","parameters":{},"state":{"instance":"on","value":true}}],
   "properties":[{"type":"devices.properties.float","parameters":{"instance":"pm2.5_density","unit":"unit.density.mcg_m3"},"state":{"instance":"pm2.5_density","value":3}}]},
  {"id":"tv","name":"Телевизор в гостиной","type":"devices.types.media_device.tv","room":"",
   "capabilities":[{"type":"devices.capabilities.on_off","parameters":{},"state":null}],
   "properties":[]}
 ],
 "scenarios":[{"id":"s1","name":"Я дома","is_active":true}]}`

var at = time.Date(2026, 9, 8, 12, 0, 0, 0, time.Local)

func TestParseBuildsDeviceStates(t *testing.T) {
	s, err := Parse(json.RawMessage(sampleUserInfo), at)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !s.At.Equal(at) || string(s.Raw) != sampleUserInfo {
		t.Error("At/Raw не сохранены")
	}
	if len(s.Devices) != 5 {
		t.Fatalf("devices = %d, want 5", len(s.Devices))
	}
	d := s.Devices["sensor-office"]
	if d.Name != "Датчик климата" || d.Room != "Кабинет" || d.Type != "devices.types.sensor.climate" {
		t.Errorf("sensor = %+v", d)
	}
	if v, ok := s.Value("sensor-office", "temperature"); !ok || v != 26.3 {
		t.Errorf("temperature = %v, %v", v, ok)
	}
	b := s.Devices["blinds"]
	if v, ok := s.Value("blinds", "open"); !ok || v != float64(100) {
		t.Errorf("open = %v, %v", v, ok)
	}
	if b.CapTypes["open"] != "devices.capabilities.range" || !b.Ranges["open"].HasRange || b.Ranges["open"].Max != 100 {
		t.Errorf("blinds meta = %+v", b)
	}
	f := s.Devices["fan"]
	if got := f.Modes["fan_speed"]; len(got) != 3 || got[1] != "medium" {
		t.Errorf("fan modes = %v", got)
	}
	if v, ok := s.Value("fan", "on"); !ok || v != false {
		t.Errorf("fan on = %v, %v", v, ok)
	}
	if _, ok := s.Value("tv", "on"); ok {
		t.Error("state:null должен давать ok=false")
	}
	if _, ok := s.Value("nope", "on"); ok {
		t.Error("неизвестное устройство должно давать ok=false")
	}
	if s.Scenarios["s1"] != "Я дома" {
		t.Errorf("scenarios = %v", s.Scenarios)
	}
}

func TestParseRejectsBadJSON(t *testing.T) {
	if _, err := Parse(json.RawMessage(`nope`), at); err == nil {
		t.Fatal("ожидалась ошибка")
	}
}

const colorAndThermostat = `{"status":"ok","rooms":[],"devices":[
 {"id":"lamp","name":"Лампа","type":"devices.types.light","room":"",
  "capabilities":[
   {"type":"devices.capabilities.on_off","parameters":{},"state":{"instance":"on","value":true}},
   {"type":"devices.capabilities.color_setting","parameters":{"color_model":"hsv","temperature_k":{"min":2700,"max":6500}},"state":{"instance":"temperature_k","value":4000}}],
  "properties":[]},
 {"id":"heater","name":"Батареи","type":"devices.types.thermostat","room":"",
  "capabilities":[{"type":"devices.capabilities.range","parameters":{"instance":"temperature","unit":"unit.temperature.celsius","range":{"min":5,"max":30,"precision":1}},"state":{"instance":"temperature","value":22}}],
  "properties":[{"type":"devices.properties.float","parameters":{"instance":"temperature","unit":"unit.temperature.celsius"},"state":{"instance":"temperature","value":25.9}}]}
],"scenarios":[]}`

func TestParseColorSettingAndPropsPrecedence(t *testing.T) {
	s, err := Parse(json.RawMessage(colorAndThermostat), at)
	if err != nil {
		t.Fatal(err)
	}
	lamp := s.Devices["lamp"]
	if lamp.CapTypes["temperature_k"] != "devices.capabilities.color_setting" || lamp.CapTypes["hsv"] != "devices.capabilities.color_setting" {
		t.Errorf("color_setting не разложен на temperature_k и hsv: %+v", lamp.CapTypes)
	}
	if r := lamp.Ranges["temperature_k"]; !r.HasRange || r.Min != 2700 || r.Max != 6500 {
		t.Errorf("temperature_k range = %+v", r)
	}
	if v, ok := s.Value("lamp", "temperature_k"); !ok || v != float64(4000) {
		t.Errorf("temperature_k value = %v %v", v, ok)
	}
	if _, ok := s.Value("lamp", "hsv"); ok {
		t.Error("hsv без state не должен иметь значения")
	}
	// Свойство «температура» (факт 25.9) важнее умения «температура» (задано 22).
	if v, ok := s.Value("heater", "temperature"); !ok || v != 25.9 {
		t.Errorf("heater temperature = %v %v, want 25.9 (свойство приоритетнее умения)", v, ok)
	}
	if heater := s.Devices["heater"]; heater.Caps["temperature"] != float64(22) || heater.CapTypes["temperature"] != "devices.capabilities.range" {
		t.Errorf("heater caps = %+v types = %+v", heater.Caps, heater.CapTypes)
	}
}
