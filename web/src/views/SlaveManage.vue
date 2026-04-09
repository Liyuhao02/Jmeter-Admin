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
        <el-table-column label="状态" width="120" align="center" sortable prop="status">
          <template #default="{ row }">
            <div class="status-cell">
              <span class="status-dot" :class="row.status"></span>
              <span class="status-label" :class="row.status">{{ getStatusText(row.status) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="最后检测" width="140" align="center">
          <template #default="{ row }">
            <span class="time-text" :title="row.last_check_time">
              {{ formatRelativeTime(row.last_check_time) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160">
          <template #default="{ row }">
            <span class="time-text">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <div class="action-btns">
              <el-button
                link
                type="primary"
                @click="handleCheck(row)"
                :loading="checkingId === row.id"
                class="action-btn check-btn"
              >
                <el-icon><CircleCheck /></el-icon>
                检测
              </el-button>
              <el-button
                link
                type="primary"
                @click="handleEdit(row)"
                class="action-btn edit-btn"
              >
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
              <el-button
                link
                type="danger"
                @click="handleDelete(row)"
                class="action-btn delete-btn"
              >
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
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
  port: 1099
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

// 格式化时间
const formatTime = (time) => {
  return formatDateTimeInShanghai(time)
}

// 格式化相对时间（如"30秒前"、"2分钟前"）
const formatRelativeTime = (time) => {
  return formatRelativeTimeInShanghai(time)
}

// 加载Slave列表
const loadSlaves = async () => {
  loading.value = true
  try {
    const res = await slaveApi.getList()
    slaveList.value = res.data?.list || res.data || []
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
    
    // 更新列表中的状态
    const index = slaveList.value.findIndex(s => s.id === row.id)
    if (index !== -1) {
      slaveList.value[index].status = status
    }
    
    if (status === 'online') {
      ElMessage.success(`${row.host}: 在线`)
    } else {
      ElMessage.error(`${row.host}: 离线`)
    }
  } catch (error) {
    console.error('检测失败:', error)
    ElMessage.error(`检测 ${row.host} 失败`)
    // 更新为离线状态
    const index = slaveList.value.findIndex(s => s.id === row.id)
    if (index !== -1) {
      slaveList.value[index].status = 'offline'
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
      
      // 更新列表中的状态
      const index = slaveList.value.findIndex(s => s.id === row.id)
      if (index !== -1) {
        slaveList.value[index].status = status
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

  .action-btns {
    display: flex;
    gap: 4px;
    flex-wrap: nowrap;
    overflow: hidden;

    .action-btn {
      padding: 4px 8px;
      font-size: 13px;

      .el-icon {
        margin-right: 4px;
        font-size: 14px;
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
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
