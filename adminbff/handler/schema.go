package handler

import (
	gcreds "github.com/LMF709268224/cftpproto/gcreds"
	gmsgpb "github.com/LMF709268224/cftpproto/gmsg"
	"github.com/LMF709268224/cftpproto/gprog"
	gprogpb "github.com/LMF709268224/cftpproto/gprog"
)

// ===================== 娉ㄥ唽鐧诲綍 (Auth) =====================

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

// ===================== 鐢ㄦ埛 (User) =====================

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

// ===================== 鍟嗗煄 =====================

type ListPipelinesRsp struct {
	Pipelines []PipelineConfig `json:"pipelines,omitempty"`
}

type PipelineDetailRsp struct {
	Config   PipelineConfig  `json:"config"`
	Instance PipelineSummary `json:"instance,omitempty"` // 濡傛灉宸茶喘涔帮紝鍒欏寘鍚疄渚嬬姸鎬?
}

type PipelineConfig struct {
	UnlockStripeProductId  string          `json:"unlock_stripe_product_id,omitempty"`
	UnlockStripePriceId    string          `json:"unlock_stripe_price_id,omitempty"`
	PackageStripeProductId string          `json:"package_stripe_product_id,omitempty"`
	PackageStripePriceId   string          `json:"package_stripe_price_id,omitempty"`
	PipelineId             string          `json:"pipeline_id,omitempty"`      // ULID (鐗堟湰鍞竴ID) [required]
	PipelineGuid           string          `json:"pipeline_guid,omitempty"`    // ULID (涓氬姟鍞竴ID) [required]
	Version                uint32          `json:"version,omitempty"`          // 鐗堟湰鍙?[required]
	Name                   string          `json:"name,omitempty"`             // 绠＄嚎鍚嶇О [required]
	CategoryTips           string          `json:"category_tips,omitempty"`    // 鍒嗙被鎻愮ず [required]
	UnlockFee              int64           `json:"unlock_fee,omitempty"`       // 瑙ｉ攣璐圭敤锛屽崟浣嶏細鍒?[required]
	PackageDiscount        int32           `json:"package_discount,omitempty"` // 濂楅鎶樻墸锛屽崟浣嶏細鍩虹偣锛?500 = 95%锛塠required]
	UnlockQuals            []Qualification `json:"unlock_quals,omitempty"`     // 瑙ｉ攣鏉′欢 [required]
	CertQuals              []Qualification `json:"cert_quals,omitempty"`       // 璇佷功瑕佹眰 [required]
	Stages                 []StageConfig   `json:"stages,omitempty"`           // 闃舵閰嶇疆 [required]
	Status                 string          `json:"status,omitempty"`           // 鐘舵€?[required]
	IsCurrent              bool            `json:"is_current,omitempty"`       // 鏄惁涓哄綋鍓嶇増鏈?[required]
	CreatedAt              string          `json:"created_at,omitempty"`       // 鍒涘缓鏃堕棿 [required]
	FinalQuals             []Qualification `json:"final_quals,omitempty"`      // 缁撲笟璧勬牸 [required]
}

type Qualification struct {
	QualId   string `json:"qual_id,omitempty"`   // 璧勬牸 ULID 瀵瑰簲gcred涓殑cred catalog id [required]
	NameHint string `json:"name_hint,omitempty"` // 璧勬牸鍚嶇О [required]
}

type StageConfig struct {
	StageId   string       `json:"stage_id,omitempty"`   // 绠＄嚎闃舵鐨?ULID [required]
	Name      string       `json:"name,omitempty"`       // 闃舵鍚嶇О [required]
	SortOrder int32        `json:"sort_order,omitempty"` // 鎺掑簭椤哄簭 [required]
	Units     []UnitConfig `json:"units,omitempty"`      // 鍗曞厓閰嶇疆 [required]
}

