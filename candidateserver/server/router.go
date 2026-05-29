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
			r.Get("/me", h.GetUserMe)
			r.Put("/profile", h.UpdateUserProfile)
			r.Put("/password", h.UpdateUserPassword)
		})

		// ===== 商城 (Mall) =====
		r.Route("/mall", func(r chi.Router) {
			r.Route("/pipelines", func(r chi.Router) {
				r.Get("/", h.ListPipelines)                          // 认证列表
				r.Get("/{pipelineId}", h.GetPipelineDetail)          // 认证详情 (含考生进度聚合)
				r.Post("/{pipelineId}/purchase", h.PurchasePipeline) // 购买/下单
			})
		})

		// ===== 管线 (pipeline) =====
		r.Route("/pipeline", func(r chi.Router) {
			r.Get("/", h.ListMyPipelines)        // 我的管线
			r.Get("/materials", h.ListMaterials) // 最近学习资料
			r.Get("/materials/{materialId}/url", h.GetAccessURL)
		})

		r.Route("/progress", func(r chi.Router) {
			r.Post("/", h.ReportProgress) // 上报进度
			r.Get("/", h.GetProgress)     // 查询进度
		})

		// ===== 考试管理 (Exam) =====
		r.Route("/exams", func(r chi.Router) {
			// 1. 考试列表查询
			// TODO: 需要 gprog 新增 gRPC 接口
			// r.Get("/waiting", h.ListExams)       // 待参加考试列表
			r.Get("/history", h.ListExamHistory) // 历史考试记录

			// 2. 基于课程单元(Unit)的考试申请 (此时还未生成独立 examId)
			// TODO: courseUnitUlid 需从 GetPipelineInstance gRPC 中获取
			r.Route("/units/{courseUnitUlid}", func(r chi.Router) {
				r.Post("/signup", h.SignupExam)        // 报名首次考试
				r.Post("/retake", h.ApplyRetake)       // 申请补考
				r.Post("/exemption", h.ApplyExemption) // 申请免考
			})

			// 3. 基于具体考试(Exam)的操作
			// TODO: examId 从管线详情或者新增接口获取
			r.Route("/{examId}", func(r chi.Router) {
				r.Get("/schedule-url", h.GetScheduleURL) // 获取预约考位链接 (对接 GEE)
				r.Get("/result", h.GetExamResult)        // 获取该场考试成绩

				// TODO: 需要 gprog 新增 NotifyExamBookingResult gRPC 接口
				r.Post("/schedule-callback", h.TermUrlCallback) // 第三方系统预约结果异步回调 (原 termurl)
			})
		})

		// ===== 资格与证书 (Credentials & Certificates) =====
		r.Route("/credentials", func(r chi.Router) {
			r.Get("/definitions", h.ListCredentialDefinitions)
			r.Get("/applications", h.ListCandidateApplications)
			r.Post("/upload-url", h.RequestUploadUrl)
			r.Post("/submit", h.SubmitApplication)
			r.Put("/update", h.UpdateApplication)
		})

		r.Route("/certificates", func(r chi.Router) {
			r.Get("/", h.ListCertificates) // 我的证书列表
		})

		// ===== 支付与对账 (Order & Billing) =====
		r.Route("/orders", func(r chi.Router) {
			// TODO: 需 gmall 补充订单列表查询
			r.Get("/", h.ListOrders)
			// TODO: 需 gmall 补充订单详情查询
			r.Get("/{orderId}", h.GetOrder)
		})

		r.Route("/invoices", func(r chi.Router) {
			r.Get("/{orderId}", h.QueryInvoice)
			r.Get("/{orderId}/pdf", h.DownloadPdf)
		})

		// ===== 消息与设置 (Messages & Profile) =====
		r.Route("/messages", func(r chi.Router) {
			r.Get("/", h.ListMessages)
			r.Put("/read", h.MarkMessagesRead)
			r.Post("/delete", h.DeleteMessage) //删除消息
		})

		r.Route("/dashboard", func(r chi.Router) {
			r.Get("/", h.Dashboard)
			r.Get("/stats", h.GetDashboardStats)
		})

		//TODO 以下微服务还没完成，先不考虑
		// ===== 会员 (Membership) =====
		r.Route("/membership", func(r chi.Router) {
			r.Get("/", h.GetMembership)
		})
		// ===== 档案 (Records) =====
		r.Route("/records", func(r chi.Router) {
			r.Get("/", h.ListRecords)
			r.Post("/", h.CreateRecord)
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
		// 不过为了兜底还是保留）
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
