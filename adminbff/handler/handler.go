package handler

import (
	"context"
	"net/http"

	gccpb "github.com/LMF709268224/cftpproto/gcc"
	gcredspb "github.com/LMF709268224/cftpproto/gcreds"
	gexampb "github.com/LMF709268224/cftpproto/gexam"
	lmspb "github.com/LMF709268224/cftpproto/glms"
	gmailpb "github.com/LMF709268224/cftpproto/gmail"
	mallpb "github.com/LMF709268224/cftpproto/gmall"
	gmidpb "github.com/LMF709268224/cftpproto/gmid"
	gmsgpb "github.com/LMF709268224/cftpproto/gmsg"
	gpaypb "github.com/LMF709268224/cftpproto/gpay"
	gprogpb "github.com/LMF709268224/cftpproto/gprog"
)

// 鈹€鈹€ Context 閿?鈹€鈹€

type ContextKey string

const (
	CtxKeyAdminID ContextKey = "admin_id"
	CtxKeyEmail   ContextKey = "email"
	CtxKeyName    ContextKey = "name"
	CtxKeyToken   ContextKey = "raw_token"
)

// 鈹€鈹€ Handler 瀹瑰櫒 鈹€鈹€

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
	CasdoorEndpoint     string
	CasdoorClientId     string
	CasdoorClientSecret string
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
	casdoorEndpoint string,
	casdoorClientId string,
	casdoorClientSecret string,
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
		CasdoorEndpoint:     casdoorEndpoint,
		CasdoorClientId:     casdoorClientId,
		CasdoorClientSecret: casdoorClientSecret,
	}
}

// 鈹€鈹€ Context 鍙栧€艰緟鍔╁嚱鏁?鈹€鈹€

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

// 鈹€鈹€ Context 娉ㄥ叆杈呭姪 鈹€鈹€

func WithCandidate(ctx context.Context, id, email, name, token string) context.Context {
	ctx = context.WithValue(ctx, CtxKeyAdminID, id)
	ctx = context.WithValue(ctx, CtxKeyEmail, email)
	ctx = context.WithValue(ctx, CtxKeyName, name)
	ctx = context.WithValue(ctx, CtxKeyToken, token)
	return ctx
}
