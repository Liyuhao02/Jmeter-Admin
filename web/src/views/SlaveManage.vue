<template>
  <div class="slave-page">
    <!-- Master 节点配置卡片 -->
    <div class="master-config-card">
      <div class="master-config-header">
        <div class="master-config-title">
          <el-icon class="title-icon"><Connection /></el-icon>
          <span>Master 节点配置</span>
        </div>
        <el-tooltip content="Slave 节点通过此 IP 将测试结果回传给 Master，多网卡环境需指定正确的 IP" placement="top">
          <el-icon class="info-icon"><InfoFilled /></el-icon>
        </el-tooltip>
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
          <div class="section-title">节点管理</div>
          <div class="section-desc">管理分布式执行的Slave节点</div>
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

      <!-- 节点表格 -->
      <el-table
        v-loading="loading"
        :data="slaveList"
        class="slaves-table"
        stripe
      >
        <el-table-column label="名称" min-width="140" sortable prop="name" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="node-name-cell">
              <el-icon class="node-icon"><Monitor /></el-icon>
              <span class="node-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="主机地址" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            <code class="address-code">{{ row.host }}</code>
          </template>
        </el-table-column>
        <el-table-column label="端口" width="100" align="center">
          <template #default="{ row }">
            <span class="port-text">{{ row.port }}</span>
          </template>
        </el-table-column>
        <el-table-column label="JMeter 状态" width="140" align="center" sortable prop="status">
          <template #default="{ row }">
            <div class="status-cell">
              <span class="status-dot" :class="row.status"></span>
              <span class="status-label" :class="row.status">{{ getStatusText(row.status) }}</span>
            </div>
            <div v-if="row.diagnostic && row.diagnostic.jmeter_latency_ms > 0" class="latency-text">
              {{ row.diagnostic.jmeter_latency_ms }}ms
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Agent 状态" width="160" align="center" sortable prop="agent_status">
          <template #default="{ row }">
            <div class="agent-status-cell">
              <div class="status-row">
                <span class="status-dot" :class="row.agent_status || 'unknown'"></span>
                <span class="status-label" :class="row.agent_status || 'unknown'">{{ getAgentStatusText(row.agent_status) }}</span>
              </div>
              <div v-if="row.diagnostic && row.diagnostic.agent_latency_ms > 0" class="latency-text">
                {{ row.diagnostic.agent_latency_ms }}ms
              </div>
              <div class="agent-check-time" v-else-if="row.agent_check_time">
                {{ formatRelativeTime(row.agent_check_time) }}
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="资源使用" width="180" sortable :sort-method="(a,b) => sortByStats(a,b,'cpu')">
          <template #default="{ row }">
            <div v-if="row.parsedStats" class="resource-cell">
              <div class="resource-row">
                <span class="resource-label">CPU</span>
                <el-progress
                  :percentage="row.parsedStats.cpu?.percent || 0"
                  :stroke-width="4"
                  :color="getResourceColor(row.parsedStats.cpu?.percent || 0, 80)"
                  :show-text="false"
                  class="resource-progress"
                />
                <span class="resource-value">{{ (row.parsedStats.cpu?.percent || 0).toFixed(0) }}%</span>
              </div>
              <div class="resource-row">
                <span class="resource-label">内存</span>
                <el-progress
                  :percentage="row.parsedStats.memory?.percent || 0"
                  :stroke-width="4"
                  :color="getResourceColor(row.parsedStats.memory?.percent || 0, 85)"
                  :show-text="false"
                  class="resource-progress"
                />
                <span class="resource-value">{{ (row.parsedStats.memory?.percent || 0).toFixed(0) }}%</span>
              </div>
              <div class="resource-row">
                <span class="resource-label">磁盘</span>
                <el-progress
                  :percentage="row.parsedStats.disk?.percent || 0"
                  :stroke-width="4"
                  :color="getResourceColor(row.parsedStats.disk?.percent || 0, 90)"
                  :show-text="false"
                  class="resource-progress"
                />
                <span class="resource-value">{{ (row.parsedStats.disk?.percent || 0).toFixed(0) }}%</span>
              </div>
            </div>
            <span v-else class="no-data">--</span>
          </template>
        </el-table-column>
        <el-table-column label="最后检测" width="120" align="center">
          <template #default="{ row }">
            <span class="time-text" :title="row.last_check_time">
              {{ formatRelativeTime(row.last_check_time) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <div class="action-btns">
              <el-tooltip content="检测连通性" placement="top">
                <el-button
                  link
                  type="primary"
                  @click="handleCheck(row)"
                  :loading="checkingId === row.id"
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
                  <el-button link type="warning" class="action-btn icon-btn diagnostic-btn">
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
                  class="action-btn icon-btn stats-btn"
                >
                  <el-icon><InfoFilled /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 空状态 -->
      <div v-if="!loading && slaveList.length === 0" class="empty-state">
        <div class="empty-icon">
          <el-icon><Monitor /></el-icon>
        </div>
        <h3 class="empty-title">暂无Slave节点</h3>
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
import { ref, reactive, onMounted, onUnmounted } from 'vue'
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
    // 解析 system_stats
    slaveList.value = data.map(s => ({
      ...s,
      parsedStats: parseSystemStats(s)
    }))
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
      slaveList.value[index].status = status
      slaveList.value[index].agent_status = agentStatus
      slaveList.value[index].agent_check_time = agentCheckTime
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
        slaveList.value[index].status = status
        slaveList.value[index].agent_status = agentStatus
        slaveList.value[index].agent_check_time = agentCheckTime
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
    if (res.data && res.data.slaves) {
      const heartbeatData = res.data.slaves
      // 更新每个 slave 的状态和最后检测时间
      heartbeatData.forEach(hb => {
        const index = slaveList.value.findIndex(s => s.id === hb.id)
        if (index !== -1) {
          slaveList.value[index].status = hb.status
          slaveList.value[index].last_check_time = hb.last_check_time
          slaveList.value[index].agent_status = hb.agent_status
          slaveList.value[index].agent_check_time = hb.agent_check_time
          slaveList.value[index].system_stats = hb.system_stats
          slaveList.value[index].agent_uptime = hb.agent_uptime
          // 同步解析 system_stats
          slaveList.value[index].parsedStats = parseSystemStats(hb)
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
  padding: 20px;
}

// Master 配置卡片
.master-config-card {
  background: rgba(255, 255, 255, 0.03);
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  padding: 20px 24px;
  margin-bottom: 20px;

  .heartbeat-status-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    font-size: 13px;

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
    margin-bottom: 16px;

    .master-config-title {
      display: flex;
      align-items: center;
      gap: 10px;
      font-size: 16px;
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

  .master-config-body {
    display: flex;
    align-items: center;
    gap: 24px;
    flex-wrap: wrap;

    .config-item {
      display: flex;
      align-items: center;
      gap: 12px;
      flex: 1;
      min-width: 300px;

      .config-label {
        font-size: 14px;
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
      font-size: 13px;

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
}

// 区域卡片
.section-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid rgba(255, 255, 255, 0.06);
  padding: 24px;
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
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 4px;
}

.section-desc {
  color: var(--text-secondary);
  font-size: 14px;
  margin-bottom: 16px;
}

.section-header-with-action {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;

  .section-header {
    flex: 1;
  }

  .section-actions {
    display: flex;
    gap: 12px;
    align-items: center;

    .btn-icon {
      margin-right: 6px;
    }
  }
}

// 节点表格
.slaves-table {
  background: transparent;
  border-radius: var(--radius-lg);
  overflow: hidden;

  :deep(.el-table__header-wrapper) {
    th.el-table__cell {
      background-color: rgba(255, 255, 255, 0.03) !important;
      color: var(--text-secondary) !important;
      font-weight: 500 !important;
      font-size: 13px !important;
      border-bottom: 1px solid rgba(255, 255, 255, 0.06) !important;
    }
  }

  :deep(.el-table__body-wrapper) {
    background-color: var(--bg-card);

    td.el-table__cell {
      border-bottom: 1px solid rgba(255, 255, 255, 0.04) !important;
      color: var(--text-primary) !important;
    }
  }

  :deep(.el-table__row) {
    background-color: var(--bg-card);

    &:hover {
      background-color: rgba(255, 255, 255, 0.02) !important;
    }
  }

  .node-name-cell {
    display: flex;
    align-items: center;
    gap: 10px;

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
  }

  .status-tag {
    font-weight: 500;
  }

  .status-cell {
    display: flex;
    align-items: center;
    justify-content: center;
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

  // 资源使用列样式
  .resource-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 4px 0;

    .resource-row {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 11px;
      line-height: 1.4;
    }

    .resource-label {
      width: 28px;
      color: var(--text-secondary);
      flex-shrink: 0;
    }

    .resource-progress {
      flex: 1;
      min-width: 0;
    }

    .resource-value {
      min-width: 32px;
      text-align: right;
      color: var(--text-primary);
      font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    }
  }

  .no-data {
    color: var(--text-secondary);
    font-size: 13px;
  }

  .action-btns {
    display: flex;
    gap: 2px;
    flex-wrap: nowrap;
    overflow: hidden;
    justify-content: flex-start;

    .action-btn {
      padding: 4px 6px;
      font-size: 13px;

      .el-icon {
        font-size: 16px;
      }

      &.icon-btn {
        padding: 6px;

        .el-icon {
          margin-right: 0;
        }
      }
    }

    .check-btn,
    .edit-btn {
      color: var(--accent-blue);
    }

    .delete-btn {
      color: var(--accent-red) !important;
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
  padding: 60px 20px;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  margin-top: 20px;

  .empty-icon {
    width: 80px;
    height: 80px;
    border-radius: 50%;
    background: linear-gradient(135deg, rgba(0, 212, 255, 0.1), rgba(0, 102, 255, 0.1));
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 16px;

    .el-icon {
      font-size: 40px;
      color: var(--accent-blue);
      opacity: 0.6;
    }
  }

  .empty-title {
    font-size: 16px;
    font-weight: 500;
    color: var(--text-primary);
    margin-bottom: 8px;
  }

  .empty-desc {
    font-size: 14px;
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
