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
	// 使用请求上下文：客户端取消请求时，发布操作须随之下发取消，
	// 避免脱离请求生命周期继续生成新的快照版本。
	snap, err := s.app.PublishSnapshot(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
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
