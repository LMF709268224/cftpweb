package handler

import (
	gcreds "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	"github.com/afnandelfin620-star/cftptest/cftp/gprog"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
)

// ===================== 注册登录 (Auth) =====================

type LoginInput struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type AuthURLRsp struct {
	URL string `json:"url"`
}

type LoginRsp struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type UserInfo struct {
	Name string `json:"name"`
}

// ===================== 用户 (User) =====================

type UserMeRsp struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Affiliation string `json:"affiliation"`
	Title       string `json:"title"`
	RealName    string `json:"real_name"`
	Bio         string `json:"bio"`
	Gender      string `json:"gender"`
	Birthday    string `json:"birthday"`
	Education   string `json:"education"`
}

type UserProfileInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Affiliation string `json:"affiliation"`
	Title       string `json:"title"`
	RealName    string `json:"real_name"`
	Bio         string `json:"bio"`
	Gender      string `json:"gender"`
	Birthday    string `json:"birthday"`
	Education   string `json:"education"`
}

type UserPasswordInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type BaseRsp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ===================== 商城 =====================

type ListPipelinesRsp struct {
	Pipelines []PipelineConfig `json:"pipelines,omitempty"`
}

type PipelineDetailRsp struct {
	Config   PipelineConfig  `json:"config"`
	Instance PipelineSummary `json:"instance,omitempty"` // 如果已购买，则包含实例状态
}

type PipelineConfig struct {
	UnlockStripeProductId  string          `json:"unlock_stripe_product_id,omitempty"`
	UnlockStripePriceId    string          `json:"unlock_stripe_price_id,omitempty"`
	PackageStripeProductId string          `json:"package_stripe_product_id,omitempty"`
	PackageStripePriceId   string          `json:"package_stripe_price_id,omitempty"`
	PipelineUlid           string          `json:"pipeline_id,omitempty"`      // ULID (版本唯一ID) [required]
	PipelineGuid           string          `json:"pipeline_guid,omitempty"`    // ULID (业务唯一ID) [required]
	Version                uint32          `json:"version,omitempty"`          // 版本号[required]
	Name                   string          `json:"name,omitempty"`             // 管线名称 [required]
	CategoryTips           string          `json:"category_tips,omitempty"`    // 分类提示 [required]
	UnlockFee              int64           `json:"unlock_fee,omitempty"`       // 解锁费用，单位：分[required]
	PackageDiscount        int32           `json:"package_discount,omitempty"` // 套餐折扣，单位：基点（如 500 = 95%）[required]
	UnlockQuals            []Qualification `json:"unlock_quals,omitempty"`     // 解锁条件 [required]
	CertQuals              []Qualification `json:"cert_quals,omitempty"`       // 证书要求 [required]
	Stages                 []StageConfig   `json:"stages,omitempty"`           // 阶段配置 [required]
	Status                 string          `json:"status,omitempty"`           // 状态[required]
	IsCurrent              bool            `json:"is_current,omitempty"`       // 是否为当前版本[required]
	CreatedAt              string          `json:"created_at,omitempty"`       // 创建时间 [required]
	FinalQuals             []Qualification `json:"final_quals,omitempty"`      // 结业资格 [required]
}

type Qualification struct {
	QualId   string `json:"qual_id,omitempty"`   // 资格 ULID 对应gcred中的cred catalog id [required]
	NameHint string `json:"name_hint,omitempty"` // 资格名称 [required]
}

type StageConfig struct {
	StageUlid string       `json:"stage_id,omitempty"`   // 管线阶段的 ULID [required]
	Name      string       `json:"name,omitempty"`       // 阶段名称 [required]
	SortOrder int32        `json:"sort_order,omitempty"` // 排序顺序 [required]
	Units     []UnitConfig `json:"units,omitempty"`      // 单元配置 [required]
}

