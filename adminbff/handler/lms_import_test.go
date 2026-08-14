package handler

import (
	"net/http/httptest"
	"testing"
)

func TestValidateImportCoursePackageAcceptsCloudflareVideoAndChapterQuiz(t *testing.T) {
	recorder := httptest.NewRecorder()
	valid := validateImportCoursePackage(recorder, importCourseJSON{
		Title: "Foundation in Crypto Regulation and Compliance",
		Chapters: []importChapterJSON{{
			Title: "Module 1",
			Lessons: []importLessonJSON{{
				Title:          "Introduction to Blockchain",
				LessonType:     "video",
				VideoProvider:  "cloudflare",
				VideoStreamUID: "576943fb6c7a2cbf7acbfb2682adc6ee",
			}},
		}},
		Quizzes: []importChapterQuizJSON{{
			ChapterTitle: "Module 1 Quiz",
			Title:        "Module 1 Quiz",
			PassingScore: 70,
			Questions: []importQuizQuestionJSON{{
				QuestionText: "Which network introduced smart contracts?",
				Options: []importQuizOptionJSON{{
					OptionText: "Ethereum",
					IsCorrect:  true,
				}},
			}},
		}},
	})
	if !valid {
		t.Fatalf("expected package to be valid, got status %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestValidateImportCoursePackageRejectsVideoWithoutMediaOrStream(t *testing.T) {
	recorder := httptest.NewRecorder()
	valid := validateImportCoursePackage(recorder, importCourseJSON{
		Title: "Course",
		Chapters: []importChapterJSON{{
			Title: "Chapter 1",
			Lessons: []importLessonJSON{{
				Title:      "Video",
				LessonType: "video",
			}},
		}},
	})
	if valid {
		t.Fatal("expected video without media_object_key or video_stream_uid to be rejected")
	}
	if recorder.Code != 400 {
		t.Fatalf("expected HTTP 400, got %d", recorder.Code)
	}
}
