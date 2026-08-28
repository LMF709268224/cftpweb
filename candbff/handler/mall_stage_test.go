package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	listReqs    []*mallpb.ListPipelineOrdersRequest
	detailReqs  []*mallpb.GetPipelineOrderDetailRequest
	createReq   *mallpb.CreateStageOrderRequest
	summaryReq  *mallpb.GetStageOrderSummaryRequest
	selectReq   *mallpb.SelectStageExemptionsRequest
	listResps   []*mallpb.ListPipelineOrdersResponse
	detailResps map[string]*mallpb.GetPipelineOrderDetailResponse
	createResp  *mallpb.CreateStageOrderResponse
	summaryResp *mallpb.GetStageOrderSummaryResponse
	selectResp  *mallpb.SelectStageExemptionsResponse
}

func (s *stageOrderMallClientStub) ListPipelineOrders(
	_ context.Context,
	req *mallpb.ListPipelineOrdersRequest,
	_ ...grpc.CallOption,
) (*mallpb.ListPipelineOrdersResponse, error) {
	s.listReqs = append(s.listReqs, req)
	index := len(s.listReqs) - 1
	if index >= len(s.listResps) {
		return &mallpb.ListPipelineOrdersResponse{}, nil
	}
	return s.listResps[index], nil
}

func (s *stageOrderMallClientStub) GetPipelineOrderDetail(
	_ context.Context,
	req *mallpb.GetPipelineOrderDetailRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetPipelineOrderDetailResponse, error) {
	s.detailReqs = append(s.detailReqs, req)
	return s.detailResps[req.GetPipelineOrderUlid()], nil
}

func (s *stageOrderMallClientStub) CreateStageOrder(
	_ context.Context,
	req *mallpb.CreateStageOrderRequest,
	_ ...grpc.CallOption,
) (*mallpb.CreateStageOrderResponse, error) {
	s.createReq = req
	return s.createResp, nil
}

func (s *stageOrderMallClientStub) GetStageOrderSummary(
	_ context.Context,
	req *mallpb.GetStageOrderSummaryRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetStageOrderSummaryResponse, error) {
	s.summaryReq = req
	return s.summaryResp, nil
}

