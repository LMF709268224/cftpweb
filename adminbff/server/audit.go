package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"adminbff/config"
	"adminbff/handler"

	"github.com/afnandelfin620-star/cftptest/cftp/util"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"
)

const (
	adminAuditSourceService = "adminbff"
	auditCaptureLimit       = 64 << 10
	auditPublishTimeout     = 3 * time.Second
)

type auditEventPublisher interface {
	Publish(context.Context, *util.NatsAuditEvent) error
	Close() error
}

type natsAuditPublisher struct {
	conn *nats.Conn
}

func newNATSAuditPublisher() (*natsAuditPublisher, error) {
	address := util.GetEndpointAddress(config.EnvNatsAddr, "nats", "4222")
	conn, err := nats.Connect(
		"nats://"+address,
		nats.Name(adminAuditSourceService),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			slog.Warn("Admin audit NATS disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			slog.Info("Admin audit NATS reconnected", "url", conn.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS at %q: %w", address, err)
	}
	return &natsAuditPublisher{conn: conn}, nil
}

func (p *natsAuditPublisher) Publish(ctx context.Context, event *util.NatsAuditEvent) error {
	return util.PublishAuditEvent(ctx, p.conn, adminAuditSourceService, event)
}

func (p *natsAuditPublisher) Close() error {
	if p == nil || p.conn == nil {
		return nil
	}
	return p.conn.Drain()
}

type auditRoute struct {
	Method string
	Path   string
}

type auditSpec struct {
	Action       string
	ResourceType string
	IDParam      string
	IDKeys       []string
	BoolAction   *boolAction
}

type boolAction struct {
	Field       string
	TrueAction  string
	FalseAction string
}

func operation(action, resourceType, idParam string, idKeys ...string) auditSpec {
	return auditSpec{Action: action, ResourceType: resourceType, IDParam: idParam, IDKeys: idKeys}
}

func operationByBool(field, trueAction, falseAction, resourceType string, idKeys ...string) auditSpec {
	return auditSpec{
		Action:       "review_application",
		ResourceType: resourceType,
		IDKeys:       idKeys,
		BoolAction: &boolAction{
			Field:       field,
			TrueAction:  trueAction,
			FalseAction: falseAction,
		},
	}
}

func route(method, path string) auditRoute {
	path = strings.TrimSuffix(path, "/*")
	if path != "/" {
		path = strings.TrimSuffix(path, "/")
	}
	return auditRoute{Method: method, Path: path}
}

