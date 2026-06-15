package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
)

// ListResourcePacks GET /api/resource-packs
func (h *Handler) ListResourcePacks(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}

	resp, err := h.Lms.ListResourcePacks(r.Context(), &lmspb.ListResourcePacksCandidateRequest{
		CandidateId: candidateID,
		PageSize:    parseUint32Query(r, "page_size"),
		PageToken:   r.URL.Query().Get("page_token"),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListResourcePackFiles GET /api/resource-packs/{pack_id}/files
func (h *Handler) ListResourcePackFiles(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}
	packID := strings.TrimSpace(chi.URLParam(r, "pack_id"))
	if !requireRequestField(w, packID, "pack_id") {
		return
	}

	resp, err := h.Lms.ListResourcePackFiles(r.Context(), &lmspb.ListResourcePackFilesCandidateRequest{
		CandidateId: candidateID,
		PackId:      packID,
		PageSize:    parseUint32Query(r, "page_size"),
		PageToken:   r.URL.Query().Get("page_token"),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetResourcePackFileViewURL GET /api/resource-pack-files/{file_id}/view-url
func (h *Handler) GetResourcePackFileViewURL(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}
	fileID := strings.TrimSpace(chi.URLParam(r, "file_id"))
	if !requireRequestField(w, fileID, "file_id") {
		return
	}

	resp, err := h.Lms.GetResourcePackFileViewURL(r.Context(), &lmspb.GetResourcePackFileViewURLRequest{
		CandidateId: candidateID,
		FileId:      fileID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func parseUint32Query(r *http.Request, key string) uint32 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(value)
}
