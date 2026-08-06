package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type writeHandlerMallClientStub struct {
	mallpb.MallServiceClient
	createReq  *mallpb.CreateBundleOrderRequest
	createResp *mallpb.CreateBundleOrderResponse
	createErr  error
}

func (s *writeHandlerMallClientStub) CreateBundleOrder(
	_ context.Context,
	req *mallpb.CreateBundleOrderRequest,
	_ ...grpc.CallOption,
) (*mallpb.CreateBundleOrderResponse, error) {
	s.createReq = req
	return s.createResp, s.createErr
}

type writeHandlerProgClientStub struct {
	gprogpb.ProgServiceClient
	signupReq  *gprogpb.CandidateSignupExamReq
	signupResp *gprogpb.CandidateSignupExamRsp
	signupErr  error
}

func (s *writeHandlerProgClientStub) CandidateSignupExam(
	_ context.Context,
	req *gprogpb.CandidateSignupExamReq,
	_ ...grpc.CallOption,
) (*gprogpb.CandidateSignupExamRsp, error) {
	s.signupReq = req
	return s.signupResp, s.signupErr
}

type writeHandlerLMSClientStub struct {
	lmspb.LmsServiceClient
	completeReq  *lmspb.CompleteLessonLearningRequest
	completeResp *lmspb.CompleteLessonLearningResponse
	completeErr  error
}

func (s *writeHandlerLMSClientStub) CompleteLessonLearning(
	_ context.Context,
	req *lmspb.CompleteLessonLearningRequest,
	_ ...grpc.CallOption,
) (*lmspb.CompleteLessonLearningResponse, error) {
	s.completeReq = req
	return s.completeResp, s.completeErr
}

func TestCreateBundleOrderForwardsCandidateAndOrderContract(t *testing.T) {
	const (
		candidateID   = "candidate-1"
		bundleID      = "bundle-config-1"
		bundleOrderID = "01KYN000000000000000000101"
	)
	client := &writeHandlerMallClientStub{
		createResp: &mallpb.CreateBundleOrderResponse{
			BundleOrderUlid: bundleOrderID,
			OrderStatus:     "WAIT_PAYMENT",
			PaymentKey:      "payment-key",
		},
	}
	handler := &Handler{Mall: client}
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/mall/bundles/"+bundleID+"/purchase",
		`{"payment_mode":"FULL_PIPELINE","selected_exemptions_json":"[]","bundle_order_ulid":"`+bundleOrderID+`"}`,
		candidateID,
		map[string]string{"bundleId": bundleID},
	)
	recorder := httptest.NewRecorder()

	handler.CreateBundleOrder(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if client.createReq == nil {
		t.Fatal("CreateBundleOrder was not called")
	}
	if client.createReq.GetCandidateUlid() != candidateID ||
		client.createReq.GetBundleCcUlid() != bundleID ||
		client.createReq.GetPaymentMode() != "FULL_PIPELINE" ||
		client.createReq.GetSelectedExemptionsJson() != "[]" ||
		client.createReq.GetBundleOrderUlid() != bundleOrderID {
		t.Fatalf("create bundle order request = %+v", client.createReq)
	}
}

func TestCreateBundleOrderGeneratesOrderIDAndPropagatesServiceError(t *testing.T) {
	client := &writeHandlerMallClientStub{
		createErr: status.Error(codes.Unavailable, "mall unavailable"),
	}
	handler := &Handler{Mall: client}
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/mall/bundles/bundle-config-1/purchase",
		`{"payment_mode":"BY_STAGE","selected_exemptions_json":"[]"}`,
		"candidate-1",
		map[string]string{"bundleId": "bundle-config-1"},
	)
	recorder := httptest.NewRecorder()

	handler.CreateBundleOrder(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusServiceUnavailable, ErrServiceUnavailable)
	if client.createReq == nil || client.createReq.GetBundleOrderUlid() == "" {
		t.Fatalf("generated bundle order ID was not forwarded: %+v", client.createReq)
	}
}

