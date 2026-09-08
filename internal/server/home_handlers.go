package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"smarthome/internal/yandex"
)

// homeRaw возвращает сырой ответ user/info и время, на которое он актуален.
// Если s.Snapshots не задан (старое поведение) — ходит в Яндекс напрямую.
// Иначе берёт последний снимок опроса, а при его отсутствии — опрашивает немедленно.
func (s *Server) homeRaw(ctx context.Context) (json.RawMessage, time.Time, error) {
	if s.Snapshots == nil {
		raw, err := s.Home.UserInfo(ctx)
		return raw, time.Now(), err
	}
	snap, has := s.Snapshots.Latest()
	if !has {
		var err error
		if snap, err = s.Snapshots.Refresh(ctx); err != nil {
			return nil, time.Time{}, err
		}
	}
	return snap.Raw, snap.At, nil
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	raw, at, err := s.homeRaw(r.Context())
	if err != nil {
		s.writeError(w, err)
		return
	}
	w.Header().Set("X-Snapshot-Age", strconv.Itoa(int(time.Since(at).Seconds())))
	writeRaw(w, raw)
}

func (s *Server) deviceActions(w http.ResponseWriter, r *http.Request) {
	var req yandex.ActionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Devices) == 0 {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request",
			Message: "ожидается {\"devices\":[{\"id\":…,\"actions\":[…]}]}"})
		return
	}
	raw, err := s.Home.DeviceActions(r.Context(), req)
	if err != nil {
		s.writeError(w, err)
		return
	}
	if s.Snapshots != nil {
		s.Snapshots.Kick()
	}
	writeRaw(w, raw)
}

func (s *Server) runScenario(w http.ResponseWriter, r *http.Request) {
	raw, err := s.Home.RunScenario(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	if s.Snapshots != nil {
		s.Snapshots.Kick()
	}
	writeRaw(w, raw)
}
