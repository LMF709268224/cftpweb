package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"google.golang.org/grpc"
)

type credentialClientStub struct {
	gcredspb.CredentialServiceClient
	listCandidateApplicationsResponse *gcredspb.ListApplicationsResponse
	listCandidateApplicationsRequest  *gcredspb.ListApplicationsRequest
	applicationDetailResponse         *gcredspb.Application
	applicationDetailRequest          *gcredspb.GetApplicationDetailRequest
	definitionDetailResponse          *gcredspb.CredentialDefinition
	definitionTranslationResponse     *gcredspb.GetCredDefTranslationsResponse
}

type credentialApplicationOrderMallStub struct {
	mallpb.MallServiceClient
	listRequests    []*mallpb.ListCredentialApplicationOrdersRequest
	detailRequests  []*mallpb.GetCredentialApplicationOrderDetailRequest
	listResponses   map[string]*mallpb.ListCredentialApplicationOrdersResponse
	detailResponses map[string]*mallpb.GetCredentialApplicationOrderDetailResponse
}

func (s *credentialApplicationOrderMallStub) ListCredentialApplicationOrders(
	_ context.Context,
	req *mallpb.ListCredentialApplicationOrdersRequest,
	_ ...grpc.CallOption,
) (*mallpb.ListCredentialApplicationOrdersResponse, error) {
	s.listRequests = append(s.listRequests, req)
	if response := s.listResponses[req.GetCursor()]; response != nil {
		return response, nil
	}
	return &mallpb.ListCredentialApplicationOrdersResponse{}, nil
}

func (s *credentialApplicationOrderMallStub) GetCredentialApplicationOrderDetail(
	_ context.Context,
	req *mallpb.GetCredentialApplicationOrderDetailRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetCredentialApplicationOrderDetailResponse, error) {
	s.detailRequests = append(s.detailRequests, req)
	return s.detailResponses[req.GetApplicationOrderUlid()], nil
}

func (s *credentialClientStub) ListCandidateApplications(
	_ context.Context,
	req *gcredspb.ListApplicationsRequest,
	_ ...grpc.CallOption,
) (*gcredspb.ListApplicationsResponse, error) {
	s.listCandidateApplicationsRequest = req
	return s.listCandidateApplicationsResponse, nil
}

func (s *credentialClientStub) GetApplicationDetail(
	_ context.Context,
	req *gcredspb.GetApplicationDetailRequest,
	_ ...grpc.CallOption,
) (*gcredspb.Application, error) {
	s.applicationDetailRequest = req
	return s.applicationDetailResponse, nil
}

func (s *credentialClientStub) GetCredentialDefinitionDetail(
	_ context.Context,
	req *gcredspb.GetCredentialDefinitionDetailRequest,
	_ ...grpc.CallOption,
) (*gcredspb.CredentialDefinition, error) {
	if s.definitionDetailResponse != nil {
		return s.definitionDetailResponse, nil
	}
	return &gcredspb.CredentialDefinition{CredDefUlid: req.GetCredDefUlid()}, nil
}

func (s *credentialClientStub) GetCredDefTranslations(
	_ context.Context,
	_ *gcredspb.GetCredDefTranslationsRequest,
	_ ...grpc.CallOption,
) (*gcredspb.GetCredDefTranslationsResponse, error) {
	if s.definitionTranslationResponse != nil {
		return s.definitionTranslationResponse, nil
	}
	return &gcredspb.GetCredDefTranslationsResponse{}, nil
}

