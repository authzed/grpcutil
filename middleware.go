package grpcutil

import (
	"context"

	"google.golang.org/grpc/health"
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

// NewAuthlessServer returns a new gRPC health server that ignores auth
// middleware.
func NewAuthlessHealthServer() *AuthlessHealthServer {
	return &AuthlessHealthServer{Server: health.NewServer()}
}
