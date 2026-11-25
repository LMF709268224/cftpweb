package handler

import (
	"fmt"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
)

type enrollmentReadView struct {
	*lmspb.CourseEnrollment
	ProgressPercentage uint32 `json:"progress_percentage"`
}

type completeCourseReadView struct {
	*lmspb.CompleteCourse
	Chapters []chapterReadView `json:"chapters"`
}

type chapterReadView struct {
	*lmspb.ChapterDetail
	Lessons []lessonDetailReadView `json:"lessons"`
}

type lessonDetailReadView struct {
	*lmspb.LessonDetail
	Lesson lessonReadView `json:"lesson"`
}

type lessonReadView struct {
	*lmspb.Lesson
	Body string `json:"body"`
}

// These fields have explicit protobuf semantics: repeated fields are lists,
// HasMore is a boolean, and ProgressPercentage is the stored uint32 value.
// Expose those same values in HTTP JSON without omitempty. A missing response
// or missing entity is an error, never an empty successful result.
func courseEnrollmentsPayload(resp *lmspb.ListCourseEnrollmentsResponse) (map[string]interface{}, error) {
	if resp == nil {
		return nil, fmt.Errorf("LMS returned no enrollment response")
	}
	items := make([]enrollmentReadView, 0, len(resp.Enrollments))
	for _, enrollment := range resp.Enrollments {
		if enrollment == nil {
			return nil, fmt.Errorf("LMS returned a missing enrollment entity")
		}
		items = append(items, enrollmentReadView{CourseEnrollment: enrollment, ProgressPercentage: enrollment.ProgressPercentage})
	}
	return map[string]interface{}{
		"enrollments": items, "has_more": resp.HasMore,
		"next_cursor": resp.NextCursor, "prev_cursor": resp.PrevCursor,
	}, nil
}

func completeCoursePayload(resp *lmspb.GetCompleteCourseResponse) (map[string]interface{}, error) {
	if resp == nil || resp.CompleteCourse == nil || resp.CompleteCourse.Course == nil {
		return nil, fmt.Errorf("LMS returned an incomplete course response")
	}
	course := resp.CompleteCourse
	chapters := make([]chapterReadView, 0, len(course.Chapters))
	for _, chapter := range course.Chapters {
		if chapter == nil || chapter.Chapter == nil {
			return nil, fmt.Errorf("LMS returned a missing chapter entity")
		}
		lessons := make([]lessonDetailReadView, 0, len(chapter.Lessons))
		for _, lesson := range chapter.Lessons {
			if lesson == nil || lesson.Lesson == nil {
				return nil, fmt.Errorf("LMS returned a missing lesson entity")
			}
			lessons = append(lessons, lessonDetailReadView{LessonDetail: lesson, Lesson: lessonReadView{Lesson: lesson.Lesson, Body: lesson.Lesson.Body}})
		}
		chapters = append(chapters, chapterReadView{ChapterDetail: chapter, Lessons: lessons})
	}
	content := completeCourseReadView{CompleteCourse: course, Chapters: chapters}
	return map[string]interface{}{"complete_course": content}, nil
}
