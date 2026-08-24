// Package httpapi 提供 HTTP 接口层，路由统一以 /api 前缀。
package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"task221-packthermal/internal/model"
	"task221-packthermal/internal/service"
)

// Server 封装 HTTP 服务。
type Server struct {
	app *service.Service
	mux *http.ServeMux
}

// New 构造 HTTP 服务并注册全部路由。
func New(app *service.Service) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler 返回 http.Handler。
func (s *Server) Handler() http.Handler {
	return logMiddleware(s.mux)
}

// routes 注册路由（统一 /api 前缀）。
func (s *Server) routes() {
	// 元信息
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /api/selfcheck", s.handleSelfCheck)
	s.mux.HandleFunc("GET /api/stats", s.handleStats)
	s.mux.HandleFunc("GET /api/audit/events", s.handleAuditEvents)
	s.mux.HandleFunc("GET /api/audit/trials/{id}", s.handleTrialAudit)

	// 验证试验
	s.mux.HandleFunc("POST /api/trials", s.handleCreateTrial)
	s.mux.HandleFunc("GET /api/trials", s.handleListTrials)
	s.mux.HandleFunc("GET /api/trials/{id}", s.handleGetTrial)
	s.mux.HandleFunc("POST /api/trials/{id}/start", s.handleStartCollecting)
	s.mux.HandleFunc("POST /api/trials/{id}/freeze", s.handleFreezeTrial)
	s.mux.HandleFunc("POST /api/trials/{id}/confirm", s.handleConfirmTrial)
	s.mux.HandleFunc("POST /api/trials/{id}/archive", s.handleArchiveTrial)

	// 层序
	s.mux.HandleFunc("POST /api/trials/{id}/layers", s.handleAddLayer)
	s.mux.HandleFunc("GET /api/trials/{id}/layers", s.handleListLayers)

	// 环境曲线
	s.mux.HandleFunc("POST /api/trials/{id}/env", s.handleAddEnv)
	s.mux.HandleFunc("GET /api/trials/{id}/env", s.handleListEnvs)

	// 传感器
	s.mux.HandleFunc("POST /api/trials/{id}/sensors", s.handleAddSensor)
	s.mux.HandleFunc("GET /api/trials/{id}/sensors", s.handleListSensors)

	// 温度时序
	s.mux.HandleFunc("POST /api/trials/{id}/tempseries", s.handleImportSeries)
	s.mux.HandleFunc("GET /api/trials/{id}/tempseries", s.handleListSeries)
	s.mux.HandleFunc("PUT /api/series/{id}/contact", s.handleMarkSeriesContact)
	s.mux.HandleFunc("PUT /api/series/{id}/valid", s.handleMarkSeriesValid)

	// 热模型
	s.mux.HandleFunc("POST /api/trials/{id}/models", s.handleCreateModel)
	s.mux.HandleFunc("GET /api/trials/{id}/models", s.handleListModels)
	s.mux.HandleFunc("POST /api/models/{id}/confirm", s.handleConfirmModel)

	// 反演
	s.mux.HandleFunc("POST /api/trials/{id}/inversions", s.handleRunInversion)
	s.mux.HandleFunc("GET /api/trials/{id}/inversions", s.handleListInversions)
	s.mux.HandleFunc("GET /api/inversions/{id}", s.handleGetInversion)

	// 快照
	s.mux.HandleFunc("POST /api/trials/{id}/snapshots", s.handlePublishSnapshot)
	s.mux.HandleFunc("GET /api/trials/{id}/snapshots", s.handleListSnapshots)
	s.mux.HandleFunc("GET /api/snapshots/{id}", s.handleGetSnapshot)
}

// ---------------------------------------------------------------------------
// 工具
// ---------------------------------------------------------------------------

// writeJSON 写出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

// writeErr 依据领域错误映射 HTTP 状态码。
func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, model.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrConflict),
		errors.Is(err, model.ErrDuplicate),
		errors.Is(err, model.ErrImmutable):
		status = http.StatusConflict
	case errors.Is(err, model.ErrInvalidInput):
		status = http.StatusBadRequest
	default:
		var se *model.StateError
		if errors.As(err, &se) {
			status = http.StatusConflict
		}
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// parseID 解析路径参数 ID。
func parseID(r *http.Request, key string) (int64, error) {
	raw := r.PathValue(key)
	if raw == "" {
		return 0, model.ErrInvalidInput
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, model.ErrInvalidInput
	}
	return id, nil
}

// decodeBody 解析 JSON 请求体（限制大小，禁止未知字段）。
func decodeBody(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// queryInt 读取查询参数整数。
func queryInt(r *http.Request, key string, def int64) int64 {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def
	}
	if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return v
	}
	return def
}

// logMiddleware 请求日志中间件。
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Microsecond))
	})
}

// actor 提取请求方标识（缺省 system）。
func actor(r *http.Request) string {
	a := strings.TrimSpace(r.Header.Get("X-Actor"))
	if a == "" {
		return "system"
	}
	return a
}
