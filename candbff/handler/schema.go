package handler

import (
	gcreds "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
)

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

type UserMeRsp struct {
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Phone       string   `json:"phone"`
	HomePhone   string   `json:"home_phone"`
	WorkPhone   string   `json:"work_phone"`
	Country     string   `json:"country"`
	Province    string   `json:"province"`
	City        string   `json:"city"`
	Region      string   `json:"region"`
	Location    string   `json:"location"`
	Address     []string `json:"address"`
	AddressText string   `json:"address_text"`
	PostalCode  string   `json:"postal_code"`
	Affiliation string   `json:"affiliation"`
	Title       string   `json:"title"`
	RealName    string   `json:"real_name"`
	Bio         string   `json:"bio"`
	Gender      string   `json:"gender"`
	Birthday    string   `json:"birthday"`
	Education   string   `json:"education"`
}

type UserProfileInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Phone       string `json:"phone"`
	HomePhone   string `json:"home_phone"`
	WorkPhone   string `json:"work_phone"`
	Country     string `json:"country"`
	Province    string `json:"province"`
	City        string `json:"city"`
	Address     string `json:"address"`
	PostalCode  string `json:"postal_code"`
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

type ListPipelinesRsp struct {
	Pipelines []PipelineConfig `json:"pipelines,omitempty"`
}

type PipelineDetailRsp struct {
	Config   PipelineConfig   `json:"config"`
	Instance PipelineSummary  `json:"instance,omitempty"`
	NextStep PipelineNextStep `json:"next_step,omitempty"`
}

type PipelineConfig struct {
	UnlockStripeProductId  string          `json:"unlock_stripe_product_id,omitempty"`
	UnlockStripePriceId    string          `json:"unlock_stripe_price_id,omitempty"`
	PackageStripeProductId string          `json:"package_stripe_product_id,omitempty"`
	PackageStripePriceId   string          `json:"package_stripe_price_id,omitempty"`
	PipelineUlid           string          `json:"pipeline_id,omitempty"`
	PipelineGuid           string          `json:"pipeline_guid,omitempty"` // ULID (业务唯一ID) [required]
	Version                uint32          `json:"version,omitempty"`       // 版本号 [required]
	Name                   string          `json:"name,omitempty"`          // 资格类别名称 [required]
	CategoryTips           string          `json:"category_tips,omitempty"` // 分类提示 [required]
	Description            string          `json:"description,omitempty"`
	ThumbnailObjectKey     string          `json:"thumbnail_object_key,omitempty"`
	ThumbnailFileHash      string          `json:"thumbnail_file_hash,omitempty"`
	UnlockFee              int64           `json:"unlock_fee,omitempty"`       // 解锁费用，单位：分 [required]
	PackageDiscount        int32           `json:"package_discount,omitempty"` // 套餐折扣，单位：基点（9500 = 95%）[required]
	UnlockQuals            []Qualification `json:"unlock_quals,omitempty"`     // 解锁条件 [required]
	CertQuals              []Qualification `json:"cert_quals,omitempty"`       // 证书要求 [required]
	Stages                 []StageConfig   `json:"stages,omitempty"`           // 阶段配置 [required]
	Status                 string          `json:"status,omitempty"`           // 资格状态 [required]
	IsCurrent              bool            `json:"is_current,omitempty"`       // 是否为当前版本 [required]
	CreatedAt              string          `json:"created_at,omitempty"`       // 创建时间 RFC3339 格式字符串 [optional]
	FinalQuals             []Qualification `json:"final_quals,omitempty"`      // 结业资格 [required]
	PurchaseCount          *int32          `json:"purchase_count,omitempty"`
}

type Qualification struct {
	QualId   string `json:"qual_id,omitempty"`   // 资格 ULID 对应gcred中的cred catalog id [required]
	NameHint string `json:"name_hint,omitempty"` // 资格名称 [required]
}

