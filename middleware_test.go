package grpcutil

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/authzed/grpcutil/internal/testpb"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestSplitMethodName(t *testing.T) {
	serviceName, methodName := SplitMethodName("/authzed.api.v0.ACLService/Check")
	if serviceName != "authzed.api.v0.ACLService" || methodName != "Check" {
		t.Errorf("expected 'authzed.api.v0.ACLService' 'Check' , got %s %s", serviceName, methodName)
	}
}

func TestWrapMethodsNoop(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	s.RegisterService(WrapMethods(testpb.HelloService_ServiceDesc, NoopUnaryInterceptor), &testServer{})
	go func() {
		_ = s.Serve(lis)
	}()

	conn, err := grpc.Dial("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client := testpb.NewHelloServiceClient(conn)
	resp, err := client.HelloUnary(context.Background(), &testpb.HelloRequest{Message: "hi"})
	require.NoError(t, err)
	require.Equal(t, "hi", resp.Message)
}

func TestWrapMethods(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()

	middleware := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		sr := req.(*testpb.HelloRequest)
		sr.Message = "yep"
		return handler(ctx, req)
	}
	s.RegisterService(WrapMethods(testpb.HelloService_ServiceDesc, middleware), &testServer{})
	go func() {
		_ = s.Serve(lis)
	}()

	conn, err := grpc.Dial("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client := testpb.NewHelloServiceClient(conn)
	resp, err := client.HelloUnary(context.Background(), &testpb.HelloRequest{Message: "hi"})
	require.NoError(t, err)
	require.Equal(t, "yep", resp.Message, "request not intercepted")
}

func TestWrapMethodsAndServerInterceptor(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)

	serverMiddleware := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		r, err := handler(ctx, req)
		sr := r.(*testpb.HelloResponse)
		sr.Message += ",friend"
		return sr, err
	}
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(serverMiddleware))

	middleware := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		r, err := handler(ctx, req)
		sr := r.(*testpb.HelloResponse)
		sr.Message += ",sup"
		return sr, err
	}
	s.RegisterService(WrapMethods(testpb.HelloService_ServiceDesc, middleware), &testServer{})
	go func() {
		_ = s.Serve(lis)
	}()

	conn, err := grpc.Dial("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client := testpb.NewHelloServiceClient(conn)
	resp, err := client.HelloUnary(context.Background(), &testpb.HelloRequest{Message: "hi"})
	require.NoError(t, err)

	// middleware happens before server middleware
	require.Equal(t, "yep,sup,friend", resp.Message, "request not intercepted, got %s", resp.Message)
}

func TestWrapStreams(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()

	counter := 0
	s.RegisterService(WrapStreams(testpb.HelloService_ServiceDesc, StreamMiddleware(&counter)), &testServer{})
	go func() {
		_ = s.Serve(lis)
	}()

	conn, err := grpc.Dial("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client := testpb.NewHelloServiceClient(conn)
	stream, err := client.HelloStreaming(context.Background(), &testpb.HelloRequest{Message: "hi"})
	require.NoError(t, err)

	err = func() error {
		for {
			_, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}()
	require.NoError(t, err)
	require.Equal(t, 1, counter, "stream not intercepted")
}

func TestWrapStreamsAndServerInterceptor(t *testing.T) {
	serverCounter := 0
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer(grpc.ChainStreamInterceptor(StreamMiddleware(&serverCounter)))

	counter := 0
	s.RegisterService(WrapStreams(testpb.HelloService_ServiceDesc, StreamMiddleware(&counter)), &testServer{})
	go func() {
		_ = s.Serve(lis)
	}()

	conn, err := grpc.Dial("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client := testpb.NewHelloServiceClient(conn)
	stream, err := client.HelloStreaming(context.Background(), &testpb.HelloRequest{Message: "hi"})
	require.NoError(t, err)

	err = func() error {
		for {
			_, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}()
	require.NoError(t, err)

	require.Equal(t, 1, counter, "service stream not intercepted")
	require.Equal(t, 1, serverCounter, "server stream not intercepted")
}

func StreamMiddleware(counter *int) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := &recvWrapper{stream, counter}
		return handler(srv, wrapper)
	}
}

type recvWrapper struct {
	grpc.ServerStream
	counter *int
}

func (s *recvWrapper) RecvMsg(m any) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}

	*s.counter++

	return nil
}

type testServer struct {
	testpb.UnimplementedHelloServiceServer
}

func (s *testServer) HelloUnary(_ context.Context, in *testpb.HelloRequest) (*testpb.HelloResponse, error) {
	return &testpb.HelloResponse{Message: in.Message}, nil
}

func (s *testServer) HelloStreaming(args *testpb.HelloRequest, stream testpb.HelloService_HelloStreamingServer) error {
	return stream.Send(&testpb.HelloResponse{Message: args.Message})
}
