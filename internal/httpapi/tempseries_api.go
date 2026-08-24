package httpapi

import (
	"net/http"

	"task221-packthermal/internal/model"
)

type importSeriesReq struct {
	SensorID int64              `json:"sensor_id"`
	Scale    string             `json:"scale"`
	Samples  []model.TempSample `json:"samples"`
}

// handleImportSeries POST /api/trials/{id}/tempseries
func (s *Server) handleImportSeries(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	var req importSeriesReq
	if err := decodeBody(w, r, &req); err != nil {
		writeErr(w, model.Wrap(model.ErrInvalidInput, "decode body: "+err.Error()))
		return
	}
	created, err := s.app.ImportSeries(r.Context(), id, req.SensorID, req.Scale, req.Samples)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleListSeries GET /api/trials/{id}/tempseries
func (s *Server) handleListSeries(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	series, err := s.app.ListSeries(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": series, "count": len(series)})
}

// handleMarkSeriesContact PUT /api/series/{id}/contact
func (s *Server) handleMarkSeriesContact(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	created, err := s.app.MarkSeriesContact(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, created)
}

// handleMarkSeriesValid PUT /api/series/{id}/valid
func (s *Server) handleMarkSeriesValid(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	created, err := s.app.MarkSeriesValid(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, created)
}
