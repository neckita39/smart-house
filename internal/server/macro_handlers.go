package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"smarthome/internal/macros"
)

func (s *Server) listMacros(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Macros.List())
}

func (s *Server) createMacro(w http.ResponseWriter, r *http.Request) {
	var m macros.Macro
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: "некорректный JSON макроса"})
		return
	}
	created, err := s.Macros.Create(m)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateMacro(w http.ResponseWriter, r *http.Request) {
	var m macros.Macro
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: "некорректный JSON макроса"})
		return
	}
	m.ID = r.PathValue("id")
	if err := s.Macros.Update(m); err != nil {
		s.writeMacroError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) deleteMacro(w http.ResponseWriter, r *http.Request) {
	if err := s.Macros.Delete(r.PathValue("id")); err != nil {
		s.writeMacroError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) runMacro(w http.ResponseWriter, r *http.Request) {
	m, ok := s.Macros.Get(r.PathValue("id"))
	if !ok {
		s.writeMacroError(w, macros.ErrNotFound)
		return
	}
	raw, err := s.Home.DeviceActions(r.Context(), m.ToActionsRequest())
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeRaw(w, raw)
}

func (s *Server) writeMacroError(w http.ResponseWriter, err error) {
	if errors.Is(err, macros.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorBody{Error: "not_found", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
}
