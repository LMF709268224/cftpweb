package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"adminserver/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// buildRouter 鏋勫缓 HTTP 璺敱, 鎵€鏈夐潰鍚戣€冪敓 Web 绔殑 API 閮藉湪杩欓噷娉ㄥ唽
func (s *Server) buildRouter(h *handler.Handler) http.Handler {
	r := chi.NewRouter()

	// ---------- 閫氱敤涓棿浠?----------
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(s.corsMiddleware)

	// ---------- 鍋ュ悍妫€鏌?(鏃犻渶璁よ瘉) ----------
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// ---------- 璁よ瘉鎺ュ彛 (鏃犻渶璁よ瘉) ----------
	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/login-url", h.GetLoginURL)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/refresh", h.RefreshToken)
	})

	// ---------- 闇€瑕佽璇佺殑 API ----------
	r.Route("/api", func(r chi.Router) {
		r.Use(s.authMiddleware)

		// ===== 鐢ㄦ埛 (User) =====
		r.Route("/user", func(r chi.Router) {
			r.Get("/me", h.GetAdminMe)
			r.Get("/list", h.ListUsers)
			r.Put("/profile", h.UpdateUserProfile)
			r.Put("/password", h.UpdateUserPassword)
		})

		// ===== 璇剧▼涓庣绾跨鐞?(GCC) =====
		r.Route("/pipelines", func(r chi.Router) {
			r.Get("/", h.ListPipelines)
			r.Get("/{pipeline_id}", h.GetPipeline)
			r.Post("/", h.CreatePipelineDraft)
			r.Put("/{pipeline_id}/structure", h.UpdatePipelineStructure)
			r.Put("/{pipeline_id}/metadata", h.UpdatePipelineMetadata)
			r.Post("/{pipeline_id}/publish", h.PublishPipeline)
			r.Delete("/{pipeline_id}", h.DeletePipeline)
		})

		r.Route("/prog/pipelines", func(r chi.Router) {
			r.Get("/", h.ListProgPipelines)
			r.Get("/{pipeline_ulid}", h.GetProgPipelineDetail)
			r.Get("/{pipeline_ulid}/logs", h.ListProgStatusTransitionLogs)
			r.Get("/logs/{transition_ulid}", h.GetProgStatusTransitionLogDetail)
			r.Post("/{pipeline_ulid}/trigger-next-stage", h.AdminTriggerProgNextStage)
			r.Post("/{pipeline_ulid}/terminate", h.AdminTerminatePipeline)
			r.Get("/{pipeline_ulid}/certificate-url", h.GetProgPipelineCertificateViewURL)
		})

		r.Route("/prog/course-units", func(r chi.Router) {
			r.Post("/{course_unit_ulid}/force-completed", h.AdminForceCourseCompleted)
			r.Post("/{course_unit_ulid}/force-signup-exam", h.AdminForceCourseSignupExam)
		})

		// ===== 鐩綍鍒嗙被绠＄悊 (Catalogs) =====
		// 璐熻矗璇剧▼鍒嗙被銆佹爣绛句綋绯荤殑 CRUD锛岀敤浜庡皢绠＄嚎褰掔被灞曠ず
		r.Route("/catalogs", func(r chi.Router) {
			r.Get("/", h.ListCatalogs)
			r.Post("/", h.CreateCatalog)
			r.Put("/{catalog_id}", h.UpdateCatalog)
		})

		// ===== GLMS 璇剧▼鍐呭閰嶇疆 (LMS Courses) =====
		r.Route("/lms", func(r chi.Router) {
			r.Route("/courses", func(r chi.Router) {
				r.Get("/", h.ListLmsCourses)
				r.Post("/", h.CreateLmsCourse)
				r.Get("/{course_id}/complete", h.GetCompleteLmsCourse)
				r.Get("/{course_id}/detail", h.GetLmsCourseDetail)
				r.Post("/{course_id}/publish", h.PublishLmsCourse)
				r.Get("/{course_id}/enrollments", h.ListLmsCourseEnrollmentsForAdmin)
				r.Get("/{course_id}/candidates/{candidate_id}/progress", h.GetLmsCandidateProgressForAdmin)
				r.Post("/{course_id}/progress/sync", h.SyncLmsCourseProgress)
				r.Get("/{course_id}", h.GetLmsCourse)
				r.Put("/{course_id}", h.UpdateLmsCourse)
				r.Delete("/{course_id}", h.DeleteLmsCourse)
				r.Get("/{course_id}/materials", h.ListLmsCourseMaterials)
				r.Post("/{course_id}/materials", h.CreateLmsCourseMaterial)
				r.Post("/{course_id}/materials/reorder", h.ReorderLmsCourseMaterials)
				r.Get("/{course_id}/supplementary-material", h.GetLmsCourseSupplementaryMaterial)
				r.Post("/{course_id}/supplementary-material", h.CreateLmsCourseSupplementaryMaterial)
				r.Put("/{course_id}/supplementary-material/{material_id}", h.UpdateLmsCourseSupplementaryMaterial)
				r.Delete("/{course_id}/supplementary-material/{material_id}", h.DeleteLmsCourseSupplementaryMaterial)
				r.Post("/{course_id}/permissions/grant", h.GrantLmsCourseAccessPermission)
				r.Post("/{course_id}/permissions/revoke", h.RevokeLmsCourseAccessPermission)
				r.Get("/{course_id}/chapters", h.ListLmsChapters)
				r.Post("/{course_id}/chapters", h.CreateLmsChapter)
				r.Post("/{course_id}/chapters/reorder", h.ReorderLmsChapters)
			})

			r.Route("/materials", func(r chi.Router) {
				r.Get("/{material_id}", h.GetLmsCourseMaterial)
				r.Put("/{material_id}", h.UpdateLmsCourseMaterial)
				r.Delete("/{material_id}", h.DeleteLmsCourseMaterial)
			})

			r.Route("/chapters", func(r chi.Router) {
				r.Get("/{chapter_id}/lessons", h.ListLmsLessons)
				r.Post("/{chapter_id}/lessons", h.CreateLmsLesson)
				r.Post("/{chapter_id}/lessons/reorder", h.ReorderLmsLessons)
				r.Get("/{chapter_id}/progress", h.GetLmsChapterProgressDetail)
				r.Get("/{chapter_id}", h.GetLmsChapter)
				r.Put("/{chapter_id}", h.UpdateLmsChapter)
				r.Delete("/{chapter_id}", h.DeleteLmsChapter)
			})

			r.Route("/lessons", func(r chi.Router) {
				r.Get("/{lesson_id}/progress", h.GetLmsLessonProgressDetail)
				r.Get("/{lesson_id}", h.GetLmsLesson)
				r.Put("/{lesson_id}", h.UpdateLmsLesson)
				r.Delete("/{lesson_id}", h.DeleteLmsLesson)
			})

			r.Route("/prerequisites", func(r chi.Router) {
				r.Get("/", h.ListLmsPrerequisites)
				r.Post("/", h.CreateLmsPrerequisite)
				r.Get("/{prerequisite_id}", h.GetLmsPrerequisite)
				r.Put("/{prerequisite_id}", h.UpdateLmsPrerequisite)
				r.Delete("/{prerequisite_id}", h.DeleteLmsPrerequisite)
			})

			r.Route("/quizzes", func(r chi.Router) {
				r.Get("/", h.ListLmsQuizzes)
				r.Post("/", h.CreateLmsQuiz)
				r.Get("/{quiz_id}/attempts", h.ListLmsQuizAttempts)
				r.Get("/{quiz_id}/questions", h.ListLmsQuizQuestions)
				r.Post("/{quiz_id}/questions", h.CreateLmsQuizQuestion)
				r.Post("/{quiz_id}/questions/reorder", h.ReorderLmsQuizQuestions)
				r.Get("/{quiz_id}", h.GetLmsQuiz)
				r.Put("/{quiz_id}", h.UpdateLmsQuiz)
				r.Delete("/{quiz_id}", h.DeleteLmsQuiz)
			})

			r.Route("/questions", func(r chi.Router) {
				r.Get("/{question_id}/options", h.ListLmsQuizOptions)
				r.Post("/{question_id}/options", h.CreateLmsQuizOption)
				r.Post("/{question_id}/options/reorder", h.ReorderLmsQuizOptions)
				r.Get("/{question_id}", h.GetLmsQuizQuestion)
				r.Put("/{question_id}", h.UpdateLmsQuizQuestion)
				r.Delete("/{question_id}", h.DeleteLmsQuizQuestion)
			})

			r.Route("/options", func(r chi.Router) {
				r.Get("/{option_id}", h.GetLmsQuizOption)
				r.Put("/{option_id}", h.UpdateLmsQuizOption)
				r.Delete("/{option_id}", h.DeleteLmsQuizOption)
			})

			r.Route("/resource-packs", func(r chi.Router) {
				r.Get("/", h.ListLmsResourcePacks)
				r.Post("/", h.CreateLmsResourcePack)
				r.Get("/{pack_id}", h.GetLmsResourcePack)
				r.Put("/{pack_id}", h.UpdateLmsResourcePack)
				r.Delete("/{pack_id}", h.DeleteLmsResourcePack)
				r.Get("/{pack_id}/files", h.ListLmsResourcePackFiles)
				r.Post("/{pack_id}/files", h.CreateLmsResourcePackFile)
			})

			r.Route("/resource-pack-files", func(r chi.Router) {
				r.Get("/{file_id}", h.GetLmsResourcePackFile)
				r.Put("/{file_id}", h.UpdateLmsResourcePackFile)
				r.Delete("/{file_id}", h.DeleteLmsResourcePackFile)
			})

			r.Get("/enrollments", h.ListLmsCourseEnrollments)
			r.Post("/enrollments/batch", h.BatchEnrollLmsCandidateCourses)
			r.Get("/enrollments/{enrollment_id}", h.GetLmsCourseEnrollmentDetail)
			r.Get("/lesson-progress", h.ListLmsLessonProgress)
			r.Get("/chapter-progress", h.ListLmsChapterProgress)
			r.Get("/quiz-attempts/{attempt_id}", h.GetLmsQuizAttemptDetail)
			r.Get("/assets", h.ListLmsCourseAssets)
			r.Get("/assets/detail", h.GetLmsCourseAssetDetail)
			r.Get("/objects", h.ListLmsObjects)
			r.Post("/upload-url", h.CreateLmsUploadURL)
			r.Post("/view-url", h.CreateLmsViewURL)
			r.Post("/import", h.ImportLmsContent)
			r.Get("/broken-assets", h.ListLmsBrokenAssets)
		})

		// ===== Messages =====
		r.Route("/messages", func(r chi.Router) {
			r.Get("/templates", h.ListTemplates)
			r.Get("/templates/detail", h.GetMessageTemplate)
			r.Post("/templates", h.CreateTemplate)
			r.Put("/templates", h.UpdateTemplate)
			r.Delete("/templates", h.DeleteMessageTemplate)
			r.Get("/sent", h.ListSentMessages)
			r.Post("/send", h.SendMessage)
		})

		// ===== 邮件管理 (Mails) =====
		r.Route("/mails", func(r chi.Router) {
			r.Post("/send", h.SendMail)
			r.Get("/sent", h.ListSentMails)
			r.Get("/", h.GetMail)
			r.Get("/status", h.GetMailStatus)
			r.Post("/cancel", h.CancelMail)
			r.Get("/stats", h.GetMailStats)

			r.Route("/templates", func(r chi.Router) {
				r.Get("/", h.ListMailTemplates)
				r.Get("/detail", h.GetMailTemplate)
				r.Get("/exists", h.HasMailTemplate)
				r.Get("/builtin-paths", h.GetAllBuiltInPaths)
				r.Post("/render", h.RenderMailTemplate)
				r.Post("/", h.CreateMailTemplate)
				r.Put("/", h.UpdateMailTemplate)
				r.Delete("/", h.DeleteMailTemplate)
			})
		})

		// ===== 资格与证书管理 (Credentials) =====
		r.Route("/credentials", func(r chi.Router) {
			r.Get("/definitions", h.ListCredentialDefinitions)
			r.Post("/definitions", h.CreateCredentialDefinition)
		})

		// ===== 资格审核中心 (Applications) =====
		r.Route("/applications", func(r chi.Router) {
			r.Get("/", h.ListApplications)
			r.Post("/audit", h.AuditApplication)
		})

		// ===== PDF模板管理 (PDF Templates) =====
		r.Route("/pdf-templates", func(r chi.Router) {
			r.Get("/", h.ListPdfTemplates)
			r.Post("/", h.CreatePdfTemplate)
			r.Put("/", h.UpdatePdfTemplate)
		})

		// ===== PDF 证书生成请求管理 (PDF Requests) =====
		r.Route("/pdf-requests", func(r chi.Router) {
			r.Get("/", h.ListPdfRequests)
		})

		// ===== 订单与支付 (Mall) =====
		r.Route("/mall", func(r chi.Router) {
			r.Get("/stages/{stage_ulid}/order-status", h.GetStageOrderStatus)
			r.Get("/stage-orders", h.ListStageOrders)
			r.Get("/orders", h.ListOrders)
		})

		// ===== Webhook 审计 (Exam Webhooks) =====
		r.Route("/audit/webhooks", func(r chi.Router) {
			r.Get("/", h.ListWebhookMessages)
			r.Get("/detail", h.GetWebhookMessageDetail)
			r.Post("/reprocess", h.ReprocessWebhookMessage)
		})

		// ===== 权限人工干预 (Permissions) =====
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/check", h.CheckCandidateQualification)
			r.Post("/grant", h.GrantUploadPermission)
			r.Post("/revoke", h.RevokeUploadPermission)
			r.Post("/mark-expired", h.MarkExpired)
			r.Post("/revoke-credential", h.RevokeCredential)
		})
	}) // <-- 补回 /api 路由的结束大括号

	// ---------- SPA 静态文件服务 ----------
	// 当请求没有匹配到任何以上的 API 路由时，进入此逻辑
	serveSPA(r, "web/build")

	return r
}

