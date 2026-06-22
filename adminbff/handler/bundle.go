package handler

import (
	"net/http"
	"strings"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateBundle(w http.ResponseWriter, r *http.Request) {
	var req mallpb.CreateBundleRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if !requireRequestFields(w, req.BundleUlid, "bundle_ulid", req.BundleGpath, "bundle_gpath", req.Name, "name") {
		return
	}
	resp, err := h.Mall.CreateBundle(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) UpdateBundleMeta(w http.ResponseWriter, r *http.Request) {
	var req mallpb.UpdateBundleMetaRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if req.BundleUlid == "" {
		req.BundleUlid = strings.TrimSpace(chi.URLParam(r, "bundle_ulid"))
	}
	if !requireRequestField(w, req.BundleUlid, "bundle_ulid") {
		return
	}
	resp, err := h.Mall.UpdateBundleMeta(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateBundlePricing(w http.ResponseWriter, r *http.Request) {
	var req mallpb.UpdateBundlePricingRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if !requireRequestFields(w, req.BundleGpath, "bundle_gpath", req.PricingJson, "pricing_json", req.ItemsJson, "items_json") {
		return
	}
	resp, err := h.Mall.UpdateBundlePricing(r.Context(), &req)
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

	req := &mallpb.GetBundleRequest{}
	if bundleULID != "" {
		req.Query = &mallpb.GetBundleRequest_BundleUlid{BundleUlid: bundleULID}
	} else {
		req.Query = &mallpb.GetBundleRequest_BundleGpath{BundleGpath: bundleGPath}
	}
	resp, err := h.Mall.GetBundle(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListBundles(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Mall.ListBundles(r.Context(), &mallpb.ListBundlesRequest{
		Limit:  int32Query(r, "limit", 20),
		Offset: int32Query(r, "offset", 0),
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
	})
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

func (h *Handler) ListBundleOrders(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Mall.ListBundleOrders(r.Context(), &mallpb.ListBundleOrdersRequest{
		CandidateUlid: strings.TrimSpace(r.URL.Query().Get("candidate_ulid")),
		BundleUlid:    strings.TrimSpace(r.URL.Query().Get("bundle_ulid")),
		OrderStatus:   strings.TrimSpace(r.URL.Query().Get("order_status")),
		Limit:         int32Query(r, "limit", 20),
		Offset:        int32Query(r, "offset", 0),
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
	resp, err := h.Mall.GetBundleOrderDetail(r.Context(), &mallpb.GetBundleOrderDetailRequest{
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
