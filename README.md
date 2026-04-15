# JMeter Admin

单文件部署的 JMeter 分布式压测管理平台

## 项目介绍

JMeter Admin 是一个轻量级的 JMeter 分布式压测管理平台，采用 **Go (Gin) + Vue 3 (Element Plus) + SQLite** 技术栈开发。前端资源嵌入后端二进制文件，编译后生成单个可执行文件，实现零依赖部署。

### 核心功能

- **JMX 脚本管理** — 支持上传、可视化树形编辑、XML 源码编辑双模式
- **脚本版本管理** — 自动保存编辑历史，SHA256 去重，支持版本预览和一键回滚
- **Slave 节点管理** — 自动心跳检测、一键连通性检查
- **Agent 节点服务** — 轻量级辅助服务，提供文件分发和系统监控能力
- **CSV 自动拆分分发** — 大文件流式拆分，按 Slave 数量均匀分发，执行后自动清理
- **分布式压测执行** — 支持单机模式与分布式模式
- **执行对比与基准线** — 多执行并行对比，标记基准线，自动计算指标差异百分比
- **实时监控增强** — P95/P99 响应时间分位数、实时错误趋势、网络吞吐量
- **错误分析增强** — 响应码分布饼图、错误时间线、Top 10 错误消息
- **Slave 系统监控** — 实时采集 CPU/内存/磁盘/网络，阈值告警，资源详情弹窗
- **JMX 编辑器增强** — 快捷添加工具栏、键盘快捷键、多选批量操作、全部展开/折叠
- **进程管理** — 进程组管理、僵尸进程自动清理、执行超时保护（默认4小时）
- **执行记录管理** — 实时日志流、错误分析、结果导出（JTL/报告/CSV）
- **Master IP 自动检测** — 多网卡环境自动识别或手动配置

## 系统要求

| 组件 | 版本 | 说明 |
|------|------|------|
| Go | >= 1.21 | 后端编译 |
| Node.js | >= 16.x | 前端构建 |
| gcc | 任意 | SQLite 编译依赖（CGO） |
| Java | >= 11 | JMeter 运行时 |
| JMeter | >= 5.6 | 压测引擎 |

## 快速开始

### 一键部署（Linux 服务器）

```bash
# 安装依赖（Go、Node.js、gcc、Java、JMeter）
./deploy.sh install-deps
source ~/.bashrc

# 编译项目
./deploy.sh install

# 启动服务
./deploy.sh start

# 访问 http://your-server-ip:8080
```

### 服务器后台部署（master / agent / all）

```bash
# 先复制环境模板
cp scripts/server-deploy.env.example scripts/server-deploy.env

# 按需修改 agent 端口、token、jmeter 路径
vim scripts/server-deploy.env

# 一键后台部署 master
./scripts/server-deploy.sh master deploy

# 一键后台部署 agent
./scripts/server-deploy.sh agent deploy

# 同机同时部署 master + agent
./scripts/server-deploy.sh all deploy

# 自动安装依赖（Go / Node.js / Java / JMeter）
./scripts/server-deploy.sh all install-deps

# 查看状态 / 日志
./scripts/server-deploy.sh all status
./scripts/server-deploy.sh master logs
```

### 本地开发

```bash
# 同时启动前后端（前端热更新）
make dev

# 或分别启动
make dev-backend    # 后端 :8080
make dev-frontend   # 前端 :3000（代理 API 到 :8080）

# 浏览器访问 http://localhost:3000
```

### 编译部署

```bash
# 完整编译（前端 + 后端）
make build-all

# 仅编译后端（需先构建前端）
make build-backend

# 交叉编译 Linux 版本
make build-linux

# 运行
./jmeter-admin
```

## 打包分享前清理

如果你准备把项目源码打包发给别人，建议先清掉本地构建产物、运行结果、数据库和上传文件，避免把无关的大文件、日志或本地环境信息一起带出去。

### 推荐命令

```bash
# 一键清理构建产物、本地数据库、上传文件、执行结果和前端依赖
make clean-share
```

### `make clean-share` 会清理什么

- 后端二进制：`jmeter-admin`、`jmeter-agent`
- 前端构建产物：`web/dist`、`web/.vite`
- 前端依赖：`web/node_modules`
- Go 本地缓存：`.gocache`
- 本地运行数据：`data`、`uploads`、`results`、`temp`
- 本地日志和输出：`jmeter.log`、`*.out`
- 本地生成配置：`config.yaml`
- macOS 元数据：`.DS_Store`

