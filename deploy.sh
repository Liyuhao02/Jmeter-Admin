#!/bin/bash
# JMeter Admin 一键部署脚本
# 用法: ./deploy.sh [install|start|stop|restart|status|install-service|install-deps]

APP_NAME="jmeter-admin"
APP_DIR=$(cd "$(dirname "$0")" && pwd)
PID_FILE="$APP_DIR/$APP_NAME.pid"
LOG_FILE="$APP_DIR/$APP_NAME.log"
SERVICE_FILE="/etc/systemd/system/$APP_NAME.service"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印信息
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查环境
check_env() {
    # 检查 Go
    if ! command -v go &> /dev/null; then
        log_error "未找到 Go，请先安装 Go 1.21+"
        exit 1
    fi
    
    # 检查 Node.js
    if ! command -v node &> /dev/null; then
        log_error "未找到 Node.js，请先安装 Node.js 18+"
        exit 1
    fi
    
    log_info "环境检查通过"
}

# 编译项目
install() {
    log_info "开始编译项目..."
    
    # 进入项目目录
    cd "$APP_DIR" || exit 1
    
    # 构建前端（如果 web/dist 已存在则跳过）
    if [ -d "web/dist" ] && [ -f "web/dist/index.html" ]; then
        log_info "检测到 web/dist 已存在，跳过前端构建"
        log_info "如需重新构建前端，请先删除 web/dist 目录"
    else
        # 需要 Node.js 环境
        if ! command -v node &> /dev/null; then
            log_error "未找到 Node.js，且 web/dist 不存在"
            log_error "方案1: 安装 Node.js 后重试"
            log_error "方案2: 在本地电脑执行 cd web && npm install && npm run build"
            log_error "       然后将 web/dist 目录上传到服务器的 $APP_DIR/web/dist"
            exit 1
        fi
        log_info "构建前端..."
        cd web && npm install && npm run build
        if [ $? -ne 0 ]; then
            log_error "前端构建失败"
            exit 1
        fi
        cd "$APP_DIR"
    fi
    
    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        log_error "未找到 Go，请先运行: ./deploy.sh install-deps"
        exit 1
    fi
    
    # 构建后端
    log_info "构建后端（嵌入前端资源）..."
    CGO_ENABLED=1 go build -o "$APP_NAME" .
    if [ $? -ne 0 ]; then
        log_error "后端构建失败"
        exit 1
    fi
    
    # 构建 Agent
    log_info "构建 jmeter-agent..."
    CGO_ENABLED=1 go build -o jmeter-agent ./cmd/agent/
    if [ $? -ne 0 ]; then
        log_error "Agent 构建失败"
        exit 1
    fi
    
    log_info "编译完成: $APP_DIR/$APP_NAME"
    log_info "编译完成: $APP_DIR/jmeter-agent"
    log_info "启动服务: ./deploy.sh start"
}

# 启动服务
start() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            log_warn "服务已经在运行 (PID: $PID)"
            return
        fi
    fi
    
    if [ ! -f "$APP_DIR/$APP_NAME" ]; then
        log_error "未找到可执行文件，请先运行: ./deploy.sh install"
        exit 1
    fi
    
    cd "$APP_DIR" || exit 1
    nohup "./$APP_NAME" > "$LOG_FILE" 2>&1 &
    echo $! > "$PID_FILE"
    log_info "服务已启动 (PID: $(cat "$PID_FILE"))"
    log_info "日志文件: $LOG_FILE"
    log_info "访问地址: http://localhost:8080"
}

# 停止服务
stop() {
    if [ ! -f "$PID_FILE" ]; then
        log_warn "服务未运行"
        return
    fi
    
    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        kill "$PID"
        # 等待进程结束
        for i in {1..10}; do
            if ! ps -p "$PID" > /dev/null 2>&1; then
                break
            fi
            sleep 1
        done
        # 强制结束
        if ps -p "$PID" > /dev/null 2>&1; then
            kill -9 "$PID" 2>/dev/null
        fi
        log_info "服务已停止"
    else
        log_warn "进程不存在"
    fi
    
    rm -f "$PID_FILE"
}

# 重启服务
restart() {
    stop
    sleep 1
    start
}

# 查看状态
status() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            log_info "服务运行中 (PID: $PID)"
            echo ""
            echo "进程信息:"
            ps -p "$PID" -o pid,ppid,cmd,%cpu,%mem
            echo ""
            echo "监听端口:"
            lsof -p "$PID" -P -n | grep LISTEN 2>/dev/null || netstat -tlnp 2>/dev/null | grep "$APP_NAME" || echo "  无法获取端口信息"
        else
            log_warn "PID 文件存在但进程不存在，可能异常退出"
            rm -f "$PID_FILE"
        fi
    else
        log_warn "服务未运行"
    fi
}

