package handler

import (
	"context"
	"net/http"
	"strings"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetStageOrderStatus(w http.ResponseWriter, r *http.Request) {
	stageULID := strings.TrimSpace(chi.URLParam(r, "stage_ulid"))
	if !requireRequestField(w, stageULID, "stage_ulid") {
		return
	}

	resp, err := h.Mall.GetStageOrderStatus(r.Context(), &mallpb.GetStageOrderStatusRequest{
		StageOrderUlid: stageULID, // this was StageUlid, but the field is actually StageOrderUlid according to the pb definition... wait, the route parameter was stage_ulid. So we map it to StageOrderUlid for now.
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListStageOrders(w http.ResponseWriter, r *http.Request) {
	candidateULID := strings.TrimSpace(r.URL.Query().Get("candidate_ulid"))
	stageULID := strings.TrimSpace(r.URL.Query().Get("stage_ulid"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	req := &mallpb.ListStageOrdersRequest{
		CandidateUlid: candidateULID,
		StageCcUlid:   stageULID, // The proto uses stage_cc_ulid
		OrderStatus:   status,    // The proto uses order_status string
		Limit:         int32(parseUint32Query(r, "limit")),
		Offset:        int32(parseUint32Query(r, "offset")),
	}

	resp, err := h.Mall.ListStageOrders(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	candidateULID := strings.TrimSpace(r.URL.Query().Get("candidate_ulid"))
	bizType := strings.TrimSpace(r.URL.Query().Get("biz_type"))
	bizRefULID := strings.TrimSpace(r.URL.Query().Get("biz_ref_ulid"))
	orderStatus := strings.TrimSpace(r.URL.Query().Get("order_status"))
	paymentStatus := strings.TrimSpace(r.URL.Query().Get("payment_status"))

	req := &mallpb.ListOrdersRequest{
		CandidateUlid: candidateULID,
		BizType:       bizType,
		BizRefUlid:    bizRefULID,
		OrderStatus:   orderStatus,
		PaymentStatus: paymentStatus,
		Limit:         int32(parseUint32Query(r, "limit")),
		Offset:        int32(parseUint32Query(r, "offset")),
	}

	resp, err := h.Mall.ListOrders(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

type adminPurgeOrderBundleDataRequest struct {
	CandidateUlid string `json:"candidate_ulid"`
	BizType       string `json:"biz_type"`
	BizRefUlid    string `json:"biz_ref_ulid"`
}

func (h *Handler) AdminPurgeOrderBundleData(w http.ResponseWriter, r *http.Request) {
	var req adminPurgeOrderBundleDataRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}

	candidateULID := strings.TrimSpace(req.CandidateUlid)
	bizType := strings.ToUpper(strings.TrimSpace(req.BizType))
	bizRefULID := strings.TrimSpace(req.BizRefUlid)
	if !requireRequestFields(w, candidateULID, "candidate_ulid", bizType, "biz_type", bizRefULID, "biz_ref_ulid") {
		return
	}

	bundleOrderULID := bizRefULID
	if bizType == "PIPELINE_UNLOCK" {
		resolved, ok := h.resolveBundleOrderForPipelineUnlock(w, r, candidateULID, bizRefULID)
		if !ok {
			return
		}
		bundleOrderULID = resolved
	} else if bizType != "BUNDLE_PURCHASE" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "biz_type does not support bundle data purge")
		return
	}

	adminULID := AdminID(r)
	if !requireRequestField(w, adminULID, "admin_ulid") {
		return
	}

	resp, err := h.Mall.AdminPurgeCandidateBundle(r.Context(), &mallpb.AdminPurgeCandidateBundleRequest{
		CandidateUlid:   candidateULID,
		BundleOrderUlid: bundleOrderULID,
		AdminUlid:       adminULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"bundle_order_ulid": bundleOrderULID,
		"result":            resp,
	})
}

func (h *Handler) resolveBundleOrderForPipelineUnlock(w http.ResponseWriter, r *http.Request, candidateULID, unlockOrderULID string) (string, bool) {
	resp, err := h.Mall.GetPipelineUnlockOrderDetail(r.Context(), &mallpb.GetPipelineUnlockOrderDetailRequest{
		PipelineUnlockOrderUlid: unlockOrderULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return "", false
	}
	if !resp.GetFound() || resp.GetDetail() == nil || resp.GetDetail().GetSummary() == nil {
		WriteError(w, http.StatusNotFound, ErrNotFound, "pipeline unlock order not found")
		return "", false
	}

	summary := resp.GetDetail().GetSummary()
	if summary.GetCandidateUlid() != "" && summary.GetCandidateUlid() != candidateULID {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "pipeline unlock order does not belong to candidate")
		return "", false
	}

	pipelineCcULID := strings.TrimSpace(summary.GetPipelineCcUlid())
	if !requireRequestField(w, pipelineCcULID, "pipeline_cc_ulid") {
		return "", false
	}

	bundleOrderULID, err := h.findBundleOrderForCandidatePipeline(r.Context(), candidateULID, pipelineCcULID)
	if err != nil {
		HandleGrpcError(w, err)
		return "", false
	}
	if bundleOrderULID == "" {
		WriteError(w, http.StatusNotFound, ErrNotFound, "related bundle order not found for pipeline unlock order")
		return "", false
	}
	return bundleOrderULID, true
}

func (h *Handler) findBundleOrderForCandidatePipeline(ctx context.Context, candidateULID, pipelineCcULID string) (string, error) {
	const pageSize int32 = 100
	for offset := int32(0); ; offset += pageSize {
		resp, err := h.Mall.ListBundleOrders(ctx, &mallpb.ListBundleOrdersRequest{
			CandidateUlid: candidateULID,
			Limit:         pageSize,
			Offset:        offset,
		})
		if err != nil {
			return "", err
		}

		for _, item := range resp.GetItems() {
			bundleOrderULID := strings.TrimSpace(item.GetBundleOrderUlid())
			if bundleOrderULID == "" {
				continue
			}
			detail, err := h.Mall.GetBundleOrderDetail(ctx, &mallpb.GetBundleOrderDetailRequest{
				BundleOrderUlid: bundleOrderULID,
			})
			if err != nil {
				return "", err
			}
			if detail.GetFound() && detail.GetDetail() != nil && strings.Contains(detail.GetDetail().GetItemsSnapshotJson(), pipelineCcULID) {
				return bundleOrderULID, nil
			}
		}

		if len(resp.GetItems()) < int(pageSize) || resp.GetTotal() <= offset+pageSize {
			return "", nil
		}
	}
}
