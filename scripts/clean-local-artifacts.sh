#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

paths=(
  ".gocache"
  "temp"
  "data"
  "uploads"
  "results"
  "jmeter.log"
  "jmeter-admin"
  "web/dist"
  "web/node_modules"
  "web/jmeter.log"
  ".DS_Store"
  "web/.DS_Store"
  "internal/.DS_Store"
  "uploads/.DS_Store"
  "results/.DS_Store"
  ".qoder/.DS_Store"
)

echo "准备清理本地依赖、缓存、运行结果和数据库文件..."
echo

for relative_path in "${paths[@]}"; do
  target_path="${ROOT_DIR}/${relative_path}"
  if [ -e "${target_path}" ]; then
    echo "删除: ${relative_path}"
    rm -rf "${target_path}"
  fi
done

find "${ROOT_DIR}" -type f \( -name "*.log" -o -name "*.jtl" -o -name "*.csv" -o -name "*.ndjson" -o -name "*.db" -o -name "*.sqlite" -o -name ".DS_Store" \) -print -delete

echo
echo "清理完成。"
echo "建议下一步执行:"
echo "  git status"
echo "确认只剩需要提交的源码和配置文件。"
