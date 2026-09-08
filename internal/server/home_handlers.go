package server

import (
	"encoding/json"
	"net/http"

	"smarthome/internal/yandex"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	raw, err := s.Home.UserInfo(r.Context())
	if err != nil {
		s.writeError(w, err)
		return
	}
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
	writeRaw(w, raw)
}

func (s *Server) runScenario(w http.ResponseWriter, r *http.Request) {
	raw, err := s.Home.RunScenario(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeRaw(w, raw)
}
