#!/usr/bin/env bash
# 服务器端部署脚本：由 GitHub Actions (deploy.yml) 通过 SSH 调用，也可手动执行
#   ssh admin@8.152.208.234 'cd ~/acustomer-video && git pull && ./scripts/deploy.sh'
# 仅更新后端容器；MySQL 数据卷与 .env 不受影响。
set -euo pipefail

cd "$(dirname "$0")/../deploy"

echo "==> 构建并重启后端容器"
docker compose up -d --build

echo "==> 等待服务健康"
st="none"
for i in $(seq 1 30); do
  st=$(docker inspect -f '{{.State.Health.Status}}' acustomer-video-server 2>/dev/null || echo none)
  [ "$st" = "healthy" ] && break
  sleep 4
done
if [ "$st" != "healthy" ]; then
  echo "!! 服务未在预期时间内变 healthy，最近日志：" >&2
  docker logs --tail 30 acustomer-video-server >&2 || true
  exit 1
fi

echo "==> 部署完成：$(curl -s -m 5 http://127.0.0.1:18080/api/v1/health)"
