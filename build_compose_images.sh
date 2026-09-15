#!/usr/bin/env bash
set -euo pipefail

# Build images for the Docker Compose deployment without changing the legacy
# abc.sh -> image_build.sh -> Buildah/K3s workflow.
# Usage:
#   ./build_compose_images.sh
#   ./build_compose_images.sh dev-latest
#   ./build_compose_images.sh dev-latest candbff candweb

IMAGE_TAG="${1:-dev-latest}"
if [ "$#" -gt 0 ]; then
  shift
fi

ALL_SERVICES=(
  candbff
  adminbff
  candweb
  adminweb
)

if [ "$#" -gt 0 ]; then
  SERVICES=("$@")
else
  SERVICES=("${ALL_SERVICES[@]}")
fi

if [[ ! "$IMAGE_TAG" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]*$ ]]; then
  echo "ERROR: Invalid Docker image tag: ${IMAGE_TAG}"
  exit 1
fi

for command_name in go npm docker; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "ERROR: Required command is not installed: ${command_name}"
    exit 1
  fi
done

if docker info >/dev/null 2>&1; then
  DOCKER_CMD=(docker)
elif sudo docker info >/dev/null 2>&1; then
  DOCKER_CMD=(sudo docker)
else
  echo "ERROR: Docker daemon is not available."
  exit 1
fi

if [ -z "${GOARCH:-}" ]; then
  case "$(uname -m)" in
    x86_64) GOARCH="amd64" ;;
    aarch64|arm64) GOARCH="arm64" ;;
    *) GOARCH="$(go env GOHOSTARCH 2>/dev/null || echo amd64)" ;;
  esac
fi

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

echo "=========================================================="
echo "CFTP Web Docker image build started"
echo "Tag: ${IMAGE_TAG} | Arch: ${GOARCH}"
echo "Services (${#SERVICES[@]}): ${SERVICES[*]}"
echo "=========================================================="

for service in "${SERVICES[@]}"; do
  case "$service" in
    candbff|adminbff)
      ;;
    candweb|adminweb)
      echo ">>> Building frontend assets for ${service}..."
      (cd "${ROOT_DIR}/${service}/vue-web" && npm ci && npm run build)
      ;;
    *)
      echo "ERROR: Unknown service: ${service}"
      exit 1
      ;;
  esac

  service_dir="${ROOT_DIR}/${service}"
  echo ">>> Building Linux binary for ${service}..."
  (cd "$service_dir" && CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build -ldflags="-s -w" -o "$service" .)

  echo ">>> Building Docker image localhost/${service}:${IMAGE_TAG}..."
  "${DOCKER_CMD[@]}" build \
    -t "${service}:${IMAGE_TAG}" \
    -t "localhost/${service}:${IMAGE_TAG}" \
    "$service_dir"
done

echo
echo "Built images:"
for service in "${SERVICES[@]}"; do
  "${DOCKER_CMD[@]}" image inspect --format '{{.RepoTags}}' "localhost/${service}:${IMAGE_TAG}"
done
