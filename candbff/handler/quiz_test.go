package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"google.golang.org/grpc"
)

type quizRegressionClient struct {
	lmspb.LmsServiceClient

	takeRequest     *lmspb.TakeQuizRequest
	paperRequest    *lmspb.GetCandidateQuizPaperRequest
	submitRequest   *lmspb.SubmitQuizRequest
	draftRequest    *lmspb.SaveQuizDraftRequest
	attemptRequest  *lmspb.GetQuizAttemptDetailCandidateRequest
	completeRequest *lmspb.CompleteQuizRequest
}

func (c *quizRegressionClient) TakeQuiz(
	_ context.Context,
	request *lmspb.TakeQuizRequest,
	_ ...grpc.CallOption,
) (*lmspb.TakeQuizResponse, error) {
	c.takeRequest = request
	return &lmspb.TakeQuizResponse{AttemptId: "attempt-1", Status: "in_progress"}, nil
}

func (c *quizRegressionClient) GetCandidateQuizPaper(
	_ context.Context,
	request *lmspb.GetCandidateQuizPaperRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetCandidateQuizPaperResponse, error) {
	c.paperRequest = request
	return &lmspb.GetCandidateQuizPaperResponse{}, nil
}

func (c *quizRegressionClient) SubmitQuiz(
	_ context.Context,
	request *lmspb.SubmitQuizRequest,
	_ ...grpc.CallOption,
) (*lmspb.SubmitQuizResponse, error) {
	c.submitRequest = request
	return &lmspb.SubmitQuizResponse{AttemptId: request.GetAttemptId(), Status: "completed"}, nil
}

func (c *quizRegressionClient) SaveQuizDraft(
	_ context.Context,
	request *lmspb.SaveQuizDraftRequest,
	_ ...grpc.CallOption,
) (*lmspb.SaveQuizDraftResponse, error) {
	c.draftRequest = request
	return &lmspb.SaveQuizDraftResponse{AttemptId: request.GetAttemptId()}, nil
}

func (c *quizRegressionClient) GetQuizAttemptDetailCandidate(
	_ context.Context,
	request *lmspb.GetQuizAttemptDetailCandidateRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetQuizAttemptDetailResponse, error) {
	c.attemptRequest = request
	return &lmspb.GetQuizAttemptDetailResponse{}, nil
}

func (c *quizRegressionClient) CompleteQuiz(
	_ context.Context,
	request *lmspb.CompleteQuizRequest,
	_ ...grpc.CallOption,
) (*lmspb.CompleteQuizResponse, error) {
	c.completeRequest = request
	return &lmspb.CompleteQuizResponse{QuizStatus: "completed"}, nil
}

