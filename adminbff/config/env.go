package config

// 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€
// 鐜鍙橀噺闆嗕腑瀹氫箟
// 鎵€鏈夌幆澧冨彉閲忚鍙栫粺涓€浣跨敤姝ゆ枃浠朵腑瀹氫箟鐨勫父閲忥紝鏂逛究杩愮淮鏌ョ湅闇€瑕侀厤缃摢浜涘彉閲?
// 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

const (
	// 鈹€鈹€ 鏈嶅姟鍣ㄧ洃鍚湴鍧€ 鈹€鈹€
	// HTTP_ADDRESS  HTTP 鏈嶅姟鍣ㄧ洃鍚湴鍧€锛岄粯璁?"0.0.0.0:8080"
	EnvHTTPAddress = "HTTP_ADDRESS"

	// 鈹€鈹€ TLS 璇佷功鐩綍 鈹€鈹€
	// TLS_DIR  TLS 璇佷功鏂囦欢鐩綍锛岀洰褰曞唴闇€鍖呭惈 ca.crt
	//          鏈缃椂鎵€鏈?gRPC 瀹㈡埛绔互 insecure 妯″紡杩愯
	EnvTLSDir = "TLS_DIR"

	// 鈹€鈹€ 閰嶇疆涓績 鈹€鈹€
	// CFGSERVER_ADDR  cfgserver 閰嶇疆涓績鍦板潃
	//                 鏈缃椂鑷姩鎷兼帴 K8s DNS: cfgserver.<ns>.svc.cluster.local:50051
	EnvCfgServerAddr = "CFGSERVER_ADDR"

	// 鈹€鈹€ Casdoor IAM 鈹€鈹€
	// CASDOOR_ENDPOINT  Casdoor 鏈嶅姟鍦板潃
	//                   鏈缃椂鑷姩鎷兼帴 K8s DNS: casdoor.<ns>.svc.cluster.local:8000
	EnvCasdoorEndpoint = "CASDOOR_ENDPOINT"

	// CASDOOR_PUBLIC_ENDPOINT  Casdoor 鍏綉璁块棶鍦板潃锛岀敤浜庡墠绔烦杞?
	EnvCasdoorPublicEndpoint = "CASDOOR_PUBLIC_ENDPOINT"

	// ROLE_ADMIN_BASIC 绠＄悊鍛樺熀纭€瑙掕壊鍚嶏紝榛樿涓?"role_admin_basic"
	EnvRoleAdminBasic = "ROLE_ADMIN_BASIC"

	// 鈹€鈹€ 涓嬫父寰湇鍔?gRPC 鍦板潃 鈹€鈹€
	// 姣忎釜鍙橀噺瀵瑰簲涓€涓笅娓告湇鍔★紝
	// 鏈缃椂鑷姩鎷兼帴 K8s DNS 鍚? <service>.<ns>.svc.cluster.local:50051

	// MALL_GRPC_ADDR  gmall 鍟嗗煄鏈嶅姟鍦板潃
	EnvMallGrpcAddr = "MALL_GRPC_ADDR"
	// LMS_GRPC_ADDR   glms 璇剧▼/璧勬枡鏈嶅姟鍦板潃
	EnvLmsGrpcAddr = "LMS_GRPC_ADDR"
	// GCC_GRPC_ADDR gcc 璇佷功鏈嶅姟鍦板潃
	EnvGccGrpcAddr = "GCC_GRPC_ADDR"
	// GPROG_GRPC_ADDR gprog 杩涘害鏈嶅姟鍦板潃
	EnvGprogGrpcAddr = "GPROG_GRPC_ADDR"
	// GMSG_GRPC_ADDR gmsg 娑堟伅鏈嶅姟鍦板潃
	EnvGmsgGrpcAddr = "GMSG_GRPC_ADDR"
	// GCREDS_GRPC_ADDR gcreds 鍑瘉鏈嶅姟鍦板潃
	EnvGcredsGrpcAddr = "GCREDS_GRPC_ADDR"
	// GEXAM_GRPC_ADDR gexam 鑰冭瘯鏈嶅姟鍦板潃
	EnvGexamGrpcAddr = "GEXAM_GRPC_ADDR"
	// GMID_GRPC_ADDR gmid ID鏄犲皠鏈嶅姟鍦板潃
	EnvGmidGrpcAddr = "GMID_GRPC_ADDR"
	// GMAIL_GRPC_ADDR gmail 閭欢鏈嶅姟鍦板潃
	EnvGmailGrpcAddr = "GMAIL_GRPC_ADDR"
	// GPAY_GRPC_ADDR gpay 鏀粯鏈嶅姟鍦板潃
	EnvGpayGrpcAddr = "GPAY_GRPC_ADDR"
	// GMBR_GRPC_ADDR gmbr membership service address
	EnvGmbrGrpcAddr = "GMBR_GRPC_ADDR"

	// 鈹€鈹€ CORS 鈹€鈹€
	// CORS_ALLOWED_ORIGINS  鍏佽鐨勮法鍩熸潵婧愶紝閫楀彿鍒嗛殧锛岄粯璁?"*" 鍏佽鎵€鏈?
	EnvCORSOrigins = "CORS_ALLOWED_ORIGINS"

	// 鈹€鈹€ 鏃ュ織 鈹€鈹€
	// LOG_LEVEL  鏃ュ織绾у埆: debug / info / warn / error锛岄粯璁?info
	EnvLogLevel = "LOG_LEVEL"
	// LOG_SOURCE  鏄惁鍦ㄦ棩蹇椾腑杈撳嚭婧愮爜鏂囦欢鍜岃鍙凤紝璁剧疆涓?"true" 寮€鍚?
	EnvLogSource = "LOG_SOURCE"
)
