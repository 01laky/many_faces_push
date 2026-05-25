// Package config loads push-worker settings from the process environment (12-factor style).
// Defaults match docker-compose.yml in this repository; the monorepo may override via env_file.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime knobs for the push worker process.
type Config struct {
	// GRPCListen is the TCP bind address for gRPC, e.g. ":50053".
	GRPCListen string

	// ExpectedWorkerToken, when non-empty, requires unary RPCs (except grpc.health) to send metadata "x-push-worker-token".
	ExpectedWorkerToken string

	// GoogleApplicationCredentials is a filesystem path to the Firebase service account JSON (FCM via Admin SDK).
	// When empty, the worker starts but SendPush returns FailedPrecondition so local stacks without Firebase still boot.
	GoogleApplicationCredentials string

	// EnableReflection registers gRPC server reflection (grpcurl). Disable in production.
	EnableReflection bool

	// GrpcTLSCertFile / GrpcTLSKeyFile enable TLS on the gRPC listener (both required together).
	GrpcTLSCertFile string
	GrpcTLSKeyFile  string
	// GrpcMTLSClientCAFile, when set with TLS cert+key, requires client certificates signed by this CA bundle.
	GrpcMTLSClientCAFile string
}

const (
	// EnvGRPCListen binds the gRPC server.
	EnvGRPCListen = "PUSH_WORKER_GRPC_LISTEN"
	// EnvExpectedToken enables shared-secret auth between many_faces_backend and this worker.
	EnvExpectedToken = "PUSH_WORKER_EXPECTED_TOKEN"
	// EnvGoogleApplicationCredentials points at the service account JSON file (never commit that file).
	EnvGoogleApplicationCredentials = "GOOGLE_APPLICATION_CREDENTIALS"
	// EnvEnableReflection accepts "1", "true", "yes" (case-insensitive) to enable gRPC reflection.
	EnvEnableReflection = "PUSH_WORKER_GRPC_REFLECTION"
	// EnvGrpcTLSCertFile / EnvGrpcTLSKeyFile enable TLS on the gRPC server (see monorepo docs/guides/push-grpc-tls-mtls.md).
	EnvGrpcTLSCertFile = "PUSH_WORKER_GRPC_TLS_CERT_FILE"
	EnvGrpcTLSKeyFile  = "PUSH_WORKER_GRPC_TLS_KEY_FILE"
	// EnvGrpcMTLSClientCAFile requires verified client certs when set (mTLS).
	EnvGrpcMTLSClientCAFile = "PUSH_WORKER_GRPC_MTLS_CLIENT_CA_FILE"
)

// LoadFromEnv parses environment variables into Config. Missing optional values use safe defaults.
func LoadFromEnv() (*Config, error) {
	listen := strings.TrimSpace(os.Getenv(EnvGRPCListen))
	if listen == "" {
		listen = ":50053"
	}

	reflection := parseBool(os.Getenv(EnvEnableReflection))

	return &Config{
		GRPCListen:                   listen,
		ExpectedWorkerToken:          strings.TrimSpace(os.Getenv(EnvExpectedToken)),
		GoogleApplicationCredentials: strings.TrimSpace(os.Getenv(EnvGoogleApplicationCredentials)),
		EnableReflection:             reflection,
		GrpcTLSCertFile:              strings.TrimSpace(os.Getenv(EnvGrpcTLSCertFile)),
		GrpcTLSKeyFile:               strings.TrimSpace(os.Getenv(EnvGrpcTLSKeyFile)),
		GrpcMTLSClientCAFile:         strings.TrimSpace(os.Getenv(EnvGrpcMTLSClientCAFile)),
	}, nil
}

func parseBool(raw string) bool {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return false
	}
	switch s {
	case "1", "true", "yes", "on":
		return true
	default:
		if v, err := strconv.ParseBool(raw); err == nil {
			return v
		}
		return false
	}
}

// Validate returns an error only for inconsistent combinations operators must fix in compose.
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if (c.GrpcTLSCertFile != "") != (c.GrpcTLSKeyFile != "") {
		return fmt.Errorf("%s and %s must both be set for TLS, or both empty", EnvGrpcTLSCertFile, EnvGrpcTLSKeyFile)
	}
	return nil
}
