package handler

import (
	"context"
	"fmt"
	"testing"

	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc"
)

type truncatedDashboardProgClient struct {
	gprogpb.ProgServiceClient
	calls int
}

func (s *truncatedDashboardProgClient) ListPipelines(_ context.Context, _ *gprogpb.ListPipelinesReq, _ ...grpc.CallOption) (*gprogpb.ListPipelinesRsp, error) {
	s.calls++
	items := make([]*gprogpb.PipelineSummary, adminDashboardPageSize)
	for index := range items {
		items[index] = &gprogpb.PipelineSummary{PipelineUlid: fmt.Sprintf("pipeline-%d-%d", s.calls, index)}
	}
	return &gprogpb.ListPipelinesRsp{
		Pipelines:  items,
		HasMore:    true,
		NextCursor: fmt.Sprintf("pipeline-cursor-%d", s.calls),
	}, nil
}

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
	prog := &truncatedDashboardProgClient{}
	pay := &truncatedDashboardPayClient{}
	handler := &Handler{Gprog: prog, Gpay: pay}

	pipelines, pipelinesExact, err := handler.listDashboardPipelines(context.Background())
	if err != nil {
		t.Fatalf("listDashboardPipelines() error = %v", err)
	}
	if pipelinesExact || len(pipelines) != adminDashboardSampleLimit || prog.calls != adminDashboardSampleLimit/adminDashboardPageSize {
		t.Fatalf("pipelines exact=%v len=%d calls=%d", pipelinesExact, len(pipelines), prog.calls)
	}

	orders, ordersExact, err := handler.listDashboardPaidOrders(context.Background())
	if err != nil {
		t.Fatalf("listDashboardPaidOrders() error = %v", err)
	}
	if ordersExact || len(orders) != adminDashboardSampleLimit || pay.calls != adminDashboardSampleLimit/adminDashboardPageSize {
		t.Fatalf("orders exact=%v len=%d calls=%d", ordersExact, len(orders), pay.calls)
	}
}