type UnitConfig struct {
	StripeProductId          string `json:"stripe_product_id,omitempty"`
	StripePriceId            string `json:"stripe_price_id,omitempty"`
	ExemptionStripeProductId string `json:"exemption_stripe_product_id,omitempty"`
	ExemptionStripePriceId   string `json:"exemption_stripe_price_id,omitempty"`
	RetakeStripeProductId    string `json:"retake_stripe_product_id,omitempty"`
	RetakeStripePriceId      string `json:"retake_stripe_price_id,omitempty"`
	GlmsCourseUlid           string `json:"glms_course_id,omitempty"`
	UnitUlid                 string `json:"unit_id,omitempty"`          // 阶段单元(课程) ULID & GLMS ID [required]
	Name                     string `json:"name,omitempty"`             // 阶段单元名称 [required]
	HasLearning              bool   `json:"has_learning,omitempty"`     // 是否有学习[required]
	HasExam                  bool   `json:"has_exam,omitempty"`         // 是否有考试 [required]
	LearningMinutes          int32  `json:"learning_minutes,omitempty"` // 课时要求，以分钟为单位[required]
	// 考试扁平化参数
	ProgramCode       string   `json:"program_code,omitempty"`        // 考试参数：课程代码[required]
	ExamCode          string   `json:"exam_code,omitempty"`           // 考试参数：考试代号 [required]
	ExamForm          string   `json:"exam_form,omitempty"`           // 考试参数：考试形式 [required]
	BaseFee           int64    `json:"base_fee,omitempty"`            // 基本费，单位：分 [required]
	ExemptionQuals    []string `json:"exemption_quals,omitempty"`     // 资格 ULID 列表  [required]
	ExemptionAuditFee int64    `json:"exemption_audit_fee,omitempty"` // 资格审核费，单位：分 [required]
	AllowRetake       bool     `json:"allow_retake,omitempty"`        // 是否允许重考[required]
	RetakeFee         int64    `json:"retake_fee,omitempty"`          // 重考费用，单位：分 [required]
}

// ===================== 课程与认证 (Courses & Pipelines) =====================

type ListMyPipelinesRsp struct {
	List []PipelineSummary `json:"list,omitempty"` // pipeline 摘要列表 [required]
}

// PipelineSummary pipeline 摘要
type PipelineSummary struct {
	PipelineUlid     string                 `json:"pipeline_ulid,omitempty"`      // pipeline 实例 ULID [required]
	CandidateUlid    string                 `json:"candidate_ulid,omitempty"`     // 考生 ULID [required]
	PipelineCcUlid   string                 `json:"pipeline_cc_ulid,omitempty"`   // gcc 中 pipeline 配置 ULID [required]
	Status           gprogpb.PipelineStatus `json:"status,omitempty"`             // pipeline 当前状态[required]
	CurrentStageUlid string                 `json:"current_stage_ulid,omitempty"` // 当前 stage 实例 ULID [optional]
	Progress         float64                `json:"progress"`                     // 当前阶段的学习进度(0-100)
	StartedAt        string                 `json:"started_at,omitempty"`         // pipeline 开始时间，RFC3339 [optional]
	CompletedAt      string                 `json:"completed_at,omitempty"`       // pipeline 完成时间，RFC3339 [optional]
	CreatedAt        string                 `json:"created_at,omitempty"`         // pipeline 创建时间，RFC3339 [required]
}

type PipelineCreateRsp struct {
	PipelineUlid       string                 `json:"pipeline_ulid,omitempty"`        // pipeline 实例 ULID [required]
	PipelineStatus     gprogpb.PipelineStatus `json:"pipeline_status,omitempty"`      // pipeline 当前状态[required]
	CurrentStageUlid   string                 `json:"current_stage_ulid,omitempty"`   // 当前 stage 实例 ULID，若无则为空 [optional]
	CurrentStageStatus gprogpb.StageStatus    `json:"current_stage_status,omitempty"` // 当前 stage 状态，若无则为 UNSPECIFIED [required]
	Message            string                 `json:"message,omitempty"`              // 人类可读说明 [required]
}

type AdminTerminatePipelineRsp struct {
	PipelineUlid   string                 `json:"pipeline_ulid,omitempty"`   // pipeline 实例 ULID [required]
	PipelineStatus gprogpb.PipelineStatus `json:"pipeline_status,omitempty"` // pipeline 当前状态[required]
	Message        string                 `json:"message,omitempty"`         // 人类可读说明 [required]
}

type AdminTriggerProgNextStageRsp struct {
	PipelineUlid       string                 `json:"pipeline_ulid,omitempty"`        // pipeline 实例 ULID [required]
	PipelineStatus     gprogpb.PipelineStatus `json:"pipeline_status,omitempty"`      // pipeline 当前状态[required]
	CurrentStageUlid   string                 `json:"current_stage_ulid,omitempty"`   // 当前 stage 实例 ULID [required]
	CurrentStageStatus gprogpb.StageStatus    `json:"current_stage_status,omitempty"` // 当前 stage 状态[required]
	Message            string                 `json:"message,omitempty"`              // 人类可读说明 [required]
}

type AdminForceCourseCompletedRsp struct {
	CourseUnitUlid   string                   `json:"course_unit_ulid,omitempty"`   // course unit 实例 ULID [required]
	CourseUnitStatus gprogpb.CourseUnitStatus `json:"course_unit_status,omitempty"` // course unit 当前状态[required]
	Message          string                   `json:"message,omitempty"`            // 人类可读说明 [required]
}