// adminAuditOperations is the searchable taxonomy for admin-side mutations.
// Actions are lower_snake_case verb-object names. Resource types are stable,
// singular domain nouns. Renaming either value is a data-contract change.
var adminAuditOperations = map[auditRoute]auditSpec{
	route(http.MethodPut, "/api/user/profile"):  operation("update_own_profile", "admin_user", ""),
	route(http.MethodPut, "/api/user/password"): operation("change_own_password", "admin_user", ""),

	route(http.MethodPut, "/api/pipelines/{pipeline_id}/translations"):    operation("update_pipeline_translations", "certification_pipeline", "pipeline_id"),
	route(http.MethodPost, "/api/pipelines/"):                             operation("create_pipeline", "certification_pipeline", "", "pipeline_id", "pipeline_ulid"),
	route(http.MethodPost, "/api/pipelines/{pipeline_id}/duplicate"):      operation("duplicate_pipeline", "certification_pipeline", "pipeline_id"),
	route(http.MethodPut, "/api/pipelines/{pipeline_id}/structure"):       operation("update_pipeline_structure", "certification_pipeline", "pipeline_id"),
	route(http.MethodPut, "/api/pipelines/{pipeline_id}/metadata"):        operation("update_pipeline_metadata", "certification_pipeline", "pipeline_id"),
	route(http.MethodPost, "/api/pipelines/{pipeline_id}/publish"):        operation("publish_pipeline", "certification_pipeline", "pipeline_id"),
	route(http.MethodPost, "/api/pipelines/{pipeline_id}/deprecate"):      operation("deprecate_pipeline", "certification_pipeline", "pipeline_id"),
	route(http.MethodDelete, "/api/pipelines/{pipeline_id}"):              operation("delete_pipeline", "certification_pipeline", "pipeline_id"),
	route(http.MethodPut, "/api/pipeline-stages/{stage_id}/translations"): operation("update_pipeline_stage_translations", "pipeline_stage", "stage_id"),
	route(http.MethodPut, "/api/pipeline-units/{unit_id}/translations"):   operation("update_pipeline_unit_translations", "pipeline_unit", "unit_id"),

	route(http.MethodPost, "/api/prog/pipelines/{pipeline_ulid}/trigger-next-stage"):      operation("trigger_next_pipeline_stage", "candidate_pipeline", "pipeline_ulid"),
	route(http.MethodPost, "/api/prog/pipelines/{pipeline_ulid}/terminate"):               operation("terminate_pipeline", "candidate_pipeline", "pipeline_ulid"),
	route(http.MethodPost, "/api/prog/pipelines/{pipeline_ulid}/pdf-requests"):            operation("create_certificate_pdf_request", "candidate_pipeline", "pipeline_ulid"),
	route(http.MethodPost, "/api/prog/certificate-tasks/{task_ulid}/retry"):               operation("retry_certificate_task", "certificate_task", "task_ulid"),
	route(http.MethodPost, "/api/prog/course-units/{course_unit_ulid}/force-completed"):   operation("force_complete_course_unit", "candidate_course_unit", "course_unit_ulid"),
	route(http.MethodPost, "/api/prog/course-units/{course_unit_ulid}/force-signup-exam"): operation("force_signup_exam", "candidate_course_unit", "course_unit_ulid"),
	route(http.MethodPost, "/api/prog/mail-tasks/{mail_task_ulid}/retry"):                 operation("retry_progress_mail", "progress_mail_task", "mail_task_ulid"),
	route(http.MethodPost, "/api/prog/mail-tasks/{mail_task_ulid}/ignore"):                operation("ignore_progress_mail", "progress_mail_task", "mail_task_ulid"),

	route(http.MethodPost, "/api/exams/{exam_ulid}/essay-grade/import"):       operation("import_essay_grades", "exam", "exam_ulid"),
	route(http.MethodPost, "/api/exams/{exam_ulid}/sync-result"):              operation("sync_exam_result", "exam", "exam_ulid"),
	route(http.MethodPost, "/api/exam-ops/reminder-mails/{mail_ulid}/retry"):  operation("retry_exam_reminder_mail", "exam_reminder_mail", "mail_ulid"),
	route(http.MethodPost, "/api/exam-ops/reminder-mails/{mail_ulid}/ignore"): operation("ignore_exam_reminder_mail", "exam_reminder_mail", "mail_ulid"),

	route(http.MethodPost, "/api/catalogs/"):                                                      operation("create_catalog", "catalog", "", "catalog_id", "catalog_ulid"),
	route(http.MethodPut, "/api/catalogs/{catalog_id}"):                                           operation("update_catalog", "catalog", "catalog_id"),
	route(http.MethodPost, "/api/lms/courses/"):                                                   operation("create_course", "course", "", "course_id", "course_ulid"),
	route(http.MethodPut, "/api/lms/courses/{course_id}/translations"):                            operation("update_course_translations", "course", "course_id"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/publish"):                                operation("publish_course", "course", "course_id"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/cleanup-assets"):                         operation("cleanup_course_assets", "course", "course_id"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/progress/sync"):                          operation("sync_course_progress", "course", "course_id"),
	route(http.MethodPut, "/api/lms/courses/{course_id}"):                                         operation("update_course", "course", "course_id"),
	route(http.MethodDelete, "/api/lms/courses/{course_id}"):                                      operation("delete_course", "course", "course_id"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/materials"):                              operation("create_course_material", "course_material", "", "material_id", "material_ulid"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/materials/reorder"):                      operation("reorder_course_materials", "course", "course_id"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/supplementary-material"):                 operation("create_supplementary_material", "supplementary_material", "", "material_id"),
	route(http.MethodPut, "/api/lms/courses/{course_id}/supplementary-material/{material_id}"):    operation("update_supplementary_material", "supplementary_material", "material_id"),
	route(http.MethodDelete, "/api/lms/courses/{course_id}/supplementary-material/{material_id}"): operation("delete_supplementary_material", "supplementary_material", "material_id"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/permissions/grant"):                      operation("grant_course_access", "course", "course_id"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/permissions/revoke"):                     operation("revoke_course_access", "course", "course_id"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/chapters"):                               operation("create_chapter", "chapter", "", "chapter_id", "chapter_ulid"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/chapters/import"):                        operation("import_chapter", "course", "course_id"),
	route(http.MethodPost, "/api/lms/courses/{course_id}/chapters/reorder"):                       operation("reorder_chapters", "course", "course_id"),
	route(http.MethodPut, "/api/lms/materials/{material_id}"):                                     operation("update_course_material", "course_material", "material_id"),
	route(http.MethodDelete, "/api/lms/materials/{material_id}"):                                  operation("delete_course_material", "course_material", "material_id"),
	route(http.MethodPost, "/api/lms/chapters/{chapter_id}/lessons"):                              operation("create_lesson", "lesson", "", "lesson_id", "lesson_ulid"),
	route(http.MethodPost, "/api/lms/chapters/{chapter_id}/lessons/reorder"):                      operation("reorder_lessons", "chapter", "chapter_id"),
	route(http.MethodPut, "/api/lms/chapters/{chapter_id}"):                                       operation("update_chapter", "chapter", "chapter_id"),
	route(http.MethodDelete, "/api/lms/chapters/{chapter_id}"):                                    operation("delete_chapter", "chapter", "chapter_id"),
	route(http.MethodPut, "/api/lms/lessons/{lesson_id}"):                                         operation("update_lesson", "lesson", "lesson_id"),
	route(http.MethodDelete, "/api/lms/lessons/{lesson_id}"):                                      operation("delete_lesson", "lesson", "lesson_id"),

	route(http.MethodPost, "/api/lms/external-coursewares/"):                              operation("create_external_courseware", "external_courseware", "", "courseware_id", "courseware_ulid"),
	route(http.MethodPut, "/api/lms/external-coursewares/{courseware_id}"):                operation("update_external_courseware", "external_courseware", "courseware_id"),
	route(http.MethodDelete, "/api/lms/external-coursewares/{courseware_id}"):             operation("delete_external_courseware", "external_courseware", "courseware_id"),
	route(http.MethodPost, "/api/lms/external-coursewares/{courseware_id}/tokens/import"): operation("import_courseware_tokens", "external_courseware", "courseware_id"),
	route(http.MethodPost, "/api/lms/prerequisites/"):                                     operation("create_prerequisite", "prerequisite", "", "prerequisite_id", "prerequisite_ulid"),
	route(http.MethodPut, "/api/lms/prerequisites/{prerequisite_id}"):                     operation("update_prerequisite", "prerequisite", "prerequisite_id"),
	route(http.MethodDelete, "/api/lms/prerequisites/{prerequisite_id}"):                  operation("delete_prerequisite", "prerequisite", "prerequisite_id"),
	route(http.MethodPost, "/api/lms/quizzes/"):                                           operation("create_quiz", "quiz", "", "quiz_id", "quiz_ulid"),
	route(http.MethodPost, "/api/lms/quizzes/{quiz_id}/questions"):                        operation("create_quiz_question", "quiz_question", "", "question_id", "question_ulid"),
	route(http.MethodPost, "/api/lms/quizzes/{quiz_id}/questions/reorder"):                operation("reorder_quiz_questions", "quiz", "quiz_id"),
	route(http.MethodPut, "/api/lms/quizzes/{quiz_id}"):                                   operation("update_quiz", "quiz", "quiz_id"),
	route(http.MethodDelete, "/api/lms/quizzes/{quiz_id}"):                                operation("delete_quiz", "quiz", "quiz_id"),
	route(http.MethodPost, "/api/lms/questions/{question_id}/options"):                    operation("create_quiz_option", "quiz_option", "", "option_id", "option_ulid"),
	route(http.MethodPost, "/api/lms/questions/{question_id}/options/reorder"):            operation("reorder_quiz_options", "quiz_question", "question_id"),
	route(http.MethodPut, "/api/lms/questions/{question_id}"):                             operation("update_quiz_question", "quiz_question", "question_id"),
	route(http.MethodDelete, "/api/lms/questions/{question_id}"):                          operation("delete_quiz_question", "quiz_question", "question_id"),
	route(http.MethodPut, "/api/lms/options/{option_id}"):                                 operation("update_quiz_option", "quiz_option", "option_id"),
	route(http.MethodDelete, "/api/lms/options/{option_id}"):                              operation("delete_quiz_option", "quiz_option", "option_id"),

	route(http.MethodPost, "/api/lms/resource-packs/"):                           operation("create_resource_pack", "resource_pack", "", "pack_id", "pack_ulid"),
	route(http.MethodPut, "/api/lms/resource-packs/{pack_id}/translations"):      operation("update_resource_pack_translations", "resource_pack", "pack_id"),
	route(http.MethodPut, "/api/lms/resource-packs/{pack_id}"):                   operation("update_resource_pack", "resource_pack", "pack_id"),
	route(http.MethodPost, "/api/lms/resource-packs/{pack_id}/publish"):          operation("publish_resource_pack", "resource_pack", "pack_id"),
	route(http.MethodPost, "/api/lms/resource-packs/{pack_id}/revert-to-draft"):  operation("revert_resource_pack_to_draft", "resource_pack", "pack_id"),
	route(http.MethodPost, "/api/lms/resource-packs/{pack_id}/duplicate"):        operation("duplicate_resource_pack", "resource_pack", "pack_id"),
	route(http.MethodPost, "/api/lms/resource-packs/{pack_id}/cleanup-assets"):   operation("cleanup_resource_pack_assets", "resource_pack", "pack_id"),
	route(http.MethodDelete, "/api/lms/resource-packs/{pack_id}"):                operation("delete_resource_pack", "resource_pack", "pack_id"),
	route(http.MethodPost, "/api/lms/resource-packs/{pack_id}/files"):            operation("create_resource_pack_file", "resource_pack_file", "", "file_id", "file_ulid"),
	route(http.MethodPut, "/api/lms/resource-pack-files/{file_id}/translations"): operation("update_resource_pack_file_translations", "resource_pack_file", "file_id"),
	route(http.MethodPut, "/api/lms/resource-pack-files/{file_id}"):              operation("update_resource_pack_file", "resource_pack_file", "file_id"),
	route(http.MethodDelete, "/api/lms/resource-pack-files/{file_id}"):           operation("delete_resource_pack_file", "resource_pack_file", "file_id"),
	route(http.MethodPost, "/api/lms/enrollments/batch"):                         operation("batch_enroll_courses", "course_enrollment", "", "candidate_id", "candidate_ulid"),
	route(http.MethodPost, "/api/lms/import"):                                    operation("import_lms_content", "lms_content", ""),

	route(http.MethodPost, "/api/messages/templates"):   operation("create_message_template", "message_template", "", "template_id", "template_ulid"),
	route(http.MethodPut, "/api/messages/templates"):    operation("update_message_template", "message_template", "", "template_id", "template_ulid"),
	route(http.MethodDelete, "/api/messages/templates"): operation("delete_message_template", "message_template", "", "template_id", "template_ulid"),
	route(http.MethodPost, "/api/messages/revoke"):      operation("revoke_message", "message", "", "message_id", "message_ulid"),
	route(http.MethodPost, "/api/messages/send"):        operation("send_message", "message", "", "message_id", "message_ulid"),
	route(http.MethodPost, "/api/mails/send"):           operation("send_mail", "mail", "", "mail_id", "mail_ulid"),
	route(http.MethodPost, "/api/mails/cancel"):         operation("cancel_mail", "mail", "", "mail_id", "mail_ulid"),
	route(http.MethodPost, "/api/mails/templates/"):     operation("create_mail_template", "mail_template", "", "template_id", "template_ulid"),
	route(http.MethodPut, "/api/mails/templates/"):      operation("update_mail_template", "mail_template", "", "template_id", "template_ulid"),
	route(http.MethodDelete, "/api/mails/templates/"):   operation("delete_mail_template", "mail_template", "", "template_id", "template_ulid"),

	route(http.MethodPost, "/api/credentials/version-files/{file_id}/ignore"):          operation("ignore_credential_version_file", "credential_version_file", "file_id"),
	route(http.MethodPut, "/api/credentials/definitions/{cred_def_ulid}/translations"): operation("update_credential_definition_translations", "credential_definition", "cred_def_ulid"),
	route(http.MethodPut, "/api/credentials/definitions/{cred_def_ulid}/attachments"):  operation("update_credential_definition_attachments", "credential_definition", "cred_def_ulid"),
	route(http.MethodPost, "/api/credentials/definitions"):                             operation("create_credential_definition", "credential_definition", "", "cred_def_ulid", "credential_definition_ulid"),
	route(http.MethodPost, "/api/applications/audit"):                                  operationByBool("approved", "approve_application", "reject_application", "credential_application", "app_id", "app_ulid", "application_id", "application_ulid"),
	route(http.MethodPut, "/api/pdf-templates/{template_id}/translations"):             operation("update_pdf_template_translations", "pdf_template", "template_id"),
	route(http.MethodPost, "/api/pdf-templates/"):                                      operation("create_pdf_template", "pdf_template", "", "template_id", "template_ulid"),
	route(http.MethodPut, "/api/pdf-templates/"):                                       operation("update_pdf_template", "pdf_template", "", "template_id", "template_ulid"),

	route(http.MethodPost, "/api/mall/orders/sync-meta"):                   operation("sync_order_metadata", "order", "", "order_id", "order_ulid"),
	route(http.MethodPost, "/api/mall/mail-tasks/{mail_task_ulid}/retry"):  operation("retry_mall_mail", "mall_mail_task", "mail_task_ulid"),
	route(http.MethodPost, "/api/mall/mail-tasks/{mail_task_ulid}/ignore"): operation("ignore_mall_mail", "mall_mail_task", "mail_task_ulid"),
	route(http.MethodPost, "/api/mall/bundles/"):                           operation("create_bundle", "bundle", "", "bundle_id", "bundle_ulid"),
	route(http.MethodPost, "/api/mall/bundles/sync-display-pricing"):       operation("sync_bundle_display_pricing", "bundle", "", "bundle_id", "bundle_ulid"),
	route(http.MethodPut, "/api/mall/bundles/{bundle_ulid}/translations"):  operation("update_bundle_translations", "bundle", "bundle_ulid"),
	route(http.MethodPost, "/api/mall/bundles/{bundle_ulid}/duplicate"):    operation("duplicate_bundle", "bundle", "bundle_ulid"),
	route(http.MethodPut, "/api/mall/bundles/{bundle_ulid}/meta"):          operation("update_bundle_metadata", "bundle", "bundle_ulid"),
	route(http.MethodPut, "/api/mall/bundles/pricing"):                     operation("update_bundle_pricing", "bundle", "", "bundle_id", "bundle_ulid"),
	route(http.MethodPost, "/api/mall/bundles/{bundle_ulid}/publish"):      operation("publish_bundle", "bundle", "bundle_ulid"),
	route(http.MethodPost, "/api/mall/bundles/{bundle_ulid}/deprecate"):    operation("deprecate_bundle", "bundle", "bundle_ulid"),
	route(http.MethodDelete, "/api/mall/bundles/{bundle_ulid}"):            operation("delete_bundle", "bundle", "bundle_ulid"),
	route(http.MethodPost, "/api/mall/bundle-orders/purge"):                operation("purge_candidate_bundle", "candidate_bundle", "", "candidate_id", "candidate_ulid"),
	route(http.MethodPost, "/api/pay/payment-requests/sync-currency"):      operation("sync_payment_request_currency", "payment_request", "", "request_id", "request_ulid"),

	route(http.MethodPost, "/api/memberships/"):                              operation("create_membership_config", "membership_config", "", "membership_id", "membership_ulid"),
	route(http.MethodPut, "/api/memberships/"):                               operation("update_membership_config", "membership_config", "", "membership_id", "membership_ulid"),
	route(http.MethodPost, "/api/memberships/grant"):                         operation("grant_membership", "candidate_membership", "", "candidate_id", "candidate_ulid"),
	route(http.MethodPost, "/api/memberships/revoke"):                        operation("revoke_membership", "candidate_membership", "", "candidate_id", "candidate_ulid"),
	route(http.MethodPost, "/api/memberships/purge"):                         operation("purge_candidate_membership", "candidate_membership", "", "candidate_id", "candidate_ulid"),
	route(http.MethodPut, "/api/memberships/{membership_ulid}/translations"): operation("update_membership_translations", "membership_config", "membership_ulid"),
	route(http.MethodPost, "/api/memberships/{membership_ulid}/publish"):     operation("publish_membership_config", "membership_config", "membership_ulid"),
	route(http.MethodPost, "/api/memberships/{membership_ulid}/deprecate"):   operation("deprecate_membership_config", "membership_config", "membership_ulid"),
	route(http.MethodPost, "/api/memberships/mails/retry"):                   operation("retry_membership_mail", "membership_mail", "", "mail_id", "mail_ulid"),
	route(http.MethodPost, "/api/memberships/mails/ignore"):                  operation("ignore_membership_mail", "membership_mail", "", "mail_id", "mail_ulid"),
	route(http.MethodPost, "/api/audit/webhooks/reprocess"):                  operation("reprocess_webhook", "webhook_message", "", "message_id", "message_ulid"),
	route(http.MethodPost, "/api/permissions/mark-expired"):                  operation("mark_credential_expired", "credential", "", "credential_id", "credential_ulid"),
	route(http.MethodPost, "/api/permissions/revoke-credential"):             operation("revoke_credential", "credential", "", "credential_id", "credential_ulid"),
}

// These POST endpoints are intentionally not audited because they only read,
// preview, render, or mint a temporary transport URL and do not change business state.
var nonAuditedAdminMutations = map[auditRoute]string{
	route(http.MethodPost, "/api/exams/{exam_ulid}/essay-grade/import/preview"):                   "preview only",
	route(http.MethodPost, "/api/lms/upload-url"):                                                 "temporary upload URL",
	route(http.MethodPost, "/api/lms/view-url"):                                                   "temporary view URL",
	route(http.MethodPost, "/api/mails/templates/render"):                                         "render preview only",
	route(http.MethodPost, "/api/credentials/resources/check"):                                    "read-only existence check",
	route(http.MethodPost, "/api/credentials/definitions/{cred_def_ulid}/attachments/upload-url"): "temporary upload URL",
	route(http.MethodPost, "/api/mall/bundles/upload-url"):                                        "temporary upload URL",
	route(http.MethodPost, "/api/pay/order-amounts/batch"):                                        "read-only batch lookup",
	route(http.MethodPost, "/api/pay/invoice-amounts/batch"):                                      "read-only batch lookup",
}

type auditResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *auditResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *auditResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if w.body.Len() < auditCaptureLimit {
		remaining := auditCaptureLimit - w.body.Len()
		if remaining > len(data) {
			remaining = len(data)
		}
		_, _ = w.body.Write(data[:remaining])
	}
	return w.ResponseWriter.Write(data)
}

func (w *auditResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

type readCloser struct {
	io.Reader
	io.Closer
}

type cappedBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func (b *cappedBuffer) Write(data []byte) (int, error) {
	if b.buffer.Len() < b.limit {
		remaining := b.limit - b.buffer.Len()
		if remaining > len(data) {
			remaining = len(data)
		}
		_, _ = b.buffer.Write(data[:remaining])
	}
	return len(data), nil
}

func (s *Server) auditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now().UTC()
		var requestCapture cappedBuffer
		requestCapture.limit = auditCaptureLimit
		if r.Body != nil && strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
			r.Body = &readCloser{Reader: io.TeeReader(r.Body, &requestCapture), Closer: r.Body}
		}

		observer := &auditResponseWriter{ResponseWriter: w}
		defer func() {
			panicValue := recover()
			statusCode := observer.status
			if panicValue != nil {
				statusCode = http.StatusInternalServerError
			} else if statusCode == 0 {
				statusCode = http.StatusOK
			}

			routePattern := route(r.Method, chi.RouteContext(r.Context()).RoutePattern()).Path
			spec, audited := adminAuditOperations[route(r.Method, routePattern)]
			if audited && s.auditPublisher != nil {
				event := buildAdminAuditEvent(r, spec, routePattern, statusCode, startedAt, requestCapture.buffer.Bytes(), observer.body.Bytes())
				publishCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), auditPublishTimeout)
				if err := s.auditPublisher.Publish(publishCtx, event); err != nil {
					slog.ErrorContext(r.Context(), "Failed to publish admin audit event",
						"action", event.Action,
						"resource_type", spec.ResourceType,
						"resource_id", event.Resource.ID,
						"error", err,
					)
				}
				cancel()
			}
			if panicValue != nil {
				panic(panicValue)
			}
		}()

		next.ServeHTTP(observer, r)
	})
}

