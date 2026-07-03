package handler

import (
	"context"
	"net/http"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	gmailpb "github.com/afnandelfin620-star/cftptest/cftp/gmail"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"
	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
)

// ── Context 键 ──

type ContextKey string

const (
	CtxKeyAdminID ContextKey = "admin_id"
	CtxKeyEmail   ContextKey = "email"
	CtxKeyName    ContextKey = "name"
	CtxKeyToken   ContextKey = "raw_token"
)

// ── Handler 容器 ──

type Handler struct {
	Lms                 lmspb.LmsServiceClient
	Mall                mallpb.MallServiceClient
	Gcc                 gccpb.CCServiceClient
	Gprog               gprogpb.ProgServiceClient
	Gmsg                gmsgpb.MessageServiceClient
	Creds               gcredspb.CredentialServiceClient
	Gexam               gexampb.GExamServiceClient
	Gmid                gmidpb.MidServiceClient
	Gmail               gmailpb.MailServiceClient
	Gpay                gpaypb.PayServiceClient
	Gmbr                gmbrpb.GmbrServiceClient
	CandidateProfiles   *CandidateProfileCache
	CasdoorEndpoint     string
	CasdoorClientId     string
	CasdoorClientSecret string
	CasdoorAppName      string
}

func New(
	lms lmspb.LmsServiceClient,
	mall mallpb.MallServiceClient,
	gcc gccpb.CCServiceClient,
	gprog gprogpb.ProgServiceClient,
	gmsg gmsgpb.MessageServiceClient,
	creds gcredspb.CredentialServiceClient,
	gexam gexampb.GExamServiceClient,
	gmid gmidpb.MidServiceClient,
	gmail gmailpb.MailServiceClient,
	gpay gpaypb.PayServiceClient,
	gmbr gmbrpb.GmbrServiceClient,
	casdoorEndpoint string,
	casdoorClientId string,
	casdoorClientSecret string,
	casdoorAppName string,
) *Handler {
	return &Handler{
		Lms:                 lms,
		Mall:                mall,
		Gcc:                 gcc,
		Gprog:               gprog,
		Gmsg:                gmsg,
		Creds:               creds,
		Gexam:               gexam,
		Gmid:                gmid,
		Gmail:               gmail,
		Gpay:                gpay,
		Gmbr:                gmbr,
		CandidateProfiles:   NewCandidateProfileCache(gmid),
		CasdoorEndpoint:     casdoorEndpoint,
		CasdoorClientId:     casdoorClientId,
		CasdoorClientSecret: casdoorClientSecret,
		CasdoorAppName:      casdoorAppName,
	}
}

// ── Context 取值辅助函数 ──

func AdminID(r *http.Request) string {
	v, _ := r.Context().Value(CtxKeyAdminID).(string)
	return v
}

func AdminName(r *http.Request) string {
	v, _ := r.Context().Value(CtxKeyName).(string)
	return v
}

func AdminEmail(r *http.Request) string {
	v, _ := r.Context().Value(CtxKeyEmail).(string)
	return v
}

func RawToken(r *http.Request) string {
	v, _ := r.Context().Value(CtxKeyToken).(string)
	return v
}

// ── Context 注入辅助 ──

func WithCandidate(ctx context.Context, id, email, name, token string) context.Context {
	ctx = context.WithValue(ctx, CtxKeyAdminID, id)
	ctx = context.WithValue(ctx, CtxKeyEmail, email)
	ctx = context.WithValue(ctx, CtxKeyName, name)
	ctx = context.WithValue(ctx, CtxKeyToken, token)
	return ctx
}