// serveSPA 处理前端的静态资源和 React/Vue 路由 fallback
func serveSPA(r *chi.Mux, publicDir string) {
	// 获取当前工作目录
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, publicDir)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// 1. 如果是以 /api/ 开头的请求，但没有匹配到路由，直接返回 404 JSON (不要把前端网页当成 JSON 塞回去)
		if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
			handler.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "API route not found"})
			return
		}

		// 2. 尝试寻找静态文件，比如 /static/css/main.css 或者 /favicon.ico
		fullPath, ok := safeJoin(filesDir, r.URL.Path)
		if !ok {
			handler.WriteError(w, http.StatusBadRequest, handler.ErrInvalidRequest, "invalid file path")
			return
		}

		// Serve an existing static file directly.
		if stat, err := os.Stat(fullPath); err == nil && !stat.IsDir() {
			serveStaticFile(w, r, fullPath)
			return
		}

		// 3. 适配 Next.js Static Export：尝试寻找对应的 .html 文件
		// 例如请求 /callback，寻找 /callback.html
		htmlPath := fullPath + ".html"
		if stat, err := os.Stat(htmlPath); err == nil && !stat.IsDir() {
			serveHTMLFile(w, r, htmlPath)
			return
		}

		// 例如请求 /courses，寻找 /courses/index.html
		indexPath := filepath.Join(fullPath, "index.html")
		if stat, err := os.Stat(indexPath); err == nil && !stat.IsDir() {
			serveHTMLFile(w, r, indexPath)
			return
		}

		// 4. 既不是 API，也不是存在的静态资源或 Next.js 页面
		// 统一把 index.html 交给浏览器（但这在 Next.js App Router 静态导出中通常会导致 hydration mismatch 或渲染错误页面，
		serveHTMLFile(w, r, filepath.Join(filesDir, "index.html"))
	})
}

func serveStaticFile(w http.ResponseWriter, r *http.Request, filename string) {
	if strings.HasPrefix(r.URL.Path, "/_next/static/") || strings.HasPrefix(r.URL.Path, "/assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	http.ServeFile(w, r, filename)
}

func serveHTMLFile(w http.ResponseWriter, r *http.Request, filename string) {
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, filename)
}

func safeJoin(baseDir, requestPath string) (string, bool) {
	cleanPath := filepath.Clean("/" + requestPath)
	cleanPath = strings.TrimPrefix(cleanPath, string(filepath.Separator))
	fullPath := filepath.Join(baseDir, cleanPath)

	rel, err := filepath.Rel(baseDir, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return fullPath, true
}