type StageConfig struct {
	RuntimeStatus string       `json:"runtime_status,omitempty"`
	StageUlid     string       `json:"stage_id,omitempty"`   // 管线阶段的 ULID [required]
	Name          string       `json:"name,omitempty"`       // 资格类别名称 [required]
	SortOrder     int32        `json:"sort_order,omitempty"` // 排序顺序 [required]
	Units         []UnitConfig `json:"units,omitempty"`      // 单元配置 [required]
}
type UnitConfig struct {
	RuntimeStatus            string   `json:"runtime_status,omitempty"`
	StripeProductId          string   `json:"stripe_product_id,omitempty"`
	StripePriceId            string   `json:"stripe_price_id,omitempty"`
	ExemptionStripeProductId string   `json:"exemption_stripe_product_id,omitempty"`
	ExemptionStripePriceId   string   `json:"exemption_stripe_price_id,omitempty"`
	RetakeStripeProductId    string   `json:"retake_stripe_product_id,omitempty"`
	RetakeStripePriceId      string   `json:"retake_stripe_price_id,omitempty"`
	GlmsCourseUlid           string   `json:"glms_course_id,omitempty"`
	AllowExemption           bool     `json:"allow_exemption,omitempty"`
	Program                  string   `json:"program,omitempty"`
	ExamUlid                 string   `json:"exam_id,omitempty"`
	FormCode                 string   `json:"form_code,omitempty"`
	UnitUlid                 string   `json:"unit_id,omitempty"`
	CourseUnitUlid           string   `json:"course_unit_ulid,omitempty"` // 单元实例ID (在runtime合并时注入)
	Name                     string   `json:"name,omitempty"`
	HasLearning              bool     `json:"has_learning,omitempty"`
	HasExam                  bool     `json:"has_exam,omitempty"`
	LearningMinutes          int32    `json:"learning_minutes,omitempty"`
	ProgramCode              string   `json:"program_code,omitempty"`
	ExamCode                 string   `json:"exam_code,omitempty"`
	ExamForm                 string   `json:"exam_form,omitempty"`
	BaseFee                  int64    `json:"base_fee,omitempty"`
	ExemptionQuals           []string `json:"exemption_quals,omitempty"`
	ExemptionAuditFee        int64    `json:"exemption_audit_fee,omitempty"`
	AllowRetake              bool     `json:"allow_retake,omitempty"`
	RetakeFee                int64    `json:"retake_fee,omitempty"`
}

type ListMyPipelinesRsp struct {
	List []PipelineSummary `json:"list,omitempty"` // pipeline 摘要列表 [required]
}

// PipelineSummary pipeline 閹芥顩?
type PipelineSummary struct {
	PipelineUlid      string  `json:"pipeline_ulid,omitempty"`    // pipeline 实例 ULID [required]
	CandidateUlid     string  `json:"candidate_ulid,omitempty"`   // 考生 ULID [required]
	PipelineCcUlid    string  `json:"pipeline_cc_ulid,omitempty"` // gcc 中 pipeline 配置 ULID [required]
	PipelineName      string  `json:"pipeline_name,omitempty"`
	Status            string  `json:"status,omitempty"`             // 资格状态 [required]
	CurrentStageUlid  string  `json:"current_stage_ulid,omitempty"` // 当前 stage 实例 ULID [required]
	Description       string  `json:"description,omitempty"`
	CurrentStageName  string  `json:"current_stage_name,omitempty"`
	Progress          float64 `json:"progress"` // 当前阶段的学习进度 (0-100)
	ProgressAvailable bool    `json:"progress_available,omitempty"`
	LmsProgress       uint32  `json:"lms_progress,omitempty"` //
	StartedAt         string  `json:"started_at,omitempty"`   // pipeline 开始时间，RFC3339 [optional]
	CompletedAt       string  `json:"completed_at,omitempty"` // pipeline 完成时间，RFC3339 [optional]
	CreatedAt         string  `json:"created_at,omitempty"`   // 创建时间 RFC3339 格式字符串 [optional]
}

type PipelineCreateRsp struct {
	PipelineUlid       string `json:"pipeline_ulid,omitempty"`        // pipeline 实例 ULID [required]
	PipelineStatus     string `json:"pipeline_status,omitempty"`      // pipeline 当前状态 [required]
	CurrentStageUlid   string `json:"current_stage_ulid,omitempty"`   // 当前 stage 实例 ULID [required]
	CurrentStageStatus string `json:"current_stage_status,omitempty"` // 当前 stage ״̬ [required]
	Message            string `json:"message,omitempty"`              // 人类可读的说明
}

