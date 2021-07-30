package grpcutil_test

import (
	"log"

	"github.com/authzed/grpcutil"
	"google.golang.org/grpc"
)

func ExampleWithSystemCertificates() {
	_, err := grpc.Dial(
		"grpc.authzed.com:443",
		grpcutil.WithSystemCerts(grpcutil.VerifyCA),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleWithBearerToken() {
	_, err := grpc.Dial(
		"grpc.authzed.com:443",
		grpcutil.WithSystemCerts(grpcutil.VerifyCA),
		grpcutil.WithBearerToken("t_your_token_here_1234567deadbeef"),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleWithInsecureBearerToken() {
	_, err := grpc.Dial(
		"grpc.authzed.com:443",
		grpc.WithInsecure(),
		grpcutil.WithInsecureBearerToken("t_your_token_here_1234567deadbeef"),
	)
	if err != nil {
		log.Fatal(err)
	}
}
