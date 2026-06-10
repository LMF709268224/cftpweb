package handler

import (
	"net/http"
)

// TODO: 会员模块需要新建 member 微服务，包含以下 gRPC 接口:
//   - GetMembership(candidate_id) → 当前会员等级、到期时间、权益列表
//   - GetMembershipBenefits(level)  → 权益详情

// GetMembership  GET /api/membership  获取当前会员信息
func (h *Handler) GetMembership(w http.ResponseWriter, r *http.Request) {
	// TODO: 需要新建 member 微服务
	WriteJSON(w, http.StatusOK, h.getMembershipRsp())
}

func (h *Handler) getMembershipRsp() GetMembershipRsp {
	return GetMembershipRsp{
		Level:     "Gold",
		ExpiresAt: "2025-12-31T23:59:59Z",
	}
}

type GetMembershipRsp struct {
	Level     string `json:"level"`      // 会员等级
	ExpiresAt string `json:"expires_at"` // 会员过期时间
}
