#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
LOG_DIR="$ROOT_DIR/logs"
RUN_DIR="$ROOT_DIR/run"
ENV_FILE="${SERVER_DEPLOY_ENV_FILE:-$ROOT_DIR/scripts/server-deploy.env}"

MASTER_BIN="$ROOT_DIR/jmeter-admin"
AGENT_BIN="$ROOT_DIR/jmeter-agent"

MASTER_PID_FILE="$RUN_DIR/master.pid"
AGENT_PID_FILE="$RUN_DIR/agent.pid"

MASTER_LOG_FILE="$LOG_DIR/master.log"
AGENT_LOG_FILE="$LOG_DIR/agent.log"

DEFAULT_MASTER_HOST="${MASTER_HOST:-0.0.0.0}"
DEFAULT_MASTER_PORT="${MASTER_PORT:-8080}"
DEFAULT_AGENT_PORT="${AGENT_PORT:-8089}"
DEFAULT_AGENT_DATA_DIR="${AGENT_DATA_DIR:-/opt/jmeter/csv-data}"
DEFAULT_AGENT_TOKEN="${AGENT_TOKEN:-}"
DEFAULT_AGENT_JMETER_PATH="${AGENT_JMETER_PATH:-}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
  echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
  echo -e "${BLUE}[STEP]${NC} $1"
}

usage() {
  cat <<EOF
服务器后台部署脚本

用法:
  ./scripts/server-deploy.sh <master|agent|all> <install-deps|build|start|stop|restart|status|deploy|logs>

示例:
  ./scripts/server-deploy.sh master deploy
  ./scripts/server-deploy.sh agent start
  ./scripts/server-deploy.sh all restart
  ./scripts/server-deploy.sh all status
  ./scripts/server-deploy.sh master logs
  ./scripts/server-deploy.sh all install-deps

配置方式:
  1. 复制 scripts/server-deploy.env.example 为 scripts/server-deploy.env
  2. 按服务器实际情况修改端口、token、jmeter 路径

说明:
  - master 会读取项目根目录下的 config.yaml
  - agent 启动参数来自 scripts/server-deploy.env
  - deploy = build + restart
  - install-deps 会复用项目根目录 deploy.sh 的依赖安装逻辑
EOF
}

load_env() {
  mkdir -p "$LOG_DIR" "$RUN_DIR"
  if [[ -f "$ENV_FILE" ]]; then
    # shellcheck disable=SC1090
    source "$ENV_FILE"
  fi

  MASTER_HOST="${MASTER_HOST:-$DEFAULT_MASTER_HOST}"
  MASTER_PORT="${MASTER_PORT:-$DEFAULT_MASTER_PORT}"
  AGENT_PORT="${AGENT_PORT:-$DEFAULT_AGENT_PORT}"
  AGENT_DATA_DIR="${AGENT_DATA_DIR:-$DEFAULT_AGENT_DATA_DIR}"
  AGENT_TOKEN="${AGENT_TOKEN:-$DEFAULT_AGENT_TOKEN}"
  AGENT_JMETER_PATH="${AGENT_JMETER_PATH:-$DEFAULT_AGENT_JMETER_PATH}"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log_error "未找到命令: $1"
    exit 1
  fi
}

install_deps() {
  if [[ ! -x "$ROOT_DIR/deploy.sh" ]]; then
    log_error "未找到 $ROOT_DIR/deploy.sh，无法自动安装依赖"
    exit 1
  fi

  log_step "开始自动安装依赖（Go / Node.js / Java / JMeter）"
  (
    cd "$ROOT_DIR"
    bash ./deploy.sh install-deps
  )
}

ensure_build_dependencies() {
  local need_install=0
  if ! command -v go >/dev/null 2>&1; then
    need_install=1
  fi

  if [[ ! -f "$ROOT_DIR/web/dist/index.html" ]]; then
    if ! command -v node >/dev/null 2>&1 || ! command -v npm >/dev/null 2>&1; then
      need_install=1
    fi
  fi

  if [[ $need_install -eq 1 ]]; then
    log_warn "检测到构建依赖缺失，准备自动安装依赖"
    install_deps
  fi
}