type AdminForceCourseSignupExamRsp struct {
	CourseUnitUlid   string                   `json:"course_unit_ulid,omitempty"`   // course unit 实例 ULID [required]
	CourseUnitStatus gprogpb.CourseUnitStatus `json:"course_unit_status,omitempty"` // course unit 当前状态[required]
	Message          string                   `json:"message,omitempty"`            // 人类可读说明 [required]
}

type PurchasePipelineInput struct {
	PipelineCcULID string `json:"pipeline_cc_ulid"`
}

type GetAccessURLRsp struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

type SignupExamInput struct {
	CourseUnitULID string `json:"course_unit_ulid"`
	FirstName      string `json:"first_name"`
	MiddleName     string `json:"middle_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	HomePhone      string `json:"home_phone"`
	WorkPhone      string `json:"work_phone"`
	Gender         string `json:"gender"`
	Birthdate      string `json:"birthdate"` // YYYY-MM-DD
	Country        string `json:"country"`
	Province       string `json:"province"`
	City           string `json:"city"`
	Address        string `json:"address"`
	PostalCode     string `json:"postal_code"`
}

type ApplyRetakeInput struct {
	CourseUnitULID string `json:"course_unit_ulid"`
}

// ===================== 首页 (Dashboard) =====================

type DashboardRsp struct {
	CandidateName       string            `json:"candidate_name"`
	Stats               DashboardStats    `json:"stats"`
	RecentPipelines     []PipelineSummary `json:"recent_pipelines"`
	PendingItems        []PendingItem     `json:"pending_items"`
	UnreadMessagesCount uint32            `json:"unread_messages_count"`
}

type DashboardUser struct {
	Name            string `json:"name"`
	MembershipLevel string `json:"membership_level"`
}

type PendingItem struct {
	Type        string `json:"type"` // message / record_review / exam_exemption
	Count       int    `json:"count"`
	Description string `json:"description"`
}

type CurrentLearning struct {
	CourseID         string `json:"course_id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	TotalProgress    int    `json:"total_progress"`
	CompletedModules int    `json:"completed_modules"`
	TotalModules     int    `json:"total_modules"`
	CurrentModule    string `json:"current_module"`
}

type DashboardStats struct {
	CoursesInProgress    int    `json:"courses_in_progress"`
	CertificationsEarned int    `json:"certifications_earned"`
	MembershipLevel      string `json:"membership_level"`
	UnreadMessages       int    `json:"unread_messages"`
}

// ===================== 学习进度 (Progress) =====================

type ProgressRecordInput struct {
	MaterialID      string  `json:"material_id"`
	CoursePackageID string  `json:"course_package_id"`
	ProgressType    int32   `json:"progress_type"`
	ProgressValue   float64 `json:"progress_value"`
	// RecordedAt      string  `json:"recorded_at"`
}

type ReportProgressInput struct {
	Records []ProgressRecordInput `json:"records"`
}

type ReportProgressRsp struct {
	AcceptedCount int32 `json:"accepted_count"`
	RejectedCount int32 `json:"rejected_count"`
}

type MaterialListItem struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Type            int32   `json:"type"`
	FileKey         string  `json:"file_key"`
	FileSize        int64   `json:"file_size"`
	DurationSeconds int32   `json:"duration_seconds"`
	ProgressValue   float64 `json:"progress_value"`
	ProgressType    int32   `json:"progress_type"`
}

type MaterialListRsp struct {
	Materials []MaterialListItem `json:"materials"`
}

type GetProgressRsp struct {
	Records []ProgressRecord `json:"records,omitempty"` // 进度记录列表 [required]
}

type ProgressRecord struct {
	CandidateUlid   string  `json:"candidate_id,omitempty"`      // 考生ID [required]
	MaterialUlid    string  `json:"material_id,omitempty"`       // 资料ID [required]
	CoursePackageId string  `json:"course_package_id,omitempty"` // 资料包ID [required]
	ProgressType    string  `json:"progress_type,omitempty"`     // 进度类型 [required]
	ProgressValue   float64 `json:"progress_value,omitempty"`    // 进度值：视频为秒数，文档为百分比 [required]
	RecordedAt      string  `json:"recorded_at,omitempty"`       // 记录时间 RFC3339 [required]
}

// ===================== 考试 (Exams) =====================

