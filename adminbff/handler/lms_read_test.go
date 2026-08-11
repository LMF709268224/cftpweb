package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type lmsReadClientStub struct {
	lmspb.LmsServiceClient
	packCountRequest      *lmspb.GetResourcePackCountRequest
	packListRequest       *lmspb.ListResourcePacksRequest
	packDetailRequest     *lmspb.GetResourcePackRequest
	fileCountRequest      *lmspb.GetResourcePackFileCountRequest
	fileListRequest       *lmspb.ListResourcePackFilesRequest
	fileDetailRequest     *lmspb.GetResourcePackFileRequest
	courseListRequest     *lmspb.ListCoursesRequest
	courseDetailRequest   *lmspb.GetCourseDetailRequest
	completeCourseRequest *lmspb.GetCompleteCourseRequest
}

func (s *lmsReadClientStub) GetResourcePackCountAdmin(_ context.Context, req *lmspb.GetResourcePackCountRequest, _ ...grpc.CallOption) (*lmspb.GetResourcePackCountResponse, error) {
	s.packCountRequest = req
	return &lmspb.GetResourcePackCountResponse{Count: 1}, nil
}

func (s *lmsReadClientStub) ListResourcePacksAdmin(_ context.Context, req *lmspb.ListResourcePacksRequest, _ ...grpc.CallOption) (*lmspb.ListResourcePacksResponse, error) {
	s.packListRequest = req
	return &lmspb.ListResourcePacksResponse{
		Packs: []*lmspb.ResourcePack{{
			PackId:      "pack-1",
			Title:       "Regression Resources",
			Description: "Read-only resource pack",
			Status:      "Active",
			Version:     3,
		}},
		HasMore:    true,
		NextCursor: "next-pack-page",
	}, nil
}

func (s *lmsReadClientStub) GetResourcePackAdmin(_ context.Context, req *lmspb.GetResourcePackRequest, _ ...grpc.CallOption) (*lmspb.ResourcePack, error) {
	s.packDetailRequest = req
	return &lmspb.ResourcePack{
		PackId:             "pack-1",
		Title:              "Regression Resources",
		Description:        "Read-only resource pack detail",
		ThumbnailObjectKey: "resources/regression.png",
		Respath:            "/resources/regression",
		Status:             "Active",
		Version:            3,
	}, nil
}

func (s *lmsReadClientStub) GetResourcePackFileCountAdmin(_ context.Context, req *lmspb.GetResourcePackFileCountRequest, _ ...grpc.CallOption) (*lmspb.GetResourcePackFileCountResponse, error) {
	s.fileCountRequest = req
	return &lmspb.GetResourcePackFileCountResponse{Count: 1}, nil
}

func (s *lmsReadClientStub) ListResourcePackFilesAdmin(_ context.Context, req *lmspb.ListResourcePackFilesRequest, _ ...grpc.CallOption) (*lmspb.ListResourcePackFilesResponse, error) {
	s.fileListRequest = req
	return &lmspb.ListResourcePackFilesResponse{
		Files: []*lmspb.ResourcePackFile{{
			FileId:        "file-1",
			PackId:        "pack-1",
			Title:         "Regression Guide",
			Description:   "Read-only PDF guide",
			FileType:      lmspb.ResourcePackFileType_RESOURCE_PACK_FILE_TYPE_PDF,
			FileName:      "regression.pdf",
			FileObjectKey: "resources/regression.pdf",
			Version:       2,
		}},
		HasMore:    true,
		NextCursor: "next-file-page",
	}, nil
}

func (s *lmsReadClientStub) GetResourcePackFileAdmin(_ context.Context, req *lmspb.GetResourcePackFileRequest, _ ...grpc.CallOption) (*lmspb.ResourcePackFile, error) {
	s.fileDetailRequest = req
	return &lmspb.ResourcePackFile{
		FileId:        "file-1",
		PackId:        "pack-1",
		Title:         "Regression Guide",
		Description:   "Read-only PDF guide detail",
		FileType:      lmspb.ResourcePackFileType_RESOURCE_PACK_FILE_TYPE_PDF,
		FileName:      "regression.pdf",
		FileSize:      2048,
		FileHash:      "sha256-regression",
		FileObjectKey: "resources/regression.pdf",
		Version:       2,
	}, nil
}

func (s *lmsReadClientStub) ListCoursesAdmin(_ context.Context, req *lmspb.ListCoursesRequest, _ ...grpc.CallOption) (*lmspb.ListCoursesResponse, error) {
	s.courseListRequest = req
	return &lmspb.ListCoursesResponse{
		Courses: []*lmspb.CourseSummary{{
			CourseUlid:   "course-1",
			Title:        "Regression Course",
			CategoryTips: "Automation",
			Status:       "Active",
			IsPublished:  true,
			Version:      4,
		}},
		HasMore:    true,
		NextCursor: "next-course-page",
	}, nil
}

