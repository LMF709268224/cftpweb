package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc"
)

type stageOrderMallClientStub struct {
	mallpb.MallServiceClient
	listReq    *mallpb.ListPipelineOrdersRequest
	detailReq  *mallpb.GetPipelineOrderDetailRequest
	createReq  *mallpb.CreateStageOrderRequest
	listResp   *mallpb.ListPipelineOrdersResponse
	detailResp *mallpb.GetPipelineOrderDetailResponse
	createResp *mallpb.CreateStageOrderResponse
}

func (s *stageOrderMallClientStub) ListPipelineOrders(
	_ context.Context,
	req *mallpb.ListPipelineOrdersRequest,
	_ ...grpc.CallOption,
) (*mallpb.ListPipelineOrdersResponse, error) {
	s.listReq = req
	return s.listResp, nil
}

func (s *stageOrderMallClientStub) GetPipelineOrderDetail(
	_ context.Context,
	req *mallpb.GetPipelineOrderDetailRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetPipelineOrderDetailResponse, error) {
	s.detailReq = req
	return s.detailResp, nil
}

func (s *stageOrderMallClientStub) CreateStageOrder(
	_ context.Context,
	req *mallpb.CreateStageOrderRequest,
	_ ...grpc.CallOption,
) (*mallpb.CreateStageOrderResponse, error) {
	s.createReq = req
	return s.createResp, nil
}

type stageOrderProgClientStub struct {
	gprogpb.ProgServiceClient
	detailReq  *gprogpb.GetPipelineDetailReq
	detailResp *gprogpb.GetPipelineDetailRsp
}

func (s *stageOrderProgClientStub) GetPipelineDetail(
	_ context.Context,
	req *gprogpb.GetPipelineDetailReq,
	_ ...grpc.CallOption,
) (*gprogpb.GetPipelineDetailRsp, error) {
	s.detailReq = req
	return s.detailResp, nil
}

func TestCreateStageOrderUsesCompletedByStageOrderContext(t *testing.T) {
	const (
		candidateID     = "01KYN000000000000000000001"
		pipelineCcUlid  = "01KYN000000000000000000002"
		pipelineUlid    = "01KYN000000000000000000003"
		stageCcUlid     = "01KYN000000000000000000004"
		stageUlid       = "01KYN000000000000000000005"
		pipelineOrderID = "01KYN000000000000000000006"
		bundleOrderID   = "01KYN000000000000000000007"
		createdStageID  = "01KYN000000000000000000008"
	)

	mall := &stageOrderMallClientStub{
		listResp: &mallpb.ListPipelineOrdersResponse{
			Items: []*mallpb.PipelineOrderSummary{{
				PipelineOrderUlid: pipelineOrderID,
			}},
		},
		detailResp: &mallpb.GetPipelineOrderDetailResponse{
			Found: true,
			Detail: &mallpb.PipelineOrderDetail{
				Summary: &mallpb.PipelineOrderSummary{
					PipelineOrderUlid: pipelineOrderID,
					CandidateUlid:     candidateID,
					PipelineCcUlid:    pipelineCcUlid,
					PaymentMode:       "BY_STAGE",
					OrderStatus:       "COMPLETED",
					BundleOrderUlid:   bundleOrderID,
				},
				InstantiatedPipelineUlid: pipelineUlid,
			},
		},
		createResp: &mallpb.CreateStageOrderResponse{
			StageOrderUlid: createdStageID,
			OrderStatus:    "WAIT_STAGE_PAYMENT",
		},
	}
	prog := &stageOrderProgClientStub{
		detailResp: &gprogpb.GetPipelineDetailRsp{
			Pipeline: &gprogpb.PipelineDetail{
				PipelineUlid:   pipelineUlid,
				CandidateUlid:  candidateID,
				PipelineCcUlid: pipelineCcUlid,
			},
			Stages: []*gprogpb.StageDetail{{
				Stage: &gprogpb.StageSummary{
					StageUlid:   stageUlid,
					StageCcUlid: stageCcUlid,
					Status:      gprogpb.StageStatus_STAGE_STATUS_WAIT_CANDIDATE,
				},
			}},
		},
	}
	h := &Handler{Mall: mall, Gprog: prog}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/mall/pipelines/"+pipelineCcUlid+"/stages/"+stageCcUlid+"/purchase",
		strings.NewReader(`{"pipeline_ulid":"`+pipelineUlid+`","stage_ulid":"`+stageUlid+`"}`),
	)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("pipelineId", pipelineCcUlid)
	routeCtx.URLParams.Add("stageId", stageCcUlid)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	req = req.WithContext(WithCandidate(ctx, candidateID, "", "", ""))
	rec := httptest.NewRecorder()

	h.CreateStageOrder(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if prog.detailReq.GetPipelineUlid() != pipelineUlid {
		t.Fatalf("runtime lookup = %+v, want pipeline_ulid %q", prog.detailReq, pipelineUlid)
	}
	filters := mall.listReq.GetFilters()
	if filters.GetCandidateUlid() != candidateID ||
		filters.GetPipelineCcUlid() != pipelineCcUlid ||
		filters.GetOrderStatus() != "COMPLETED" ||
		filters.GetPaymentMode() != "BY_STAGE" {
		t.Fatalf("pipeline order filters = %+v", filters)
	}
	if mall.detailReq.GetPipelineOrderUlid() != pipelineOrderID {
		t.Fatalf("pipeline order detail request = %+v", mall.detailReq)
	}
	if mall.createReq.GetCandidateUlid() != candidateID ||
		mall.createReq.GetPipelineCcUlid() != pipelineCcUlid ||
		mall.createReq.GetPipelineOrderUlid() != pipelineOrderID ||
		mall.createReq.GetStageUlid() != stageUlid ||
		mall.createReq.GetStageCcUlid() != stageCcUlid ||
		mall.createReq.GetBundleOrderUlid() != bundleOrderID {
		t.Fatalf("create stage order request = %+v", mall.createReq)
	}
}

func TestBuildPipelineNextStepReturnsRuntimeAndConfigStageIDs(t *testing.T) {
	const (
		stageCcUlid = "stage-config"
		stageUlid   = "stage-runtime"
	)
	config := &gccpb.PipelineConfig{
		Stages: []*gccpb.StageConfig{{
			StageUlid: stageCcUlid,
			Name:      "Level 2",
			Units: []*gccpb.UnitConfig{{
				UnitUlid:       "unit-config",
				GlmsCourseUlid: "course",
			}},
		}},
	}
	runtime := &gprogpb.GetPipelineDetailRsp{
		Pipeline: &gprogpb.PipelineDetail{
			Status: gprogpb.PipelineStatus_PIPELINE_STATUS_RUNNING,
		},
		Stages: []*gprogpb.StageDetail{{
			Stage: &gprogpb.StageSummary{
				StageUlid:   stageUlid,
				StageCcUlid: stageCcUlid,
				Status:      gprogpb.StageStatus_STAGE_STATUS_WAIT_CANDIDATE,
			},
		}},
	}

	got := buildPipelineNextStep(runtime, config, &gprogpb.PipelineSummary{
		Status: gprogpb.PipelineStatus_PIPELINE_STATUS_RUNNING,
	})

	if got.StageUlid != stageUlid || got.StageCcUlid != stageCcUlid {
		t.Fatalf("next step stage IDs = runtime:%q config:%q", got.StageUlid, got.StageCcUlid)
	}
	if got.Action != "wait_candidate" {
		t.Fatalf("next step action = %q, want wait_candidate", got.Action)
	}
}
