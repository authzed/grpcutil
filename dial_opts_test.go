package grpcutil_test

import (
	"log"

	"google.golang.org/grpc"

	"github.com/authzed/grpcutil"
)

func ExampleWithSystemCerts() {
	withSysCerts, err := grpcutil.WithSystemCerts(grpcutil.VerifyCA)
	if err != nil {
		log.Fatal(err)
	}

	_, err = grpc.Dial("grpc.authzed.com:443", withSysCerts)
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleWithBearerToken() {
	withSystemCerts, err := grpcutil.WithSystemCerts(grpcutil.VerifyCA)
	if err != nil {
		log.Fatal(err)
	}

	_, err = grpc.Dial(
		"grpc.authzed.com:443",
		withSystemCerts,
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
