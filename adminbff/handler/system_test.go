package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	gmailpb "github.com/afnandelfin620-star/cftptest/cftp/gmail"
	"google.golang.org/grpc"
)

type systemCredentialClientStub struct {
	gcredspb.CredentialServiceClient
	response *gcredspb.GetApplicationCountResponse
	err      error
}

func (s *systemCredentialClientStub) GetApplicationCount(
	_ context.Context,
	_ *gcredspb.GetApplicationCountRequest,
	_ ...grpc.CallOption,
) (*gcredspb.GetApplicationCountResponse, error) {
	return s.response, s.err
}

type systemMailClientStub struct {
	gmailpb.MailServiceClient
	response *gmailpb.GetMailCountResponse
	err      error
}

type systemExamClientStub struct {
	gexampb.GExamServiceClient
	response *gexampb.GetPendingGradingExamCountResponse
	err      error
}

func (s *systemExamClientStub) GetPendingGradingExamCount(
	_ context.Context,
	_ *gexampb.GetPendingGradingExamCountRequest,
	_ ...grpc.CallOption,
) (*gexampb.GetPendingGradingExamCountResponse, error) {
	return s.response, s.err
}

func (s *systemMailClientStub) GetMailCount(
	_ context.Context,
	_ *gmailpb.GetMailCountRequest,
	_ ...grpc.CallOption,
) (*gmailpb.GetMailCountResponse, error) {
	return s.response, s.err
}

func TestGetSystemRedDotsSeparatesMessageAndMailCounts(t *testing.T) {
	h := &Handler{
		Creds: &systemCredentialClientStub{response: &gcredspb.GetApplicationCountResponse{Count: 3}},
		Gexam: &systemExamClientStub{response: &gexampb.GetPendingGradingExamCountResponse{Count: 7}},
		Gmail: &systemMailClientStub{response: &gmailpb.GetMailCountResponse{Count: 5}},
	}
	recorder := httptest.NewRecorder()

	h.GetSystemRedDots(recorder, httptest.NewRequest(http.MethodGet, "/api/system/reddots", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response struct {
		Data SystemRedDotsRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.Applications != 3 || response.Data.ExamGrading != 7 || response.Data.Mails != 5 || response.Data.Messages != 0 {
		t.Fatalf("red dots = %+v, want applications=3 exam_grading=7 mails=5 messages=0", response.Data)
	}
}

func TestGetSystemRedDotsReportsDownstreamFailure(t *testing.T) {
	tests := []struct {
		name     string
		credsErr error
		examErr  error
		mailErr  error
	}{
		{name: "application count failure", credsErr: errors.New("credentials unavailable")},
		{name: "essay grading count failure", examErr: errors.New("exam unavailable")},
		{name: "mail count failure", mailErr: errors.New("mail unavailable")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := &Handler{
				Creds: &systemCredentialClientStub{
					response: &gcredspb.GetApplicationCountResponse{},
					err:      test.credsErr,
				},
				Gexam: &systemExamClientStub{
					response: &gexampb.GetPendingGradingExamCountResponse{},
					err:      test.examErr,
				},
				Gmail: &systemMailClientStub{
					response: &gmailpb.GetMailCountResponse{},
					err:      test.mailErr,
				},
			}
			recorder := httptest.NewRecorder()

			h.GetSystemRedDots(recorder, httptest.NewRequest(http.MethodGet, "/api/system/reddots", nil))

			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
			}
			var response struct {
				ErrorCode ErrorCode `json:"error_code"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.ErrorCode != ErrServiceUnavailable {
				t.Fatalf("error code = %q, want %q", response.ErrorCode, ErrServiceUnavailable)
			}
		})
	}
}
