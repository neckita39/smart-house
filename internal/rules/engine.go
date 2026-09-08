package rules

import (
	"sync"
	"time"

	"smarthome/internal/snapshot"
)

type CondResult struct {
	Index   int    `json:"index"`
	OK      bool   `json:"ok"`
	Current any    `json:"current,omitempty"`
	Kind    string `json:"kind"`
}

// Evaluate вычисляет все условия правила на снимке; ok — все истинны.
func Evaluate(r Rule, snap snapshot.Snapshot, now time.Time) (bool, []CondResult) {
	all := true
	res := make([]CondResult, 0, len(r.When))
	for i, c := range r.When {
		cr := CondResult{Index: i}
		switch {
		case c.Time != nil:
			cr.Kind = "time"
			cr.Current = now.Format("15:04")
			cr.OK = inWindow(*c.Time, now)
		default:
			if c.Property != "" {
				cr.Kind = "property"
			} else {
				cr.Kind = "capability"
			}
			cur, ok := snap.Value(c.DeviceID, c.instance())
			cr.Current = cur
			cr.OK = ok && compare(cur, c.Op, c.Value)
		}
		if !cr.OK {
			all = false
		}
		res = append(res, cr)
	}
	return all, res
}

func inWindow(w TimeWindow, now time.Time) bool {
	after, err1 := parseClock(w.After)
	before, err2 := parseClock(w.Before)
	if err1 != nil || err2 != nil {
		return false
	}
	cur := now.Hour()*60 + now.Minute()
	if after <= before {
		return cur >= after && cur < before
	}
	return cur >= after || cur < before // через полночь
}

func compare(cur any, op string, want any) bool {
	if cf, ok1 := toFloat(cur); ok1 {
		wf, ok2 := toFloat(want)
		if !ok2 {
			return false
		}
		switch op {
		case "<":
			return cf < wf
		case "<=":
			return cf <= wf
		case ">":
			return cf > wf
		case ">=":
			return cf >= wf
		case "==":
			return cf == wf
		case "!=":
			return cf != wf
		}
		return false
	}
	switch op {
	case "==":
		return cur == want
	case "!=":
		return cur != want
	}
	return false
}

type ruleState struct {
	trueSince time.Time
	armed     bool
	lastFired time.Time
}

// Engine хранит состояние срабатывания правил между снимками.
type Engine struct {
	now   func() time.Time
	mu    sync.Mutex
	state map[string]*ruleState
}

func NewEngine(now func() time.Time) *Engine {
	if now == nil {
		now = time.Now
	}
	return &Engine{now: now, state: map[string]*ruleState{}}
}

func (e *Engine) LastFired(id string) (time.Time, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	st, ok := e.state[id]
	if !ok || st.lastFired.IsZero() {
		return time.Time{}, false
	}
	return st.lastFired, true
}

// Tick оценивает правила на снимке и возвращает сработавшие.
func (e *Engine) Tick(all []Rule, snap snapshot.Snapshot) []Rule {
	now := e.now()
	e.mu.Lock()
	defer e.mu.Unlock()
	var fired []Rule
	for _, r := range all {
		st, ok := e.state[r.ID]
		if !ok {
			st = &ruleState{armed: true}
			e.state[r.ID] = st
		}
		if !r.Enabled {
			st.trueSince, st.armed = time.Time{}, true
			continue
		}
		ok, _ = Evaluate(r, snap, now)
		if !ok {
			st.trueSince, st.armed = time.Time{}, true
			continue
		}
		if st.trueSince.IsZero() {
			st.trueSince = now
		}
		if !st.armed {
			continue
		}
		if now.Sub(st.trueSince) < time.Duration(r.ForMinutes)*time.Minute {
			continue
		}
		if !st.lastFired.IsZero() && now.Sub(st.lastFired) < time.Duration(r.CooldownMinutes)*time.Minute {
			continue
		}
		st.lastFired, st.armed = now, false
		fired = append(fired, r)
	}
	return fired
}
