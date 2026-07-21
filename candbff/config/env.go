package config

// ────────────────────────────────────────────────────────────────────
// 环境变量集中定义
// 所有环境变量读取统一使用此文件中定义的常量，方便运维查看需要配置哪些变量
// ────────────────────────────────────────────────────────────────────

const (
	// ── 服务器监听地址 ──
	// HTTP_ADDRESS  HTTP 服务器监听地址，默认 "0.0.0.0:8080"
	EnvHTTPAddress = "HTTP_ADDRESS"

	// ── TLS 证书目录 ──
	// TLS_DIR  TLS 证书文件目录，目录内需包含 ca.crt
	//          未设置时所有 gRPC 客户端以 insecure 模式运行
	EnvTLSDir = "TLS_DIR"

	// ── 配置中心 ──
	// CFGSERVER_ADDR  cfgserver 配置中心地址
	//                 未设置时自动拼接 K8s DNS: cfgserver.<ns>.svc.cluster.local:50051
	EnvCfgServerAddr = "CFGSERVER_ADDR"

	// ── Casdoor IAM ──
	// CASDOOR_ENDPOINT  Casdoor 服务地址
	//                   未设置时自动拼接 K8s DNS: casdoor.<ns>.svc.cluster.local:8000
	EnvCasdoorEndpoint = "CASDOOR_ENDPOINT"

	// CASDOOR_PUBLIC_ENDPOINT  Casdoor 公网访问地址，用于前端跳转
	EnvCasdoorPublicEndpoint = "CASDOOR_PUBLIC_ENDPOINT"

	// STRIPE_PUBLISHABLE_KEY  Stripe publishable key，可安全暴露给浏览器用于初始化 Stripe.js
	EnvStripePublishableKey = "STRIPE_PUBLISHABLE_KEY"

	// ROLE_STUDENT_BASIC 学生基础角色名，默认为 "role_student_basic"
	EnvRoleStudentBasic = "ROLE_STUDENT_BASIC"

	// ── 下游微服务 gRPC 地址 ──
	// 每个变量对应一个下游服务，
	// 未设置时自动拼接 K8s DNS 名: <service>.<ns>.svc.cluster.local:50051

	// MALL_GRPC_ADDR  gmall 商城服务地址
	EnvMallGrpcAddr = "MALL_GRPC_ADDR"
	// LMS_GRPC_ADDR   glms 课程/资料服务地址
	EnvLmsGrpcAddr = "LMS_GRPC_ADDR"
	// GCC_GRPC_ADDR gcc 证书服务地址
	EnvGccGrpcAddr = "GCC_GRPC_ADDR"
	// GPROG_GRPC_ADDR gprog 进度服务地址
	EnvGprogGrpcAddr = "GPROG_GRPC_ADDR"
	// GMSG_GRPC_ADDR gmsg 消息服务地址
	EnvGmsgGrpcAddr = "GMSG_GRPC_ADDR"
	// GCREDS_GRPC_ADDR gcreds 凭证服务地址
	EnvGcredsGrpcAddr = "GCREDS_GRPC_ADDR"
	// GEXAM_GRPC_ADDR gexam 考试服务地址
	EnvGexamGrpcAddr = "GEXAM_GRPC_ADDR"
	// EXAM_CALLBACK_BASE_URL public origin or full callback URL for third-party exam callbacks.
	EnvExamCallbackBaseURL = "EXAM_CALLBACK_BASE_URL"
	// GMID_GRPC_ADDR gmid ID映射服务地址
	EnvGmidGrpcAddr = "GMID_GRPC_ADDR"
	// GPAY_GRPC_ADDR gpay 支付服务地址
	EnvGpayGrpcAddr = "GPAY_GRPC_ADDR"
	// GMBR_GRPC_ADDR gmbr 会员服务地址
	EnvGmbrGrpcAddr = "GMBR_GRPC_ADDR"
	// GMAIL_GRPC_ADDR gmail 邮件服务地址
	EnvGmailGrpcAddr = "GMAIL_GRPC_ADDR"

	// ── CORS ──
	// CORS_ALLOWED_ORIGINS  允许的跨域来源，逗号分隔，默认 "*" 允许所有
	EnvCORSOrigins = "CORS_ALLOWED_ORIGINS"

	// ── 日志 ──
	// LOG_LEVEL  日志级别: debug / info / warn / error，默认 info
	EnvLogLevel = "LOG_LEVEL"
	// LOG_SOURCE  是否在日志中输出源码文件和行号，设置为 "true" 开启
	EnvLogSource = "LOG_SOURCE"
)
