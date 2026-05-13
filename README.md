# many_faces_push

**Canonical GitHub repository:** [github.com/01laky/many_faces_push](https://github.com/01laky/many_faces_push) — default branch **`main`**.  
In the **many_faces_main** monorepo this tree is linked as the **`many_faces_push/`** git submodule (see [monorepo submodule guide](https://github.com/01laky/many_faces_main/blob/main/docs/guides/git-submodules.md)).

## Status (skeleton)

This repository is a **placeholder shell** only: **no gRPC server, no Firebase Admin, and no FCM sending** yet. The next implementation steps follow the agent prompt in the monorepo:

**[`docs/prompts/push-notifications-fcm-go-grpc-firebase-worker-agent-prompt.md`](https://github.com/01laky/many_faces_main/blob/main/docs/prompts/push-notifications-fcm-go-grpc-firebase-worker-agent-prompt.md)**

## Intended role (summary)

- **Go gRPC worker** colocated with optional Docker tooling: isolate **Firebase service account credentials** and **FCM dispatch** from **`many_faces_backend`**.
- **`many_faces_backend`** remains the system of record for users, devices, and authorization; it will call this worker **only via gRPC** once implemented.
- **Browsers and mobile apps** must **never** call this worker directly.

## Planned layout (target)

| Path | Purpose |
| ---- | ------- |
| `README.md` | This file. |
| `docker-compose.yml` | Local **`push-worker`** service (to be wired like other infra submodules). |
| `Dockerfile` | Multi-stage Go build → minimal runtime image. |
| `proto/` | Canonical **`.proto`** for `PushService` (v1) — empty until the first proto PR. |
| `cmd/push-worker/` | Process entrypoint: config, gRPC server, graceful shutdown. |
| `internal/` | Service implementation, FCM adapter, retries — **no** face/domain ACL here. |

## Ports (reserved — do not collide)

Align internal gRPC with monorepo conventions (see prompt **§2.2**):

- **`many_faces_ai`** — commonly `50051` in examples.
- **`many_faces_elastic`** search-worker — commonly `50052`.

**Default internal gRPC port for this worker:** **`50053`** (host debug mapping to be documented when compose is finalized).

## Clone (standalone)

```bash
git clone https://github.com/01laky/many_faces_push.git
cd many_faces_push
```

Use **HTTPS** or **SSH** remote interchangeably; match the URL style your org uses in `.gitmodules`.

## Firebase / iOS client plist (local only)

Download **`GoogleService-Info.plist`** from the Firebase Console (iOS app) and place it at the **repository root** next to `go.mod`. The file is **gitignored** — never commit it (it includes an API key and app identifiers).

For Android, the Firebase **`google-services.json`** (or a copy named `google-service.json`) belongs in the **Android / React Native** app module, not in the Go worker; if you keep a copy next to `go.mod` for local reference, it is **gitignored** as well — do not commit it.

Use the plist (or JSON) as the source of truth while wiring env vars (see **`.env.example`**): `PROJECT_ID`, `BUNDLE_ID`, `GCM_SENDER_ID`, `GOOGLE_APP_ID`, `STORAGE_BUCKET`. The **Go push worker** will use **Firebase Admin + a service account JSON** for FCM (`GOOGLE_APPLICATION_CREDENTIALS`); the plist’s **`API_KEY`** is for the **iOS client**, not for server-side Admin.

## License / product

Repository policy for a license file will follow the same approach as sibling **`many_faces_*`** infra repos once code is added.

## Out of scope for first code drop

See the monorepo prompt **non-goals** (no public REST on the worker, no in-app notification center taxonomy in v1, no SMS/email/web push in this track).
