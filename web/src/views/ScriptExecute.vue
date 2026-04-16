<template>
  <div class="script-execute-page" v-loading="pageLoading">
    <section class="section-card execute-hero">
      <div class="execute-hero-main">
        <button class="back-btn" @click="goBack">
          <el-icon><ArrowLeft /></el-icon>
          <span>返回脚本列表</span>
        </button>
        <div class="execute-title-block">
          <div class="execute-kicker">EXECUTION</div>
          <h1>{{ scriptInfo.name || '执行脚本' }}</h1>
          <p>{{ scriptInfo.description || '配置本地或分布式执行策略，确认节点资源后直接发起压测。' }}</p>
        </div>
      </div>
      <div class="execute-hero-actions">
        <el-button @click="refreshAll" :loading="refreshing">
          <el-icon><Refresh /></el-icon>
          刷新信息
        </el-button>
        <el-button type="primary" plain @click="goEdit">
          <el-icon><Edit /></el-icon>
          编辑脚本
        </el-button>
      </div>
      <div class="hero-fact-grid">
        <div v-for="item in heroFacts" :key="item.label" class="hero-fact-card">
          <span class="hero-fact-label">{{ item.label }}</span>
          <strong class="hero-fact-value">{{ item.value }}</strong>
          <span class="hero-fact-desc">{{ item.desc }}</span>
        </div>
      </div>
    </section>

    <div class="execute-layout">
      <main class="execute-main">
        <section class="section-card configuration-panel">
          <div class="panel-heading">
            <div>
              <div class="section-label">MODE</div>
              <div class="section-title">执行配置</div>
              <div class="section-desc">先确定执行方式，再补充备注和执行开关。核心配置压缩到一屏里，不再来回跳。</div>
            </div>
          </div>

          <div class="mode-grid">
            <button
              class="mode-card"
              :class="{ active: executionMode === 'local' }"
              @click="executionMode = 'local'"
            >
              <div class="mode-icon is-local">
                <el-icon><Monitor /></el-icon>
              </div>
              <div class="mode-copy">
                <strong>本地执行</strong>
                <span>仅在当前 Master 所在机器执行，适合回归和快速验证。</span>
              </div>
              <span v-if="executionMode === 'local'" class="mode-check">
                <el-icon><Check /></el-icon>
              </span>
            </button>

            <button
              class="mode-card"
              :class="{ active: executionMode === 'distributed' }"
              @click="executionMode = 'distributed'"
            >
              <div class="mode-icon is-distributed">
                <el-icon><Connection /></el-icon>
              </div>
              <div class="mode-copy">
                <strong>分布式执行</strong>
                <span>选择一个或多个 Slave 并行施压，可选让 Master 一起参与执行。</span>
              </div>
              <span v-if="executionMode === 'distributed'" class="mode-check">
                <el-icon><Check /></el-icon>
              </span>
            </button>
          </div>

          <div class="config-grid">
            <div class="config-card config-card--wide">
              <label class="field-label" for="remarks">执行备注</label>
              <el-input
                id="remarks"
                v-model="remarks"
                type="textarea"
                :rows="3"
                maxlength="200"
                show-word-limit
                placeholder="例如：支付链路回归、晚高峰压测、带错误明细排查"
              />
            </div>

            <div class="config-card">
              <div class="field-label">执行选项</div>
              <div class="option-list">
                <div class="option-item">
                  <div>
                    <strong>失败请求明细</strong>
                    <p>保存失败样本的请求和响应细节，执行结果页可直接排查 HTTP 问题。</p>
                  </div>
                  <el-switch
                    v-model="saveHTTPDetails"
                    inline-prompt
                    active-text="开"
                    inactive-text="关"
                  />
                </div>

                <div v-if="executionMode === 'distributed'" class="option-item">
                  <div>
                    <strong>Master 参与执行</strong>
                    <p>开启后 Master 既负责调度，也会参与施压；关闭后 Master 只做调度。</p>
                  </div>
                  <el-switch
                    v-model="includeMaster"
                    inline-prompt
                    active-text="参与"
                    inactive-text="仅调度"
                  />
                </div>

                <div v-if="executionMode === 'distributed'" class="option-item">
                  <div>
                    <strong>CSV 数据分片</strong>
                    <p>分布式模式下按节点拆分 CSV，降低多个节点重复消费同一批数据的概率。</p>
                  </div>
                  <el-switch
                    v-model="splitCSV"
                    inline-prompt
                    active-text="开启"
                    inactive-text="关闭"
                  />
                </div>
              </div>
            </div>
          </div>
        </section>

        <section v-if="executionMode === 'distributed'" class="section-card distributed-panel">
          <div class="panel-heading">
            <div>
              <div class="section-label">DISTRIBUTED</div>
              <div class="section-title">分布式编排</div>
              <div class="section-desc">把 Master 回调地址和待选 Slave 节点放在同一区块里，确认之后就能直接执行。</div>
            </div>
            <div class="panel-meta">
              <span class="panel-chip">已选 {{ selectedSlaves.length }} 台</span>
              <span class="panel-chip">在线 {{ onlineSlaves.length }} 台</span>
            </div>
          </div>

          <div class="distributed-grid">
            <div class="config-card">
              <label class="field-label">Master 回调地址</label>
              <el-select
                v-model="masterHostname"
                filterable
                allow-create
                placeholder="选择或输入 Master IP"
                @change="handleMasterIPChange"
              >
                <el-option
                  v-for="iface in networkInterfaces"
                  :key="iface.ip"
                  :label="`${iface.ip} (${iface.name})`"
                  :value="iface.ip"
                />
              </el-select>
              <div class="callback-preview">
                <span>回调基地址</span>
                <code>{{ masterCallbackBaseURL || '未配置' }}</code>
              </div>
            </div>

            <div class="config-card selection-summary-card">
              <div class="selection-summary-top">
                <div>
                  <div class="field-label">节点选择</div>
                  <p>在线节点可直接加入本次执行，离线节点会被自动禁用。</p>
                </div>
                <div class="selection-actions">
                  <el-button @click="selectAllOnline" :disabled="!onlineSlaves.length">全选在线</el-button>
                  <el-button @click="clearSlaveSelection" :disabled="!selectedSlaves.length">清空</el-button>
                </div>
              </div>
              <div class="selection-metrics">
                <div class="selection-metric">
                  <span>已选节点</span>
                  <strong>{{ selectedSlaves.length }}</strong>
                </div>
                <div class="selection-metric">
                  <span>平均 CPU</span>
                  <strong>{{ selectedAggregate.cpu }}</strong>
                </div>
                <div class="selection-metric">
                  <span>平均内存</span>
                  <strong>{{ selectedAggregate.memory }}</strong>
                </div>
                <div class="selection-metric">
                  <span>连接数</span>
                  <strong>{{ selectedAggregate.connections }}</strong>
                </div>
              </div>
            </div>
          </div>

          <div class="slave-card-grid" v-loading="loadingSlaves">
            <button
              v-for="slave in slaveNodes"
              :key="slave.id"
              class="slave-select-card"
              :class="{
                active: selectedSlaves.includes(slave.id),
                offline: slave.status !== 'online'
              }"
              :disabled="slave.status !== 'online'"
              :aria-pressed="selectedSlaves.includes(slave.id)"
              @click="toggleSlave(slave)"
            >
              <div class="slave-select-header">
                <div>
                  <strong>{{ slave.name }}</strong>
                  <div class="slave-address">{{ slave.host }}:{{ slave.port }}</div>
                </div>
                <div class="slave-select-side">
                  <span class="slave-status" :class="`is-${getNodeTone(slave)}`">{{ getNodeStatusLabel(slave) }}</span>
                  <span v-if="selectedSlaves.includes(slave.id)" class="slave-order-badge">
                    {{ selectedSlaves.indexOf(slave.id) + 1 }}
                  </span>
                </div>
              </div>
              <div class="slave-badges">
                <span class="slave-badge">{{ formatPercent(getStatsValue(slave, 'cpu')) }} CPU</span>
                <span class="slave-badge">{{ formatPercent(getStatsValue(slave, 'memory')) }} 内存</span>
                <span class="slave-badge">{{ formatCount(getConnections(slave)) }} 连接</span>
                <span class="slave-badge" :class="`is-${getEnvironmentTone(slave)}`">{{ getEnvironmentStatusTag(slave) }}</span>
                <span class="slave-badge" v-if="getEnvironmentVersion(slave)">{{ getEnvironmentVersion(slave) }}</span>
              </div>
              <div class="slave-select-footer">
                <span>{{ selectedSlaves.includes(slave.id) ? '已加入本次执行' : '点击加入本次执行' }}</span>
                <span>{{ getNodeLoadLabel(slave) }}</span>
              </div>
            </button>

            <div v-if="!loadingSlaves && !slaveNodes.length" class="empty-panel">
              暂无 Slave 节点，请先到节点管理页添加并检测节点。
            </div>
          </div>
        </section>

        <section v-if="adviceCards.length" class="section-card advice-panel">
          <div class="panel-heading">
            <div>
              <div class="section-label">ADVICE</div>
              <div class="section-title">执行建议</div>
              <div class="section-desc">这些提示不会阻塞执行，但会帮助你在开始前减少失败和结果偏差。</div>
            </div>
            <span class="panel-chip">{{ adviceCards.length }} 条</span>
          </div>

          <div class="advice-stack">
            <article
              v-for="(item, index) in adviceCards"
              :key="item.key"
              class="advice-item"
              :class="`is-${item.tone}`"
            >
              <div class="advice-index">{{ index + 1 }}</div>
              <div class="advice-copy">
                <div class="advice-copy-top">
                  <strong>{{ item.title }}</strong>
                  <span class="advice-tag">{{ item.tag }}</span>
                </div>
                <p>{{ item.desc }}</p>
              </div>
            </article>
          </div>
        </section>
      </main>

      <aside class="execute-side">
        <section class="section-card side-panel side-panel--primary">
          <div class="panel-heading">
            <div>
              <div class="section-label">SUMMARY</div>
              <div class="section-title">本次执行摘要</div>
            </div>
          </div>

          <div class="summary-hero-card">
            <div class="summary-hero-copy">
              <div class="summary-hero-title">
                {{ executionMode === 'distributed' ? '分布式执行工作台' : '本地执行工作台' }}
              </div>
              <div class="summary-hero-desc">{{ footerHint }}</div>
            </div>
            <div class="summary-hero-actions">
              <el-button @click="goBack">返回</el-button>
              <el-button
                type="primary"
                :loading="executing"
                :disabled="!canExecute"
                @click="handleExecute"
              >
                <el-icon v-if="!executing"><VideoPlay /></el-icon>
                {{ executing ? '启动中...' : '开始执行' }}
              </el-button>
            </div>
          </div>

          <div class="summary-grid">
            <div v-for="item in summaryItems" :key="item.label" class="summary-item">
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}</strong>
            </div>
          </div>

          <div class="summary-block">
            <div class="summary-block-label">主脚本文件</div>
            <div class="summary-block-value">{{ mainFileName }}</div>
          </div>

          <div class="summary-block">
            <div class="summary-block-label">回调基地址</div>
            <code>{{ executionMode === 'distributed' ? (masterCallbackBaseURL || '未配置') : '本地执行无需回调' }}</code>
          </div>
        </section>

        <section class="section-card side-panel">
          <div class="panel-heading">
            <div>
              <div class="section-label">RESOURCES</div>
              <div class="section-title">节点资源概览</div>
            </div>
          </div>

          <div class="node-stack">
            <article class="node-resource-card" :class="`is-${getNodeTone(masterNode)}`">
              <div class="node-resource-header">
                <div>
                  <strong>Master</strong>
                  <div class="node-resource-address">{{ masterNode.host || masterHostname || '未配置' }}</div>
                </div>
                <span class="node-role-pill">{{ executionMode === 'distributed' ? (includeMaster ? '参与执行' : '仅调度') : '本地执行' }}</span>
              </div>
              <div class="node-resource-grid">
                <div class="node-metric">
                  <span>CPU</span>
                  <strong>{{ formatPercent(getStatsValue(masterNode, 'cpu')) }}</strong>
                </div>
                <div class="node-metric">
                  <span>连接数</span>
                  <strong>{{ formatCount(getConnections(masterNode)) }}</strong>
                </div>
                <div class="node-metric">
                  <span>内存</span>
                  <strong>{{ formatPercent(getStatsValue(masterNode, 'memory')) }}</strong>
                </div>
                <div class="node-metric">
                  <span>磁盘</span>
                  <strong>{{ formatPercent(getStatsValue(masterNode, 'disk')) }}</strong>
                </div>
              </div>
              <div class="node-env-strip">
                <span class="node-env-badge" :class="`is-${getEnvironmentTone(masterNode)}`">{{ getEnvironmentStatusTag(masterNode) }}</span>
                <span class="node-env-badge" v-if="getEnvironmentVersion(masterNode)">{{ getEnvironmentVersion(masterNode) }}</span>
                <span class="node-env-badge">{{ getPluginCountLabel(masterNode) }}</span>
              </div>
              <div v-if="getEnvironmentWarnings(masterNode).length" class="node-env-note">
                {{ getEnvironmentWarnings(masterNode)[0] }}
              </div>
            </article>

            <template v-if="executionMode === 'distributed'">
              <div class="selected-node-header">
                <span>已选 Slave</span>
                <strong>{{ selectedSlaveCards.length }}</strong>
              </div>

              <div v-if="selectedSlaveCards.length" class="selected-node-summary-strip">
                <span>平均 CPU {{ selectedAggregate.cpu }}</span>
                <span>平均内存 {{ selectedAggregate.memory }}</span>
                <span>连接 {{ selectedAggregate.connections }}</span>
              </div>

              <div v-if="selectedSlaveCards.length" class="selected-node-list">
                <article v-for="slave in selectedSlaveCards" :key="slave.id" class="selected-node-card">
                  <div class="selected-node-top">
                    <div>
                      <strong>{{ slave.name }}</strong>
                      <div class="node-resource-address">{{ slave.host }}:{{ slave.port }}</div>
                    </div>
                    <div class="selected-node-actions">
                      <span class="node-role-pill is-soft">{{ getNodeLoadLabel(slave) }}</span>
                      <button class="remove-node-btn" @click="toggleSlave(slave)">移除</button>
                    </div>
                  </div>
                  <div class="selected-node-metrics">
                    <span>CPU {{ formatPercent(getStatsValue(slave, 'cpu')) }}</span>
                    <span>内存 {{ formatPercent(getStatsValue(slave, 'memory')) }}</span>
                    <span>磁盘 {{ formatPercent(getStatsValue(slave, 'disk')) }}</span>
                    <span>连接 {{ formatCount(getConnections(slave)) }}</span>
                  </div>
                  <div class="node-env-strip compact">
                    <span class="node-env-badge" :class="`is-${getEnvironmentTone(slave)}`">{{ getEnvironmentStatusTag(slave) }}</span>
                    <span class="node-env-badge" v-if="getEnvironmentVersion(slave)">{{ getEnvironmentVersion(slave) }}</span>
                    <span class="node-env-badge">{{ getPluginCountLabel(slave) }}</span>
                  </div>
                  <div v-if="getEnvironmentWarnings(slave).length" class="node-env-note">
                    {{ getEnvironmentWarnings(slave)[0] }}
                  </div>
                </article>
              </div>

              <div v-else class="empty-panel compact">
                还没有选择任何 Slave 节点。
              </div>
            </template>
          </div>
        </section>
      </aside>
    </div>

    <section class="section-card execute-footer">
      <div class="execute-footer-left">
        <div class="execute-footer-title">确认配置后即可开始执行</div>
        <div class="execute-footer-desc">
          {{ footerHint }}
        </div>
      </div>
      <div class="execute-footer-actions">
        <el-button @click="goBack">取消</el-button>
        <el-button
          type="primary"
          :loading="executing"
          :disabled="!canExecute"
          @click="handleExecute"
        >
          <el-icon v-if="!executing"><VideoPlay /></el-icon>
          {{ executing ? '正在启动执行...' : '开始执行' }}
        </el-button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  Check,
  Connection,
  Edit,
  Monitor,
  Refresh,
  VideoPlay
} from '@element-plus/icons-vue'
import { executionApi } from '@/api/execution'
import { scriptApi } from '@/api/script'
import { slaveApi } from '@/api/slave'