# 安装所有依赖（Go + Node.js + JMeter）
install_deps() {
    log_info "========================================"
    log_info "  JMeter Admin 依赖一键安装"
    log_info "========================================"
    echo ""

    # 检测系统架构
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)  GO_ARCH="amd64"; NODE_ARCH="x64" ;;
        aarch64) GO_ARCH="arm64"; NODE_ARCH="arm64" ;;
        *) log_error "不支持的架构: $ARCH"; exit 1 ;;
    esac
    log_info "系统架构: $ARCH ($GO_ARCH)"

    # ========== 国内镜像源配置 ==========
    # Go 镜像: 中国科学技术大学
    GO_MIRROR="https://mirrors.ustc.edu.cn/golang"
    # Node.js 镜像: 淘宝镜像 (npmmirror)
    NODE_MIRROR="https://npmmirror.com/mirrors/node"
    # JMeter 镜像: 清华镜像
    JMETER_MIRROR="https://mirrors.tuna.tsinghua.edu.cn/apache/jmeter/binaries"
    # npm 镜像
    NPM_REGISTRY="https://registry.npmmirror.com"

    log_info "使用国内镜像源加速下载"

    # 通用下载函数
    download_file() {
        local url="$1"
        local output="$2"
        log_info "下载: $url"
        if command -v curl &> /dev/null; then
            curl -fSL --connect-timeout 30 --retry 3 -o "$output" "$url"
        elif command -v wget &> /dev/null; then
            wget --timeout=30 --tries=3 -O "$output" "$url"
        else
            log_error "未找到 curl 或 wget，请先安装: sudo apt-get install -y curl"
            exit 1
        fi
        # 验证下载文件
        if [ ! -f "$output" ] || [ ! -s "$output" ]; then
            log_error "下载失败，文件不存在或为空: $output"
            return 1
        fi
        return 0
    }

    # ========== 安装 Go ==========
    GO_VERSION="1.22.2"
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin

    if command -v go &> /dev/null; then
        CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go 已安装: $CURRENT_GO"
    else
        GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
        cd /tmp
        rm -f "$GO_TAR"

        # 优先使用国内镜像，失败回退官方源
        log_info "正在通过国内镜像下载 Go $GO_VERSION ..."
        if ! download_file "${GO_MIRROR}/${GO_TAR}" "$GO_TAR"; then
            log_warn "国内镜像下载失败，尝试官方源..."
            download_file "https://go.dev/dl/${GO_TAR}" "$GO_TAR" || {
                log_error "Go 下载失败"
                log_warn "请手动下载并上传到 /tmp/$GO_TAR"
                log_warn "然后重新运行: ./deploy.sh install-deps"
                exit 1
            }
        fi

        # 验证文件格式
        if command -v file &> /dev/null && ! file "$GO_TAR" | grep -q 'gzip'; then
            log_error "下载的文件不是有效的 tar.gz 格式"
            rm -f "$GO_TAR"
            exit 1
        fi

        log_info "正在安装 Go ..."
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf "$GO_TAR"
        rm -f "$GO_TAR"

        # 配置环境变量
        if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
            echo '' >> ~/.bashrc
            echo '# Go 环境变量' >> ~/.bashrc
            echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
            echo 'export GOPATH=$HOME/go' >> ~/.bashrc
            echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
        fi
        if [ -d /etc/profile.d ] && [ ! -f /etc/profile.d/go.sh ]; then
            sudo bash -c 'echo "export PATH=\$PATH:/usr/local/go/bin" > /etc/profile.d/go.sh'
            sudo chmod +x /etc/profile.d/go.sh
        fi

        # 配置 Go 国内代理
        /usr/local/go/bin/go env -w GOPROXY=https://goproxy.cn,direct
        /usr/local/go/bin/go env -w GOSUMDB=sum.golang.google.cn
        log_info "Go 代理已设置为 goproxy.cn"

        if /usr/local/go/bin/go version &> /dev/null; then
            log_info "Go $GO_VERSION 安装成功: $(/usr/local/go/bin/go version)"
        else
            log_error "Go 安装失败，请检查 /usr/local/go/ 目录"
            exit 1
        fi
    fi

    # ========== 安装 Node.js ==========
    # 检测 glibc 版本，选择兼容的 Node.js
    GLIBC_VER=$(ldd --version 2>&1 | head -1 | grep -oP '\d+\.\d+$' || echo "2.17")
    GLIBC_MAJOR=$(echo "$GLIBC_VER" | cut -d. -f1)
    GLIBC_MINOR=$(echo "$GLIBC_VER" | cut -d. -f2)
    
    if [ "$GLIBC_MAJOR" -ge 2 ] && [ "$GLIBC_MINOR" -ge 28 ]; then
        NODE_VERSION="20.12.2"
        log_info "glibc $GLIBC_VER >= 2.28，使用 Node.js $NODE_VERSION"
    else
        NODE_VERSION="16.20.2"
        log_warn "glibc $GLIBC_VER < 2.28 (可能是 CentOS 7)，使用 Node.js $NODE_VERSION (兼容模式)"
    fi
    export PATH=$PATH:/usr/local/node/bin

    if command -v node &> /dev/null; then
        CURRENT_NODE=$(node --version)
        log_info "Node.js 已安装: $CURRENT_NODE"
    else
        NODE_TAR="node-v${NODE_VERSION}-linux-${NODE_ARCH}.tar.xz"
        cd /tmp
        rm -f "$NODE_TAR"

        log_info "正在通过国内镜像下载 Node.js $NODE_VERSION ..."
        if ! download_file "${NODE_MIRROR}/v${NODE_VERSION}/${NODE_TAR}" "$NODE_TAR"; then
            log_warn "国内镜像下载失败，尝试官方源..."
            download_file "https://nodejs.org/dist/v${NODE_VERSION}/${NODE_TAR}" "$NODE_TAR" || {
                log_error "Node.js 下载失败"
                exit 1
            }
        fi

        log_info "正在安装 Node.js ..."
        sudo rm -rf /usr/local/node
        sudo mkdir -p /usr/local/node
        sudo tar -xJf "$NODE_TAR" -C /usr/local/node --strip-components=1
        rm -f "$NODE_TAR"

        # 配置环境变量
        if ! grep -q '/usr/local/node/bin' ~/.bashrc; then
            echo '' >> ~/.bashrc
            echo '# Node.js 环境变量' >> ~/.bashrc
            echo 'export PATH=$PATH:/usr/local/node/bin' >> ~/.bashrc
        fi

        # 配置 npm 国内镜像
        /usr/local/node/bin/npm config set registry $NPM_REGISTRY
        log_info "npm 镜像已设置为 $NPM_REGISTRY"

        if /usr/local/node/bin/node --version &> /dev/null; then
            log_info "Node.js $NODE_VERSION 安装成功: $(/usr/local/node/bin/node --version)"
        else
            log_error "Node.js 安装失败"
            exit 1
        fi
    fi

    # ========== 安装 gcc（SQLite CGO 依赖）==========
    if ! command -v gcc &> /dev/null; then
        log_info "正在安装 gcc（SQLite 编译依赖）..."
        if command -v apt-get &> /dev/null; then
            sudo apt-get update -y && sudo apt-get install -y gcc build-essential
        elif command -v yum &> /dev/null; then
            sudo yum install -y gcc gcc-c++ make
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y gcc gcc-c++ make
        else
            log_error "无法自动安装 gcc，请手动安装"
            exit 1
        fi
        log_info "gcc 安装成功"
    else
        log_info "gcc 已安装"
    fi

    # ========== 安装 JMeter ==========
    JMETER_VERSION="5.6.3"
    JMETER_HOME="/opt/apache-jmeter-${JMETER_VERSION}"
    if command -v jmeter &> /dev/null; then
        log_info "JMeter 已安装: $(jmeter --version 2>/dev/null | head -1 || echo '已存在')"
    elif [ -d "$JMETER_HOME" ]; then
        log_info "JMeter 已存在: $JMETER_HOME"
        if ! grep -q "$JMETER_HOME/bin" ~/.bashrc; then
            echo '' >> ~/.bashrc
            echo '# JMeter 环境变量' >> ~/.bashrc
            echo "export JMETER_HOME=$JMETER_HOME" >> ~/.bashrc
            echo 'export PATH=$PATH:$JMETER_HOME/bin' >> ~/.bashrc
        fi
        export PATH=$PATH:$JMETER_HOME/bin
    else
        # 检查 Java
        if ! command -v java &> /dev/null; then
            log_info "正在安装 Java（JMeter 依赖）..."
            if command -v apt-get &> /dev/null; then
                sudo apt-get update -y && sudo apt-get install -y default-jdk
            elif command -v yum &> /dev/null; then
                sudo yum install -y java-11-openjdk java-11-openjdk-devel
            elif command -v dnf &> /dev/null; then
                sudo dnf install -y java-11-openjdk java-11-openjdk-devel
            fi
        fi

        JMETER_TAR="apache-jmeter-${JMETER_VERSION}.tgz"
        cd /tmp
        rm -f "$JMETER_TAR"

        log_info "正在通过国内镜像下载 JMeter $JMETER_VERSION ..."
        if ! download_file "${JMETER_MIRROR}/${JMETER_TAR}" "$JMETER_TAR"; then
            log_warn "国内镜像下载失败，尝试 Apache 官方源..."
            download_file "https://archive.apache.org/dist/jmeter/binaries/${JMETER_TAR}" "$JMETER_TAR" || {
                log_error "JMeter 下载失败"
                exit 1
            }
        fi
        sudo tar -xzf "$JMETER_TAR" -C /opt/
        rm -f "$JMETER_TAR"

        # 配置环境变量
        if ! grep -q "$JMETER_HOME/bin" ~/.bashrc; then
            echo '' >> ~/.bashrc
            echo '# JMeter 环境变量' >> ~/.bashrc
            echo "export JMETER_HOME=$JMETER_HOME" >> ~/.bashrc
            echo 'export PATH=$PATH:$JMETER_HOME/bin' >> ~/.bashrc
        fi
        export PATH=$PATH:$JMETER_HOME/bin

        if [ -f "$JMETER_HOME/bin/jmeter" ]; then
            log_info "JMeter $JMETER_VERSION 安装成功"
        else
            log_error "JMeter 安装失败"
            exit 1
        fi
    fi

    echo ""
    log_info "========================================"
    log_info "  所有依赖安装完成！"
    log_info "========================================"
    echo ""
    log_info "已安装组件:"
    echo "  Go:      $(go version 2>/dev/null || echo '未找到')"
    echo "  Node.js: $(node --version 2>/dev/null || echo '未找到')"
    echo "  npm:     $(npm --version 2>/dev/null || echo '未找到')"
    echo "  gcc:     $(gcc --version 2>/dev/null | head -1 || echo '未找到')"
    echo "  Java:    $(java -version 2>&1 | head -1 || echo '未找到')"
    echo "  JMeter:  $(jmeter --version 2>/dev/null | head -1 || echo $JMETER_HOME)"
    echo ""
    log_warn "请执行 source ~/.bashrc 或重新登录使环境变量生效"
    log_info "然后运行: ./deploy.sh install && ./deploy.sh start"
}

