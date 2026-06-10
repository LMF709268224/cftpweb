#!/bin/bash
set -e

# 构建镜像 (多阶段构建: Node编译前端 -> Go编译后端 -> 组装成极简镜像)
cd ..
buildah build -f adminserver/Dockerfile -t localhost/adminserver:v1 .
cd adminserver

# 导出并导入到 k3s 内部镜像池
echo "Importing into k3s..."
buildah push --format docker localhost/adminserver:v1 docker-archive:adminserver.tar
sudo k3s ctr images import adminserver.tar

echo "Verifying image in k3s:"
sudo k3s ctr images ls | grep adminserver

# 清理临时文件
rm adminserver.tar
echo "Done!"
