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

      <!-- 执行选项 -->
      <div class="options-section">
        <div class="section-label">执行选项</div>
        <div class="compact-options">
          <div class="option-row">
            <div class="option-info">
              <span class="option-title">失败请求明细</span>
              <el-tooltip content="保存失败样本的请求头、请求体、响应头和响应体，执行结果页可直接查看对应 HTTP 信息。" placement="top">
                <el-icon class="help-icon"><InfoFilled /></el-icon>
              </el-tooltip>
            </div>
            <el-switch
              v-model="saveHTTPDetails"
              inline-prompt
              active-text="开启"
              inactive-text="关闭"
              size="small"
            />
          </div>

          <div v-if="executionMode === 'distributed'" class="option-row">
            <div class="option-info">
              <span class="option-title">Master 参与执行</span>
              <el-tooltip content="开启后会同时在当前服务器和所选 Slave 上执行同一脚本，结果会自动合并到同一条执行记录。" placement="top">
                <el-icon class="help-icon"><InfoFilled /></el-icon>
              </el-tooltip>
            </div>
            <el-switch
              v-model="includeMaster"
              inline-prompt
              active-text="参与"
              inactive-text="仅调度"
              size="small"
            />
          </div>

          <div v-if="executionMode === 'distributed'" class="option-row">
            <div class="option-info">
              <span class="option-title">CSV 数据分片</span>
              <el-tooltip content="分布式模式下自动将 CSV 文件按节点数均匀拆分，每个节点只读取属于自己的那部分数据，避免 token/参数重复消费。" placement="top">
                <el-icon class="help-icon"><InfoFilled /></el-icon>
              </el-tooltip>
            </div>
            <el-switch
              v-model="splitCSV"
              inline-prompt
              active-text="开启"
              inactive-text="关闭"
              size="small"
            />
          </div>
        </div>
        <div v-if="splitCSV && executionMode === 'distributed'" class="detail-notice compact">
          执行前将自动拆分脚本中引用的 CSV 文件，并通过 Agent 分发到各 Slave 节点。请确保所有 Slave 的 Agent 服务已启动。
        </div>
        <div v-if="dependencyLoading" class="detail-notice compact">
          正在分析脚本依赖...
        </div>
        <div v-else-if="dependencySummaryItems.length || dependencyWarnings.length" class="dependency-summary">
          <div v-if="dependencySummaryItems.length" class="dependency-chip-row">
            <span v-for="item in dependencySummaryItems" :key="item.label" class="dependency-chip">
              {{ item.label }} {{ item.count }}
            </span>
          </div>
          <div v-for="warning in dependencyWarnings" :key="warning" class="detail-notice compact warning">
            {{ warning }}
          </div>
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
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Warning, VideoPlay, InfoFilled, CircleCheckFilled, WarningFilled } from '@element-plus/icons-vue'
import { slaveApi } from '@/api/slave'
import { executionApi } from '@/api/execution'
import { scriptApi } from '@/api/script'

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

const router = useRouter()

// 数据状态
const executionMode = ref('local')
const slaves = ref([])
const selectedSlaves = ref([])
const loadingSlaves = ref(false)
const executing = ref(false)
const remarks = ref('')
const saveHTTPDetails = ref(false)
const includeMaster = ref(false)
const splitCSV = ref(false)
const masterHostname = ref('')
const networkInterfaces = ref([])
const dependencyLoading = ref(false)
const dependencyReport = ref(null)

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

const dependencySummaryItems = computed(() => {
  const report = dependencyReport.value
  if (!report) return []
  const items = []
  if (report.csv_files?.length) items.push({ label: 'CSV', count: report.csv_files.length })
  if (report.file_dependencies?.length) items.push({ label: '本地文件', count: report.file_dependencies.length })
  if (report.plugin_dependencies?.length) items.push({ label: '插件组件', count: report.plugin_dependencies.length })
  if (report.missing_dependencies?.length) items.push({ label: '缺失依赖', count: report.missing_dependencies.length })
  return items
})

