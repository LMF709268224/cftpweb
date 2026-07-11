package handler

import (
	"net/http"
	"strconv"
	"strings"

	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	"github.com/go-chi/chi/v5"
)

func int32Query(r *http.Request, key string, fallback int32) int32 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return fallback
	}
	return int32(v)
}

func uint32Query(r *http.Request, key string, fallback uint32) uint32 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return fallback
	}
	return uint32(v)
}

func (h *Handler) ListMemberships(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	resp, err := h.Gmbr.ListMemberships(r.Context(), &gmbrpb.ListMembershipsRequest{
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
		SortOrder: gmbrpb.SortOrder(page.Sort),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMembership(w http.ResponseWriter, r *http.Request) {
	membershipULID := strings.TrimSpace(chi.URLParam(r, "membership_ulid"))
	if !requireRequestField(w, membershipULID, "membership_ulid") {
		return
	}
	resp, err := h.Gmbr.GetMembership(r.Context(), &gmbrpb.GetMembershipRequest{
		MembershipUlid: membershipULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetActiveMembership(w http.ResponseWriter, r *http.Request) {
	candidateULID := strings.TrimSpace(r.URL.Query().Get("candidate_ulid"))
	if !requireRequestField(w, candidateULID, "candidate_ulid") {
		return
	}
	resp, err := h.Gmbr.GetActiveMembership(r.Context(), &gmbrpb.GetActiveMembershipRequest{
		CandidateUlid: candidateULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListUserMemberships(w http.ResponseWriter, r *http.Request) {
	candidateULID := strings.TrimSpace(r.URL.Query().Get("candidate_ulid"))
	if !requireRequestField(w, candidateULID, "candidate_ulid") {
		return
	}
	page := parseCursorPage(r, 20)
	resp, err := h.Gmbr.ListUserMemberships(r.Context(), &gmbrpb.ListUserMembershipsRequest{
		Filters: &gmbrpb.UserMembershipFilters{
			CandidateUlid: candidateULID,
		},
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
		SortOrder: gmbrpb.SortOrder(page.Sort),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListMembershipBillings(w http.ResponseWriter, r *http.Request) {
	candidateULID := strings.TrimSpace(r.URL.Query().Get("candidate_ulid"))
	membershipRecordULID := firstNonEmpty(
		strings.TrimSpace(r.URL.Query().Get("membership_record_ulid")),
		strings.TrimSpace(r.URL.Query().Get("membership_record_id")),
	)
	page := parseCursorPage(r, 20)
	resp, err := h.Gmbr.ListMembershipBillings(r.Context(), &gmbrpb.ListMembershipBillingsRequest{
		Filters: &gmbrpb.MembershipBillingFilters{
			CandidateUlid:        candidateULID,
			MembershipRecordUlid: membershipRecordULID,
		},
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
		SortOrder: gmbrpb.SortOrder(page.Sort),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminGrantMembership(w http.ResponseWriter, r *http.Request) {
	var req gmbrpb.AdminGrantMembershipRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if !requireRequestFields(w, req.CandidateUlid, "candidate_ulid", req.MembershipGpath, "membership_gpath") {
		return
	}
	resp, err := h.Gmbr.AdminGrantMembership(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminRevokeMembership(w http.ResponseWriter, r *http.Request) {
	var req gmbrpb.AdminRevokeMembershipRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if req.MembershipRecordUlid == "" {
		req.MembershipRecordUlid = strings.TrimSpace(r.URL.Query().Get("membership_record_id"))
	}
	if !requireRequestField(w, req.MembershipRecordUlid, "membership_record_ulid") {
		return
	}
	resp, err := h.Gmbr.AdminRevokeMembership(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminCreateMembershipConfig(w http.ResponseWriter, r *http.Request) {
	var req gmbrpb.AdminCreateMembershipConfigRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if !requireRequestFields(w, req.MembershipUlid, "membership_ulid", req.MembershipGpath, "membership_gpath", req.Name, "name") {
		return
	}
	resp, err := h.Gmbr.AdminCreateMembershipConfig(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) AdminUpdateMembershipConfig(w http.ResponseWriter, r *http.Request) {
	var req gmbrpb.AdminUpdateMembershipConfigRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if !requireRequestField(w, req.MembershipGpath, "membership_gpath") {
		return
	}
	resp, err := h.Gmbr.AdminUpdateMembershipConfig(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminPublishMembershipConfig(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, ErrInvalidRequest, "AdminPublishMembershipConfig is not supported by the current gmbr API")
}

func (h *Handler) AdminDeprecateMembershipConfig(w http.ResponseWriter, r *http.Request) {
	membershipULID := strings.TrimSpace(chi.URLParam(r, "membership_ulid"))
	membershipGpath := strings.TrimSpace(r.URL.Query().Get("membership_gpath"))
	if membershipGpath == "" && membershipULID != "" {
		resp, err := h.Gmbr.GetMembership(r.Context(), &gmbrpb.GetMembershipRequest{MembershipUlid: membershipULID})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		membershipGpath = resp.GetMembershipGpath()
	}
	if !requireRequestField(w, membershipGpath, "membership_gpath") {
		return
	}
	resp, err := h.Gmbr.AdminDeprecateMembershipConfig(r.Context(), &gmbrpb.AdminDeprecateMembershipConfigRequest{
		MembershipGpath: membershipGpath,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminPurgeCandidateMembership(w http.ResponseWriter, r *http.Request) {
	var req gmbrpb.AdminPurgeCandidateMembershipRequest
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
	resp, err := h.Gmbr.AdminPurgeCandidateMembership(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListMembershipMails(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	resp, err := h.Gmbr.ListMembershipMails(r.Context(), &gmbrpb.ListMembershipMailsRequest{
		Filters: &gmbrpb.MembershipMailFilters{
			CandidateUlid:    optionalString(strings.TrimSpace(r.URL.Query().Get("candidate_ulid"))),
			TaskStatus:       optionalString(strings.TrimSpace(r.URL.Query().Get("task_status"))),
			NotificationType: optionalString(strings.TrimSpace(r.URL.Query().Get("notification_type"))),
		},
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
		SortOrder: gmbrpb.SortOrder(page.Sort),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMembershipMailDetail(w http.ResponseWriter, r *http.Request) {
	mailULID := strings.TrimSpace(chi.URLParam(r, "mail_ulid"))
	if !requireRequestField(w, mailULID, "mail_ulid") {
		return
	}
	resp, err := h.Gmbr.GetMembershipMailDetail(r.Context(), &gmbrpb.GetMembershipMailDetailRequest{
		MailUlid: mailULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) RetryMembershipMail(w http.ResponseWriter, r *http.Request) {
	var req gmbrpb.RetryMembershipMailRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if req.AdminUlid == "" {
		req.AdminUlid = AdminID(r)
	}
	if !requireRequestFields(w, req.MailUlid, "mail_ulid", req.AdminUlid, "admin_ulid") {
		return
	}
	resp, err := h.Gmbr.RetryMembershipMail(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) IgnoreMembershipMail(w http.ResponseWriter, r *http.Request) {
	var req gmbrpb.IgnoreMembershipMailRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	if req.AdminUlid == "" {
		req.AdminUlid = AdminID(r)
	}
	if !requireRequestFields(w, req.MailUlid, "mail_ulid", req.AdminUlid, "admin_ulid") {
		return
	}
	resp, err := h.Gmbr.IgnoreMembershipMail(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
