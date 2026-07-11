package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
)

const resourcePackFilePackPageSize = 20

type allResourcePackFilesPageToken struct {
	PackIDs       []string `json:"pack_ids"`
	PackNextToken string   `json:"pack_next_token"`
	FileToken     string   `json:"file_token"`
	PackListDone  bool     `json:"pack_list_done"`
}

// ListLmsResourcePacks GET /api/lms/resource-packs
func (h *Handler) ListLmsResourcePacks(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	filters := &lmspb.ResourcePackFilters{
		Status: r.URL.Query().Get("status"),
	}

	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Lms.GetResourcePackCountAdmin(ctx, &lmspb.GetResourcePackCountRequest{
			Filters:   filters,
			Limit:     limit,
			Cursor:    cursor,
			SortOrder: lmspb.SortOrder(page.Sort),
		})
		if err != nil {
			return 0, "", err
		}
		return resp.GetCount(), resp.GetNextCursor(), nil
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}

	resp, err := h.Lms.ListResourcePacksAdmin(r.Context(), &lmspb.ListResourcePacksRequest{
		Filters:   filters,
		PageSize:  page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:    firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"total":       total.Total,
		"exact":       total.Exact,
		"has_more":    resp.GetHasMore(),
		"next_cursor": resp.GetNextCursor(),
		"prev_cursor": resp.GetPrevCursor(),
		"packs":       resp.GetPacks(),
	})
}

