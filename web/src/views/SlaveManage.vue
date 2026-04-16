<template>
  <div class="slave-page">
    <section class="page-header-bar">
      <div class="page-header-copy">
        <div class="page-header-kicker">CLUSTER</div>
        <h1>节点管理</h1>
        <p>集中查看节点状态、环境差异和分布式执行可用性。</p>
      </div>
      <div class="page-header-pills">
        <span class="workspace-pill">节点 {{ slaveList.length }}</span>
        <span class="workspace-pill">在线 {{ onlineSlaveCount }}</span>
        <span class="workspace-pill">Agent {{ agentOnlineCount }}</span>
        <span class="workspace-pill">高负载 {{ busySlaveCount }}</span>
      </div>
    </section>

    <!-- Master 节点配置卡片 -->
    <div class="master-config-card">
      <div class="master-config-header">
        <div class="master-config-title">
          <el-icon class="title-icon"><Connection /></el-icon>
          <span>Master 节点配置</span>
        </div>
        <div class="master-config-meta">
          <span class="env-chip" :class="`is-${getEnvironmentTone(masterNode)}`">{{ getEnvironmentStatusTag(masterNode) }}</span>
          <span class="env-chip" v-if="getEnvironmentVersion(masterNode)">{{ getEnvironmentVersion(masterNode) }}</span>
          <span class="env-chip subtle">心跳 {{ lastHeartbeatTime || '等待检测' }}</span>
          <el-tooltip content="Slave 节点通过此 IP 将测试结果回传给 Master，多网卡环境需指定正确的 IP" placement="top">
            <el-icon class="info-icon"><InfoFilled /></el-icon>
          </el-tooltip>
        </div>
      </div>
      <div class="master-config-body">
        <div class="config-item">
          <span class="config-label">主节点地址</span>
          <el-select
            v-model="masterHostname"
            filterable
            allow-create
            placeholder="选择或输入 Master IP"
            @change="handleMasterIPChange"
            class="master-ip-select"
            size="default"
          >
            <el-option
              v-for="iface in networkInterfaces"
              :key="iface.ip"
              :label="`${iface.ip} (${iface.name})`"
              :value="iface.ip"
            />
          </el-select>
        </div>
        <div class="config-status">
          <template v-if="masterHostname">
            <el-icon class="status-icon success"><CircleCheckFilled /></el-icon>
            <span class="status-text success">已配置: {{ masterHostname }}</span>
          </template>
          <template v-else>
            <el-icon class="status-icon warning"><WarningFilled /></el-icon>
            <span class="status-text warning">未配置，多网卡环境可能导致 Slave 连接失败</span>
          </template>
        </div>
      </div>

      <div class="master-dashboard">
        <div class="master-primary-panel">
          <div class="master-overview-card is-callback">
            <span class="overview-card-label">Master 回调基地址</span>
            <code class="overview-card-code">{{ masterCallbackBaseURL || '未配置' }}</code>
          </div>
          <div class="master-overview-grid">
            <div class="master-overview-card">
              <span class="overview-card-label">CPU</span>
              <strong class="overview-card-value">{{ formatPercent(masterNode.parsedStats?.cpu?.percent) }}</strong>
            </div>
            <div class="master-overview-card">
              <span class="overview-card-label">内存</span>
              <strong class="overview-card-value">{{ formatPercent(masterNode.parsedStats?.memory?.percent) }}</strong>
            </div>
            <div class="master-overview-card">
              <span class="overview-card-label">磁盘</span>
              <strong class="overview-card-value">{{ formatPercent(masterNode.parsedStats?.disk?.percent) }}</strong>
            </div>
            <div class="master-overview-card">
              <span class="overview-card-label">连接</span>
              <strong class="overview-card-value">{{ formatCount(masterNode.parsedStats?.network?.connections) }}</strong>
            </div>
          </div>
        </div>

        <div class="master-env-panel" :class="`is-${getEnvironmentTone(masterNode)}`">
          <div class="master-env-summary">
            <div class="master-env-copy">
              <span class="master-env-label">Master 环境检测</span>
              <strong>{{ getEnvironmentHeadline(masterNode) }}</strong>
              <p>{{ getEnvironmentDescription(masterNode) }}</p>
            </div>
            <div class="master-env-badges">
              <span class="env-badge" :class="`is-${getEnvironmentTone(masterNode)}`">{{ getEnvironmentStatusTag(masterNode) }}</span>
              <span class="env-badge" v-if="getEnvironmentVersion(masterNode)">{{ getEnvironmentVersion(masterNode) }}</span>
            </div>
          </div>

          <div class="master-env-grid">
            <div class="master-env-card">
              <span>JMeter</span>
              <strong>{{ getEnvironmentVersion(masterNode) || '未检测到' }}</strong>
            </div>
            <div class="master-env-card">
              <span>插件</span>
              <strong>{{ getPluginCountLabel(masterNode) }}</strong>
            </div>
            <div class="master-env-card">
              <span>配置</span>
              <strong>{{ getPropertiesStateLabel(masterNode) }}</strong>
            </div>
            <div class="master-env-card">
              <span>告警</span>
              <strong>{{ getEnvironmentWarnings(masterNode).length }} 条</strong>
            </div>
          </div>

          <div v-if="getEnvironmentWarnings(masterNode).length" class="master-env-warning-list">
            <div
              v-for="(warning, index) in getEnvironmentWarnings(masterNode).slice(0, 2)"
              :key="`master-warning-${index}`"
              class="master-env-warning"
            >
              <el-icon><WarningFilled /></el-icon>
              <span>{{ warning }}</span>
            </div>
          </div>
        </div>
      </div>

      <div v-if="clusterAdvisories.length" class="cluster-alert-strip">
        <div
          v-for="(advice, index) in clusterAdvisories.slice(0, 4)"
          :key="`cluster-advice-${index}`"
          class="cluster-alert-card"
          :class="`is-${advice.tone}`"
        >
          <span class="cluster-alert-label">{{ advice.label }}</span>
          <strong>{{ advice.title }}</strong>
          <p>{{ advice.detail }}</p>
        </div>
      </div>

      <!-- 心跳状态指示 -->
      <div class="heartbeat-status-bar">
        <div class="heartbeat-indicator">
          <span class="pulse-dot"></span>
          <span class="heartbeat-text">心跳检测运行中</span>
        </div>
        <span class="heartbeat-divider">·</span>
        <span class="heartbeat-info">每 30 秒自动检测</span>
        <span class="heartbeat-divider">·</span>
        <span class="heartbeat-info">上次检测: {{ lastHeartbeatTime || '-' }}</span>
      </div>
    </div>

    <!-- 节点管理区域 -->
    <div class="section-card">
      <div class="section-header-with-action">
        <div class="section-header">
          <div class="section-label">NODES</div>
          <div class="section-title">节点列表</div>
          <div class="section-desc">查看每个节点的连通性、资源负载、JMeter 安装版本与环境告警</div>
        </div>
        <div class="section-actions">
          <el-button @click="handleCheckAll" :loading="checkingAll">
            <el-icon v-if="!checkingAll" class="btn-icon"><CircleCheck /></el-icon>
            全部检测
          </el-button>
          <el-button type="primary" @click="handleAdd">
            <el-icon class="btn-icon"><Plus /></el-icon>
            添加节点
          </el-button>
        </div>
      </div>

      <div class="list-utility-bar">
        <span class="utility-chip">共 {{ slaveList.length }} 个节点</span>
        <span class="utility-chip">在线 {{ onlineSlaveCount }} 个</span>
        <span class="utility-chip">Agent 在线 {{ agentOnlineCount }} 个</span>
        <span class="utility-chip">平均 CPU {{ averageCpuText }}</span>
        <span class="utility-chip">高负载 {{ busySlaveCount }} 个</span>
        <span class="utility-chip">环境差异 {{ conflictNodeCount }} 个</span>
      </div>

      <!-- 节点表格 -->
      <div class="table-shell">
        <el-table
          v-loading="loading"
          :data="slaveList"
          class="slaves-table"
          stripe
        >
        <el-table-column label="名称" min-width="156" sortable prop="name" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="node-name-cell">
              <el-icon class="node-icon"><Monitor /></el-icon>
              <div class="node-name-stack">
                <span class="node-name">{{ row.name }}</span>
                <span class="node-subtext">节点 #{{ row.id }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="地址" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="node-address-stack">
              <code class="address-code">{{ row.host }}</code>
              <span class="node-subtext">RMI {{ row.port }} · Agent {{ row.agent_port || '--' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="连通状态" width="180" align="center">
          <template #default="{ row }">
            <div class="connectivity-cell">
              <div class="status-cell">
                <span class="status-dot" :class="row.status"></span>
                <span class="status-label" :class="row.status">JMeter {{ getStatusText(row.status) }}</span>
              </div>
              <div class="status-cell">
                <span class="status-dot" :class="row.agent_status || 'unknown'"></span>
                <span class="status-label" :class="row.agent_status || 'unknown'">Agent {{ getAgentStatusText(row.agent_status) }}</span>
              </div>
            </div>
            <div v-if="row.diagnostic && (row.diagnostic.jmeter_latency_ms > 0 || row.diagnostic.agent_latency_ms > 0)" class="latency-text">
              {{ row.diagnostic.jmeter_latency_ms || '--' }}ms / {{ row.diagnostic.agent_latency_ms || '--' }}ms
            </div>
          </template>
        </el-table-column>
        <el-table-column label="资源概览" width="210" sortable :sort-method="(a,b) => sortByStats(a,b,'cpu')">
          <template #default="{ row }">
            <div v-if="row.parsedStats" class="resource-cell">
              <span class="resource-badge">CPU {{ (row.parsedStats.cpu?.percent || 0).toFixed(0) }}%</span>
              <span class="resource-badge">内存 {{ (row.parsedStats.memory?.percent || 0).toFixed(0) }}%</span>
              <span class="resource-badge">磁盘 {{ (row.parsedStats.disk?.percent || 0).toFixed(0) }}%</span>
              <span class="resource-badge">连接 {{ formatCount(row.parsedStats.network?.connections) }}</span>
            </div>
            <span v-else class="no-data">--</span>
          </template>
        </el-table-column>
        <el-table-column label="环境概览" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="environment-cell">
              <div class="environment-cell-top">
                <span class="env-chip" :class="`is-${getEnvironmentTone(row)}`">{{ getEnvironmentStatusTag(row) }}</span>
                <span class="env-chip" v-if="getEnvironmentVersion(row)">{{ getEnvironmentVersion(row) }}</span>
                <span class="env-chip">{{ getPluginCountLabel(row) }}</span>
              </div>
              <div class="environment-meta">
                <span>{{ getPropertiesStateLabel(row) }}</span>
                <span v-if="getNodeConflictDetails(row).length" :class="`conflict-text is-${getNodeConflictTone(row)}`">
                  {{ getNodeConflictDetails(row).length }} 项需关注
                </span>
              </div>
              <div class="environment-warning-cell" :class="{ 'is-empty': !getEnvironmentWarnings(row).length }">
                {{ getNodeConflictSummary(row) }}
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="最后检测" width="108" align="center">
          <template #default="{ row }">
            <span class="time-text" :title="row.last_check_time">
              {{ formatRelativeTime(row.last_check_time) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="182" align="center">
          <template #default="{ row }">
            <div class="action-btns">
              <el-tooltip content="检测连通性" placement="top">
                <el-button
                  link
                  type="primary"
                  @click="handleCheck(row)"
                  :loading="checkingId === row.id"
                  :aria-label="`检测节点 ${row.name}`"
                  class="action-btn icon-btn"
                >
                  <el-icon><CircleCheck /></el-icon>
                </el-button>
              </el-tooltip>
              <el-popover
                v-if="row.diagnostic && (row.diagnostic.jmeter_error || row.diagnostic.agent_error)"
                placement="top"
                :width="320"
                trigger="hover"
                popper-class="diagnostic-popover"
              >
                <template #reference>
                  <el-button
                    link
                    type="warning"
                    class="action-btn icon-btn diagnostic-btn"
                    :aria-label="`查看节点 ${row.name} 的连接诊断`"
                  >
                    <el-icon><WarningFilled /></el-icon>
                  </el-button>
                </template>
                <div class="diagnostic-panel">
                  <div class="diagnostic-title">
                    <el-icon><WarningFilled /></el-icon>
                    连接诊断
                  </div>
                  <div v-if="row.diagnostic.jmeter_error" class="diagnostic-item">
                    <div class="diagnostic-label">JMeter RMI:</div>
                    <el-tag size="small" type="danger" effect="dark">{{ getErrorText(row.diagnostic.jmeter_error) }}</el-tag>
                    <div class="diagnostic-latency" v-if="row.diagnostic.jmeter_latency_ms > 0">
                      延迟: {{ row.diagnostic.jmeter_latency_ms }}ms
                    </div>
                  </div>
                  <div v-if="row.diagnostic.agent_error" class="diagnostic-item">
                    <div class="diagnostic-label">Agent:</div>
                    <el-tag size="small" type="danger" effect="dark">{{ getErrorText(row.diagnostic.agent_error) }}</el-tag>
                    <div class="diagnostic-latency" v-if="row.diagnostic.agent_latency_ms > 0">
                      延迟: {{ row.diagnostic.agent_latency_ms }}ms
                    </div>
                  </div>
                  <div v-if="row.diagnostic.suggestion" class="diagnostic-suggestion">
                    <div class="suggestion-title">诊断建议:</div>
                    <div class="suggestion-content">{{ row.diagnostic.suggestion }}</div>
                  </div>
                </div>
              </el-popover>
              <el-tooltip content="编辑节点" placement="top">
                <el-button
                  link
                  type="primary"
                  @click="handleEdit(row)"
                  :aria-label="`编辑节点 ${row.name}`"
                  class="action-btn icon-btn edit-btn"
                >
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除节点" placement="top">
                <el-button
                  link
                  type="danger"
                  @click="handleDelete(row)"
                  :aria-label="`删除节点 ${row.name}`"
                  class="action-btn icon-btn delete-btn"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="资源详情" placement="top">
                <el-button
                  link
                  type="info"
                  @click="showStatsDialog(row)"
                  :disabled="!row.parsedStats"
                  :aria-label="`查看节点 ${row.name} 的资源详情`"
                  class="action-btn icon-btn stats-btn"
                >
                  <el-icon><InfoFilled /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
        </el-table>
      </div>

      <!-- 空状态 -->
      <div v-if="!loading && slaveList.length === 0" class="empty-state">
        <div class="empty-icon">
          <el-icon><Monitor /></el-icon>
        </div>
        <h3 class="empty-title">暂无节点</h3>
        <p class="empty-desc">请点击上方按钮添加节点</p>
      </div>
    </div>

    <!-- 资源详情弹窗 -->
    <el-dialog
      v-model="statsDialogVisible"
      title="节点资源详情"
      width="500px"
      class="stats-dialog"
      destroy-on-close
    >
      <div v-if="currentSlaveStats" class="stats-content">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="CPU 核心数">{{ currentSlaveStats.cpu?.count || '--' }}</el-descriptions-item>
          <el-descriptions-item label="CPU 使用率">
            <div style="display:flex;align-items:center;gap:10px">
              <el-progress
                :percentage="currentSlaveStats.cpu?.percent || 0"
                :color="getResourceColor(currentSlaveStats.cpu?.percent, 80)"
                style="flex:1"
              />
              <span>{{ currentSlaveStats.cpu?.percent?.toFixed(1) || 0 }}%</span>
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="内存总量">{{ formatMB(currentSlaveStats.memory?.total_mb) }}</el-descriptions-item>
          <el-descriptions-item label="内存使用率">
            <div style="display:flex;align-items:center;gap:10px">
              <el-progress
                :percentage="currentSlaveStats.memory?.percent || 0"
                :color="getResourceColor(currentSlaveStats.memory?.percent, 85)"
                style="flex:1"
              />
              <span>{{ currentSlaveStats.memory?.percent?.toFixed(1) || 0 }}%</span>
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="磁盘总量">{{ formatMB(currentSlaveStats.disk?.total_mb) }}</el-descriptions-item>
          <el-descriptions-item label="磁盘使用率">
            <div style="display:flex;align-items:center;gap:10px">
              <el-progress
                :percentage="currentSlaveStats.disk?.percent || 0"
                :color="getResourceColor(currentSlaveStats.disk?.percent, 90)"
                style="flex:1"
              />
              <span>{{ currentSlaveStats.disk?.percent?.toFixed(1) || 0 }}%</span>
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="网络连接数">{{ currentSlaveStats.network?.connections || '--' }}</el-descriptions-item>
          <el-descriptions-item label="Agent 运行时长">{{ formatUptime(currentSlaveUptime) }}</el-descriptions-item>
        </el-descriptions>
      </div>
      <div v-else style="text-align:center;color:#999;padding:40px 20px">
        <el-icon :size="48" style="margin-bottom:12px;opacity:0.5"><Monitor /></el-icon>
        <p>Agent 离线，无资源数据</p>
      </div>
    </el-dialog>

    <!-- 添加/编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑Slave节点' : '添加Slave节点'"
      width="480px"
      :close-on-click-modal="false"
      class="slave-dialog"
      @close="resetForm"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="90px"
        class="slave-form"
      >
        <el-form-item label="节点名称" prop="name">
          <el-input
            v-model="form.name"
            placeholder="请输入节点名称"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="主机地址" prop="host">
          <el-input
            v-model="form.host"
            placeholder="请输入主机地址，如 192.168.1.100"
          />
        </el-form-item>

        <el-form-item label="端口" prop="port">
          <el-input-number
            v-model="form.port"
            :min="1"
            :max="65535"
            :controls="false"
            placeholder="请输入端口"
            style="width: 100%"
          />
        </el-form-item>

        <!-- Agent 配置分组 -->
        <div class="agent-config-divider">
          <span class="divider-line"></span>
          <span class="divider-text">Agent 配置</span>
          <span class="divider-line"></span>
        </div>

        <el-form-item label="Agent 端口" prop="agent_port">
          <el-input-number
            v-model="form.agent_port"
            :min="1"
            :max="65535"
            :controls="false"
            placeholder="请输入 Agent 端口"
            style="width: 100%"
          />
        </el-form-item>

        <el-form-item label="Agent Token" prop="agent_token">
          <el-input
            v-model="form.agent_token"
            type="password"
            show-password
            placeholder="留空则不鉴权"
            maxlength="255"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="handleSubmit">
            确认
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Monitor, Edit, Delete, CircleCheck, Connection, InfoFilled, CircleCheckFilled, WarningFilled } from '@element-plus/icons-vue'
import { slaveApi } from '@/api/slave'
import { formatDateTimeInShanghai, formatRelativeTimeInShanghai } from '@/utils/datetime'

// 自动刷新定时器
let heartbeatTimer = null

// 数据状态
const slaveList = ref([])
const loading = ref(false)
const checkingId = ref(null)
const checkingAll = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const currentId = ref(null)

// 资源详情弹窗状态
const statsDialogVisible = ref(false)
const currentSlaveStats = ref(null)
const currentSlaveUptime = ref(0)

// Master 配置状态
const masterHostname = ref('')
const networkInterfaces = ref([])

// 心跳状态
const lastHeartbeatTime = ref('')
const masterNode = ref({})

// 表单
const formRef = ref(null)
const form = reactive({
  name: '',
  host: '',
  port: 1099,
  agent_port: 8089,
  agent_token: ''
})

// 表单校验规则
const rules = {
  name: [
    { required: true, message: '请输入节点名称', trigger: 'blur' },
    { min: 1, max: 50, message: '长度在1-50个字符', trigger: 'blur' }
  ],
  host: [
    { required: true, message: '请输入主机地址', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9.-]+$/, message: '请输入有效的主机地址', trigger: 'blur' }
  ],
  port: [
    { required: true, message: '请输入端口', trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: '端口范围1-65535', trigger: 'blur' }
  ]
}

// 获取状态类型
const getStatusType = (status) => {
  const map = {
    'online': 'success',
    'offline': 'danger',
    'unknown': 'info'
  }
  return map[status] || 'info'
}

// 获取状态文本
const getStatusText = (status) => {
  const map = {
    'online': '在线',
    'offline': '离线',
    'unknown': '未检测'
  }
  return map[status] || '未检测'
}

// 获取 Agent 状态文本
const getAgentStatusText = (status) => {
  const map = {
    'online': '在线',
    'offline': '离线',
    'unknown': '未知'
  }
  return map[status] || '未知'
}

// 获取错误类型文本
const getErrorText = (errorType) => {
  const map = {
    'connection_refused': '端口未监听',
    'timeout': '连接超时',
    'auth_failed': '认证失败',
    'unknown': '未知错误'
  }
  return map[errorType] || errorType
}

// 格式化时间
const formatTime = (time) => {
  return formatDateTimeInShanghai(time)
}

// 格式化相对时间（如"30秒前"、"2分钟前"）
const formatRelativeTime = (time) => {
  return formatRelativeTimeInShanghai(time)
}

// 解析 system_stats
const parseSystemStats = (slave) => {
  if (slave.system_stats && typeof slave.system_stats === 'string') {
    try {
      return JSON.parse(slave.system_stats)
    } catch { return null }
  }
  return slave.system_stats || null
}

const parseEnvironmentInfo = (node) => {
  if (!node) return null
  if (node.environment_info && typeof node.environment_info === 'string') {
    try {
      return node.environment_info ? JSON.parse(node.environment_info) : null
    } catch {
      return null
    }
  }
  return node.environment_info || null
}

const normalizeNodeRecord = (node) => ({
  ...node,
  parsedStats: parseSystemStats(node),
  parsedEnvironment: parseEnvironmentInfo(node)
})

const applyNodeSnapshot = (target, snapshot = {}) => {
  if (!target) return
  if (snapshot.status) target.status = snapshot.status
  if (snapshot.last_check_time) target.last_check_time = snapshot.last_check_time
  if (snapshot.agent_status) target.agent_status = snapshot.agent_status
  if (snapshot.agent_check_time) target.agent_check_time = snapshot.agent_check_time
  if (Object.prototype.hasOwnProperty.call(snapshot, 'system_stats')) {
    target.system_stats = snapshot.system_stats
    target.parsedStats = parseSystemStats(snapshot)
  }
  if (Object.prototype.hasOwnProperty.call(snapshot, 'environment_info')) {
    target.environment_info = snapshot.environment_info
    target.parsedEnvironment = parseEnvironmentInfo(snapshot)
  }
  if (Object.prototype.hasOwnProperty.call(snapshot, 'agent_uptime')) {
    target.agent_uptime = snapshot.agent_uptime
  }
}

const getEnvironmentWarnings = (node) => {
  const warnings = node?.parsedEnvironment?.warnings
  return Array.isArray(warnings) ? warnings.filter(Boolean) : []
}

const getEnvironmentVersion = (node) => {
  return node?.parsedEnvironment?.jmeter_version || ''
}

const getPluginCountLabel = (node) => {
  const count = node?.parsedEnvironment?.plugin_jars?.length || 0
  return count ? `插件 ${count} 个` : '插件未采集'
}

const getPropertiesStateLabel = (node) => {
  return node?.parsedEnvironment?.properties_fingerprint ? '配置已采集' : '配置未采集'
}

const getEnvironmentTone = (node) => {
  if (!node?.parsedEnvironment) return 'neutral'
  if (!getEnvironmentVersion(node)) return 'danger'
  if (getEnvironmentWarnings(node).length) return 'warning'
  return 'success'
}

const getEnvironmentStatusTag = (node) => {
  if (!node?.parsedEnvironment) return '未检测'
  if (!getEnvironmentVersion(node)) return '未安装 JMeter'
  if (getEnvironmentWarnings(node).length) return '需关注'
  return '环境正常'
}

const getEnvironmentHeadline = (node) => {
  if (!node?.parsedEnvironment) return '尚未采集到 Master 环境报告'
  if (!getEnvironmentVersion(node)) return '当前 Master 未检测到可执行的 JMeter 环境'
  return `当前 Master 已检测到 ${getEnvironmentVersion(node)}`
}

const getEnvironmentDescription = (node) => {
  if (!node?.parsedEnvironment) return '等待心跳刷新或页面初始化后自动采集。'
  const warnings = getEnvironmentWarnings(node)
  if (warnings.length) return warnings[0]
  return 'JMeter、插件指纹和 properties 指纹已采集，可作为分布式执行时的基准节点。'
}

const getStatPercent = (node, key) => {
  const value = Number(node?.parsedStats?.[key]?.percent || 0)
  return Number.isFinite(value) ? value : 0
}

const getConnectionCount = (node) => {
  const value = Number(node?.parsedStats?.network?.connections || 0)
  return Number.isFinite(value) ? value : 0
}

const getNodeConflictDetails = (node) => {
  const details = []
  const env = node?.parsedEnvironment
  const masterEnv = masterNode.value?.parsedEnvironment

  if (node?.status !== 'online') {
    details.push('JMeter RMI 不可达')
  }
  if ((node?.agent_status || 'unknown') !== 'online') {
    details.push('Agent 离线或未响应')
  }
  if (!env) {
    details.push('未采集环境报告')
  } else {
    if (!getEnvironmentVersion(node)) {
      details.push('未检测到 JMeter 版本')
    }
    if (masterEnv?.jmeter_version && env?.jmeter_version && env.jmeter_version !== masterEnv.jmeter_version) {
      details.push(`JMeter 版本与 Master 不一致 (${env.jmeter_version})`)
    }
    if (masterEnv?.plugin_fingerprint && env?.plugin_fingerprint && env.plugin_fingerprint !== masterEnv.plugin_fingerprint) {
      details.push('插件清单与 Master 不一致')
    }
    if (masterEnv?.properties_fingerprint && env?.properties_fingerprint && env.properties_fingerprint !== masterEnv.properties_fingerprint) {
      details.push('properties 指纹与 Master 不一致')
    }
  }

  getEnvironmentWarnings(node).forEach((warning) => details.push(warning))
  return [...new Set(details.filter(Boolean))]
}

const getNodeConflictTone = (node) => {
  const details = getNodeConflictDetails(node)
  if (!details.length) return 'success'
  if (node?.status !== 'online' || (node?.agent_status || 'unknown') !== 'online') return 'danger'
  if (details.some(item => item.includes('不一致') || item.includes('未检测到'))) return 'warning'
  return 'warning'
}

const getNodeConflictSummary = (node) => {
  const details = getNodeConflictDetails(node)
  if (!details.length) return '环境正常，可参与执行'
  return details[0]
}

const masterPressureWarnings = computed(() => {
  const warnings = []
  const cpu = getStatPercent(masterNode.value, 'cpu')
  const memory = getStatPercent(masterNode.value, 'memory')
  const disk = getStatPercent(masterNode.value, 'disk')
  const connections = getConnectionCount(masterNode.value)

  if (!masterHostname.value) {
    warnings.push({
      tone: 'warning',
      label: '回调链路',
      title: 'Master 回调地址仍未确认',
      detail: '多网卡环境下 Slave 可能无法把结果回传给 Master。'
    })
  }
  if (cpu >= 80) {
    warnings.push({
      tone: 'danger',
      label: 'Master 负载',
      title: `CPU 已到 ${formatPercent(cpu)}`,
      detail: 'Master 本机负载较高，继续让它参与施压可能影响结果稳定性。'
    })
  }
  if (memory >= 85) {
    warnings.push({
      tone: 'danger',
      label: 'Master 内存',
      title: `内存占用 ${formatPercent(memory)}`,
      detail: '建议先释放 Master 内存或改为仅调度，不要让它直接参与压测。'
    })
  }
  if (disk >= 90) {
    warnings.push({
      tone: 'warning',
      label: 'Master 磁盘',
      title: `磁盘占用 ${formatPercent(disk)}`,
      detail: '结果文件和日志写入可能受影响，建议先检查可用磁盘空间。'
    })
  }
  if (connections >= 3000) {
    warnings.push({
      tone: 'warning',
      label: 'Master 连接',
      title: `连接数 ${formatCount(connections)}`,
      detail: '当前连接数较高，若 Master 同时施压与回传，链路抖动概率会增加。'
    })
  }
  return warnings
})

const conflictNodeCount = computed(() => {
  return slaveList.value.filter(node => getNodeConflictDetails(node).length > 0).length
})

const clusterAdvisories = computed(() => {
  const advices = [...masterPressureWarnings.value]
  const offlineNodes = slaveList.value.filter(node => node.status !== 'online')
  const agentOfflineNodes = slaveList.value.filter(node => (node.agent_status || 'unknown') !== 'online')
  const envConflictNodes = slaveList.value.filter(node => {
    const details = getNodeConflictDetails(node)
    return details.some(item => item.includes('不一致') || item.includes('未采集环境报告') || item.includes('未检测到 JMeter 版本'))
  })

  if (offlineNodes.length) {
    advices.push({
      tone: 'warning',
      label: '节点连通',
      title: `${offlineNodes.length} 台节点 JMeter 不在线`,
      detail: '这些节点当前无法参与分布式调度，建议先重新检测连通性。'
    })
  }

  if (agentOfflineNodes.length) {
    advices.push({
      tone: 'warning',
      label: 'Agent 回传',
      title: `${agentOfflineNodes.length} 台节点 Agent 不在线`,
      detail: '没有 Agent 的节点无法稳定回传资源监控和环境信息。'
    })
  }

  if (envConflictNodes.length) {
    advices.push({
      tone: 'warning',
      label: '环境冲突',
      title: `${envConflictNodes.length} 台节点与 Master 存在差异`,
      detail: '版本、插件或 properties 指纹不一致，分布式执行结果可能不可比。'
    })
  }

  if (!advices.length) {
    advices.push({
      tone: 'success',
      label: '当前集群',
      title: 'Master 与节点状态稳定',
      detail: '回调地址、资源负载和环境检测均处于可执行状态，可以直接开始调度。'
    })
  }

  return advices
})

// 获取资源颜色（根据阈值）
const getResourceColor = (percent, threshold) => {
  if (percent >= threshold + 10) return '#F56C6C'  // 红色（超严重）
  if (percent >= threshold) return '#E6A23C'        // 橙色（警告）
  if (percent >= threshold - 20) return '#409EFF'   // 蓝色（正常偏高）
  return '#67C23A'                                   // 绿色（健康）
}

// 按资源排序
const sortByStats = (a, b, type) => {
  const aVal = a.parsedStats?.[type]?.percent || 0
  const bVal = b.parsedStats?.[type]?.percent || 0
  return aVal - bVal
}

// 格式化 MB 为 GB/MB
const formatMB = (mb) => {
  if (!mb) return '--'
  if (mb >= 1024) return (mb / 1024).toFixed(1) + ' GB'
  return mb + ' MB'
}

const formatPercent = (value) => {
  const num = Number(value)
  if (!Number.isFinite(num)) return '--'
  return `${num.toFixed(0)}%`
}

const formatCount = (value) => {
  const num = Number(value)
  if (!Number.isFinite(num)) return '--'
  return num.toLocaleString('zh-CN')
}

// 格式化运行时长
const formatUptime = (seconds) => {
  if (!seconds) return '--'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (h > 24) {
    const d = Math.floor(h / 24)
    return `${d}天${h % 24}小时`
  }
  return h > 0 ? `${h}小时${m}分钟` : `${m}分钟`
}

const masterCallbackBaseURL = computed(() => {
  if (!masterHostname.value || typeof window === 'undefined') return ''
  try {
    const baseURL = new URL(window.location.origin)
    baseURL.hostname = masterHostname.value
    return baseURL.origin
  } catch {
    return ''
  }
})

const onlineSlaveCount = computed(() => slaveList.value.filter(item => item.status === 'online').length)

const agentOnlineCount = computed(() => slaveList.value.filter(item => item.agent_status === 'online').length)

const averageCpuText = computed(() => {
  const values = slaveList.value
    .map(item => Number(item.parsedStats?.cpu?.percent))
    .filter(value => Number.isFinite(value))
  if (!values.length) return '--'
  const avg = values.reduce((sum, value) => sum + value, 0) / values.length
  return `${avg.toFixed(0)}%`
})

const busySlaveCount = computed(() => {
  return slaveList.value.filter(item => {
    const stats = item.parsedStats || {}
    return Number(stats.cpu?.percent || 0) >= 80
      || Number(stats.memory?.percent || 0) >= 85
      || Number(stats.disk?.percent || 0) >= 90
  }).length
})

// 显示资源详情弹窗
const showStatsDialog = (row) => {
  currentSlaveStats.value = row.parsedStats
  currentSlaveUptime.value = row.agent_uptime || 0
  statsDialogVisible.value = true
}

// 加载Slave列表
const loadSlaves = async () => {
  loading.value = true
  try {
    const res = await slaveApi.getList()
    const data = res.data?.list || res.data || []
    slaveList.value = data.map(normalizeNodeRecord)
  } catch (error) {
    console.error('加载Slave列表失败:', error)
    ElMessage.error('加载Slave列表失败')
  } finally {
    loading.value = false
  }
}

// 添加
const handleAdd = () => {
  isEdit.value = false
  currentId.value = null
  resetForm()
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  form.name = row.name
  form.host = row.host
  form.port = row.port
  form.agent_port = row.agent_port || 8089
  form.agent_token = row.agent_token || ''
  dialogVisible.value = true
}

// 删除
const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除Slave节点 "${row.name}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await slaveApi.delete(row.id)
    ElMessage.success('删除成功')
    loadSlaves()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除失败:', error)
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  }
}

