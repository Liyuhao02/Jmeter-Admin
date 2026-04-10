# JMeter Agent系统

<cite>
**本文档引用的文件**
- [main.go](file://main.go)
- [cmd/agent/main.go](file://cmd/agent/main.go)
- [config/config.go](file://config/config.go)
- [internal/router/router.go](file://internal/router/router.go)
- [internal/agent/server.go](file://internal/agent/server.go)
- [internal/database/db.go](file://internal/database/db.go)
- [internal/model/script.go](file://internal/model/script.go)
- [internal/model/slave.go](file://internal/model/slave.go)
- [internal/model/execution.go](file://internal/model/execution.go)
- [internal/service/execution.go](file://internal/service/execution.go)
- [internal/service/slave.go](file://internal/service/slave.go)
- [internal/service/csv_split.go](file://internal/service/csv_split.go)
- [internal/handler/execution.go](file://internal/handler/execution.go)
- [internal/handler/slave.go](file://internal/handler/slave.go)
- [README.md](file://README.md)
</cite>

## 目录
1. [简介](#简介)
2. [项目结构](#项目结构)
3. [核心组件](#核心组件)
4. [架构概览](#架构概览)
5. [详细组件分析](#详细组件分析)
6. [依赖关系分析](#依赖关系分析)
7. [性能考虑](#性能考虑)
8. [故障排除指南](#故障排除指南)
9. [结论](#结论)

## 简介

JMeter Admin是一个基于Go语言开发的分布式JMeter压力测试管理系统，采用Go (Gin) + Vue 3 (Element Plus) + SQLite技术栈构建。该系统提供了完整的JMeter分布式压测管理功能，包括脚本管理、Slave节点管理、Agent节点服务、CSV自动拆分分发、分布式压测执行等核心功能。

系统的核心特点包括：
- 单文件部署的分布式压测管理平台
- 轻量级的Agent节点服务，提供文件分发和系统监控能力
- 自动化的CSV文件拆分和分发机制
- 实时监控和错误分析功能
- 支持本地模式和分布式模式的压测执行

## 项目结构

项目的整体架构采用分层设计，主要包含以下层次：

```mermaid
graph TB
subgraph "应用层"
Web[Web前端界面]
API[RESTful API接口]
end
subgraph "服务层"
Handler[HTTP处理器]
Service[业务逻辑层]
end
subgraph "数据访问层"
Model[数据模型]
DB[(SQLite数据库)]
end
subgraph "基础设施层"
Config[配置管理]
Agent[Agent服务]
end
Web --> API
API --> Handler
Handler --> Service
Service --> Model
Model --> DB
Service --> Agent
Config --> Handler
Config --> Service
```

**图表来源**
- [main.go:28-66](file://main.go#L28-L66)
- [internal/router/router.go:14-117](file://internal/router/router.go#L14-L117)

### 核心模块组织

项目采用按功能模块划分的组织方式：

- **cmd/**: 应用程序入口点
  - `agent/`: Agent节点服务入口
  - `main.go`: 主应用程序入口

- **config/**: 配置管理模块
  - `config.go`: 配置结构定义和加载逻辑

- **internal/**: 核心业务逻辑
  - `agent/`: Agent服务实现
  - `database/`: 数据库初始化和管理
  - `handler/`: HTTP请求处理器
  - `model/`: 数据模型定义
  - `router/`: 路由配置
  - `service/`: 业务逻辑服务

- **web/**: 前端Vue.js应用
  - `src/`: 源代码
  - `public/`: 静态资源

**章节来源**
- [README.md:118-152](file://README.md#L118-L152)

## 核心组件

### 主应用程序组件

主应用程序负责系统的整体初始化和协调工作：

```mermaid
classDiagram
class MainApplication {
+init() void
+createDirectories() error
+setupDatabase() error
+setupRouter() *gin.Engine
+startServer() error
}
class ConfigManager {
+LoadConfig(path string) error
+SaveConfig(path string) error
+GlobalConfig Config
}
class DatabaseManager {
+InitDB() error
+CloseDB() error
+DB *sql.DB
}
class RouterManager {
+SetupRouter(embed.FS) *gin.Engine
+corsMiddleware() gin.HandlerFunc
}
MainApplication --> ConfigManager : "使用"
MainApplication --> DatabaseManager : "使用"
MainApplication --> RouterManager : "使用"
```

**图表来源**
- [main.go:28-66](file://main.go#L28-L66)
- [config/config.go:42-86](file://config/config.go#L42-L86)
- [internal/database/db.go:15-34](file://internal/database/db.go#L15-L34)

### Agent节点服务组件

Agent服务是运行在每个Slave节点上的轻量级辅助服务：

```mermaid
classDiagram
class AgentServer {
-dataDir string
-token string
-mux *http.ServeMux
+NewServer(dataDir, token) *AgentServer
+setupRoutes() void
+Start(addr string) error
+handleHealth() void
+handleUpload() void
+handleFileOperations() void
}
class SystemStats {
+CPU CPUStats
+Memory MemoryStats
+Disk DiskStats
+Network NetworkStats
}
class FileOperation {
+isValidFilename(filename string) bool
+writeJSON(w, status, data) void
}
AgentServer --> SystemStats : "收集"
AgentServer --> FileOperation : "使用"
```

**图表来源**
- [internal/agent/server.go:89-113](file://internal/agent/server.go#L89-L113)
- [internal/agent/server.go:25-87](file://internal/agent/server.go#L25-L87)

**章节来源**
- [main.go:28-66](file://main.go#L28-L66)
- [cmd/agent/main.go:14-49](file://cmd/agent/main.go#L14-L49)

## 架构概览

系统采用典型的三层架构设计，结合微服务思想实现了分布式压测管理：

```mermaid
graph TB
subgraph "用户界面层"
UI[Vue.js前端]
Router[前端路由]
end
subgraph "API网关层"
Gin[Gin框架]
CORS[CORS中间件]
Static[静态文件服务]
end
subgraph "业务逻辑层"
ExecutionSvc[执行服务]
SlaveSvc[节点服务]
CSVService[CSV处理服务]
AgentClient[Agent客户端]
end
subgraph "数据持久层"
SQLite[(SQLite数据库)]
FS[(文件系统)]
end
subgraph "外部系统"
JMeter[JMeter引擎]
SlaveNodes[Slave节点]
AgentServices[Agent服务]
end
UI --> Router
Router --> Gin
Gin --> ExecutionSvc
Gin --> SlaveSvc
ExecutionSvc --> CSVService
ExecutionSvc --> AgentClient
SlaveSvc --> AgentClient
AgentClient --> AgentServices
ExecutionSvc --> JMeter
ExecutionSvc --> SlaveNodes
ExecutionSvc --> SQLite
ExecutionSvc --> FS
SlaveSvc --> SQLite
```

**图表来源**
- [internal/router/router.go:14-117](file://internal/router/router.go#L14-L117)
- [internal/service/execution.go:132-686](file://internal/service/execution.go#L132-L686)
- [internal/service/slave.go:448-524](file://internal/service/slave.go#L448-L524)

### 数据流分析

系统的核心数据流包括执行流程、监控流程和文件处理流程：

```mermaid
sequenceDiagram
participant Client as "客户端"
participant API as "API处理器"
participant Service as "业务服务"
participant Agent as "Agent服务"
participant JMeter as "JMeter引擎"
Client->>API : 创建执行请求
API->>Service : CreateExecution()
Service->>Service : 检查Slave节点状态
Service->>Agent : 检查Agent健康状态
Agent-->>Service : Agent健康检查结果
Service->>Service : CSV文件拆分和分发
Service->>JMeter : 启动分布式压测
JMeter-->>Service : 执行结果
Service->>Service : 解析结果和生成报告
Service-->>API : 执行结果
API-->>Client : 返回执行状态
```

**图表来源**
- [internal/handler/execution.go:39-54](file://internal/handler/execution.go#L39-L54)
- [internal/service/execution.go:132-686](file://internal/service/execution.go#L132-L686)

**章节来源**
- [internal/service/execution.go:132-686](file://internal/service/execution.go#L132-L686)

## 详细组件分析

### 配置管理系统

配置管理模块提供了灵活的配置加载和保存机制：

```mermaid
classDiagram
class Config {
+Server ServerConfig
+Frontend FrontendConfig
+JMeter JMeterConfig
+Slave SlaveConfig
+Dirs DirsConfig
}
class ServerConfig {
+Port int
}
class JMeterConfig {
+Path string
+MasterHostname string
+AgentCSVDataDir string
}
class SlaveConfig {
+HeartbeatInterval int
}
class DirsConfig {
+Data string
+Uploads string
+Results string
}
Config --> ServerConfig : "包含"
Config --> JMeterConfig : "包含"
Config --> SlaveConfig : "包含"
Config --> DirsConfig : "包含"
```

**图表来源**
- [config/config.go:10-42](file://config/config.go#L10-L42)

配置系统支持以下特性：
- YAML格式配置文件
- 默认值设置
- 运行时配置更新
- 配置验证和持久化

**章节来源**
- [config/config.go:42-115](file://config/config.go#L42-L115)

### 数据库管理系统

数据库模块负责SQLite数据库的初始化和表结构管理：

```mermaid
erDiagram
SCRIPTS {
integer id PK
text name
text description
text file_path
datetime created_at
datetime updated_at
}
SCRIPT_FILES {
integer id PK
integer script_id FK
text file_name
text file_path
text file_type
datetime created_at
datetime updated_at
}
SLAVES {
integer id PK
text name
text host
integer port
text status
datetime created_at
integer agent_port
text agent_token
text agent_status
datetime agent_check_time
text system_stats
integer agent_uptime
}
EXECUTIONS {
integer id PK
integer script_id FK
text script_name
text slave_ids
text status
datetime start_time
datetime end_time
integer duration
text remarks
text result_path
text report_path
text summary_data
text log_path
integer is_baseline
datetime created_at
}
SCRIPT_VERSIONS {
integer id PK
integer script_id FK
integer version_number
text content
text content_hash
text change_summary
datetime created_at
}
SCRIPT_FILES }o--|| SCRIPTS : "属于"
EXECUTIONS }o--|| SCRIPTS : "属于"
SCRIPT_VERSIONS }o--|| SCRIPTS : "属于"
```

**图表来源**
- [internal/database/db.go:37-128](file://internal/database/db.go#L37-L128)
- [internal/database/db.go:244-260](file://internal/database/db.go#L244-L260)

数据库设计特点：
- 支持脚本版本管理
- 完整的执行记录跟踪
- Slave节点状态监控
- 索引优化查询性能

**章节来源**
- [internal/database/db.go:15-287](file://internal/database/db.go#L15-L287)

### 执行管理系统

执行管理模块是系统的核心业务逻辑，负责分布式压测的完整生命周期管理：

```mermaid
flowchart TD
Start([创建执行请求]) --> ValidateInput["验证输入参数"]
ValidateInput --> CheckSlaves["检查Slave节点状态"]
CheckSlaves --> CreateRecord["创建执行记录"]
CreateRecord --> CreateDirs["创建结果目录"]
CreateDirs --> CheckCSV{"需要CSV拆分?"}
CheckCSV --> |是| SplitCSV["拆分CSV文件"]
CheckCSV --> |否| BuildCommand["构建JMeter命令"]
SplitCSV --> UploadFiles["上传文件到Agent"]
UploadFiles --> BuildCommand
BuildCommand --> StartJMeter["启动JMeter执行"]
StartJMeter --> MonitorProgress["监控执行进度"]
MonitorProgress --> ParseResults["解析执行结果"]
ParseResults --> UpdateStatus["更新执行状态"]
UpdateStatus --> GenerateReport["生成报告"]
GenerateReport --> End([执行完成])
```

**图表来源**
- [internal/service/execution.go:132-686](file://internal/service/execution.go#L132-L686)

执行管理的关键特性：
- 支持本地和分布式模式
- 自动CSV文件拆分和分发
- 实时日志流和监控指标
- 执行超时保护机制
- 错误详情捕获和分析

**章节来源**
- [internal/service/execution.go:132-686](file://internal/service/execution.go#L132-L686)

### Slave节点管理

Slave节点管理模块负责分布式节点的发现、监控和维护：

```mermaid
sequenceDiagram
participant Master as "Master节点"
participant Slave as "Slave节点"
participant Agent as "Agent服务"
Master->>Slave : TCP连接测试
Slave-->>Master : 连接结果
Master->>Agent : HTTP健康检查
Agent-->>Master : 健康状态和系统信息
Master->>Agent : 系统资源监控
Agent-->>Master : CPU/Memory/Disk/Network信息
Master->>Master : 更新节点状态
```

**图表来源**
- [internal/service/slave.go:295-446](file://internal/service/slave.go#L295-L446)

节点管理功能包括：
- 自动心跳检测
- 连通性诊断
- 系统资源监控
- 故障自动恢复

**章节来源**
- [internal/service/slave.go:448-524](file://internal/service/slave.go#L448-L524)

### Agent文件服务

Agent文件服务提供CSV文件的远程分发和清理功能：

```mermaid
classDiagram
class AgentServer {
+dataDir string
+token string
+handleHealth() http.HandlerFunc
+handleUpload() http.HandlerFunc
+handleFileOperations() http.HandlerFunc
+authMiddleware(next) http.HandlerFunc
}
class FileOperation {
+isValidFilename(filename) bool
+handleSingleDelete() void
+handleBatchDelete() void
}
class SystemStats {
+collectSystemStats() *SystemStats
+CPUStats CPUStats
+MemoryStats MemoryStats
+DiskStats DiskStats
+NetworkStats NetworkStats
}
AgentServer --> FileOperation : "委托"
AgentServer --> SystemStats : "使用"
```

**图表来源**
- [internal/agent/server.go:89-127](file://internal/agent/server.go#L89-L127)

Agent服务特性：
- 文件上传和删除
- 系统资源监控
- 鉴权保护
- 健康检查

**章节来源**
- [internal/agent/server.go:105-326](file://internal/agent/server.go#L105-L326)

## 依赖关系分析

系统采用模块化设计，各组件之间的依赖关系清晰明确：

```mermaid
graph TB
subgraph "核心依赖"
Gin[gin-gonic/gin]
SQLite[mattn/go-sqlite3]
YAML[yaml.v3]
end
subgraph "系统监控"
Gopsutil[shirou/gopsutil]
end
subgraph "Agent依赖"
Net[net/http]
OS[os]
Path[path/filepath]
Time[time]
end
subgraph "业务逻辑"
Execution[execution.go]
Slave[slave.go]
CSV[csv_split.go]
end
subgraph "数据访问"
DB[db.go]
Model[model/]
end
subgraph "接口层"
Handler[handler/]
Router[router.go]
end
Execution --> Gin
Execution --> SQLite
Execution --> YAML
Execution --> Gopsutil
Slave --> Gin
Slave --> SQLite
CSV --> OS
CSV --> Path
DB --> SQLite
Handler --> Gin
Handler --> Execution
Handler --> Slave
Router --> Gin
Router --> Handler
```

**图表来源**
- [go.mod:1-20](file://go.mod#L1-L20)
- [internal/service/execution.go:3-31](file://internal/service/execution.go#L3-L31)

### 外部依赖管理

系统对外部依赖的管理策略：
- 使用Go modules进行依赖管理
- 最小化第三方库依赖
- 专注于核心功能实现
- 保持代码简洁和可维护性

**章节来源**
- [go.mod:1-20](file://go.mod#L1-L20)

## 性能考虑

系统在设计时充分考虑了性能优化：

### 内存管理
- JVM堆内存动态计算，基于系统可用内存的80%
- 进程组管理和僵尸进程清理
- 文件句柄和连接池优化

### 并发处理
- Slave节点心跳检测使用goroutine并发
- CSV文件处理采用流式读取
- 日志文件使用缓冲区优化I/O

### 存储优化
- SQLite数据库索引优化查询性能
- 文件系统直接访问减少中间层
- 执行结果压缩和清理机制

## 故障排除指南

### 常见问题诊断

**Agent连接问题**
- 检查Agent服务是否启动
- 验证端口和防火墙设置
- 确认Token配置正确

**JMeter执行失败**
- 检查JMeter路径配置
- 验证Slave节点状态
- 查看执行日志获取详细错误信息

**CSV文件处理异常**
- 确认CSV文件格式正确
- 检查磁盘空间充足
- 验证文件权限设置

### 日志分析

系统提供了多层次的日志记录：
- 执行过程日志
- 错误详情日志
- 系统资源监控日志
- Agent健康检查日志

**章节来源**
- [internal/service/execution.go:559-668](file://internal/service/execution.go#L559-L668)

## 结论

JMeter Agent系统是一个设计精良的分布式压测管理平台，具有以下优势：

**技术优势**
- 采用现代化的技术栈和架构设计
- 模块化设计便于维护和扩展
- 完善的错误处理和故障恢复机制

**功能特性**
- 全面的分布式压测管理功能
- 实时监控和分析能力
- 用户友好的Web界面

**部署便利性**
- 单文件部署支持
- 跨平台兼容性
- 简化的配置管理

该系统为JMeter分布式压测提供了完整的解决方案，适合中小型团队和企业级应用场景。通过持续的功能扩展和性能优化，可以满足不断增长的压测需求。