// CreateLmsResourcePack POST /api/lms/resource-packs
func (h *Handler) CreateLmsResourcePack(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateResourcePackRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PackId = newLmsID()
	if !requireRequestField(w, req.Title, "title") {
		return
	}

	resp, err := h.Lms.CreateResourcePackAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsResourcePack GET /api/lms/resource-packs/{pack_id}
func (h *Handler) GetLmsResourcePack(w http.ResponseWriter, r *http.Request) {
	packID, ok := requiredURLParam(w, r, "pack_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetResourcePackAdmin(r.Context(), &lmspb.GetResourcePackRequest{PackId: packID})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsResourcePack PUT /api/lms/resource-packs/{pack_id}
func (h *Handler) UpdateLmsResourcePack(w http.ResponseWriter, r *http.Request) {
	packID, ok := requiredURLParam(w, r, "pack_id")
	if !ok {
		return
	}
	var req lmspb.UpdateResourcePackRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PackId = packID
	if !requireRequestField(w, req.Title, "title") || !requirePositiveVersion(w, req.Version) {
		return
	}

	resp, err := h.Lms.UpdateResourcePackAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// PublishLmsResourcePack POST /api/lms/resource-packs/{pack_id}/publish
func (h *Handler) PublishLmsResourcePack(w http.ResponseWriter, r *http.Request) {
	packID, ok := requiredURLParam(w, r, "pack_id")
	if !ok {
		return
	}
	var body versionOnlyReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if !requirePositiveVersion(w, body.Version) {
		return
	}

	resp, err := h.Lms.PublishResourcePackAdmin(r.Context(), &lmspb.PublishResourcePackRequest{
		PackId:  packID,
		Version: body.Version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// RevertLmsResourcePackToDraft POST /api/lms/resource-packs/{pack_id}/revert-to-draft
func (h *Handler) RevertLmsResourcePackToDraft(w http.ResponseWriter, r *http.Request) {
	packID, ok := requiredURLParam(w, r, "pack_id")
	if !ok {
		return
	}
	var body versionOnlyReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if !requirePositiveVersion(w, body.Version) {
		return
	}

	resp, err := h.Lms.RevertResourcePackToDraftAdmin(r.Context(), &lmspb.RevertResourcePackToDraftRequest{
		PackId:  packID,
		Version: body.Version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DuplicateLmsResourcePack POST /api/lms/resource-packs/{pack_id}/duplicate
func (h *Handler) DuplicateLmsResourcePack(w http.ResponseWriter, r *http.Request) {
	packID, ok := requiredURLParam(w, r, "pack_id")
	if !ok {
		return
	}
	var req lmspb.DuplicateResourcePackRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if !requireRequestField(w, req.Title, "title") {
		return
	}
	req.PackId = newLmsID()
	req.FromPackId = packID

	resp, err := h.Lms.DuplicateResourcePackAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsResourcePack DELETE /api/lms/resource-packs/{pack_id}
func (h *Handler) DeleteLmsResourcePack(w http.ResponseWriter, r *http.Request) {
	packID, ok := requiredURLParam(w, r, "pack_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}
	if !requirePositiveVersion(w, version) {
		return
	}

	resp, err := h.Lms.DeleteResourcePackAdmin(r.Context(), &lmspb.DeleteResourcePackRequest{
		PackId:  packID,
		Version: version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CleanUpDeprecatedResourcePackAssets POST /api/lms/resource-packs/{pack_id}/cleanup-assets
func (h *Handler) CleanUpDeprecatedResourcePackAssets(w http.ResponseWriter, r *http.Request) {
	packID, ok := requiredURLParam(w, r, "pack_id")
	if !ok {
		return
	}

	resp, err := h.Lms.CleanUpDeprecatedResourcePackAssetsAdmin(r.Context(), &lmspb.CleanUpDeprecatedResourcePackAssetsRequest{
		PackId: packID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsResourcePackFiles GET /api/lms/resource-packs/{pack_id}/files
func (h *Handler) ListLmsResourcePackFiles(w http.ResponseWriter, r *http.Request) {
	packID, ok := requiredURLParam(w, r, "pack_id")
	if !ok {
		return
	}
	page := parseCursorPage(r, 20)
	filters := &lmspb.ResourcePackFileFilters{
		PackId: packID,
	}

	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Lms.GetResourcePackFileCountAdmin(ctx, &lmspb.GetResourcePackFileCountRequest{
			Filters:   filters,
			Limit:     limit,
			Cursor:    cursor,
			SortOrder: lmspb.SortOrder(page.Sort),
		})
		if err != nil {
			return 0, "", err
		}
		return resp.GetCount(), resp.GetNextCursor(), nil
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}

	resp, err := h.Lms.ListResourcePackFilesAdmin(r.Context(), &lmspb.ListResourcePackFilesRequest{
		Filters:   filters,
		PageSize:  page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:    firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"total":       total.Total,
		"exact":       total.Exact,
		"has_more":    resp.GetHasMore(),
		"next_cursor": resp.GetNextCursor(),
		"prev_cursor": resp.GetPrevCursor(),
		"files":       resp.GetFiles(),
	})
}

// CreateLmsResourcePackFile POST /api/lms/resource-packs/{pack_id}/files
func (h *Handler) CreateLmsResourcePackFile(w http.ResponseWriter, r *http.Request) {
	packID, ok := requiredURLParam(w, r, "pack_id")
	if !ok {
		return
	}
	var req lmspb.CreateResourcePackFileRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PackId = packID
	req.FileId = newLmsID()
	if !validateResourcePackFilePayload(w, req.Title, req.FileType) {
		return
	}

	resp, err := h.Lms.CreateResourcePackFileAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsResourcePackFile GET /api/lms/resource-pack-files/{file_id}
func (h *Handler) GetLmsResourcePackFile(w http.ResponseWriter, r *http.Request) {
	fileID, ok := requiredURLParam(w, r, "file_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetResourcePackFileAdmin(r.Context(), &lmspb.GetResourcePackFileRequest{FileId: fileID})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsResourcePackFile PUT /api/lms/resource-pack-files/{file_id}
func (h *Handler) UpdateLmsResourcePackFile(w http.ResponseWriter, r *http.Request) {
	fileID, ok := requiredURLParam(w, r, "file_id")
	if !ok {
		return
	}
	var req lmspb.UpdateResourcePackFileRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.FileId = fileID
	if !validateResourcePackFilePayload(w, req.Title, req.FileType) || !requirePositiveVersion(w, req.Version) {
		return
	}

	resp, err := h.Lms.UpdateResourcePackFileAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsResourcePackFile DELETE /api/lms/resource-pack-files/{file_id}
func (h *Handler) DeleteLmsResourcePackFile(w http.ResponseWriter, r *http.Request) {
	fileID, ok := requiredURLParam(w, r, "file_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}
	if !requirePositiveVersion(w, version) {
		return
	}

	resp, err := h.Lms.DeleteResourcePackFileAdmin(r.Context(), &lmspb.DeleteResourcePackFileRequest{
		FileId:  fileID,
		Version: version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func validateResourcePackFilePayload(w http.ResponseWriter, title string, fileType lmspb.ResourcePackFileType) bool {
	if !requireRequestField(w, title, "title") {
		return false
	}
	if fileType == lmspb.ResourcePackFileType_RESOURCE_PACK_FILE_TYPE_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "file_type is required")
		return false
	}
	return true
}

// ListAllLmsResourcePackFiles GET /api/lms/resource-pack-files
func (h *Handler) ListAllLmsResourcePackFiles(w http.ResponseWriter, r *http.Request) {
	packID := r.URL.Query().Get("pack_id")
	if packID != "" {
		page := parseCursorPage(r, 20)
		filters := &lmspb.ResourcePackFileFilters{
			PackId: packID,
		}
		
		total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
			resp, err := h.Lms.GetResourcePackFileCountAdmin(ctx, &lmspb.GetResourcePackFileCountRequest{
				Filters:   filters,
				Limit:     limit,
				Cursor:    cursor,
				SortOrder: lmspb.SortOrder(page.Sort),
			})
			if err != nil {
				return 0, "", err
			}
			return resp.GetCount(), resp.GetNextCursor(), nil
		})
		if err != nil {
			writeLmsError(w, err)
			return
		}

		resp, err := h.Lms.ListResourcePackFilesAdmin(r.Context(), &lmspb.ListResourcePackFilesRequest{
			Filters:   filters,
			PageSize:  page.PageSize,
			SortOrder: lmspb.SortOrder(page.Sort),
			Cursor:    firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
		})
		if err != nil {
			writeLmsError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, map[string]interface{}{
			"total":       total.Total,
			"exact":       total.Exact,
			"has_more":    resp.GetHasMore(),
			"next_cursor": resp.GetNextCursor(),
			"prev_cursor": resp.GetPrevCursor(),
			"files":       resp.GetFiles(),
		})
		return
	}

	pageSize := parseUint32Query(r, "page_size")
	if pageSize == 0 {
		pageSize = 10
	}

	state, ok := decodeAllResourcePackFilesPageToken(r.URL.Query().Get("page_token"))
	if !ok {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid page_token")
		return
	}
	var allFiles []*lmspb.ResourcePackFile
	for uint32(len(allFiles)) < pageSize {
		if len(state.PackIDs) == 0 {
			if state.PackListDone {
				break
			}
			packsResp, err := h.Lms.ListResourcePacksAdmin(r.Context(), &lmspb.ListResourcePacksRequest{
				PageSize: resourcePackFilePackPageSize,
				Cursor:   state.PackNextToken,
			})
			if err != nil {
				writeLmsError(w, err)
				return
			}
			state.PackNextToken = packsResp.GetNextCursor()
			state.PackListDone = state.PackNextToken == ""
			for _, pack := range packsResp.GetPacks() {
				if strings.TrimSpace(pack.GetPackId()) != "" {
					state.PackIDs = append(state.PackIDs, pack.GetPackId())
				}
			}
			if len(state.PackIDs) == 0 {
				break
			}
		}

		packID := state.PackIDs[0]
		remaining := pageSize - uint32(len(allFiles))
		filesResp, err := h.Lms.ListResourcePackFilesAdmin(r.Context(), &lmspb.ListResourcePackFilesRequest{
			Filters: &lmspb.ResourcePackFileFilters{
				PackId: packID,
			},
			PageSize: remaining,
			Cursor:   state.FileToken,
		})
		if err != nil {
			writeLmsError(w, err)
			return
		}
		if filesResp != nil {
			allFiles = append(allFiles, filesResp.GetFiles()...)
			state.FileToken = filesResp.GetNextCursor()
		}
		if state.FileToken != "" {
			break
		}
		state.PackIDs = state.PackIDs[1:]
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"files":           allFiles,
		"next_page_token": encodeAllResourcePackFilesPageToken(state),
	})
}

func decodeAllResourcePackFilesPageToken(raw string) (allResourcePackFilesPageToken, bool) {
	if strings.TrimSpace(raw) == "" {
		return allResourcePackFilesPageToken{}, true
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return allResourcePackFilesPageToken{}, false
	}
	var token allResourcePackFilesPageToken
	if err := json.Unmarshal(data, &token); err != nil {
		return allResourcePackFilesPageToken{}, false
	}
	return token, true
}

func encodeAllResourcePackFilesPageToken(token allResourcePackFilesPageToken) string {
	if token.FileToken == "" && len(token.PackIDs) == 0 && token.PackNextToken == "" {
		return ""
	}
	data, err := json.Marshal(token)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(data)
}