// 检测连通性
const handleCheck = async (row) => {
  checkingId.value = row.id
  try {
    const res = await slaveApi.checkConnectivity(row.id)
    // 兼容处理：优先用 status，fallback 到 online
    const status = res.data?.status || (res.data?.online ? 'online' : 'offline')
    const agentStatus = res.data?.agent_status || 'unknown'
    const agentCheckTime = res.data?.agent_check_time || row.agent_check_time
    const diagnostic = res.data?.diagnostic

    // 更新列表中的状态
    const index = slaveList.value.findIndex(s => s.id === row.id)
    if (index !== -1) {
      applyNodeSnapshot(slaveList.value[index], {
        ...res.data,
        status,
        agent_status: agentStatus,
        agent_check_time: agentCheckTime
      })
      slaveList.value[index].diagnostic = diagnostic
    }

    if (status === 'online' && agentStatus === 'online') {
      ElMessage.success(`${row.host}: JMeter 在线, Agent 在线`)
    } else if (status === 'online') {
      ElMessage.warning(`${row.host}: JMeter 在线, Agent 离线`)
    } else {
      ElMessage.error(`${row.host}: JMeter 离线`)
    }
  } catch (error) {
    console.error('检测失败:', error)
    ElMessage.error(`检测 ${row.host} 失败`)
    // 更新为离线状态
    const index = slaveList.value.findIndex(s => s.id === row.id)
    if (index !== -1) {
      slaveList.value[index].status = 'offline'
      slaveList.value[index].agent_status = 'unknown'
      slaveList.value[index].diagnostic = null
    }
  } finally {
    checkingId.value = null
  }
}

