package server

import (
	"github.com/afnandelfin620-star/cftptest/cftp/util"
	"log/slog"

	"adminbff/config"

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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GrpcClientPool 绠＄悊鍒版墍鏈変笅娓稿井鏈嶅姟鐨?gRPC 杩炴帴
type GrpcClientPool struct {
	// gRPC 杩炴帴
	mallConn  *grpc.ClientConn
	lmsConn   *grpc.ClientConn
	gccConn   *grpc.ClientConn
	gprogConn *grpc.ClientConn
	gmsgConn  *grpc.ClientConn
	credsConn *grpc.ClientConn
	gexamConn *grpc.ClientConn
	gmidConn  *grpc.ClientConn
	gmailConn *grpc.ClientConn
	gpayConn  *grpc.ClientConn
	gmbrConn  *grpc.ClientConn

	// gRPC 瀹㈡埛绔?
	Mall  mallpb.MallServiceClient
	Lms   lmspb.LmsServiceClient
	Gcc   gccpb.CCServiceClient
	Gprog gprogpb.ProgServiceClient
	Gmsg  gmsgpb.MessageServiceClient
	Creds gcredspb.CredentialServiceClient
	Gexam gexampb.GExamServiceClient
	Gmid  gmidpb.MidServiceClient
	Gmail gmailpb.MailServiceClient
	Gpay  gpaypb.PayServiceClient
	Gmbr  gmbrpb.GmbrServiceClient
}

// dialGrpc 寤虹珛 gRPC 杩炴帴
func dialGrpc(addr string, creds credentials.TransportCredentials) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// grpcAddr 瑙ｆ瀽 gRPC 鏈嶅姟鍦板潃
// 浼樺厛璇荤幆澧冨彉閲忥紝鍚﹀垯鎷兼帴 K8s DNS: <service>.<namespace>.svc.cluster.local:<port>
func grpcAddr(envKey, service string) string {
	return util.GetEndpointAddress(envKey, service, "50051")
}

// NewGrpcClientPool 鍒濆鍖栧埌鎵€鏈変笅娓稿井鏈嶅姟鐨?gRPC 杩炴帴
func NewGrpcClientPool(creds credentials.TransportCredentials) (*GrpcClientPool, error) {
	pool := &GrpcClientPool{}

	var err error

	// --- mall ---
	addr := grpcAddr(config.EnvMallGrpcAddr, "gmall")
	pool.mallConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "mall", "addr", addr, "error", err)
		return nil, err
	}
	pool.Mall = mallpb.NewMallServiceClient(pool.mallConn)

	// --- glms ---
	addr = grpcAddr(config.EnvLmsGrpcAddr, "glms")
	pool.lmsConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "glms", "addr", addr, "error", err)
		return nil, err
	}
	pool.Lms = lmspb.NewLmsServiceClient(pool.lmsConn)

	// --- gcc ---
	addr = grpcAddr(config.EnvGccGrpcAddr, "gcc")
	pool.gccConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "gcc", "addr", addr, "error", err)
		return nil, err
	}
	pool.Gcc = gccpb.NewCCServiceClient(pool.gccConn)

	// --- gprog ---
	addr = grpcAddr(config.EnvGprogGrpcAddr, "gprog")
	pool.gprogConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "gprog", "addr", addr, "error", err)
		return nil, err
	}
	pool.Gprog = gprogpb.NewProgServiceClient(pool.gprogConn)

	// --- gmsg ---
	addr = grpcAddr(config.EnvGmsgGrpcAddr, "gmsg")
	pool.gmsgConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "gmsg", "addr", addr, "error", err)
		return nil, err
	}
	pool.Gmsg = gmsgpb.NewMessageServiceClient(pool.gmsgConn)

	// --- gcreds ---
	addr = grpcAddr(config.EnvGcredsGrpcAddr, "gcreds")
	pool.credsConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "gcreds", "addr", addr, "error", err)
		return nil, err
	}
	pool.Creds = gcredspb.NewCredentialServiceClient(pool.credsConn)

	// --- gexam ---
	addr = grpcAddr(config.EnvGexamGrpcAddr, "gexam")
	pool.gexamConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "gexam", "addr", addr, "error", err)
		return nil, err
	}
	pool.Gexam = gexampb.NewGExamServiceClient(pool.gexamConn)

	// --- gmid ---
	addr = grpcAddr(config.EnvGmidGrpcAddr, "gmid")
	pool.gmidConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "gmid", "addr", addr, "error", err)
		return nil, err
	}
	pool.Gmid = gmidpb.NewMidServiceClient(pool.gmidConn)

	// --- gmail ---
	addr = grpcAddr(config.EnvGmailGrpcAddr, "gmail")
	pool.gmailConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "gmail", "addr", addr, "error", err)
		return nil, err
	}
	pool.Gmail = gmailpb.NewMailServiceClient(pool.gmailConn)

	// --- gpay ---
	addr = grpcAddr(config.EnvGpayGrpcAddr, "gpay")
	pool.gpayConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "gpay", "addr", addr, "error", err)
		return nil, err
	}
	pool.Gpay = gpaypb.NewPayServiceClient(pool.gpayConn)

	// --- gmbr ---
	addr = grpcAddr(config.EnvGmbrGrpcAddr, "gmbr")
	pool.gmbrConn, err = dialGrpc(addr, creds)
	if err != nil {
		slog.Error("Failed to create gRPC client", "service", "gmbr", "addr", addr, "error", err)
		return nil, err
	}
	pool.Gmbr = gmbrpb.NewGmbrServiceClient(pool.gmbrConn)

	return pool, nil
}

// Close 鍏抽棴鎵€鏈?gRPC 杩炴帴
func (p *GrpcClientPool) Close() {
	conns := []*grpc.ClientConn{
		p.mallConn, p.lmsConn, p.gccConn, p.gprogConn, p.gmsgConn,
		p.credsConn, p.gexamConn, p.gmidConn, p.gmailConn, p.gpayConn, p.gmbrConn,
	}
	for _, c := range conns {
		if c != nil {
			c.Close()
		}
	}
}
