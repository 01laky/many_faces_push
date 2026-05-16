package server

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pushv1 "github.com/01laky/many_faces_push/gen/manyfaces/push/v1"
)

type noopPush struct {
	pushv1.UnimplementedPushServiceServer
}

func (noopPush) SendPush(context.Context, *pushv1.SendPushRequest) (*pushv1.SendPushResponse, error) {
	return &pushv1.SendPushResponse{}, nil
}

func TestUnaryAuthInterceptor_NoTokenRequiredWhenExpectedEmpty(t *testing.T) {
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(UnaryAuthInterceptor("")))
	pushv1.RegisterPushServiceServer(srv, noopPush{})

	lis := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ctx := context.Background()
	if _, err := pushv1.NewPushServiceClient(conn).SendPush(ctx, &pushv1.SendPushRequest{}); err != nil {
		t.Fatalf("SendPush: %v", err)
	}
}

func TestUnaryAuthInterceptor_HealthCheckWithoutMetadataWhenSecretSet(t *testing.T) {
	const secret = "push-secret"
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(UnaryAuthInterceptor(secret)))
	pushv1.RegisterPushServiceServer(srv, noopPush{})

	hs := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, hs)
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	lis := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ctx := context.Background()
	if _, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{}); err != nil {
		t.Fatalf("health Check: %v", err)
	}
}

func TestUnaryAuthInterceptor_SendPushRejectsWrongToken(t *testing.T) {
	const secret = "push-secret"
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(UnaryAuthInterceptor(secret)))
	pushv1.RegisterPushServiceServer(srv, noopPush{})

	lis := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	md := metadata.Pairs("x-push-worker-token", "nope")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	_, err = pushv1.NewPushServiceClient(conn).SendPush(ctx, &pushv1.SendPushRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("got %v", err)
	}
}

func TestUnaryAuthInterceptor_RejectsWhenMetadataMissing(t *testing.T) {
	ic := UnaryAuthInterceptor("push-secret")
	info := &grpc.UnaryServerInfo{FullMethod: "/manyfaces.push.v1.PushService/SendPush"}
	_, err := ic(context.Background(), nil, info, func(context.Context, any) (any, error) {
		t.Fatal("handler should not run")
		return nil, nil
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestUnaryAuthInterceptor_RejectsDuplicateTokenHeaders(t *testing.T) {
	ic := UnaryAuthInterceptor("push-secret")
	info := &grpc.UnaryServerInfo{FullMethod: "/manyfaces.push.v1.PushService/SendPush"}
	md := metadata.Pairs("x-push-worker-token", "push-secret", "x-push-worker-token", "push-secret")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err := ic(ctx, nil, info, func(context.Context, any) (any, error) {
		t.Fatal("handler should not run")
		return nil, nil
	})
	if err == nil {
		t.Fatal("expected error for duplicate metadata values")
	}
}

func TestUnaryAuthInterceptor_SendPushAllowsCorrectToken(t *testing.T) {
	const secret = "push-secret"
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(UnaryAuthInterceptor(secret)))
	pushv1.RegisterPushServiceServer(srv, noopPush{})

	lis := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	md := metadata.Pairs("x-push-worker-token", secret)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	if _, err := pushv1.NewPushServiceClient(conn).SendPush(ctx, &pushv1.SendPushRequest{}); err != nil {
		t.Fatal(err)
	}
}