// 全部检测
const handleCheckAll = async () => {
  if (slaveList.value.length === 0) {
    ElMessage.warning('暂无节点可检测')
    return
  }

  checkingAll.value = true
  ElMessage.info('开始检测所有节点...')

  const results = { online: 0, offline: 0, failed: 0 }

  for (const row of slaveList.value) {
    try {
      const res = await slaveApi.checkConnectivity(row.id)
      const status = res.data?.status || (res.data?.online ? 'online' : 'offline')
      const agentStatus = res.data?.agent_status || 'unknown'
      const agentCheckTime = res.data?.agent_check_time || row.agent_check_time

      // 更新列表中的状态
      const index = slaveList.value.findIndex(s => s.id === row.id)
      if (index !== -1) {
        applyNodeSnapshot(slaveList.value[index], {
          ...res.data,
          status,
          agent_status: agentStatus,
          agent_check_time: agentCheckTime
        })
      }

      if (status === 'online') {
        results.online++
      } else {
        results.offline++
      }
    } catch (error) {
      console.error(`检测 ${row.host} 失败:`, error)
      results.failed++
      // 更新为离线状态
      const index = slaveList.value.findIndex(s => s.id === row.id)
      if (index !== -1) {
        slaveList.value[index].status = 'offline'
        slaveList.value[index].agent_status = 'unknown'
      }
    }
  }

  checkingAll.value = false

  let message = `检测完成：${results.online} 个在线`
  if (results.offline > 0) {
    message += `，${results.offline} 个离线`
  }
  if (results.failed > 0) {
    message += `，${results.failed} 个检测失败`
  }

  if (results.online === slaveList.value.length) {
    ElMessage.success(message)
  } else {
    ElMessage.warning(message)
  }
}

