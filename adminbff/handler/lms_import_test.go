package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestValidateImportCoursePackageAcceptsCloudflareVideoAndChapterQuiz(t *testing.T) {
	recorder := httptest.NewRecorder()
	valid := validateImportCoursePackage(recorder, importCourseJSON{
		Title:       "Foundation in Crypto Regulation and Compliance",
		CourseGPath: "/gcc/pipeline/core/foundation_in_crypto_regulation_and_compliance",
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
		Title:       "Course",
		CourseGPath: "/gcc/pipeline/core/course",
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

func TestValidateImportCoursePackageRejectsMissingCourseGPath(t *testing.T) {
	recorder := httptest.NewRecorder()
	valid := validateImportCoursePackage(recorder, importCourseJSON{Title: "Course"})
	if valid {
		t.Fatal("expected package without course_gpath to be rejected")
	}
	if recorder.Code != 400 {
		t.Fatalf("expected HTTP 400, got %d", recorder.Code)
	}
}

func TestBuildImportCoursePayloadPreservesGLMSFieldsAndRemovesPackageFields(t *testing.T) {
	payload, err := buildImportCoursePayload(`{
		"package_type":"cftp-lms-course-package",
		"package_version":1,
		"category_tips":"Short Course",
		"course_gpath":"/gcc/pipeline/core/course",
		"title":"Course",
		"description":"Description",
		"duration_min":60,
		"materials":[{"title":"Workbook"}],
		"chapters":[],
		"quizzes":[{"title":"Quiz"}]
	}`)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"course_gpath", "title", "description", "duration_min", "materials", "chapters"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected %s to be forwarded", key)
		}
	}
	for _, key := range []string{"package_type", "package_version", "category_tips", "quizzes"} {
		if _, ok := decoded[key]; ok {
			t.Fatalf("expected %s to be removed", key)
		}
	}
}
