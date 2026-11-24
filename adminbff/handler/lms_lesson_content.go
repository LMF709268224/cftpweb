package handler

import (
	"net/http"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UpdateLmsLessonContent changes only title/body. The LMS update RPC replaces
// all lesson fields, so retain the other fields from its authoritative record.
// Never reconstruct media metadata from the frontend's abbreviated form.
func (h *Handler) UpdateLmsLessonContent(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "lesson_id")
	if !ok {
		return
	}
	var input struct {
		Title   *string `json:"title"`
		Body    *string `json:"body"`
		Version uint32  `json:"version"`
	}
	if err := ReadJSON(r, &input); err != nil || input.Title == nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "title and version are required")
		return
	}
	if !requireRequestField(w, *input.Title, "title") || !requirePositiveVersion(w, input.Version) {
		return
	}
	result, err := h.Lms.GetLessonAdmin(r.Context(), &lmspb.GetLessonRequest{LessonUlid: id})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	if result == nil || result.Lesson == nil || result.Lesson.LessonUlid != id {
		WriteError(w, http.StatusBadGateway, ErrInternal, "LMS returned an incomplete lesson")
		return
	}
	lesson := result.Lesson
	if lesson.Version != input.Version {
		WriteError(w, http.StatusConflict, ErrPrecondition, "lesson version conflict; reload before saving")
		return
	}
	body := lesson.Body
	if input.Body != nil {
		body = *input.Body
	}
	response, err := h.Lms.UpdateLessonAdmin(r.Context(), &lmspb.UpdateLessonRequest{
		LessonUlid: lesson.LessonUlid, Title: *input.Title, Body: body,
		Version: input.Version, SortOrder: lesson.SortOrder, LessonType: lesson.LessonType,
		MediaObjectKey: lesson.MediaObjectKey, MediaFileHash: lesson.MediaFileHash,
		ExternalUrl: lesson.ExternalUrl, ExternalCoursewareUlid: lesson.ExternalCoursewareUlid,
		VideoProvider: lesson.VideoProvider, VideoStreamUid: lesson.VideoStreamUid,
		VideoEmbedCode: lesson.VideoEmbedCode, MetaJson: lesson.MetaJson,
	})
	if err != nil {
		if status.Code(err) == codes.Aborted {
			WriteError(w, http.StatusConflict, ErrPrecondition, status.Convert(err).Message())
			return
		}
		writeLmsError(w, err)
		return
	}
	if response == nil || !response.Success {
		WriteError(w, http.StatusBadGateway, ErrInternal, "LMS did not confirm the lesson update")
		return
	}
	WriteJSON(w, http.StatusOK, response)
}