// 重置表单
const resetForm = () => {
  if (formRef.value) {
    formRef.value.resetFields()
  }
  form.name = ''
  form.host = ''
  form.port = 1099
  form.agent_port = 8089
  form.agent_token = ''
}

// 提交表单
const handleSubmit = async () => {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await slaveApi.update(currentId.value, form)
      ElMessage.success('更新成功')
    } else {
      await slaveApi.create(form)
      ElMessage.success('添加成功')
    }
    dialogVisible.value = false
    loadSlaves()
  } catch (error) {
    console.error('提交失败:', error)
    ElMessage.error(error.response?.data?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// 加载 Master 配置
const loadMasterConfig = async () => {
  try {
    const [ifacesRes, hostnameRes] = await Promise.all([
      slaveApi.getNetworkInterfaces(),
      slaveApi.getMasterHostname()
    ])
    networkInterfaces.value = ifacesRes.data || []
    masterHostname.value = hostnameRes.data?.master_hostname || ''

    // 如果还没配置，自动选择第一个检测到的 IP
    if (!masterHostname.value && networkInterfaces.value.length > 0) {
      masterHostname.value = networkInterfaces.value[0].ip
      // 自动保存
      await slaveApi.updateMasterHostname(masterHostname.value)
      ElMessage.success('已自动配置 Master IP')
    }
  } catch (err) {
    console.error('加载 Master 配置失败:', err)
    ElMessage.error('加载 Master 配置失败')
  }
}

// 处理 Master IP 变更
const handleMasterIPChange = async (val) => {
  try {
    await slaveApi.updateMasterHostname(val)
    ElMessage.success('Master IP 已更新')
  } catch (err) {
    ElMessage.error('保存失败')
  }
}

// 刷新心跳状态（只更新状态和最后检测时间，不刷新整个列表）
const refreshHeartbeatStatus = async () => {
  try {
    const res = await slaveApi.getHeartbeatStatus()
    if (res.data?.master) {
      masterNode.value = normalizeNodeRecord(res.data.master)
    }
    if (res.data && res.data.slaves) {
      const heartbeatData = res.data.slaves
      // 更新每个 slave 的状态和最后检测时间
      heartbeatData.forEach(hb => {
        const index = slaveList.value.findIndex(s => s.id === hb.id)
        if (index !== -1) {
          applyNodeSnapshot(slaveList.value[index], hb)
        }
      })
      // 更新最后检测时间显示
      if (res.data.last_check_time) {
        lastHeartbeatTime.value = formatTime(res.data.last_check_time)
      }
    }
  } catch (error) {
    console.error('刷新心跳状态失败:', error)
  }
}

// 启动自动刷新
const startAutoRefresh = () => {
  // 立即刷新一次
  refreshHeartbeatStatus()
  // 每10秒刷新一次
  heartbeatTimer = setInterval(() => {
    refreshHeartbeatStatus()
  }, 10000)
}

// 停止自动刷新
const stopAutoRefresh = () => {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer)
    heartbeatTimer = null
  }
}

