package httpapi

import (
	"net/http"
)

// handleHealthz GET /healthz
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "task221-packthermal"})
}

// handleSelfCheck GET /api/selfcheck
func (s *Server) handleSelfCheck(w http.ResponseWriter, r *http.Request) {
	report, err := s.app.SelfCheck(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	status := http.StatusOK
	if report["integrity_check"] != 1 {
		status = http.StatusInternalServerError
	}
	writeJSON(w, status, report)
}

// handleStats GET /api/stats
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.app.Stats(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// handleAuditEvents GET /api/audit/events?limit=N
func (s *Server) handleAuditEvents(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 0)
	events, err := s.app.ListAuditAll(r.Context(), int(limit))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events, "count": len(events)})
}

// handleTrialAudit GET /api/audit/trials/{id}
func (s *Server) handleTrialAudit(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	events, err := s.app.ListAudit(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events, "count": len(events)})
}