ensure_frontend_assets() {
  if [[ -f "$ROOT_DIR/web/dist/index.html" ]]; then
    return
  fi

  ensure_build_dependencies
  require_cmd node
  require_cmd npm

  log_step "未检测到 web/dist，开始构建前端资源"
  (
    cd "$ROOT_DIR/web"
    npm install
    npm run build
  )
}

build_master() {
  ensure_build_dependencies
  require_cmd go
  ensure_frontend_assets

  log_step "构建 master 二进制"
  (
    cd "$ROOT_DIR"
    CGO_ENABLED=1 go build -o "$MASTER_BIN" .
  )
  log_info "master 构建完成: $MASTER_BIN"
}

build_agent() {
  ensure_build_dependencies
  require_cmd go

  log_step "构建 agent 二进制"
  (
    cd "$ROOT_DIR"
    CGO_ENABLED=1 go build -o "$AGENT_BIN" ./cmd/agent
  )
  log_info "agent 构建完成: $AGENT_BIN"
}

build_role() {
  case "$1" in
    master)
      build_master
      ;;
    agent)
      build_agent
      ;;
    all)
      build_master
      build_agent
      ;;
    *)
      log_error "未知角色: $1"
      usage
      exit 1
      ;;
  esac
}

is_running() {
  local pid_file=$1
  if [[ -f "$pid_file" ]]; then
    local pid
    pid=$(cat "$pid_file")
    if [[ -n "$pid" ]] && ps -p "$pid" >/dev/null 2>&1; then
      return 0
    fi
  fi
  return 1
}

start_master() {
  if is_running "$MASTER_PID_FILE"; then
    log_warn "master 已在运行 (PID: $(cat "$MASTER_PID_FILE"))"
    return
  fi

  if [[ ! -x "$MASTER_BIN" ]]; then
    log_warn "未找到 master 二进制，先执行构建"
    build_master
  fi

  log_step "启动 master"
  (
    cd "$ROOT_DIR"
    nohup "$MASTER_BIN" >"$MASTER_LOG_FILE" 2>&1 &
    echo $! >"$MASTER_PID_FILE"
  )
  log_info "master 已启动 (PID: $(cat "$MASTER_PID_FILE"))"
  log_info "访问地址: http://${MASTER_HOST}:${MASTER_PORT}"
  log_info "日志文件: $MASTER_LOG_FILE"
}

start_agent() {
  if is_running "$AGENT_PID_FILE"; then
    log_warn "agent 已在运行 (PID: $(cat "$AGENT_PID_FILE"))"
    return
  fi

  if [[ ! -x "$AGENT_BIN" ]]; then
    log_warn "未找到 agent 二进制，先执行构建"
    build_agent
  fi

  mkdir -p "$AGENT_DATA_DIR"

  local cmd=("$AGENT_BIN" "-port" "$AGENT_PORT" "-data-dir" "$AGENT_DATA_DIR")
  if [[ -n "$AGENT_TOKEN" ]]; then
    cmd+=("-token" "$AGENT_TOKEN")
  fi
  if [[ -n "$AGENT_JMETER_PATH" ]]; then
    cmd+=("-jmeter-path" "$AGENT_JMETER_PATH")
  fi

  log_step "启动 agent"
  (
    cd "$ROOT_DIR"
    nohup "${cmd[@]}" >"$AGENT_LOG_FILE" 2>&1 &
    echo $! >"$AGENT_PID_FILE"
  )
  log_info "agent 已启动 (PID: $(cat "$AGENT_PID_FILE"))"
  log_info "监听地址: http://0.0.0.0:${AGENT_PORT}"
  log_info "日志文件: $AGENT_LOG_FILE"
}

start_role() {
  case "$1" in
    master)
      start_master
      ;;
    agent)
      start_agent
      ;;
    all)
      start_master
      start_agent
      ;;
    *)
      log_error "未知角色: $1"
      usage
      exit 1
      ;;
  esac
}