type PipelineRuntimeRsp struct {
	Config             PipelineConfig   `json:"config"`
	Instance           PipelineSummary  `json:"instance,omitempty"`
	PipelineStatus     string           `json:"pipeline_status,omitempty"`
	CurrentStageUlid   string           `json:"current_stage_ulid,omitempty"`
	CurrentStageStatus string           `json:"current_stage_status,omitempty"`
	CurrentStageName   string           `json:"current_stage_name,omitempty"`
	CurrentUnitStatus  string           `json:"current_unit_status,omitempty"`
	NextStep           PipelineNextStep `json:"next_step,omitempty"`
}

type PipelineNextStep struct {
	Action           string `json:"action,omitempty"`
	Message          string `json:"message,omitempty"`
	StageUlid        string `json:"stage_id,omitempty"`
	StageName        string `json:"stage_name,omitempty"`
	CourseUnitUlid   string `json:"course_unit_ulid,omitempty"`
	CourseUnitCcUlid string `json:"course_unit_cc_ulid,omitempty"`
	CourseUlid       string `json:"course_id,omitempty"`
	Program          string `json:"program,omitempty"`
	ExamUlid         string `json:"exam_id,omitempty"`
	FormCode         string `json:"form_code,omitempty"`
	Status           string `json:"status,omitempty"`
	PipelineStatus   string `json:"pipeline_status,omitempty"`
	AllowRetake      bool   `json:"allow_retake,omitempty"`
	AllowExemption   bool   `json:"allow_exemption,omitempty"`
}

type PipelineTimelineRsp struct {
	Logs  []StatusTransitionLogSummary `json:"logs,omitempty"`
	Total int32                        `json:"total,omitempty"`
}

