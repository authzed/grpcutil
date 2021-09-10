package grpcutil

import (
	"context"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/health"
)

// IgnoreAuthMixin is a struct that can be embedded to make a gRPC handler
// ignore any auth requirements set by the gRPC community auth middleware.
type IgnoreAuthMixin struct{}

var _ grpc_auth.ServiceAuthFuncOverride = (*IgnoreAuthMixin)(nil)

// AuthFuncOverride implements the grpc_auth.ServiceAuthFuncOverride by
// performing a no-op.
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
