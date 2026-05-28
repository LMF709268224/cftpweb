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
			r.Post("/{pipeline_id}/publish", h.PublishPipeline)
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
				r.Post("/{course_id}/publish", h.PublishLmsCourse)
				r.Get("/{course_id}/enrollments", h.ListLmsCourseEnrollmentsForAdmin)
				r.Get("/{course_id}/candidates/{candidate_id}/progress", h.GetLmsCandidateProgressForAdmin)
				r.Get("/{course_id}", h.GetLmsCourse)
				r.Put("/{course_id}", h.UpdateLmsCourse)
				r.Delete("/{course_id}", h.DeleteLmsCourse)
				r.Get("/{course_id}/materials", h.ListLmsCourseMaterials)
				r.Post("/{course_id}/materials", h.CreateLmsCourseMaterial)
				r.Post("/{course_id}/materials/reorder", h.ReorderLmsCourseMaterials)
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
				r.Get("/{chapter_id}", h.GetLmsChapter)
				r.Put("/{chapter_id}", h.UpdateLmsChapter)
				r.Delete("/{chapter_id}", h.DeleteLmsChapter)
			})

			r.Route("/lessons", func(r chi.Router) {
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

			r.Get("/objects", h.ListLmsObjects)
			r.Post("/upload-url", h.CreateLmsUploadURL)
			r.Post("/view-url", h.CreateLmsViewURL)
			r.Post("/import", h.ImportLmsContent)
			r.Get("/broken-assets", h.ListLmsBrokenAssets)
		})

		// ===== Messages =====
		r.Route("/messages", func(r chi.Router) {
			r.Get("/templates", h.ListTemplates)
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

			r.Route("/templates", func(r chi.Router) {
				r.Get("/", h.ListMailTemplates)
				r.Get("/detail", h.GetMailTemplate)
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

		// ===== 鏉冮檺浜哄伐骞查 (Permissions) =====
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/check", h.CheckCandidateQualification)
			r.Post("/grant", h.GrantUploadPermission)
			r.Post("/revoke", h.RevokeUploadPermission)
			r.Post("/mark-expired", h.MarkExpired)
			r.Post("/revoke-credential", h.RevokeCredential)
		})
	}) // <-- 琛ュ洖 /api 璺敱鐨勭粨鏉熷ぇ鎷彿

	// ---------- SPA 闈欐€佹枃浠舵湇鍔?----------
	// 褰撹姹傛病鏈夊尮閰嶅埌浠讳綍浠ヤ笂鐨?API 璺敱鏃讹紝杩涘叆姝ら€昏緫
	serveSPA(r, "web/build")

	return r
}

// serveSPA 澶勭悊鍓嶇鐨勯潤鎬佽祫婧愬拰 React/Vue 璺敱 fallback
func serveSPA(r *chi.Mux, publicDir string) {
	// 鑾峰彇褰撳墠宸ヤ綔鐩綍
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, publicDir)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// 1. 濡傛灉鏄互 /api/ 寮€澶寸殑璇锋眰锛屼絾娌℃湁鍖归厤鍒拌矾鐢憋紝鐩存帴杩斿洖 404 JSON (涓嶈鎶婂墠绔綉椤靛綋鎴?JSON 濉炲洖鍘?
		if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
			handler.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "API route not found"})
			return
		}

		// 2. 灏濊瘯瀵绘壘闈欐€佹枃浠讹紝姣斿 /static/css/main.css 鎴栬€?/favicon.ico
		fullPath, ok := safeJoin(filesDir, r.URL.Path)
		if !ok {
			handler.WriteError(w, http.StatusBadRequest, handler.ErrInvalidRequest, "invalid file path")
			return
		}

		// Serve an existing static file directly.
		if stat, err := os.Stat(fullPath); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, fullPath)
			return
		}

		// 3. 閫傞厤 Next.js Static Export锛氬皾璇曞鎵惧搴旂殑 .html 鏂囦欢
		// 渚嬪璇锋眰 /callback锛屽鎵?/callback.html
		htmlPath := fullPath + ".html"
		if stat, err := os.Stat(htmlPath); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, htmlPath)
			return
		}

		// 渚嬪璇锋眰 /courses锛屽鎵?/courses/index.html
		indexPath := filepath.Join(fullPath, "index.html")
		if stat, err := os.Stat(indexPath); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, indexPath)
			return
		}

		// 4. 鏃笉鏄?API锛屼篃涓嶆槸瀛樺湪鐨勯潤鎬佽祫婧愭垨 Next.js 椤甸潰
		// 缁熶竴鎶?index.html 浜ょ粰娴忚鍣紙浣嗚繖鍦?Next.js App Router 闈欐€佸鍑轰腑閫氬父浼氬鑷?hydration mismatch 鎴栨覆鏌撻敊璇〉闈紝
		http.ServeFile(w, r, filepath.Join(filesDir, "index.html"))
	})
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
