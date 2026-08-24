package httpapi

import (
	"net/http"

	"task221-packthermal/internal/model"
)

type addLayerReq struct {
	Seq          int     `json:"seq"`
	Material     string  `json:"material"`
	ThicknessMM  float64 `json:"thickness_mm"`
	Conductivity float64 `json:"conductivity"`
	Density      float64 `json:"density"`
	SpecificHeat float64 `json:"specific_heat"`
	AreaM2       float64 `json:"area_m2"`
}

// handleAddLayer POST /api/trials/{id}/layers
func (s *Server) handleAddLayer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	var req addLayerReq
	if err := decodeBody(w, r, &req); err != nil {
		writeErr(w, model.Wrap(model.ErrInvalidInput, "decode body: "+err.Error()))
		return
	}
	layer := &model.Layer{
		Seq:          req.Seq,
		Material:     req.Material,
		ThicknessMM:  req.ThicknessMM,
		Conductivity: req.Conductivity,
		Density:      req.Density,
		SpecificHeat: req.SpecificHeat,
		AreaM2:       req.AreaM2,
	}
	created, err := s.app.AddLayer(r.Context(), id, layer)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleListLayers GET /api/trials/{id}/layers
func (s *Server) handleListLayers(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	layers, err := s.app.ListLayers(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"layers": layers, "count": len(layers)})
}
