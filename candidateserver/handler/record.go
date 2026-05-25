package handler

import (
	"net/http"
)

// TODO: 档案模块需要新建 record 微服务，包含以下 gRPC 接口:
//   - ListRecords(candidate_id)  → 档案列表
//   - CreateRecord(record)       → 上传档案
//   - ReviewRecord(record_id)    → 审核档案

// ListRecords  GET /api/records  档案列表
func (h *Handler) ListRecords(w http.ResponseWriter, r *http.Request) {
	// TODO: 需要新建 record 微服务
	WriteJSON(w, http.StatusOK, nil)
}

// CreateRecord  POST /api/records  上传档案
func (h *Handler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	// TODO: 需要新建 record 微服务
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
