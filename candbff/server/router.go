package server

import (
	"net/http"
	"time"

	"candbff/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// buildRouter 构建 HTTP 路由，所有面向考生 Web 端的 API 都在这里注册
func (s *Server) buildRouter(h *handler.Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(s.corsMiddleware)

	normalTimeout := middleware.Timeout(60 * time.Second)
	pdfPreviewTimeout := middleware.Timeout(30 * time.Minute)

	r.With(normalTimeout).Get("/health", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Public Webhooks (No Candidate Auth)
	r.Route("/api/public/webhooks", func(r chi.Router) {
		r.Use(normalTimeout)
		r.Post("/exams/callback/{urlType}/{examId}", h.ThirdPartyExamCallback)
	})
	r.Route("/api/public/pdf-preview", func(r chi.Router) {
		r.Use(pdfPreviewTimeout)
		r.Get("/lessons/{lessonId}", h.PreviewLessonPDFPublic)
		r.Head("/lessons/{lessonId}", h.PreviewLessonPDFPublic)
		r.Get("/resource", h.PreviewResourceURLPublic)
		r.Head("/resource", h.PreviewResourceURLPublic)
		r.Get("/resource-pack-files/{fileId}", h.PreviewResourcePackFilePDFPublic)
		r.Head("/resource-pack-files/{fileId}", h.PreviewResourcePackFilePDFPublic)
	})

	r.With(normalTimeout).Get("/api/public/config", h.GetPublicConfig)

	r.Route("/api/auth", func(r chi.Router) {
		r.Use(normalTimeout)
		r.Get("/login-url", h.GetLoginURL)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/refresh", h.RefreshToken)
	})

	r.With(s.authMiddleware, pdfPreviewTimeout).Get("/api/pipeline/resource-preview", h.PreviewResourceURL)
	r.With(s.authMiddleware, pdfPreviewTimeout).Head("/api/pipeline/resource-preview", h.PreviewResourceURL)
	r.With(s.authMiddleware, pdfPreviewTimeout).Get("/api/pipeline/lessons/{lessonId}/preview", h.PreviewLessonPDF)
	r.With(s.authMiddleware, pdfPreviewTimeout).Head("/api/pipeline/lessons/{lessonId}/preview", h.PreviewLessonPDF)

	r.Route("/api", func(r chi.Router) {
		r.Use(normalTimeout)
		r.Use(s.authMiddleware)

		r.Route("/user", func(r chi.Router) {
			r.Get("/me", h.GetUserMe)
			r.Put("/profile", h.UpdateUserProfile)
			r.Put("/password", h.UpdateUserPassword)
		})

		r.Route("/membership", func(r chi.Router) {
			r.Get("/plans", h.ListMembershipPlans)
			r.Get("/active", h.GetActiveMembership)
			r.Get("/history", h.ListUserMemberships)
			r.Get("/billings", h.ListMembershipBillings)
			r.Post("/cancel", h.CancelMembership)
		})

		r.Route("/mall", func(r chi.Router) {
			r.Route("/pipelines", func(r chi.Router) {
				r.Get("/", h.ListPipelines)
				r.Get("/{pipelineId}", h.GetPipelineDetail)
				r.Get("/{pipelineId}/thumbnail-url", h.GetMallPipelineThumbnailURL)
				r.Get("/{pipelineId}/runtime", h.GetPipelineRuntime)
				r.Get("/{pipelineId}/timeline", h.GetPipelineTimeline)
			})
			r.Route("/bundles", func(r chi.Router) {
				r.Get("/", h.ListBundles)
				r.Get("/{bundleId}", h.GetBundleDetail)
				r.Get("/{bundleId}/thumbnail-url", h.GetBundleThumbnailURL)
				r.Post("/{bundleId}/purchase", h.CreateBundleOrder)
				r.Post("/{bundleId}/unlock", h.UnlockPipelineInBundle)
			})
			r.Route("/payments", func(r chi.Router) {
				r.Post("/initiate", h.InitiatePayment)
			})
		})
		r.Get("/mall/courses/{courseId}", h.GetMallCourseSummary)
		r.Get("/mall/courses/{courseId}/thumbnail-url", h.GetMallCourseThumbnailURL)

		r.Route("/pipeline", func(r chi.Router) {
			r.Get("/", h.ListMyPipelines)
			r.Get("/materials", h.ListMaterials)
			r.Get("/materials/{materialId}/url", h.GetAccessURL)
			r.Get("/resource-preview-url", h.GetResourcePreviewURL)
			r.Get("/courses/{courseId}/complete", h.GetPipelineCourse)
			r.Get("/lessons/{lessonId}", h.GetPipelineLessonDetail)
			r.Get("/lessons/{lessonId}/preview-url", h.GetLessonPreviewURL)
			r.Post("/lessons/{lessonId}/complete", h.CompletePipelineLesson)
			r.Get("/{pipelineUlid}/certificate-url", h.GetPipelineCertificateViewURL)
		})

		r.Route("/progress", func(r chi.Router) {
			r.Post("/courses/{courseId}/sync", h.SyncCourseProgress)
			r.Post("/", h.ReportProgress)
			r.Get("/", h.GetProgress)
		})

		r.Route("/enrollments", func(r chi.Router) {
			r.Get("/", h.ListCandidateEnrollments)
			r.Get("/{enrollmentId}", h.GetCandidateEnrollmentDetail)
		})

		r.Route("/resource-packs", func(r chi.Router) {
			r.Get("/", h.ListResourcePacks)
			r.Get("/{pack_id}/files", h.ListResourcePackFiles)
		})

		r.Route("/resource-pack-files", func(r chi.Router) {
			r.Get("/{file_id}/view-url", h.GetResourcePackFileViewURL)
			r.Get("/{file_id}/thumbnail-url", h.GetResourcePackFileThumbnailURL)
			r.Get("/{file_id}/preview-url", h.GetResourcePackFilePreviewURL)
		})

		r.Route("/quizzes", func(r chi.Router) {
			r.Post("/{quizId}/take", h.TakeQuiz)
			r.Get("/attempts/{attemptId}/paper", h.GetQuizPaper)
			r.Post("/attempts/{attemptId}/submit", h.SubmitQuiz)
		})

		r.Route("/exams", func(r chi.Router) {
			r.Get("/", h.ListExams)
			r.Get("/history", h.ListExamHistory)

			r.Route("/units/{courseUnitUlid}", func(r chi.Router) {
				r.Post("/signup", h.SignupExam)
				r.Post("/retake", h.ApplyRetake)
				r.Post("/retake-payment", h.PrepareRetakePayment)
				r.Post("/exemption", h.ApplyExemption)
			})

			r.Route("/{examId}", func(r chi.Router) {
				r.Get("/schedule-url", h.GetScheduleURL)
				r.Get("/result", h.GetExamResult)
				r.Get("/schedule-callback/{urlType}", h.TermUrlRedirectCallback)
				r.Post("/schedule-callback", h.TermUrlCallback)
			})
		})

		r.Route("/credentials", func(r chi.Router) {
			r.Get("/definitions", h.ListCredentialDefinitions)
			r.Get("/applications", h.ListCandidateApplications)
			r.Get("/qualifications", h.CheckCandidateQualifications)
			r.Get("/upload-permission", h.CheckUploadPermission)
			r.Post("/application-orders", h.CreateCredentialApplicationOrder)
			r.Post("/upload-url", h.RequestUploadUrl)
			r.Post("/submit", h.SubmitApplication)
			r.Put("/update", h.UpdateApplication)
		})

		r.Route("/certificates", func(r chi.Router) {
			r.Get("/", h.ListCertificates)
		})

		r.Route("/orders", func(r chi.Router) {
			r.Get("/", h.ListOrders)
		})
		r.Route("/invoices", func(r chi.Router) {
			r.Get("/{orderId}", h.QueryInvoice)
			r.Get("/{orderId}/pdf", h.DownloadPdf)
		})

		r.Route("/messages", func(r chi.Router) {
			r.Get("/", h.ListMessages)
			r.Get("/unread-count", h.GetUnreadMessageCount)
			r.Put("/read", h.MarkMessagesRead)
			r.Post("/delete", h.DeleteMessage)
			r.Get("/{messageId}", h.GetMessage)
		})

		r.Route("/dashboard", func(r chi.Router) {
			r.Get("/", h.Dashboard)
			r.Get("/stats", h.GetDashboardStats)
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "API route not found"})
	})
	return r
}
