package handler

import (
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
	resp, err := h.Lms.ListResourcePacksAdmin(r.Context(), &lmspb.ListResourcePacksRequest{
		PageSize:  parseUint32Query(r, "page_size"),
		PageToken: r.URL.Query().Get("page_token"),
		Status:    r.URL.Query().Get("status"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsResourcePack POST /api/lms/resource-packs
func (h *Handler) CreateLmsResourcePack(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateResourcePackRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.PackId) == "" {
		req.PackId = newLmsID()
	}
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

// ListLmsResourcePackFiles GET /api/lms/resource-packs/{pack_id}/files
func (h *Handler) ListLmsResourcePackFiles(w http.ResponseWriter, r *http.Request) {
	packID, ok := requiredURLParam(w, r, "pack_id")
	if !ok {
		return
	}
	resp, err := h.Lms.ListResourcePackFilesAdmin(r.Context(), &lmspb.ListResourcePackFilesRequest{
		PackId:    packID,
		PageSize:  parseUint32Query(r, "page_size"),
		PageToken: r.URL.Query().Get("page_token"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
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
	if strings.TrimSpace(req.FileId) == "" {
		req.FileId = newLmsID()
	}
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
		resp, err := h.Lms.ListResourcePackFilesAdmin(r.Context(), &lmspb.ListResourcePackFilesRequest{
			PackId:    packID,
			PageSize:  parseUint32Query(r, "page_size"),
			PageToken: r.URL.Query().Get("page_token"),
		})
		if err != nil {
			writeLmsError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, resp)
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
				PageSize:  resourcePackFilePackPageSize,
				PageToken: state.PackNextToken,
			})
			if err != nil {
				writeLmsError(w, err)
				return
			}
			state.PackNextToken = packsResp.GetNextPageToken()
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
			PackId:    packID,
			PageSize:  remaining,
			PageToken: state.FileToken,
		})
		if err != nil {
			writeLmsError(w, err)
			return
		}
		if filesResp != nil {
			allFiles = append(allFiles, filesResp.GetFiles()...)
			state.FileToken = filesResp.GetNextPageToken()
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