type StatusTransitionLogSummary struct {
	TransitionUlid string `json:"transition_ulid,omitempty"`
	EntityType     string `json:"entity_type,omitempty"`
	EntityUlid     string `json:"entity_ulid,omitempty"`
	FromStatus     string `json:"from_status,omitempty"`
	ToStatus       string `json:"to_status,omitempty"`
	ReasonCode     string `json:"reason_code,omitempty"`
	ReasonMessage  string `json:"reason_message,omitempty"`
	TriggerSource  string `json:"trigger_source,omitempty"`
	EventType      string `json:"event_type,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
}

type PurchasePipelineInput struct {
	PipelineCcULID string `json:"pipeline_cc_ulid"`
}

type GetAccessURLRsp struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
	Title     string `json:"title,omitempty"`
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

type PrepareRetakePaymentInput struct {
	CourseUnitCcULID string `json:"course_unit_cc_ulid"`
	RetriedCount     uint32 `json:"retried_count"`
	SuccessURL       string `json:"success_url"`
	CancelURL        string `json:"cancel_url"`
	BundleOrderUlid  string `json:"bundle_order_ulid"`
}

type PrepareRetakePaymentRsp struct {
	CourseRetakeOrderUlid string `json:"course_retake_order_ulid,omitempty"`
	CourseUnitUlid        string `json:"course_unit_ulid,omitempty"`
	CourseUnitStatus      string `json:"course_unit_status,omitempty"`
	OrderStatus           string `json:"order_status,omitempty"`
	PayOrderUlid          string `json:"pay_order_ulid,omitempty"`
	PaymentKey            string `json:"payment_key,omitempty"`
	PaymentRequired       bool   `json:"payment_required"`
	Paid                  bool   `json:"paid"`
	ReusedExisting        bool   `json:"reused_existing,omitempty"`
	Message               string `json:"message,omitempty"`
}

type ExamRetakeState struct {
	Eligible              bool   `json:"eligible"`
	Message               string `json:"message,omitempty"`
	NextRetriedCount      uint32 `json:"next_retried_count,omitempty"`
	RequiresPayment       bool   `json:"requires_payment"`
	PaymentFound          bool   `json:"payment_found"`
	PaymentPaid           bool   `json:"payment_paid"`
	CourseRetakeOrderUlid string `json:"course_retake_order_ulid,omitempty"`
	OrderStatus           string `json:"order_status,omitempty"`
	PayOrderUlid          string `json:"pay_order_ulid,omitempty"`
	Action                string `json:"action"`
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

type SyncCourseProgressRsp struct {
	Success               bool   `json:"success"`
	CourseStatus          string `json:"course_status,omitempty"`
	ProgressPercentage    uint32 `json:"progress_percentage,omitempty"`
	CompletedLessonsCount uint32 `json:"completed_lessons_count,omitempty"`
	PassedQuizzesCount    uint32 `json:"passed_quizzes_count,omitempty"`
}

type PipelineCourseRsp struct {
	CompleteCourse interface{}                 `json:"complete_course,omitempty"`
	QuizProgress   map[string]QuizProgressItem `json:"quiz_progress,omitempty"`
}

type QuizProgressItem struct {
	QuizID    string `json:"quiz_id,omitempty"`
	IsPassed  bool   `json:"is_passed"`
	Status    string `json:"status,omitempty"`
	AttemptID string `json:"attempt_id,omitempty"`
}

type MaterialListItem struct {
	ID              string  `json:"id"`
	CourseID        string  `json:"course_id,omitempty"`
	CourseTitle     string  `json:"course_title,omitempty"`
	Title           string  `json:"title"`
	Type            int32   `json:"type"`
	FileKey         string  `json:"file_key"`
	FileSize        int64   `json:"file_size"`
	FileHash        string  `json:"file_hash,omitempty"`
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
	CandidateUlid   string  `json:"candidate_id,omitempty"`      // 考生逻辑 ID (ULID)
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

type ExamListItem struct {
	ExamUlid             string           `json:"exam_id,omitempty"`
	PipelineUlid         string           `json:"pipeline_ulid,omitempty"`
	BundleOrderUlid      string           `json:"bundle_order_ulid,omitempty"`
	CourseUnitUlid       string           `json:"course_unit_ulid,omitempty"`
	CourseUnitCcUlid     string           `json:"course_unit_cc_ulid,omitempty"`
	CourseUnitStatus     string           `json:"course_unit_status,omitempty"`
	RetriedCount         uint32           `json:"retried_count,omitempty"`
	RetakeEligible       bool             `json:"retake_eligible,omitempty"`
	RetakeMessage        string           `json:"retake_message,omitempty"`
	NextRetriedCount     uint32           `json:"next_retried_count,omitempty"`
	Retake               *ExamRetakeState `json:"retake,omitempty"`
	ProgramCode          string           `json:"program_code,omitempty"`
	ExamCode             string           `json:"exam_code,omitempty"`
	ExamStatus           string           `json:"exam_status,omitempty"`
	ResultStatus         string           `json:"result_status,omitempty"`
	TotalScore           float64          `json:"total_score,omitempty"`
	IsPassed             bool             `json:"is_passed,omitempty"`
	CandidateFirstName   string           `json:"candidate_first_name,omitempty"`
	CandidateLastName    string           `json:"candidate_last_name,omitempty"`
	CandidateEmail       string           `json:"candidate_email,omitempty"`
	ConfirmationNumber   string           `json:"confirmation_number,omitempty"`
	AppointmentStartTime string           `json:"appointment_start_time,omitempty"`
	AppointmentEndTime   string           `json:"appointment_end_time,omitempty"`
	SiteName             string           `json:"site_name,omitempty"`
	LastTermurlTimestamp string           `json:"last_termurl_timestamp,omitempty"`
	LastTermurlType      string           `json:"last_termurl_type,omitempty"`
}

type ListExamsRsp struct {
	Exams []ExamListItem `json:"exams"`
	Total uint32         `json:"total"`
}

type CandidateSignupExamRsp struct {
	CourseUnitUlid   string `json:"course_unit_ulid,omitempty"`   // course unit 实例 ULID [required]
	CourseUnitStatus string `json:"course_unit_status,omitempty"` // course unit 当前状态 [required]
	Message          string `json:"message,omitempty"`            // 人类可读的说明
}

// CandidateApplyRetake
type CandidateApplyRetakeRsp struct {
	CourseUnitUlid   string `json:"course_unit_ulid,omitempty"`   // course unit 实例 ULID [required]
	CourseUnitStatus string `json:"course_unit_status,omitempty"` // course unit 当前状态 [required]
	Message          string `json:"message,omitempty"`            // 人类可读的说明
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
	HasResult       bool        `json:"has_result"`
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
	InvoiceNumber string  `json:"invoice_number"`
	Status        string  `json:"status"`
	SubTotal      float64 `json:"sub_total"`
	TotalTax      float64 `json:"total_tax"`
	Total         float64 `json:"total"`
	Currency      string  `json:"currency"`
	InvoiceUrl    string  `json:"invoice_url"`
}

// ===================== 订单 (Orders) =====================

type OrderItem struct {
	OrderID              string  `json:"order_id"`
	ProductName          string  `json:"product_name"`
	BizType              string  `json:"biz_type,omitempty"`
	BizRefUlid           string  `json:"biz_ref_ulid,omitempty"`
	Status               string  `json:"status"` // completed / pending / cancelled
	RawStatus            string  `json:"raw_status"`
	PipelineID           string  `json:"pipeline_id"`
	CreatedAt            string  `json:"created_at"`
	PaymentMethod        string  `json:"payment_method"`
	Amount               float64 `json:"amount"`
	Currency             string  `json:"currency"`
	PayOrderUlid         string  `json:"pay_order_ulid,omitempty"`
	PipelinePayOrderUlid string  `json:"pipeline_pay_order_ulid,omitempty"`
	CanViewInvoice       bool    `json:"can_view_invoice"`
	CanCancel            bool    `json:"can_cancel"`
}

type PurchasePipelineReq struct {
	PaymentMode                     string `json:"payment_mode"` // FULL_PIPELINE or BY_STAGE
	CandidateSelectedExemptionsJson string `json:"candidate_selected_exemptions_json"`
}

type CreateBundleOrderReq struct {
	PaymentMode            string `json:"payment_mode"`
	SelectedExemptionsJson string `json:"selected_exemptions_json"`
	BundleOrderUlid        string `json:"bundle_order_ulid,omitempty"`
}

type UnlockPipelineInBundleReq struct {
	PipelineCcUlid string `json:"pipeline_cc_ulid"`
}

type PipelineExemptionOptionsRsp struct {
	Stages []PipelineExemptionStage `json:"stages"`
}

type PipelineExemptionStage struct {
	Index     int32                   `json:"index"`
	StageUlid string                  `json:"stage_id"`
	StageName string                  `json:"stage_name,omitempty"`
	SortOrder int32                   `json:"sort_order,omitempty"`
	Units     []PipelineExemptionUnit `json:"units"`
}

type PipelineExemptionUnit struct {
	UnitUlid       string                  `json:"unit_id"`
	UnitName       string                  `json:"unit_name,omitempty"`
	AllowExemption bool                    `json:"allow_exemption"`
	ExemptionQuals []PipelineExemptionQual `json:"exemption_quals"`
	Qualified      bool                    `json:"qualified"`
	Message        string                  `json:"message,omitempty"`
}

type PipelineExemptionQual struct {
	QualId           string `json:"qual_id"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	Category         string `json:"category,omitempty"`
	Eligible         bool   `json:"eligible"`
	CredentialStatus string `json:"credential_status,omitempty"`
	Message          string `json:"message,omitempty"`
}

