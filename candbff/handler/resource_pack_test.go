package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type resourcePackRegressionClient struct {
	lmspb.LmsServiceClient
	mu sync.Mutex

	packsRequest    *lmspb.ListResourcePacksCandidateRequest
	filesRequests   []*lmspb.ListResourcePackFilesCandidateRequest
	viewURLRequest  *lmspb.GetResourcePackFileViewURLRequest
	createRequests  []*lmspb.CreateViewURLCandidateRequest
	packsResponse   *lmspb.ListResourcePacksResponse
	filesByPack     map[string]*lmspb.ListResourcePackFilesResponse
	viewURLResponse *lmspb.GetResourcePackFileViewURLResponse
	viewURLError    error
}

func (c *resourcePackRegressionClient) ListResourcePacks(
	_ context.Context,
	request *lmspb.ListResourcePacksCandidateRequest,
	_ ...grpc.CallOption,
) (*lmspb.ListResourcePacksResponse, error) {
	c.packsRequest = request
	if c.packsResponse != nil {
		return c.packsResponse, nil
	}
	return &lmspb.ListResourcePacksResponse{}, nil
}

func (c *resourcePackRegressionClient) ListResourcePackFiles(
	_ context.Context,
	request *lmspb.ListResourcePackFilesCandidateRequest,
	_ ...grpc.CallOption,
) (*lmspb.ListResourcePackFilesResponse, error) {
	c.filesRequests = append(c.filesRequests, request)
	if response := c.filesByPack[request.GetFilters().GetPackId()]; response != nil {
		return response, nil
	}
	return &lmspb.ListResourcePackFilesResponse{}, nil
}

func (c *resourcePackRegressionClient) CreateViewURL(
	_ context.Context,
	request *lmspb.CreateViewURLCandidateRequest,
	_ ...grpc.CallOption,
) (*lmspb.CreateViewURLResponse, error) {
	c.mu.Lock()
	c.createRequests = append(c.createRequests, request)
	c.mu.Unlock()
	return &lmspb.CreateViewURLResponse{
		ObjectKey: request.GetObjectKey(),
		ViewUrl:   "https://files.example.test/" + request.GetObjectKey(),
		ExpiresAt: "2026-08-05T12:00:00Z",
	}, nil
}

func (c *resourcePackRegressionClient) GetResourcePackFileViewURL(
	_ context.Context,
	request *lmspb.GetResourcePackFileViewURLRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetResourcePackFileViewURLResponse, error) {
	c.viewURLRequest = request
	if c.viewURLError != nil {
		return nil, c.viewURLError
	}
	if c.viewURLResponse != nil {
		return c.viewURLResponse, nil
	}
	return &lmspb.GetResourcePackFileViewURLResponse{}, nil
}

func (c *resourcePackRegressionClient) GetResourcePackTranslations(context.Context, *lmspb.GetResourcePackTranslationsRequest, ...grpc.CallOption) (*lmspb.GetResourcePackTranslationsResponse, error) {
	return &lmspb.GetResourcePackTranslationsResponse{Translations: map[string]*lmspb.ResourcePackTranslation{
		"zh-CN": {Title: "本地化合集"},
	}}, nil
}

func (c *resourcePackRegressionClient) GetResourcePackFileTranslations(context.Context, *lmspb.GetResourcePackFileTranslationsRequest, ...grpc.CallOption) (*lmspb.GetResourcePackFileTranslationsResponse, error) {
	return &lmspb.GetResourcePackFileTranslationsResponse{Translations: map[string]*lmspb.ResourcePackFileTranslation{
		"zh-CN": {Title: "本地化资源"},
	}}, nil
}