func TestListCredentialDefinitionsIncludesReferenceAttachments(t *testing.T) {
	const definitionID = "01J00000000000000000000001"
	client := &credentialClientStub{
		listCandidateApplicationsResponse: &gcredspb.ListApplicationsResponse{},
		definitionDetailResponse: &gcredspb.CredentialDefinition{
			CredDefUlid: definitionID,
			Name:        "Work Experience Qualification",
			FileConstraints: []*gcredspb.CredentialFileConstraint{{
				Name:       "Employment Certificate",
				Type:       gcredspb.CredentialFileType_CREDENTIAL_FILE_TYPE_PDF,
				IsRequired: true,
			}},
			Attachments: []*gcredspb.CredentialAttachment{{
				AttachmentId: "attachment-1",
				Name:         "Work Experience Template",
				FileName:     "work-experience.docx",
				FileType:     gcredspb.CredentialFileType_CREDENTIAL_FILE_TYPE_TEXT,
				FileExt:      "docx",
				FileSize:     4096,
				DownloadUrl:  "https://downloads.example/work-experience.docx",
			}},
		},
		definitionTranslationResponse: &gcredspb.GetCredDefTranslationsResponse{
			Translations: map[string]*gcredspb.CredDefTranslation{
				"zh-CN": {
					Name: "工作经验证明",
					FileConstraintNames: map[string]string{
						"employment_certificate": "雇佣证明",
					},
				},
			},
		},
	}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/credentials/definitions?qual_ulids="+definitionID,
		"",
		"01J00000000000000000000000",
		nil,
	)
	request.Header.Set("Accept-Language", "zh-CN")

	(&Handler{Creds: client}).ListCredentialDefinitions(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			Definitions []struct {
				Name            string `json:"name"`
				FileConstraints []struct {
					Name        string `json:"name"`
					DisplayName string `json:"display_name"`
				} `json:"file_constraints"`
				Attachments []struct {
					Name        string `json:"name"`
					DownloadURL string `json:"download_url"`
				} `json:"attachments"`
			} `json:"definitions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Definitions) != 1 || len(payload.Data.Definitions[0].Attachments) != 1 {
		t.Fatalf("definitions = %#v", payload.Data.Definitions)
	}
	attachment := payload.Data.Definitions[0].Attachments[0]
	if attachment.Name != "Work Experience Template" || attachment.DownloadURL != "https://downloads.example/work-experience.docx" {
		t.Fatalf("attachment = %#v", attachment)
	}
	definition := payload.Data.Definitions[0]
	if definition.Name != "工作经验证明" || len(definition.FileConstraints) != 1 {
		t.Fatalf("definition = %#v", definition)
	}
	constraint := definition.FileConstraints[0]
	if constraint.Name != "Employment Certificate" || constraint.DisplayName != "雇佣证明" {
		t.Fatalf("file constraint = %#v", constraint)
	}
}

func TestGetCandidateApplicationReturnsOwnedUploadedFiles(t *testing.T) {
	const (
		candidateID   = "01J00000000000000000000000"
		applicationID = "01J00000000000000000000002"
	)
	client := &credentialClientStub{applicationDetailResponse: &gcredspb.Application{
		AppUlid:       applicationID,
		CandidateUlid: candidateID,
		CredDefUlid:   "01J00000000000000000000001",
		Status:        "APPLICATION_STATUS_APPROVED",
		Files: []*gcredspb.FileInfo{{
			FileName:  "employment-proof.pdf",
			FileExt:   ".pdf",
			FileSize:  4096,
			FileUsage: "employment_certificate",
			ViewUrl:   "https://files.example/employment-proof.pdf?signature=temporary",
		}},
	}}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/credentials/applications/"+applicationID,
		"",
		candidateID,
		map[string]string{"appId": applicationID},
	)

	(&Handler{Creds: client}).GetCandidateApplication(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.applicationDetailRequest == nil || client.applicationDetailRequest.GetAppUlid() != applicationID {
		t.Fatalf("detail request = %#v", client.applicationDetailRequest)
	}
	var payload struct {
		Data struct {
			Files []struct {
				FileName string `json:"file_name"`
				ViewURL  string `json:"view_url"`
			} `json:"files"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Files) != 1 || payload.Data.Files[0].FileName != "employment-proof.pdf" || payload.Data.Files[0].ViewURL == "" {
		t.Fatalf("files = %#v", payload.Data.Files)
	}
}