type UnitConfig struct {
	StripeProductId          string `json:"stripe_product_id,omitempty"`
	StripePriceId            string `json:"stripe_price_id,omitempty"`
	ExemptionStripeProductId string `json:"exemption_stripe_product_id,omitempty"`
	ExemptionStripePriceId   string `json:"exemption_stripe_price_id,omitempty"`
	RetakeStripeProductId    string `json:"retake_stripe_product_id,omitempty"`
	RetakeStripePriceId      string `json:"retake_stripe_price_id,omitempty"`
	GlmsCourseId             string `json:"glms_course_id,omitempty"`
	UnitId                   string `json:"unit_id,omitempty"`          // 闃舵鍗曞厓(璇剧▼) ULID & GLMS ID [required]
	Name                     string `json:"name,omitempty"`             // 闃舵鍗曞厓鍚嶇О [required]
	HasLearning              bool   `json:"has_learning,omitempty"`     // 鏄惁鏈夊涔?[required]
	HasExam                  bool   `json:"has_exam,omitempty"`         // 鏄惁鏈夎€冭瘯 [required]
	LearningMinutes          int32  `json:"learning_minutes,omitempty"` // 璇炬椂瑕佹眰锛屼互鍒嗛挓涓哄崟浣?[required]
	// 鑰冭瘯鎵佸钩鍖栧弬鏁?
	ProgramCode       string   `json:"program_code,omitempty"`        // 鑰冭瘯鍙傛暟锛氳绋嬩唬鍙?[required]
	ExamCode          string   `json:"exam_code,omitempty"`           // 鑰冭瘯鍙傛暟锛氳€冭瘯浠ｅ彿 [required]
	ExamForm          string   `json:"exam_form,omitempty"`           // 鑰冭瘯鍙傛暟锛氳€冭瘯褰㈠紡 [required]
	BaseFee           int64    `json:"base_fee,omitempty"`            // 鍩烘湰璐癸紝鍗曚綅锛氬垎 [required]
	ExemptionQuals    []string `json:"exemption_quals,omitempty"`     // 璧勬牸 ULID 鍒楄〃  [required]
	ExemptionAuditFee int64    `json:"exemption_audit_fee,omitempty"` // 璧勬牸瀹℃牳璐癸紝鍗曚綅锛氬垎 [required]
	AllowRetake       bool     `json:"allow_retake,omitempty"`        // 鏄惁鍏佽閲嶈€?[required]
	RetakeFee         int64    `json:"retake_fee,omitempty"`          // 閲嶈€冭垂鐢紝鍗曚綅锛氬垎 [required]
}

// ===================== 璇剧▼涓庤璇?(Courses & Pipelines) =====================

type ListMyPipelinesRsp struct {
	List []PipelineSummary `json:"list,omitempty"` // pipeline 鎽樿鍒楄〃 [required]
}

// PipelineSummary pipeline 鎽樿
type PipelineSummary struct {
	PipelineUlid     string                 `json:"pipeline_ulid,omitempty"`      // pipeline 瀹炰緥 ULID [required]
	CandidateUlid    string                 `json:"candidate_ulid,omitempty"`     // 鑰冪敓 ULID [required]
	PipelineCcUlid   string                 `json:"pipeline_cc_ulid,omitempty"`   // gcc 涓?pipeline 閰嶇疆 ULID [required]
	Status           gprogpb.PipelineStatus `json:"status,omitempty"`             // pipeline 褰撳墠鐘舵€?[required]
	CurrentStageUlid string                 `json:"current_stage_ulid,omitempty"` // 褰撳墠 stage 瀹炰緥 ULID [optional]
	Progress         float64                `json:"progress"`                     // 褰撳墠闃舵鐨勫涔犺繘搴?(0-100)
	StartedAt        string                 `json:"started_at,omitempty"`         // pipeline 寮€濮嬫椂闂达紝RFC3339 [optional]
	CompletedAt      string                 `json:"completed_at,omitempty"`       // pipeline 瀹屾垚鏃堕棿锛孯FC3339 [optional]
	CreatedAt        string                 `json:"created_at,omitempty"`         // pipeline 鍒涘缓鏃堕棿锛孯FC3339 [required]
}