// 初始化
onMounted(() => {
  loadSlaves()
  loadMasterConfig()
  startAutoRefresh()
})

// 页面销毁时清除定时器
onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped lang="scss">
.slave-page {
  padding: 6px 0 14px;
}

.page-header-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(148, 163, 184, 0.12);
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.12), transparent 32%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.015)),
    var(--bg-panel);
  box-shadow: 0 22px 48px rgba(2, 8, 23, 0.12);
}

.page-header-bar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  flex-wrap: wrap;
}

.page-header-copy {
  min-width: 0;

  h1 {
    margin: 2px 0 6px;
    color: var(--text-primary);
    font-size: 24px;
    line-height: 1.15;
  }

  p {
    max-width: 640px;
    color: var(--text-secondary);
    font-size: 12px;
    line-height: 1.5;
  }
}

.page-header-kicker {
  color: var(--accent-blue);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.page-header-pills {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.workspace-pill {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(148, 163, 184, 0.12);
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 600;
}

// Master 配置卡片
.master-config-card {
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.1), transparent 34%),
    linear-gradient(180deg, rgba(56, 189, 248, 0.05), rgba(255, 255, 255, 0.015)),
    var(--bg-panel);
  border-radius: var(--radius-lg);
  border: 1px solid rgba(148, 163, 184, 0.14);
  padding: 14px 16px;
  margin-bottom: 12px;
  box-shadow:
    0 20px 44px rgba(2, 8, 23, 0.14),
    inset 0 1px 0 rgba(255, 255, 255, 0.03);

  .heartbeat-status-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    font-size: 12px;

    .heartbeat-indicator {
      display: flex;
      align-items: center;
      gap: 8px;

      .pulse-dot {
        width: 8px;
        height: 8px;
        background: var(--accent-green);
        border-radius: 50%;
        animation: pulse 2s infinite;
      }

      .heartbeat-text {
        color: var(--accent-green);
        font-weight: 500;
      }
    }

    .heartbeat-divider {
      color: var(--text-secondary);
      opacity: 0.5;
    }

    .heartbeat-info {
      color: var(--text-secondary);
    }
  }

  @keyframes pulse {
    0% {
      transform: scale(0.95);
      box-shadow: 0 0 0 0 rgba(74, 222, 128, 0.7);
    }
    70% {
      transform: scale(1);
      box-shadow: 0 0 0 6px rgba(74, 222, 128, 0);
    }
    100% {
      transform: scale(0.95);
      box-shadow: 0 0 0 0 rgba(74, 222, 128, 0);
    }
  }

  .master-config-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
    margin-bottom: 10px;

    .master-config-title {
      display: flex;
      align-items: center;
      gap: 10px;
      font-size: 15px;
      font-weight: 600;
      color: var(--text-primary);

      .title-icon {
        font-size: 18px;
        color: var(--accent-blue);
      }
    }

    .info-icon {
      font-size: 16px;
      color: var(--text-secondary);
      cursor: help;
      transition: color 0.2s;

      &:hover {
        color: var(--accent-blue);
      }
    }
  }

  .master-config-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .master-config-body {
    display: flex;
    align-items: center;
    gap: 14px;
    flex-wrap: wrap;

    .config-item {
      display: flex;
      align-items: center;
      gap: 12px;
      flex: 1;
      min-width: 300px;

      .config-label {
        font-size: 13px;
        color: var(--text-secondary);
        white-space: nowrap;
      }

      .master-ip-select {
        flex: 1;
        max-width: 320px;
      }
    }

    .config-status {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 12px;

      .status-icon {
        font-size: 14px;

        &.success {
          color: var(--accent-green);
        }

        &.warning {
          color: var(--accent-orange);
        }
      }

      .status-text {
        &.success {
          color: var(--accent-green);
        }

        &.warning {
          color: var(--accent-orange);
        }
      }
    }
  }

  .master-dashboard {
    display: grid;
    grid-template-columns: minmax(0, 1.35fr) minmax(320px, 0.95fr);
    gap: 10px;
    margin-top: 10px;
  }

  .master-primary-panel {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .master-overview-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
  }

  .master-overview-card {
    min-height: 62px;
    padding: 10px 12px;
    border-radius: 14px;
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
      rgba(15, 23, 42, 0.62);
    border: 1px solid rgba(148, 163, 184, 0.12);
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 8px;

    &.is-callback {
      background:
        radial-gradient(circle at top left, rgba(56, 189, 248, 0.12), transparent 42%),
        rgba(15, 23, 42, 0.62);
    }

    .overview-card-label {
      color: var(--text-secondary);
      font-size: 11px;
      letter-spacing: 0.04em;
      text-transform: uppercase;
    }

    .overview-card-value {
      color: var(--text-primary);
      font-size: 18px;
      font-weight: 700;
      font-family: 'Consolas', 'Monaco', monospace;
    }

    .overview-card-code {
      color: var(--text-primary);
      font-size: 12px;
      line-height: 1.45;
      word-break: break-all;
    }
  }

  .master-env-panel {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 12px 14px;
    border-radius: 14px;
    border: 1px solid rgba(148, 163, 184, 0.12);
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.026), rgba(255, 255, 255, 0.012)),
      rgba(15, 23, 42, 0.56);

    &.is-warning {
      border-color: rgba(245, 158, 11, 0.24);
      background: linear-gradient(135deg, rgba(245, 158, 11, 0.08), rgba(15, 23, 42, 0.56));
    }

    &.is-danger {
      border-color: rgba(239, 68, 68, 0.24);
      background: linear-gradient(135deg, rgba(239, 68, 68, 0.08), rgba(15, 23, 42, 0.56));
    }
  }

  .master-env-summary {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
  }

  .master-env-copy {
    min-width: 0;

    strong {
      display: block;
      margin-top: 3px;
      color: var(--text-primary);
      font-size: 16px;
    }

    p {
      margin-top: 4px;
      color: var(--text-secondary);
      font-size: 12px;
      line-height: 1.5;
    }
  }

  .master-env-label {
    color: var(--accent-blue);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .master-env-badges {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 8px;
  }

  .master-env-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
  }

  .master-env-card {
    min-height: 68px;
    padding: 10px 12px;
    border-radius: 12px;
    background: rgba(15, 23, 42, 0.52);
    border: 1px solid rgba(148, 163, 184, 0.12);
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 8px;

    span {
      color: var(--text-secondary);
      font-size: 11px;
    }

    strong {
      color: var(--text-primary);
      font-size: 16px;
      line-height: 1.35;
      word-break: break-word;
    }
  }

  .master-env-warning-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .master-env-warning {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 8px 10px;
    border-radius: 10px;
    background: rgba(245, 158, 11, 0.08);
    border: 1px solid rgba(245, 158, 11, 0.16);
    color: #f8d27a;
    font-size: 11px;
    line-height: 1.45;
  }

  .cluster-alert-strip {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    margin-top: 10px;
  }

  .cluster-alert-card {
    min-height: 72px;
    padding: 10px 12px;
    border-radius: 12px;
    border: 1px solid rgba(148, 163, 184, 0.12);
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.024), rgba(255, 255, 255, 0.012)),
      rgba(15, 23, 42, 0.56);
    display: flex;
    flex-direction: column;
    gap: 4px;

    &.is-success {
      border-color: rgba(74, 222, 128, 0.18);
      background: linear-gradient(135deg, rgba(74, 222, 128, 0.07), rgba(15, 23, 42, 0.56));
    }

    &.is-warning {
      border-color: rgba(245, 158, 11, 0.2);
      background: linear-gradient(135deg, rgba(245, 158, 11, 0.08), rgba(15, 23, 42, 0.56));
    }

    &.is-danger {
      border-color: rgba(239, 68, 68, 0.2);
      background: linear-gradient(135deg, rgba(239, 68, 68, 0.08), rgba(15, 23, 42, 0.56));
    }

    .cluster-alert-label {
      color: var(--accent-blue);
      font-size: 11px;
      font-weight: 700;
      letter-spacing: 0.14em;
      text-transform: uppercase;
    }

    strong {
      color: var(--text-primary);
      font-size: 14px;
      line-height: 1.35;
    }

    p {
      color: var(--text-secondary);
      font-size: 11px;
      line-height: 1.45;
      display: -webkit-box;
      -webkit-box-orient: vertical;
      -webkit-line-clamp: 2;
      overflow: hidden;
    }
  }
}

