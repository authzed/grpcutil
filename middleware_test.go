package grpcutil

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	testpb "google.golang.org/grpc/test/grpc_testing"
)

func TestWrapMethodsNoop(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	s.RegisterService(WrapMethods(testpb.TestService_ServiceDesc, NoopUnaryInterceptor), &testServer{})
	go s.Serve(lis)

	conn, err := grpc.Dial("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	client := testpb.NewTestServiceClient(conn)
	_, err = client.UnaryCall(context.Background(), &testpb.SimpleRequest{ResponseType: testpb.PayloadType_COMPRESSABLE, ResponseSize: 1})
	if err != nil {
		t.Error(err)
	}
}

func TestWrapMethods(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()

	middleware := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		sr := req.(*testpb.SimpleRequest)
		sr.ResponseSize += 1
		return handler(ctx, req)
	}
	s.RegisterService(WrapMethods(testpb.TestService_ServiceDesc, middleware), &testServer{})
	go s.Serve(lis)

	conn, err := grpc.Dial("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	client := testpb.NewTestServiceClient(conn)
	resp, err := client.UnaryCall(context.Background(), &testpb.SimpleRequest{ResponseType: testpb.PayloadType_COMPRESSABLE, ResponseSize: 1})
	if err != nil {
		t.Error(err)
	}

	if len(resp.Payload.Body) != 2 {
		t.Error("request not intercepted")
	}
}

func TestWrapMethodsAndServerInterceptor(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)

	serverMiddleware := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		r, err := handler(ctx, req)
		sr := r.(*testpb.SimpleResponse)
		sr.Payload.Body = append(sr.Payload.Body, byte(2))
		return sr, err
	}
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(serverMiddleware))

	middleware := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		r, err := handler(ctx, req)
		sr := r.(*testpb.SimpleResponse)
		sr.Payload.Body = append(sr.Payload.Body, byte(1))
		return sr, err
	}
	s.RegisterService(WrapMethods(testpb.TestService_ServiceDesc, middleware), &testServer{})
	go s.Serve(lis)

	conn, err := grpc.Dial("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	client := testpb.NewTestServiceClient(conn)
	resp, err := client.UnaryCall(context.Background(), &testpb.SimpleRequest{ResponseType: testpb.PayloadType_COMPRESSABLE, ResponseSize: 0})
	if err != nil {
		t.Error(err)
	}

	// middleware happens before server middleware
	if string(resp.Payload.Body) != "\u0000\u0001\u0002" {
		t.Errorf("request not intercepted, got %b", resp.Payload.Body)
	}
}

type testServer struct {
	testpb.UnimplementedTestServiceServer

	security           string // indicate the authentication protocol used by this server.
	earlyFail          bool   // whether to error out the execution of a service handler prematurely.
	setAndSendHeader   bool   // whether to call setHeader and sendHeader.
	setHeaderOnly      bool   // whether to only call setHeader, not sendHeader.
	multipleSetTrailer bool   // whether to call setTrailer multiple times.
	unaryCallSleepTime time.Duration
}

func (s *testServer) EmptyCall(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		var str []string
		for _, entry := range md["user-agent"] {
			str = append(str, "ua", entry)
		}
		grpc.SendHeader(ctx, metadata.Pairs(str...))
	}
	return new(testpb.Empty), nil
}

func (s *testServer) UnaryCall(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
	payload, err := newPayload(in.GetResponseType(), in.GetResponseSize())
	if err != nil {
		return nil, err
	}

	return &testpb.SimpleResponse{
		Payload: payload,
	}, nil
}

func newPayload(t testpb.PayloadType, size int32) (*testpb.Payload, error) {
	if size < 0 {
		return nil, fmt.Errorf("requested a response with invalid length %d", size)
	}
	body := make([]byte, size)
	switch t {
	case testpb.PayloadType_COMPRESSABLE:
	case testpb.PayloadType_UNCOMPRESSABLE:
		return nil, fmt.Errorf("PayloadType UNCOMPRESSABLE is not supported")
	default:
		return nil, fmt.Errorf("unsupported payload type: %d", t)
	}
	return &testpb.Payload{
		Type: t,
		Body: body,
	}, nil
}