func TestSignupExamNormalizesCandidateProfileAndUsesRouteUnit(t *testing.T) {
	client := &writeHandlerProgClientStub{
		signupResp: &gprogpb.CandidateSignupExamRsp{
			CourseUnitUlid: "course-unit-route",
			Message:        "signup accepted",
		},
	}
	handler := &Handler{Gprog: client}
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/exams/units/course-unit-route/signup",
		`{
			"course_unit_ulid":"course-unit-body",
			"first_name":" Jane ",
			"middle_name":" Q ",
			"last_name":" Candidate ",
			"email":" jane@example.test ",
			"home_phone":"+65 6123 4567",
			"phone":"+65 8123 4567",
			"gender":"female",
			"birthdate":"1990-01-02",
			"country":" SG ",
			"province":" Singapore ",
			"city":" Singapore ",
			"address":" 1 Regression Road ",
			"postal_code":" 018956 "
		}`,
		"candidate-1",
		map[string]string{"courseUnitUlid": "course-unit-route"},
	)
	recorder := httptest.NewRecorder()

	handler.SignupExam(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.signupReq == nil {
		t.Fatal("CandidateSignupExam was not called")
	}
	if client.signupReq.GetCourseUnitUlid() != "course-unit-route" ||
		client.signupReq.GetCandidateUlid() != "candidate-1" ||
		client.signupReq.GetCandidateFirstName() != "Jane" ||
		client.signupReq.GetCandidateMiddleName() != "Q" ||
		client.signupReq.GetCandidateLastName() != "Candidate" ||
		client.signupReq.GetCandidateEmail() != "jane@example.test" ||
		client.signupReq.GetCandidateHomePhone() != "+6561234567" ||
		client.signupReq.GetCandidateWorkPhone() != "+6581234567" ||
		client.signupReq.GetCandidateGender() != "Female" ||
		client.signupReq.GetCandidateCountry() != "SG" ||
		client.signupReq.GetCandidatePostalCode() != "018956" ||
		client.signupReq.GetSourceSystem() != "candbff" {
		t.Fatalf("candidate signup request = %+v", client.signupReq)
	}
}

func TestCompletePipelineLessonForwardsOwnershipAndResponse(t *testing.T) {
	client := &writeHandlerLMSClientStub{
		completeResp: &lmspb.CompleteLessonLearningResponse{
			LessonStatus:         "completed",
			CourseCompleted:      true,
			CourseProgressStatus: "completed",
			CompletedAt:          "2026-08-06T10:00:00Z",
		},
	}
	handler := &Handler{Lms: client}
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/pipeline/lessons/lesson-1/complete",
		"",
		"candidate-1",
		map[string]string{"lessonId": "lesson-1"},
	)
	recorder := httptest.NewRecorder()

	handler.CompletePipelineLesson(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.completeReq == nil ||
		client.completeReq.GetCandidateUlid() != "candidate-1" ||
		client.completeReq.GetLessonUlid() != "lesson-1" {
		t.Fatalf("complete lesson request = %+v", client.completeReq)
	}
	var response struct {
		Data struct {
			LessonStatus    string `json:"lesson_status"`
			CourseCompleted bool   `json:"course_completed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
	if response.Data.LessonStatus != "completed" || !response.Data.CourseCompleted {
		t.Fatalf("complete lesson response = %+v", response.Data)
	}
}

func TestCompletePipelineLessonPropagatesServiceError(t *testing.T) {
	handler := &Handler{Lms: &writeHandlerLMSClientStub{
		completeErr: status.Error(codes.FailedPrecondition, "lesson cannot be completed"),
	}}
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/pipeline/lessons/lesson-1/complete",
		"",
		"candidate-1",
		map[string]string{"lessonId": "lesson-1"},
	)
	recorder := httptest.NewRecorder()

	handler.CompletePipelineLesson(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusConflict, ErrPrecondition)
}