func TestGetCandidateApplicationHidesAnotherCandidatesFiles(t *testing.T) {
	const applicationID = "01J00000000000000000000002"
	client := &credentialClientStub{applicationDetailResponse: &gcredspb.Application{
		AppUlid:       applicationID,
		CandidateUlid: "01J00000000000000000000009",
		Files: []*gcredspb.FileInfo{{
			FileName: "private.pdf",
			ViewUrl:  "https://files.example/private.pdf?signature=temporary",
		}},
	}}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/credentials/applications/"+applicationID,
		"",
		"01J00000000000000000000000",
		map[string]string{"appId": applicationID},
	)

	(&Handler{Creds: client}).GetCandidateApplication(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusNotFound, ErrNotFound)
}

func TestListCredentialDefinitionsUsesActionableApplications(t *testing.T) {
	const (
		candidateID   = "01J00000000000000000000000"
		definitionID  = "01J00000000000000000000001"
		applicationID = "01J00000000000000000000002"
	)
	client := &credentialClientStub{
		listCandidateApplicationsResponse: &gcredspb.ListApplicationsResponse{
			Applications: []*gcredspb.ApplicationSummary{{
				AppUlid:       applicationID,
				CandidateUlid: candidateID,
				CredDefUlid:   definitionID,
				Status:        "APPLICATION_STATUS_PENDING_UPLOAD",
			}},
		},
		definitionDetailResponse: &gcredspb.CredentialDefinition{
			CredDefUlid: definitionID,
			Name:        "Pending upload qualification",
		},
	}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/credentials/definitions",
		"",
		candidateID,
		nil,
	)

	(&Handler{Creds: client}).ListCredentialDefinitions(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			Definitions []struct {
				CredDefULID       string                 `json:"cred_def_ulid"`
				LatestApplication map[string]interface{} `json:"latest_application"`
			} `json:"definitions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Definitions) != 1 {
		t.Fatalf("definitions = %#v, want one actionable definition", payload.Data.Definitions)
	}
	definition := payload.Data.Definitions[0]
	if definition.CredDefULID != definitionID || definition.LatestApplication["app_ulid"] != applicationID {
		t.Fatalf("definition = %#v", definition)
	}
	if definition.LatestApplication["status"] != "APPLICATION_STATUS_PENDING_UPLOAD" {
		t.Fatalf("latest application status = %v", definition.LatestApplication["status"])
	}

	req := client.listCandidateApplicationsRequest
	if req == nil {
		t.Fatal("ListCandidateApplications was not called")
	}
	if req.GetFilters().GetCandidateUlid() != candidateID {
		t.Fatalf("candidate_ulid = %q, want %q", req.GetFilters().GetCandidateUlid(), candidateID)
	}
	wantStatuses := []string{"PendingUpload", "Reupload"}
	if got := req.GetFilters().GetStatuses(); len(got) != len(wantStatuses) || got[0] != wantStatuses[0] || got[1] != wantStatuses[1] {
		t.Fatalf("statuses = %#v, want %#v", got, wantStatuses)
	}
	if req.GetPageSize() != 100 || req.GetSortOrder() != gcredspb.SortOrder_SORT_ORDER_DESC {
		t.Fatalf("unexpected pagination/sort request: %#v", req)
	}
}

func TestLatestCredentialApplicationUsesCandidateScopedLatestQuery(t *testing.T) {
	const (
		candidateID = "01J00000000000000000000000"
		credDefID   = "01J00000000000000000000001"
		appID       = "01J00000000000000000000002"
	)
	client := &credentialClientStub{
		listCandidateApplicationsResponse: &gcredspb.ListApplicationsResponse{
			Applications: []*gcredspb.ApplicationSummary{
				{
					AppUlid:       appID,
					CandidateUlid: candidateID,
					CredDefUlid:   credDefID,
					Status:        "PENDING",
				},
			},
		},
	}
	h := &Handler{Creds: client}

	got, err := h.latestCredentialApplication(context.Background(), candidateID, credDefID)
	if err != nil {
		t.Fatalf("latestCredentialApplication returned error: %v", err)
	}
	if got["app_ulid"] != appID {
		t.Fatalf("app_ulid = %v, want %q", got["app_ulid"], appID)
	}

	req := client.listCandidateApplicationsRequest
	if req == nil {
		t.Fatal("ListCandidateApplications was not called")
	}
	if req.GetFilters().GetCandidateUlid() != candidateID {
		t.Fatalf("candidate_ulid = %q, want %q", req.GetFilters().GetCandidateUlid(), candidateID)
	}
	if req.GetFilters().GetCredDefUlid() != credDefID {
		t.Fatalf("cred_def_ulid = %q, want %q", req.GetFilters().GetCredDefUlid(), credDefID)
	}
	if req.GetPageSize() != 1 {
		t.Fatalf("page_size = %d, want 1", req.GetPageSize())
	}
	if req.GetSortOrder() != gcredspb.SortOrder_SORT_ORDER_DESC {
		t.Fatalf("sort_order = %v, want SORT_ORDER_DESC", req.GetSortOrder())
	}
}

func TestLatestCredentialApplicationReturnsNilWhenNoApplicationExists(t *testing.T) {
	client := &credentialClientStub{
		listCandidateApplicationsResponse: &gcredspb.ListApplicationsResponse{},
	}
	h := &Handler{Creds: client}

	got, err := h.latestCredentialApplication(
		context.Background(),
		"01J00000000000000000000000",
		"01J00000000000000000000001",
	)
	if err != nil {
		t.Fatalf("latestCredentialApplication returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("latest application = %#v, want nil", got)
	}
}

func TestListCredentialApplicationOrdersReturnsSelectedQualifications(t *testing.T) {
	const (
		candidateID = "01J00000000000000000000000"
		firstOrder  = "01J00000000000000000000001"
		firstQual   = "01J00000000000000000000002"
		secondOrder = "01J00000000000000000000003"
		secondQual  = "01J00000000000000000000004"
	)
	firstSummary := &mallpb.CredentialApplicationOrderSummary{
		ApplicationOrderUlid: firstOrder,
		CandidateUlid:        candidateID,
		OrderStatus:          "UPLOAD_READY",
		CreatedAt:            "2026-08-29T10:00:00Z",
		PayOrderUlid:         "pay-order-1",
	}
	secondSummary := &mallpb.CredentialApplicationOrderSummary{
		ApplicationOrderUlid: secondOrder,
		CandidateUlid:        candidateID,
		OrderStatus:          "UNDER_REVIEW",
		CreatedAt:            "2026-08-28T10:00:00Z",
		PayOrderUlid:         "pay-order-2",
	}
	client := &credentialApplicationOrderMallStub{
		listResponses: map[string]*mallpb.ListCredentialApplicationOrdersResponse{
			"": {
				Items:      []*mallpb.CredentialApplicationOrderSummary{firstSummary},
				NextCursor: "next-page",
				HasMore:    true,
			},
			"next-page": {Items: []*mallpb.CredentialApplicationOrderSummary{secondSummary}},
		},
		detailResponses: map[string]*mallpb.GetCredentialApplicationOrderDetailResponse{
			firstOrder: {
				Found: true,
				Detail: &mallpb.CredentialApplicationOrderDetail{
					Summary:              firstSummary,
					ApplicationItemsJson: `[{"qual_id":"` + firstQual + `","item_status":"PENDING","qual_name_hint":"Finance exemption"}]`,
				},
			},
			secondOrder: {
				Found: true,
				Detail: &mallpb.CredentialApplicationOrderDetail{
					Summary:              secondSummary,
					ApplicationItemsJson: `[{"qual_id":"` + secondQual + `","item_status":"SUBMITTED","qual_name_hint":"Fintech exemption"}]`,
				},
			},
		},
	}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(http.MethodGet, "/api/credentials/application-orders", "", candidateID, nil)

	(&Handler{Mall: client}).ListCredentialApplicationOrdersForCandidate(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			Orders []struct {
				OrderStatus string `json:"order_status"`
				Items       []struct {
					QualID string `json:"qual_id"`
				} `json:"items"`
			} `json:"orders"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Orders) != 2 ||
		payload.Data.Orders[0].OrderStatus != "UPLOAD_READY" || payload.Data.Orders[0].Items[0].QualID != firstQual ||
		payload.Data.Orders[1].OrderStatus != "UNDER_REVIEW" || payload.Data.Orders[1].Items[0].QualID != secondQual {
		t.Fatalf("data = %#v", payload.Data)
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("list request count = %d, want 2", len(client.listRequests))
	}
	for index, req := range client.listRequests {
		if req.GetFilters().GetCandidateUlid() != candidateID || req.GetFilters().GetOrderStatus() != "" || req.GetPageSize() != 100 || req.GetSortOrder() != mallpb.SortOrder_SORT_ORDER_DESC {
			t.Fatalf("unexpected list request: %#v", req)
		}
		wantCursor := []string{"", "next-page"}[index]
		if req.GetCursor() != wantCursor {
			t.Fatalf("list request cursor = %q, want %q", req.GetCursor(), wantCursor)
		}
	}
	if len(client.detailRequests) != 2 {
		t.Fatalf("detail request count = %d, want 2", len(client.detailRequests))
	}
}

