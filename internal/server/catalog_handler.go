package server

import (
	"net/http"
	"strconv"
	"time"

	"smarthome/internal/catalog"
)

func (s *Server) catalog(w http.ResponseWriter, r *http.Request) {
	raw, at, err := s.homeRaw(r.Context())
	if err != nil {
		s.writeError(w, err)
		return
	}
	cat, err := catalog.Build(raw)
	if err != nil {
		s.writeError(w, err)
		return
	}
	w.Header().Set("X-Snapshot-Age", strconv.Itoa(int(time.Since(at).Seconds())))
	writeJSON(w, http.StatusOK, cat)
}
