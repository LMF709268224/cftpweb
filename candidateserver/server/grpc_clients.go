package server

import (
	"github.com/afnandelfin620-star/cftptest/cftp/util"
	"log/slog"

	"candidateserver/config"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"

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

	// gRPC 客户端
	Mall  mallpb.MallServiceClient
	Lms   lmspb.LmsServiceClient
	Gcc   gccpb.CCServiceClient
	Gprog gprogpb.ProgServiceClient
	Gmsg  gmsgpb.MessageServiceClient
	Creds gcredspb.CredentialServiceClient
	Gexam gexampb.GExamServiceClient
	Gmid  gmidpb.MidServiceClient
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
	addr := grpcAddr(config.EnvMallGrpcAddr, "mall")
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

	return pool, nil
}

// Close 关闭所有 gRPC 连接
func (p *GrpcClientPool) Close() {
	conns := []*grpc.ClientConn{
		p.mallConn, p.lmsConn, p.gccConn, p.gprogConn, p.gmsgConn,
		p.credsConn, p.gexamConn, p.gmidConn,
	}
	for _, c := range conns {
		if c != nil {
			c.Close()
		}
	}
}