func TestCredentialHandlersRejectInvalidRequestsBeforeCallingServices(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		body   string
		handle func(*Handler, http.ResponseWriter, *http.Request)
	}{
		{
			name:   "qualification IDs are required",
			method: http.MethodGet,
			target: "/api/credentials/qualifications",
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.CheckCandidateQualifications(w, r)
			},
		},
		{
			name:   "application order fields are required",
			method: http.MethodPost,
			target: "/api/credentials/application-orders",
			body:   `{}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.CreateCredentialApplicationOrder(w, r)
			},
		},
		{
			name:   "application order accepts exactly one qualification",
			method: http.MethodPost,
			target: "/api/credentials/application-orders",
			body:   `{"pipeline_cc_ulid":"pipeline-1","bundle_ulid":"bundle-1","qual_ulids":["qualification-1","qualification-2"]}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.CreateCredentialApplicationOrder(w, r)
			},
		},
		{
			name:   "upload fields are required",
			method: http.MethodPost,
			target: "/api/credentials/upload-url",
			body:   `{}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.RequestUploadUrl(w, r)
			},
		},
		{
			name:   "credential definition is required for submission",
			method: http.MethodPost,
			target: "/api/credentials/submit",
			body:   `{}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.SubmitApplication(w, r)
			},
		},
		{
			name:   "all submitted file fields are required",
			method: http.MethodPost,
			target: "/api/credentials/submit",
			body:   `{"cred_def_ulid":"credential-1","files":[{"file_hash":"hash","file_name":"proof.pdf","file_ext":"pdf"}]}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.SubmitApplication(w, r)
			},
		},
		{
			name:   "application ID is required for update",
			method: http.MethodPut,
			target: "/api/credentials/update",
			body:   `{}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.UpdateApplication(w, r)
			},
		},
		{
			name:   "all updated file fields are required",
			method: http.MethodPut,
			target: "/api/credentials/update",
			body:   `{"app_ulid":"application-1","files":[{"file_hash":"hash","file_name":"proof.pdf","file_ext":"pdf"}]}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.UpdateApplication(w, r)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := newCandidateHandlerRequest(test.method, test.target, test.body, "candidate-1", nil)
			recorder := httptest.NewRecorder()

			test.handle(&Handler{}, recorder, request)

			assertHandlerAPIError(t, recorder, http.StatusBadRequest, ErrInvalidRequest)
		})
	}
}