type PipelineCreateRsp struct {
	PipelineUlid       string                 `json:"pipeline_ulid,omitempty"`        // pipeline 瀹炰緥 ULID [required]
	PipelineStatus     gprogpb.PipelineStatus `json:"pipeline_status,omitempty"`      // pipeline 褰撳墠鐘舵€?[required]
	CurrentStageUlid   string                 `json:"current_stage_ulid,omitempty"`   // 褰撳墠 stage 瀹炰緥 ULID锛岃嫢鏃犲垯涓虹┖ [optional]
	CurrentStageStatus gprogpb.StageStatus    `json:"current_stage_status,omitempty"` // 褰撳墠 stage 鐘舵€侊紝鑻ユ棤鍒欎负 UNSPECIFIED [required]
	Message            string                 `json:"message,omitempty"`              // 浜虹被鍙璇存槑 [required]
}

type AdminTerminatePipelineRsp struct {
	PipelineUlid   string                 `json:"pipeline_ulid,omitempty"`   // pipeline 瀹炰緥 ULID [required]
	PipelineStatus gprogpb.PipelineStatus `json:"pipeline_status,omitempty"` // pipeline 褰撳墠鐘舵€?[required]
	Message        string                 `json:"message,omitempty"`         // 浜虹被鍙璇存槑 [required]
}

type AdminTriggerProgNextStageRsp struct {
	PipelineUlid       string                 `json:"pipeline_ulid,omitempty"`        // pipeline 瀹炰緥 ULID [required]
	PipelineStatus     gprogpb.PipelineStatus `json:"pipeline_status,omitempty"`      // pipeline 褰撳墠鐘舵€?[required]
	CurrentStageUlid   string                 `json:"current_stage_ulid,omitempty"`   // 褰撳墠 stage 瀹炰緥 ULID [required]
	CurrentStageStatus gprogpb.StageStatus    `json:"current_stage_status,omitempty"` // 褰撳墠 stage 鐘舵€?[required]
	Message            string                 `json:"message,omitempty"`              // 浜虹被鍙璇存槑 [required]
}

type AdminForceCourseCompletedRsp struct {
	CourseUnitUlid   string                   `json:"course_unit_ulid,omitempty"`   // course unit 瀹炰緥 ULID [required]
	CourseUnitStatus gprogpb.CourseUnitStatus `json:"course_unit_status,omitempty"` // course unit 褰撳墠鐘舵€?[required]
	Message          string                   `json:"message,omitempty"`            // 浜虹被鍙璇存槑 [required]
}

type AdminForceCourseSignupExamRsp struct {
	CourseUnitUlid   string                   `json:"course_unit_ulid,omitempty"`   // course unit 瀹炰緥 ULID [required]
	CourseUnitStatus gprogpb.CourseUnitStatus `json:"course_unit_status,omitempty"` // course unit 褰撳墠鐘舵€?[required]
	Message          string                   `json:"message,omitempty"`            // 浜虹被鍙璇存槑 [required]
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

// ===================== 棣栭〉 (Dashboard) =====================

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

// ===================== 瀛︿範杩涘害 (Progress) =====================

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
	Records []ProgressRecord `json:"records,omitempty"` // 杩涘害璁板綍鍒楄〃 [required]
}

type ProgressRecord struct {
	CandidateId     string  `json:"candidate_id,omitempty"`      // 鑰冪敓ID [required]
	MaterialId      string  `json:"material_id,omitempty"`       // 璧勬枡ID [required]
	CoursePackageId string  `json:"course_package_id,omitempty"` // 璧勬枡鍖匢D [required]
	ProgressType    string  `json:"progress_type,omitempty"`     // 杩涘害绫诲瀷 [required]
	ProgressValue   float64 `json:"progress_value,omitempty"`    // 杩涘害鍊硷細瑙嗛涓虹鏁帮紝鏂囨。涓虹櫨鍒嗘瘮 [required]
	RecordedAt      string  `json:"recorded_at,omitempty"`       // 璁板綍鏃堕棿 RFC3339 [required]
}

// ===================== 鑰冭瘯 (Exams) =====================

type ExamResultDetailRsp struct {
	ExamId           string  `json:"exam_id,omitempty"`
	TotalScore       float64 `json:"total_score,omitempty"`
	IsPassed         bool    `json:"is_passed,omitempty"`
	ScoreDetailsJson string  `json:"score_details_json,omitempty"`
}

