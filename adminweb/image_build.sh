#!/bin/bash
set -euo pipefail

SERVICE_NAME="adminweb"
IMAGE_TAG="localhost/${SERVICE_NAME}:v1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${SCRIPT_DIR}"

echo "Building admin frontend..."
cd web
pnpm install
pnpm build
cd ..

echo "Cleaning old binary and building ${SERVICE_NAME}..."
rm -f "./${SERVICE_NAME}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${SERVICE_NAME}" .

echo "Building Docker image ${IMAGE_TAG}..."
buildah build -t "${IMAGE_TAG}" .

echo "Importing ${IMAGE_TAG} into k3s..."
buildah push --format docker "${IMAGE_TAG}" "docker-archive:${SERVICE_NAME}.tar:${IMAGE_TAG}"
sudo k3s ctr images import "${SERVICE_NAME}.tar"

echo "Verifying image in k3s:"
sudo k3s ctr images ls | grep "${SERVICE_NAME}"

rm -f "${SERVICE_NAME}.tar"
echo "Done!"
