<template>
  <div class="execution-list-page">
    <section class="workspace-hero">
      <div class="workspace-hero-main">
        <div class="workspace-copy">
          <div class="workspace-kicker">WORKSPACE</div>
          <h1>执行记录管理</h1>
          <p>统一查看运行状态、基准对比和关键性能指标，运行中的任务、已完成结果与后续回溯都在同一条操作链路里完成。</p>
        </div>
        <div class="workspace-hero-pills">
          <span class="workspace-pill">总执行 {{ stats.total }}</span>
          <span class="workspace-pill">运行中 {{ stats.running }}</span>
          <span class="workspace-pill">失败 {{ stats.failed }}</span>
        </div>
      </div>
    </section>

    <!-- 统计卡片区域 -->
    <div class="stats-section" v-loading="statsLoading">
      <button
        type="button"
        class="stat-card"
        :class="{ active: filters.status === '' }"
        :aria-pressed="filters.status === ''"
        aria-label="查看全部执行记录"
        @click="filterByStatus('')"
      >
        <div class="stat-icon total">
          <el-icon><Document /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.total }}</div>
          <div class="stat-label">总执行数</div>
        </div>
      </button>
      <button
        type="button"
        class="stat-card"
        :class="{ active: filters.status === 'running' }"
        :aria-pressed="filters.status === 'running'"
        aria-label="筛选运行中的执行记录"
        @click="filterByStatus('running')"
      >
        <div class="stat-icon running">
          <el-icon><Loading /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.running }}</div>
          <div class="stat-label">运行中</div>
        </div>
      </button>
      <button
        type="button"
        class="stat-card"
        :class="{ active: filters.status === 'success' }"
        :aria-pressed="filters.status === 'success'"
        aria-label="筛选已完成的执行记录"
        @click="filterByStatus('success')"
      >
        <div class="stat-icon completed">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.completed }}</div>
          <div class="stat-label">已完成</div>
        </div>
      </button>
      <button
        type="button"
        class="stat-card"
        :class="{ active: filters.status === 'failed' }"
        :aria-pressed="filters.status === 'failed'"
        aria-label="筛选失败的执行记录"
        @click="filterByStatus('failed')"
      >
        <div class="stat-icon failed">
          <el-icon><CircleClose /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.failed }}</div>
          <div class="stat-label">失败</div>
        </div>
      </button>
      <button
        type="button"
        class="stat-card"
        :class="{ active: filters.status === 'stopped' }"
        :aria-pressed="filters.status === 'stopped'"
        aria-label="筛选已停止的执行记录"
        @click="filterByStatus('stopped')"
      >
        <div class="stat-icon stopped">
          <el-icon><VideoPause /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.stopped }}</div>
          <div class="stat-label">已停止</div>
        </div>
      </button>
    </div>

    <!-- 筛选区域 -->
    <div class="section-card filter-section">
      <div class="filter-head">
        <div class="section-label">FILTERS</div>
        <div class="filter-head-copy">按脚本、状态、日期和备注快速定位执行记录，运行中的任务会自动续刷。</div>
      </div>
      <div class="filter-bar">
        <el-select
          v-model="filters.script_id"
          placeholder="选择脚本"
          clearable
          class="filter-select"
        >
          <el-option
            v-for="script in scripts"
            :key="script.id"
            :label="script.name"
            :value="script.id"
          />
        </el-select>

        <el-select
          v-model="filters.status"
          placeholder="执行状态"
          clearable
          class="filter-select"
        >
          <el-option label="运行中" value="running" />
          <el-option label="已完成" value="success" />
          <el-option label="失败" value="failed" />
          <el-option label="已停止" value="stopped" />
        </el-select>

        <el-date-picker
          v-model="dateRange"
          type="daterange"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          format="YYYY-MM-DD"
          value-format="YYYY-MM-DD"
          class="filter-date"
        />

        <el-input
          v-model="filters.keyword"
          placeholder="搜索备注..."
          clearable
          class="filter-input"
          @clear="handleSearch"
          @keyup.enter="handleSearch"
        />

        <div class="filter-buttons">
          <el-button type="primary" @click="handleSearch">
            <el-icon><Search /></el-icon>
            查询
          </el-button>
          <el-button @click="resetFilters">
            <el-icon><RefreshRight /></el-icon>
            重置
          </el-button>
        </div>
      </div>
    </div>

    <!-- 执行记录区域 -->
    <div class="section-card">
      <div class="section-header-with-action">
        <div class="section-header">
          <div class="section-label">EXECUTIONS</div>
          <div class="section-title">执行记录</div>
          <div class="section-desc">聚焦运行状态、基准线和核心指标，常用操作在一行里就能完成，不需要反复切换页面。</div>
        </div>
        <div class="section-actions">
          <el-button 
            :disabled="selectedExecutions.length !== 2" 
            type="primary" 
            @click="openCompareDialog"
            class="compare-btn"
          >
            <el-icon><TrendCharts /></el-icon>
            对比 ({{ selectedExecutions.length }}/2)
          </el-button>
          <div v-if="hasRunning" class="auto-refresh-indicator">
            <span class="refresh-dot"></span>
            <span class="refresh-text">自动刷新中</span>
          </div>
          <el-button @click="fetchExecutions" class="refresh-btn">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </div>

      <div class="list-utility-bar">
        <span class="utility-chip">共 {{ pagination.total }} 条记录</span>
        <span class="utility-chip" v-if="hasRunning">运行中任务会自动轮询刷新</span>
        <span class="utility-chip" v-if="selectedExecutions.length">已选 {{ selectedExecutions.length }} 条记录</span>
      </div>

      <!-- 执行记录表格 -->
      <div class="execution-table-shell">
        <el-table
          ref="tableRef"
          v-loading="tableLoading"
          :data="executionList"
          class="executions-table"
          stripe
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="42" align="center" />
        <el-table-column label="#" width="60" align="center">
          <template #default="{ $index }">
            <span class="index-text">{{ (pagination.page - 1) * pagination.page_size + $index + 1 }}</span>
          </template>
        </el-table-column>

        <el-table-column label="脚本名称" min-width="176" sortable prop="script_name" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="script-name-cell">
              <el-icon class="script-icon"><Document /></el-icon>
              <div class="script-name-stack">
                <span class="script-name">{{ row.script_name }}</span>
                <span class="script-id">#{{ row.id }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="132" align="center">
          <template #default="{ row }">
            <el-tooltip :content="getStatusText(row)" placement="top">
              <el-tag
                :type="getStatusType(row)"
                size="small"
                class="status-tag"
              >
                {{ getStatusShortText(row) }}
              </el-tag>
            </el-tooltip>
          </template>
        </el-table-column>

        <el-table-column label="备注" width="54" align="center">
          <template #default="{ row }">
            <el-tooltip
              :content="row.remarks || '暂无备注'"
              placement="top"
              :disabled="!row.remarks"
              :show-after="220"
            >
              <button
                type="button"
                class="remarks-trigger"
                :class="{ empty: !row.remarks }"
                :aria-label="row.remarks ? `查看备注：${row.remarks}` : '暂无备注'"
              >
                <el-icon><ChatDotRound /></el-icon>
              </button>
            </el-tooltip>
          </template>
        </el-table-column>

        <el-table-column label="开始时间" width="156" sortable prop="created_at">
          <template #default="{ row }">
            <span class="time-text">{{ formatDateTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="执行时长" width="108" align="right" header-align="right" sortable prop="duration">
          <template #default="{ row }">
            <span class="duration-text">{{ formatDuration(getDurationSeconds(row)) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="样本数" width="94" align="right" header-align="right" sortable :sort-method="(a, b) => getSummaryField(a, 'total_samples') - getSummaryField(b, 'total_samples')">
          <template #default="{ row }">
            <span class="metric-value metric-blue">{{ formatNumber(getSummaryField(row, 'total_samples')) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="平均RT" width="116" align="right" header-align="right" sortable :sort-method="(a, b) => getResponseTime(a) - getResponseTime(b)">
          <template #default="{ row }">
            <div class="metric-with-unit">
              <span
                class="metric-value"
                :class="{ 'metric-orange': getResponseTime(row) > 1000 }"
              >
                {{ formatNumber(getResponseTime(row)) }}
              </span>
              <span class="unit">ms</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="TPS / 请求率" width="144" align="right" header-align="right" sortable :sort-method="(a, b) => getThroughput(a) - getThroughput(b)">
          <template #default="{ row }">
            <div class="metric-with-unit">
              <span class="metric-value metric-blue">{{ formatNumber(getThroughput(row)) }}</span>
              <span class="unit">{{ getThroughputUnit(row) }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="错误率" width="108" align="right" header-align="right" sortable :sort-method="(a, b) => getErrorRateNum(a) - getErrorRateNum(b)">
          <template #default="{ row }">
            <div class="metric-with-unit">
              <span
                class="metric-value"
                :class="{ 'metric-red': getErrorRate(row) > 5 }"
              >
                {{ getErrorRate(row) }}
              </span>
              <span class="unit">%</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column
          label="操作"
          fixed="right"
          width="156"
          align="center"
          header-align="center"
          class-name="action-column"
          label-class-name="action-column-header"
        >
          <template #default="{ row }">
            <div class="action-btns">
              <el-tooltip content="查看详情" placement="top">
                <el-button
                  text
                  type="primary"
                  @click="viewDetail(row.id)"
                  class="action-btn action-icon view-btn"
                  aria-label="查看详情"
                >
                  <el-icon><View /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip
                v-if="canToggleBaseline(row)"
                :content="row.is_baseline ? '取消基准线' : '设为基准线'"
                placement="top"
              >
                <el-button
                  text
                  :type="row.is_baseline ? 'warning' : 'default'"
                  @click="toggleBaseline(row)"
                  class="action-btn action-icon baseline-btn"
                  :aria-label="row.is_baseline ? '取消基准线' : '设为基准线'"
                >
                  <el-icon>
                    <StarFilled v-if="row.is_baseline" />
                    <Star v-else />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip
                v-if="row.status === 'running'"
                content="停止执行"
                placement="top"
              >
                <el-button
                  text
                  type="warning"
                  @click="handleStop(row)"
                  :loading="stoppingId === row.id"
                  :disabled="stoppingId === row.id"
                  class="action-btn action-icon stop-btn"
                  aria-label="停止执行"
                >
                  <el-icon v-if="stoppingId !== row.id"><VideoPause /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip
                v-if="row.status !== 'running'"
                content="删除记录"
                placement="top"
              >
                <el-button
                  text
                  type="danger"
                  @click="handleDelete(row)"
                  :loading="deletingId === row.id"
                  :disabled="deletingId === row.id"
                  class="action-btn action-icon delete-btn"
                  aria-label="删除记录"
                >
                  <el-icon v-if="deletingId !== row.id"><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
        </el-table>
      </div>

      <!-- 空状态 -->
      <div v-if="!tableLoading && executionList.length === 0" class="empty-state">
        <div class="empty-icon">
          <el-icon><DocumentDelete /></el-icon>
        </div>
        <h3 class="empty-title">暂无执行记录</h3>
        <p class="empty-desc">请在脚本列表中执行脚本</p>
      </div>

      <!-- 分页 -->
      <div class="pagination-wrapper" v-if="pagination.total > 0">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.page_size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>

    <!-- 对比弹窗 -->
    <el-dialog 
      v-model="compareDialogVisible" 
      title="执行对比" 
      width="720px"
      class="compare-dialog"
      destroy-on-close
    >
      <div class="compare-container" v-loading="compareLoading">
        <template v-if="compareResult">
          <!-- 顶部：两次执行的基本信息 -->
          <div class="compare-header">
            <div class="compare-col">
              <div class="compare-title">执行 #{{ compareResult?.execution1?.id }}</div>
              <div class="compare-info">{{ compareResult?.execution1?.start_time }}</div>
              <el-tag v-if="compareResult?.execution1?.is_baseline" type="warning" size="small" class="baseline-tag">
                <el-icon><StarFilled /></el-icon> 基准线
              </el-tag>
            </div>
            <div class="compare-vs">VS</div>
            <div class="compare-col">
              <div class="compare-title">执行 #{{ compareResult?.execution2?.id }}</div>
              <div class="compare-info">{{ compareResult?.execution2?.start_time }}</div>
              <el-tag v-if="compareResult?.execution2?.is_baseline" type="warning" size="small" class="baseline-tag">
                <el-icon><StarFilled /></el-icon> 基准线
              </el-tag>
            </div>
          </div>
          
          <!-- 指标对比列表 -->
          <div class="compare-metrics">
            <div 
              v-for="diff in compareResult?.differences" 
              :key="diff.metric" 
              class="metric-row"
            >
              <span class="metric-label">{{ diff.label }}</span>
              <span class="metric-value">{{ formatMetricValue(diff.value1, diff.unit) }}</span>
              <span 
                class="metric-arrow" 
                :class="{ improved: diff.improved, worsened: !diff.improved && diff.diff_pct !== 0 }"
              >
                <template v-if="diff.diff_pct > 0">
                  <span class="arrow-up">↑</span> +{{ diff.diff_pct.toFixed(1) }}%
                </template>
                <template v-else-if="diff.diff_pct < 0">
                  <span class="arrow-down">↓</span> {{ diff.diff_pct.toFixed(1) }}%
                </template>
                <template v-else>
                  <span class="arrow-flat">—</span> 0%
                </template>
              </span>
              <span class="metric-value">{{ formatMetricValue(diff.value2, diff.unit) }}</span>
            </div>
          </div>
        </template>
        <div v-else-if="!compareLoading" class="compare-empty">
          <el-icon :size="48"><DocumentDelete /></el-icon>
          <p>暂无对比数据</p>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  View,
  CircleClose, 
  Refresh, 
  Document, 
  ChatDotRound,
  DocumentDelete, 
  Search,
  RefreshRight,
  Loading,
  CircleCheck,
  VideoPause,
  Delete,
  Star,
  StarFilled,
  TrendCharts
} from '@element-plus/icons-vue'
import { executionApi } from '@/api/execution'
import { scriptApi } from '@/api/script'
import { formatDateTimeInShanghai, parseServerDateTime } from '@/utils/datetime'

const router = useRouter()
const route = useRoute()
const statsLoading = ref(false)
const tableLoading = ref(false)
const executionList = ref([])
const stoppingId = ref(null)
const deletingId = ref(null)

const refreshTimer = ref(null)
const clockTimer = ref(null)
const nowTick = ref(Date.now())
const liveMetricsMap = ref({})
const listRequestInFlight = ref(false)
const statsRequestInFlight = ref(false)
const pageVisible = ref(typeof document === 'undefined' ? true : document.visibilityState === 'visible')
let liveMetricsHydrationToken = 0
const LIST_REFRESH_INTERVAL = 3000

// el-table ref
const tableRef = ref(null)

// 选中的执行（用于对比）
const selectedExecutions = ref([])
// 存储选中的执行 ID（用于刷新后恢复选择状态）
const selectedIds = ref(new Set())

// 对比弹窗相关
const compareDialogVisible = ref(false)
const compareLoading = ref(false)
const compareResult = ref(null)

// 统计数据
const stats = ref({
  total: 0,
  running: 0,
  completed: 0,
  failed: 0,
  stopped: 0
})

// 从 URL 恢复筛选状态
const filters = ref({
  script_id: route.query.script_id ? Number(route.query.script_id) : '',
  status: route.query.status || '',
  keyword: route.query.keyword || ''
})
const dateRange = ref([
  route.query.start_date || '',
  route.query.end_date || ''
].filter(Boolean))
const pagination = ref({
  page: route.query.page ? Number(route.query.page) : 1,
  page_size: route.query.page_size ? Number(route.query.page_size) : 10,
  total: 0
})

// 脚本列表
const scripts = ref([])

// 计算是否有运行中的任务
const hasRunning = computed(() => {
  return executionList.value.some(item => item.status === 'running')
})

// 处理表格选择变化
const handleSelectionChange = (val) => {
  selectedExecutions.value = val
  selectedIds.value = new Set(val.map(item => item.id))
}

// 切换基准线
const toggleBaseline = async (row) => {
  try {
    const action = row.is_baseline ? 'unset' : 'set'
    await executionApi.setBaseline(row.id, action)
    ElMessage.success(row.is_baseline ? '已取消基准线' : '已设为基准线')
    fetchExecutions()
  } catch (err) {
    console.error('基准线操作失败:', err)
    ElMessage.error('操作失败')
  }
}

const canToggleBaseline = (row) => {
  return row?.status !== 'running'
}

// 打开对比弹窗
const openCompareDialog = async () => {
  if (selectedExecutions.value.length !== 2) {
    ElMessage.warning('请选择两条执行进行对比')
    return
  }
  compareDialogVisible.value = true
  compareLoading.value = true
  try {
    const id1 = selectedExecutions.value[0].id
    const id2 = selectedExecutions.value[1].id
    const res = await executionApi.compareExecutions(id1, id2)
    compareResult.value = res.data
  } catch (err) {
    console.error('对比失败:', err)
    ElMessage.error('获取对比数据失败')
  } finally {
    compareLoading.value = false
  }
}

// 格式化指标值
const formatMetricValue = (value, unit) => {
  if (value === null || value === undefined) return '-'
  const n = parseFloat(value)
  if (isNaN(n)) return '-'
  if (Number.isInteger(n)) return `${n.toLocaleString()} ${unit || ''}`
  return `${n.toFixed(2)} ${unit || ''}`
}

// 获取状态类型
const getStatusType = (status) => {
  const normalized = typeof status === 'object' ? (status?.status_tone || status?.status || 'info') : status
  const map = {
    running: 'primary',
    success: 'success',
    failed: 'danger',
    stopped: 'info',
    info: 'primary',
    warning: 'warning',
    danger: 'danger'
  }
  return map[normalized] || 'info'
}

// 获取状态显示文本
const getStatusText = (status) => {
  const normalized = typeof status === 'object' ? (status?.display_status || status?.status) : status
  const textMap = {
    running: '运行中',
    success: '已完成',
    completed_success: '完成(全部成功)',
    completed_with_errors: '完成(部分失败)',
    completed_all_failed: '完成(全部失败)',
    completed_no_samples: '完成(无有效样本)',
    process_failed: '执行失败',
    failed: '失败',
    stopped: '已停止'
  }
  return textMap[normalized] || normalized
}

const getStatusShortText = (status) => {
  const normalized = typeof status === 'object' ? (status?.display_status || status?.status) : status
  const textMap = {
    running: '运行中',
    success: '已完成',
    completed_success: '全成功',
    completed_with_errors: '部分失败',
    completed_all_failed: '全失败',
    completed_no_samples: '无样本',
    process_failed: '执行失败',
    failed: '失败',
    stopped: '已停止'
  }
  return textMap[normalized] || normalized
}

// 格式化日期时间
const formatDateTime = (dateStr) => {
  return formatDateTimeInShanghai(dateStr)
}

// 格式化执行时长
const formatDuration = (seconds) => {
  if (!seconds || seconds === 0) return '-'
  const sec = parseInt(seconds)
  if (sec < 60) return `${sec}s`
  const min = Math.floor(sec / 60)
  const remainSec = sec % 60
  return `${min}m ${remainSec}s`
}

const getDurationSeconds = (row) => {
  const stored = Number(row?.duration || 0)
  const startAt = parseServerDateTime(row.start_time || row.created_at)
  const endAt = parseServerDateTime(row.end_time)
  const summarySpanMs = Number(getSummaryField(row, 'sample_span_ms') || 0)

  if (row?.status !== 'running') {
    if (stored > 0) return stored
    if (startAt && endAt) {
      return Math.max(Math.floor((endAt.getTime() - startAt.getTime()) / 1000), 0)
    }
    if (summarySpanMs > 0) {
      return Math.max(Math.round(summarySpanMs / 1000), 0)
    }
    return stored
  }

  const liveDuration = Number(liveMetricsMap.value[row.id]?.duration_seconds || 0)
  if (liveDuration > 0) return liveDuration

  if (!startAt) return stored

  return Math.max(Math.floor((nowTick.value - startAt.getTime()) / 1000), stored, 0)
}

// 格式化数字
const formatNumber = (num) => {
  if (num === null || num === undefined || num === '') return '-'
  const n = parseFloat(num)
  if (isNaN(n)) return '-'
  if (Number.isInteger(n)) return n.toLocaleString()
  return n.toFixed(2).replace(/\.?0+$/, '')
}

// 从 summary_data 解析字段
const getSummaryField = (row, field) => {
  if (!row.summary_data) return null
  try {
    const summary = typeof row.summary_data === 'string' 
      ? JSON.parse(row.summary_data) 
      : row.summary_data
    return summary[field]
  } catch {
    return null
  }
}

// 获取错误率
const getErrorRate = (row) => {
  const rate = getSummaryField(row, 'error_rate')
  if (rate === null || rate === undefined) return '-'
  const num = parseFloat(rate)
  if (isNaN(num)) return '-'
  return num.toFixed(2)
}

// 获取错误率数值（用于排序）
const getErrorRateNum = (row) => {
  const rate = getSummaryField(row, 'error_rate')
  if (rate === null || rate === undefined) return 0
  const num = parseFloat(rate)
  return isNaN(num) ? 0 : num
}

// 获取响应时间
const getResponseTime = (row) => {
  if (row?.status === 'running') {
    const liveValue = liveMetricsMap.value[row.id]?.avg_rt
    if (liveValue !== null && liveValue !== undefined) {
      const liveNum = parseFloat(liveValue)
      if (!isNaN(liveNum)) return liveNum
    }
  }
  const time = getSummaryField(row, 'avg_response_time')
  if (time === null || time === undefined || time === '') return null
  const num = parseFloat(time)
  return isNaN(num) ? null : num
}

const getThroughput = (row) => {
  if (row?.status === 'running') {
    const live = liveMetricsMap.value[row.id] || {}
    const liveValue = live.current_primary_throughput ?? (live.has_transaction_samples ? live.current_tps : live.current_request_rate)
    if (liveValue !== null && liveValue !== undefined) {
      const liveNum = parseFloat(liveValue)
      if (!isNaN(liveNum)) return liveNum
    }
  }
  const hasTransactionSamples = Number(getSummaryField(row, 'transaction_samples') || 0) > 0
  const field = hasTransactionSamples ? 'transaction_tps' : 'request_rate'
  const value = getSummaryField(row, field) ?? getSummaryField(row, 'primary_throughput') ?? getSummaryField(row, 'throughput')
  if (value === null || value === undefined || value === '') return null
  const num = parseFloat(value)
  return isNaN(num) ? null : num
}

const getThroughputUnit = (row) => {
  if (row?.status === 'running') {
    const live = liveMetricsMap.value[row.id] || {}
    return live.primary_throughput_unit || (live.has_transaction_samples ? 'tps' : 'req/s')
  }
  return getSummaryField(row, 'primary_throughput_unit') || (Number(getSummaryField(row, 'transaction_samples') || 0) > 0 ? 'tps' : 'req/s')
}

const hydrateRunningMetrics = async (rows) => {
  const token = ++liveMetricsHydrationToken
  const runningRows = rows.filter(item => item.status === 'running')
  if (!runningRows.length) {
    liveMetricsMap.value = {}
    return
  }

  const results = await Promise.allSettled(
    runningRows.map(row => executionApi.getLiveMetrics(row.id).then(res => [row.id, res.data || {}]))
  )

  const nextMap = {}
  results.forEach((result) => {
    if (result.status === 'fulfilled') {
      const [id, metrics] = result.value
      nextMap[id] = metrics
    }
  })

  if (token === liveMetricsHydrationToken) {
    liveMetricsMap.value = nextMap
  }
}

// 获取执行统计
const fetchStats = async ({ background = false } = {}) => {
  if (statsRequestInFlight.value) return
  statsRequestInFlight.value = true
  if (!background) {
    statsLoading.value = true
  }
  try {
    const res = await executionApi.getStats()
    stats.value = res.data || { total: 0, running: 0, completed: 0, failed: 0, stopped: 0 }
  } catch (error) {
    console.error('获取执行统计失败:', error)
  } finally {
    statsRequestInFlight.value = false
    if (!background) {
      statsLoading.value = false
    }
  }
}

// 获取脚本列表（用于筛选下拉）
const fetchScripts = async () => {
  try {
    const res = await scriptApi.getList({ page: 1, page_size: 1000 })
    scripts.value = res.data?.list || []
  } catch (error) {
    console.error('获取脚本列表失败:', error)
  }
}

// 同步筛选条件到 URL
const syncFiltersToURL = () => {
  const query = {}
  if (filters.value.script_id) query.script_id = filters.value.script_id
  if (filters.value.status) query.status = filters.value.status
  if (filters.value.keyword) query.keyword = filters.value.keyword
  if (pagination.value.page > 1) query.page = pagination.value.page
  if (pagination.value.page_size !== 10) query.page_size = pagination.value.page_size
  if (dateRange.value && dateRange.value.length === 2) {
    query.start_date = dateRange.value[0]
    query.end_date = dateRange.value[1]
  }
  router.replace({ query })
}

// 获取执行列表
const fetchExecutions = async ({ background = false } = {}) => {
  if (listRequestInFlight.value) {
    return
  }

  listRequestInFlight.value = true
  if (!background) {
    tableLoading.value = true
  }
  try {
    const params = {
      page: pagination.value.page,
      page_size: pagination.value.page_size
    }
    // 添加筛选参数
    if (filters.value.script_id) {
      params.script_id = filters.value.script_id
    }
    if (filters.value.status) {
      params.status = filters.value.status
    }
    if (filters.value.keyword) {
      params.keyword = filters.value.keyword
    }
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_date = dateRange.value[0]
      params.end_date = dateRange.value[1]
    }

    const res = await executionApi.getList(params)
    executionList.value = res.data?.list || []
    pagination.value.total = res.data?.total || 0
    void hydrateRunningMetrics(executionList.value)

    // 恢复选中状态
    if (selectedIds.value.size > 0) {
      nextTick(() => {
        if (tableRef.value) {
          executionList.value.forEach(row => {
            if (selectedIds.value.has(row.id)) {
              tableRef.value.toggleRowSelection(row, true)
            }
          })
        }
      })
    }
  } catch (error) {
    console.error('获取执行列表失败:', error)
  } finally {
    listRequestInFlight.value = false
    if (!background) {
      tableLoading.value = false
    }
  }
}

// 搜索
const handleSearch = () => {
  pagination.value.page = 1
  syncFiltersToURL()
  fetchExecutions()
}

// 重置筛选
const resetFilters = () => {
  filters.value = {
    script_id: '',
    status: '',
    keyword: ''
  }
  dateRange.value = []
  pagination.value.page = 1
  pagination.value.page_size = 10
  router.replace({ query: {} })
  fetchExecutions()
}

// 通过状态筛选
const filterByStatus = (status) => {
  filters.value.status = status
  pagination.value.page = 1
  syncFiltersToURL()
  fetchExecutions()
}

// 处理分页大小变化
const handleSizeChange = (size) => {
  pagination.value.page_size = size
  pagination.value.page = 1
  syncFiltersToURL()
  fetchExecutions()
}

// 处理页码变化
const handlePageChange = (page) => {
  pagination.value.page = page
  syncFiltersToURL()
  fetchExecutions()
}

// 查看详情
const viewDetail = (id) => {
  router.push(`/executions/${id}`)
}

// 停止执行
const handleStop = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要停止执行 "${row.script_name}" 吗？`,
      '确认停止',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    stoppingId.value = row.id
    await executionApi.stop(row.id)
    ElMessage.success('停止命令已发送')
    fetchExecutions()
    fetchStats()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('停止执行失败:', error)
    }
  } finally {
    stoppingId.value = null
  }
}

// 删除执行记录
const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除执行记录 "${row.script_name}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    deletingId.value = row.id
    await executionApi.delete(row.id)
    ElMessage.success('删除成功')
    fetchExecutions()
    fetchStats()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除执行记录失败:', error)
    }
  } finally {
    deletingId.value = null
  }
}

// 设置自动刷新
const refreshRunningSnapshot = async () => {
  await fetchExecutions({ background: true })
  await fetchStats({ background: true })
}

const setupAutoRefresh = () => {
  if (refreshTimer.value) {
    clearTimeout(refreshTimer.value)
    refreshTimer.value = null
  }

  const tick = async () => {
    if (!pageVisible.value) {
      refreshTimer.value = window.setTimeout(tick, LIST_REFRESH_INTERVAL)
      return
    }

    if (hasRunning.value) {
      await refreshRunningSnapshot()
    }

    refreshTimer.value = window.setTimeout(tick, LIST_REFRESH_INTERVAL)
  }

  refreshTimer.value = window.setTimeout(tick, LIST_REFRESH_INTERVAL)
}

const setupClockTicker = () => {
  if (clockTimer.value) {
    clearInterval(clockTimer.value)
    clockTimer.value = null
  }

  clockTimer.value = setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)
}

const handleVisibilityChange = () => {
  pageVisible.value = document.visibilityState === 'visible'
  if (pageVisible.value && hasRunning.value) {
    refreshRunningSnapshot()
  }
}

onMounted(() => {
  fetchStats()
  fetchScripts()
  fetchExecutions()
  setupAutoRefresh()
  setupClockTicker()
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

// 监听 URL query 变化（从详情页返回时）
watch(() => route.query, (newQuery) => {
  filters.value.script_id = newQuery.script_id ? Number(newQuery.script_id) : ''
  filters.value.status = newQuery.status || ''
  filters.value.keyword = newQuery.keyword || ''
  pagination.value.page = newQuery.page ? Number(newQuery.page) : 1
  pagination.value.page_size = newQuery.page_size ? Number(newQuery.page_size) : 10
  if (newQuery.start_date && newQuery.end_date) {
    dateRange.value = [newQuery.start_date, newQuery.end_date]
  } else {
    dateRange.value = []
  }
}, { immediate: false })

onUnmounted(() => {
  if (refreshTimer.value) {
    clearTimeout(refreshTimer.value)
    refreshTimer.value = null
  }
  if (clockTimer.value) {
    clearInterval(clockTimer.value)
    clockTimer.value = null
  }
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<style scoped lang="scss">
.execution-list-page {
  padding: 6px 0 14px;
}

.workspace-hero {
  margin-bottom: 12px;
  padding: 16px 18px;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(148, 163, 184, 0.12);
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.12), transparent 32%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.015)),
    var(--bg-panel);
  box-shadow: 0 22px 48px rgba(2, 8, 23, 0.12);
}

.workspace-hero-main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
}

.workspace-copy {
  min-width: 0;

  h1 {
    margin: 6px 0 8px;
    color: var(--text-primary);
    font-size: 24px;
    line-height: 1.15;
  }

  p {
    max-width: 760px;
    color: var(--text-secondary);
    font-size: 13px;
    line-height: 1.6;
  }
}

.workspace-kicker {
  color: var(--accent-blue);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.workspace-hero-pills {
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
  font-size: 12px;
  font-weight: 600;
}

// 统计卡片区
.stats-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  min-height: 84px;
  padding: 14px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.015)),
    var(--bg-card-elevated);
  border-radius: var(--radius-lg);
  border: 1px solid rgba(148, 163, 184, 0.14);
  box-shadow: 0 18px 40px rgba(2, 8, 23, 0.14);
  cursor: pointer;
  transition: all 0.25s ease;
  appearance: none;
  text-align: left;
  color: inherit;
  font: inherit;

  &:hover {
    border-color: rgba(255, 255, 255, 0.12);
    transform: translateY(-2px);
  }

  &:focus-visible {
    outline: 2px solid rgba(54, 191, 250, 0.85);
    outline-offset: 2px;
  }

  &.active {
    border-color: rgba(54, 191, 250, 0.4);
    background:
      linear-gradient(180deg, rgba(0, 170, 255, 0.09), rgba(0, 170, 255, 0.03)),
      var(--bg-card);
    box-shadow: inset 0 0 0 1px rgba(54, 191, 250, 0.12);
  }
}

.stat-icon {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  font-size: 20px;

  &.total {
    background: rgba(0, 102, 255, 0.1);
    color: var(--accent-blue);
  }

  &.running {
    background: rgba(0, 102, 255, 0.1);
    color: var(--accent-blue);
  }

  &.completed {
    background: rgba(0, 204, 106, 0.1);
    color: var(--accent-green);
  }

  &.failed {
    background: rgba(255, 69, 58, 0.1);
    color: var(--accent-red);
  }

  &.stopped {
    background: rgba(148, 163, 184, 0.1);
    color: var(--text-secondary);
  }
}

.stat-content {
  flex: 1;
}

.stat-content .stat-value {
  font-size: 24px;
  font-weight: 700;
  font-family: 'Consolas', 'Monaco', monospace;
  color: var(--text-primary);
  line-height: 1.2;
}

.stat-content .stat-label {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
}

// 筛选区域
.filter-section {
  padding: 14px 16px;
  margin-bottom: 12px;
  background:
    linear-gradient(180deg, rgba(56, 189, 248, 0.04), rgba(255, 255, 255, 0.015)),
    var(--bg-panel);
}

.filter-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}

.filter-head-copy {
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.6;
}

.filter-bar {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(160px, 0.9fr) minmax(320px, 1.45fr) minmax(220px, 1fr) auto;
  gap: 10px;
  align-items: center;
}

.filter-select {
  width: 100%;
  min-width: 0;
}

.filter-date {
  width: 100%;
  min-width: 0;
}

.filter-input {
  width: 100%;
  min-width: 0;
}

.filter-buttons {
  display: flex;
  gap: 8px;
  margin-left: 0;
  justify-content: flex-end;
  flex-wrap: wrap;
}

// 区域卡片
.section-card {
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.015)),
    var(--bg-panel);
  border-radius: var(--radius-lg);
  border: 1px solid rgba(148, 163, 184, 0.12);
  padding: 16px;
  box-shadow: 0 22px 48px rgba(2, 8, 23, 0.12);
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
  max-width: 720px;
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
  margin-bottom: 12px;

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
  }
}

// 自动刷新指示器
.auto-refresh-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  background: rgba(0, 102, 255, 0.1);
  border: 1px solid rgba(0, 102, 255, 0.2);
  border-radius: var(--radius-full);
}

.refresh-dot {
  width: 6px;
  height: 6px;
  background: var(--accent-blue);
  border-radius: 50%;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.refresh-text {
  font-size: 13px;
  color: var(--accent-blue);
}

.list-utility-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 8px;
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
  font-size: 12px;
  font-weight: 600;
}

.refresh-btn {
  border-radius: var(--radius-md);
  padding: 8px 16px;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: var(--text-primary);

  .el-icon {
    margin-right: 6px;
  }
}

.execution-table-shell {
  overflow-x: auto;
  overflow-y: hidden;
  padding: 6px;
  border-radius: 16px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.01)),
    rgba(15, 23, 42, 0.52);
  border: 1px solid rgba(148, 163, 184, 0.08);
}

// 执行记录表格
.executions-table {
  background: transparent;
  border-radius: var(--radius-lg);
  overflow: hidden;
  min-width: 1288px;
  border: 1px solid rgba(148, 163, 184, 0.08);

  :deep(.el-table__fixed-right::before) {
    display: none;
  }

  :deep(.el-table__body-wrapper) {
    scrollbar-width: thin;
  }

  :deep(.el-table__header-wrapper) {
    th.el-table__cell {
      background:
        linear-gradient(180deg, rgba(255, 255, 255, 0.045), rgba(255, 255, 255, 0.018)) !important;
      color: var(--text-secondary) !important;
      font-weight: 500 !important;
      font-size: 13px !important;
      border-bottom: 1px solid rgba(255, 255, 255, 0.06) !important;
      height: 46px;
    }
  }

  :deep(.el-table__body-wrapper) {
    background-color: var(--bg-card);

    td.el-table__cell {
      border-bottom: 1px solid rgba(255, 255, 255, 0.04) !important;
      color: var(--text-primary) !important;
      padding-top: 10px;
      padding-bottom: 10px;
    }
  }

  :deep(.el-table__row) {
    background-color: var(--bg-card);
    transition: background-color 0.2s ease, box-shadow 0.2s ease;

    &:hover {
      background:
        linear-gradient(90deg, rgba(56, 189, 248, 0.03), rgba(255, 255, 255, 0.015)) !important;
    }
  }

  :deep(.action-column-header .cell) {
    justify-content: center;
    text-align: center;
  }

  :deep(.action-column .cell) {
    padding-left: 6px;
    padding-right: 6px;
  }

  .script-name-cell {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;

    .script-icon {
      font-size: 18px;
      color: var(--accent-blue);
      flex-shrink: 0;
    }
  }

  .script-name-stack {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .script-name {
      color: var(--text-primary);
      font-weight: 600;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
  }

  .script-id {
    color: var(--text-secondary);
    font-size: 11px;
    font-family: 'Consolas', 'Monaco', monospace;
  }

  .index-text {
    color: var(--text-secondary);
    font-size: 13px;
  }

  .time-text {
    color: var(--text-secondary);
    font-size: 13px;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  }

  .duration-text {
    color: var(--text-primary);
    font-family: 'Consolas', 'Monaco', monospace;
    font-weight: 500;
    font-size: 13px;
  }

  .remarks-text {
    display: none;
  }

  .status-tag {
    font-weight: 600;
    max-width: 112px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    padding-inline: 10px;
    border-radius: 999px;
    font-size: 12px;
    min-height: 28px;
  }

  .metric-with-unit {
    display: flex;
    align-items: baseline;
    justify-content: flex-end;
    gap: 4px;
    min-width: 0;
  }

  .metric-value {
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-weight: 600;
    color: var(--text-primary);
  }

  .metric-blue {
    color: var(--accent-blue);
  }

  .metric-orange {
    color: var(--accent-orange);
  }

  .metric-red {
    color: var(--accent-red);
  }

  .unit {
    font-size: 11px;
    color: var(--text-secondary);
    font-weight: 400;
  }

  .remarks-trigger {
    width: 30px;
    height: 30px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 999px;
    border: 1px solid rgba(148, 163, 184, 0.12);
    background: rgba(255, 255, 255, 0.04);
    color: #f8fafc;
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      border-color: rgba(56, 189, 248, 0.22);
      color: #38bdf8;
      background: rgba(56, 189, 248, 0.08);
    }

    &.empty {
      color: var(--text-secondary);
      opacity: 0.55;
    }
  }

  .action-btns {
    display: flex;
    justify-content: center;
    gap: 4px;
    flex-wrap: nowrap;
    overflow: visible;
    padding: 4px 6px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.025);
    border: 1px solid rgba(255, 255, 255, 0.05);

    .action-btn {
      margin: 0;
      min-width: auto;
      white-space: nowrap;
      border-radius: 999px;

      .el-icon {
        font-size: 14px;
      }
    }

    .action-icon {
      width: 30px;
      height: 30px;
      padding: 0;
      border-radius: 999px;
      background: rgba(255, 255, 255, 0.045);
      border: 1px solid rgba(255, 255, 255, 0.06);
      transition: transform 0.18s ease, border-color 0.18s ease, background-color 0.18s ease;

      .el-icon {
        margin-right: 0;
        font-size: 13px;
      }

      &:hover {
        transform: translateY(-1px);
      }
    }

    .view-btn {
      color: var(--accent-blue);
      background: rgba(56, 189, 248, 0.08);
      border-color: rgba(56, 189, 248, 0.14);
    }

    .stop-btn {
      color: var(--accent-orange);
    }

    .delete-btn {
      color: var(--accent-red) !important;
    }
    
    .delete-btn:hover {
      color: #ff5c52 !important;
    }

    .baseline-btn {
      color: #eab308;
      background: rgba(234, 179, 8, 0.06);
      border-color: rgba(234, 179, 8, 0.14);
    }

    .baseline-btn:hover {
      color: #facc15;
      background: rgba(234, 179, 8, 0.12);
    }
  }
}

// 响应式
@media (max-width: 1400px) {
  .filter-bar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .workspace-hero-main {
    flex-direction: column;
  }

  .workspace-hero-pills {
    justify-content: flex-start;
  }

  .filter-buttons {
    grid-column: 1 / -1;
    justify-content: flex-start;
  }
}

@media (max-width: 1024px) {
  .filter-bar {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .workspace-hero {
    padding: 18px;
  }

  .workspace-copy h1 {
    font-size: 24px;
  }

  .filter-head {
    align-items: flex-start;
  }

  .section-card {
    padding: 18px;
  }

  .list-utility-bar {
    margin-bottom: 12px;
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

// 分页
.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 18px;
}

// 对比按钮
.compare-btn {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  border: none;
  
  &:hover:not(:disabled) {
    background: linear-gradient(135deg, #4d8aff, #3b6fdb);
  }
  
  &:disabled {
    opacity: 0.6;
    background: linear-gradient(135deg, #64748b, #475569);
  }
}

// 对比弹窗样式
.compare-dialog {
  :deep(.el-dialog) {
    background: var(--bg-card);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 16px;
  }
  
  :deep(.el-dialog__header) {
    margin-right: 0;
    padding: 20px 24px 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    
    .el-dialog__title {
      color: var(--text-primary);
      font-weight: 600;
    }
  }
  
  :deep(.el-dialog__body) {
    padding: 24px;
  }
}

.compare-container {
  min-height: 200px;
}

.compare-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.compare-vs {
  font-size: 18px;
  font-weight: bold;
  color: rgba(255, 255, 255, 0.5);
  padding: 0 16px;
}

.compare-col {
  text-align: center;
  flex: 1;
  
  .compare-title {
    font-size: 16px;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.9);
    margin-bottom: 6px;
  }
  
  .compare-info {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.5);
    margin-bottom: 8px;
  }
  
  .baseline-tag {
    background: rgba(234, 179, 8, 0.15);
    border-color: rgba(234, 179, 8, 0.3);
    color: #eab308;
  }
}

.compare-metrics {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.metric-row {
  display: flex;
  align-items: center;
  padding: 14px 16px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.04);
  
  .metric-label {
    flex: 0 0 120px;
    color: rgba(255, 255, 255, 0.7);
    font-size: 14px;
  }
  
  .metric-value {
    flex: 1;
    text-align: center;
    color: rgba(255, 255, 255, 0.9);
    font-weight: 500;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 15px;
  }
  
  .metric-arrow {
    flex: 0 0 100px;
    text-align: center;
    font-weight: 600;
    font-size: 14px;
    color: rgba(255, 255, 255, 0.5);
    
    .arrow-up {
      color: #22c55e;
    }
    
    .arrow-down {
      color: #ef4444;
    }
    
    .arrow-flat {
      color: rgba(255, 255, 255, 0.3);
    }
    
    &.improved {
      color: #22c55e;
    }
    
    &.worsened {
      color: #ef4444;
    }
  }
}

.compare-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: rgba(255, 255, 255, 0.4);
  
  p {
    margin-top: 12px;
    font-size: 14px;
  }
}

// 基准线行高亮
:deep(.el-table__row.baseline-row) {
  background: rgba(234, 179, 8, 0.06) !important;
  
  td:first-child {
    border-left: 3px solid #eab308;
  }
}
</style>
