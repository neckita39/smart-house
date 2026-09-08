package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"authorized": s.Auth.Authorized(),
		"login_url":  s.Auth.LoginURL(),
	})
}

func (s *Server) authCode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Code) == "" {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: "введите код подтверждения"})
		return
	}
	if err := s.Auth.Login(r.Context(), strings.TrimSpace(body.Code)); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "login_failed", Message: "не удалось войти: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"authorized": true})
}
