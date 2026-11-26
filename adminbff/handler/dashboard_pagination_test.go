package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	"google.golang.org/grpc"
)

type pagedDashboardPayClient struct {
	gpaypb.PayServiceClient
	calls int
	now   time.Time
}

func (s *pagedDashboardPayClient) ListOrders(_ context.Context, request *gpaypb.ListOrdersRequest, _ ...grpc.CallOption) (*gpaypb.ListOrdersResponse, error) {
	filters := request.GetFilters()
	if filters.GetStatusFilter() != gpaypb.OrderStatus_ORDER_STATUS_COMPLETED || filters.PaidAtFrom == nil || *filters.PaidAtFrom != s.now.Add(-48*time.Hour).Unix() || filters.PaidAtTo == nil || *filters.PaidAtTo != s.now.Unix()+1 || request.GetPageSize() != adminDashboardPageSize {
		return nil, fmt.Errorf("unexpected order filters: %+v", request)
	}
	s.calls++
	pageSize := adminDashboardPageSize
	if s.calls == 6 {
		pageSize = 1
	}
	items := make([]*gpaypb.OrderSummary, pageSize)
	for index := range items {
		items[index] = &gpaypb.OrderSummary{OrderUlid: fmt.Sprintf("order-%d-%d", s.calls, index), PaidAt: s.now.Add(-time.Hour).Unix()}
	}
	return &gpaypb.ListOrdersResponse{
		Orders:     items,
		HasMore:    s.calls < 6,
		NextCursor: fmt.Sprintf("order-cursor-%d", s.calls),
	}, nil
}

func TestDashboardAggregationReadsAllTimeFilteredPages(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	pay := &pagedDashboardPayClient{now: now}
	handler := &Handler{Gpay: pay}

	orders, ordersExact, err := handler.listDashboardPaidOrders(context.Background(), now)
	if err != nil {
		t.Fatalf("listDashboardPaidOrders() error = %v", err)
	}
	if !ordersExact || len(orders) != 501 || pay.calls != 6 {
		t.Fatalf("orders exact=%v len=%d calls=%d", ordersExact, len(orders), pay.calls)
	}
}

type unfilteredDashboardPayClient struct {
	gpaypb.PayServiceClient
	paidAt int64
}

func (s *unfilteredDashboardPayClient) ListOrders(_ context.Context, _ *gpaypb.ListOrdersRequest, _ ...grpc.CallOption) (*gpaypb.ListOrdersResponse, error) {
	return &gpaypb.ListOrdersResponse{Orders: []*gpaypb.OrderSummary{{PaidAt: s.paidAt}}}, nil
}

func TestDashboardAggregationRejectsOrdersOutsideRequestedRange(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	pay := &unfilteredDashboardPayClient{paidAt: now.Add(-49 * time.Hour).Unix()}
	_, _, err := (&Handler{Gpay: pay}).listDashboardPaidOrders(context.Background(), now)
	if err == nil {
		t.Fatal("orders outside the paid_at filter must not appear in an exact dashboard total")
	}
}
