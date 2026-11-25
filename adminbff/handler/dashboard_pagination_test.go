package handler

import (
	"context"
	"fmt"
	"testing"

	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	"google.golang.org/grpc"
)

type truncatedDashboardPayClient struct {
	gpaypb.PayServiceClient
	calls int
}

func (s *truncatedDashboardPayClient) ListOrders(_ context.Context, _ *gpaypb.ListOrdersRequest, _ ...grpc.CallOption) (*gpaypb.ListOrdersResponse, error) {
	s.calls++
	items := make([]*gpaypb.OrderSummary, adminDashboardPageSize)
	for index := range items {
		items[index] = &gpaypb.OrderSummary{OrderUlid: fmt.Sprintf("order-%d-%d", s.calls, index)}
	}
	return &gpaypb.ListOrdersResponse{
		Orders:     items,
		HasMore:    true,
		NextCursor: fmt.Sprintf("order-cursor-%d", s.calls),
	}, nil
}

func TestDashboardAggregationMarksTruncatedResultsAsInexact(t *testing.T) {
	pay := &truncatedDashboardPayClient{}
	handler := &Handler{Gpay: pay}

	orders, ordersExact, err := handler.listDashboardPaidOrders(context.Background())
	if err != nil {
		t.Fatalf("listDashboardPaidOrders() error = %v", err)
	}
	if ordersExact || len(orders) != adminDashboardSampleLimit || pay.calls != adminDashboardSampleLimit/adminDashboardPageSize {
		t.Fatalf("orders exact=%v len=%d calls=%d", ordersExact, len(orders), pay.calls)
	}
}
