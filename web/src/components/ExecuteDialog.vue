<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="$emit('update:visible', $event)"
    width="580px"
    :close-on-click-modal="false"
    class="execute-dialog"
    @open="handleOpen"
  >
    <!-- 自定义标题 -->
    <template #header>
      <div class="dialog-header">
        <div class="header-icon">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M13 2L3 14H12L11 22L21 10H12L13 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <span class="header-title">执行脚本</span>
      </div>
    </template>

    <div class="dialog-content">
      <!-- 脚本名称 -->
      <div class="script-section">
        <span class="script-label">脚本名称</span>
        <span class="script-name">{{ scriptName }}</span>
      </div>

      <!-- 执行模式选择 -->
      <div class="mode-section">
        <div class="section-label">选择执行模式</div>
        <div class="mode-cards">
          <div
            class="mode-card"
            :class="{ active: executionMode === 'local' }"
            @click="executionMode = 'local'"
          >
            <div class="mode-icon local">
              <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <rect x="2" y="3" width="20" height="14" rx="2" stroke="currentColor" stroke-width="2"/>
                <path d="M8 21H16M12 17V21" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
              </svg>
            </div>
            <div class="mode-info">
              <span class="mode-title">本地执行</span>
              <span class="mode-desc">在当前服务器执行测试</span>
            </div>
            <div class="mode-check" v-if="executionMode === 'local'">
              <el-icon><Check /></el-icon>
            </div>
          </div>

          <div
            class="mode-card"
            :class="{ active: executionMode === 'distributed' }"
            @click="executionMode = 'distributed'"
          >
            <div class="mode-icon distributed">
              <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="2"/>
                <circle cx="4" cy="6" r="2" stroke="currentColor" stroke-width="2"/>
                <circle cx="20" cy="6" r="2" stroke="currentColor" stroke-width="2"/>
                <circle cx="4" cy="18" r="2" stroke="currentColor" stroke-width="2"/>
                <circle cx="20" cy="18" r="2" stroke="currentColor" stroke-width="2"/>
                <path d="M9.5 10.5L5.5 7.5M14.5 10.5L18.5 7.5M9.5 13.5L5.5 16.5M14.5 13.5L18.5 16.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
            </div>
            <div class="mode-info">
              <span class="mode-title">分布式执行</span>
              <span class="mode-desc">在多个节点上并行执行</span>
            </div>
            <div class="mode-check" v-if="executionMode === 'distributed'">
              <el-icon><Check /></el-icon>
            </div>
          </div>
        </div>
      </div>

      <!-- 执行备注 -->
      <div class="remarks-section">
        <div class="section-label">执行备注</div>
        <el-input
          v-model="remarks"
          type="textarea"
          :rows="2"
          placeholder="输入执行备注，如：回归测试、压测50并发等"
          class="remarks-input"
          maxlength="200"
          show-word-limit
        />
      </div>

      <div class="detail-section">
        <div class="detail-switch-row">
          <div class="detail-copy">
            <div class="section-label">失败请求明细</div>
            <div class="detail-desc">
              保存失败样本的请求头、请求体、响应头和响应体，执行结果页可直接查看对应 HTTP 信息。
            </div>
          </div>
          <el-switch
            v-model="saveHTTPDetails"
            inline-prompt
            active-text="开启"
            inactive-text="关闭"
          />
        </div>
        <div v-if="executionMode === 'distributed'" class="detail-notice">
          分布式模式会由各 Slave 本地收集失败明细，并在执行结束后自动回传到 Master。
        </div>
      </div>

      <div v-if="executionMode === 'distributed'" class="detail-section">
        <div class="detail-switch-row">
          <div class="detail-copy">
            <div class="section-label">Master 参与执行</div>
            <div class="detail-desc">
              开启后会同时在当前服务器和所选 Slave 上执行同一脚本，结果会自动合并到同一条执行记录。
            </div>
          </div>
          <el-switch
            v-model="includeMaster"
            inline-prompt
            active-text="参与"
            inactive-text="仅调度"
          />
        </div>
      </div>

      <!-- Slave节点选择（分布式模式展开） -->
      <transition name="slide-down">
        <div v-show="executionMode === 'distributed'" class="slave-section">
          <!-- Master IP 配置 -->
          <div class="master-ip-config">
            <div class="config-header">
              <span class="config-label">Master 回调地址</span>
              <el-tooltip content="Slave 节点通过此 IP 将测试结果回传给 Master，多网卡时需要指定正确的 IP" placement="top">
                <el-icon class="info-icon"><InfoFilled /></el-icon>
              </el-tooltip>
            </div>
            <div class="ip-selector">
              <el-select
                v-model="masterHostname"
                filterable
                allow-create
                placeholder="选择或输入 Master IP"
                @change="handleMasterIPChange"
                style="width: 100%;"
              >
                <el-option
                  v-for="iface in networkInterfaces"
                  :key="iface.ip"
                  :label="`${iface.ip} (${iface.name})`"
                  :value="iface.ip"
                />
              </el-select>
              <div class="ip-status" v-if="masterHostname">
                <el-icon class="status-icon" :style="{color: '#67c23a'}"><CircleCheckFilled /></el-icon>
                <span class="status-text">已配置: {{ masterHostname }}</span>
              </div>
              <div class="ip-status ip-warning" v-else>
                <el-icon class="status-icon" :style="{color: '#e6a23c'}"><WarningFilled /></el-icon>
                <span class="status-text">未配置，多网卡环境可能导致 Slave 连接失败</span>
              </div>
            </div>
          </div>

          <div class="section-header">
            <span class="section-label">选择执行节点</span>
            <el-button
              size="small"
              @click="selectAllOnline"
              :disabled="onlineSlaves.length === 0"
            >
              全选在线节点
            </el-button>
          </div>

          <!-- Slave mini卡片列表 -->
          <div class="slave-grid" v-loading="loadingSlaves">
            <div v-if="slaves.length === 0 && !loadingSlaves" class="empty-slaves">
              <span>暂无Slave节点，请先添加</span>
            </div>

            <div
              v-for="slave in slaves"
              :key="slave.id"
              class="slave-mini-card"
              :class="{
                selected: selectedSlaves.includes(slave.id),
                offline: slave.status !== 'online'
              }"
              @click="toggleSlave(slave)"
            >
              <!-- 选中指示 -->
              <div class="selection-indicator" v-if="selectedSlaves.includes(slave.id)">
                <el-icon><Check /></el-icon>
              </div>

              <!-- 节点信息 -->
              <div class="mini-card-content">
                <span class="mini-node-name">{{ slave.name }}</span>
                <span class="mini-node-address">{{ slave.host }}:{{ slave.port }}</span>
              </div>

              <!-- 状态标签 -->
              <div class="mini-status">
                <el-tag
                  :type="getStatusType(slave.status)"
                  size="small"
                >
                  {{ getStatusText(slave.status) }}
                </el-tag>
              </div>
            </div>
          </div>

          <!-- 提示信息 -->
          <div v-if="onlineSlaves.length === 0 && !loadingSlaves" class="warning-box">
            <el-icon><Warning /></el-icon>
            <span>当前没有可用的在线Slave节点，请先添加并检测节点</span>
          </div>

          <!-- 分布式执行须知 -->
          <div class="distribute-notice">
            <div class="notice-header">
              <svg class="notice-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2"/>
                <path d="M12 8V12M12 16H12.01" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
              </svg>
              <span class="notice-title">分布式执行须知</span>
            </div>
            <div class="notice-content">
              请确保所选 Slave 节点已启动 jmeter-server 服务。启动命令：<code>jmeter-server -Dserver.rmi.ssl.disable=true</code>
            </div>
          </div>
        </div>
      </transition>
    </div>

    <!-- 底部按钮 -->
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleCancel">取消</el-button>
        <el-button
          type="primary"
          :disabled="!canExecute || executing"
          :loading="executing"
          @click="handleExecute"
        >
          <el-icon v-if="!executing"><VideoPlay /></el-icon>
          确认执行
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Warning, VideoPlay, InfoFilled, CircleCheckFilled, WarningFilled } from '@element-plus/icons-vue'
import { slaveApi } from '@/api/slave'
import { executionApi } from '@/api/execution'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  scriptId: {
    type: [Number, String],
    default: null
  },
  scriptName: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:visible', 'success'])

