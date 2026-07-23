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
	CtxKeyCandidateID ContextKey = "candidate_id"
	CtxKeyEmail       ContextKey = "email"
	CtxKeyName        ContextKey = "name"
	CtxKeyToken       ContextKey = "raw_token"
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
	Gpay                gpaypb.PayServiceClient
	Gmbr                gmbrpb.GmbrServiceClient
	Gmail               gmailpb.MailServiceClient
	CasdoorEndpoint     string
	CasdoorClientId     string
	CasdoorClientSecret string
	CasdoorAppName      string
	CasdoorOrgName      string
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
	gpay gpaypb.PayServiceClient,
	gmbr gmbrpb.GmbrServiceClient,
	gmail gmailpb.MailServiceClient,
	casdoorEndpoint string,
	casdoorClientId string,
	casdoorClientSecret string,
	casdoorAppName string,
	casdoorOrgName string,
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
		Gpay:                gpay,
		Gmbr:                gmbr,
		Gmail:               gmail,
		CasdoorEndpoint:     casdoorEndpoint,
		CasdoorClientId:     casdoorClientId,
		CasdoorClientSecret: casdoorClientSecret,
		CasdoorAppName:      casdoorAppName,
		CasdoorOrgName:      casdoorOrgName,
	}
}

// ── Context 取值辅助函数 ──

func CandidateID(r *http.Request) string {
	v, _ := r.Context().Value(CtxKeyCandidateID).(string)
	return v
}

func CandidateName(r *http.Request) string {
	v, _ := r.Context().Value(CtxKeyName).(string)
	return v
}

// ── Context 注入辅助 ──

func WithCandidate(ctx context.Context, id, email, name, token string) context.Context {
	ctx = context.WithValue(ctx, CtxKeyCandidateID, id)
	ctx = context.WithValue(ctx, CtxKeyEmail, email)
	ctx = context.WithValue(ctx, CtxKeyName, name)
	ctx = context.WithValue(ctx, CtxKeyToken, token)
	return ctx
}
