// Package catalog сжимает ответ user/info Яндекса до того, что нужно Claude,
// чтобы собирать макросы: устройства, их управляемые умения и показания.
package catalog

import (
	"encoding/json"
	"strings"
)

type Capability struct {
	Type     string   `json:"type"`
	Instance string   `json:"instance"`
	Kind     string   `json:"kind"` // bool | number | mode | color
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Step     *float64 `json:"step,omitempty"`
	Unit     string   `json:"unit,omitempty"`
	Modes    []string `json:"modes,omitempty"`
	Value    any      `json:"value,omitempty"`
}

type Property struct {
	Instance string `json:"instance"`
	Unit     string `json:"unit,omitempty"`
	Value    any    `json:"value"`
}

type Device struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Room         string       `json:"room"`
	Type         string       `json:"type"`
	Capabilities []Capability `json:"capabilities"`
	Properties   []Property   `json:"properties"`
}

type Scenario struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Catalog struct {
	Devices   []Device   `json:"devices"`
	Scenarios []Scenario `json:"scenarios"`
}

type rawState struct {
	Instance string `json:"instance"`
	Value    any    `json:"value"`
}

type rawCapability struct {
	Type       string `json:"type"`
	Parameters struct {
		Instance string `json:"instance"`
		Unit     string `json:"unit"`
		Range    *struct {
			Min       float64 `json:"min"`
			Max       float64 `json:"max"`
			Precision float64 `json:"precision"`
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
}

type rawProperty struct {
	Parameters struct {
		Instance string `json:"instance"`
		Unit     string `json:"unit"`
	} `json:"parameters"`
	State *rawState `json:"state"`
}

type rawUserInfo struct {
	Rooms []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"rooms"`
	Devices []struct {
		ID           string          `json:"id"`
		Name         string          `json:"name"`
		Type         string          `json:"type"`
		Room         string          `json:"room"`
		Capabilities []rawCapability `json:"capabilities"`
		Properties   []rawProperty   `json:"properties"`
	} `json:"devices"`
	Scenarios []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"scenarios"`
}

func Build(raw json.RawMessage) (Catalog, error) {
	var info rawUserInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return Catalog{}, err
	}
	rooms := map[string]string{}
	for _, r := range info.Rooms {
		rooms[r.ID] = r.Name
	}

	cat := Catalog{Devices: []Device{}, Scenarios: []Scenario{}}
	for _, d := range info.Devices {
		dev := Device{ID: d.ID, Name: d.Name, Room: rooms[d.Room], Type: d.Type,
			Capabilities: []Capability{}, Properties: []Property{}}
		for _, c := range d.Capabilities {
			dev.Capabilities = append(dev.Capabilities, convert(c)...)
		}
		for _, p := range d.Properties {
			prop := Property{Instance: p.Parameters.Instance, Unit: p.Parameters.Unit}
			if p.State != nil {
				prop.Value = p.State.Value
			}
			dev.Properties = append(dev.Properties, prop)
		}
		cat.Devices = append(cat.Devices, dev)
	}
	for _, s := range info.Scenarios {
		cat.Scenarios = append(cat.Scenarios, Scenario{ID: s.ID, Name: s.Name})
	}
	return cat, nil
}

// convert разворачивает одно умение Яндекса в одну или несколько записей каталога
// (color_setting может дать и temperature_k, и hsv/rgb).
func convert(c rawCapability) []Capability {
	kind := c.Type[strings.LastIndex(c.Type, ".")+1:]
	p := c.Parameters
	valueOf := func(instance string) any {
		if c.State != nil && c.State.Instance == instance {
			return c.State.Value
		}
		return nil
	}
	switch kind {
	case "on_off":
		return []Capability{{Type: c.Type, Instance: "on", Kind: "bool", Value: valueOf("on")}}
	case "toggle":
		return []Capability{{Type: c.Type, Instance: p.Instance, Kind: "bool", Value: valueOf(p.Instance)}}
	case "range":
		out := Capability{Type: c.Type, Instance: p.Instance, Kind: "number", Unit: p.Unit, Value: valueOf(p.Instance)}
		if p.Range != nil {
			out.Min, out.Max = &p.Range.Min, &p.Range.Max
			if p.Range.Precision != 0 {
				out.Step = &p.Range.Precision
			}
		}
		return []Capability{out}
	case "mode":
		modes := make([]string, 0, len(p.Modes))
		for _, m := range p.Modes {
			modes = append(modes, m.Value)
		}
		return []Capability{{Type: c.Type, Instance: p.Instance, Kind: "mode", Modes: modes, Value: valueOf(p.Instance)}}
	case "color_setting":
		var out []Capability
		if p.TemperatureK != nil {
			step := 100.0
			out = append(out, Capability{Type: c.Type, Instance: "temperature_k", Kind: "number",
				Min: &p.TemperatureK.Min, Max: &p.TemperatureK.Max, Step: &step, Value: valueOf("temperature_k")})
		}
		if p.ColorModel != "" {
			out = append(out, Capability{Type: c.Type, Instance: p.ColorModel, Kind: "color", Value: valueOf(p.ColorModel)})
		}
		return out
	}
	return nil
}
