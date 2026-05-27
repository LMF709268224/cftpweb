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

// buildRouter 构建 HTTP 路由, 所有面向考生 Web 端的 API 都在这里注册
func (s *Server) buildRouter(h *handler.Handler) http.Handler {
	r := chi.NewRouter()

	// ---------- 通用中间件 ----------
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(s.corsMiddleware)

	// ---------- 健康检查 (无需认证) ----------
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// ---------- 认证接口 (无需认证) ----------
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

		// ===== 课程与管线管理 (GCC) =====
		r.Route("/pipelines", func(r chi.Router) {
			r.Get("/", h.ListPipelines)
			r.Get("/{pipeline_id}", h.GetPipeline)
			r.Post("/", h.CreatePipelineDraft)
			r.Put("/{pipeline_id}/structure", h.UpdatePipelineStructure)
			r.Post("/{pipeline_id}/publish", h.PublishPipeline)
		})

		// ===== 目录分类管理 (Catalogs) =====
		// 负责课程分类、标签体系的 CRUD，用于将管线归类展示
		r.Route("/catalogs", func(r chi.Router) {
			r.Get("/", h.ListCatalogs)
			r.Post("/", h.CreateCatalog)
			r.Put("/{catalog_id}", h.UpdateCatalog)
		})

		// ===== 消息管理 (Messages) =====
		r.Route("/messages", func(r chi.Router) {
			r.Get("/templates", h.ListTemplates)
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

			r.Route("/templates", func(r chi.Router) {
				r.Get("/", h.ListMailTemplates)
				r.Get("/detail", h.GetMailTemplate)
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

		// 检查该文件在磁盘上是否存在且不是目录
		if stat, err := os.Stat(fullPath); err == nil && !stat.IsDir() {
			// 文件存在，直接正常返回静态文件
			http.ServeFile(w, r, fullPath)
			return
		}

		// 3. 适配 Next.js Static Export：尝试寻找对应的 .html 文件
		// 例如请求 /callback，寻找 /callback.html
		htmlPath := fullPath + ".html"
		if stat, err := os.Stat(htmlPath); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, htmlPath)
			return
		}

		// 例如请求 /courses，寻找 /courses/index.html
		indexPath := filepath.Join(fullPath, "index.html")
		if stat, err := os.Stat(indexPath); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, indexPath)
			return
		}

		// 4. 既不是 API，也不是存在的静态资源或 Next.js 页面
		// 统一把 index.html 交给浏览器（但这在 Next.js App Router 静态导出中通常会导致 hydration mismatch 或渲染错误页面，
		// 不过为了兜底还是保留）
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