func TestResourceCatalogPreservesAccessFlagsAndLockedThumbnails(t *testing.T) {
	for _, access := range []struct {
		name  string
		value *bool
	}{{"locked", proto.Bool(false)}, {"authorized", proto.Bool(true)}, {"unspecified", nil}} {
		t.Run(access.name, func(t *testing.T) {
			// Decode the wire response to verify the upgraded protocol retains optional false.
			wire, err := proto.Marshal(&lmspb.ListResourcePacksResponse{Packs: []*lmspb.ResourcePack{{PackId: "pack-1", CanAccess: access.value}}})
			if err != nil {
				t.Fatal(err)
			}
			packs := &lmspb.ListResourcePacksResponse{}
			if err := proto.Unmarshal(wire, packs); err != nil {
				t.Fatal(err)
			}
			client := &resourcePackRegressionClient{
				packsResponse: packs,
				filesByPack: map[string]*lmspb.ListResourcePackFilesResponse{
					"pack-1": {Files: []*lmspb.ResourcePackFile{{FileId: "file-1", PackId: "pack-1", ThumbnailObjectKey: "thumb.png", CanAccess: access.value}}},
				},
			}
			handler := &Handler{Lms: client}
			for _, endpoint := range []struct {
				path   string
				key    string
				title  string
				handle http.HandlerFunc
			}{{"/api/resource-packs", "packs", "本地化合集", handler.ListResourcePacks}, {"/api/resource-packs/pack-1/files", "files", "本地化资源", handler.ListResourcePackFiles}} {
				recorder := httptest.NewRecorder()
				request := newCandidateHandlerRequest(http.MethodGet, endpoint.path, "", "candidate-1", map[string]string{"pack_id": "pack-1"})
				request.Header.Set("Accept-Language", "zh-CN")
				endpoint.handle(recorder, request)
				if recorder.Code != http.StatusOK {
					t.Fatalf("status = %d; body=%s", recorder.Code, recorder.Body)
				}
				var response struct {
					Data map[string]json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				var items []map[string]any
				if err := json.Unmarshal(response.Data[endpoint.key], &items); err != nil {
					t.Fatal(err)
				}
				if len(items) != 1 {
					t.Fatalf("items = %#v", items)
				}
				value, present := items[0]["can_access"]
				if (access.value == nil && present) || (access.value != nil && (!present || value != *access.value)) {
					t.Fatalf("can_access = %#v, present = %v", value, present)
				}
				if items[0]["title"] != endpoint.title {
					t.Fatalf("translation missing: %#v", items[0])
				}
				if endpoint.key == "files" && items[0]["thumbnail_url"] != "https://files.example.test/thumb.png" {
					t.Fatalf("thumbnail missing: %#v", items[0])
				}
			}
			if client.viewURLRequest != nil {
				t.Fatal("catalog must not request full content URLs")
			}
		})
	}
}

func TestEmptyResourceCatalogReturnsArrays(t *testing.T) {
	handler := &Handler{Lms: &resourcePackRegressionClient{}}
	for _, endpoint := range []struct {
		key    string
		handle http.HandlerFunc
	}{{"packs", handler.ListResourcePacks}, {"files", handler.ListResourcePackFiles}} {
		recorder := httptest.NewRecorder()
		endpoint.handle(recorder, newCandidateHandlerRequest(http.MethodGet, "/api/resource-packs/pack-1/files", "", "candidate-1", map[string]string{"pack_id": "pack-1"}))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d; body=%s", recorder.Code, recorder.Body)
		}
		var response struct {
			Data map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if string(response.Data[endpoint.key]) != "[]" {
			t.Fatalf("empty %s = %s", endpoint.key, response.Data[endpoint.key])
		}
	}
}

func TestResourceContentHandlersPreservePermissionDenied(t *testing.T) {
	client := &resourcePackRegressionClient{viewURLError: status.Error(codes.PermissionDenied, "resource access denied")}
	handler := &Handler{Lms: client}
	for _, handle := range []http.HandlerFunc{handler.GetResourcePackFileViewURL, handler.GetResourcePackFilePreviewURL} {
		recorder := httptest.NewRecorder()
		handle(recorder, newCandidateHandlerRequest(http.MethodGet, "/api/resource-pack-files/file-1/view-url", "", "candidate-1", map[string]string{"file_id": "file-1"}))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d; body=%s", recorder.Code, recorder.Body)
		}
		if client.viewURLRequest.GetCandidateUlid() != "candidate-1" || client.viewURLRequest.GetFileId() != "file-1" {
			t.Fatalf("content request lost its scope: %#v", client.viewURLRequest)
		}
	}
}

func TestListResourcePacksForwardsCandidatePagination(t *testing.T) {
	client := &resourcePackRegressionClient{
		packsResponse: &lmspb.ListResourcePacksResponse{
			Packs:      []*lmspb.ResourcePack{{PackId: "pack-1", Title: "Pack 1"}},
			NextCursor: "next-pack",
			HasMore:    true,
		},
	}
	handler := &Handler{Lms: client}
	recorder := httptest.NewRecorder()

	handler.ListResourcePacks(
		recorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/resource-packs?page_size=15&page_token=pack-cursor",
			"",
			"candidate-1",
			nil,
		),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.packsRequest.GetFilters().GetCandidateUlid() != "candidate-1" ||
		client.packsRequest.GetPageSize() != 15 ||
		client.packsRequest.GetCursor() != "pack-cursor" {
		t.Fatalf("packs request = %#v", client.packsRequest)
	}
}

