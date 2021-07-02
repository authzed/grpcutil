// Package grpcutil implements various utilities to simplify common gRPC APIs.
package grpcutil

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// WithSystemCerts is a dial option for requiring TLS with the system
// certificate pool.
//
// This function panics if the system pool cannot be loaded.
func WithSystemCerts(insecureSkipVerify bool) grpc.DialOption {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		panic(err)
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs:            certPool,
		InsecureSkipVerify: insecureSkipVerify,
	}))
}

type grpcMetadataCreds map[string]string

func (gmc grpcMetadataCreds) RequireTransportSecurity() bool { return true }
func (gmc grpcMetadataCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return gmc, nil
}

// WithBearerToken is a dial option to add a standard HTTP Bearer token to all
// requests sent from a client.
func WithBearerToken(token string) grpc.DialOption {
	return grpc.WithPerRPCCredentials(grpcMetadataCreds{"authorization": "Bearer " + token})
}