type CreateCredentialApplicationOrderReq struct {
	PipelineCcUlid string   `json:"pipeline_cc_ulid"`
	BundleUlid     string   `json:"bundle_ulid"`
	QualUlids      []string `json:"qual_ulids"`
	LegacyQualIDs  []string `json:"qual_ids,omitempty"`
}

type PurchasePipelineRsp struct {
	PipelineOrderUlid    string `json:"pipeline_order_ulid"`
	OrderStatus          string `json:"order_status"`
	ReviewOrderUlid      string `json:"review_order_ulid"`
	PipelinePayOrderUlid string `json:"pipeline_pay_order_ulid"`
	PaymentUrl           string `json:"payment_url"`
	PaymentKey           string `json:"payment_key,omitempty"`
	ReusedExisting       bool   `json:"reused_existing"`
	Message              string `json:"message,omitempty"`
}

type InitiatePaymentReq struct {
	BizType     string   `json:"biz_type"`
	BizRefUlid  string   `json:"biz_ref_ulid"`
	SuccessUrl  string   `json:"success_url"`
	CancelUrl   string   `json:"cancel_url"`
	CouponCodes []string `json:"coupon_codes"`
}

type OrderListRsp struct {
	TotalOrders int         `json:"total_orders"`
	Completed   int         `json:"completed"`
	TotalAmount float64     `json:"total_amount"`
	Page        int         `json:"page"`
	PageSize    int         `json:"page_size"`
	TotalPages  int         `json:"total_pages"`
	Orders      []OrderItem `json:"orders"`
}