type ExamResultDetailRsp struct {
	ExamUlid         string  `json:"exam_id,omitempty"`
	TotalScore       float64 `json:"total_score,omitempty"`
	IsPassed         bool    `json:"is_passed,omitempty"`
	ScoreDetailsJson string  `json:"score_details_json,omitempty"`
}

type CandidateSignupExamRsp struct {
	CourseUnitUlid   string                 `json:"course_unit_ulid,omitempty"`   // course unit 实例 ULID [required]
	CourseUnitStatus gprog.CourseUnitStatus `json:"course_unit_status,omitempty"` // course unit 当前状态[required]
	Message          string                 `json:"message,omitempty"`            // 人类可读说明 [required]
}

// CandidateApplyRetake
type CandidateApplyRetakeRsp struct {
	CourseUnitUlid   string                 `json:"course_unit_ulid,omitempty"`   // course unit 实例 ULID [required]
	CourseUnitStatus gprog.CourseUnitStatus `json:"course_unit_status,omitempty"` // course unit 当前状态[required]
	Message          string                 `json:"message,omitempty"`            // 人类可读说明 [required]
}

type GetScheduleURLRsp struct {
	URL string `json:"url"`
}

type TermUrlInput struct {
	URLType      string `json:"url_type"`
	CallbackBody string `json:"callback_body"`
}

type TermUrlCallbackRsp struct {
	ExamID     string `json:"exam_id"`
	ExamStatus string `json:"exam_status"`
}

type ExamResultRsp struct {
	ExamID          string      `json:"exam_id"`
	TotalScore      float64     `json:"total_score"`
	IsPassed        bool        `json:"is_passed"`
	ScoreDetails    interface{} `json:"score_details,omitempty"`
	ScoreDetailsRaw string      `json:"score_details_raw,omitempty"`
}

// ===================== 支付 (Payments) =====================

type LineItemInput struct {
	Name        string `json:"name"`        // 商品名称，如 "Level 1 认证考试费用"
	Description string `json:"description"` // 商品详细描述
	UnitAmount  int64  `json:"unit_amount"` // 商品单价（单位：分，如 100 代表 1.00 元）
	Quantity    int64  `json:"quantity"`    // 购买数量
	Currency    string `json:"currency"`    // 货币币种，如 "usd", "cny"
}

type CreatePaymentInput struct {
	// OrderID    string            `json:"order_id"`
	LineItems  []LineItemInput   `json:"line_items"`  // 订单包含的商品列表
	Metadata   map[string]string `json:"metadata"`    // 扩展元数据，需包含 pipeline_ulid, stage_ulid 以便回调对账
	SuccessURL string            `json:"success_url"` // 支付成功后的跳转地址
	CancelURL  string            `json:"cancel_url"`  // 支付取消后的返回地址
}

type CreatePaymentRsp struct {
	OrderID           string `json:"order_id"`
	Status            string `json:"status"`
	StripeCheckoutURL string `json:"stripe_checkout_url"`
}

type GetPaymentStatusRsp struct {
	OrderID           string `json:"order_id"`
	RequestStatus     string `json:"request_status"`
	SessionStatus     string `json:"session_status"`
	PaymentStatus     string `json:"payment_status"`
	TotalAmount       int64  `json:"total_amount"`
	Currency          string `json:"currency"`
	StripeCheckoutURL string `json:"stripe_checkout_url"`
}

type CancelPaymentRsp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ===================== 发票 (Invoices) =====================

type QueryInvoiceRsp struct {
	InvoiceID     string  `json:"invoice_id"`
	PaymentID     string  `json:"payment_id"`
	RequestStatus string  `json:"request_status"`
	SubTotal      float64 `json:"sub_total"`
	TotalTax      float64 `json:"total_tax"`
	Total         float64 `json:"total"`
	ErrorMsg      string  `json:"error_msg"`
}

// ===================== 订单 (Orders) =====================

type OrderItem struct {
	OrderID       string  `json:"order_id"`
	ProductName   string  `json:"product_name"`
	Status        string  `json:"status"` // completed / pending / cancelled
	CreatedAt     string  `json:"created_at"`
	PaymentMethod string  `json:"payment_method"`
	Amount        float64 `json:"amount"`
}

type OrderListRsp struct {
	TotalOrders int         `json:"total_orders"`
	Completed   int         `json:"completed"`
	TotalAmount float64     `json:"total_amount"`
	Orders      []OrderItem `json:"orders"`
}

// ===================== 消息 (Messages) =====================

type MessageListInput struct {
	Limit  int `json:"limit"`   //  每页数量，默认 20
	LastId int `json:"last_id"` // 上一页最后的id
}

