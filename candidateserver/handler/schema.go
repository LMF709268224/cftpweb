package handler

import (
	gcreds "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	"github.com/afnandelfin620-star/cftptest/cftp/gprog"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
)

// ===================== 濞夈劌鍞介惂璇茬秿 (Auth) =====================

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

// ===================== 閻劍鍩?(User) =====================

type UserMeRsp struct {
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Phone       string   `json:"phone"`
	Region      string   `json:"region"`
	Location    string   `json:"location"`
	Address     []string `json:"address"`
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

// ===================== 閸熷棗鐓?=====================

type ListPipelinesRsp struct {
	Pipelines []PipelineConfig `json:"pipelines,omitempty"`
}

type PipelineDetailRsp struct {
	Config   PipelineConfig   `json:"config"`
	Instance PipelineSummary  `json:"instance,omitempty"` // ????????????????????
	NextStep PipelineNextStep `json:"next_step,omitempty"`
}

type PipelineConfig struct {
	UnlockStripeProductId  string          `json:"unlock_stripe_product_id,omitempty"`
	UnlockStripePriceId    string          `json:"unlock_stripe_price_id,omitempty"`
	PackageStripeProductId string          `json:"package_stripe_product_id,omitempty"`
	PackageStripePriceId   string          `json:"package_stripe_price_id,omitempty"`
	PipelineId             string          `json:"pipeline_id,omitempty"`   // ULID (閻楀牊婀伴崬顖欑ID) [required]
	PipelineGuid           string          `json:"pipeline_guid,omitempty"` // ULID (娑撴艾濮熼崬顖欑ID) [required]
	Version                uint32          `json:"version,omitempty"`       // 閻楀牊婀伴崣?[required]
	Name                   string          `json:"name,omitempty"`          // 缁狅紕鍤庨崥宥囆?[required]
	CategoryTips           string          `json:"category_tips,omitempty"` // 閸掑棛琚幓鎰仛 [required]
	ThumbnailObjectKey     string          `json:"thumbnail_object_key,omitempty"`
	ThumbnailFileHash      string          `json:"thumbnail_file_hash,omitempty"`
	UnlockFee              int64           `json:"unlock_fee,omitempty"`       // 鐟欙綁鏀ｇ拹鍦暏閿涘苯宕熸担宥忕窗閸?[required]
	PackageDiscount        int32           `json:"package_discount,omitempty"` // 婵傛顦甸幎妯诲⒏閿涘苯宕熸担宥忕窗閸╄櫣鍋ｉ敍?500 = 95%閿涘required]
	UnlockQuals            []Qualification `json:"unlock_quals,omitempty"`     // 鐟欙綁鏀ｉ弶鈥叉 [required]
	CertQuals              []Qualification `json:"cert_quals,omitempty"`       // 鐠囦椒鍔熺憰浣圭湴 [required]
	Stages                 []StageConfig   `json:"stages,omitempty"`           // 闂冭埖顔岄柊宥囩枂 [required]
	Status                 string          `json:"status,omitempty"`           // 閻樿埖鈧?[required]
	IsCurrent              bool            `json:"is_current,omitempty"`       // 閺勵垰鎯佹稉鍝勭秼閸撳秶澧楅張?[required]
	CreatedAt              string          `json:"created_at,omitempty"`       // 閸掓稑缂撻弮鍫曟？ [required]
	FinalQuals             []Qualification `json:"final_quals,omitempty"`      // 缂佹挷绗熺挧鍕壐 [required]
	PurchaseCount          *int32          `json:"purchase_count,omitempty"`
}

type Qualification struct {
	QualId   string `json:"qual_id,omitempty"`   // 鐠у嫭鐗?ULID 鐎电懓绨瞘cred娑擃厾娈慶red catalog id [required]
	NameHint string `json:"name_hint,omitempty"` // 鐠у嫭鐗搁崥宥囆?[required]
}

type StageConfig struct {
	RuntimeStatus gprogpb.StageStatus `json:"runtime_status,omitempty"`
	StageId       string              `json:"stage_id,omitempty"`   // 缁狅紕鍤庨梼鑸殿唽閻?ULID [required]
	Name          string              `json:"name,omitempty"`       // 闂冭埖顔岄崥宥囆?[required]
	SortOrder     int32               `json:"sort_order,omitempty"` // 閹烘帒绨い鍝勭碍 [required]
	Units         []UnitConfig        `json:"units,omitempty"`      // 閸楁洖鍘撻柊宥囩枂 [required]
}
type UnitConfig struct {
	RuntimeStatus            gprog.CourseUnitStatus `json:"runtime_status,omitempty"`
	StripeProductId          string                 `json:"stripe_product_id,omitempty"`
	StripePriceId            string                 `json:"stripe_price_id,omitempty"`
	ExemptionStripeProductId string                 `json:"exemption_stripe_product_id,omitempty"`
	ExemptionStripePriceId   string                 `json:"exemption_stripe_price_id,omitempty"`
	RetakeStripeProductId    string                 `json:"retake_stripe_product_id,omitempty"`
	RetakeStripePriceId      string                 `json:"retake_stripe_price_id,omitempty"`
	GlmsCourseId             string                 `json:"glms_course_id,omitempty"`
	AllowExemption           bool                   `json:"allow_exemption,omitempty"`
	Program                  string                 `json:"program,omitempty"`
	ExamId                   string                 `json:"exam_id,omitempty"`
	FormCode                 string                 `json:"form_code,omitempty"`
	UnitId                   string                 `json:"unit_id,omitempty"`
	Name                     string                 `json:"name,omitempty"`
	HasLearning              bool                   `json:"has_learning,omitempty"`
	HasExam                  bool                   `json:"has_exam,omitempty"`
	LearningMinutes          int32                  `json:"learning_minutes,omitempty"`
	ProgramCode              string                 `json:"program_code,omitempty"`
	ExamCode                 string                 `json:"exam_code,omitempty"`
	ExamForm                 string                 `json:"exam_form,omitempty"`
	BaseFee                  int64                  `json:"base_fee,omitempty"`
	ExemptionQuals           []string               `json:"exemption_quals,omitempty"`
	ExemptionAuditFee        int64                  `json:"exemption_audit_fee,omitempty"`
	AllowRetake              bool                   `json:"allow_retake,omitempty"`
	RetakeFee                int64                  `json:"retake_fee,omitempty"`
}

type ListMyPipelinesRsp struct {
	List []PipelineSummary `json:"list,omitempty"` // pipeline 閹芥顩﹂崚妤勩€?[required]
}

// PipelineSummary pipeline 閹芥顩?
type PipelineSummary struct {
	PipelineUlid      string                 `json:"pipeline_ulid,omitempty"`    // pipeline 鐎圭偘绶?ULID [required]
	CandidateUlid     string                 `json:"candidate_ulid,omitempty"`   // 閼板啰鏁?ULID [required]
	PipelineCcUlid    string                 `json:"pipeline_cc_ulid,omitempty"` // gcc 娑?pipeline 闁板秶鐤?ULID [required]
	PipelineName      string                 `json:"pipeline_name,omitempty"`
	Status            gprogpb.PipelineStatus `json:"status,omitempty"`             // pipeline 瑜版挸澧犻悩鑸碘偓?[required]
	CurrentStageUlid  string                 `json:"current_stage_ulid,omitempty"` // 瑜版挸澧?stage 鐎圭偘绶?ULID [optional]
	CurrentStageName  string                 `json:"current_stage_name,omitempty"`
	Progress          float64                `json:"progress"` // 瑜版挸澧犻梼鑸殿唽閻ㄥ嫬顒熸稊鐘虹箻鎼?(0-100)
	ProgressAvailable bool                   `json:"progress_available,omitempty"`
	LmsProgress       uint32                 `json:"lms_progress,omitempty"` // LMS 鐠囧墽鈻兼潻娑樺閻ф儳鍨庡В?[optional]
	StartedAt         string                 `json:"started_at,omitempty"`   // pipeline 瀵偓婵妞傞梻杈剧礉RFC3339 [optional]
	CompletedAt       string                 `json:"completed_at,omitempty"` // pipeline 鐎瑰本鍨氶弮鍫曟？閿涘FC3339 [optional]
	CreatedAt         string                 `json:"created_at,omitempty"`   // pipeline 閸掓稑缂撻弮鍫曟？閿涘FC3339 [required]
}

type PipelineCreateRsp struct {
	PipelineUlid       string                 `json:"pipeline_ulid,omitempty"`        // pipeline 鐎圭偘绶?ULID [required]
	PipelineStatus     gprogpb.PipelineStatus `json:"pipeline_status,omitempty"`      // pipeline 瑜版挸澧犻悩鑸碘偓?[required]
	CurrentStageUlid   string                 `json:"current_stage_ulid,omitempty"`   // 瑜版挸澧?stage 鐎圭偘绶?ULID閿涘矁瀚㈤弮鐘插灟娑撹櫣鈹?[optional]
	CurrentStageStatus gprogpb.StageStatus    `json:"current_stage_status,omitempty"` // 瑜版挸澧?stage 閻樿埖鈧緤绱濋懟銉︽￥閸掓瑤璐?UNSPECIFIED [required]
	Message            string                 `json:"message,omitempty"`              // 娴滆櫣琚崣顖濐嚢鐠囧瓨妲?[required]
}

type PipelineRuntimeRsp struct {
	Config             PipelineConfig         `json:"config"`
	Instance           PipelineSummary        `json:"instance,omitempty"`
	PipelineStatus     gprogpb.PipelineStatus `json:"pipeline_status,omitempty"`
	CurrentStageUlid   string                 `json:"current_stage_ulid,omitempty"`
	CurrentStageStatus gprogpb.StageStatus    `json:"current_stage_status,omitempty"`
	CurrentStageName   string                 `json:"current_stage_name,omitempty"`
	CurrentUnitStatus  gprog.CourseUnitStatus `json:"current_unit_status,omitempty"`
	NextStep           PipelineNextStep       `json:"next_step,omitempty"`
}

type PipelineNextStep struct {
	Action           string                 `json:"action,omitempty"`
	Message          string                 `json:"message,omitempty"`
	StageId          string                 `json:"stage_id,omitempty"`
	StageName        string                 `json:"stage_name,omitempty"`
	CourseUnitUlid   string                 `json:"course_unit_ulid,omitempty"`
	CourseUnitCcUlid string                 `json:"course_unit_cc_ulid,omitempty"`
	CourseId         string                 `json:"course_id,omitempty"`
	Program          string                 `json:"program,omitempty"`
	ExamId           string                 `json:"exam_id,omitempty"`
	FormCode         string                 `json:"form_code,omitempty"`
	Status           gprog.CourseUnitStatus `json:"status,omitempty"`
	PipelineStatus   gprogpb.PipelineStatus `json:"pipeline_status,omitempty"`
	AllowRetake      bool                   `json:"allow_retake,omitempty"`
	AllowExemption   bool                   `json:"allow_exemption,omitempty"`
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

// ===================== 妫ｆ牠銆?(Dashboard) =====================

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

// ===================== 鐎涳缚绡勬潻娑樺 (Progress) =====================

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
	Records []ProgressRecord `json:"records,omitempty"` // 鏉╂稑瀹崇拋鏉跨秿閸掓銆?[required]
}

type ProgressRecord struct {
	CandidateId     string  `json:"candidate_id,omitempty"`      // 閼板啰鏁揑D [required]
	MaterialId      string  `json:"material_id,omitempty"`       // 鐠у嫭鏋D [required]
	CoursePackageId string  `json:"course_package_id,omitempty"` // 鐠у嫭鏋￠崠鍖 [required]
	ProgressType    string  `json:"progress_type,omitempty"`     // 鏉╂稑瀹崇猾璇茬€?[required]
	ProgressValue   float64 `json:"progress_value,omitempty"`    // 鏉╂稑瀹抽崐纭风窗鐟欏棝顣舵稉铏诡潡閺佸府绱濋弬鍥ㄣ€傛稉铏规閸掑棙鐦?[required]
	RecordedAt      string  `json:"recorded_at,omitempty"`       // 鐠佹澘缍嶉弮鍫曟？ RFC3339 [required]
}

// ===================== 閼板啳鐦?(Exams) =====================

type ExamResultDetailRsp struct {
	ExamId           string  `json:"exam_id,omitempty"`
	TotalScore       float64 `json:"total_score,omitempty"`
	IsPassed         bool    `json:"is_passed,omitempty"`
	ScoreDetailsJson string  `json:"score_details_json,omitempty"`
}

type ExamListItem struct {
	ExamId               string  `json:"exam_id,omitempty"`
	ProgramCode          string  `json:"program_code,omitempty"`
	ExamCode             string  `json:"exam_code,omitempty"`
	ExamStatus           string  `json:"exam_status,omitempty"`
	ResultStatus         string  `json:"result_status,omitempty"`
	TotalScore           float64 `json:"total_score,omitempty"`
	IsPassed             bool    `json:"is_passed,omitempty"`
	CandidateFirstName   string  `json:"candidate_first_name,omitempty"`
	CandidateLastName    string  `json:"candidate_last_name,omitempty"`
	CandidateEmail       string  `json:"candidate_email,omitempty"`
	ConfirmationNumber   string  `json:"confirmation_number,omitempty"`
	AppointmentStartTime string  `json:"appointment_start_time,omitempty"`
	AppointmentEndTime   string  `json:"appointment_end_time,omitempty"`
	SiteName             string  `json:"site_name,omitempty"`
	LastTermurlTimestamp string  `json:"last_termurl_timestamp,omitempty"`
	LastTermurlType      string  `json:"last_termurl_type,omitempty"`
}

type ListExamsRsp struct {
	Exams []ExamListItem `json:"exams"`
	Total uint32         `json:"total"`
}

type CandidateSignupExamRsp struct {
	CourseUnitUlid   string                 `json:"course_unit_ulid,omitempty"`   // course unit 鐎圭偘绶?ULID [required]
	CourseUnitStatus gprog.CourseUnitStatus `json:"course_unit_status,omitempty"` // course unit 瑜版挸澧犻悩鑸碘偓?[required]
	Message          string                 `json:"message,omitempty"`            // 娴滆櫣琚崣顖濐嚢鐠囧瓨妲?[required]
}

// CandidateApplyRetake
type CandidateApplyRetakeRsp struct {
	CourseUnitUlid   string                 `json:"course_unit_ulid,omitempty"`   // course unit 鐎圭偘绶?ULID [required]
	CourseUnitStatus gprog.CourseUnitStatus `json:"course_unit_status,omitempty"` // course unit 瑜版挸澧犻悩鑸碘偓?[required]
	Message          string                 `json:"message,omitempty"`            // 娴滆櫣琚崣顖濐嚢鐠囧瓨妲?[required]
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

// ===================== 閺€顖欑帛 (Payments) =====================

type LineItemInput struct {
	Name        string `json:"name"`        // 閸熷棗鎼ч崥宥囆為敍灞筋洤 "Level 1 鐠併倛鐦夐懓鍐槸鐠愬湱鏁?
	Description string `json:"description"` // 閸熷棗鎼х拠锔剧矎閹诲繗鍫?
	UnitAmount  int64  `json:"unit_amount"` // 閸熷棗鎼ч崡鏇氱幆閿涘牆宕熸担宥忕窗閸掑棴绱濇俊?100 娴狅綀銆?1.00 閸忓喛绱?
	Quantity    int64  `json:"quantity"`    // 鐠愵厺鎷遍弫浼村櫤
	Currency    string `json:"currency"`    // 鐠愌冪鐢胶顫掗敍灞筋洤 "usd", "cny"
}

type CreatePaymentInput struct {
	// OrderID    string            `json:"order_id"`
	LineItems  []LineItemInput   `json:"line_items"`  // 鐠併垹宕熼崠鍛儓閻ㄥ嫬鏅㈤崫浣稿灙鐞?
	Metadata   map[string]string `json:"metadata"`    // 閹碘晛鐫嶉崗鍐╂殶閹诡噯绱濋棁鈧崠鍛儓 pipeline_ulid, stage_ulid 娴犮儰绌堕崶鐐剁殶鐎电澶?
	SuccessURL string            `json:"success_url"` // 閺€顖欑帛閹存劕濮涢崥搴ｆ畱鐠哄疇娴嗛崷鏉挎絻
	CancelURL  string            `json:"cancel_url"`  // 閺€顖欑帛閸欐牗绉烽崥搴ｆ畱鏉╂柨娲栭崷鏉挎絻
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

// ===================== 閸欐垹銈?(Invoices) =====================

type QueryInvoiceRsp struct {
	InvoiceID     string  `json:"invoice_id"`
	PaymentID     string  `json:"payment_id"`
	RequestStatus string  `json:"request_status"`
	SubTotal      float64 `json:"sub_total"`
	TotalTax      float64 `json:"total_tax"`
	Total         float64 `json:"total"`
	ErrorMsg      string  `json:"error_msg"`
}

// ===================== 鐠併垹宕?(Orders) =====================

type OrderItem struct {
	OrderID       string  `json:"order_id"`
	ProductName   string  `json:"product_name"`
	Status        string  `json:"status"` // completed / pending / cancelled
	CreatedAt     string  `json:"created_at"`
	PaymentMethod string  `json:"payment_method"`
	Amount        float64 `json:"amount"`
}

type PurchasePipelineReq struct {
	PaymentMode                     string `json:"payment_mode"` // FULL_PIPELINE or BY_STAGE
	CandidateSelectedExemptionsJson string `json:"candidate_selected_exemptions_json"`
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

type PreviewPaymentReq struct {
	BizType     string   `json:"biz_type"`
	BizRefUlid  string   `json:"biz_ref_ulid"`
	CouponCodes []string `json:"coupon_codes"`
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
	Orders      []OrderItem `json:"orders"`
}

// ===================== 濞戝牊浼?(Messages) =====================

type MessageListInput struct {
	Limit  int `json:"limit"`   //  濮ｅ繘銆夐弫浼村櫤閿涘矂绮拋?20
	LastId int `json:"last_id"` // 娑撳﹣绔存い鍨付閸氬海娈慽d
}

type MessageItem struct {
	Id         uint64 `json:"id"`          // 閼奉亜顤僆D, 娑撳鏁?
	MessageId  string `json:"message_id"`  // 濞戝牊浼匢D
	UserId     string `json:"user_id"`     // 閼板啰鏁揑D
	TemplateId string `json:"template_id"` // 濡剝婢業D
	// payload 鐎涙ê鍋嶆稉?JSON 鐎涙顑佹稉璇х礉閸撳秶顏?Vue3 閻╁瓨甯?JSON.parse 閸楀啿褰?
	Payload   string               `json:"payload"`
	MsgType   gmsgpb.MsgType       `json:"msg_type"`
	MsgSource gmsgpb.MsgSource     `json:"msg_source"`
	SenderId  string               `json:"sender_id"`
	Status    gmsgpb.MessageStatus `json:"status"`
	// 閺冨爼妫挎担璺ㄦ暏 string 鐞涖劏鎻?(閺嶇厧绱? "2026-04-27T16:00:00Z")
	CreatedAt string `json:"created_at"`
}

type MessageListRsp struct {
	Messages []MessageItem `json:"messages"`
	HasMore  bool          `json:"has_more"`
}

type MessageOperationInput struct {
	MessageIDs []string `json:"message_ids"`
}

// ===================== 娴兼艾鎲?(Membership) =====================

type MembershipRsp struct {
	Level        string   `json:"level"` // student / certified / charterholder
	ExpiresAt    string   `json:"expires_at"`
	SubscribedAt string   `json:"subscribed_at"`
	AutoRenew    bool     `json:"auto_renew"`
	Benefits     []string `json:"benefits"`
}

// ===================== 鐠囦椒鍔?(Certificates) =====================

type ListCredentialsRsp struct {
	Credentials []CredentialsItem `json:"credentials"`
}

type CredentialsItem struct {
	CatalogId   string `json:"catalog_id,omitempty"`  // 鐠у嫭鐗哥猾璇插焼ID ULID [required]
	Name        string `json:"name,omitempty"`        // 鐠у嫭鐗哥猾璇插焼閸氬秶袨 [required]
	Description string `json:"description,omitempty"` // 鐠у嫭鐗哥猾璇插焼閹诲繗鍫?[required]

	Eligible         bool                    `json:"eligible,omitempty"`          // 閺勵垰鎯侀幐浣规箒閺堝鏅ョ挧鍕壐
	CredentialStatus gcreds.CredentialStatus `json:"credential_status,omitempty"` // 瑜版挸澧犵挧鍕壐閻樿埖鈧緤绱遍懟銉︽￥鐠佹澘缍嶉崚娆庤礋 UNSPECIFIED
	Message          string                  `json:"message,omitempty"`           // 娴滆櫣琚崣顖濐嚢閻ㄥ嫯顕╅弰?

	CertificateInfo // 鐠囦椒鍔熸穱鈩冧紖 [optional]
}

type CertificateFileInfo struct {
	FileHash  string                    `json:"file_hash,omitempty"`  // SHA256 [required]
	FileName  string                    `json:"file_name,omitempty"`  // 閺傚洣娆㈤崥?[required]
	FileType  gcreds.CredentialFileType `json:"file_type,omitempty"`  // 閺傚洣娆㈢猾璇茬€?[required]
	FileExt   string                    `json:"file_ext,omitempty"`   // 閺傚洣娆㈤幍鈺佺潔閸?[required]
	FileSize  uint64                    `json:"file_size,omitempty"`  // 閺傚洣娆㈡径褍鐨?[required]
	FileUsage string                    `json:"file_usage,omitempty"` // 閺傚洣娆㈤悽銊┾偓? 婵?"front_view" [optional]
	ViewUrl   string                    `json:"view_url,omitempty"`   // 閺傚洣娆㈤崣顖濐問闂傤噣鎽奸幒?[optional]
}

type CertificateInfo struct {
	CredId      string                  `json:"cred_id,omitempty"`      // 閸烆垯绔碔D ULID [required]
	CredGuid    string                  `json:"cred_guid,omitempty"`    // 鐠恒劎澧楅張顑跨瑹閸斺€虫暜娑撯偓 ID
	CandidateId string                  `json:"candidate_id,omitempty"` // 閼板啰鏁撻柅鏄忕帆 ID (ULID)
	Version     uint32                  `json:"version,omitempty"`      // 閻楀牊婀伴崣?[required]
	Status      gcreds.CredentialStatus `json:"status,omitempty"`       // 鐠у嫭鐗搁悩鑸碘偓?[required]
	Files       []CertificateFileInfo   `json:"files,omitempty"`        // 閺傚洣娆㈤崚妤勩€?[required]
	AuditorId   string                  `json:"auditor_id,omitempty"`   // 鐎光剝鐗虫禍绡扗 ULID [optional]
	AuditRemark string                  `json:"audit_remark,omitempty"` // 鐎光剝鐗虫径鍥ㄦ暈 [optional]
	ValidUntil  string                  `json:"valid_until,omitempty"`  // 閺堝鏅ラ張?RFC3339 閺嶇厧绱＄€涙顑佹稉?[optional]
	CreatedAt   string                  `json:"created_at,omitempty"`   // 閸掓稑缂撻弮鍫曟？ RFC3339 閺嶇厧绱＄€涙顑佹稉?[optional]
}

type CertificateItem struct {
	CatalogId   string `json:"catalog_id,omitempty"`  // 鐠у嫭鐗哥猾璇插焼ID ULID [required]
	Name        string `json:"name,omitempty"`        // 鐠у嫭鐗哥猾璇插焼閸氬秶袨 [required]
	Description string `json:"description,omitempty"` // 鐠у嫭鐗哥猾璇插焼閹诲繗鍫?[required]

	CertificateInfo // 鐠囦椒鍔熸穱鈩冧紖 [optional]
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