### 等价命令

如果不想走 `make`，也可以直接执行下面这组命令：

```bash
rm -f jmeter-admin jmeter-agent jmeter.log *.out config.yaml
rm -rf .gocache temp data uploads results web/dist web/node_modules web/.vite
find . -name '.DS_Store' -delete
```

### 注意事项

- 以上命令会删除本地数据库、上传脚本、执行结果和日志，执行前请确认这些内容已经备份。
- 清理后如果你还要本地开发，前端依赖需要重新安装：

```bash
cd web && npm install
```

- 如果你只是想清理编译产物，不想删除本地数据，可以使用：

```bash
make clean
```

## 配置说明

`config.yaml` 文件首次启动时自动生成：

```yaml
server:
  port: 8080

jmeter:
  path: "jmeter"
  master_hostname: ""
  agent_csv_data_dir: "/opt/jmeter/csv-data"

slave:
  heartbeat_interval: 30

dirs:
  data: "./data"
  uploads: "./uploads"
  results: "./results"
```

配置项说明：

| 配置项 | 说明 |
|--------|------|
| `server.port` | HTTP 服务端口 |
| `jmeter.path` | JMeter 可执行文件路径 |
| `jmeter.master_hostname` | Master IP（多网卡必填，可页面配置） |
| `jmeter.agent_csv_data_dir` | Agent CSV 文件存放目录 |
| `slave.heartbeat_interval` | Slave 心跳检测间隔（秒） |
| `dirs.data` | 数据库目录 |
| `dirs.uploads` | 上传目录 |
| `dirs.results` | 结果目录 |

## 项目结构

```
jmeter-admin/
├── main.go                     # 后端入口
├── cmd/agent/main.go           # Agent 入口
├── config.yaml                 # 配置文件
├── Makefile                    # 构建命令
├── deploy.sh                   # 一键部署脚本
├── config/                     # 配置管理
│   └── config.go
├── internal/                   # 后端核心
│   ├── agent/                  # Agent 服务
│   │   └── server.go
│   ├── database/               # 数据库初始化
│   │   └── db.go
│   ├── handler/                # API 处理器
│   ├── model/                  # 数据模型
│   ├── router/                 # 路由注册
│   └── service/                # 业务逻辑
│       ├── agent_client.go     # Agent 客户端
│       └── csv_split.go        # CSV 拆分逻辑
├── web/                        # 前端项目
│   ├── src/
│   │   ├── api/                # API 调用封装
│   │   ├── components/         # 组件
│   │   ├── views/              # 页面
│   │   ├── utils/              # 工具
│   │   └── router/             # 前端路由
│   ├── vite.config.js
│   └── package.json
├── data/                       # SQLite 数据库
├── uploads/                    # 上传文件
└── results/                    # 执行结果
```

## API 文档

### 脚本管理 `/api/scripts`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | / | 脚本列表 |
| POST | / | 创建脚本 |
| GET | /:id | 脚本详情 |
| PUT | /:id | 更新脚本 |
| DELETE | /:id | 删除脚本 |
| GET | /:id/download | 下载主脚本 |
| GET | /:id/content | 获取 JMX 内容 |
| PUT | /:id/content | 保存 JMX 内容 |
| POST | /:id/files | 上传附件 |
| DELETE | /:id/files/:fileId | 删除附件 |

### 脚本版本管理 `/api/scripts/:id/versions`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | / | 获取版本历史列表 |
| GET | /:versionId | 获取历史版本内容 |
| POST | /:versionId/restore | 恢复到指定版本 |

### Slave 管理 `/api/slaves`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | / | Slave 列表 |
| POST | / | 添加 Slave |
| PUT | /:id | 更新 Slave |
| DELETE | /:id | 删除 Slave |
| POST | /:id/check | 连通性检测 |
| GET | /heartbeat-status | 心跳状态 |

### 执行管理 `/api/executions`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | / | 执行列表（支持筛选） |
| GET | /stats | 统计汇总 |
| POST | / | 创建执行 |
| GET | /:id | 执行详情 |
| DELETE | /:id | 删除执行 |
| POST | /:id/stop | 停止执行 |
| GET | /:id/log | 实时日志（SSE） |
| GET | /:id/live-metrics | 实时监控指标（含 P95/P99） |
| PUT | /:id/baseline | 设置/取消基准线 |
| GET | /compare | 多执行对比 |
| GET | /:id/errors | 错误分析（响应码分布/时间线/Top10） |
| POST | /:id/error-details/upload | 上传错误详情 |
| GET | /:id/download/jtl | 下载 JTL |
| GET | /:id/download/report | 下载报告 |
| GET | /:id/download/errors | 导出错误 CSV |
| GET | /:id/download/all | 下载全部结果 |

