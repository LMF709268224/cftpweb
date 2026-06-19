#!/bin/bash
set -e

# 1. 清理旧的二进制文件并构建新文件
echo "Cleaning old binary and building new candbff..."
rm -f ./candbff
# CGO_ENABLED=0 确保静态编译
# -ldflags="-s -w" 可以大幅缩小二进制体积（去掉调试信息）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o candbff .

# 2. 构建容器镜像
echo "Building Docker image..."
buildah build -t candbff:v1 .

# 3. 导出并导入到 k3s 内部镜像池
echo "Importing into k3s..."
buildah push --format docker candbff:v1 docker-archive:candbff.tar
sudo k3s ctr images import candbff.tar

# 4. 验证导入结果
echo "Verifying image in k3s:"
sudo k3s ctr images ls | grep candbff

# 清理临时文件
rm candbff.tar
echo "Done!"