func TestQuizHandlersKeepCandidateScopeAndForwardAnswers(t *testing.T) {
	client := &quizRegressionClient{}
	handler := &Handler{Lms: client}

	takeRecorder := httptest.NewRecorder()
	handler.TakeQuiz(
		takeRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/quizzes/quiz-1/take",
			"",
			"candidate-1",
			map[string]string{"quizId": "quiz-1"},
		),
	)
	if takeRecorder.Code != http.StatusOK {
		t.Fatalf("take status = %d; body=%q", takeRecorder.Code, takeRecorder.Body.String())
	}
	if client.takeRequest.GetCandidateUlid() != "candidate-1" ||
		client.takeRequest.GetQuizUlid() != "quiz-1" {
		t.Fatalf("take request = %#v", client.takeRequest)
	}

	paperRecorder := httptest.NewRecorder()
	handler.GetQuizPaper(
		paperRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/quizzes/attempts/attempt-1/paper",
			"",
			"candidate-1",
			map[string]string{"attemptId": "attempt-1"},
		),
	)
	if paperRecorder.Code != http.StatusOK {
		t.Fatalf("paper status = %d; body=%q", paperRecorder.Code, paperRecorder.Body.String())
	}
	if client.paperRequest.GetCandidateUlid() != "candidate-1" ||
		client.paperRequest.GetAttemptId() != "attempt-1" {
		t.Fatalf("paper request = %#v", client.paperRequest)
	}

	body := `{"submissions":[{"question_id":"question-1","selected_option_ids":["option-1","option-2"]}]}`
	submitRecorder := httptest.NewRecorder()
	handler.SubmitQuiz(
		submitRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/quizzes/attempts/attempt-1/submit",
			body,
			"candidate-1",
			map[string]string{"attemptId": "attempt-1"},
		),
	)
	if submitRecorder.Code != http.StatusOK {
		t.Fatalf("submit status = %d; body=%q", submitRecorder.Code, submitRecorder.Body.String())
	}
	assertQuizSubmissionRequest(t, client.submitRequest.GetCandidateUlid(), client.submitRequest.GetAttemptId(), client.submitRequest.GetSubmissions())

	draftRecorder := httptest.NewRecorder()
	handler.SaveQuizDraft(
		draftRecorder,
		newCandidateHandlerRequest(
			http.MethodPut,
			"/api/quizzes/attempts/attempt-1/draft",
			body,
			"candidate-1",
			map[string]string{"attemptId": "attempt-1"},
		),
	)
	if draftRecorder.Code != http.StatusOK {
		t.Fatalf("draft status = %d; body=%q", draftRecorder.Code, draftRecorder.Body.String())
	}
	assertQuizSubmissionRequest(t, client.draftRequest.GetCandidateUlid(), client.draftRequest.GetAttemptId(), client.draftRequest.GetSubmissions())

	attemptRecorder := httptest.NewRecorder()
	handler.GetQuizAttemptDetail(
		attemptRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/quizzes/attempts/attempt-1",
			"",
			"candidate-1",
			map[string]string{"attemptId": "attempt-1"},
		),
	)
	if attemptRecorder.Code != http.StatusOK {
		t.Fatalf("attempt status = %d; body=%q", attemptRecorder.Code, attemptRecorder.Body.String())
	}
	if client.attemptRequest.GetCandidateUlid() != "candidate-1" ||
		client.attemptRequest.GetAttemptId() != "attempt-1" {
		t.Fatalf("attempt request = %#v", client.attemptRequest)
	}

	completeRecorder := httptest.NewRecorder()
	handler.CompleteLmsQuiz(
		completeRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/quizzes/quiz-1/complete",
			"",
			"candidate-1",
			map[string]string{"quizId": "quiz-1"},
		),
	)
	if completeRecorder.Code != http.StatusOK {
		t.Fatalf("complete status = %d; body=%q", completeRecorder.Code, completeRecorder.Body.String())
	}
	if client.completeRequest.GetCandidateUlid() != "candidate-1" ||
		client.completeRequest.GetQuizUlid() != "quiz-1" {
		t.Fatalf("complete request = %#v", client.completeRequest)
	}
}

func assertQuizSubmissionRequest(
	t *testing.T,
	candidateID string,
	attemptID string,
	submissions []*lmspb.QuizAnswerSubmission,
) {
	t.Helper()
	if candidateID != "candidate-1" || attemptID != "attempt-1" {
		t.Fatalf("submission scope = (%q, %q)", candidateID, attemptID)
	}
	if len(submissions) != 1 ||
		submissions[0].GetQuestionUlid() != "question-1" ||
		!reflect.DeepEqual(submissions[0].GetSelectedOptionIds(), []string{"option-1", "option-2"}) {
		t.Fatalf("submissions = %#v", submissions)
	}
}

func TestQuizHandlersRejectMissingIDsAndMalformedBodies(t *testing.T) {
	handler := &Handler{}

	takeRecorder := httptest.NewRecorder()
	handler.TakeQuiz(
		takeRecorder,
		newCandidateHandlerRequest(http.MethodPost, "/api/quizzes//take", "", "candidate-1", nil),
	)
	assertHandlerAPIError(t, takeRecorder, http.StatusBadRequest, ErrInvalidRequest)

	submitRecorder := httptest.NewRecorder()
	handler.SubmitQuiz(
		submitRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/quizzes/attempts/attempt-1/submit",
			`{"submissions":`,
			"candidate-1",
			map[string]string{"attemptId": "attempt-1"},
		),
	)
	assertHandlerAPIError(t, submitRecorder, http.StatusBadRequest, ErrInvalidRequest)
}
