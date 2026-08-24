package httpapi

import (
	"net/http"

	"task221-packthermal/internal/model"
)

type addSensorReq struct {
	Code     string `json:"code"`
	Position string `json:"position"`
	Scale    string `json:"scale"`
}

// handleAddSensor POST /api/trials/{id}/sensors
func (s *Server) handleAddSensor(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	var req addSensorReq
	if err := decodeBody(w, r, &req); err != nil {
		writeErr(w, model.Wrap(model.ErrInvalidInput, "decode body: "+err.Error()))
		return
	}
	created, err := s.app.AddSensor(r.Context(), id, req.Code, req.Position, req.Scale)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleListSensors GET /api/trials/{id}/sensors
func (s *Server) handleListSensors(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	sensors, err := s.app.ListSensors(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sensors": sensors, "count": len(sensors)})
}
