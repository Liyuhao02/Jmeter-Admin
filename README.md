# JMeter Admin

单文件部署的 JMeter 分布式压测管理平台

## 项目介绍

JMeter Admin 是一个轻量级的 JMeter 分布式压测管理平台，采用 **Go (Gin) + Vue 3 (Element Plus) + SQLite** 技术栈开发。前端资源嵌入后端二进制文件，编译后生成单个可执行文件，实现零依赖部署。

### 核心功能

- **JMX 脚本管理** — 支持上传、可视化树形编辑、XML 源码编辑双模式
- **Slave 节点管理** — 自动心跳检测、一键连通性检查
- **分布式压测执行** — 支持单机模式与分布式模式
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

## 配置说明

`config.yaml` 文件首次启动时自动生成：

```yaml
server:
  port: 8080                    # HTTP 服务端口

jmeter:
  path: "jmeter"               # JMeter 可执行文件路径
  master_hostname: ""          # Master IP（多网卡必填，可页面配置）

dirs:
  data: "./data"               # 数据库目录
  uploads: "./uploads"         # 上传目录
  results: "./results"         # 结果目录
```

## 项目结构

```
jmeter-admin/
├── main.go                     # 后端入口
├── config.yaml                 # 配置文件
├── Makefile                    # 构建命令
├── deploy.sh                   # 一键部署脚本
├── config/                     # 配置管理
│   └── config.go
├── internal/                   # 后端核心
│   ├── database/               # 数据库初始化
│   ├── handler/                # API 处理器
│   ├── model/                  # 数据模型
│   ├── router/                 # 路由注册
│   └── service/                # 业务逻辑
├── web/                        # 前端项目
│   ├── src/
│   │   ├── api/                # API 调用封装
│   │   ├── components/         # 组件（JmxTreeEditor, ExecuteDialog）
│   │   ├── views/              # 页面
│   │   ├── utils/              # 工具（jmxParser）
│   │   └── router/             # 前端路由
│   ├── vite.config.js
│   └── package.json
├── data/                       # SQLite 数据库（运行时生成）
├── uploads/                    # 上传文件（运行时生成）
└── results/                    # 执行结果（运行时生成）
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
| GET | /:id/errors | 错误分析 |
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
| port | INTEGER | 端口 |
| status | TEXT | 状态（online/offline） |
| last_check_time | TEXT | 最后检测时间 |
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
| created_at | TEXT | 创建时间 |

## 分布式压测配置

### Master 节点配置

1. 在 `config.yaml` 中配置 `master_hostname`，或在页面「系统设置」中选择网卡 IP
2. 多网卡环境必须显式指定，否则 Slave 无法回传数据

### Slave 节点配置

1. 在页面「节点管理」中添加 Slave，填写 `host:port`
2. Slave 端启动 jmeter-server：

```bash
jmeter-server -Dserver.rmi.ssl.disable=true
```

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

## License

MIT