const route = useRoute()
const router = useRouter()

const scriptId = computed(() => Number(route.params.id || 0))

const loadingScript = ref(false)
const loadingSlaves = ref(false)
const loadingMaster = ref(false)
const dependencyLoading = ref(false)
const refreshing = ref(false)
const executing = ref(false)

const scriptInfo = ref({})
const fileList = ref([])
const executionMode = ref('local')
const remarks = ref('')
const saveHTTPDetails = ref(false)
const includeMaster = ref(false)
const splitCSV = ref(false)
const masterHostname = ref('')
const networkInterfaces = ref([])
const masterNode = ref({})
const slaveNodes = ref([])
const selectedSlaves = ref([])
const dependencyReport = ref(null)

const pageLoading = computed(() => {
  return loadingScript.value || loadingSlaves.value || loadingMaster.value
})

const isJmxFile = (name = '') => String(name).toLowerCase().endsWith('.jmx')

const mainFile = computed(() => {
  const files = fileList.value || []
  return files.find((item) => isJmxFile(item.file_name || item.name)) || null
})

const mainFileName = computed(() => {
  return mainFile.value?.file_name || scriptInfo.value.file_name || '未检测到主文件'
})

const attachedFiles = computed(() => {
  return (fileList.value || []).filter((item) => !isJmxFile(item.file_name || item.name))
})