// 数据状态
const executionMode = ref('local')
const slaves = ref([])
const selectedSlaves = ref([])
const loadingSlaves = ref(false)
const executing = ref(false)
const remarks = ref('')
const saveHTTPDetails = ref(false)
const includeMaster = ref(false)
const masterHostname = ref('')
const networkInterfaces = ref([])

// 计算属性
const onlineSlaves = computed(() => {
  return slaves.value.filter(s => s.status === 'online')
})

const canExecute = computed(() => {
  if (executionMode.value === 'local') {
    return true
  }
  // 分布式模式必须至少选一个slave
  return selectedSlaves.value.length > 0
})

// 监听visible变化，关闭时重置状态
watch(() => props.visible, (val) => {
  if (!val) {
    resetForm()
  }
})

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

// 重置表单
const resetForm = () => {
  executionMode.value = 'local'
  selectedSlaves.value = []
  remarks.value = ''
  saveHTTPDetails.value = false
  includeMaster.value = false
  masterHostname.value = ''
  networkInterfaces.value = []
}

// 弹窗打开时加载slave列表和Master配置
const handleOpen = () => {
  loadSlaves()
  loadMasterConfig()
}

// 加载Slave列表
const loadSlaves = async () => {
  loadingSlaves.value = true
  try {
    const res = await slaveApi.getList()
    slaves.value = res.data?.list || res.data || []
  } catch (error) {
    console.error('加载Slave列表失败:', error)
    ElMessage.error('加载Slave列表失败')
  } finally {
    loadingSlaves.value = false
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
    }
  } catch (err) {
    console.error('加载 Master 配置失败:', err)
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

// 切换slave选择
const toggleSlave = (slave) => {
  if (slave.status !== 'online') return
  
  const index = selectedSlaves.value.indexOf(slave.id)
  if (index === -1) {
    selectedSlaves.value.push(slave.id)
  } else {
    selectedSlaves.value.splice(index, 1)
  }
}

// 全选在线节点
const selectAllOnline = () => {
  selectedSlaves.value = onlineSlaves.value.map(s => s.id)
}

// 取消
const handleCancel = () => {
  emit('update:visible', false)
}

// 执行
const handleExecute = async () => {
  if (executionMode.value === 'distributed' && selectedSlaves.value.length === 0) {
    ElMessage.warning('请至少选择一个Slave节点')
    return
  }

  executing.value = true

  try {
    // 分布式模式预检查
    if (executionMode.value === 'distributed') {
      const offlineSlaves = []
      for (const slaveId of selectedSlaves.value) {
        try {
          const res = await slaveApi.checkConnectivity(slaveId)
          const isOnline = res.data?.online || res.data?.status === 'online'
          if (!isOnline) {
            const slave = slaves.value.find(s => s.id === slaveId)
            offlineSlaves.push(slave?.name || `ID:${slaveId}`)
          }
        } catch {
          const slave = slaves.value.find(s => s.id === slaveId)
          offlineSlaves.push(slave?.name || `ID:${slaveId}`)
        }
      }

      if (offlineSlaves.length === selectedSlaves.value.length) {
        ElMessage.error(`所有选中的 Slave 节点均不可用: ${offlineSlaves.join(', ')}`)
        executing.value = false
        return
      }

      if (offlineSlaves.length > 0) {
        // 部分离线，提示用户但允许继续
        try {
          await ElMessageBox.confirm(
            `以下节点不可用: ${offlineSlaves.join(', ')}，是否使用剩余在线节点继续执行？`,
            '部分节点离线',
            { confirmButtonText: '继续执行', cancelButtonText: '取消', type: 'warning' }
          )
        } catch {
          executing.value = false
          return
        }
      }
    }

    const data = {
      script_id: props.scriptId,
      slave_ids: executionMode.value === 'distributed' ? selectedSlaves.value : [],
      remarks: remarks.value,
      save_http_details: saveHTTPDetails.value,
      include_master: executionMode.value === 'distributed' ? includeMaster.value : false
    }
    await executionApi.create(data)
    ElMessage.success('执行已启动')
    emit('success')
    emit('update:visible', false)
  } catch (error) {
    console.error('启动执行失败:', error)
    ElMessage.error(error.response?.data?.message || '启动执行失败')
  } finally {
    executing.value = false
  }
}
</script>

<style scoped lang="scss">
// 弹窗标题
.dialog-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-blue);
  border-radius: var(--radius-sm);
  
  svg {
    width: 18px;
    height: 18px;
    color: #fff;
  }
}

