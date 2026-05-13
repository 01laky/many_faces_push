#!/usr/bin/env bash
# Stops many_faces_push docker compose services (push-worker).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
docker compose down
