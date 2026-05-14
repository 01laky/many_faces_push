#!/usr/bin/env bash
# TLS + mTLS smoke for many_faces_push: generates a demo CA + server + client PEM chain,
# starts docker-compose.tls-smoke.yml, grpcurl grpc.health.v1.Health/Check, then optional .NET channel test.
#
# Prerequisites: bash, openssl, docker compose v2, grpcurl, dotnet 10 SDK when RUN_DOTNET_TLS_SMOKE=1 (default).
#
# Usage (from monorepo root):
#   chmod +x many_faces_push/scripts/smoke-grpc-tls.sh
#   many_faces_push/scripts/smoke-grpc-tls.sh
#
# Environment:
#   RUN_DOTNET_TLS_SMOKE=0 — skip dotnet test (grpcurl only).
#   PUSH_DIR — override path to many_faces_push (default: script parent).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUSH_DIR="${PUSH_DIR:-$(cd "$SCRIPT_DIR/.." && pwd)}"
COMPOSE_FILE="$PUSH_DIR/docker-compose.tls-smoke.yml"
PROJECT_NAME="${PUSH_TLS_SMOKE_PROJECT_NAME:-mf-push-tls-smoke}"
RUN_DOTNET_TLS_SMOKE="${RUN_DOTNET_TLS_SMOKE:-1}"

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required" >&2
  exit 1
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required" >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose v2 is required" >&2
  exit 1
fi
if ! command -v grpcurl >/dev/null 2>&1; then
  echo "grpcurl is required" >&2
  exit 1
fi

CERT_DIR="$(mktemp -d "${TMPDIR:-/tmp}/mf-push-tls-smoke.XXXXXX")"
export PUSH_TLS_SMOKE_CERT_DIR="$(cd "$CERT_DIR" && pwd)"

cleanup() {
  if [[ -f "$COMPOSE_FILE" ]]; then
    (cd "$PUSH_DIR" && docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down -v 2>/dev/null) || true
  fi
  rm -rf "$CERT_DIR" 2>/dev/null || true
}
trap cleanup EXIT

echo "== Generating demo CA + server + client PEMs in $PUSH_TLS_SMOKE_CERT_DIR"
openssl genrsa -out "$CERT_DIR/ca.key" 2048
openssl req -x509 -new -nodes -key "$CERT_DIR/ca.key" -sha256 -days 1 \
  -subj "/CN=Many Faces Push TLS Smoke CA" -out "$CERT_DIR/ca.crt"

openssl genrsa -out "$CERT_DIR/server.key" 2048
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" \
  -subj "/CN=localhost"
openssl x509 -req -in "$CERT_DIR/server.csr" -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial \
  -out "$CERT_DIR/server.crt" -days 1 -sha256 \
  -extfile <(printf '%s\n' 'subjectAltName=DNS:localhost,IP:127.0.0.1')

openssl genrsa -out "$CERT_DIR/client.key" 2048
openssl req -new -key "$CERT_DIR/client.key" -out "$CERT_DIR/client.csr" \
  -subj "/CN=many-faces-push-tls-smoke-client"
openssl x509 -req -in "$CERT_DIR/client.csr" -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial \
  -out "$CERT_DIR/client.crt" -days 1 -sha256

chmod 755 "$CERT_DIR"
# Server PEMs must stay world-readable: distroless runs as non-root and bind-mounts this dir.
# Client private key must NOT be world-readable: OpenSSL 3 (Ubuntu 24.04 CI) refuses mTLS client keys that are too permissive.
chmod a+r "$CERT_DIR"/*.crt "$CERT_DIR/server.key" 2>/dev/null || true
chmod 600 "$CERT_DIR/client.key"

echo "== Starting push-worker (TLS + mTLS)"
(cd "$PUSH_DIR" && docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d --build)

# Give the container a moment to bind before the first probe (cold CI runners).
sleep 2

echo "== Waiting for grpc.health Check (grpcurl, TLS + mTLS)"
GRPC_OK=0
for _ in $(seq 1 60); do
  if grpcurl -servername localhost -cacert "$CERT_DIR/ca.crt" -cert "$CERT_DIR/client.crt" -key "$CERT_DIR/client.key" \
    -d '{}' 127.0.0.1:59215 grpc.health.v1.Health/Check >/dev/null 2>&1; then
    echo "   grpcurl Health/Check succeeded"
    GRPC_OK=1
    break
  fi
  sleep 2
done
if [[ "$GRPC_OK" != "1" ]]; then
  echo "grpcurl Health/Check did not succeed in time" >&2
  (cd "$PUSH_DIR" && docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs --tail 120 push-worker) >&2 || true
  grpcurl -servername localhost -cacert "$CERT_DIR/ca.crt" -cert "$CERT_DIR/client.crt" -key "$CERT_DIR/client.key" \
    -d '{}' 127.0.0.1:59215 grpc.health.v1.Health/Check >&2 || true
  exit 1
fi

if [[ "$RUN_DOTNET_TLS_SMOKE" == "1" ]]; then
  if ! command -v dotnet >/dev/null 2>&1; then
    echo "dotnet not on PATH; set RUN_DOTNET_TLS_SMOKE=0 to skip .NET smoke" >&2
    exit 1
  fi
  ROOT="$(cd "$PUSH_DIR/.." && pwd)"
  echo "== .NET GrpcChannel + grpc.health (monorepo: $ROOT)"
  export PUSH_TLS_SMOKE=1
  export PUSH_TLS_SMOKE_GRPC_PORT=59215
  export PUSH_TLS_SMOKE_CA="$CERT_DIR/ca.crt"
  export PUSH_TLS_SMOKE_CLIENT_CERT="$CERT_DIR/client.crt"
  export PUSH_TLS_SMOKE_CLIENT_KEY="$CERT_DIR/client.key"
  (cd "$ROOT/many_faces_backend" && dotnet test BeDemo.Api.Tests/BeDemo.Api.Tests.csproj -c Release \
    --filter "FullyQualifiedName~PushWorkerTlsEndToEndSmokeTests" -v minimal --nologo)
fi

echo "== Push TLS smoke completed successfully"