const onlineSlaves = computed(() => slaveNodes.value.filter((item) => item.status === 'online'))

const selectedSlaveCards = computed(() => {
  const selectedSet = new Set(selectedSlaves.value)
  return slaveNodes.value.filter((item) => selectedSet.has(item.id))
})

const canExecute = computed(() => {
  if (executing.value) return false
  if (executionMode.value === 'local') return !!scriptId.value
  return !!scriptId.value && selectedSlaves.value.length > 0 && !!masterHostname.value
})

const heroFacts = computed(() => {
  const dependencyState = dependencyReport.value?.missing_dependencies?.length
    ? '存在缺失'
    : dependencyLoading.value
      ? '分析中'
      : '已分析'
  return [
    {
      label: '主脚本',
      value: mainFileName.value,
      desc: '默认使用已上传的 .jmx 主文件'
    },
    {
      label: '关联文件',
      value: `${attachedFiles.value.length} 个`,
      desc: 'CSV、JSON 等依赖文件'
    },
    {
      label: '在线 Slave',
      value: `${onlineSlaves.value.length} / ${slaveNodes.value.length}`,
      desc: '可直接参与分布式执行'
    },
    {
      label: '依赖概况',
      value: dependencyState,
      desc: 'CSV / 插件 / 缺失依赖'
    }
  ]
})

