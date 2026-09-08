package rules

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smarthome/internal/snapshot"
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

func snap(t *testing.T) *snapshot.Snapshot {
	t.Helper()
	s, err := snapshot.Parse(json.RawMessage(sampleUserInfo), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return &s
}

func officeHot() Rule {
	return Rule{Name: "Кабинет жарко", Enabled: true,
		When: []Condition{
			{DeviceID: "sensor-office", Property: "temperature", Op: ">", Value: 25},
			{DeviceID: "blinds", Capability: "open", Op: ">", Value: 50},
			{Time: &TimeWindow{After: "11:00", Before: "17:00"}},
		},
		Then: []Action{{DeviceID: "blinds", Type: "devices.capabilities.range", Instance: "open", Value: 40}},
	}
}

func TestCreateDefaultsAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.json")
	st, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := st.Create(officeHot(), snap(t), func(string) bool { return false })
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.ID == "" || r.CooldownMinutes != DefaultCooldownMinutes {
		t.Errorf("created = %+v", r)
	}
	re, _ := NewStore(path)
	got, ok := re.Get(r.ID)
	if !ok || got.Name != "Кабинет жарко" || len(got.When) != 3 || got.When[2].Time.After != "11:00" {
		t.Errorf("reloaded = %+v ok=%v", got, ok)
	}
	if got.When[0].Value != float64(25) {
		t.Errorf("value after JSON = %#v, want float64(25)", got.When[0].Value)
	}
}

func TestValidateRejectsBadRules(t *testing.T) {
	s := snap(t)
	noMacro := func(string) bool { return false }
	cases := map[string]Rule{
		"без имени":                       {When: officeHot().When, Then: officeHot().Then},
		"без условий":                     {Name: "x", Then: officeHot().Then},
		"без действий":                    {Name: "x", When: officeHot().When},
		"неизвестное устройство":          {Name: "x", When: []Condition{{DeviceID: "nope", Property: "temperature", Op: ">", Value: 1}}, Then: officeHot().Then},
		"неизвестный instance":            {Name: "x", When: []Condition{{DeviceID: "sensor-office", Property: "co2", Op: ">", Value: 1}}, Then: officeHot().Then},
		"плохой оператор":                 {Name: "x", When: []Condition{{DeviceID: "sensor-office", Property: "temperature", Op: "~", Value: 1}}, Then: officeHot().Then},
		"плохое время":                    {Name: "x", When: []Condition{{Time: &TimeWindow{After: "25:00", Before: "17:00"}}}, Then: officeHot().Then},
		"условие без вида":                {Name: "x", When: []Condition{{Op: ">", Value: 1}}, Then: officeHot().Then},
		"режим вне списка":                {Name: "x", When: officeHot().When, Then: []Action{{DeviceID: "fan", Type: "devices.capabilities.mode", Instance: "fan_speed", Value: "turbo"}}},
		"число вне диапазона":             {Name: "x", When: officeHot().When, Then: []Action{{DeviceID: "blinds", Type: "devices.capabilities.range", Instance: "open", Value: 140}}},
		"неизвестный макрос":              {Name: "x", When: officeHot().When, Then: []Action{{MacroID: "m1"}}},
		"неизвестный сценарий":            {Name: "x", When: officeHot().When, Then: []Action{{ScenarioID: "s9"}}},
		"умение не у устройства":          {Name: "x", When: officeHot().When, Then: []Action{{DeviceID: "fan", Type: "devices.capabilities.range", Instance: "brightness", Value: 10}}},
		"instance-свойство вместо умения": {Name: "x", When: officeHot().When, Then: []Action{{DeviceID: "sensor-office", Type: "devices.capabilities.range", Instance: "temperature", Value: 10}}},
	}
	for name, r := range cases {
		if err := r.Validate(s, noMacro); err == nil {
			t.Errorf("%s: ожидалась ошибка", name)
		}
	}
	ok := officeHot()
	ok.Then = append(ok.Then, Action{MacroID: "m1"}, Action{ScenarioID: "s1"})
	if err := ok.Validate(s, func(id string) bool { return id == "m1" }); err != nil {
		t.Errorf("валидное правило отвергнуто: %v", err)
	}
	if err := officeHot().Validate(nil, noMacro); err != nil {
		t.Errorf("без снимка проверки существования должны пропускаться: %v", err)
	}
}

func TestUpdateDeleteNotFound(t *testing.T) {
	st, _ := NewStore(filepath.Join(t.TempDir(), "rules.json"))
	r, _ := st.Create(officeHot(), nil, nil)
	r.Name = "Кабинет жарко (v2)"
	r.Enabled = false
	if err := st.Update(r, nil, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := st.Get(r.ID)
	if got.Name != "Кабинет жарко (v2)" || got.Enabled {
		t.Errorf("after update = %+v", got)
	}
	bad := r
	bad.ID = "nope"
	if err := st.Update(bad, nil, nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("update unknown: %v", err)
	}
	if err := st.Delete(r.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.Delete(r.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if got := st.List(); got == nil || len(got) != 0 {
		t.Errorf("List = %#v, want пустой срез", got)
	}
}

func TestValidateMessagesAreRussian(t *testing.T) {
	err := Rule{Name: "x", When: []Condition{{DeviceID: "nope", Property: "t", Op: ">", Value: 1}}, Then: officeHot().Then}.Validate(snap(t), nil)
	if err == nil || !strings.Contains(err.Error(), "устройство") {
		t.Errorf("err = %v", err)
	}
}
