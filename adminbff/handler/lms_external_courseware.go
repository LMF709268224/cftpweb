package handler

import (
	"net/http"
	"strings"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
)

func (h *Handler) ListExternalCoursewares(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 50)
	resp, err := h.Lms.ListExternalCoursewaresAdmin(r.Context(), &lmspb.ListExternalCoursewaresRequest{
		Filters: &lmspb.ExternalCoursewareFilters{
			Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")),
		},
		PageSize:  page.PageSize,
		Cursor:    firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
		SortOrder: lmspb.SortOrder(page.Sort),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateExternalCourseware(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateExternalCoursewareRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CoursewareUlid = newLmsID()
	req.Name = strings.TrimSpace(req.Name)
	req.BaseUrl = strings.TrimSpace(req.BaseUrl)
	if !requireRequestFields(w, req.Name, "name", req.BaseUrl, "base_url") {
		return
	}
	resp, err := h.Lms.CreateExternalCoursewareAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetExternalCourseware(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "courseware_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetExternalCoursewareDetailAdmin(r.Context(), &lmspb.GetExternalCoursewareDetailRequest{CoursewareUlid: id})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateExternalCourseware(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "courseware_id")
	if !ok {
		return
	}
	var req lmspb.UpdateExternalCoursewareRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CoursewareUlid = id
	req.Name = strings.TrimSpace(req.Name)
	req.BaseUrl = strings.TrimSpace(req.BaseUrl)
	if !requireRequestFields(w, req.Name, "name", req.BaseUrl, "base_url") || !requirePositiveVersion(w, req.Version) {
		return
	}
	resp, err := h.Lms.UpdateExternalCoursewareAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeleteExternalCourseware(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "courseware_id")
	if !ok {
		return
	}
	resp, err := h.Lms.DeleteExternalCoursewareAdmin(r.Context(), &lmspb.DeleteExternalCoursewareRequest{CoursewareUlid: id})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ImportExternalCoursewareTokens(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "courseware_id")
	if !ok {
		return
	}
	var body struct {
		TokenURLs []string `json:"token_urls"`
	}
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	tokenURLs := make([]string, 0, len(body.TokenURLs))
	for _, tokenURL := range body.TokenURLs {
		if value := strings.TrimSpace(tokenURL); value != "" {
			tokenURLs = append(tokenURLs, value)
		}
	}
	if len(tokenURLs) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "token_urls is required")
		return
	}
	if len(tokenURLs) > 1000 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "token_urls must contain at most 1000 items")
		return
	}
	resp, err := h.Lms.ImportCoursewareTokensAdmin(r.Context(), &lmspb.ImportCoursewareTokensRequest{
		CoursewareUlid: id,
		TokenUrls:      tokenURLs,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetExternalCoursewareTokenStats(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "courseware_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetCoursewareTokenStatsAdmin(r.Context(), &lmspb.GetCoursewareTokenStatsRequest{CoursewareUlid: id})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListExternalCoursewareTokens(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "courseware_id")
	if !ok {
		return
	}
	page := parseCursorPage(r, 50)
	resp, err := h.Lms.ListCoursewareTokensAdmin(r.Context(), &lmspb.ListCoursewareTokensRequest{
		Filters: &lmspb.CoursewareTokenFilters{
			CoursewareUlid:  id,
			CandidateUlid:   strings.TrimSpace(r.URL.Query().Get("candidate_ulid")),
			UnallocatedOnly: parseBoolQuery(r, "unallocated_only"),
		},
		PageSize:  page.PageSize,
		Cursor:    firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
		SortOrder: lmspb.SortOrder(page.Sort),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
