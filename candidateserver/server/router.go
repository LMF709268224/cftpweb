package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"candidateserver/handler"

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
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(s.corsMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Public Webhooks (No Candidate Auth)
	r.Route("/api/public/webhooks", func(r chi.Router) {
		r.Post("/exams/callback/{urlType}/{examId}", h.ThirdPartyExamCallback)
	})

	r.Get("/api/public/config", h.GetPublicConfig)

	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/login-url", h.GetLoginURL)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/refresh", h.RefreshToken)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(s.authMiddleware)

		r.Route("/user", func(r chi.Router) {
			r.Get("/me", h.GetUserMe)
			r.Put("/profile", h.UpdateUserProfile)
			r.Put("/password", h.UpdateUserPassword)
		})

		r.Route("/mall", func(r chi.Router) {
			r.Route("/pipelines", func(r chi.Router) {
				r.Get("/", h.ListPipelines)
				r.Get("/{pipelineId}", h.GetPipelineDetail)
				r.Get("/{pipelineId}/thumbnail-url", h.GetMallPipelineThumbnailURL)
				r.Get("/{pipelineId}/runtime", h.GetPipelineRuntime)
				r.Get("/{pipelineId}/timeline", h.GetPipelineTimeline)
				r.Post("/{pipelineId}/purchase", h.PurchasePipeline)
				r.Post("/{pipelineId}/unlock", h.UnlockPipeline)
				r.Get("/{pipelineId}/active-order", h.GetActivePipelineOrder)
				r.Get("/{pipelineId}/eligibility", h.CheckPipelineEligibility)
			})
			r.Route("/payments", func(r chi.Router) {
				r.Post("/preview", h.PreviewPayment)
				r.Post("/initiate", h.InitiatePayment)
			})
		})
		r.Get("/mall/courses/{courseId}", h.GetMallCourseSummary)
		r.Get("/mall/courses/{courseId}/thumbnail-url", h.GetMallCourseThumbnailURL)

		r.Route("/pipeline", func(r chi.Router) {
			r.Get("/", h.ListMyPipelines)
			r.Get("/materials", h.ListMaterials)
			r.Get("/materials/{materialId}/url", h.GetAccessURL)
			r.Get("/courses/{courseId}/complete", h.GetPipelineCourse)
			r.Get("/lessons/{lessonId}", h.GetPipelineLessonDetail)
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
			r.Post("/upload-url", h.RequestUploadUrl)
			r.Post("/submit", h.SubmitApplication)
			r.Put("/update", h.UpdateApplication)
		})

		r.Route("/certificates", func(r chi.Router) {
			r.Get("/", h.ListCertificates)
		})

		r.Route("/orders", func(r chi.Router) {
			r.Get("/", h.ListOrders)
			r.Get("/{orderId}", h.GetOrder)
		})

		r.Route("/invoices", func(r chi.Router) {
			r.Get("/{orderId}", h.QueryInvoice)
			r.Get("/{orderId}/pdf", h.DownloadPdf)
		})

		r.Route("/messages", func(r chi.Router) {
			r.Get("/", h.ListMessages)
			r.Put("/read", h.MarkMessagesRead)
			r.Post("/delete", h.DeleteMessage)
			r.Get("/{messageId}", h.GetMessageDetail)
		})

		r.Route("/dashboard", func(r chi.Router) {
			r.Get("/", h.Dashboard)
			r.Get("/stats", h.GetDashboardStats)
		})

		r.Route("/membership", func(r chi.Router) {
			r.Get("/", h.GetMembership)
		})

		r.Route("/records", func(r chi.Router) {
			r.Get("/", h.ListRecords)
			r.Post("/", h.CreateRecord)
		})
	})

	serveSPA(r, "web/build")
	return r
}

func serveSPA(r *chi.Mux, publicDir string) {
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, publicDir)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			handler.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "API route not found"})
			return
		}

		fullPath, ok := safeJoin(filesDir, r.URL.Path)
		if !ok {
			handler.WriteError(w, http.StatusBadRequest, handler.ErrInvalidRequest, "invalid file path")
			return
		}

		if stat, err := os.Stat(fullPath); err == nil && !stat.IsDir() {
			serveStaticFile(w, r, fullPath)
			return
		}

		htmlPath := fullPath + ".html"
		if stat, err := os.Stat(htmlPath); err == nil && !stat.IsDir() {
			serveHTMLFile(w, r, htmlPath)
			return
		}

		indexPath := filepath.Join(fullPath, "index.html")
		if stat, err := os.Stat(indexPath); err == nil && !stat.IsDir() {
			serveHTMLFile(w, r, indexPath)
			return
		}

		serveHTMLFile(w, r, filepath.Join(filesDir, "index.html"))
	})
}

func serveStaticFile(w http.ResponseWriter, r *http.Request, filename string) {
	if strings.HasPrefix(r.URL.Path, "/_next/static/") {
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
