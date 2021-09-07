package grpcutil

import (
	"context"

	"github.com/fullstorydev/grpchan"
	grpcvalidate "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// IgnoreAuthMix implements the ServiceAuthFuncOverride interface to ignore any
// auth requirements set by github.com/grpc-ecosystem/go-grpc-middleware/auth.
type IgnoreAuthMixin struct{}

func (m IgnoreAuthMixin) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, nil
}

// AuthlessHealthServer implements a gRPC health endpoint that will ignore any auth
// requirements set by github.com/grpc-ecosystem/go-grpc-middleware/auth.
type AuthlessHealthServer struct {
	*health.Server
	IgnoreAuthMixin
}

// NewAuthlessHealthServer returns a new gRPC health server that ignores auth
// middleware.
func NewAuthlessHealthServer() *AuthlessHealthServer {
	return &AuthlessHealthServer{Server: health.NewServer()}
}

// SetServicesHealthy sets the service to SERVING
func (s *AuthlessHealthServer) SetServicesHealthy(svcDesc ...*grpc.ServiceDesc) {
	for _, d := range svcDesc {
		s.SetServingStatus(
			d.ServiceName,
			healthpb.HealthCheckResponse_SERVING,
		)
	}
}

var defaultUnaryMiddleware = grpcvalidate.UnaryServerInterceptor()

// RegisterService registers a service on the registrar with the default middleware enabled
func RegisterService(srv grpc.ServiceRegistrar, svcDesc *grpc.ServiceDesc, ss interface{}) {
	srv.RegisterService(grpchan.InterceptServer(svcDesc, defaultUnaryMiddleware, nil), ss)
}