const filteredDependencyWarnings = computed(() => {
  const warnings = dependencyReport.value?.warnings || []
  return warnings.filter((warning) => {
    if (executionMode.value !== 'distributed' && warning.includes('CSV 数据分片')) return false
    if (splitCSV.value && warning.includes('当前未开启 CSV 数据分片')) return false
    return true
  })
})

const adviceCards = computed(() => {
  const cards = []
  const report = dependencyReport.value || {}

  if (report.csv_files?.length) {
    cards.push({
      key: 'csv',
      tag: '数据',
      tone: executionMode.value === 'distributed' && !splitCSV.value ? 'warning' : 'info',
      title: executionMode.value === 'distributed' && !splitCSV.value ? 'CSV 数据建议开启分片' : '检测到 CSV 数据文件',
      desc: executionMode.value === 'distributed' && !splitCSV.value
        ? '当前脚本引用了 CSV 文件，分布式执行建议开启数据分片，避免多个节点重复消费同一批数据。'
        : '脚本中包含 CSV 数据源，当前配置与执行模式基本匹配。'
    })
  }

  if (report.plugin_dependencies?.length) {
    cards.push({
      key: 'plugin',
      tag: '插件',
      tone: 'warning',
      title: '插件一致性提醒',
      desc: `检测到 ${report.plugin_dependencies.length} 个第三方 JMeter 插件，请确认 Master 与所有 Slave 已安装同版本插件。`
    })
  }

  if (report.missing_dependencies?.length) {
    cards.push({
      key: 'missing',
      tag: '高风险',
      tone: 'danger',
      title: '依赖缺失',
      desc: `有 ${report.missing_dependencies.length} 个脚本依赖未在当前脚本关联文件中发现，执行时可能直接失败。`
    })
  }

  filteredDependencyWarnings.value.slice(0, 3).forEach((warning, index) => {
    cards.push({
      key: `warning-${index}`,
      tag: '提示',
      tone: 'warning',
      title: '执行提醒',
      desc: warning
    })
  })

  return cards
})

const summaryItems = computed(() => {
  return [
    { label: '执行模式', value: executionMode.value === 'distributed' ? '分布式执行' : '本地执行' },
    { label: '失败明细', value: saveHTTPDetails.value ? '已开启' : '未开启' },
    {
      label: 'Master 角色',
      value: executionMode.value === 'distributed'
        ? (includeMaster.value ? '参与执行' : '仅调度')
        : '本地执行'
    },
    {
      label: 'CSV 分片',
      value: executionMode.value === 'distributed'
        ? (splitCSV.value ? '已开启' : '未开启')
        : '不适用'
    }
  ]
})

const footerHint = computed(() => {
  if (executionMode.value === 'local') {
    return '当前将以本地模式启动，执行开始后会自动跳转到执行详情页。'
  }
  if (!selectedSlaves.value.length) {
    return '分布式模式下还没有选择 Slave 节点，先在上方挑好节点再启动。'
  }
  return `当前将使用 ${selectedSlaves.value.length} 台 Slave${includeMaster.value ? '，并让 Master 一起参与施压' : ''}。`
})

const masterCallbackBaseURL = computed(() => {
  if (!masterHostname.value) return ''
  if (typeof window === 'undefined') return `http://${masterHostname.value}:8080`
  const protocol = window.location.protocol || 'http:'
  const currentPort = window.location.port || '8080'
  const backendPort = currentPort === '3000' ? '8080' : currentPort
  return `${protocol}//${masterHostname.value}:${backendPort}`
})

const selectedAggregate = computed(() => {
  const statsNodes = selectedSlaveCards.value
  const avg = (type) => {
    const values = statsNodes
      .map((node) => Number(getStatsValue(node, type)))
      .filter((value) => Number.isFinite(value))
    if (!values.length) return '--'
    return `${(values.reduce((sum, value) => sum + value, 0) / values.length).toFixed(0)}%`
  }
  const connections = statsNodes.reduce((sum, node) => sum + Number(getConnections(node) || 0), 0)
  return {
    cpu: avg('cpu'),
    memory: avg('memory'),
    connections: connections ? formatCount(connections) : '--'
  }
})

const parseSystemStats = (node) => {
  if (!node) return null
  if (typeof node.system_stats === 'string') {
    try {
      return node.system_stats ? JSON.parse(node.system_stats) : null
    } catch {
      return null
    }
  }
  return node.system_stats || null
}

const parseEnvironmentInfo = (node) => {
  if (!node) return null
  if (typeof node.environment_info === 'string') {
    try {
      return node.environment_info ? JSON.parse(node.environment_info) : null
    } catch {
      return null
    }
  }
  return node.environment_info || null
}

const normalizeNode = (node) => {
  const stats = parseSystemStats(node)
  const env = parseEnvironmentInfo(node)
  return {
    ...node,
    parsedStats: stats,
    parsedEnvironment: env
  }
}

const getStatsValue = (node, type) => {
  return Number(node?.parsedStats?.[type]?.percent || 0)
}

