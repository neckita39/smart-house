package macros

import (
	"errors"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data", "macros.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s, path
}

func sample() Macro {
	return Macro{Name: "Кино", Actions: []Action{
		{DeviceID: "lamp", Type: "devices.capabilities.on_off", Instance: "on", Value: false},
		{DeviceID: "led", Type: "devices.capabilities.on_off", Instance: "on", Value: true},
		{DeviceID: "led", Type: "devices.capabilities.range", Instance: "brightness", Value: 30},
	}}
}

func TestCreateAssignsIDAndPersists(t *testing.T) {
	s, path := newTestStore(t)

	created, err := s.Create(sample())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("ID не назначен")
	}
	if got := s.List(); len(got) != 1 || got[0].ID != created.ID {
		t.Errorf("List = %+v", got)
	}

	reloaded, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := reloaded.Get(created.ID)
	if !ok || m.Name != "Кино" || len(m.Actions) != 3 {
		t.Errorf("после перезагрузки: ok=%v, macro=%+v", ok, m)
	}
}

func TestListOnEmptyStoreIsEmptySlice(t *testing.T) {
	s, _ := newTestStore(t)
	if got := s.List(); got == nil || len(got) != 0 {
		t.Errorf("List = %#v, want пустой (не nil) срез", got)
	}
}

func TestCreateValidates(t *testing.T) {
	s, _ := newTestStore(t)
	cases := map[string]Macro{
		"без имени":     {Name: " ", Actions: sample().Actions},
		"без действий":  {Name: "x"},
		"без device_id": {Name: "x", Actions: []Action{{Type: "t", Instance: "i"}}},
	}
	for name, m := range cases {
		if _, err := s.Create(m); err == nil {
			t.Errorf("%s: ожидалась ошибка валидации", name)
		}
	}
}

func TestUpdateReplacesAndPersists(t *testing.T) {
	s, path := newTestStore(t)
	created, _ := s.Create(sample())

	created.Name = "Кино 2"
	created.Actions = created.Actions[:1]
	if err := s.Update(created); err != nil {
		t.Fatalf("Update: %v", err)
	}
	reloaded, _ := NewStore(path)
	m, _ := reloaded.Get(created.ID)
	if m.Name != "Кино 2" || len(m.Actions) != 1 {
		t.Errorf("после Update: %+v", m)
	}
}

func TestUpdateUnknownReturnsErrNotFound(t *testing.T) {
	s, _ := newTestStore(t)
	m := sample()
	m.ID = "nope"
	if err := s.Update(m); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestDeleteRemoves(t *testing.T) {
	s, path := newTestStore(t)
	created, _ := s.Create(sample())

	if err := s.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get(created.ID); ok {
		t.Error("макрос остался после Delete")
	}
	if err := s.Delete(created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("повторный Delete err = %v, want ErrNotFound", err)
	}
	reloaded, _ := NewStore(path)
	if len(reloaded.List()) != 0 {
		t.Error("удаление не сохранилось на диск")
	}
}

func TestToActionsRequestGroupsByDevice(t *testing.T) {
	req := sample().ToActionsRequest()
	if len(req.Devices) != 2 {
		t.Fatalf("devices = %d, want 2 (lamp, led)", len(req.Devices))
	}
	if req.Devices[0].ID != "lamp" || len(req.Devices[0].Actions) != 1 {
		t.Errorf("devices[0] = %+v", req.Devices[0])
	}
	led := req.Devices[1]
	if led.ID != "led" || len(led.Actions) != 2 {
		t.Fatalf("devices[1] = %+v", led)
	}
	if led.Actions[1].Type != "devices.capabilities.range" || led.Actions[1].State.Instance != "brightness" || led.Actions[1].State.Value != 30 {
		t.Errorf("led.Actions[1] = %+v", led.Actions[1])
	}
}
