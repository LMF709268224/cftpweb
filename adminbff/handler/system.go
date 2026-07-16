package handler

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

// GetSystemRedDots 聚合全站侧边栏红点数据
func (h *Handler) GetSystemRedDots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var rsp SystemRedDotsRsp
	var wg sync.WaitGroup

	// Applications (gcreds) - 这个目前微服务接口是支持按 Statuses 过滤的
	wg.Add(1)
	go func() {
		defer wg.Done()
		countResult, err := countCursorAll(ctx, func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
			res, err := h.Creds.GetApplicationCount(ctx, &gcredspb.GetApplicationCountRequest{
				Filters: &gcredspb.ApplicationFilters{
					Statuses: []string{"Pending", "Reupload"},
				},
				Limit:  limit,
				Cursor: cursor,
			})
			if err != nil {
				return 0, "", err
			}
			return res.GetCount(), res.GetNextCursor(), nil
		})
		if err == nil {
			atomic.AddUint32(&rsp.Applications, countResult.Total)
		}
	}()

	// TODO: 等待微服务团队在 cftp/gexam 中补充 GetAdminInterventionTaskCount 等接口
	// atomic.AddUint32(&rsp.Exams, 0)
	
	// TODO: 等待微服务团队在 cftp/gprog 的 CertificateTaskFilters 补充 Status 筛选字段
	// atomic.AddUint32(&rsp.Prog, 0)
	
	// TODO: 等待微服务团队在 cftp/gmall 和 gpay 补充相应 AdminCount 的 Status 过滤接口
	// atomic.AddUint32(&rsp.Orders, 0)
	// atomic.AddUint32(&rsp.Invoices, 0)

	// TODO: 等待微服务团队在 cftp/gmsg 的 MessageStatus 补充 FAILED 状态，目前只有 READ/UNREAD 等
	// atomic.AddUint32(&rsp.Messages, 0)

	wg.Wait()

	WriteJSON(w, http.StatusOK, rsp)
}