func (s *stageOrderMallClientStub) SelectStageExemptions(
	_ context.Context,
	req *mallpb.SelectStageExemptionsRequest,
	_ ...grpc.CallOption,
) (*mallpb.SelectStageExemptionsResponse, error) {
	s.selectReq = req
	return s.selectResp, nil
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
		selectionJSON   = `{"stages":[{"stage_cc_ulid":"01KYN000000000000000000004","exempted_unit_cc_ulids":[],"waived_unit_cc_ulids":[]}]}`
	)

	mall := &stageOrderMallClientStub{
		listResps: []*mallpb.ListPipelineOrdersResponse{{
			Items: []*mallpb.PipelineOrderSummary{{
				PipelineOrderUlid: pipelineOrderID,
			}},
		}},
		detailResps: map[string]*mallpb.GetPipelineOrderDetailResponse{
			pipelineOrderID: {
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
		strings.NewReader(`{"pipeline_ulid":"`+pipelineUlid+`","stage_ulid":"`+stageUlid+`","selected_exemptions_json":`+strconv.Quote(selectionJSON)+`}`),
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
	if len(mall.listReqs) != 1 {
		t.Fatalf("list requests = %d, want 1", len(mall.listReqs))
	}
	filters := mall.listReqs[0].GetFilters()
	if filters.GetCandidateUlid() != candidateID ||
		filters.GetPipelineCcUlid() != pipelineCcUlid ||
		filters.GetOrderStatus() != "COMPLETED" ||
		filters.GetPaymentMode() != "BY_STAGE" {
		t.Fatalf("pipeline order filters = %+v", filters)
	}
	if len(mall.detailReqs) != 1 || mall.detailReqs[0].GetPipelineOrderUlid() != pipelineOrderID {
		t.Fatalf("pipeline order detail requests = %+v", mall.detailReqs)
	}
	if mall.createReq.GetCandidateUlid() != candidateID ||
		mall.createReq.GetPipelineCcUlid() != pipelineCcUlid ||
		mall.createReq.GetPipelineOrderUlid() != pipelineOrderID ||
		mall.createReq.GetStageUlid() != stageUlid ||
		mall.createReq.GetStageCcUlid() != stageCcUlid ||
		mall.createReq.GetBundleOrderUlid() != bundleOrderID ||
		mall.createReq.GetSelectedExemptionsJson() != selectionJSON {
		t.Fatalf("create stage order request = %+v", mall.createReq)
	}
}

func TestSelectStageExemptionsVerifiesOwnershipAndForwardsSelection(t *testing.T) {
	const (
		candidateID  = "01KYN000000000000000000021"
		stageOrderID = "01KYN000000000000000000022"
		stageCcUlid  = "01KYN000000000000000000023"
		firstUnitID  = "01KYN000000000000000000024"
		secondUnitID = "01KYN000000000000000000025"
		waivedUnitID = "01KYN000000000000000000026"
	)

	mall := &stageOrderMallClientStub{
		summaryResp: &mallpb.GetStageOrderSummaryResponse{
			Found: true,
			Summary: &mallpb.StageOrderSummary{
				StageOrderUlid: stageOrderID,
				CandidateUlid:  candidateID,
				StageCcUlid:    stageCcUlid,
				OrderStatus:    "WAIT_EXEMPTION_SELECTION",
			},
		},
		selectResp: &mallpb.SelectStageExemptionsResponse{
			StageOrderUlid: stageOrderID,
			OrderStatus:    "WAIT_STAGE_PAYMENT",
		},
	}
	h := &Handler{Mall: mall}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/mall/stage-orders/"+stageOrderID+"/exemptions",
		strings.NewReader(`{"stage_cc_ulid":"`+stageCcUlid+`","exempted_unit_cc_ulids":["`+firstUnitID+`"," `+firstUnitID+` ","`+secondUnitID+`"],"waived_unit_cc_ulids":[" `+waivedUnitID+` ","`+waivedUnitID+`"]}`),
	)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("stageOrderId", stageOrderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	req = req.WithContext(WithCandidate(ctx, candidateID, "", "", ""))
	rec := httptest.NewRecorder()

	h.SelectStageExemptions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if mall.summaryReq.GetStageOrderUlid() != stageOrderID {
		t.Fatalf("stage order summary request = %+v", mall.summaryReq)
	}
	if mall.selectReq.GetStageOrderUlid() != stageOrderID {
		t.Fatalf("select stage exemptions request = %+v", mall.selectReq)
	}
	const expectedJSON = `{"exempted_unit_cc_ulids":["01KYN000000000000000000024","01KYN000000000000000000025"],"stage_cc_ulid":"01KYN000000000000000000023","waived_unit_cc_ulids":["01KYN000000000000000000026"]}`
	if mall.selectReq.GetExemptionsJson() != expectedJSON {
		t.Fatalf("exemptions json = %s, want %s", mall.selectReq.GetExemptionsJson(), expectedJSON)
	}
}

func TestSelectStageExemptionsRejectsAnotherCandidateOrder(t *testing.T) {
	const (
		candidateID  = "01KYN000000000000000000031"
		ownerID      = "01KYN000000000000000000032"
		stageOrderID = "01KYN000000000000000000033"
		stageCcUlid  = "01KYN000000000000000000034"
	)

	mall := &stageOrderMallClientStub{
		summaryResp: &mallpb.GetStageOrderSummaryResponse{
			Found: true,
			Summary: &mallpb.StageOrderSummary{
				StageOrderUlid: stageOrderID,
				CandidateUlid:  ownerID,
				StageCcUlid:    stageCcUlid,
				OrderStatus:    "WAIT_EXEMPTION_SELECTION",
			},
		},
	}
	h := &Handler{Mall: mall}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/mall/stage-orders/"+stageOrderID+"/exemptions",
		strings.NewReader(`{"stage_cc_ulid":"`+stageCcUlid+`","exempted_unit_cc_ulids":[]}`),
	)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("stageOrderId", stageOrderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	req = req.WithContext(WithCandidate(ctx, candidateID, "", "", ""))
	rec := httptest.NewRecorder()

	h.SelectStageExemptions(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if mall.selectReq != nil {
		t.Fatalf("SelectStageExemptions should not be called for another candidate's order")
	}
}

func TestCompletedByStagePipelineOrderFollowsPagination(t *testing.T) {
	const (
		candidateID       = "01KYN000000000000000000011"
		pipelineCcUlid    = "01KYN000000000000000000012"
		targetPipelineID  = "01KYN000000000000000000013"
		firstOrderID      = "01KYN000000000000000000014"
		targetOrderID     = "01KYN000000000000000000015"
		targetBundleOrder = "01KYN000000000000000000016"
	)

	mall := &stageOrderMallClientStub{
		listResps: []*mallpb.ListPipelineOrdersResponse{
			{
				Items: []*mallpb.PipelineOrderSummary{{
					PipelineOrderUlid: firstOrderID,
				}},
				HasMore:    true,
				NextCursor: "next-page",
			},
			{
				Items: []*mallpb.PipelineOrderSummary{{
					PipelineOrderUlid: targetOrderID,
				}},
			},
		},
		detailResps: map[string]*mallpb.GetPipelineOrderDetailResponse{
			firstOrderID: {
				Found: true,
				Detail: &mallpb.PipelineOrderDetail{
					Summary: &mallpb.PipelineOrderSummary{
						PipelineOrderUlid: firstOrderID,
					},
					InstantiatedPipelineUlid: "different-runtime-pipeline",
				},
			},
			targetOrderID: {
				Found: true,
				Detail: &mallpb.PipelineOrderDetail{
					Summary: &mallpb.PipelineOrderSummary{
						PipelineOrderUlid: targetOrderID,
						BundleOrderUlid:   targetBundleOrder,
					},
					InstantiatedPipelineUlid: targetPipelineID,
				},
			},
		},
	}
	h := &Handler{Mall: mall}

	got, err := h.completedByStagePipelineOrder(
		context.Background(),
		candidateID,
		pipelineCcUlid,
		targetPipelineID,
	)
	if err != nil {
		t.Fatalf("completedByStagePipelineOrder() error = %v", err)
	}
	if got == nil || got.GetPipelineOrderUlid() != targetOrderID || got.GetBundleOrderUlid() != targetBundleOrder {
		t.Fatalf("completedByStagePipelineOrder() = %+v", got)
	}
	if len(mall.listReqs) != 2 {
		t.Fatalf("list requests = %d, want 2", len(mall.listReqs))
	}
	if mall.listReqs[0].GetCursor() != "" || mall.listReqs[1].GetCursor() != "next-page" {
		t.Fatalf("list cursors = %q, %q", mall.listReqs[0].GetCursor(), mall.listReqs[1].GetCursor())
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