type MessageItem struct {
	Id         uint64 `json:"id"`          // 自增ID, 主键
	MessageId  string `json:"message_id"`  // 消息ID
	UserUlid   string `json:"user_id"`     // 考生ID
	TemplateId string `json:"template_id"` // 模板ID
	// payload 存储为 JSON 字符串，前端 Vue3 直接 JSON.parse 即可
	Payload   string               `json:"payload"`
	MsgType   gmsgpb.MsgType       `json:"msg_type"`
	MsgSource gmsgpb.MsgSource     `json:"msg_source"`
	SenderId  string               `json:"sender_id"`
	Status    gmsgpb.MessageStatus `json:"status"`
	// 时间使用 string 表达 (格式: "2026-04-27T16:00:00Z")
	CreatedAt string `json:"created_at"`
}

type MessageListRsp struct {
	Messages []MessageItem `json:"messages"`
	HasMore  bool          `json:"has_more"`
}

type MessageOperationInput struct {
	MessageIDs []string `json:"message_ids"`
}

// ===================== 会员 (Membership) =====================

type MembershipRsp struct {
	Level        string   `json:"level"` // student / certified / charterholder
	ExpiresAt    string   `json:"expires_at"`
	SubscribedAt string   `json:"subscribed_at"`
	AutoRenew    bool     `json:"auto_renew"`
	Benefits     []string `json:"benefits"`
}

// ===================== 证书 (Certificates) =====================

type ListCredentialsRsp struct {
	Credentials []CredentialsItem `json:"credentials"`
}

type CredentialsItem struct {
	CatalogId   string `json:"catalog_id,omitempty"`  // 资格类别ID ULID [required]
	Name        string `json:"name,omitempty"`        // 资格类别名称 [required]
	Description string `json:"description,omitempty"` // 资格类别描述 [required]

	Eligible         bool                    `json:"eligible,omitempty"`          // 是否持有有效资格
	CredentialStatus gcreds.CredentialStatus `json:"credential_status,omitempty"` // 当前资格状态；若无记录则为 UNSPECIFIED
	Message          string                  `json:"message,omitempty"`           // 人类可读的说明

	CertificateInfo // 证书信息 [optional]
}

type CertificateFileInfo struct {
	FileHash  string                    `json:"file_hash,omitempty"`  // SHA256 [required]
	FileName  string                    `json:"file_name,omitempty"`  // 文件名[required]
	FileType  gcreds.CredentialFileType `json:"file_type,omitempty"`  // 文件类型 [required]
	FileExt   string                    `json:"file_ext,omitempty"`   // 文件扩展名[required]
	FileSize  uint64                    `json:"file_size,omitempty"`  // 文件大小 [required]
	FileUsage string                    `json:"file_usage,omitempty"` // 文件用途, 如"front_view" [optional]
}

type CertificateInfo struct {
	CredUlid      string                  `json:"cred_id,omitempty"`      // 唯一ID ULID [required]
	CredGuid      string                  `json:"cred_guid,omitempty"`    // 跨版本业务唯一 ID
	CandidateUlid string                  `json:"candidate_id,omitempty"` // 考生逻辑 ID (ULID)
	Version       uint32                  `json:"version,omitempty"`      // 版本号[required]
	Status        gcreds.CredentialStatus `json:"status,omitempty"`       // 资格状态[required]
	Files         []CertificateFileInfo   `json:"files,omitempty"`        // 文件列表 [required]
	AuditorUlid   string                  `json:"auditor_id,omitempty"`   // 审核人ID ULID [optional]
	AuditRemark   string                  `json:"audit_remark,omitempty"` // 审核备注 [optional]
	ValidUntil    string                  `json:"valid_until,omitempty"`  // 有效期 RFC3339 格式字符串[optional]
	CreatedAt     string                  `json:"created_at,omitempty"`   // 创建时间 RFC3339 格式字符串[optional]
	Source        string                  `json:"source,omitempty"`       // 证书来源
}

type CertificateItem struct {
	CatalogId   string `json:"catalog_id,omitempty"`  // 资格类别ID ULID [required]
	Name        string `json:"name,omitempty"`        // 资格类别名称 [required]
	Description string `json:"description,omitempty"` // 资格类别描述 [required]

	CertificateInfo // 证书信息 [optional]
}

type ListCertificatesRsp struct {
	Certificates []CertificateItem `json:"certificates"`
}


type SystemRedDotsRsp struct {
	Applications uint32 `json:"applications"`
	Exams        uint32 `json:"exams"`
	Prog         uint32 `json:"prog"`
	Orders       uint32 `json:"orders"`
	Invoices     uint32 `json:"invoices"`
	Messages     uint32 `json:"messages"`
}
