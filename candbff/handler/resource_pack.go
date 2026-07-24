package handler

import (
	"net/http"
	"strconv"
	"strings"
	"sync"

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
		Filters: &lmspb.ResourcePackCandidateFilters{
			CandidateUlid: candidateID,
		},
		PageSize: parseUint32Query(r, "page_size"),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
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
		Filters: &lmspb.ResourcePackFileCandidateFilters{
			CandidateUlid: candidateID,
			PackId:        packID,
		},
		PageSize: parseUint32Query(r, "page_size"),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	type extFile struct {
		*lmspb.ResourcePackFile
		ThumbnailUrl string `json:"thumbnail_url,omitempty"`
	}

	extFiles := make([]extFile, len(resp.GetFiles()))
	var wg sync.WaitGroup
	for i, file := range resp.GetFiles() {
		extFiles[i] = extFile{ResourcePackFile: file}
		if objectKey := strings.TrimSpace(file.GetThumbnailObjectKey()); objectKey != "" {
			wg.Add(1)
			go func(index int, key string) {
				defer wg.Done()
				viewResp, err := h.Lms.CreateViewURL(r.Context(), &lmspb.CreateViewURLCandidateRequest{
					CandidateUlid: candidateID,
					ObjectKey:     key,
				})
				if err == nil {
					extFiles[index].ThumbnailUrl = viewResp.GetViewUrl()
				}
			}(i, objectKey)
		}
	}
	wg.Wait()

	out := struct {
		Files         []extFile `json:"files,omitempty"`
		NextPageToken string    `json:"next_page_token,omitempty"`
		NextCursor    string    `json:"next_cursor,omitempty"`
		HasMore       bool      `json:"has_more"`
	}{
		Files:         extFiles,
		NextPageToken: resp.GetNextCursor(),
		NextCursor:    resp.GetNextCursor(),
		HasMore:       resp.GetHasMore(),
	}

	WriteJSON(w, http.StatusOK, out)
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

	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL:       viewURL,
		ExpiresAt: viewResp.GetExpiresAt(),
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
	const pageSize uint32 = 100
	const maxPages = 1000
	packCursor := ""
	seenPackCursors := make(map[string]struct{})
	for packPage := 0; packPage < maxPages; packPage++ {
		packsResp, err := h.Lms.ListResourcePacks(r.Context(), &lmspb.ListResourcePacksCandidateRequest{
			Filters: &lmspb.ResourcePackCandidateFilters{
				CandidateUlid: candidateID,
			},
			Cursor:   packCursor,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, err
		}

		for _, pack := range packsResp.GetPacks() {
			fileCursor := ""
			seenFileCursors := make(map[string]struct{})
			filesComplete := false
			for filePage := 0; filePage < maxPages; filePage++ {
				listResp, err := h.Lms.ListResourcePackFiles(r.Context(), &lmspb.ListResourcePackFilesCandidateRequest{
					Filters: &lmspb.ResourcePackFileCandidateFilters{
						CandidateUlid: candidateID,
						PackId:        pack.GetPackId(),
					},
					PageSize: pageSize,
					Cursor:   fileCursor,
				})
				if err != nil {
					return nil, err
				}
				for _, file := range listResp.GetFiles() {
					if file.GetFileId() == fileID {
						return file, nil
					}
				}
				if !listResp.GetHasMore() {
					filesComplete = true
					break
				}
				nextCursor := strings.TrimSpace(listResp.GetNextCursor())
				if nextCursor == "" || nextCursor == fileCursor {
					return nil, status.Error(codes.Internal, "resource pack file cursor did not advance")
				}
				if _, ok := seenFileCursors[nextCursor]; ok {
					return nil, status.Error(codes.Internal, "resource pack file cursor loop detected")
				}
				seenFileCursors[nextCursor] = struct{}{}
				fileCursor = nextCursor
			}
			if !filesComplete {
				return nil, status.Error(codes.Internal, "resource pack file pagination exceeded max pages")
			}
		}

		if !packsResp.GetHasMore() {
			return nil, status.Error(codes.NotFound, "resource pack file not found")
		}
		nextCursor := strings.TrimSpace(packsResp.GetNextCursor())
		if nextCursor == "" || nextCursor == packCursor {
			return nil, status.Error(codes.Internal, "resource pack cursor did not advance")
		}
		if _, ok := seenPackCursors[nextCursor]; ok {
			return nil, status.Error(codes.Internal, "resource pack cursor loop detected")
		}
		seenPackCursors[nextCursor] = struct{}{}
		packCursor = nextCursor
	}

	return nil, status.Error(codes.Internal, "resource pack pagination exceeded max pages")
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
