#!/bin/bash
set -e

# 1. 编译前端 Vue 项目
echo "Building frontend vue-web..."
cd vue-web
npm ci
npm run build
cd ..

# 2. 清理旧的二进制文件并构建新文件
echo "Cleaning old binary and building new candweb..."
rm -f ./candweb
# CGO_ENABLED=0 确保静态编译
# -ldflags="-s -w" 可以大幅缩小二进制体积（去掉调试信息）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o candweb .

# 3. 构建容器镜像
echo "Building Docker image..."
buildah build -t candweb:v1 .

# 4. 导出并导入到 k3s 内部镜像池
echo "Importing into k3s..."
buildah push --format docker candweb:v1 docker-archive:candweb.tar
sudo k3s ctr images import candweb.tar

# 5. 验证导入结果
echo "Verifying image in k3s:"
sudo k3s ctr images ls | grep candweb

# 清理临时文件
rm candweb.tar
echo "Done!"
