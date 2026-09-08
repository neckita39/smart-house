// Package rules — локальные автоматизации: правила «условия → действия»,
// проверяемые на каждом снимке состояния дома.
package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"smarthome/internal/snapshot"
)

const DefaultCooldownMinutes = 30

var ErrNotFound = errors.New("правило не найдено")

type TimeWindow struct {
	After  string `json:"after"`
	Before string `json:"before"`
}

type Condition struct {
	DeviceID   string      `json:"device_id,omitempty"`
	Property   string      `json:"property,omitempty"`
	Capability string      `json:"capability,omitempty"`
	Op         string      `json:"op,omitempty"`
	Value      any         `json:"value,omitempty"`
	Time       *TimeWindow `json:"time,omitempty"`
}

type Action struct {
	DeviceID   string `json:"device_id,omitempty"`
	Type       string `json:"type,omitempty"`
	Instance   string `json:"instance,omitempty"`
	Value      any    `json:"value,omitempty"`
	MacroID    string `json:"macro_id,omitempty"`
	ScenarioID string `json:"scenario_id,omitempty"`
}

type Rule struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Enabled         bool        `json:"enabled"`
	When            []Condition `json:"when"`
	ForMinutes      int         `json:"for_minutes"`
	CooldownMinutes int         `json:"cooldown_minutes"`
	Then            []Action    `json:"then"`
}

var validOps = map[string]bool{"<": true, "<=": true, ">": true, ">=": true, "==": true, "!=": true}

// instance условия/действия: свойство или умение.
func (c Condition) instance() string {
	if c.Property != "" {
		return c.Property
	}
	return c.Capability
}

// Validate проверяет форму правила и, если передан снимок, существование
// устройств, instance, режимов и диапазонов. macroExists может быть nil.
func (r Rule) Validate(snap *snapshot.Snapshot, macroExists func(string) bool) error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("у правила должно быть имя")
	}
	if len(r.When) == 0 {
		return errors.New("правило должно содержать хотя бы одно условие")
	}
	if len(r.Then) == 0 {
		return errors.New("правило должно содержать хотя бы одно действие")
	}
	if r.ForMinutes < 0 || r.CooldownMinutes < 0 {
		return errors.New("for_minutes и cooldown_minutes не могут быть отрицательными")
	}
	for i, c := range r.When {
		if err := c.validate(snap); err != nil {
			return fmt.Errorf("условие %d: %w", i+1, err)
		}
	}
	for i, a := range r.Then {
		if err := a.validate(snap, macroExists); err != nil {
			return fmt.Errorf("действие %d: %w", i+1, err)
		}
	}
	return nil
}

func (c Condition) validate(snap *snapshot.Snapshot) error {
	kinds := 0
	if c.Property != "" {
		kinds++
	}
	if c.Capability != "" {
		kinds++
	}
	if c.Time != nil {
		kinds++
	}
	if kinds != 1 {
		return errors.New("нужно ровно одно из: property, capability или time")
	}
	if c.Time != nil {
		if _, err := parseClock(c.Time.After); err != nil {
			return fmt.Errorf("time.after: %w", err)
		}
		if _, err := parseClock(c.Time.Before); err != nil {
			return fmt.Errorf("time.before: %w", err)
		}
		return nil
	}
	if c.DeviceID == "" {
		return errors.New("нужен device_id")
	}
	if !validOps[c.Op] {
		return fmt.Errorf("неизвестный оператор %q (допустимы < <= > >= == !=)", c.Op)
	}
	if c.Value == nil {
		return errors.New("нужно значение value")
	}
	if snap == nil {
		return nil
	}
	d, ok := snap.Devices[c.DeviceID]
	if !ok {
		return fmt.Errorf("устройство %s не найдено в доме", c.DeviceID)
	}
	inst := c.instance()
	if _, ok := d.Props[inst]; ok {
		return nil
	}
	if _, ok := d.Caps[inst]; ok {
		return nil
	}
	if _, ok := d.CapTypes[inst]; ok {
		return nil
	}
	return fmt.Errorf("у устройства «%s» нет показания или умения %q", d.Name, inst)
}

func (a Action) validate(snap *snapshot.Snapshot, macroExists func(string) bool) error {
	switch {
	case a.MacroID != "":
		if macroExists != nil && !macroExists(a.MacroID) {
			return fmt.Errorf("макрос %s не найден", a.MacroID)
		}
		return nil
	case a.ScenarioID != "":
		if snap != nil {
			if _, ok := snap.Scenarios[a.ScenarioID]; !ok {
				return fmt.Errorf("сценарий Яндекса %s не найден", a.ScenarioID)
			}
		}
		return nil
	}
	if a.DeviceID == "" || a.Type == "" || a.Instance == "" {
		return errors.New("нужны device_id, type и instance (или macro_id / scenario_id)")
	}
	if snap == nil {
		return nil
	}
	d, ok := snap.Devices[a.DeviceID]
	if !ok {
		return fmt.Errorf("устройство %s не найдено в доме", a.DeviceID)
	}
	if _, ok := d.CapTypes[a.Instance]; !ok {
		if _, ok := d.Caps[a.Instance]; !ok {
			return fmt.Errorf("у устройства «%s» нет умения %q", d.Name, a.Instance)
		}
	}
	if modes, ok := d.Modes[a.Instance]; ok {
		s, isStr := a.Value.(string)
		found := false
		for _, m := range modes {
			if isStr && m == s {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("у «%s» нет режима %v (доступны: %s)", d.Name, a.Value, strings.Join(modes, ", "))
		}
	}
	if rg, ok := d.Ranges[a.Instance]; ok && rg.HasRange {
		f, isNum := toFloat(a.Value)
		if !isNum || f < rg.Min || f > rg.Max {
			return fmt.Errorf("значение %v для «%s» вне диапазона %g..%g", a.Value, d.Name, rg.Min, rg.Max)
		}
	}
	return nil
}

// parseClock разбирает "HH:MM" в минуты от полуночи.
func parseClock(s string) (int, error) {
	hh, mm, ok := strings.Cut(s, ":")
	if !ok {
		return 0, fmt.Errorf("ожидается ЧЧ:ММ, получено %q", s)
	}
	h, err1 := strconv.Atoi(hh)
	m, err2 := strconv.Atoi(mm)
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("ожидается ЧЧ:ММ, получено %q", s)
	}
	return h*60 + m, nil
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}