const dependencyWarnings = computed(() => dependencyReport.value?.warnings || [])

const escapeHtml = (value) => {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

const trimList = (items = [], limit = 8) => {
  if (!Array.isArray(items)) return []
  if (items.length <= limit) return items
  return [...items.slice(0, limit), `... 另有 ${items.length - limit} 项`]
}

const renderEnvironmentDifferenceHtml = (report) => {
  const baselineNode = report?.baseline?.node || '基线节点'
  const warnings = Array.isArray(report?.warnings) ? report.warnings : []
  const differences = Array.isArray(report?.differences) ? report.differences : []

  const warningHtml = warnings.length
    ? `<div class="env-confirm-block"><div class="env-confirm-title">检测到的差异</div><ul>${warnings.map(item => `<li>${escapeHtml(item)}</li>`).join('')}</ul></div>`
    : ''

  const differenceHtml = differences.length
    ? `<div class="env-confirm-block"><div class="env-confirm-title">差异明细（基线：${escapeHtml(baselineNode)}）</div>${differences.map((diff) => {
      const values = []
      if (diff.baseline) {
        values.push(`<div><strong>基线：</strong>${escapeHtml(diff.baseline)}</div>`)
      }
      if (diff.current) {
        values.push(`<div><strong>当前：</strong>${escapeHtml(diff.current)}</div>`)
      }
      if (diff.added?.length) {
        values.push(`<div><strong>当前多出：</strong><code>${trimList(diff.added).map(escapeHtml).join('</code><br/><code>')}</code></div>`)
      }
      if (diff.missing?.length) {
        values.push(`<div><strong>相对基线缺少：</strong><code>${trimList(diff.missing).map(escapeHtml).join('</code><br/><code>')}</code></div>`)
      }
      return `<div class="env-diff-item">
        <div class="env-diff-summary">${escapeHtml(diff.summary || diff.category || '环境差异')}</div>
        <div class="env-diff-node">${escapeHtml(diff.node || '')}</div>
        <div class="env-diff-values">${values.join('')}</div>
      </div>`
    }).join('')}</div>`
    : ''

  return `
    <div class="env-confirm">
      <p>分布式执行前发现环境差异，这些差异不一定会导致执行失败，但可能影响结果一致性。</p>
      ${warningHtml}
      ${differenceHtml}
      <p class="env-confirm-foot">如果你确认这些差异可接受，可以选择“忽略并继续执行”。</p>
    </div>
  `
}

// 监听visible变化，关闭时重置状态
watch(() => props.visible, (val) => {
  if (!val) {
    resetForm()
  }
})

watch([executionMode, splitCSV, () => props.scriptId, () => props.visible], ([, , scriptId, visible]) => {
  if (!visible || !scriptId) return
  loadScriptDependencies()
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
  splitCSV.value = false
  masterHostname.value = ''
  networkInterfaces.value = []
  dependencyReport.value = null
}

// 弹窗打开时加载slave列表和Master配置
const handleOpen = () => {
  loadSlaves()
  loadMasterConfig()
  loadScriptDependencies()
}

const loadScriptDependencies = async () => {
  if (!props.scriptId) return
  dependencyLoading.value = true
  try {
    const res = await scriptApi.getDependencies(props.scriptId, {
      distributed: executionMode.value === 'distributed',
      split_csv: executionMode.value === 'distributed' ? splitCSV.value : false
    })
    dependencyReport.value = res.data || null
  } catch (error) {
    console.error('分析脚本依赖失败:', error)
    dependencyReport.value = null
  } finally {
    dependencyLoading.value = false
  }
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
const submitExecution = async (ignoreEnvironmentWarnings = false) => {
  const data = {
    script_id: props.scriptId,
    slave_ids: executionMode.value === 'distributed' ? selectedSlaves.value : [],
    remarks: remarks.value,
    save_http_details: saveHTTPDetails.value,
    include_master: executionMode.value === 'distributed' ? includeMaster.value : false,
    split_csv: executionMode.value === 'distributed' ? splitCSV.value : false,
    ignore_environment_warnings: ignoreEnvironmentWarnings
  }
  return executionApi.create(data)
}

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

    let res
    try {
      res = await submitExecution(false)
    } catch (error) {
      const resp = error?.response
      const envReport = resp?.data?.data?.environment_report
      const canIgnore = resp?.status === 409 && resp?.data?.code === 40901 && resp?.data?.data?.can_ignore
      if (!canIgnore || !envReport) {
        throw error
      }

      try {
        await ElMessageBox.confirm(
          renderEnvironmentDifferenceHtml(envReport),
          '环境一致性存在差异',
          {
            confirmButtonText: '忽略并继续执行',
            cancelButtonText: '取消',
            type: 'warning',
            dangerouslyUseHTMLString: true,
            customClass: 'environment-confirm-dialog'
          }
        )
      } catch {
        return
      }

      res = await submitExecution(true)
    }
    const newExecutionId = res.data?.id
    
    ElMessage.success('执行已启动')
    emit('success')
    emit('update:visible', false)
    
    // 弹出选择框，询问是否查看详情
    if (newExecutionId) {
      setTimeout(() => {
        ElMessageBox.confirm(
          '执行任务已创建成功，是否立即查看执行详情？',
          '执行成功',
          {
            confirmButtonText: '查看详情',
            cancelButtonText: '留在当前页',
            type: 'success',
          }
        ).then(() => {
          router.push({ path: `/executions/${newExecutionId}` })
        }).catch(() => {
          // 留在当前页，不做任何操作
        })
      }, 300)
    }
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

// 执行选项区域
.options-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.compact-options {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
}

.option-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);

  &:last-child {
    border-bottom: none;
  }
}

.option-info {
  display: flex;
  align-items: center;
  gap: 6px;
}

.option-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.help-icon {
  font-size: 14px;
  color: var(--text-secondary);
  cursor: help;

  &:hover {
    color: var(--accent-blue);
  }
}

.detail-notice {
  font-size: 12px;
  line-height: 1.6;
  color: #f7c46c;

  &.compact {
    margin-top: 4px;
    padding: 8px 12px;
    background: rgba(247, 196, 108, 0.08);
    border-radius: var(--radius-sm);
    font-size: 11px;
  }
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

:deep(.environment-confirm-dialog) {
  .el-message-box__message {
    max-height: 60vh;
    overflow: auto;
  }
}

:deep(.env-confirm) {
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-secondary);
}

:deep(.env-confirm-block) {
  margin-top: 12px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
}

:deep(.env-confirm-title) {
  margin-bottom: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

:deep(.env-confirm ul) {
  margin: 0;
  padding-left: 18px;
}

:deep(.env-diff-item) {
  padding: 10px 0;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

:deep(.env-diff-item:first-child) {
  border-top: none;
  padding-top: 0;
}

:deep(.env-diff-summary) {
  font-weight: 600;
  color: var(--text-primary);
}

:deep(.env-diff-node) {
  font-size: 12px;
  color: var(--accent-blue);
}

:deep(.env-diff-values) {
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-secondary);
}

:deep(.env-diff-values code) {
  display: inline-block;
  margin-top: 4px;
  padding: 2px 6px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 8px;
  color: var(--text-primary);
  white-space: pre-wrap;
}

:deep(.env-confirm-foot) {
  margin-top: 12px;
  color: #f7c46c;
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

.dependency-summary {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 10px;
}

.dependency-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.dependency-chip {
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(59, 130, 246, 0.12);
  border: 1px solid rgba(59, 130, 246, 0.18);
  color: var(--accent-blue);
  font-size: 12px;
  line-height: 1;
}

.detail-notice.warning {
  color: #f59e0b;
  border-color: rgba(245, 158, 11, 0.2);
  background: rgba(245, 158, 11, 0.08);
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
