package server

import (
	"net/http"

	"smarthome/internal/catalog"
)

func (s *Server) catalog(w http.ResponseWriter, r *http.Request) {
	raw, err := s.Home.UserInfo(r.Context())
	if err != nil {
		s.writeError(w, err)
		return
	}
	cat, err := catalog.Build(raw)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cat)
}