type CandidateSignupExamRsp struct {
	CourseUnitUlid   string                 `json:"course_unit_ulid,omitempty"`   // course unit 瀹炰緥 ULID [required]
	CourseUnitStatus gprog.CourseUnitStatus `json:"course_unit_status,omitempty"` // course unit 褰撳墠鐘舵€?[required]
	Message          string                 `json:"message,omitempty"`            // 浜虹被鍙璇存槑 [required]
}

// CandidateApplyRetake
type CandidateApplyRetakeRsp struct {
	CourseUnitUlid   string                 `json:"course_unit_ulid,omitempty"`   // course unit 瀹炰緥 ULID [required]
	CourseUnitStatus gprog.CourseUnitStatus `json:"course_unit_status,omitempty"` // course unit 褰撳墠鐘舵€?[required]
	Message          string                 `json:"message,omitempty"`            // 浜虹被鍙璇存槑 [required]
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

// ===================== 鏀粯 (Payments) =====================

type LineItemInput struct {
	Name        string `json:"name"`        // 鍟嗗搧鍚嶇О锛屽 "Level 1 璁よ瘉鑰冭瘯璐圭敤"
	Description string `json:"description"` // 鍟嗗搧璇︾粏鎻忚堪
	UnitAmount  int64  `json:"unit_amount"` // 鍟嗗搧鍗曚环锛堝崟浣嶏細鍒嗭紝濡?100 浠ｈ〃 1.00 鍏冿級
	Quantity    int64  `json:"quantity"`    // 璐拱鏁伴噺
	Currency    string `json:"currency"`    // 璐у竵甯佺锛屽 "usd", "cny"
}

type CreatePaymentInput struct {
	// OrderID    string            `json:"order_id"`
	LineItems  []LineItemInput   `json:"line_items"`  // 璁㈠崟鍖呭惈鐨勫晢鍝佸垪琛?
	Metadata   map[string]string `json:"metadata"`    // 鎵╁睍鍏冩暟鎹紝闇€鍖呭惈 pipeline_ulid, stage_ulid 浠ヤ究鍥炶皟瀵硅处
	SuccessURL string            `json:"success_url"` // 鏀粯鎴愬姛鍚庣殑璺宠浆鍦板潃
	CancelURL  string            `json:"cancel_url"`  // 鏀粯鍙栨秷鍚庣殑杩斿洖鍦板潃
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

// ===================== 鍙戠エ (Invoices) =====================

type QueryInvoiceRsp struct {
	InvoiceID     string  `json:"invoice_id"`
	PaymentID     string  `json:"payment_id"`
	RequestStatus string  `json:"request_status"`
	SubTotal      float64 `json:"sub_total"`
	TotalTax      float64 `json:"total_tax"`
	Total         float64 `json:"total"`
	ErrorMsg      string  `json:"error_msg"`
}

// ===================== 璁㈠崟 (Orders) =====================

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

// ===================== 娑堟伅 (Messages) =====================

type MessageListInput struct {
	Limit  int `json:"limit"`   //  姣忛〉鏁伴噺锛岄粯璁?20
	LastId int `json:"last_id"` // 涓婁竴椤垫渶鍚庣殑id
}

type MessageItem struct {
	Id         uint64 `json:"id"`          // 鑷ID, 涓婚敭
	MessageId  string `json:"message_id"`  // 娑堟伅ID
	UserId     string `json:"user_id"`     // 鑰冪敓ID
	TemplateId string `json:"template_id"` // 妯℃澘ID
	// payload 瀛樺偍涓?JSON 瀛楃涓诧紝鍓嶇 Vue3 鐩存帴 JSON.parse 鍗冲彲
	Payload   string               `json:"payload"`
	MsgType   gmsgpb.MsgType       `json:"msg_type"`
	MsgSource gmsgpb.MsgSource     `json:"msg_source"`
	SenderId  string               `json:"sender_id"`
	Status    gmsgpb.MessageStatus `json:"status"`
	// 鏃堕棿浣跨敤 string 琛ㄨ揪 (鏍煎紡: "2026-04-27T16:00:00Z")
	CreatedAt string `json:"created_at"`
}

type MessageListRsp struct {
	Messages []MessageItem `json:"messages"`
	HasMore  bool          `json:"has_more"`
}

type MessageOperationInput struct {
	MessageIDs []string `json:"message_ids"`
}

// ===================== 浼氬憳 (Membership) =====================

type MembershipRsp struct {
	Level        string   `json:"level"` // student / certified / charterholder
	ExpiresAt    string   `json:"expires_at"`
	SubscribedAt string   `json:"subscribed_at"`
	AutoRenew    bool     `json:"auto_renew"`
	Benefits     []string `json:"benefits"`
}

// ===================== 璇佷功 (Certificates) =====================

type ListCredentialsRsp struct {
	Credentials []CredentialsItem `json:"credentials"`
}

type CredentialsItem struct {
	CatalogId   string `json:"catalog_id,omitempty"`  // 璧勬牸绫诲埆ID ULID [required]
	Name        string `json:"name,omitempty"`        // 璧勬牸绫诲埆鍚嶇О [required]
	Description string `json:"description,omitempty"` // 璧勬牸绫诲埆鎻忚堪 [required]

	Eligible         bool                    `json:"eligible,omitempty"`          // 鏄惁鎸佹湁鏈夋晥璧勬牸
	CredentialStatus gcreds.CredentialStatus `json:"credential_status,omitempty"` // 褰撳墠璧勬牸鐘舵€侊紱鑻ユ棤璁板綍鍒欎负 UNSPECIFIED
	Message          string                  `json:"message,omitempty"`           // 浜虹被鍙鐨勮鏄?

	CertificateInfo // 璇佷功淇℃伅 [optional]
}

type CertificateFileInfo struct {
	FileHash  string                    `json:"file_hash,omitempty"`  // SHA256 [required]
	FileName  string                    `json:"file_name,omitempty"`  // 鏂囦欢鍚?[required]
	FileType  gcreds.CredentialFileType `json:"file_type,omitempty"`  // 鏂囦欢绫诲瀷 [required]
	FileExt   string                    `json:"file_ext,omitempty"`   // 鏂囦欢鎵╁睍鍚?[required]
	FileSize  uint64                    `json:"file_size,omitempty"`  // 鏂囦欢澶у皬 [required]
	FileUsage string                    `json:"file_usage,omitempty"` // 鏂囦欢鐢ㄩ€? 濡?"front_view" [optional]
}

type CertificateInfo struct {
	CredId      string                  `json:"cred_id,omitempty"`      // 鍞竴ID ULID [required]
	CredGuid    string                  `json:"cred_guid,omitempty"`    // 璺ㄧ増鏈笟鍔″敮涓€ ID
	CandidateId string                  `json:"candidate_id,omitempty"` // 鑰冪敓閫昏緫 ID (ULID)
	Version     uint32                  `json:"version,omitempty"`      // 鐗堟湰鍙?[required]
	Status      gcreds.CredentialStatus `json:"status,omitempty"`       // 璧勬牸鐘舵€?[required]
	Files       []CertificateFileInfo   `json:"files,omitempty"`        // 鏂囦欢鍒楄〃 [required]
	AuditorId   string                  `json:"auditor_id,omitempty"`   // 瀹℃牳浜篒D ULID [optional]
	AuditRemark string                  `json:"audit_remark,omitempty"` // 瀹℃牳澶囨敞 [optional]
	ValidUntil  string                  `json:"valid_until,omitempty"`  // 鏈夋晥鏈?RFC3339 鏍煎紡瀛楃涓?[optional]
	CreatedAt   string                  `json:"created_at,omitempty"`   // 鍒涘缓鏃堕棿 RFC3339 鏍煎紡瀛楃涓?[optional]
}

type CertificateItem struct {
	CatalogId   string `json:"catalog_id,omitempty"`  // 璧勬牸绫诲埆ID ULID [required]
	Name        string `json:"name,omitempty"`        // 璧勬牸绫诲埆鍚嶇О [required]
	Description string `json:"description,omitempty"` // 璧勬牸绫诲埆鎻忚堪 [required]

	CertificateInfo // 璇佷功淇℃伅 [optional]
}

type ListCertificatesRsp struct {
	Certificates []CertificateItem `json:"certificates"`
}
