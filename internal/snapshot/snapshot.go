// Package snapshot превращает ответ user/info Яндекса в снимок состояния дома,
// с которым работают движок правил и обработчики API.
package snapshot

import (
	"encoding/json"
	"time"
)

type Range struct {
	Min, Max float64
	HasRange bool
}

type DeviceState struct {
	ID, Name, Type, Room string
	Props                map[string]any      // instance → значение свойства
	Caps                 map[string]any      // instance → состояние умения
	CapTypes             map[string]string   // instance → тип умения (devices.capabilities.*)
	Modes                map[string][]string // instance → допустимые режимы
	Ranges               map[string]Range    // instance → диапазон
}

type Snapshot struct {
	At        time.Time
	Raw       json.RawMessage
	Devices   map[string]DeviceState
	Scenarios map[string]string
}

type rawState struct {
	Instance string `json:"instance"`
	Value    any    `json:"value"`
}

type rawUserInfo struct {
	Rooms []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"rooms"`
	Devices []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Type         string `json:"type"`
		Room         string `json:"room"`
		Capabilities []struct {
			Type       string `json:"type"`
			Parameters struct {
				Instance string `json:"instance"`
				Range    *struct {
					Min float64 `json:"min"`
					Max float64 `json:"max"`
				} `json:"range"`
				Modes []struct {
					Value string `json:"value"`
				} `json:"modes"`
				ColorModel   string `json:"color_model"`
				TemperatureK *struct {
					Min float64 `json:"min"`
					Max float64 `json:"max"`
				} `json:"temperature_k"`
			} `json:"parameters"`
			State *rawState `json:"state"`
		} `json:"capabilities"`
		Properties []struct {
			Parameters struct {
				Instance string `json:"instance"`
			} `json:"parameters"`
			State *rawState `json:"state"`
		} `json:"properties"`
	} `json:"devices"`
	Scenarios []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"scenarios"`
}

func Parse(raw json.RawMessage, at time.Time) (Snapshot, error) {
	var info rawUserInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return Snapshot{}, err
	}
	rooms := map[string]string{}
	for _, r := range info.Rooms {
		rooms[r.ID] = r.Name
	}
	s := Snapshot{At: at, Raw: raw, Devices: map[string]DeviceState{}, Scenarios: map[string]string{}}
	for _, d := range info.Devices {
		st := DeviceState{ID: d.ID, Name: d.Name, Type: d.Type, Room: rooms[d.Room],
			Props: map[string]any{}, Caps: map[string]any{}, CapTypes: map[string]string{},
			Modes: map[string][]string{}, Ranges: map[string]Range{}}
		for _, p := range d.Properties {
			if p.State != nil {
				st.Props[p.Parameters.Instance] = p.State.Value
			}
		}
		for _, c := range d.Capabilities {
			inst := c.Parameters.Instance
			switch c.Type {
			case "devices.capabilities.on_off":
				inst = "on"
			case "devices.capabilities.color_setting":
				// color_setting: сразу несколько instance; метаданные по каждому
				if c.Parameters.TemperatureK != nil {
					st.CapTypes["temperature_k"] = c.Type
					st.Ranges["temperature_k"] = Range{Min: c.Parameters.TemperatureK.Min, Max: c.Parameters.TemperatureK.Max, HasRange: true}
				}
				if c.Parameters.ColorModel != "" {
					st.CapTypes[c.Parameters.ColorModel] = c.Type
				}
				if c.State != nil {
					st.Caps[c.State.Instance] = c.State.Value
				}
				continue
			}
			st.CapTypes[inst] = c.Type
			if c.Parameters.Range != nil {
				st.Ranges[inst] = Range{Min: c.Parameters.Range.Min, Max: c.Parameters.Range.Max, HasRange: true}
			}
			if len(c.Parameters.Modes) > 0 {
				modes := make([]string, 0, len(c.Parameters.Modes))
				for _, m := range c.Parameters.Modes {
					modes = append(modes, m.Value)
				}
				st.Modes[inst] = modes
			}
			if c.State != nil {
				st.Caps[c.State.Instance] = c.State.Value
			}
		}
		s.Devices[d.ID] = st
	}
	for _, sc := range info.Scenarios {
		s.Scenarios[sc.ID] = sc.Name
	}
	return s, nil
}

// Value возвращает текущее значение свойства (приоритет) или умения по instance.
func (s Snapshot) Value(deviceID, instance string) (any, bool) {
	d, ok := s.Devices[deviceID]
	if !ok {
		return nil, false
	}
	if v, ok := d.Props[instance]; ok {
		return v, true
	}
	if v, ok := d.Caps[instance]; ok {
		return v, true
	}
	return nil, false
}
