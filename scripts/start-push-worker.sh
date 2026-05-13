#!/usr/bin/env bash
# Starts the push-worker container(s) for many_faces_push.
# Uses docker-compose.credentials.yml when FIREBASE_SA_HOST_PATH is set to an existing file, or when
# ./firebase-sa.json exists next to this script's repo root (many_faces_push/firebase-sa.json).
# The monorepo start-all-dev.sh exports FIREBASE_SA_HOST_PATH when ENABLE_PUSH_WORKER=1 and that file exists.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE_FILES=(-f docker-compose.yml)

# Resolve service account path: explicit env wins, then conventional local filename in the submodule root.
if [[ -z "${FIREBASE_SA_HOST_PATH:-}" && -f "$ROOT/firebase-sa.json" ]]; then
	export FIREBASE_SA_HOST_PATH="$ROOT/firebase-sa.json"
fi

if [[ -n "${FIREBASE_SA_HOST_PATH:-}" ]]; then
	if [[ ! -f "$FIREBASE_SA_HOST_PATH" ]]; then
		echo "start-push-worker: FIREBASE_SA_HOST_PATH is set but file not found: $FIREBASE_SA_HOST_PATH" >&2
		exit 1
	fi
	COMPOSE_FILES+=(-f docker-compose.credentials.yml)
fi

docker compose "${COMPOSE_FILES[@]}" up -d --build
