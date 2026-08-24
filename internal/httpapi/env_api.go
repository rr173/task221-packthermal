package httpapi

import (
	"net/http"

	"task221-packthermal/internal/model"
)

type addEnvReq struct {
	Name    string            `json:"name"`
	Samples []model.EnvSample `json:"samples"`
}

// handleAddEnv POST /api/trials/{id}/env
func (s *Server) handleAddEnv(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	var req addEnvReq
	if err := decodeBody(w, r, &req); err != nil {
		writeErr(w, model.Wrap(model.ErrInvalidInput, "decode body: "+err.Error()))
		return
	}
	created, err := s.app.AddEnv(r.Context(), id, req.Name, req.Samples)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleListEnvs GET /api/trials/{id}/env
func (s *Server) handleListEnvs(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	envs, err := s.app.ListEnvs(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"env_profiles": envs, "count": len(envs)})
}
