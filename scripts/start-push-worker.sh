#!/usr/bin/env bash
# Starts the push-worker container defined in many_faces_push/docker-compose.yml (standalone bridge network).
# The monorepo start-all-dev.sh connects push-worker-dev to many_faces_main_dev-network when ENABLE_PUSH_WORKER=1.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
docker compose up -d --build