# 安装 systemd 服务
install_service() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 sudo 运行此命令"
        exit 1
    fi
    
    if [ ! -f "$APP_DIR/$APP_NAME" ]; then
        log_error "未找到可执行文件，请先运行: ./deploy.sh install"
        exit 1
    fi
    
    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=JMeter Admin Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/$APP_NAME
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "$APP_NAME"
    
    log_info "Systemd 服务已安装: $SERVICE_FILE"
    log_info "使用以下命令管理服务:"
    echo "  systemctl start $APP_NAME"
    echo "  systemctl stop $APP_NAME"
    echo "  systemctl restart $APP_NAME"
    echo "  systemctl status $APP_NAME"
}

# 主入口
case "${1:-}" in
    install)
        install
        ;;
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    install-service)
        install_service
        ;;
    install-deps)
        install_deps
        ;;
    *)
        echo "JMeter Admin 部署脚本"
        echo ""
        echo "用法: $0 [install-deps|install|start|stop|restart|status|install-service]"
        echo ""
        echo "命令:"
        echo "  install-deps    一键安装所有依赖（Go + Node.js + gcc + Java + JMeter）"
        echo "  install         编译项目（需要 Go 和 Node.js 环境）"
        echo "  start           后台启动服务"
        echo "  stop            停止服务"
        echo "  restart         重启服务"
        echo "  status          查看服务状态"
        echo "  install-service 安装 systemd 服务（需要 root 权限）"
        echo ""
        echo "首次一键部署:"
        echo "  $0 install-deps     # 第1步：安装所有依赖（Go + Node.js + gcc + Java + JMeter）"
        echo "  source ~/.bashrc    # 第2步：刷新环境变量"
        echo "  $0 install          # 第3步：编译项目（前端+后端）"
        echo "  $0 start            # 第4步：启动服务"
        echo ""
        echo "  或者一条命令搞定（安装依赖后）:"
        echo "  $0 install-deps && source ~/.bashrc && $0 install && $0 start"
        exit 1
        ;;
esac
