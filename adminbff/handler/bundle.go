package handler

import (
	"net/http"
	"strings"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

func (h *Handler) CreateBundle(w http.ResponseWriter, r *http.Request) {
	var req mallpb.CreateBundleDraftRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.BundleUlid = strings.TrimSpace(req.BundleUlid)
	req.BundleGpath = strings.TrimSpace(req.BundleGpath)
	req.Name = strings.TrimSpace(req.Name)
	if req.BundleUlid == "" {
		req.BundleUlid = ulid.Make().String()
	}
	if !requireRequestFields(w, req.BundleGpath, "bundle_gpath", req.Name, "name") {
		return
	}
	resp, err := h.Mall.CreateBundleDraft(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) UpdateBundleMeta(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BundleUlid         string  `json:"bundle_ulid"`
		TargetUlid         string  `json:"target_ulid"`
		Name               *string `json:"name"`
		NewName            *string `json:"new_name"`
		Description        *string `json:"description"`
		ThumbnailObjectKey *string `json:"thumbnail_object_key"`
		ThumbnailFileHash  *string `json:"thumbnail_file_hash"`
	}
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	targetULID := strings.TrimSpace(firstNonEmpty(body.TargetUlid, body.BundleUlid, chi.URLParam(r, "bundle_ulid")))
	if !requireRequestField(w, targetULID, "bundle_ulid") {
		return
	}
	newName := body.NewName
	if newName == nil {
		newName = body.Name
	}
	req := &mallpb.UpdateBundleMetadataRequest{
		TargetUlid:         targetULID,
		NewName:            newName,
		Description:        body.Description,
		ThumbnailObjectKey: body.ThumbnailObjectKey,
		ThumbnailFileHash:  body.ThumbnailFileHash,
	}
	resp, err := h.Mall.UpdateBundleMetadata(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateBundlePricing(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BundleUlid  string `json:"bundle_ulid"`
		BundleGpath string `json:"bundle_gpath"`
		PricingJson string `json:"pricing_json"`
		ItemsJson   string `json:"items_json"`
	}
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	bundleULID := strings.TrimSpace(firstNonEmpty(body.BundleUlid, chi.URLParam(r, "bundle_ulid")))
	if bundleULID == "" && strings.TrimSpace(body.BundleGpath) != "" {
		getResp, err := h.Mall.AdminGetBundle(r.Context(), &mallpb.AdminGetBundleRequest{
			Query: &mallpb.AdminGetBundleRequest_BundleGpath{BundleGpath: strings.TrimSpace(body.BundleGpath)},
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		bundleULID = getResp.GetBundle().GetBundleUlid()
	}
	if !requireRequestFields(w, bundleULID, "bundle_ulid", body.PricingJson, "pricing_json", body.ItemsJson, "items_json") {
		return
	}
	resp, err := h.Mall.UpdateBundleStructure(r.Context(), &mallpb.UpdateBundleStructureRequest{
		BundleUlid:  bundleULID,
		PricingJson: body.PricingJson,
		ItemsJson:   body.ItemsJson,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetBundle(w http.ResponseWriter, r *http.Request) {
	bundleULID := strings.TrimSpace(chi.URLParam(r, "bundle_ulid"))
	bundleGPath := strings.TrimSpace(r.URL.Query().Get("bundle_gpath"))
	if bundleULID == "" && bundleGPath == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "bundle_ulid or bundle_gpath is required")
		return
	}

	req := &mallpb.AdminGetBundleRequest{}
	if bundleULID != "" {
		req.Query = &mallpb.AdminGetBundleRequest_BundleUlid{BundleUlid: bundleULID}
	} else {
		req.Query = &mallpb.AdminGetBundleRequest_BundleGpath{BundleGpath: bundleGPath}
	}
	resp, err := h.Mall.AdminGetBundle(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) DuplicateBundle(w http.ResponseWriter, r *http.Request) {
	fromBundleULID := strings.TrimSpace(chi.URLParam(r, "bundle_ulid"))
	if !requireRequestField(w, fromBundleULID, "bundle_ulid") {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		getResp, err := h.Mall.AdminGetBundle(r.Context(), &mallpb.AdminGetBundleRequest{
			Query: &mallpb.AdminGetBundleRequest_BundleUlid{BundleUlid: fromBundleULID},
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		name = strings.TrimSpace(getResp.GetBundle().GetName())
		if name == "" {
			name = fromBundleULID
		}
		name += " (Copy)"
	}

	resp, err := h.Mall.DuplicateBundleDraft(r.Context(), &mallpb.DuplicateBundleDraftRequest{
		FromBundleUlid: fromBundleULID,
		BundleUlid:     ulid.Make().String(),
		Name:           name,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) ListBundles(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	filters := &mallpb.BundleAdminFilters{
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
	}
	if _, ok := r.URL.Query()["is_current_only"]; ok {
		isCurrentOnly := parseBoolQuery(r, "is_current_only")
		filters.IsCurrentOnly = &isCurrentOnly
	}
	req := &mallpb.ListBundlesAdminRequest{
		Filters:  filters,
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
	}
	resp, err := h.Mall.ListBundlesAdmin(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) PublishBundle(w http.ResponseWriter, r *http.Request) {
	bundleULID := strings.TrimSpace(chi.URLParam(r, "bundle_ulid"))
	if !requireRequestField(w, bundleULID, "bundle_ulid") {
		return
	}
	resp, err := h.Mall.PublishBundle(r.Context(), &mallpb.PublishBundleRequest{BundleUlid: bundleULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeprecateBundle(w http.ResponseWriter, r *http.Request) {
	bundleULID := strings.TrimSpace(chi.URLParam(r, "bundle_ulid"))
	if !requireRequestField(w, bundleULID, "bundle_ulid") {
		return
	}
	resp, err := h.Mall.DeprecateBundle(r.Context(), &mallpb.DeprecateBundleRequest{BundleUlid: bundleULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeleteBundle(w http.ResponseWriter, r *http.Request) {
	bundleULID := strings.TrimSpace(chi.URLParam(r, "bundle_ulid"))
	if !requireRequestField(w, bundleULID, "bundle_ulid") {
		return
	}
	resp, err := h.Mall.DeleteBundle(r.Context(), &mallpb.DeleteBundleRequest{BundleUlid: bundleULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetBundleJsonSchemas(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Mall.GetBundleJsonSchemas(r.Context(), &mallpb.GetBundleJsonSchemasRequest{})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminSyncBundleDisplayPricing(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BundleUlid string `json:"bundle_ulid"`
	}
	if r.Body != nil {
		_ = ReadJSON(r, &body)
	}
	bundleULID := strings.TrimSpace(firstNonEmpty(body.BundleUlid, r.URL.Query().Get("bundle_ulid")))
	req := &mallpb.AdminSyncBundleDisplayPricingRequest{}
	if bundleULID != "" {
		req.BundleUlid = &bundleULID
	}
	resp, err := h.Mall.AdminSyncBundleDisplayPricing(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListBundleOrders(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	resp, err := h.Mall.ListBundleOrders(r.Context(), &mallpb.ListBundleOrdersRequest{
		Filters: &mallpb.BundleOrderFilters{
			CandidateUlid: strings.TrimSpace(r.URL.Query().Get("candidate_ulid")),
			BundleUlid:    strings.TrimSpace(r.URL.Query().Get("bundle_ulid")),
			OrderStatus:   strings.TrimSpace(r.URL.Query().Get("order_status")),
		},
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetBundleOrderSummary(w http.ResponseWriter, r *http.Request) {
	bundleOrderULID := strings.TrimSpace(chi.URLParam(r, "bundle_order_ulid"))
	if !requireRequestField(w, bundleOrderULID, "bundle_order_ulid") {
		return
	}
	resp, err := h.Mall.GetBundleOrderSummary(r.Context(), &mallpb.GetBundleOrderSummaryRequest{
		BundleOrderUlid: bundleOrderULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetBundleOrderDetail(w http.ResponseWriter, r *http.Request) {
	bundleOrderULID := strings.TrimSpace(chi.URLParam(r, "bundle_order_ulid"))
	if !requireRequestField(w, bundleOrderULID, "bundle_order_ulid") {
		return
	}
	resp, err := h.Mall.AdminGetBundleOrderDetail(r.Context(), &mallpb.AdminGetBundleOrderDetailRequest{
		BundleOrderUlid: bundleOrderULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminPurgeCandidateBundle(w http.ResponseWriter, r *http.Request) {
	var req mallpb.AdminPurgeCandidateBundleRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if req.AdminUlid == "" {
		req.AdminUlid = AdminID(r)
	}
	if !requireRequestFields(w, req.CandidateUlid, "candidate_ulid", req.BundleOrderUlid, "bundle_order_ulid", req.AdminUlid, "admin_ulid") {
		return
	}
	resp, err := h.Mall.AdminPurgeCandidateBundle(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateBundleUploadURL(w http.ResponseWriter, r *http.Request) {
	var req mallpb.CreateBundleUploadURLRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if !requireRequestFields(w, req.FileName, "file_name", req.ContentType, "content_type", req.BundleUlid, "bundle_ulid", req.FileHash, "file_hash") {
		return
	}
	resp, err := h.Mall.CreateBundleUploadURL(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
