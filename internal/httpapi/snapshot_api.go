package httpapi

import (
	"net/http"
)

// handlePublishSnapshot POST /api/trials/{id}/snapshots
func (s *Server) handlePublishSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	snap, err := s.app.PublishSnapshot(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	// 最新发布版本必须保持 published 状态返回，不得误报为已替代。
	writeJSON(w, http.StatusCreated, snap)
}

// handleListSnapshots GET /api/trials/{id}/snapshots
func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	snaps, err := s.app.ListSnapshots(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshots": snaps, "count": len(snaps)})
}

// handleGetSnapshot GET /api/snapshots/{id}
func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeErr(w, err)
		return
	}
	snap, err := s.app.GetSnapshot(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}
