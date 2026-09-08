package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"smarthome/internal/macros"
	"smarthome/internal/yandex"
)

type Home interface {
	DeviceActions(ctx context.Context, req yandex.ActionsRequest) (json.RawMessage, error)
	RunScenario(ctx context.Context, id string) (json.RawMessage, error)
}

// Runner выполняет действия правила: устройства — одним запросом, сценарии — по очереди.
type Runner struct {
	Home   Home
	Macros *macros.Store
}

func (r *Runner) Execute(ctx context.Context, rule Rule) (string, error) {
	var acts []macros.Action
	var scenarios []string
	for _, a := range rule.Then {
		switch {
		case a.MacroID != "":
			if r.Macros == nil {
				return "", fmt.Errorf("макросы недоступны")
			}
			m, ok := r.Macros.Get(a.MacroID)
			if !ok {
				return "", fmt.Errorf("макрос %s не найден", a.MacroID)
			}
			acts = append(acts, m.Actions...)
		case a.ScenarioID != "":
			scenarios = append(scenarios, a.ScenarioID)
		default:
			acts = append(acts, macros.Action{DeviceID: a.DeviceID, Type: a.Type, Instance: a.Instance, Value: a.Value})
		}
	}

	var problems []string
	devices := 0
	if len(acts) > 0 {
		req := macros.Macro{Actions: acts}.ToActionsRequest()
		devices = len(req.Devices)
		raw, err := r.Home.DeviceActions(ctx, req)
		if err != nil {
			return fmt.Sprintf("устройств: %d, сценариев: %d", devices, len(scenarios)), err
		}
		problems = append(problems, actionErrors(raw)...)
	}
	for _, id := range scenarios {
		if _, err := r.Home.RunScenario(ctx, id); err != nil {
			problems = append(problems, fmt.Sprintf("сценарий %s: %v", id, err))
		}
	}
	details := fmt.Sprintf("устройств: %d, сценариев: %d", devices, len(scenarios))
	if len(problems) > 0 {
		details += "; ошибки: " + strings.Join(problems, "; ")
		return details, errors.New("часть действий не выполнена")
	}
	return details, nil
}

// actionErrors извлекает из ответа devices/actions действия со статусом ≠ DONE.
func actionErrors(raw json.RawMessage) []string {
	var resp struct {
		Devices []struct {
			ID           string `json:"id"`
			Capabilities []struct {
				State struct {
					Instance     string `json:"instance"`
					ActionResult struct {
						Status       string `json:"status"`
						ErrorCode    string `json:"error_code"`
						ErrorMessage string `json:"error_message"`
					} `json:"action_result"`
				} `json:"state"`
			} `json:"capabilities"`
		} `json:"devices"`
	}
	if json.Unmarshal(raw, &resp) != nil {
		return nil
	}
	var out []string
	for _, d := range resp.Devices {
		for _, c := range d.Capabilities {
			ar := c.State.ActionResult
			if ar.Status != "" && ar.Status != "DONE" {
				msg := ar.ErrorMessage
				if msg == "" {
					msg = ar.ErrorCode
				}
				out = append(out, fmt.Sprintf("%s/%s: %s", d.ID, c.State.Instance, msg))
			}
		}
	}
	return out
}