const getConnections = (node) => {
  return Number(node?.parsedStats?.network?.connections || 0)
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

const getNodeTone = (node) => {
  if (!node) return 'neutral'
  if (node.status !== 'online') return 'danger'
  const cpu = getStatsValue(node, 'cpu')
  const memory = getStatsValue(node, 'memory')
  const disk = getStatsValue(node, 'disk')
  if (cpu >= 85 || memory >= 90 || disk >= 92) return 'danger'
  if (cpu >= 65 || memory >= 75 || disk >= 80) return 'warning'
  return 'success'
}

const getNodeStatusLabel = (node) => {
  if (!node) return '未知'
  if (node.status !== 'online') return '离线'
  if (node.agent_status && node.agent_status !== 'online') return 'Agent 异常'
  return '在线'
}

const getNodeLoadLabel = (node) => {
  const tone = getNodeTone(node)
  if (tone === 'danger') return '高负载'
  if (tone === 'warning') return '需关注'
  return '负载平稳'
}

const escapeHtml = (value) => {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

const renderEnvironmentDifferenceHtml = (report) => {
  const warnings = Array.isArray(report?.warnings) ? report.warnings.slice(0, 6) : []
  const differences = Array.isArray(report?.differences) ? report.differences.slice(0, 6) : []

  const warningList = warnings.length
    ? `<ul class="env-warning-list">${warnings.map((item) => `<li>${escapeHtml(item)}</li>`).join('')}</ul>`
    : '<p>已检测到环境差异，请确认后再继续执行。</p>'

  const differenceList = differences.length
    ? `<div class="env-diff-list">${differences.map((item) => `
      <div class="env-diff-row">
        <div class="env-diff-title">${escapeHtml(item.summary || item.category || '环境差异')}</div>
        <div class="env-diff-node">${escapeHtml(item.node || '')}</div>
      </div>`).join('')}</div>`
    : ''

  return `
    <div class="env-confirm-shell">
      <p>分布式执行前发现环境差异，这些差异不一定会导致执行失败，但可能影响结果一致性。</p>
      ${warningList}
      ${differenceList}
      <p class="env-confirm-foot">如果你确认这些差异可接受，可以继续执行。</p>
    </div>
  `
}

const loadScriptDetail = async () => {
  if (!scriptId.value) return
  loadingScript.value = true
  try {
    const res = await scriptApi.getDetail(scriptId.value)
    scriptInfo.value = res.data?.script || res.data || {}
    fileList.value = res.data?.files || scriptInfo.value.files || []
  } catch (error) {
    console.error('获取脚本详情失败:', error)
    ElMessage.error('获取脚本详情失败')
  } finally {
    loadingScript.value = false
  }
}

const loadScriptDependencies = async () => {
  if (!scriptId.value) return
  dependencyLoading.value = true
  try {
    const res = await scriptApi.getDependencies(scriptId.value, {
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

const loadNodeResources = async () => {
  loadingSlaves.value = true
  try {
    const res = await slaveApi.getHeartbeatStatus({ silent: true })
    masterNode.value = normalizeNode(res.data?.master || {})
    slaveNodes.value = (res.data?.slaves || []).map(normalizeNode)
  } catch (error) {
    console.error('加载节点状态失败:', error)
    ElMessage.error('加载节点状态失败')
  } finally {
    loadingSlaves.value = false
  }
}

const loadMasterConfig = async () => {
  loadingMaster.value = true
  try {
    const [interfacesRes, hostnameRes] = await Promise.all([
      slaveApi.getNetworkInterfaces(),
      slaveApi.getMasterHostname()
    ])
    networkInterfaces.value = interfacesRes.data || []
    masterHostname.value = hostnameRes.data?.master_hostname || ''

    if (!masterHostname.value && networkInterfaces.value.length) {
      masterHostname.value = networkInterfaces.value[0].ip
      await slaveApi.updateMasterHostname(masterHostname.value)
    }
  } catch (error) {
    console.error('加载 Master 配置失败:', error)
  } finally {
    loadingMaster.value = false
  }
}

const refreshAll = async () => {
  refreshing.value = true
  await Promise.all([
    loadScriptDetail(),
    loadNodeResources(),
    loadMasterConfig(),
    loadScriptDependencies()
  ])
  refreshing.value = false
}

const handleMasterIPChange = async (value) => {
  try {
    await slaveApi.updateMasterHostname(value)
    ElMessage.success('Master 回调地址已更新')
  } catch (error) {
    console.error('保存 Master 地址失败:', error)
    ElMessage.error('保存 Master 地址失败')
  }
}

const toggleSlave = (slave) => {
  if (!slave || slave.status !== 'online') return
  const index = selectedSlaves.value.indexOf(slave.id)
  if (index === -1) {
    selectedSlaves.value.push(slave.id)
  } else {
    selectedSlaves.value.splice(index, 1)
  }
}

const clearSlaveSelection = () => {
  selectedSlaves.value = []
}

const selectAllOnline = () => {
  selectedSlaves.value = onlineSlaves.value.map((item) => item.id)
}

const submitExecution = async (ignoreEnvironmentWarnings = false) => {
  const payload = {
    script_id: scriptId.value,
    slave_ids: executionMode.value === 'distributed' ? selectedSlaves.value : [],
    remarks: remarks.value,
    save_http_details: saveHTTPDetails.value,
    include_master: executionMode.value === 'distributed' ? includeMaster.value : false,
    split_csv: executionMode.value === 'distributed' ? splitCSV.value : false,
    ignore_environment_warnings: ignoreEnvironmentWarnings
  }
  return executionApi.create(payload)
}

const handleExecute = async () => {
  if (!scriptId.value) {
    ElMessage.error('无效的脚本 ID')
    return
  }

  if (executionMode.value === 'distributed' && !selectedSlaves.value.length) {
    ElMessage.warning('请至少选择一个在线 Slave 节点')
    return
  }

  executing.value = true

  try {
    if (executionMode.value === 'distributed') {
      const checks = await Promise.all(
        selectedSlaves.value.map(async (slaveId) => {
          const slave = slaveNodes.value.find((item) => item.id === slaveId)
          try {
            const res = await slaveApi.checkConnectivity(slaveId)
            const online = res.data?.status === 'online' || res.data?.online
            return { slave, online }
          } catch {
            return { slave, online: false }
          }
        })
      )

      const offlineNames = checks
        .filter((item) => !item.online)
        .map((item) => item.slave?.name || `ID:${item.slave?.id ?? '未知'}`)

      if (offlineNames.length === selectedSlaves.value.length) {
        ElMessage.error(`所有选中的 Slave 节点均不可用：${offlineNames.join('、')}`)
        return
      }

      if (offlineNames.length) {
        try {
          await ElMessageBox.confirm(
            `以下节点暂时不可用：${offlineNames.join('、')}。是否忽略这些节点并继续执行？`,
            '部分节点离线',
            {
              type: 'warning',
              confirmButtonText: '继续执行',
              cancelButtonText: '取消'
            }
          )
        } catch {
          return
        }
      }
    }

    let response
    try {
      response = await submitExecution(false)
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
            type: 'warning',
            confirmButtonText: '忽略并继续执行',
            cancelButtonText: '取消',
            dangerouslyUseHTMLString: true,
            customClass: 'environment-confirm-dialog'
          }
        )
      } catch {
        return
      }

      response = await submitExecution(true)
    }

    const newExecutionId = response.data?.id
    ElMessage.success('执行已启动，正在跳转详情页')
    if (newExecutionId) {
      router.push({ path: `/executions/${newExecutionId}` })
      return
    }
    router.push({ path: '/executions' })
  } catch (error) {
    console.error('启动执行失败:', error)
    ElMessage.error(error?.response?.data?.message || '启动执行失败')
  } finally {
    executing.value = false
  }
}

const goBack = () => {
  router.push('/scripts')
}

const goEdit = () => {
  if (!scriptId.value) return
  router.push(`/scripts/${scriptId.value}/edit`)
}

watch([executionMode, splitCSV, scriptId], () => {
  if (!scriptId.value) return
  loadScriptDependencies()
})

onMounted(async () => {
  await Promise.all([
    loadScriptDetail(),
    loadNodeResources(),
    loadMasterConfig(),
    loadScriptDependencies()
  ])
})
</script>

<style scoped lang="scss">
.script-execute-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.section-card {
  padding: 16px;
  border-radius: 18px;
  position: relative;
  overflow: hidden;
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.08), transparent 34%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.036), rgba(255, 255, 255, 0.014)),
    rgba(15, 23, 42, 0.54);
  border: 1px solid rgba(148, 163, 184, 0.12);
  box-shadow:
    0 22px 48px rgba(2, 8, 23, 0.16),
    inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.execute-hero {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.execute-hero-main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.execute-title-block {
  flex: 1;

  h1 {
    font-size: 28px;
    line-height: 1.15;
    margin-bottom: 8px;
    color: var(--text-primary);
  }

  p {
    max-width: 760px;
    color: var(--text-secondary);
    font-size: 14px;
  }
}

.execute-kicker,
.section-label {
  color: var(--accent-blue);
  letter-spacing: 0.18em;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}

.execute-hero-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.03);
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    border-color: rgba(0, 170, 255, 0.32);
    color: #ffffff;
  }
}

