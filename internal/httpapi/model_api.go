package httpapi

import (
	"net/http"

	"task221-packthermal/internal/model"
)

type createModelReq struct {
	Name string `json:"name"`
}

// handleCreateModel POST /api/trials/{id}/models
func (s *Server) handleCreateModel(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	var req createModelReq
	if err := decodeBody(w, r, &req); err != nil {
		writeErr(w, model.Wrap(model.ErrInvalidInput, "decode body: "+err.Error()))
		return
	}
	created, err := s.app.CreateModel(r.Context(), id, req.Name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleListModels GET /api/trials/{id}/models
func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	models, err := s.app.ListModels(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": models, "count": len(models)})
}

// handleConfirmModel POST /api/models/{id}/confirm
func (s *Server) handleConfirmModel(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	m, err := s.app.ConfirmModel(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}