stop_process() {
  local name=$1
  local pid_file=$2

  if [[ ! -f "$pid_file" ]]; then
    log_warn "$name 未运行"
    return
  fi

  local pid
  pid=$(cat "$pid_file")

  if [[ -z "$pid" ]] || ! ps -p "$pid" >/dev/null 2>&1; then
    log_warn "$name 的 PID 文件存在但进程已不在，清理旧 PID"
    rm -f "$pid_file"
    return
  fi

  log_step "停止 $name (PID: $pid)"
  kill "$pid" >/dev/null 2>&1 || true
  for _ in {1..15}; do
    if ! ps -p "$pid" >/dev/null 2>&1; then
      rm -f "$pid_file"
      log_info "$name 已停止"
      return
    fi
    sleep 1
  done

  log_warn "$name 未正常退出，执行强制终止"
  kill -9 "$pid" >/dev/null 2>&1 || true
  rm -f "$pid_file"
  log_info "$name 已强制停止"
}

stop_role() {
  case "$1" in
    master)
      stop_process "master" "$MASTER_PID_FILE"
      ;;
    agent)
      stop_process "agent" "$AGENT_PID_FILE"
      ;;
    all)
      stop_process "agent" "$AGENT_PID_FILE"
      stop_process "master" "$MASTER_PID_FILE"
      ;;
    *)
      log_error "未知角色: $1"
      usage
      exit 1
      ;;
  esac
}

status_one() {
  local name=$1
  local pid_file=$2
  local log_file=$3

  if is_running "$pid_file"; then
    local pid
    pid=$(cat "$pid_file")
    log_info "$name 运行中 (PID: $pid)"
    ps -p "$pid" -o pid,ppid,%cpu,%mem,etime,command
    echo "日志: $log_file"
  else
    log_warn "$name 未运行"
  fi
}

status_role() {
  case "$1" in
    master)
      status_one "master" "$MASTER_PID_FILE" "$MASTER_LOG_FILE"
      ;;
    agent)
      status_one "agent" "$AGENT_PID_FILE" "$AGENT_LOG_FILE"
      ;;
    all)
      status_one "master" "$MASTER_PID_FILE" "$MASTER_LOG_FILE"
      echo ""
      status_one "agent" "$AGENT_PID_FILE" "$AGENT_LOG_FILE"
      ;;
    *)
      log_error "未知角色: $1"
      usage
      exit 1
      ;;
  esac
}

logs_role() {
  case "$1" in
    master)
      tail -n 100 -f "$MASTER_LOG_FILE"
      ;;
    agent)
      tail -n 100 -f "$AGENT_LOG_FILE"
      ;;
    all)
      log_info "master 日志: $MASTER_LOG_FILE"
      [[ -f "$MASTER_LOG_FILE" ]] && tail -n 80 "$MASTER_LOG_FILE" || true
      echo ""
      log_info "agent 日志: $AGENT_LOG_FILE"
      [[ -f "$AGENT_LOG_FILE" ]] && tail -n 80 "$AGENT_LOG_FILE" || true
      ;;
    *)
      log_error "未知角色: $1"
      usage
      exit 1
      ;;
  esac
}

deploy_role() {
  build_role "$1"
  stop_role "$1"
  start_role "$1"
}

main() {
  if [[ $# -lt 2 ]]; then
    usage
    exit 1
  fi

  local role=$1
  local action=$2

  load_env

  case "$action" in
    install-deps)
      install_deps
      ;;
    build)
      build_role "$role"
      ;;
    start)
      start_role "$role"
      ;;
    stop)
      stop_role "$role"
      ;;
    restart)
      stop_role "$role"
      start_role "$role"
      ;;
    status)
      status_role "$role"
      ;;
    deploy)
      deploy_role "$role"
      ;;
    logs)
      logs_role "$role"
      ;;
    *)
      log_error "未知动作: $action"
      usage
      exit 1
      ;;
  esac
}

main "$@"