// 区域卡片
.section-card {
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.08), transparent 34%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.015)),
    var(--bg-panel);
  border-radius: var(--radius-lg);
  border: 1px solid rgba(148, 163, 184, 0.12);
  padding: 16px;
  margin-bottom: 12px;
  box-shadow:
    0 22px 48px rgba(2, 8, 23, 0.12),
    inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.env-badge,
.env-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(148, 163, 184, 0.12);
  color: var(--text-secondary);
  font-size: 12px;
  white-space: nowrap;

  &.is-success {
    color: #4ade80;
    border-color: rgba(74, 222, 128, 0.22);
    background: rgba(74, 222, 128, 0.08);
  }

  &.is-warning {
    color: #f8d27a;
    border-color: rgba(245, 158, 11, 0.2);
    background: rgba(245, 158, 11, 0.08);
  }

  &.is-danger {
    color: #ff8e87;
    border-color: rgba(239, 68, 68, 0.2);
    background: rgba(239, 68, 68, 0.08);
  }

  &.subtle {
    color: var(--text-secondary);
    background: rgba(255, 255, 255, 0.035);
  }
}

.environment-cell {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.environment-cell-top,
.environment-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.environment-meta {
  color: var(--text-secondary);
  font-size: 12px;
}

.conflict-text {
  font-weight: 600;

  &.is-warning {
    color: #f8d27a;
  }

  &.is-danger {
    color: #ff8e87;
  }

  &.is-success {
    color: #4ade80;
  }
}

.environment-warning-cell {
  color: #f8d27a;
  font-size: 12px;
  line-height: 1.6;

  &.is-empty {
    color: var(--text-secondary);
  }
}

// 区域标签
.section-label {
  color: var(--accent-blue);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 1px;
  text-transform: uppercase;
  margin-bottom: 4px;
}

.section-title {
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 4px;
}

.section-desc {
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.section-header-with-action {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 18px;

  .section-header {
    flex: 1;
    min-width: 0;
  }

  .section-actions {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: flex-end;
    flex-wrap: wrap;

    .btn-icon {
      margin-right: 6px;
    }
  }
}

.list-utility-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.utility-chip {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(148, 163, 184, 0.12);
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 600;
}

// 节点表格
.table-shell {
  overflow-x: auto;
  overflow-y: hidden;
  padding: 6px;
  border-radius: 16px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.01)),
    rgba(15, 23, 42, 0.52);
  border: 1px solid rgba(148, 163, 184, 0.08);
}

