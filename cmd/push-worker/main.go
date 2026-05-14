// Command push-worker is the Many Faces FCM gRPC sidecar (many_faces_push).
// many_faces_backend calls it over gRPC; mobile apps and browsers must never reach this process directly.
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pushv1 "github.com/01laky/many_faces_push/gen/manyfaces/push/v1"
	"github.com/01laky/many_faces_push/internal/config"
	"github.com/01laky/many_faces_push/internal/grpccreds"
	"github.com/01laky/many_faces_push/internal/server"
)

func main() {
	// JSON logs on stdout keep parity with other Many Faces containers (Seq / Docker log drivers).
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	log := slog.New(handler)
	slog.SetDefault(log)

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		log.Error("configuration validation failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	var msgClient *messaging.Client
	if cfg.GoogleApplicationCredentials != "" {
		app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(cfg.GoogleApplicationCredentials))
		if err != nil {
			log.Error("failed to initialize firebase app", "error", err)
			os.Exit(1)
		}
		msgClient, err = app.Messaging(ctx)
		if err != nil {
			log.Error("failed to initialize firebase messaging client", "error", err)
			os.Exit(1)
		}
		log.Info("firebase messaging initialized", "credentials_path", cfg.GoogleApplicationCredentials)
	} else {
		log.Warn("GOOGLE_APPLICATION_CREDENTIALS not set — SendPush will return FailedPrecondition until a service account is mounted")
	}

	lis, err := net.Listen("tcp", cfg.GRPCListen)
	if err != nil {
		log.Error("failed to listen for gRPC", "addr", cfg.GRPCListen, "error", err)
		os.Exit(1)
	}

	serverCreds, err := grpccreds.LoadServerCredentials(cfg.GrpcTLSCertFile, cfg.GrpcTLSKeyFile, cfg.GrpcMTLSClientCAFile)
	if err != nil {
		log.Error("failed to configure gRPC TLS", "error", err)
		os.Exit(1)
	}

	var serverOpts []grpc.ServerOption
	if serverCreds != nil {
		serverOpts = append(serverOpts, grpc.Creds(serverCreds))
	}
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(server.UnaryAuthInterceptor(cfg.ExpectedWorkerToken)))
	grpcServer := grpc.NewServer(serverOpts...)

	pushv1.RegisterPushServiceServer(grpcServer, server.NewPushService(msgClient, log))

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	if cfg.EnableReflection {
		reflection.Register(grpcServer)
		log.Info("gRPC reflection enabled (disable in production via PUSH_WORKER_GRPC_REFLECTION)")
	}

	go func() {
		tlsMode := "plaintext"
		if serverCreds != nil {
			tlsMode = "tls"
			if cfg.GrpcMTLSClientCAFile != "" {
				tlsMode = "mtls"
			}
		}
		log.Info("push-worker gRPC listening", "addr", cfg.GRPCListen, "tls", tlsMode, "auth_token_configured", cfg.ExpectedWorkerToken != "")
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server stopped with error", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("shutdown signal received, stopping gRPC gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Info("gRPC graceful stop completed")
	case <-shutdownCtx.Done():
		log.Warn("graceful stop timed out, forcing stop")
		grpcServer.Stop()
	}
}
