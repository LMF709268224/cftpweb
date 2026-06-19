package server

import (
	"net/http"
	"time"

	"adminbff/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// buildRouter 閺嬪嫬缂?HTTP 鐠侯垳鏁? 閹碘偓閺堝娼伴崥鎴ｂ偓鍐晸 Web 缁旑垳娈?API 闁棄婀潻娆撳櫡濞夈劌鍞?
func (s *Server) buildRouter(h *handler.Handler) http.Handler {
	r := chi.NewRouter()

	// ---------- 闁氨鏁ゆ稉顓㈡？娴?----------
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(s.corsMiddleware)

	// ---------- 閸嬨儱鎮嶅Λ鈧弻?(閺冪娀娓剁拋銈堢槈) ----------
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// ---------- 鐠併倛鐦夐幒銉ュ經 (閺冪娀娓剁拋銈堢槈) ----------
	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/login-url", h.GetLoginURL)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/refresh", h.RefreshToken)
	})

	// ---------- 闂団偓鐟曚浇顓荤拠浣烘畱 API ----------
	r.Route("/api", func(r chi.Router) {
		r.Use(s.authMiddleware)

		// ===== 閻劍鍩?(User) =====
		r.Route("/user", func(r chi.Router) {
			r.Get("/me", h.GetAdminMe)
			r.Get("/list", h.ListUsers)
			r.Put("/profile", h.UpdateUserProfile)
			r.Put("/password", h.UpdateUserPassword)
		})

		// ===== 鐠囧墽鈻兼稉搴ｎ吀缁捐法顓搁悶?(GCC) =====
		r.Route("/pipelines", func(r chi.Router) {
			r.Get("/", h.ListPipelines)
			r.Get("/{pipeline_id}", h.GetPipeline)
			r.Post("/", h.CreatePipelineDraft)
			r.Put("/{pipeline_id}/structure", h.UpdatePipelineStructure)
			r.Put("/{pipeline_id}/metadata", h.UpdatePipelineMetadata)
			r.Post("/{pipeline_id}/publish", h.PublishPipeline)
			r.Post("/{pipeline_id}/deprecate", h.DeprecatePipeline)
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

		// ===== 閻╊喖缍嶉崚鍡欒缁狅紕鎮?(Catalogs) =====
		// 鐠愮喕鐭楃拠鍓р柤閸掑棛琚妴浣圭垼缁涘彞缍嬬化鑽ゆ畱 CRUD閿涘瞼鏁ゆ禍搴＄殺缁狅紕鍤庤ぐ鎺旇鐏炴洜銇?
		r.Route("/catalogs", func(r chi.Router) {
			r.Get("/", h.ListCatalogs)
			r.Post("/", h.CreateCatalog)
			r.Put("/{catalog_id}", h.UpdateCatalog)
		})

		// ===== GLMS 鐠囧墽鈻奸崘鍛啇闁板秶鐤?(LMS Courses) =====
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

		// ===== 閭欢绠＄悊 (Mails) =====
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

		// ===== 璧勬牸涓庤瘉涔︾鐞?(Credentials) =====
		r.Route("/credentials", func(r chi.Router) {
			r.Get("/definitions", h.ListCredentialDefinitions)
			r.Post("/definitions", h.CreateCredentialDefinition)
		})

		// ===== 璧勬牸瀹℃牳涓績 (Applications) =====
		r.Route("/applications", func(r chi.Router) {
			r.Get("/", h.ListApplications)
			r.Post("/audit", h.AuditApplication)
		})

		// ===== PDF妯℃澘绠＄悊 (PDF Templates) =====
		r.Route("/pdf-templates", func(r chi.Router) {
			r.Get("/", h.ListPdfTemplates)
			r.Post("/", h.CreatePdfTemplate)
			r.Put("/", h.UpdatePdfTemplate)
		})

		// ===== PDF 璇佷功鐢熸垚璇锋眰绠＄悊 (PDF Requests) =====
		r.Route("/pdf-requests", func(r chi.Router) {
			r.Get("/", h.ListPdfRequests)
		})

		// ===== 璁㈠崟涓庢敮浠?(Mall) =====
		r.Route("/mall", func(r chi.Router) {
			r.Get("/stages/{stage_ulid}/order-status", h.GetStageOrderStatus)
			r.Get("/stage-orders", h.ListStageOrders)
			r.Get("/orders", h.ListOrders)
			r.Get("/invoices", h.ListInvoices)
		})

		// ===== Webhook 瀹¤ (Exam Webhooks) =====
		r.Route("/audit/webhooks", func(r chi.Router) {
			r.Get("/", h.ListWebhookMessages)
			r.Get("/detail", h.GetWebhookMessageDetail)
			r.Post("/reprocess", h.ReprocessWebhookMessage)
		})

		// ===== 鏉冮檺浜哄伐骞查 (Permissions) =====
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/check", h.CheckCandidateQualification)
			r.Post("/grant", h.GrantUploadPermission)
			r.Post("/revoke", h.RevokeUploadPermission)
			r.Post("/mark-expired", h.MarkExpired)
			r.Post("/revoke-credential", h.RevokeCredential)
		})
	}) // <-- 琛ュ洖 /api 璺敱鐨勭粨鏉熷ぇ鎷彿

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "API route not found"})
	})

	return r
}