.slaves-table {
  background: transparent;
  border-radius: var(--radius-lg);
  overflow: hidden;
  min-width: 100%;
  border: none;

  :deep(.el-table__header-wrapper) {
    th.el-table__cell {
      background:
        linear-gradient(180deg, rgba(255, 255, 255, 0.045), rgba(255, 255, 255, 0.018)) !important;
      color: var(--text-secondary) !important;
      font-weight: 500 !important;
      font-size: 13px !important;
      border-bottom: 1px solid rgba(255, 255, 255, 0.06) !important;
      height: 50px;
    }
  }

  :deep(.el-table__body-wrapper) {
    background-color: var(--bg-card);

    td.el-table__cell {
      border-bottom: 1px solid rgba(255, 255, 255, 0.04) !important;
      color: var(--text-primary) !important;
      padding-top: 11px;
      padding-bottom: 11px;
    }
  }

  :deep(.el-table__row) {
    background-color: var(--bg-card);

    &:hover {
      background:
        linear-gradient(90deg, rgba(56, 189, 248, 0.03), rgba(255, 255, 255, 0.015)) !important;
    }
  }

  .node-name-cell {
    display: flex;
    align-items: center;
    gap: 10px;

    .node-name-stack {
      min-width: 0;
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .node-icon {
      font-size: 18px;
      color: var(--accent-blue);
    }

    .node-name {
      color: var(--text-primary);
      font-weight: 500;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .node-address-stack {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .node-subtext {
    color: var(--text-secondary);
    font-size: 12px;
    line-height: 1.4;
  }

  .address-code {
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    font-size: 13px;
    color: var(--text-secondary);
    background: var(--bg-secondary);
    padding: 2px 8px;
    border-radius: var(--radius-sm);
  }

  .port-text {
    color: var(--text-primary);
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  }

  .time-text {
    color: var(--text-secondary);
    font-size: 13px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 28px;
    padding: 0 10px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.045);
    border: 1px solid rgba(148, 163, 184, 0.08);
  }

  .status-tag {
    font-weight: 500;
  }

  .status-cell {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 6px;

    .status-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;

      &.online {
        background: var(--accent-green);
        box-shadow: 0 0 0 2px rgba(74, 222, 128, 0.2);
      }

      &.offline {
        background: var(--accent-red);
        box-shadow: 0 0 0 2px rgba(248, 113, 113, 0.2);
      }

      &.unknown {
        background: var(--text-secondary);
      }
    }

    .status-label {
      font-size: 13px;
      font-weight: 500;

      &.online {
        color: var(--accent-green);
      }

      &.offline {
        color: var(--accent-red);
      }

      &.unknown {
        color: var(--text-secondary);
      }
    }
  }

  .agent-status-cell {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;

    .status-row {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 6px;
    }

    .status-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;

      &.online {
        background: var(--accent-green);
        box-shadow: 0 0 0 2px rgba(74, 222, 128, 0.2);
      }

      &.offline {
        background: var(--accent-red);
        box-shadow: 0 0 0 2px rgba(248, 113, 113, 0.2);
      }

      &.unknown {
        background: var(--text-secondary);
      }
    }

    .status-label {
      font-size: 13px;
      font-weight: 500;

      &.online {
        color: var(--accent-green);
      }

      &.offline {
        color: var(--accent-red);
      }

      &.unknown {
        color: var(--text-secondary);
      }
    }

    .agent-check-time {
      font-size: 11px;
      color: var(--text-secondary);
      opacity: 0.7;
    }
  }

  .connectivity-cell {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
  }

  // 资源使用列样式
  .resource-cell {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    padding: 2px 0;
  }

  .resource-badge {
    display: inline-flex;
    align-items: center;
    min-height: 28px;
    padding: 0 10px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.045);
    border: 1px solid rgba(148, 163, 184, 0.1);
    color: var(--text-primary);
    font-size: 12px;
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  }

  .no-data {
    color: var(--text-secondary);
    font-size: 13px;
  }

  .action-btns {
    display: flex;
    gap: 4px;
    flex-wrap: nowrap;
    overflow: visible;
    justify-content: center;
    padding: 4px 6px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.025);
    border: 1px solid rgba(255, 255, 255, 0.05);

    .action-btn {
      padding: 0;
      font-size: 13px;
      min-width: auto;
      border-radius: 999px;

      .el-icon {
        font-size: 14px;
      }

      &.icon-btn {
        width: 30px;
        height: 30px;
        border-radius: 999px;
        background: rgba(255, 255, 255, 0.045);
        border: 1px solid rgba(255, 255, 255, 0.06);
        transition: transform 0.18s ease, border-color 0.18s ease, background-color 0.18s ease;

        .el-icon {
          margin-right: 0;
        }

        &:hover {
          transform: translateY(-1px);
        }
      }
    }

    .check-btn,
    .edit-btn {
      color: var(--accent-blue);
    }

    .check-btn,
    .edit-btn,
    .stats-btn {
      background: rgba(56, 189, 248, 0.06);
      border-color: rgba(56, 189, 248, 0.12);
    }

    .delete-btn {
      color: var(--accent-red) !important;
      background: rgba(248, 113, 113, 0.06);
      border-color: rgba(248, 113, 113, 0.12);
    }

    .delete-btn:hover {
      color: #ff5c52 !important;
    }

    .stats-btn {
      color: var(--text-secondary) !important;
    }

    .stats-btn:hover:not(:disabled) {
      color: var(--accent-blue) !important;
    }

    .stats-btn:disabled {
      opacity: 0.4;
    }
  }
}

