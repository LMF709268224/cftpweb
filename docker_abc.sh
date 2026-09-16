#!/usr/bin/env bash
set -euo pipefail

# Docker Compose equivalent of the legacy per-service abc.sh scripts.
# The legacy scripts remain unchanged and continue to deploy to K3s.
# Usage:
#   ./docker_abc.sh
#   ./docker_abc.sh candbff candweb
#   ./docker_abc.sh --env prod
#   ./docker_abc.sh --env prod candbff candweb

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
COMPOSE_DIR="${ROOT_DIR}/deployment/docker-compose"
DEPLOY_ENV="test"

if [ "${1:-}" = "--env" ]; then
  if [ "$#" -lt 2 ]; then
    echo "ERROR: --env requires an environment name. Supported value: prod"
    exit 1
  fi

  DEPLOY_ENV="$2"
  shift 2
fi

case "$DEPLOY_ENV" in
  test)
    ENV_FILE="${COMPOSE_DIR}/.env"
    ;;
  prod)
    ENV_FILE="${COMPOSE_DIR}/.env.prod"
    ;;
  *)
    echo "ERROR: Unsupported environment: ${DEPLOY_ENV}. Supported value: prod"
    exit 1
    ;;
esac

if [ ! -f "$ENV_FILE" ]; then
  echo "ERROR: Missing ${ENV_FILE}"
  echo "Create it from .env.example and configure the ${DEPLOY_ENV} environment first."
  exit 1
fi

IMAGE_TAG=$(awk -F= '$1 == "IMAGE_TAG" { print substr($0, index($0, "=") + 1) }' "$ENV_FILE" | tail -n 1 | tr -d '\r')
IMAGE_TAG="${IMAGE_TAG:-dev-latest}"

echo ">>> Deploying ${DEPLOY_ENV} environment using ${ENV_FILE}..."
echo ">>> Updating source code..."
git -C "$ROOT_DIR" pull --ff-only

echo ">>> Building Docker Compose images with tag ${IMAGE_TAG}..."
if [ "$#" -gt 0 ]; then
  bash "${ROOT_DIR}/build_compose_images.sh" "$IMAGE_TAG" "$@"
else
  bash "${ROOT_DIR}/build_compose_images.sh" "$IMAGE_TAG"
fi

if docker info >/dev/null 2>&1; then
  DOCKER_CMD=(docker)
elif sudo docker info >/dev/null 2>&1; then
  DOCKER_CMD=(sudo docker)
else
  echo "ERROR: Docker daemon is not available."
  exit 1
fi

COMPOSE_CMD=(
  "${DOCKER_CMD[@]}"
  compose
  --project-directory "$COMPOSE_DIR"
  --env-file "$ENV_FILE"
)

# docker-compose.yml uses this value for the containers' env_file as well as
# Compose interpolation, so both always come from the selected environment.
export PORTAL_ENV_FILE="$ENV_FILE"

echo ">>> Validating Docker Compose configuration..."
"${COMPOSE_CMD[@]}" config --quiet

echo ">>> Starting Docker Compose services..."
if [ "$#" -gt 0 ]; then
  "${COMPOSE_CMD[@]}" up -d --no-deps "$@"
else
  "${COMPOSE_CMD[@]}" up -d
fi

"${COMPOSE_CMD[@]}" ps