.hero-fact-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.hero-fact-card,
.config-card,
.summary-item,
.node-resource-card,
.selected-node-card,
.advice-item,
.slave-select-card {
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.06), transparent 42%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), rgba(255, 255, 255, 0.01)),
    rgba(15, 23, 42, 0.5);
  border: 1px solid rgba(148, 163, 184, 0.12);
  border-radius: 18px;
  box-shadow:
    0 14px 30px rgba(2, 8, 23, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.hero-fact-card {
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-height: 96px;
}

.hero-fact-label {
  color: var(--text-secondary);
  font-size: 12px;
}

.hero-fact-value {
  color: var(--text-primary);
  font-size: 20px;
  line-height: 1.25;
  word-break: break-word;
}

.hero-fact-desc {
  color: rgba(148, 163, 184, 0.82);
  font-size: 12px;
}

.execute-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.48fr) 356px;
  gap: 14px;
  align-items: start;
}

.execute-main,
.execute-side {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.execute-side {
  position: sticky;
  top: 96px;
  align-self: start;
}

.side-panel {
  position: static;
}

.side-panel--primary {
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.12), transparent 34%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
    rgba(15, 23, 42, 0.54);
}

.configuration-panel {
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.12), transparent 34%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), rgba(255, 255, 255, 0.012)),
    rgba(15, 23, 42, 0.54);
}

.distributed-panel {
  background:
    radial-gradient(circle at top left, rgba(34, 197, 94, 0.11), transparent 36%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), rgba(255, 255, 255, 0.012)),
    rgba(15, 23, 42, 0.54);
}

.advice-panel {
  background:
    radial-gradient(circle at top left, rgba(245, 158, 11, 0.08), transparent 34%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), rgba(255, 255, 255, 0.012)),
    rgba(15, 23, 42, 0.54);
}

.panel-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.section-title {
  font-size: 17px;
  font-weight: 700;
  color: var(--text-primary);
  margin-top: 6px;
}

.section-desc {
  margin-top: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.65;
}

.panel-chip {
  display: inline-flex;
  align-items: center;
  height: 34px;
  padding: 0 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.panel-meta {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.mode-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.mode-card {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.05), transparent 44%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.025), rgba(255, 255, 255, 0.01)),
    rgba(15, 23, 42, 0.42);
  cursor: pointer;
  color: inherit;
  text-align: left;
  transition: all 0.22s ease;

  &:hover {
    transform: translateY(-1px);
    border-color: rgba(0, 170, 255, 0.24);
  }

  &.active {
    border-color: rgba(0, 170, 255, 0.42);
    background: linear-gradient(135deg, rgba(0, 102, 255, 0.14), rgba(0, 170, 255, 0.08));
    box-shadow: inset 0 0 0 1px rgba(0, 170, 255, 0.12);
  }
}