type CancelOrderRsp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	OrderID string `json:"order_id,omitempty"`
}

// ===================== 消息 (Messages) =====================

type MessageListInput struct {
	Limit  int `json:"limit"`   // 每页数量，默认 20
	LastId int `json:"last_id"` // 上一页最后的id
}

type MessageItem struct {
	Id         uint64 `json:"id"`          // 自增ID, 主键
	MessageId  string `json:"message_id"`  // 消息ID
	UserUlid   string `json:"user_id"`     // 考生ID
	TemplateId string `json:"template_id"` // 模板ID
	// payload 存储为 JSON 字符串，前端 Vue3 直接 JSON.parse 即可
	Payload         string               `json:"payload"`
	TemplatePayload string               `json:"template_payload,omitempty"`
	Title           string               `json:"title,omitempty"`
	Content         string               `json:"content,omitempty"`
	MsgType         gmsgpb.MsgType       `json:"msg_type"`
	MsgSource       gmsgpb.MsgSource     `json:"msg_source"`
	SenderId        string               `json:"sender_id"`
	Status          gmsgpb.MessageStatus `json:"status"`
	// 时间使用 string 表达 (格式: "2026-04-27T16:00:00Z")
	CreatedAt string `json:"created_at"`
}

type MessageListRsp struct {
	Messages []MessageItem `json:"messages"`
	HasMore  bool          `json:"has_more"`
}

type MessageUnreadCountRsp struct {
	UnreadCount uint32 `json:"unread_count"`
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
	FileName  string                    `json:"file_name,omitempty"`  // 文件名 [required]
	FileType  gcreds.CredentialFileType `json:"file_type,omitempty"`  // 文件类型 [required]
	FileExt   string                    `json:"file_ext,omitempty"`   // 文件扩展名 [required]
	FileSize  uint64                    `json:"file_size,omitempty"`  // 文件大小 [required]
	FileUsage string                    `json:"file_usage,omitempty"` // 文件用途, 如 "front_view" [optional]
	ViewUrl   string                    `json:"view_url,omitempty"`   //
}

type CertificateInfo struct {
	CredUlid      string                  `json:"cred_id,omitempty"`      // ΨһID ULID [required]
	CredGuid      string                  `json:"cred_guid,omitempty"`    // 跨版本业务唯一 ID
	CandidateUlid string                  `json:"candidate_id,omitempty"` // 考生逻辑 ID (ULID)
	Version       uint32                  `json:"version,omitempty"`      // 版本号 [required]
	Status        gcreds.CredentialStatus `json:"status,omitempty"`       // 资格状态 [required]
	Files         []CertificateFileInfo   `json:"files,omitempty"`        // 文件列表 [required]
	AuditorUlid   string                  `json:"auditor_id,omitempty"`   // 审核人ID ULID [optional]
	AuditRemark   string                  `json:"audit_remark,omitempty"` // 审核备注 [optional]
	ValidUntil    string                  `json:"valid_until,omitempty"`  // 有效期 RFC3339 格式字符串 [optional]
	CreatedAt     string                  `json:"created_at,omitempty"`   // 创建时间 RFC3339 格式字符串 [optional]
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

type SubmitQuizInput struct {
	Submissions []QuestionSubmissionInput `json:"submissions"`
}

type QuestionSubmissionInput struct {
	QuestionId        string   `json:"question_id"`
	SelectedOptionIds []string `json:"selected_option_ids"`
}

// AppointmentResponse 第三方考试系统回调的XML解析结构
type AppointmentResponse struct {
	Appointments []Appointment `xml:"Appointment"`
}

type Appointment struct {
	Outcome            string `xml:"Outcome"`
	ConfirmationNumber string `xml:"ConfirmationNumber"`
	StartDateTime      string `xml:"StartDateTime"`
}