.header-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

// 内容区域
.dialog-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

// 脚本信息
.script-section {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  background: var(--bg-secondary);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
}

.script-label {
  font-size: 13px;
  color: var(--text-secondary);
}

.script-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

// 模式选择
.mode-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-label {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

// 备注区域
.remarks-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px 18px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
}

.detail-switch-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.detail-copy {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.detail-desc {
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-secondary);
  max-width: 380px;
}

.detail-notice {
  font-size: 12px;
  line-height: 1.6;
  color: #f7c46c;
}

.remarks-input {
  :deep(.el-textarea__inner) {
    background-color: var(--bg-secondary);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    font-size: 14px;
    resize: none;

    &:hover {
      border-color: rgba(255, 255, 255, 0.15);
    }

    &:focus {
      border-color: var(--accent-blue);
      box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.1);
    }

    &::placeholder {
      color: var(--text-secondary);
    }
  }
}

.mode-cards {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.mode-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  background: var(--bg-card);
  border: 2px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.25s ease;

  &:hover {
    border-color: rgba(255, 255, 255, 0.12);
    background: var(--bg-hover);
  }

  &.active {
    border-color: var(--accent-blue);
    background: rgba(0, 102, 255, 0.08);
  }
}

.mode-icon {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  flex-shrink: 0;

  svg {
    width: 24px;
    height: 24px;
  }

  &.local {
    background: rgba(0, 102, 255, 0.1);
    color: var(--accent-blue);
  }

  &.distributed {
    background: rgba(0, 204, 106, 0.1);
    color: var(--accent-green);
  }
}