.mode-icon {
  width: 52px;
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  flex-shrink: 0;
  font-size: 22px;

  &.is-local {
    color: var(--accent-blue);
    background: rgba(0, 170, 255, 0.12);
  }

  &.is-distributed {
    color: var(--accent-green);
    background: rgba(0, 204, 106, 0.12);
  }
}

.mode-copy {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;

  strong {
    font-size: 18px;
    color: var(--text-primary);
  }

  span {
    color: var(--text-secondary);
    font-size: 13px;
    line-height: 1.65;
  }
}

.mode-check {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--accent-blue);
  color: #ffffff;
  flex-shrink: 0;
}

.config-grid,
.distributed-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
}

.config-card {
  padding: 14px;
  min-width: 0;
}

.config-card--wide {
  min-height: 100%;
}

.field-label {
  display: block;
  margin-bottom: 10px;
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
}

.option-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.option-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;

  strong {
    display: block;
    margin-bottom: 4px;
    color: var(--text-primary);
    font-size: 14px;
  }

  p {
    color: var(--text-secondary);
    font-size: 12px;
    line-height: 1.7;
  }
}

.selection-summary-card {
  display: flex;
  flex-direction: column;
  gap: 16px;

  p {
    color: var(--text-secondary);
    font-size: 12px;
    margin-top: 6px;
  }
}

.selection-summary-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.selection-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.selection-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.selection-metric {
  min-width: 0;
  padding: 12px 14px;
  border-radius: 14px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
    rgba(15, 23, 42, 0.44);
  border: 1px solid rgba(148, 163, 184, 0.12);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.02);

  span {
    display: block;
    color: var(--text-secondary);
    font-size: 11px;
    margin-bottom: 6px;
  }

  strong {
    color: var(--text-primary);
    font-size: 18px;
  }
}

.callback-preview {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;

  span {
    color: var(--text-secondary);
    font-size: 12px;
  }

  code {
    display: block;
    padding: 11px 12px;
    border-radius: 12px;
    background: rgba(6, 11, 20, 0.62);
    color: #d7e8ff;
    font-size: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.slave-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(208px, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.slave-select-card {
  width: 100%;
  padding: 16px;
  text-align: left;
  cursor: pointer;
  transition: all 0.2s ease;
  color: inherit;

  &:hover:not(.offline) {
    border-color: rgba(0, 170, 255, 0.2);
    transform: translateY(-1px);
    box-shadow:
      0 18px 32px rgba(2, 8, 23, 0.12),
      inset 0 1px 0 rgba(255, 255, 255, 0.04);
  }

  &.active {
    border-color: rgba(0, 170, 255, 0.45);
    background:
      radial-gradient(circle at top left, rgba(56, 189, 248, 0.14), transparent 44%),
      linear-gradient(135deg, rgba(0, 102, 255, 0.12), rgba(0, 170, 255, 0.06));
    box-shadow:
      0 0 0 1px rgba(56, 189, 248, 0.1),
      0 18px 36px rgba(2, 8, 23, 0.14);
  }

  &.offline {
    cursor: not-allowed;
    opacity: 0.5;
  }
}

.slave-select-header,
.selected-node-top,
.node-resource-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;

  strong {
    font-size: 16px;
    color: var(--text-primary);
  }
}

.slave-select-side {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.slave-address,
.node-resource-address {
  margin-top: 4px;
  color: var(--text-secondary);
  font-size: 12px;
  word-break: break-all;
}

.slave-status,
.node-role-pill,
.slave-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 12px;
  white-space: nowrap;
}

.slave-status.is-success,
.node-role-pill,
.node-role-pill.is-soft,
.slave-badge {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-secondary);
}

.slave-badge.is-success,
.node-env-badge.is-success {
  color: #4ade80;
  border-color: rgba(74, 222, 128, 0.2);
  background: rgba(74, 222, 128, 0.08);
}

.slave-badge.is-warning,
.node-env-badge.is-warning {
  color: #f8d27a;
  border-color: rgba(245, 158, 11, 0.2);
  background: rgba(245, 158, 11, 0.08);
}

.slave-badge.is-danger,
.node-env-badge.is-danger {
  color: #ff8e87;
  border-color: rgba(239, 68, 68, 0.2);
  background: rgba(239, 68, 68, 0.08);
}

.slave-status.is-warning {
  background: rgba(255, 149, 0, 0.12);
  color: #ffcc7a;
}

.slave-status.is-danger {
  background: rgba(255, 69, 58, 0.12);
  color: #ff8e87;
}

.slave-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.slave-order-badge {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: rgba(56, 189, 248, 0.16);
  border: 1px solid rgba(56, 189, 248, 0.28);
  color: #7dd3fc;
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
}

.slave-select-footer {
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  color: var(--text-secondary);
  font-size: 12px;
}

.advice-stack {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 10px;
}

.advice-item {
  min-height: 138px;
  padding: 13px 14px;
  display: flex;
  gap: 14px;
  align-items: flex-start;
  min-width: 0;
  position: relative;
  overflow: hidden;

  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 14px;
    bottom: 14px;
    width: 3px;
    border-radius: 999px;
    background: rgba(56, 189, 248, 0.5);
  }

  &.is-warning {
    border-color: rgba(255, 149, 0, 0.18);
    background: linear-gradient(135deg, rgba(255, 149, 0, 0.07), rgba(15, 23, 42, 0.45));

    &::before {
      background: rgba(255, 149, 0, 0.72);
    }
  }

  &.is-danger {
    border-color: rgba(255, 69, 58, 0.18);
    background: linear-gradient(135deg, rgba(255, 69, 58, 0.08), rgba(15, 23, 42, 0.45));

    &::before {
      background: rgba(255, 99, 94, 0.78);
    }
  }
}

.advice-index {
  width: 34px;
  height: 34px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
}

.advice-copy {
  min-width: 0;
  flex: 1;

  .advice-copy-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    margin-bottom: 6px;
  }

  strong {
    font-size: 16px;
    color: var(--text-primary);
  }

  p {
    color: var(--text-secondary);
    font-size: 13px;
    line-height: 1.75;
  }
}

.advice-tag {
  min-width: 54px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}