### 系统配置 `/api/config`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /network-interfaces | 网卡列表 |
| GET | /master-hostname | Master IP |
| PUT | /master-hostname | 更新 Master IP |

## 数据库表结构

### scripts — 脚本表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| name | TEXT | 脚本名称 |
| description | TEXT | 描述 |
| file_path | TEXT | 主 JMX 文件路径 |
| created_at | TEXT | 创建时间 |
| updated_at | TEXT | 更新时间 |

### script_files — 脚本附件表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| script_id | INTEGER | 关联脚本 ID |
| file_name | TEXT | 文件名 |
| file_path | TEXT | 文件路径 |
| file_type | TEXT | 文件类型（jmx/csv/jar/other） |
| created_at | TEXT | 创建时间 |
| updated_at | TEXT | 更新时间 |

### slaves — Slave 节点表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| name | TEXT | 节点名称 |
| host | TEXT | 主机地址 |
| port | INTEGER | JMeter RMI 端口 |
| agent_port | INTEGER | Agent 端口 |
| agent_token | TEXT | Agent 鉴权 Token |
| status | TEXT | 状态（online/offline） |
| last_check_time | TEXT | 最后检测时间 |
| system_stats | TEXT | Agent 系统资源 JSON |
| agent_uptime | INTEGER | Agent 运行秒数 |
| created_at | TEXT | 创建时间 |

### executions — 执行记录表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| script_id | INTEGER | 关联脚本 ID |
| script_name | TEXT | 脚本名称（冗余） |
| slave_ids | TEXT | 选中 Slave ID（JSON 数组） |
| status | TEXT | 状态（running/success/failed/stopped） |
| start_time | TEXT | 开始时间 |
| end_time | TEXT | 结束时间 |
| duration | INTEGER | 执行时长（秒） |
| remarks | TEXT | 备注 |
| result_path | TEXT | 结果目录路径 |
| report_path | TEXT | 报告目录路径 |
| summary_data | TEXT | 汇总数据（JSON） |
| log_path | TEXT | 日志文件路径 |
| is_baseline | INTEGER | 是否为基准线（0/1） |
| created_at | TEXT | 创建时间 |

### script_versions — 脚本版本表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| script_id | INTEGER | 关联脚本 ID |
| content_hash | TEXT | 内容哈希（SHA256） |
| content | TEXT | JMX 内容 |
| created_at | DATETIME | 创建时间 |

## Agent 节点服务

Agent 是轻量级辅助服务，运行在每台 JMeter Slave 节点上，为 Master 提供：
- CSV 数据文件的远程分发和清理
- 节点系统资源实时监控（CPU/内存/磁盘/网络）
- 健康检查和连通性诊断

### Agent 编译

```bash
# 在项目根目录
go build -o jmeter-agent ./cmd/agent/

# 或交叉编译 Linux 版本
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o jmeter-agent ./cmd/agent/
```

注意：Agent 依赖 gopsutil 进行系统监控，编译需要 CGO 支持（gcc）。

### Agent 部署步骤

**1. 前置依赖**
- Go 1.21+（仅编译时需要）
- gcc（CGO 编译需要）
- JMeter 5.6+（Slave 节点本身需要运行 jmeter-server）
- Java 11+

**2. 部署方式一：直接复制二进制**
```bash
# 在 Master 机器上编译
go build -o jmeter-agent ./cmd/agent/

# 复制到 Slave 机器
scp jmeter-agent user@slave-host:/opt/jmeter/

# 在 Slave 上启动
ssh user@slave-host
cd /opt/jmeter
mkdir -p csv-data
./jmeter-agent -port 8089 -data-dir ./csv-data
```

**3. 部署方式二：使用 deploy.sh 编译**
```bash
# deploy.sh install 会同时编译 jmeter-admin（Master）和 jmeter-agent（Agent）两个二进制
./deploy.sh install
# 编译产物：./jmeter-admin 和 ./jmeter-agent
```