func buildAdminAuditEvent(r *http.Request, spec auditSpec, routePattern string, statusCode int, startedAt time.Time, requestBody, responseBody []byte) *util.NatsAuditEvent {
	action := resolveAuditAction(spec, requestBody)
	params := routeParams(r)
	resourceID := params[spec.IDParam]
	if resourceID == "" {
		keys := append([]string{}, spec.IDKeys...)
		keys = append(keys, spec.ResourceType+"_ulid", spec.ResourceType+"_id")
		resourceID = findJSONText(responseBody, keys)
		if resourceID == "" {
			resourceID = findJSONText(requestBody, keys)
		}
	}
	resourceID = truncate(resourceID, 64)
	resourceDisplayName := resourceID
	if resourceDisplayName == "" {
		resourceDisplayName = strings.ReplaceAll(spec.ResourceType, "_", " ")
	}

	status := "SUCCESS"
	if statusCode >= http.StatusBadRequest {
		status = "FAILED"
	}

	details := map[string]any{
		"schema_version": "admin.audit.v1",
		"http_method":    r.Method,
		"route_pattern":  routePattern,
		"http_status":    statusCode,
		"request_id":     middleware.GetReqID(r.Context()),
		"duration_ms":    time.Since(startedAt).Milliseconds(),
	}
	if len(params) > 0 {
		details["path_params"] = params
	}
	if status == "FAILED" {
		if errorCode := findJSONText(responseBody, []string{"error_code"}); errorCode != "" {
			details["error_code"] = errorCode
		}
	}
	detailBytes, err := json.Marshal(details)
	if err != nil {
		detailBytes = []byte("{}")
	}

	traceID := strings.TrimSpace(r.Header.Get("X-Trace-ID"))
	if traceID == "" {
		traceID = middleware.GetReqID(r.Context())
	}
	return &util.NatsAuditEvent{
		Action:      truncate(action, 128),
		Status:      status,
		SummaryText: truncate("{operator} "+strings.ReplaceAll(action, "_", " ")+" {resource}", 512),
		Operator: util.NatsAuditOperator{
			ID:    truncate(handler.AdminID(r), 64),
			Name:  truncate(handler.AdminName(r), 128),
			Email: truncate(handler.AdminEmail(r), 256),
			Role:  "admin",
		},
		Context: util.NatsAuditContext{
			ClientIP:    clientIP(r),
			UserAgent:   truncate(r.UserAgent(), 1024),
			GeoLocation: truncate(strings.TrimSpace(r.Header.Get("X-Client-Geo")), 256),
			TraceID:     truncate(traceID, 128),
			RequestURI:  truncate(r.URL.EscapedPath(), 512),
		},
		Resource: util.NatsAuditResource{
			Type:        truncate(spec.ResourceType, 64),
			ID:          resourceID,
			DisplayName: truncate(resourceDisplayName, 128),
		},
		Details:   string(detailBytes),
		CreatedAt: startedAt.Format(time.RFC3339Nano),
	}
}