// 资源详情弹窗样式
.stats-dialog {
  :deep(.el-descriptions__label) {
    background-color: rgba(255, 255, 255, 0.03) !important;
    color: var(--text-secondary) !important;
  }

  :deep(.el-descriptions__content) {
    background-color: var(--bg-card) !important;
    color: var(--text-primary) !important;
  }

  :deep(.el-descriptions__body) {
    background-color: transparent !important;
  }

  :deep(.el-descriptions__cell) {
    border-color: rgba(255, 255, 255, 0.06) !important;
  }

  .stats-content {
    padding: 8px 0;
  }
}

// 空状态
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 16px;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  margin-top: 14px;

  .empty-icon {
    width: 68px;
    height: 68px;
    border-radius: 50%;
    background: linear-gradient(135deg, rgba(0, 212, 255, 0.1), rgba(0, 102, 255, 0.1));
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 12px;

    .el-icon {
      font-size: 34px;
      color: var(--accent-blue);
      opacity: 0.6;
    }
  }

  .empty-title {
    font-size: 15px;
    font-weight: 500;
    color: var(--text-primary);
    margin-bottom: 8px;
  }

  .empty-desc {
    font-size: 13px;
    color: var(--text-secondary);
  }
}

// 弹窗表单
.slave-form {
  :deep(.el-form-item__label) {
    color: var(--text-primary);
  }

  :deep(.el-input-number .el-input__inner) {
    text-align: left;
  }

  .agent-config-divider {
    display: flex;
    align-items: center;
    gap: 12px;
    margin: 20px 0 16px 0;

    .divider-line {
      flex: 1;
      height: 1px;
      background: rgba(255, 255, 255, 0.1);
    }

    .divider-text {
      font-size: 13px;
      color: var(--text-secondary);
      font-weight: 500;
      white-space: nowrap;
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

// 延迟文本
.latency-text {
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0.7;
  margin-top: 2px;
}

// 诊断按钮
.diagnostic-btn {
  color: var(--accent-orange) !important;
}

@media (max-width: 1280px) {
  .page-header-pills {
    justify-content: flex-start;
  }

  .cluster-alert-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .nodes-summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .master-config-card {
    .master-dashboard {
      grid-template-columns: 1fr;
    }

    .master-overview-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .master-env-summary {
      flex-direction: column;
    }

    .master-env-badges {
      justify-content: flex-start;
    }

    .master-env-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
}

@media (max-width: 768px) {
  .page-header-bar {
    padding: 18px;
  }

  .page-header-copy h1 {
    font-size: 24px;
  }

  .cluster-alert-strip {
    grid-template-columns: 1fr;
  }

  .nodes-summary-grid {
    grid-template-columns: 1fr;
  }

  .master-config-card {
    .master-config-body {
      gap: 16px;

      .config-item {
        min-width: 100%;
      }
    }

    .master-overview-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .master-env-grid {
      grid-template-columns: 1fr;
    }
  }

  .section-card {
    padding: 18px;
  }

  .slaves-table {
    min-width: 980px;
  }
}

// 诊断面板样式
.diagnostic-panel {
  padding: 8px 4px;

  .diagnostic-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    font-weight: 600;
    color: var(--accent-orange);
    margin-bottom: 12px;
    padding-bottom: 8px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);

    .el-icon {
      font-size: 16px;
    }
  }

  .diagnostic-item {
    margin-bottom: 12px;

    .diagnostic-label {
      font-size: 12px;
      color: var(--text-secondary);
      margin-bottom: 4px;
    }

    .diagnostic-latency {
      font-size: 11px;
      color: var(--text-secondary);
      opacity: 0.7;
      margin-top: 4px;
    }
  }

  .diagnostic-suggestion {
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid rgba(255, 255, 255, 0.1);

    .suggestion-title {
      font-size: 12px;
      font-weight: 600;
      color: var(--accent-blue);
      margin-bottom: 6px;
    }

    .suggestion-content {
      font-size: 12px;
      color: var(--text-primary);
      line-height: 1.5;
    }
  }
}
</style>