### Agent 启动参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-port` | 8089 | Agent HTTP 监听端口 |
| `-data-dir` | ./csv-data | CSV 文件存放目录 |
| `-token` | (空) | 鉴权 token，为空则不校验 |

### Agent 启动示例

```bash
# 基本启动
./jmeter-agent -port 8089 -data-dir /opt/jmeter/csv-data

# 启用 token 鉴权
./jmeter-agent -port 8089 -data-dir /opt/jmeter/csv-data -token "your-secret-token"

# 后台运行
nohup ./jmeter-agent -port 8089 -data-dir /opt/jmeter/csv-data > agent.log 2>&1 &
```

### Agent API 端点

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/health` | 否 | 健康检查 + 系统资源监控 |
| POST | `/api/files/upload` | 是 | 上传文件（CSV 等） |
| DELETE | `/api/files/{filename}` | 是 | 删除单个文件 |
| DELETE | `/api/files/batch` | 是 | 批量删除文件 |

### Agent /health 响应示例

```json
{
  "status": "ok",
  "version": "1.0.0",
  "data_dir": "/opt/jmeter/csv-data",
  "uptime_seconds": 3600,
  "sys_stats": {
    "cpu": {"percent": 45.5, "count": 8},
    "memory": {"total_mb": 16384, "used_mb": 8192, "percent": 50.0},
    "disk": {"total_mb": 1048576, "used_mb": 524288, "percent": 50.0},
    "network": {"connections": 42}
  }
}
```

### Agent + Master 联动配置

在 Master 的 Slave 管理页面中添加 Slave 时，需要填写：
- **主机地址**: Slave 的 IP
- **JMeter RMI 端口**: 默认 1099（jmeter-server 端口）
- **Agent 端口**: 默认 8089
- **Agent Token**: 如果 Agent 启用了 token 鉴权

Master 会通过心跳检测（默认 30 秒）自动检测 Agent 连通性和采集系统资源。

## 分布式压测配置

### Master 节点配置

1. 在 `config.yaml` 中配置 `master_hostname`，或在页面「系统设置」中选择网卡 IP
2. 多网卡环境必须显式指定，否则 Slave 无法回传数据

### Slave 节点配置

1. 在页面「节点管理」中添加 Slave，填写主机地址、JMeter RMI 端口、Agent 端口
2. Slave 端启动 jmeter-server：

```bash
jmeter-server -Dserver.rmi.ssl.disable=true
```

3. 启动 Agent 服务（参考上文 Agent 部署步骤）

### 多网卡环境注意事项

- Master 有多个网卡时，RMI 回调 IP 可能错误
- 必须在配置中指定 `master_hostname` 为 Slave 可访问的 IP
- 系统会自动将 `-Djava.rmi.server.hostname` 传递给 JMeter

## 服务管理

```bash
./deploy.sh start     # 启动
./deploy.sh stop      # 停止
./deploy.sh restart   # 重启
./deploy.sh status    # 状态
```

### systemd 服务（可选）

```bash
sudo ./deploy.sh install-service
sudo systemctl enable jmeter-admin
sudo systemctl start jmeter-admin
```

## 常见问题

### Q: 编译报错 `CGO_ENABLED` 相关？

确保系统已安装 gcc：

```bash
# Ubuntu/Debian
sudo apt-get install -y gcc build-essential

# CentOS/RHEL
sudo yum install -y gcc gcc-c++ make
```

### Q: 前端构建慢？

使用 npmmirror 镜像加速：

```bash
npm config set registry https://registry.npmmirror.com
```

### Q: Slave 连接失败？

依次检查：

1. `master_hostname` 配置是否正确
2. 防火墙是否开放端口（默认 50000, 1099）
3. Slave 端是否禁用 RMI SSL：`-Dserver.rmi.ssl.disable=true`

### Q: JMeter OOM？

系统自动根据可用内存分配 JVM 堆（80% 可用内存），无需手动配置。

### Q: SQLite 迁移报错？

删除数据库文件重新创建：

```bash
rm -f data/jmeter-admin.db
./jmeter-admin
```

## JMX 编辑器快捷键

| 快捷键 | 功能 |
|--------|------|
| Delete / Backspace | 删除选中节点 |
| Ctrl+D | 复制节点 |
| Ctrl+Shift+E | 启用/禁用节点 |
| Ctrl+↑ | 上移节点 |
| Ctrl+↓ | 下移节点 |
| Ctrl+Click | 多选节点 |

## License

MIT
