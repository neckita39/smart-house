package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"smarthome/internal/rules"
	"smarthome/internal/snapshot"
)

type ruleView struct {
	rules.Rule
	LastFired *time.Time `json:"last_fired,omitempty"`
}

func (s *Server) ruleView(r rules.Rule) ruleView {
	v := ruleView{Rule: r}
	if s.Engine != nil {
		if t, ok := s.Engine.LastFired(r.ID); ok {
			v.LastFired = &t
		}
	}
	return v
}

func (s *Server) macroExists(id string) bool {
	if s.Macros == nil {
		return false
	}
	_, ok := s.Macros.Get(id)
	return ok
}

// snapshotOrNil возвращает указатель на последний снимок, если он уже есть;
// иначе nil — Validate тогда пропускает проверки существования устройств вместо
// того, чтобы отвергать любое правило до первого опроса дома (или пока токен мёртв).
func (s *Server) snapshotOrNil() *snapshot.Snapshot {
	if s.Snapshots == nil {
		return nil
	}
	snap, has := s.Snapshots.Latest()
	if !has {
		return nil
	}
	return &snap
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	all := s.Rules.List()
	out := make([]ruleView, 0, len(all))
	for _, x := range all {
		out = append(out, s.ruleView(x))
	}
	writeJSON(w, http.StatusOK, out)
}

// decodeRule читает правило; enabled по умолчанию true.
func decodeRule(r *http.Request) (rules.Rule, error) {
	var body struct {
		rules.Rule
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return rules.Rule{}, errors.New("некорректный JSON правила")
	}
	rule := body.Rule
	rule.Enabled = body.Enabled == nil || *body.Enabled
	return rule, nil
}

func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	rule, err := decodeRule(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
		return
	}
	created, err := s.Rules.Create(rule, s.snapshotOrNil(), s.macroExists)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, s.ruleView(created))
}

func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	rule, err := decodeRule(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
		return
	}
	rule.ID = r.PathValue("id")
	if err := s.Rules.Update(rule, s.snapshotOrNil(), s.macroExists); err != nil {
		s.writeRuleError(w, err)
		return
	}
	// Store.Update может нормализовать поля (например, дефолтный cooldown при 0) —
	// отвечаем сохранённым правилом, а не тем, что прислал клиент.
	stored, _ := s.Rules.Get(rule.ID)
	writeJSON(w, http.StatusOK, s.ruleView(stored))
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	if err := s.Rules.Delete(r.PathValue("id")); err != nil {
		s.writeRuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) runRule(w http.ResponseWriter, r *http.Request) {
	rule, ok := s.Rules.Get(r.PathValue("id"))
	if !ok {
		s.writeRuleError(w, rules.ErrNotFound)
		return
	}
	details, err := s.Runner.Execute(r.Context(), rule)
	ev := rules.Event{Time: time.Now(), RuleID: rule.ID, RuleName: rule.Name, OK: err == nil, Details: "вручную: " + details}
	if err != nil {
		ev.Details += " — " + err.Error()
	}
	if s.Events != nil {
		_ = s.Events.Append(ev)
	}
	if err == nil && s.Engine != nil {
		if snap, has := s.Snapshots.Latest(); has {
			// Отмечаем ручной запуск как срабатывание, чтобы last_fired
			// отражал его и учитывался кулдауном для автозапуска.
			s.Engine.Tick([]rules.Rule{rule}, snap)
		}
	}
	s.Snapshots.Kick()
	writeJSON(w, http.StatusOK, map[string]any{"ok": err == nil, "details": ev.Details})
}

func (s *Server) checkRule(w http.ResponseWriter, r *http.Request) {
	rule, ok := s.Rules.Get(r.PathValue("id"))
	if !ok {
		s.writeRuleError(w, rules.ErrNotFound)
		return
	}
	snap, has := s.Snapshots.Latest()
	if !has {
		var err error
		if snap, err = s.Snapshots.Refresh(r.Context()); err != nil {
			s.writeError(w, err)
			return
		}
	}
	okAll, conds := rules.Evaluate(rule, snap, time.Now())
	writeJSON(w, http.StatusOK, map[string]any{"ok": okAll, "conditions": conds,
		"snapshot_age": int(time.Since(snap.At).Seconds())})
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
	}
	if s.Events == nil {
		writeJSON(w, http.StatusOK, []rules.Event{})
		return
	}
	writeJSON(w, http.StatusOK, s.Events.List(limit))
}

func (s *Server) writeRuleError(w http.ResponseWriter, err error) {
	if errors.Is(err, rules.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorBody{Error: "not_found", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
}
