package httpapi

import (
	"context"
	"net/http"

	"task221-packthermal/internal/model"
)

type createTrialReq struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

// handleCreateTrial POST /api/trials
func (s *Server) handleCreateTrial(w http.ResponseWriter, r *http.Request) {
	var req createTrialReq
	if err := decodeBody(w, r, &req); err != nil {
		writeErr(w, model.Wrap(model.ErrInvalidInput, "decode body: "+err.Error()))
		return
	}
	t, err := s.app.CreateTrial(context.Background(), req.Code, req.Title)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// handleListTrials GET /api/trials
func (s *Server) handleListTrials(w http.ResponseWriter, r *http.Request) {
	trials, err := s.app.ListTrials(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"trials": trials, "count": len(trials)})
}

// handleGetTrial GET /api/trials/{id}
func (s *Server) handleGetTrial(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	t, err := s.app.GetTrial(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// handleStartCollecting POST /api/trials/{id}/start
func (s *Server) handleStartCollecting(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	t, err := s.app.StartCollecting(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// handleFreezeTrial POST /api/trials/{id}/freeze
func (s *Server) handleFreezeTrial(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	t, err := s.app.FreezeTrial(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// handleConfirmTrial POST /api/trials/{id}/confirm
func (s *Server) handleConfirmTrial(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	t, err := s.app.ConfirmTrial(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// handleArchiveTrial POST /api/trials/{id}/archive
func (s *Server) handleArchiveTrial(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	t, err := s.app.ArchiveTrial(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
