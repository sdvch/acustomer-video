#!/usr/bin/env bash
# 幂等重建 AI 工具的 skills 软链。
# 物理源：.agents/skills/（唯一）；.cursor/skills、.qoder/skills、.github/skills 为相对软链引用。
# fresh clone 后运行一次：./scripts/sync-agent-links.sh
# 校验模式（不修改任何东西，CI 可用）：./scripts/sync-agent-links.sh --check
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

CHECK=0
[[ "${1:-}" == "--check" ]] && CHECK=1

# 链接清单：<软链路径>|<相对 target>
LINKS=(
  ".cursor/skills|../.agents/skills"
  ".qoder/skills|../.agents/skills"
  ".github/skills|../.agents/skills"
)

fail=0
for entry in "${LINKS[@]}"; do
  link="${entry%%|*}"
  target="${entry##*|}"

  # 悬空软链：是软链但 target 不存在
  if [[ -L "$link" && ! -e "$link" ]]; then
    echo "DANGLING $link -> $(readlink "$link")（target 不存在）" >&2
    fail=1
    continue
  fi

  # 已是正确软链
  if [[ -L "$link" && "$(readlink "$link")" == "$target" ]]; then
    echo "OK      $link -> $target"
    continue
  fi

  # 已存在真实目录/文件（非软链）：不覆盖，人工确认
  if [[ -e "$link" && ! -L "$link" ]]; then
    echo "SKIP    $link 已存在且不是软链（真实目录/文件），不覆盖，请人工确认。" >&2
    fail=1
    continue
  fi

  if [[ $CHECK -eq 1 ]]; then
    echo "MISS    $link（--check 模式不修复）" >&2
    fail=1
    continue
  fi

  # target 错误的旧软链：删除重建
  [[ -L "$link" ]] && rm "$link"
  mkdir -p "$(dirname "$link")"
  ln -s "$target" "$link"
  echo "FIXED   $link -> $target"
done

exit $fail