func (s *lmsReadClientStub) GetCourseDetailAdmin(_ context.Context, req *lmspb.GetCourseDetailRequest, _ ...grpc.CallOption) (*lmspb.GetCourseDetailResponse, error) {
	s.courseDetailRequest = req
	return &lmspb.GetCourseDetailResponse{CourseDetail: &lmspb.CourseDetail{
		Course:        &lmspb.Course{CourseUlid: "course-1", Title: "Regression Course", Description: "Read-only LMS course"},
		ChapterCount:  2,
		LessonCount:   3,
		QuizCount:     1,
		MaterialCount: 1,
	}}, nil
}

func (s *lmsReadClientStub) GetCompleteCourseAdmin(_ context.Context, req *lmspb.GetCompleteCourseRequest, _ ...grpc.CallOption) (*lmspb.GetCompleteCourseResponse, error) {
	s.completeCourseRequest = req
	return &lmspb.GetCompleteCourseResponse{CompleteCourse: &lmspb.CompleteCourse{
		Course:    &lmspb.Course{CourseUlid: "course-1", Title: "Regression Course"},
		Materials: []*lmspb.CourseMaterial{{MaterialUlid: "material-1", Title: "Regression Material"}},
	}}, nil
}

func requestWithURLParam(method, target, key, value string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(key, value)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func TestListLmsResourcePacksReturnsReadOnlyPage(t *testing.T) {
	client := &lmsReadClientStub{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/lms/resource-packs?status=Active&cursor=current-pack&page_size=10", nil)

	(&Handler{Lms: client}).ListLmsResourcePacks(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.packCountRequest.GetFilters().GetStatus() != "Active" || client.packListRequest.GetFilters().GetStatus() != "Active" || client.packListRequest.GetCursor() != "current-pack" || client.packListRequest.GetPageSize() != 10 {
		t.Fatalf("resource pack requests = %+v / %+v", client.packCountRequest, client.packListRequest)
	}
	var payload struct {
		Data struct {
			Total      uint32 `json:"total"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
			Packs      []struct {
				PackID string `json:"pack_id"`
				Title  string `json:"title"`
			} `json:"packs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Total != 1 || !payload.Data.HasMore || payload.Data.NextCursor != "next-pack-page" || len(payload.Data.Packs) != 1 || payload.Data.Packs[0].PackID != "pack-1" || payload.Data.Packs[0].Title != "Regression Resources" {
		t.Fatalf("resource pack page = %+v", payload.Data)
	}
}

func TestGetLmsResourcePackReturnsReadOnlyDetail(t *testing.T) {
	client := &lmsReadClientStub{}
	recorder := httptest.NewRecorder()

	(&Handler{Lms: client}).GetLmsResourcePack(recorder, requestWithURLParam(http.MethodGet, "/api/lms/resource-packs/pack-1", "pack_id", "pack-1"))

	if recorder.Code != http.StatusOK || client.packDetailRequest.GetPackId() != "pack-1" {
		t.Fatalf("status = %d, request = %+v; body=%s", recorder.Code, client.packDetailRequest, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			PackID  string `json:"pack_id"`
			Title   string `json:"title"`
			Respath string `json:"respath"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.PackID != "pack-1" || payload.Data.Title != "Regression Resources" || payload.Data.Respath != "/resources/regression" {
		t.Fatalf("resource pack detail = %+v", payload.Data)
	}
}

func TestListLmsResourcePackFilesReturnsReadOnlyPage(t *testing.T) {
	client := &lmsReadClientStub{}
	recorder := httptest.NewRecorder()
	request := requestWithURLParam(http.MethodGet, "/api/lms/resource-packs/pack-1/files?cursor=current-file&page_size=10", "pack_id", "pack-1")

	(&Handler{Lms: client}).ListLmsResourcePackFiles(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.fileCountRequest.GetFilters().GetPackId() != "pack-1" || client.fileListRequest.GetFilters().GetPackId() != "pack-1" || client.fileListRequest.GetCursor() != "current-file" || client.fileListRequest.GetPageSize() != 10 {
		t.Fatalf("resource file requests = %+v / %+v", client.fileCountRequest, client.fileListRequest)
	}
	var payload struct {
		Data struct {
			Total      uint32 `json:"total"`
			NextCursor string `json:"next_cursor"`
			Files      []struct {
				FileID string `json:"file_id"`
				Title  string `json:"title"`
			} `json:"files"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Total != 1 || payload.Data.NextCursor != "next-file-page" || len(payload.Data.Files) != 1 || payload.Data.Files[0].FileID != "file-1" || payload.Data.Files[0].Title != "Regression Guide" {
		t.Fatalf("resource file page = %+v", payload.Data)
	}
}

func TestGetLmsResourcePackFileReturnsReadOnlyDetail(t *testing.T) {
	client := &lmsReadClientStub{}
	recorder := httptest.NewRecorder()

	(&Handler{Lms: client}).GetLmsResourcePackFile(recorder, requestWithURLParam(http.MethodGet, "/api/lms/resource-pack-files/file-1", "file_id", "file-1"))

	if recorder.Code != http.StatusOK || client.fileDetailRequest.GetFileId() != "file-1" {
		t.Fatalf("status = %d, request = %+v; body=%s", recorder.Code, client.fileDetailRequest, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			FileID        string `json:"file_id"`
			FileName      string `json:"file_name"`
			FileObjectKey string `json:"file_object_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.FileID != "file-1" || payload.Data.FileName != "regression.pdf" || payload.Data.FileObjectKey != "resources/regression.pdf" {
		t.Fatalf("resource file detail = %+v", payload.Data)
	}
}

func TestListLmsCoursesMapsPublishedFilterForReadOnlyPage(t *testing.T) {
	client := &lmsReadClientStub{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/lms/courses?status=PUBLISHED&category_tips=Automation&cursor=current-course&page_size=10", nil)

	(&Handler{Lms: client}).ListLmsCourses(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	filters := client.courseListRequest.GetFilters()
	if filters.GetStatus() != "Active" || !filters.GetCurrentOnly() || filters.GetCategoryTips() != "Automation" || client.courseListRequest.GetCursor() != "current-course" || client.courseListRequest.GetPageSize() != 10 {
		t.Fatalf("course list request = %+v", client.courseListRequest)
	}
	var payload struct {
		Data struct {
			NextCursor string `json:"next_cursor"`
			Courses    []struct {
				CourseULID string `json:"course_ulid"`
				Title      string `json:"title"`
			} `json:"courses"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.NextCursor != "next-course-page" || len(payload.Data.Courses) != 1 || payload.Data.Courses[0].CourseULID != "course-1" || payload.Data.Courses[0].Title != "Regression Course" {
		t.Fatalf("course page = %+v", payload.Data)
	}
}

func TestGetLmsCourseDetailAndCompleteTreeAreReadOnly(t *testing.T) {
	client := &lmsReadClientStub{}
	h := &Handler{Lms: client}
	detailRecorder := httptest.NewRecorder()
	completeRecorder := httptest.NewRecorder()

	h.GetLmsCourseDetail(detailRecorder, requestWithURLParam(http.MethodGet, "/api/lms/courses/course-1/detail", "course_id", "course-1"))
	h.GetCompleteLmsCourse(completeRecorder, requestWithURLParam(http.MethodGet, "/api/lms/courses/course-1/complete", "course_id", "course-1"))

	if detailRecorder.Code != http.StatusOK || completeRecorder.Code != http.StatusOK || client.courseDetailRequest.GetCourseUlid() != "course-1" || client.completeCourseRequest.GetCourseUlid() != "course-1" {
		t.Fatalf("detail status = %d, complete status = %d, requests = %+v / %+v", detailRecorder.Code, completeRecorder.Code, client.courseDetailRequest, client.completeCourseRequest)
	}
	var detailPayload struct {
		Data struct {
			CourseDetail struct {
				ChapterCount uint32 `json:"chapter_count"`
				LessonCount  uint32 `json:"lesson_count"`
			} `json:"course_detail"`
		} `json:"data"`
	}
	if err := json.Unmarshal(detailRecorder.Body.Bytes(), &detailPayload); err != nil {
		t.Fatalf("decode detail response: %v", err)
	}
	var completePayload struct {
		Data struct {
			CompleteCourse struct {
				Materials []struct {
					MaterialULID string `json:"material_ulid"`
				} `json:"materials"`
			} `json:"complete_course"`
		} `json:"data"`
	}
	if err := json.Unmarshal(completeRecorder.Body.Bytes(), &completePayload); err != nil {
		t.Fatalf("decode complete response: %v", err)
	}
	if detailPayload.Data.CourseDetail.ChapterCount != 2 || detailPayload.Data.CourseDetail.LessonCount != 3 || len(completePayload.Data.CompleteCourse.Materials) != 1 || completePayload.Data.CompleteCourse.Materials[0].MaterialULID != "material-1" {
		t.Fatalf("course detail = %+v, complete = %+v", detailPayload.Data, completePayload.Data)
	}
}