func TestListResourcePackFilesKeepsPackScopeAndEnrichesThumbnail(t *testing.T) {
	client := &resourcePackRegressionClient{
		filesByPack: map[string]*lmspb.ListResourcePackFilesResponse{
			"pack-1": {
				Files: []*lmspb.ResourcePackFile{
					{
						FileId:             "file-1",
						PackId:             "pack-1",
						Title:              "Guide",
						ThumbnailObjectKey: "thumbnails/guide.png",
					},
				},
				NextCursor: "next-file",
				HasMore:    true,
			},
		},
	}
	handler := &Handler{Lms: client}
	recorder := httptest.NewRecorder()

	handler.ListResourcePackFiles(
		recorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/resource-packs/pack-1/files?page_size=8&cursor=file-cursor",
			"",
			"candidate-1",
			map[string]string{"pack_id": "pack-1"},
		),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if len(client.filesRequests) != 1 {
		t.Fatalf("ListResourcePackFiles calls = %d, want 1", len(client.filesRequests))
	}
	filesRequest := client.filesRequests[0]
	if filesRequest.GetFilters().GetCandidateUlid() != "candidate-1" ||
		filesRequest.GetFilters().GetPackId() != "pack-1" ||
		filesRequest.GetPageSize() != 8 ||
		filesRequest.GetCursor() != "file-cursor" {
		t.Fatalf("files request = %#v", filesRequest)
	}
	if len(client.createRequests) != 1 ||
		client.createRequests[0].GetCandidateUlid() != "candidate-1" ||
		client.createRequests[0].GetObjectKey() != "thumbnails/guide.png" {
		t.Fatalf("thumbnail requests = %#v", client.createRequests)
	}

	var response struct {
		Data struct {
			Files []struct {
				FileID       string `json:"file_id"`
				ThumbnailURL string `json:"thumbnail_url"`
			} `json:"files"`
			NextCursor string `json:"next_cursor"`
			HasMore    bool   `json:"has_more"`
		} `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data.Files) != 1 ||
		response.Data.Files[0].FileID != "file-1" ||
		response.Data.Files[0].ThumbnailURL != "https://files.example.test/thumbnails/guide.png" {
		t.Fatalf("files response = %#v", response.Data.Files)
	}
	if response.Data.NextCursor != "next-file" || !response.Data.HasMore {
		t.Fatalf("pagination response = %#v", response.Data)
	}
}

func TestResourcePackViewURLHandlersKeepCandidateAndFileScope(t *testing.T) {
	client := &resourcePackRegressionClient{
		packsResponse: &lmspb.ListResourcePacksResponse{
			Packs: []*lmspb.ResourcePack{{PackId: "pack-1"}},
		},
		filesByPack: map[string]*lmspb.ListResourcePackFilesResponse{
			"pack-1": {
				Files: []*lmspb.ResourcePackFile{
					{FileId: "file-1", PackId: "pack-1", ThumbnailObjectKey: "thumb/file-1.png"},
				},
			},
		},
		viewURLResponse: &lmspb.GetResourcePackFileViewURLResponse{
			ViewUrl:   "https://files.example.test/file-1",
			ExpiresAt: "2026-08-05T12:00:00Z",
		},
	}
	handler := &Handler{Lms: client}

	thumbnailRecorder := httptest.NewRecorder()
	handler.GetResourcePackFileThumbnailURL(
		thumbnailRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/resource-pack-files/file-1/thumbnail-url",
			"",
			"candidate-1",
			map[string]string{"file_id": "file-1"},
		),
	)
	if thumbnailRecorder.Code != http.StatusOK {
		t.Fatalf("thumbnail status = %d; body=%q", thumbnailRecorder.Code, thumbnailRecorder.Body.String())
	}
	if client.packsRequest.GetFilters().GetCandidateUlid() != "candidate-1" {
		t.Fatalf("pack lookup request = %#v", client.packsRequest)
	}
	if len(client.filesRequests) != 1 ||
		client.filesRequests[0].GetFilters().GetCandidateUlid() != "candidate-1" ||
		client.filesRequests[0].GetFilters().GetPackId() != "pack-1" {
		t.Fatalf("file lookup requests = %#v", client.filesRequests)
	}
	if len(client.createRequests) != 1 ||
		client.createRequests[0].GetCandidateUlid() != "candidate-1" ||
		client.createRequests[0].GetObjectKey() != "thumb/file-1.png" {
		t.Fatalf("thumbnail URL request = %#v", client.createRequests)
	}

	previewRecorder := httptest.NewRecorder()
	handler.GetResourcePackFilePreviewURL(
		previewRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/resource-pack-files/file-1/preview-url",
			"",
			"candidate-1",
			map[string]string{"file_id": "file-1"},
		),
	)
	if previewRecorder.Code != http.StatusOK {
		t.Fatalf("preview status = %d; body=%q", previewRecorder.Code, previewRecorder.Body.String())
	}
	if client.viewURLRequest.GetCandidateUlid() != "candidate-1" ||
		client.viewURLRequest.GetFileId() != "file-1" {
		t.Fatalf("preview request = %#v", client.viewURLRequest)
	}
}

func TestResourcePackHandlersRejectMissingIDsAndEmptyViewURL(t *testing.T) {
	handler := &Handler{Lms: &resourcePackRegressionClient{}}

	filesRecorder := httptest.NewRecorder()
	handler.ListResourcePackFiles(
		filesRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/resource-packs//files",
			"",
			"candidate-1",
			nil,
		),
	)
	assertHandlerAPIError(t, filesRecorder, http.StatusBadRequest, ErrInvalidRequest)

	previewRecorder := httptest.NewRecorder()
	handler.GetResourcePackFilePreviewURL(
		previewRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/resource-pack-files/file-1/preview-url",
			"",
			"candidate-1",
			map[string]string{"file_id": "file-1"},
		),
	)
	assertHandlerAPIError(t, previewRecorder, http.StatusBadGateway, ErrServiceUnavailable)
}
