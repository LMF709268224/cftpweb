package server

import (
	"github.com/LMF709268224/cftpproto/util"
	"log/slog"

	"candbff/config"

	gccpb "github.com/LMF709268224/cftpproto/gcc"
	gcredspb "github.com/LMF709268224/cftpproto/gcreds"
	gexampb "github.com/LMF709268224/cftpproto/gexam"
	lmspb "github.com/LMF709268224/cftpproto/glms"
	mallpb "github.com/LMF709268224/cftpproto/gmall"
	gmbrpb "github.com/LMF709268224/cftpproto/gmbr"
	gmidpb "github.com/LMF709268224/cftpproto/gmid"
	gmsgpb "github.com/LMF709268224/cftpproto/gmsg"
	gpaypb "github.com/LMF709268224/cftpproto/gpay"
	gprogpb "github.com/LMF709268224/cftpproto/gprog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GrpcClientPool 管理到所有下游微服务的 gRPC 连接
type GrpcClientPool struct {
	// gRPC 连接
	mallConn  *grpc.ClientConn
	lmsConn   *grpc.ClientConn
	gccConn   *grpc.ClientConn
	gprogConn *grpc.ClientConn
	gmsgConn  *grpc.ClientConn
	credsConn *grpc.ClientConn
	gexamConn *grpc.ClientConn
	gmidConn  *grpc.ClientConn
	gpayConn  *grpc.ClientConn
	gmbrConn  *grpc.ClientConn

	// gRPC 客户端
	Mall  mallpb.MallServiceClient
	Lms   lmspb.LmsServiceClient
	Gcc   gccpb.CCServiceClient
	Gprog gprogpb.ProgServiceClient
	Gmsg  gmsgpb.MessageServiceClient
	Creds gcredspb.CredentialServiceClient
	Gexam gexampb.GExamServiceClient
	Gmid  gmidpb.MidServiceClient
	Gpay  gpaypb.PayServiceClient
	Gmbr  gmbrpb.GmbrServiceClient
}

// dialGrpc 建立 gRPC 连接
func dialGrpc(addr string, creds credentials.TransportCredentials) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// grpcAddr 解析 gRPC 服务地址
// 优先读环境变量，否则拼接 K8s DNS: <service>.<namespace>.svc.cluster.local:<port>
func grpcAddr(envKey, service string) string {
	return util.GetEndpointAddress(envKey, service, "50051")
}

// NewGrpcClientPool 初始化到所有下游微服务的 gRPC 连接
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

// Close 关闭所有 gRPC 连接
func (p *GrpcClientPool) Close() {
	conns := []*grpc.ClientConn{
		p.mallConn, p.lmsConn, p.gccConn, p.gprogConn, p.gmsgConn,
		p.credsConn, p.gexamConn, p.gmidConn, p.gpayConn, p.gmbrConn,
	}
	for _, c := range conns {
		if c != nil {
			c.Close()
		}
	}
}
