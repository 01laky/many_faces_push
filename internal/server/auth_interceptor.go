// Package server implements the gRPC push surface (auth interceptor + PushService).
package server

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Metadata key agreed with many_faces_backend PushOptions.WorkerAuthToken (gRPC metadata keys are case-insensitive on the wire).
const metadataWorkerTokenKey = "x-push-worker-token"

// UnaryAuthInterceptor enforces PUSH_WORKER_EXPECTED_TOKEN when non-empty.
// grpc.health.v1 is exempt so orchestrators can probe without secrets; application RPCs still require the header when configured.
func UnaryAuthInterceptor(expectedToken string) grpc.UnaryServerInterceptor {
	if expectedToken == "" {
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing gRPC metadata")
		}
		vals := md.Get(metadataWorkerTokenKey)
		if len(vals) != 1 || vals[0] != expectedToken {
			return nil, status.Error(codes.Unauthenticated, "invalid or missing x-push-worker-token")
		}
		return handler(ctx, req)
	}
}
