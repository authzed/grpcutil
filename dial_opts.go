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

type secureMetadataCreds map[string]string

func (c secureMetadataCreds) RequireTransportSecurity() bool { return true }
func (c secureMetadataCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return c, nil
}

// WithBearerToken is a dial option to add a standard HTTP Bearer token to all
// requests sent from a client.
func WithBearerToken(token string) grpc.DialOption {
	return grpc.WithPerRPCCredentials(secureMetadataCreds{"authorization": "Bearer " + token})
}

type insecureMetadataCreds map[string]string

func (c insecureMetadataCreds) RequireTransportSecurity() bool { return false }
func (c insecureMetadataCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return c, nil
}

// WithInsecureBearerToken is a dial option to add a standard HTTP Bearer token
// to all requests sent from an insecure client.
//
// Must be used in conjunction with `grpc.WithInsecure()`.
func WithInsecureBearerToken(token string) grpc.DialOption {
	return grpc.WithPerRPCCredentials(insecureMetadataCreds{"authorization": "Bearer " + token})
}