.summary-hero-card {
  padding: 14px;
  border-radius: 18px;
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.14), transparent 38%),
    rgba(7, 13, 24, 0.58);
  border: 1px solid rgba(56, 189, 248, 0.16);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.summary-hero-title {
  color: var(--text-primary);
  font-size: 20px;
  font-weight: 700;
  line-height: 1.2;
}

.summary-hero-desc {
  margin-top: 8px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.75;
}

.summary-hero-actions {
  display: flex;
  gap: 10px;
  margin-top: 14px;
}

.summary-hero-actions > * {
  flex: 1;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin: 12px 0;
}

.summary-item,
.summary-block {
  padding: 12px 14px;
}

.summary-item {
  display: flex;
  flex-direction: column;
  gap: 6px;

  span {
    color: var(--text-secondary);
    font-size: 12px;
  }

  strong {
    color: var(--text-primary);
    font-size: 16px;
  }
}

.summary-block {
  margin-top: 12px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.05);

  code {
    display: block;
    margin-top: 8px;
    padding: 10px 12px;
    border-radius: 12px;
    background: rgba(6, 11, 20, 0.62);
    color: #d7e8ff;
    font-size: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.summary-block-label {
  color: var(--text-secondary);
  font-size: 12px;
}

.summary-block-value {
  margin-top: 8px;
  color: var(--text-primary);
  font-size: 15px;
  font-weight: 600;
  word-break: break-word;
}

.node-stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.node-resource-card,
.selected-node-card {
  padding: 12px;
}

.node-resource-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;

  &.compact {
    gap: 8px;
  }
}

.node-metric {
  min-width: 0;
  padding: 11px 12px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.05);

  span {
    display: block;
    color: var(--text-secondary);
    font-size: 11px;
    margin-bottom: 6px;
  }

  strong {
    color: var(--text-primary);
    font-size: 15px;
  }
}

.selected-node-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
}

.selected-node-summary-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;

  span {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 34px;
    padding: 0 10px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.04);
    color: var(--text-secondary);
    font-size: 12px;
    text-align: center;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.selected-node-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.selected-node-card {
  padding: 12px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.026), rgba(255, 255, 255, 0.01)),
    rgba(10, 17, 31, 0.58);
}

.selected-node-metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;

  span {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 30px;
    padding: 0 10px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.05);
    color: var(--text-secondary);
    font-size: 11px;
    white-space: nowrap;
  }
}

.selected-node-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.node-env-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;

  &.compact {
    margin-top: 8px;
  }
}

.node-env-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 30px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(148, 163, 184, 0.12);
  color: var(--text-secondary);
  font-size: 11px;
  white-space: nowrap;
}

.node-env-note {
  margin-top: 10px;
  color: #f8d27a;
  font-size: 12px;
  line-height: 1.6;
}

.remove-node-btn {
  border: none;
  background: transparent;
  color: #ff6b66;
  cursor: pointer;
  font-size: 13px;
  padding: 0;

  &:hover {
    color: #ff938f;
  }
}

.empty-panel {
  min-height: 112px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 18px;
  border-radius: 16px;
  border: 1px dashed rgba(255, 255, 255, 0.12);
  color: var(--text-secondary);
  font-size: 13px;
  text-align: center;

  &.compact {
    min-height: 80px;
  }
}

.execute-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  position: sticky;
  bottom: 16px;
  z-index: 5;
  backdrop-filter: blur(16px);
  margin-top: 16px;
  padding: 12px 14px;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  background:
    linear-gradient(180deg, rgba(10, 18, 30, 0.86), rgba(8, 14, 26, 0.9));
  box-shadow: 0 16px 36px rgba(2, 8, 23, 0.24);
}

.execute-footer-left {
  min-width: 0;
}

.execute-footer-title {
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 700;
}

.execute-footer-desc {
  margin-top: 6px;
  color: var(--text-secondary);
  font-size: 13px;
}

.execute-footer-actions {
  display: flex;
  gap: 12px;
  flex-shrink: 0;
}

:deep(.el-input__wrapper),
:deep(.el-textarea__inner),
:deep(.el-select__wrapper) {
  background: rgba(255, 255, 255, 0.04);
  border-radius: 14px;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.06);
}

:deep(.el-textarea__inner) {
  min-height: 118px !important;
}

:deep(.environment-confirm-dialog .el-message-box__message) {
  max-height: 58vh;
  overflow: auto;
}

:deep(.env-confirm-shell) {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.7;
}

:deep(.env-warning-list) {
  margin: 12px 0 0 18px;
}

:deep(.env-diff-list) {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

:deep(.env-diff-row) {
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.04);
}

:deep(.env-diff-title) {
  color: var(--text-primary);
  font-weight: 600;
}

:deep(.env-diff-node) {
  margin-top: 4px;
  color: var(--accent-blue);
  font-size: 12px;
}

:deep(.env-confirm-foot) {
  margin-top: 14px;
  color: #ffcc7a;
}

@media (max-width: 1500px) {
  .execute-layout {
    grid-template-columns: minmax(0, 1fr);
  }

  .execute-side {
    position: static;
  }
}

@media (max-width: 1080px) {
  .hero-fact-grid,
  .summary-grid,
  .mode-grid,
  .config-grid,
  .distributed-grid,
  .selection-metrics {
    grid-template-columns: minmax(0, 1fr);
  }

  .execute-hero-main,
  .selection-summary-top,
  .execute-footer {
    flex-direction: column;
    align-items: stretch;
  }

  .execute-hero-actions,
  .execute-footer-actions,
  .summary-hero-actions {
    justify-content: stretch;
  }

  .execute-footer-actions > *,
  .summary-hero-actions > * {
    flex: 1;
  }
}

@media (max-width: 760px) {
  .panel-heading,
  .slave-select-header,
  .selected-node-top,
  .node-resource-header,
  .option-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .slave-card-grid,
  .node-resource-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .advice-copy-top,
  .slave-select-footer,
  .selected-node-summary-strip {
    flex-direction: column;
    align-items: flex-start;
  }

  .selected-node-card .node-resource-grid.compact {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .selected-node-summary-strip {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
