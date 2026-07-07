package server

import (
	"net/http"
	"time"

	"adminbff/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// buildRouter 构建 HTTP 路由
func (s *Server) buildRouter(h *handler.Handler) http.Handler {
	r := chi.NewRouter()

	// ---------- 管理后台接口----------
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(s.corsMiddleware)

	// ---------- 健康检查 (公开) ----------
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// ---------- 认证 (公开) ----------
	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/login-url", h.GetLoginURL)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/refresh", h.RefreshToken)
	})

	// ---------- 需要认证的 API ----------
	r.Route("/api", func(r chi.Router) {
		r.Use(s.authMiddleware)

		// ===== 用户 (User) =====
		r.Route("/user", func(r chi.Router) {
			r.Get("/me", h.GetAdminMe)
			r.Get("/list", h.ListUsers)
			r.Put("/profile", h.UpdateUserProfile)
			r.Put("/password", h.UpdateUserPassword)
		})

		// ===== 运营看板 =====
		r.Get("/dashboard/ops", h.OpsDashboard)

		// ===== 课程与认证 (GCC) =====
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

		r.Route("/prog/certificate-tasks", func(r chi.Router) {
			r.Get("/", h.ListProgCertificateTasks)
			r.Get("/{task_ulid}", h.GetProgCertificateTaskDetail)
			r.Post("/{task_ulid}/retry", h.RetryProgCertificateTask)
		})

		r.Route("/prog/course-units", func(r chi.Router) {
			r.Post("/{course_unit_ulid}/force-completed", h.AdminForceCourseCompleted)
			r.Post("/{course_unit_ulid}/force-signup-exam", h.AdminForceCourseSignupExam)
		})

		r.Route("/exams", func(r chi.Router) {
			r.Get("/", h.ListAdminExams)
			r.Get("/{exam_ulid}", h.GetAdminExamDetail)
			r.Get("/{exam_ulid}/result", h.GetAdminExamResult)
			r.Get("/{exam_ulid}/transitions", h.GetAdminExamTransitions)
			r.Post("/{exam_ulid}/sync-result", h.SyncAdminExamResult)
		})

		// ===== 目录 (Catalogs) =====
		// 提供目录的 CRUD 操作
		r.Route("/catalogs", func(r chi.Router) {
			r.Get("/", h.ListCatalogs)
			r.Post("/", h.CreateCatalog)
			r.Put("/{catalog_id}", h.UpdateCatalog)
		})

		// ===== GLMS 学习管理系统 (LMS Courses) =====
		r.Route("/lms", func(r chi.Router) {
			r.Route("/courses", func(r chi.Router) {
				r.Get("/", h.ListLmsCourses)
				r.Post("/", h.CreateLmsCourse)
				r.Get("/{course_id}/complete", h.GetCompleteLmsCourse)
				r.Get("/{course_id}/detail", h.GetLmsCourseDetail)
				r.Post("/{course_id}/publish", h.PublishLmsCourse)
				r.Post("/{course_id}/cleanup-assets", h.CleanUpDeprecatedCourseAssets)
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
				r.Post("/{pack_id}/publish", h.PublishLmsResourcePack)
				r.Post("/{pack_id}/revert-to-draft", h.RevertLmsResourcePackToDraft)
				r.Post("/{pack_id}/duplicate", h.DuplicateLmsResourcePack)
				r.Post("/{pack_id}/cleanup-assets", h.CleanUpDeprecatedResourcePackAssets)
				r.Delete("/{pack_id}", h.DeleteLmsResourcePack)
				r.Get("/{pack_id}/files", h.ListLmsResourcePackFiles)
				r.Post("/{pack_id}/files", h.CreateLmsResourcePackFile)
			})

			r.Route("/resource-pack-files", func(r chi.Router) {
				r.Get("/", h.ListAllLmsResourcePackFiles)
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

		// ===== 邮件接口 (Mails) =====
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
			r.Get("/definitions/{cred_def_ulid}", h.GetCredentialDefinitionDetail)
			r.Post("/definitions", h.CreateCredentialDefinition)
		})

		// ===== 应用 (Applications) =====
		r.Route("/applications", func(r chi.Router) {
			r.Get("/", h.ListApplications)
			r.Get("/{app_id}", h.GetApplication)
			r.Post("/audit", h.AuditApplication)
		})

		// ===== PDF模板接口 (PDF Templates) =====
		r.Route("/pdf-templates", func(r chi.Router) {
			r.Get("/", h.ListPdfTemplates)
			r.Post("/", h.CreatePdfTemplate)
			r.Put("/", h.UpdatePdfTemplate)
		})

		// ===== PDF 生成请求接口 (PDF Requests) =====
		r.Route("/pdf-requests", func(r chi.Router) {
			r.Get("/", h.ListPdfRequests)
		})

		// ===== 商城 (Mall) =====
		r.Route("/mall", func(r chi.Router) {
			r.Get("/stages/{stage_ulid}/order-status", h.GetStageOrderStatus)
			r.Get("/stage-orders", h.ListStageOrders)
			r.Get("/orders", h.ListOrders)
			r.Get("/invoices", h.ListInvoices)
			r.Get("/bundle-orders", h.ListBundleOrders)
			r.Route("/bundles", func(r chi.Router) {
				r.Get("/", h.ListBundles)
				r.Post("/", h.CreateBundle)
				r.Get("/schemas", h.GetBundleJsonSchemas)
				r.Post("/sync-display-pricing", h.AdminSyncBundleDisplayPricing)
				r.Get("/{bundle_ulid}", h.GetBundle)
				r.Put("/{bundle_ulid}/meta", h.UpdateBundleMeta)
				r.Put("/pricing", h.UpdateBundlePricing)
				r.Post("/{bundle_ulid}/publish", h.PublishBundle)
				r.Post("/{bundle_ulid}/deprecate", h.DeprecateBundle)
				r.Delete("/{bundle_ulid}", h.DeleteBundle)
				r.Post("/upload-url", h.CreateBundleUploadURL)
			})
			r.Route("/bundle-orders", func(r chi.Router) {
				r.Get("/", h.ListBundleOrders)
				r.Get("/{bundle_order_ulid}/summary", h.GetBundleOrderSummary)
				r.Get("/{bundle_order_ulid}", h.GetBundleOrderDetail)
				r.Post("/purge", h.AdminPurgeCandidateBundle)
			})
		})

		// ===== Webhook 审计 (Exam Webhooks) =====
		// ===== Memberships =====
		r.Route("/memberships", func(r chi.Router) {
			r.Get("/", h.ListMemberships)
			r.Post("/", h.AdminCreateMembershipConfig)
			r.Put("/", h.AdminUpdateMembershipConfig)
			r.Get("/active", h.GetActiveMembership)
			r.Get("/users", h.ListUserMemberships)
			r.Get("/billings", h.ListMembershipBillings)
			r.Post("/grant", h.AdminGrantMembership)
			r.Post("/revoke", h.AdminRevokeMembership)
			r.Post("/purge", h.AdminPurgeCandidateMembership)
			r.Get("/{membership_ulid}", h.GetMembership)
			r.Post("/{membership_ulid}/publish", h.AdminPublishMembershipConfig)
			r.Post("/{membership_ulid}/deprecate", h.AdminDeprecateMembershipConfig)
			r.Route("/mails", func(r chi.Router) {
				r.Get("/", h.ListMembershipMails)
				r.Get("/{mail_ulid}", h.GetMembershipMailDetail)
				r.Post("/retry", h.RetryMembershipMail)
				r.Post("/ignore", h.IgnoreMembershipMail)
			})
		})

		// ===== Webhook Audit =====
		r.Route("/audit", func(r chi.Router) {
			r.Route("/logs", func(r chi.Router) {
				r.Get("/", h.ListAuditLogs)
				r.Get("/{audit_ulid}", h.GetAuditLogDetail)
			})
			r.Route("/webhooks", func(r chi.Router) {
				r.Get("/", h.ListWebhookMessages)
				r.Get("/detail", h.GetWebhookMessageDetail)
				r.Post("/reprocess", h.ReprocessWebhookMessage)
			})
		})

		// ===== 权限管理 (Permissions) =====
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/check", h.CheckCandidateQualification)
			r.Post("/grant", h.GrantUploadPermission)
			r.Post("/revoke", h.RevokeUploadPermission)
			r.Post("/mark-expired", h.MarkExpired)
			r.Post("/revoke-credential", h.RevokeCredential)
		})
	}) // <-- /api 路由分组结束
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "API route not found"})
	})

	return r
}
