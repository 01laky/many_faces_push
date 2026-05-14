# many_faces_push

**Canonical GitHub repository:** [github.com/01laky/many_faces_push](https://github.com/01laky/many_faces_push) — default branch **`main`**.  
In the **many_faces_main** monorepo this tree is linked as the **`many_faces_push/`** git submodule (see [monorepo submodule guide](https://github.com/01laky/many_faces_main/blob/main/docs/guides/git-submodules.md)).

## Status (v1 slice)

Shipping today:

- **gRPC `PushService`** with **`SendPush`** (FCM multicast via Firebase Admin **HTTP v1** under the hood).
- **`grpc.health.v1`** for probes.
- Optional **shared-secret** metadata auth (`x-push-worker-token` ↔ `PUSH_WORKER_EXPECTED_TOKEN`).
- Optional **TLS** and **mTLS** on the gRPC listener (`PUSH_WORKER_GRPC_TLS_*` — see monorepo [`docs/guides/push-grpc-tls-mtls.md`](https://github.com/01laky/many_faces_main/blob/main/docs/guides/push-grpc-tls-mtls.md)).
- Optional **gRPC reflection** for `grpcurl` (`PUSH_WORKER_GRPC_REFLECTION`, default **on** in `docker-compose.yml` — turn **off** in production images).
- **Dockerfile** → `gcr.io/distroless/static-debian12:nonroot`.

**Monorepo local-dev guide:** [`docs/guides/push-notifications-local-dev.md`](https://github.com/01laky/many_faces_main/blob/main/docs/guides/push-notifications-local-dev.md)  
**Full product / security spec:** [`docs/prompts/push-notifications-fcm-go-grpc-firebase-worker-agent-prompt.md`](https://github.com/01laky/many_faces_main/blob/main/docs/prompts/push-notifications-fcm-go-grpc-firebase-worker-agent-prompt.md)

## Token path (locked for v1)

Many Faces uses **direct FCM registration tokens** on the mobile client (Expo/EAS per current docs) — **path A** in the agent prompt §1.1. **`many_faces_backend`** persists tokens; this worker **never** reads PostgreSQL.

## Intended role (summary)

- **Go gRPC worker** isolates **Firebase service account credentials** and **FCM dispatch** from **`many_faces_backend`**.
- **`many_faces_backend`** remains the system of record for users, devices, and authorization; it calls this worker **only via gRPC**.
- **Browsers and mobile apps** must **never** call this worker directly.

## Repository layout

| Path | Purpose |
| ---- | ------- |
| `README.md` | This file. |
| `docker-compose.yml` | Local **`push-worker`** service (base compose). |
| `docker-compose.tls-smoke.yml` | CI/local **TLS + mTLS** smoke (host port **59215**); see `scripts/smoke-grpc-tls.sh`. |
| `docker-compose.credentials.yml` | Optional merge file — bind-mounts **`FIREBASE_SA_HOST_PATH`** → `/run/secrets/firebase-sa.json`; used by `scripts/start-push-worker.sh` when a file path is resolved. |
| `Dockerfile` | Multi-stage **Go 1.25** build → distroless `nonroot`. |
| `proto/README.md` | Pointer to canonical `.proto` in **`many_faces_proto`** (monorepo submodule). |
| `gen/` | Generated **Go** stubs (`protoc` — see below; inputs live under **`many_faces_proto/proto`**). |
| `cmd/push-worker/` | Entrypoint: config, Firebase init, gRPC server, graceful shutdown. |
| `internal/config` | Environment parsing + validation. |
| `internal/grpccreds` | gRPC server TLS / mTLS credential loading (mirrors `many_faces_elastic/internal/grpccreds`). |
| `internal/server` | gRPC service + auth interceptor. |
| `internal/msgutil` | Pure FCM payload mapping + tests. |
| `.env.example` | Documented env vars (no secrets). |

## Ports (reserved — do not collide)

| Component | Internal gRPC | Typical host map |
| --------- | ------------- | ---------------- |
| `many_faces_ai` | `50051` | (see AI repo) |
| `many_faces_elastic` search-worker | `50052` | `59202` |
| **`many_faces_push` push-worker** | **`50053`** | **`59203`** |

Backend example: `Push__WorkerGrpcUrl=http://push-worker-dev:50053` on `many_faces_main_dev-network`.

## Regenerating Go stubs (from `many_faces_proto`)

When the contract under **`many_faces_proto/proto/manyfaces/push/v1/push.proto`** changes, regenerate Go into **`gen/`** from this repo root inside **`many_faces_main`** (Docker example when `protoc` is not on the host):

```bash
# Run from many_faces_push/ with many_faces_proto as a sibling submodule (monorepo default).
docker run --rm \
  -v "$(pwd)":/w \
  -v "$(pwd)/../many_faces_proto":/mfproto:ro \
  -w /w golang:1.25-bookworm bash -c '
  apt-get update -qq && apt-get install -y -qq protobuf-compiler >/dev/null
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
  export PATH="$PATH:$(go env GOPATH)/bin"
  mkdir -p gen
  protoc -I /mfproto/proto \
    --go_out=gen --go_opt=paths=source_relative \
    --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
    manyfaces/push/v1/push.proto
'
```

**Standalone clone** of `many_faces_push`: clone **`many_faces_proto`** beside this repo (or adjust the `-v` mount) so `/mfproto/proto` resolves.

## gRPC ↔ FCM error mapping (operator)

| Worker / FCM signal | `PerTokenResult.outcome_code` | `permanent_invalid` | Backend action (recommended) |
| ------------------- | ------------------------------ | ------------------- | ------------------------------ |
| Success | `OK` | false | None |
| Unregistered token | `UNREGISTERED` | **true** | Delete SQL row for that registration token |
| Sender ID mismatch | `SENDER_ID_MISMATCH` | true | Delete / re-register device |
| Invalid argument | `INVALID_ARGUMENT` | varies | Fix client payload; often delete bad token |
| Quota / unavailable | `QUOTA_EXCEEDED` / `UNAVAILABLE` | false | Retry with backoff at backend |
| Unknown | `UNKNOWN` | false | Log + investigate |

Full tokens are **never** logged; responses include **`token_sha256_prefix`** (first 8 hex chars of SHA-256) for correlation only.

## Firebase / iOS client plist (local only)

Download **`GoogleService-Info.plist`** from the Firebase Console (iOS app) and place it at the **repository root** next to `go.mod`. The file is **gitignored** — never commit it (it includes an API key and app identifiers).

For Android, the Firebase **`google-services.json`** (or a copy named `google-service.json`) belongs in the **Android / React Native** app module, not in the Go worker; if you keep a copy next to `go.mod` for local reference, it is **gitignored** as well — do not commit it.

Use the plist (or JSON) as the source of truth while wiring env vars (see **`.env.example`**). **Server-side FCM** uses **`GOOGLE_APPLICATION_CREDENTIALS`** (service account JSON path inside the container). The plist **`API_KEY`** is for the **iOS client**, not Admin.

## Clone (standalone)

```bash
git clone https://github.com/01laky/many_faces_push.git
cd many_faces_push
```

Use **HTTPS** or **SSH** remote interchangeably; match the URL style your org uses in `.gitmodules`.

## License / product

Repository policy for a license file will follow the same approach as sibling **`many_faces_*`** infra repos.

## Out of scope for v1 (see monorepo prompt)

No public REST on the worker, no in-app notification center taxonomy in v1, no SMS/email/web push in this track.
