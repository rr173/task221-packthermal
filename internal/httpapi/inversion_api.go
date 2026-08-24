package httpapi

import (
	"net/http"
)

type runInversionReq struct {
	ModelID int64 `json:"model_id"`
}

// handleRunInversion POST /api/trials/{id}/inversions
func (s *Server) handleRunInversion(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	var req runInversionReq
	if err := decodeBody(w, r, &req); err != nil {
		writeErr(w, err)
		return
	}
	created, err := s.app.RunInversion(r.Context(), id, req.ModelID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleListInversions GET /api/trials/{id}/inversions
func (s *Server) handleListInversions(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	invs, err := s.app.ListInversions(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"inversions": invs, "count": len(invs)})
}

// handleGetInversion GET /api/inversions/{id}
func (s *Server) handleGetInversion(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	inv, err := s.app.GetInversion(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}