func resolveAuditAction(spec auditSpec, requestBody []byte) string {
	if spec.BoolAction == nil || !json.Valid(requestBody) {
		return spec.Action
	}
	var payload map[string]any
	if err := json.Unmarshal(requestBody, &payload); err != nil {
		return spec.Action
	}
	value, ok := payload[spec.BoolAction.Field].(bool)
	if !ok {
		return spec.Action
	}
	if value {
		return spec.BoolAction.TrueAction
	}
	return spec.BoolAction.FalseAction
}

func routeParams(r *http.Request) map[string]string {
	ctx := chi.RouteContext(r.Context())
	if ctx == nil || len(ctx.URLParams.Keys) == 0 {
		return nil
	}
	params := make(map[string]string, len(ctx.URLParams.Keys))
	for i, key := range ctx.URLParams.Keys {
		if i < len(ctx.URLParams.Values) {
			params[key] = truncate(ctx.URLParams.Values[i], 256)
		}
	}
	return params
}

func findJSONText(data []byte, keys []string) string {
	if len(data) == 0 || len(keys) == 0 || !json.Valid(data) {
		return ""
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return ""
	}
	for _, key := range keys {
		if text := findJSONTextValue(value, strings.ToLower(key)); text != "" {
			return text
		}
	}
	return ""
}

func findJSONTextValue(value any, wanted string) string {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if strings.ToLower(key) == wanted {
				if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
					return strings.TrimSpace(text)
				}
			}
		}
		for _, item := range typed {
			if text := findJSONTextValue(item, wanted); text != "" {
				return text
			}
		}
	case []any:
		for _, item := range typed {
			if text := findJSONTextValue(item, wanted); text != "" {
				return text
			}
		}
	}
	return ""
}

func clientIP(r *http.Request) string {
	address := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(address); err == nil {
		return truncate(host, 128)
	}
	return truncate(address, 128)
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	return string([]rune(value)[:max])
}