.mode-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
}

.mode-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.mode-desc {
  font-size: 12px;
  color: var(--text-secondary);
}

.mode-check {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-blue);
  border-radius: 50%;
  color: #fff;
  font-size: 12px;
  font-weight: bold;
}

// Slave 选择区域
.slave-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

// Slave 网格
.slave-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  max-height: 220px;
  overflow-y: auto;
  padding: 4px;
}

// Slave mini 卡片
.slave-mini-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  min-width: 140px;
  background: var(--bg-card);
  border: 2px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.25s ease;

  &:hover:not(.offline) {
    border-color: rgba(255, 255, 255, 0.12);
    transform: translateY(-2px);
  }

  &.selected {
    border-color: var(--accent-blue);
    background: rgba(0, 102, 255, 0.08);
  }

  &.offline {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.selection-indicator {
  position: absolute;
  top: -6px;
  left: -6px;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-blue);
  border-radius: 50%;
  color: #fff;
  font-size: 12px;
}

.mini-card-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.mini-node-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mini-node-address {
  font-size: 11px;
  color: var(--text-secondary);
  font-family: 'Monaco', 'Menlo', monospace;
}

.mini-status {
  margin-top: 4px;
}

// 空状态
.empty-slaves {
  width: 100%;
  padding: 30px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 13px;
}

// 警告提示
.warning-box {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: rgba(255, 149, 0, 0.08);
  border: 1px solid rgba(255, 149, 0, 0.15);
  border-radius: var(--radius-sm);
  color: var(--accent-orange);
  font-size: 13px;

  .el-icon {
    font-size: 16px;
  }
}

// 底部区域
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

// Master IP 配置区域
.master-ip-config {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);

  .config-header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 10px;

    .config-label {
      font-size: 13px;
      font-weight: 500;
      color: var(--text-primary, #e5e5e7);
    }

    .info-icon {
      font-size: 14px;
      color: var(--text-secondary, #86868b);
      cursor: help;
    }
  }

  .ip-selector {
    .ip-status {
      display: flex;
      align-items: center;
      gap: 6px;
      margin-top: 8px;
      font-size: 12px;
      color: var(--text-secondary, #86868b);

      .status-icon {
        font-size: 14px;
      }
    }

    .ip-warning {
      color: #e6a23c;
    }
  }
}

// 分布式执行须知（暗色风格）
.distribute-notice {
  margin-top: 16px;
  padding: 12px 16px;
  background: rgba(255, 165, 0, 0.06);
  border: 1px solid rgba(255, 165, 0, 0.15);
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.6;

  .notice-header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 6px;
  }

  .notice-icon {
    width: 14px;
    height: 14px;
    color: rgba(255, 165, 0, 0.7);
    flex-shrink: 0;
  }

  .notice-title {
    font-size: 12px;
    font-weight: 500;
    color: rgba(255, 165, 0, 0.7);
  }

  .notice-content {
    color: rgba(255, 255, 255, 0.5);
    padding-left: 20px;
  }

  code {
    background: rgba(255, 255, 255, 0.08);
    padding: 2px 6px;
    border-radius: 4px;
    font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
    font-size: 11px;
    color: rgba(255, 255, 255, 0.7);
  }
}

// 滑入动画
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.3s ease;
}

.slide-down-enter-from,
.slide-down-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
