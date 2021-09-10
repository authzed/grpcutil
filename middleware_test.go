package grpcutil

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"

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

func TestWrapStreams(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()

	counter := 0
	s.RegisterService(WrapStreams(testpb.TestService_ServiceDesc, StreamMiddleware(&counter)), &testServer{})
	go s.Serve(lis)

	conn, err := grpc.Dial("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	client := testpb.NewTestServiceClient(conn)
	stream, err := client.StreamingOutputCall(context.Background(), &testpb.StreamingOutputCallRequest{
		ResponseType: testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: []*testpb.ResponseParameters{
			{
				Size: 0,
			},
		},
	})
	if err != nil {
		t.Error(err)
	}

	if err := func() error {
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}(); err != nil {
		t.Error(err)
	}

	if counter != 1 {
		t.Error("stream not intercepted")
	}
}

func TestWrapStreamsAndServerInterceptor(t *testing.T) {
	serverCounter := 0
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer(grpc.ChainStreamInterceptor(StreamMiddleware(&serverCounter)))

	counter := 0
	s.RegisterService(WrapStreams(testpb.TestService_ServiceDesc, StreamMiddleware(&counter)), &testServer{})
	go s.Serve(lis)

	conn, err := grpc.Dial("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	client := testpb.NewTestServiceClient(conn)
	stream, err := client.StreamingOutputCall(context.Background(), &testpb.StreamingOutputCallRequest{
		ResponseType: testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: []*testpb.ResponseParameters{
			{
				Size: 0,
			},
		},
	})
	if err != nil {
		t.Error(err)
	}

	if err := func() error {
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}(); err != nil {
		t.Error(err)
	}

	if counter != 1 || serverCounter != 1 {
		t.Error("stream not intercepted")
	}
}

func StreamMiddleware(counter *int) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := &recvWrapper{stream, counter}
		return handler(srv, wrapper)
	}
}

type recvWrapper struct {
	grpc.ServerStream
	counter *int
}

func (s *recvWrapper) RecvMsg(m interface{}) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}

	*s.counter++

	return nil
}

type testServer struct {
	testpb.UnimplementedTestServiceServer
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

func (s *testServer) StreamingOutputCall(args *testpb.StreamingOutputCallRequest, stream testpb.TestService_StreamingOutputCallServer) error {
	cs := args.GetResponseParameters()
	for _, c := range cs {
		payload, err := newPayload(args.GetResponseType(), c.GetSize())
		if err != nil {
			return err
		}

		if err := stream.Send(&testpb.StreamingOutputCallResponse{
			Payload: payload,
		}); err != nil {
			return err
		}
	}
	return nil
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
