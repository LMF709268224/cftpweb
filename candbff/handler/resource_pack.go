package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
)

// ListResourcePacks GET /api/resource-packs
func (h *Handler) ListResourcePacks(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}

	resp, err := h.Lms.ListResourcePacks(r.Context(), &lmspb.ListResourcePacksCandidateRequest{
		CandidateUlid: candidateID,
		PageSize:      parseUint32Query(r, "page_size"),
		PageToken:     r.URL.Query().Get("page_token"),
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
		CandidateUlid: candidateID,
		PackId:        packID,
		PageSize:      parseUint32Query(r, "page_size"),
		PageToken:     r.URL.Query().Get("page_token"),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	b, _ := json.MarshalIndent(resp.GetFiles(), "", "  ")
	slog.Info("ListResourcePackFiles returned", "pack_id", packID, "files", string(b))

	WriteJSON(w, http.StatusOK, resp)
}

// GetResourcePackFileThumbnailURL GET /api/resource-pack-files/{file_id}/thumbnail-url
func (h *Handler) GetResourcePackFileThumbnailURL(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}
	fileID := strings.TrimSpace(chi.URLParam(r, "file_id"))
	if !requireRequestField(w, fileID, "file_id") {
		return
	}

	file, err := h.findResourcePackFileForCandidate(r, candidateID, fileID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	objectKey := strings.TrimSpace(file.GetThumbnailObjectKey())
	if objectKey == "" {
		WriteJSON(w, http.StatusOK, GetAccessURLRsp{})
		return
	}

	resp, err := h.Lms.CreateViewURL(r.Context(), &lmspb.CreateViewURLCandidateRequest{
		CandidateUlid: candidateID,
		ObjectKey:     objectKey,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL:       resp.GetViewUrl(),
		ExpiresAt: resp.GetExpiresAt(),
	})
}

// GetResourcePackFilePreviewURL GET /api/resource-pack-files/{file_id}/preview-url
func (h *Handler) GetResourcePackFilePreviewURL(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}
	fileID := strings.TrimSpace(chi.URLParam(r, "file_id"))
	if !requireRequestField(w, fileID, "file_id") {
		return
	}

	viewResp, err := h.Lms.GetResourcePackFileViewURL(r.Context(), &lmspb.GetResourcePackFileViewURLRequest{
		CandidateUlid: candidateID,
		FileId:        fileID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	viewURL := strings.TrimSpace(viewResp.GetViewUrl())
	if viewURL == "" {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "empty view url")
		return
	}

	file, err := h.findResourcePackFileForCandidate(r, candidateID, fileID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if file.GetFileType() != lmspb.ResourcePackFileType_RESOURCE_PACK_FILE_TYPE_PDF {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "resource pack file is not a pdf")
		return
	}

	title := firstNonEmpty(strings.TrimSpace(file.GetTitle()), strings.TrimSpace(file.GetFileName()), fileID)
	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL:       viewURL,
		ExpiresAt: viewResp.GetExpiresAt(),
		Title:     title,
	})
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
		CandidateUlid: candidateID,
		FileId:        fileID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) findResourcePackFileForCandidate(r *http.Request, candidateID string, fileID string) (*lmspb.ResourcePackFile, error) {
	filesResp, err := h.Lms.ListResourcePacks(r.Context(), &lmspb.ListResourcePacksCandidateRequest{
		CandidateUlid: candidateID,
		PageSize:      500,
	})
	if err != nil {
		return nil, err
	}

	for _, pack := range filesResp.GetPacks() {
		pageToken := ""
		for {
			listResp, err := h.Lms.ListResourcePackFiles(r.Context(), &lmspb.ListResourcePackFilesCandidateRequest{
				CandidateUlid: candidateID,
				PackId:        pack.GetPackId(),
				PageSize:      500,
				PageToken:     pageToken,
			})
			if err != nil {
				return nil, err
			}
			for _, file := range listResp.GetFiles() {
				if file.GetFileId() == fileID {
					return file, nil
				}
			}
			pageToken = listResp.GetNextPageToken()
			if strings.TrimSpace(pageToken) == "" {
				break
			}
		}
	}

	return nil, status.Error(codes.NotFound, "resource pack file not found")
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

