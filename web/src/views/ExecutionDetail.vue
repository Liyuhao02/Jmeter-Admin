<template>
  <div class="execution-detail-page">
    <!-- 顶部信息栏 -->
    <div class="section-card header-section">
      <div class="header-content">
        <div class="header-left">
          <button class="back-btn" @click="goBack">
            <el-icon><ArrowLeft /></el-icon>
            <span>返回</span>
          </button>
          <div class="script-info">
            <h1 class="script-name">{{ execution.script_name || '执行详情' }}</h1>
            <el-tag
              :type="getStatusType(execution)"
              size="small"
              class="status-tag"
            >
              {{ getStatusText(execution) }}
            </el-tag>
          </div>
        </div>
        <div class="header-right">
          <span class="execution-time" v-if="execution.created_at">
            <el-icon><Clock /></el-icon>
            {{ formatDateTime(execution.created_at) }}
          </span>
          <!-- 导出下拉按钮组 - 仅在非运行状态时显示 -->
          <el-dropdown 
            v-if="execution.status !== 'running'" 
            trigger="click"
            class="export-dropdown"
          >
            <el-button type="primary">
              <el-icon><Download /></el-icon>
              导出
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu class="export-menu">
                <el-dropdown-item 
                  @click="downloadJTL"
                  :disabled="!hasResultFile"
                >
                  <el-icon><Document /></el-icon>
                  <span>导出 JTL 文件</span>
                </el-dropdown-item>
                <el-dropdown-item 
                  @click="downloadReport"
                  :disabled="!hasReportDir"
                >
                  <el-icon><FolderOpened /></el-icon>
                  <span>导出 HTML 报告</span>
                </el-dropdown-item>
                <el-dropdown-item 
                  @click="downloadErrors"
                  :disabled="!hasErrors"
                >
                  <el-icon><DocumentCopy /></el-icon>
                  <span>导出错误记录 (CSV)</span>
                </el-dropdown-item>
                <el-dropdown-item divided @click="downloadAll">
                  <el-icon><Files /></el-icon>
                  <span>导出完整结果 (ZIP)</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button
            v-if="execution.status === 'running'"
            type="danger"
            @click="handleStop"
            :loading="stopping"
          >
            <el-icon v-if="!stopping"><CircleClose /></el-icon>
            停止执行
          </el-button>
        </div>
      </div>
    </div>

    <!-- 执行概览 -->
    <div class="section-card">
      <div class="section-header">
        <div class="section-label">OVERVIEW</div>
        <div class="section-title">执行概览</div>
      </div>
      
      <!-- 基准线对比卡片 -->
      <div 
        v-if="baselineComparison && !execution.is_baseline" 
        class="baseline-compare-card"
        v-loading="baselineLoading"
      >
        <div class="baseline-header">
          <div class="baseline-title">
            <el-icon><TrendCharts /></el-icon>
            <span>与基准线对比</span>
            <el-tag type="warning" size="small" class="baseline-tag">
              <el-icon><StarFilled /></el-icon> 基准: #{{ baselineComparison.execution1.id }}
            </el-tag>
          </div>
        </div>
        <div class="baseline-metrics">
          <div 
            v-for="diff in baselineComparison.differences" 
            :key="diff.metric" 
            class="baseline-metric-item"
          >
            <span class="metric-name">{{ diff.label }}</span>
            <span 
              class="metric-change" 
              :class="{ improved: diff.improved, worsened: !diff.improved && diff.diff_pct !== 0 }"
            >
              <span v-if="diff.diff_pct > 0" class="arrow-up">▲</span>
              <span v-else-if="diff.diff_pct < 0" class="arrow-down">▼</span>
              <span v-else class="arrow-flat">—</span>
              {{ Math.abs(diff.diff_pct).toFixed(1) }}%
            </span>
          </div>
        </div>
      </div>
      
      <div class="overview-panel">
        <div class="overview-hero">
          <div class="overview-status-strip">
            <span class="overview-status-dot" :class="`is-${overviewStatusTone}`"></span>
            <span class="overview-status-text">{{ getStatusText(execution) }}</span>
            <span class="overview-status-divider"></span>
            <span class="overview-status-note">{{ overviewStatusNote }}</span>
          </div>
          <div class="overview-primary-grid">
            <div
              v-for="metric in overviewPrimaryMetrics"
              :key="metric.key"
              class="overview-primary-card"
              :class="`is-${metric.tone}`"
            >
              <div class="overview-card-label">{{ metric.label }}</div>
              <div class="overview-card-value">{{ metric.value }}</div>
              <div class="overview-card-caption">{{ metric.caption }}</div>
            </div>
          </div>
        </div>
        <div class="overview-mini-grid">
          <div
            v-for="metric in overviewMiniMetrics"
            :key="metric.key"
            class="overview-mini-card"
          >
            <div class="overview-mini-label">{{ metric.label }}</div>
            <div class="overview-mini-value" :class="`text-${metric.color}`">
              {{ metric.value }}
            </div>
            <div class="overview-mini-caption">{{ metric.caption }}</div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="diagnosticCards.length || diagnosticWarnings.length" class="section-card">
      <div class="section-header">
        <div class="section-label">DIAGNOSTICS</div>
        <div class="section-title">执行诊断</div>
      </div>
      <div class="diagnostic-grid" v-if="diagnosticCards.length">
        <div v-for="card in diagnosticCards" :key="card.key" class="diagnostic-card">
          <div class="diagnostic-label">{{ card.label }}</div>
          <div class="diagnostic-value" :class="`text-${card.color}`">{{ card.value }}</div>
          <div class="diagnostic-caption">{{ card.caption }}</div>
        </div>
      </div>
      <div v-if="diagnosticWarnings.length" class="diagnostic-warning-stack">
        <div v-for="warning in diagnosticWarnings" :key="warning" class="diagnostic-warning-item">
          <el-icon><WarningFilled /></el-icon>
          <span>{{ warning }}</span>
        </div>
      </div>
    </div>

    <div v-if="executionConclusion" class="section-card">
      <div class="section-header">
        <div class="section-label">CONCLUSION</div>
        <div class="section-title">执行结论</div>
      </div>
      <div class="conclusion-panel" :class="`is-${executionConclusion.level || 'info'}`">
        <div class="conclusion-main">
          <div class="conclusion-title-row">
            <span class="conclusion-badge" :class="`is-${executionConclusion.level || 'info'}`">
              {{ conclusionLevelText }}
            </span>
            <span class="conclusion-title">{{ executionConclusion.title }}</span>
          </div>
          <div class="conclusion-summary">{{ executionConclusion.summary }}</div>
        </div>
        <div class="conclusion-grid">
          <div class="conclusion-list-card">
            <div class="conclusion-list-title">关键观察</div>
            <div v-for="item in conclusionHighlights" :key="item" class="conclusion-list-item">
              {{ item }}
            </div>
          </div>
          <div class="conclusion-list-card">
            <div class="conclusion-list-title">建议动作</div>
            <div v-for="item in conclusionRecommendations" :key="item" class="conclusion-list-item">
              {{ item }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="timelineStages.length" class="section-card">
      <div class="section-header">
        <div class="section-label">TIMELINE</div>
        <div class="section-title">执行链路</div>
      </div>
      <div class="timeline-grid">
        <div v-for="stage in timelineStages" :key="stage.key" class="timeline-card" :class="`is-${stage.tone}`">
          <div class="timeline-step">{{ stage.step }}</div>
          <div class="timeline-name">{{ stage.name }}</div>
          <div class="timeline-time">{{ stage.time }}</div>
          <div class="timeline-desc">{{ stage.description }}</div>
        </div>
      </div>
    </div>

    <div v-if="samplerStats.length" class="section-card">
      <div class="section-header">
        <div class="section-label">SAMPLERS</div>
        <div class="section-title">接口维度分析</div>
      </div>
      <div class="sampler-overview-grid">
        <div v-for="card in samplerOverviewCards" :key="card.key" class="sampler-overview-card">
          <div class="sampler-overview-label">{{ card.label }}</div>
          <div class="sampler-overview-name">{{ card.name }}</div>
          <div class="sampler-overview-value">{{ card.value }}</div>
          <div class="sampler-overview-caption">{{ card.caption }}</div>
        </div>
      </div>
      <el-table :data="displaySamplerStats" style="width: 100%" class="sampler-table">
        <el-table-column type="index" label="#" width="60" />
        <el-table-column prop="label" label="请求名称" min-width="180" show-overflow-tooltip />
        <el-table-column prop="url" label="URL" min-width="220" show-overflow-tooltip />
        <el-table-column prop="count" label="样本数" width="100" sortable />
        <el-table-column prop="error" label="错误数" width="100" sortable />
        <el-table-column label="错误率" width="110" sortable>
          <template #default="{ row }">{{ formatNumber(row.error_rate) }}%</template>
        </el-table-column>
        <el-table-column label="平均RT" width="120" sortable>
          <template #default="{ row }">{{ formatNumber(row.avg_rt) }} ms</template>
        </el-table-column>
        <el-table-column label="P95" width="120" sortable>
          <template #default="{ row }">{{ formatNumber(row.p95) }} ms</template>
        </el-table-column>
        <el-table-column label="吞吐" width="110" sortable>
          <template #default="{ row }">{{ formatNumber(row.throughput) }}/s</template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 节点监控面板 - 仅运行中显示 -->
    <div v-if="execution?.status === 'running' && nodeMetrics.length > 0" class="section-card node-metrics-panel">
      <div class="section-header">
        <div class="section-label">NODE MONITORING</div>
        <div class="section-title">节点实时监控</div>
      </div>
      <div class="metrics-grid">
        <div v-for="node in nodeMetrics" :key="node.id" class="node-card">
          <div class="node-header">
            <span class="node-name">{{ node.name }}</span>
            <span class="node-role">{{ node.role === 'slave' ? 'Slave' : 'Master' }}</span>
            <span class="node-status" :class="node.online ? 'online' : 'offline'">
              {{ node.online ? '在线' : '离线' }}
            </span>
          </div>
          <div v-if="node.stats" class="node-stats">
            <div class="stat-item">
              <span class="stat-label">CPU</span>
              <el-progress :percentage="Math.round(node.stats.cpu?.percent || 0)" :stroke-width="6" 
                :color="getResourceColor(node.stats.cpu?.percent || 0)" />
            </div>
            <div class="stat-item">
              <span class="stat-label">内存</span>
              <el-progress :percentage="Math.round(node.stats.memory?.percent || 0)" :stroke-width="6"
                :color="getResourceColor(node.stats.memory?.percent || 0)" />
            </div>
            <div class="stat-item">
              <span class="stat-label">磁盘</span>
              <el-progress :percentage="Math.round(node.stats.disk?.percent || 0)" :stroke-width="6"
                :color="getResourceColor(node.stats.disk?.percent || 0)" />
            </div>
            <div class="stat-item">
              <span class="stat-label">连接数</span>
              <span class="stat-value">{{ node.stats.network?.connections || 0 }}</span>
            </div>
          </div>
          <div v-else class="node-offline">
            数据不可用
          </div>
        </div>
      </div>
    </div>

    <div class="section-card">
      <div class="section-header">
        <div class="section-label">LIVE METRICS</div>
        <div class="section-title">实时趋势</div>
      </div>
      <div class="live-metrics-summary">
        <div class="live-summary-card">
          <span class="live-summary-label">{{ livePrimaryThroughputTitle }}</span>
          <span class="live-summary-value text-blue">{{ formatNumber(primaryMetricValue('primary_throughput')) || '-' }}</span>
        </div>
        <div class="live-summary-card">
          <span class="live-summary-label">{{ isExecutionRunning ? '请求次数（次/秒）' : '峰值请求次数（次/秒）' }}</span>
          <span class="live-summary-value text-green">{{ formatNumber(primaryMetricValue('request_rate')) || '-' }}</span>
        </div>
        <div class="live-summary-card">
          <span class="live-summary-label">平均RT</span>
          <span class="live-summary-value text-purple">{{ liveMetrics.avg_rt ? `${formatNumber(liveMetrics.avg_rt)} ms` : '-' }}</span>
        </div>
        <div class="live-summary-card">
          <span class="live-summary-label">{{ isExecutionRunning ? '响应时间' : '结束时响应时间' }}</span>
          <span class="live-summary-value text-purple">{{ primaryMetricValue('response_time') ? `${formatNumber(primaryMetricValue('response_time'))} ms` : '-' }}</span>
        </div>
        <div class="live-summary-card">
          <span class="live-summary-label">{{ isExecutionRunning ? '并发数' : '峰值并发数' }}</span>
          <span class="live-summary-value">{{ formatNumber(primaryMetricValue('concurrency')) || '-' }}</span>
        </div>
        <div class="live-summary-card">
          <span class="live-summary-label">成功率</span>
          <span class="live-summary-value text-green">{{ liveMetrics.success_rate !== undefined ? `${formatNumber(liveMetrics.success_rate)}%` : '-' }}</span>
        </div>
      </div>
      <div class="live-charts-grid">
        <MetricTrendChart
          :title="livePrimaryThroughputChartTitle"
          :value="formatNumber(primaryMetricValue('primary_throughput'))"
          :unit="livePrimaryThroughputUnit"
          :subline="chartSubline('primary_throughput')"
          :points="liveMetrics.points || []"
          :field="livePrimaryThroughputField"
          color="#38bdf8"
          :show-expand="true"
          :max-x-ticks="4"
          @expand="openExpandedChart('primary_throughput')"
        />
        <MetricTrendChart
          title="请求次数（次/秒）"
          :value="formatNumber(primaryMetricValue('request_rate'))"
          unit="req/s"
          :subline="chartSubline('request_rate')"
          :points="liveMetrics.points || []"
          field="request_rate"
          color="#22c55e"
          :show-expand="true"
          :max-x-ticks="4"
          @expand="openExpandedChart('request_rate')"
        />
        <MetricTrendChart
          title="平均RT"
          :value="liveMetrics.avg_rt ? formatNumber(liveMetrics.avg_rt) : '-'"
          unit="ms"
          :subline="chartSubline('avg_rt')"
          :points="liveMetrics.points || []"
          field="avg_rt"
          color="#a855f7"
          :show-expand="true"
          :max-x-ticks="4"
          @expand="openExpandedChart('avg_rt')"
        />
        <MetricTrendChart
          title="响应时间"
          :value="primaryMetricValue('response_time') ? formatNumber(primaryMetricValue('response_time')) : '-'"
          unit="ms"
          :subline="chartSubline('response_time')"
          :points="liveMetrics.points || []"
          field="avg_rt"
          color="#ec4899"
          :show-expand="true"
          :max-x-ticks="4"
          @expand="openExpandedChart('response_time')"
        />
        <MetricTrendChart
          title="并发数"
          :value="formatNumber(primaryMetricValue('concurrency'))"
          :subline="chartSubline('concurrency')"
          :points="liveMetrics.points || []"
          field="concurrency"
          color="#f59e0b"
          :show-expand="true"
          :max-x-ticks="4"
          @expand="openExpandedChart('concurrency')"
        />
        <MetricTrendChart
          title="成功率"
          :value="liveMetrics.success_rate !== undefined ? formatNumber(liveMetrics.success_rate) : '-'"
          unit="%"
          :subline="`错误率 ${liveMetrics.error_rate !== undefined ? formatNumber(liveMetrics.error_rate) : '-'}%`"
          :points="liveMetrics.points || []"
          field="success_rate"
          color="#84cc16"
          :show-expand="true"
          :max-x-ticks="4"
          @expand="openExpandedChart('success_rate')"
        />
        <MetricTrendChart
          title="P95/P99 响应时间"
          :value="formatNumber(primaryMetricValue('p95_rt'))"
          unit="ms"
          :subline="'P99: ' + formatNumber(primaryMetricValue('p99_rt')) + ' ms'"
          :points="liveMetrics.points || []"
          field="p95_rt"
          color="#f59e0b"
          second-field="p99_rt"
          second-color="#ef4444"
          second-label="P99"
          :show-expand="true"
          :max-x-ticks="4"
          @expand="openExpandedChart('p95_p99')"
        />
        <MetricTrendChart
          title="错误数趋势"
          :value="String(primaryMetricValue('error_count') || 0)"
          unit="errors"
          :points="liveMetrics.points || []"
          field="error_count"
          color="#ef4444"
          :show-expand="true"
          :max-x-ticks="4"
          @expand="openExpandedChart('error_count')"
        />
        <MetricTrendChart
          title="网络吞吐量"
          :value="formatBytesRateValue(primaryMetricValue('bytes_per_sec'))"
          unit="KB/s"
          :points="bytesKBPoints"
          field="bytes_per_sec"
          color="#22c55e"
          :show-expand="true"
          :max-x-ticks="4"
          @expand="openExpandedChart('bytes_per_sec')"
        />
      </div>
    </div>

    <!-- 详细统计 -->
    <div class="section-card">
      <div class="section-header">
        <div class="section-label">STATISTICS</div>
        <div class="section-title">详细统计</div>
      </div>
      <div class="stats-meta-row">
        <div class="stats-meta-card" v-for="meta in summaryMeta" :key="meta.label">
          <span class="stats-meta-label">{{ meta.label }}</span>
          <span class="stats-meta-value">{{ meta.value }}</span>
        </div>
      </div>
      <div class="detail-groups">
        <div class="detail-group-card" v-for="group in detailStatGroups" :key="group.title">
          <div class="detail-group-title">{{ group.title }}</div>
          <div class="detail-group-list">
            <div class="detail-group-item" v-for="item in group.items" :key="item.name">
              <span class="metric-name">{{ item.name }}</span>
              <span class="metric-value" :class="getMetricValueClass(item)">{{ item.value }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 错误分析 -->
    <div class="section-card error-analysis-section" v-loading="errorLoading">
      <div class="section-header">
        <div class="section-label">ERRORS</div>
        <div class="section-title">
          <el-icon><WarningFilled /></el-icon>
          错误分析
        </div>
        <div class="error-actions">
          <el-button size="small" @click="fetchErrors" :loading="errorLoading">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
          <el-button size="small" @click="downloadErrors" :disabled="!hasErrors">
            <el-icon><Download /></el-icon> 导出错误记录
          </el-button>
        </div>
      </div>

      <!-- 无错误时 -->
      <div v-if="errorAnalysis && errorAnalysis.total_errors === 0" class="no-errors">
        <el-icon class="success-icon"><CircleCheckFilled /></el-icon>
        <span>本次执行无错误记录</span>
      </div>

      <!-- 有错误时 -->
      <template v-else-if="errorAnalysis && errorAnalysis.total_errors > 0">
        <!-- 错误概览卡片 -->
        <div class="error-stats-row">
          <div class="error-stat-card">
            <div class="error-stat-value">{{ errorAnalysis.total_errors.toLocaleString() }}</div>
            <div class="error-stat-label">错误总数</div>
          </div>
          <div class="error-stat-card">
            <div class="error-stat-value">{{ errorAnalysis.error_types?.length || 0 }}</div>
            <div class="error-stat-label">错误类型数</div>
          </div>
          <div class="error-stat-card">
            <div class="error-stat-value" :style="{ color: '#ff4d4f' }">
              {{ (errorAnalysis.error_rate || 0).toFixed(2) }}%
            </div>
            <div class="error-stat-label">错误率</div>
          </div>
          <div class="error-stat-card" v-if="errorAnalysis.truncated">
            <div class="error-stat-value" style="color: #fa8c16; font-size: 14px;">
              仅展示前10000条
            </div>
            <div class="error-stat-label">记录截断</div>
          </div>
        </div>

        <!-- 错误分布图表行 -->
        <div class="error-charts-row" v-if="errorAnalysis.response_code_distribution?.length || errorAnalysis.error_timeline?.length">
          <!-- 响应码分布饼图 -->
          <div class="error-pie-section" v-if="errorAnalysis.response_code_distribution?.length">
            <div class="error-chart-card">
              <div class="error-chart-title">响应码分布</div>
              <div class="error-pie-content">
                <svg viewBox="0 0 100 100" class="error-pie-chart">
                  <!-- 背景圆环 -->
                  <circle cx="50" cy="50" r="40" fill="none" stroke="rgba(255,255,255,0.08)" stroke-width="18"/>
                  <!-- 各段弧 -->
                  <circle v-for="(segment, idx) in pieSegments" :key="idx"
                    cx="50" cy="50" r="40" fill="none"
                    :stroke="segment.color"
                    stroke-width="18"
                    :stroke-dasharray="`${segment.length} ${251.2 - segment.length}`"
                    :stroke-dashoffset="segment.offset"
                    transform="rotate(-90 50 50)"/>
                  <!-- 中心文字 -->
                  <text x="50" y="48" text-anchor="middle" class="pie-center-label">错误数</text>
                  <text x="50" y="62" text-anchor="middle" class="pie-center-value">{{ errorAnalysis.total_errors.toLocaleString() }}</text>
                </svg>
                <!-- 图例 -->
                <div class="error-pie-legend">
                  <div class="legend-item" v-for="(segment, idx) in pieSegments" :key="idx">
                    <span class="legend-color" :style="{ backgroundColor: segment.color }"></span>
                    <span class="legend-code" :title="segment.code">{{ segment.code }}</span>
                    <span class="legend-count">{{ segment.count.toLocaleString() }}</span>
                    <span class="legend-percent">{{ segment.percentage.toFixed(1) }}%</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 错误趋势时间线 -->
          <div class="error-timeline-section" v-if="errorAnalysis.error_timeline?.length">
            <MetricTrendChart
              title="错误趋势"
              :value="latestErrorCount"
              unit="errors"
              :points="errorTimelinePoints"
              field="error_count"
              color="#ef4444"
              :show-expand="true"
              @expand="openExpandedChart('error_timeline')"
            />
          </div>
        </div>

        <!-- Top 错误信息 -->
        <div class="top-errors-section" v-if="topErrorMessages.length">
          <div class="error-chart-card">
            <div class="error-chart-title">Top 错误信息</div>
            <div class="top-errors-list">
              <div class="top-error-item" v-for="(msg, idx) in topErrorMessages.slice(0, 10)" :key="idx">
                <span class="rank">{{ idx + 1 }}</span>
                <el-tooltip :content="msg.message" placement="top" :show-after="300">
                  <span class="message">{{ msg.message }}</span>
                </el-tooltip>
                <span class="count">{{ msg.count.toLocaleString() }} 次</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Tab 切换 -->
        <el-tabs v-model="errorActiveTab" class="error-tabs">
          <el-tab-pane label="错误类型分布" name="types">
            <el-table
              :data="errorAnalysis.error_types"
              style="width: 100%"
              row-key="label"
              @expand-change="(row, expanded) => { selectedErrorType = expanded ? row : null }"
            >
              <el-table-column type="expand">
                <template #default="{ row }">
                  <div class="sample-errors-wrapper">
                    <div class="sample-errors-title">样例错误 (最多5条)</div>
                    <el-table :data="formatSampleErrors(row.sample_errors)" size="small" class="sample-errors-table">
                      <el-table-column prop="timestamp" label="时间" width="160" />
                      <el-table-column prop="elapsed" label="响应时间(ms)" width="110" />
                      <el-table-column prop="thread_name" label="线程名" min-width="160" show-overflow-tooltip />
                      <el-table-column prop="url" label="URL" min-width="200" show-overflow-tooltip />
                      <el-table-column label="响应预览" min-width="260">
                        <template #default="{ row }">
                          <span
                            class="response-preview-cell"
                            :class="{ 'is-empty': !hasResponseDetail(row) }"
                            @click="hasResponseDetail(row) && showResponseDetail(row)"
                          >
                            {{ formatResponsePreview(row) }}
                          </span>
                        </template>
                      </el-table-column>
                      <el-table-column label="失败原因" min-width="300">
                        <template #default="{ row }">
                          <span 
                            class="failure-message-cell" 
                            :class="{ 'is-empty': !row.failure_message || row.failure_message === '-' }"
                            @click="row.failure_message && row.failure_message !== '-' && showFailureMessage(row.failure_message)"
                          >
                            {{ row.failure_message }}
                          </span>
                        </template>
                      </el-table-column>
                    </el-table>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="label" label="请求名称" min-width="160" show-overflow-tooltip>
                <template #default="{ row }">
                  <span>{{ row.label }}</span>
                  <el-tag 
                    v-if="errorAnalysis?.type_truncated && errorAnalysis.type_truncated[row.label + '|' + row.response_code]"
                    type="warning" 
                    size="small" 
                    style="margin-left: 8px;"
                  >
                    已达上限(10000条)
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="响应码" width="120">
                <template #default="{ row }">
                  <el-tag 
                    size="small" 
                    :style="{ backgroundColor: getErrorCodeColor(row.response_code) + '20', color: getErrorCodeColor(row.response_code), borderColor: getErrorCodeColor(row.response_code) + '40' }"
                  >
                    {{ row.response_code }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="response_message" label="响应信息" min-width="180" show-overflow-tooltip />
              <el-table-column label="响应预览" min-width="260">
                <template #default="{ row }">
                  <span
                    class="response-preview-cell"
                    :class="{ 'is-empty': !row.response_preview }"
                  >
                    {{ row.response_preview || '-' }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column prop="count" label="数量" width="100" sortable>
                <template #default="{ row }">
                  <span class="error-count">{{ row.count.toLocaleString() }}</span>
                </template>
              </el-table-column>
              <el-table-column label="占比" width="150">
                <template #default="{ row }">
                  <el-progress 
                    :percentage="parseFloat(row.percentage.toFixed(1))" 
                    :color="'#ff4d4f'"
                    :stroke-width="8"
                    :show-text="true"
                  />
                </template>
              </el-table-column>
              <el-table-column prop="first_time" label="首次出现" width="160" />
              <el-table-column prop="last_time" label="末次出现" width="160" />
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="错误记录明细" name="records">
            <el-alert
              v-if="errorAnalysis.truncated"
              title="错误记录超过10000条，仅展示前10000条"
              type="warning"
              :closable="false"
              show-icon
              style="margin-bottom: 12px;"
            />
            <el-alert
              v-if="errorAnalysis.detail_upload_warning"
              :title="errorAnalysis.detail_upload_warning"
              type="warning"
              :closable="false"
              show-icon
              style="margin-bottom: 12px;"
            >
              <template #default>
                <div v-if="errorAnalysis.missing_detail_sources?.length">
                  未回传节点：{{ errorAnalysis.missing_detail_sources.join('、') }}
                </div>
              </template>
            </el-alert>
            <!-- 筛选栏 -->
            <div class="error-filters-row">
              <el-select
                v-model="errorFilterCode"
                placeholder="按响应码筛选"
                clearable
                size="small"
                class="error-filter-select"
                @change="errorPage = 1"
              >
                <el-option
                  v-for="opt in errorCodeOptions"
                  :key="opt.value"
                  :label="opt.label"
                  :value="opt.value"
                />
              </el-select>
              <el-select
                v-model="errorFilterLabel"
                placeholder="按请求名称筛选"
                clearable
                size="small"
                class="error-filter-select"
                @change="errorPage = 1"
              >
                <el-option
                  v-for="opt in errorLabelOptions"
                  :key="opt.value"
                  :label="opt.label"
                  :value="opt.value"
                />
              </el-select>
              <span class="error-filter-result" v-if="errorFilterCode || errorFilterLabel">
                共 {{ filteredErrorRecordsRaw.length }} 条匹配记录
              </span>
            </div>
            <el-table :data="formatErrorRecords" style="width: 100%" v-loading="errorLoading">
              <el-table-column prop="timestamp" label="时间" width="170" />
              <el-table-column prop="source" label="来源节点" width="190" show-overflow-tooltip />
              <el-table-column prop="label" label="请求名称" min-width="150" show-overflow-tooltip />
              <el-table-column label="响应码" width="120">
                <template #default="{ row }">
                  <el-tag 
                    size="small" 
                    :style="{ backgroundColor: getErrorCodeColor(row.response_code) + '20', color: getErrorCodeColor(row.response_code), borderColor: getErrorCodeColor(row.response_code) + '40' }"
                  >
                    {{ row.response_code }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="response_message" label="响应信息" min-width="170" show-overflow-tooltip />
              <el-table-column label="响应预览" min-width="260">
                <template #default="{ row }">
                  <span
                    class="response-preview-cell"
                    :class="{ 'is-empty': !hasResponseDetail(row) }"
                    @click="hasResponseDetail(row) && showResponseDetail(row)"
                  >
                    {{ formatResponsePreview(row) }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column label="失败原因" min-width="300">
                <template #default="{ row }">
                  <span 
                    class="failure-message-cell" 
                    :class="{ 'is-empty': !row.failure_message || row.failure_message === '-' }"
                    @click="row.failure_message && row.failure_message !== '-' && showFailureMessage(row.failure_message)"
                  >
                    {{ row.failure_message }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column prop="elapsed" label="响应时间(ms)" width="120" />
              <el-table-column prop="url" label="URL" min-width="250" show-overflow-tooltip />
              <el-table-column label="详情" width="80" fixed="right">
                <template #default="{ row }">
                  <el-button type="primary" link size="small" @click="showErrorDetail(row)">
                    查看
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <el-pagination
              v-model:current-page="errorPage"
              v-model:page-size="errorPageSize"
              :page-sizes="[50, 100, 200]"
              :total="filteredErrorRecordsRaw.length"
              layout="total, sizes, prev, pager, next"
              class="error-pagination"
            />
          </el-tab-pane>
        </el-tabs>
      </template>
    </div>

    <!-- 测试报告 -->
    <div class="section-card">
      <div class="section-header">
        <div class="section-label">REPORT</div>
        <div class="section-title">测试报告</div>
        <el-button 
          v-if="execution.status === 'success'"
          type="primary" 
          link 
          size="small"
          @click="reportFullscreen = true"
        >
          <el-icon><FullScreen /></el-icon> 全屏查看
        </el-button>
      </div>
      <div class="report-wrapper">
        <div v-if="execution.status === 'success'" class="report-container">
          <iframe 
            :src="reportUrl" 
            class="report-iframe"
            frameborder="0"
          ></iframe>
        </div>
        <div v-else class="report-placeholder">
          <div class="placeholder-icon">
            <el-icon :size="48"><Document /></el-icon>
          </div>
          <p class="placeholder-text">{{ getReportPlaceholderText() }}</p>
        </div>
      </div>
    </div>

    <!-- 执行日志 -->
    <div class="section-card terminal-section">
      <div class="section-header">
        <div class="section-label">LOGS</div>
        <div class="section-title">执行日志</div>
      </div>
      <div class="terminal-window">
        <!-- 终端标题栏 -->
        <div class="terminal-header">
          <div class="terminal-controls">
            <span class="control-dot red"></span>
            <span class="control-dot yellow"></span>
            <span class="control-dot green"></span>
          </div>
          <div class="terminal-title">
            <el-icon><Monitor /></el-icon>
            <span>Terminal</span>
          </div>
          <div class="terminal-toolbar">
            <el-input 
              v-model="logSearch" 
              placeholder="搜索日志..." 
              size="small" 
              clearable 
              class="log-search-input"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
            <span v-if="logSearch" class="search-count">{{ matchCount }} 条匹配</span>
          </div>
          <div class="terminal-actions">
            <button
              v-if="execution.status === 'running'"
              class="terminal-btn"
              @click="toggleLogStream"
              :title="isStreaming ? '停止实时日志' : '开启实时日志'"
            >
              <el-icon v-if="!isStreaming"><VideoPlay /></el-icon>
              <el-icon v-else><VideoPause /></el-icon>
            </button>
            <button class="terminal-btn" @click="copyLogs" title="复制日志">
              <el-icon><CopyDocument /></el-icon>
            </button>
            <button class="terminal-btn" @click="exportLogs" title="导出日志">
              <el-icon><Download /></el-icon>
            </button>
            <button class="terminal-btn" @click="refreshLog" title="刷新日志">
              <el-icon><Refresh /></el-icon>
            </button>
          </div>
        </div>
        
        <!-- 终端内容区 -->
        <div ref="logContainer" class="terminal-body">
          <div v-if="logLines.length === 0" class="terminal-empty">
            暂无日志
          </div>
          <div 
            v-for="(line, index) in logLines" 
            :key="index"
            class="log-line"
            :class="{ 'cursor-line': index === logLines.length - 1 && execution.status === 'running' }"
          >
            <span class="log-line-number">{{ String(index + 1).padStart(4, '0') }}</span>
            <span class="log-line-content" v-html="formatLogLine(line)"></span>
          </div>
        </div>
        
        <!-- 终端状态栏 -->
        <div class="terminal-footer">
          <div class="connection-status">
            <span 
              class="status-indicator"
              :class="sseConnected ? 'connected' : 'disconnected'"
            ></span>
            <span class="status-text">
              {{ isStreaming ? (sseConnected ? 'Streaming' : 'Connecting') : 'Idle' }}
            </span>
          </div>
          <div class="log-stats">
            <span>{{ logLines.length }} lines</span>
            <span v-if="logTrimmed">仅保留最近 {{ MAX_LOG_LINES }} 行</span>
          </div>
        </div>
      </div>
    </div>

    <el-dialog
      v-model="expandedChartVisible"
      :title="expandedChartConfig?.title || '趋势详情'"
      width="960px"
      class="chart-detail-dialog"
    >
      <MetricTrendChart
        v-if="expandedChartConfig"
        :title="expandedChartConfig.title"
        :value="expandedChartConfig.value"
        :unit="expandedChartConfig.unit"
        :subline="expandedChartConfig.subline"
        :points="liveMetrics.points || []"
        :field="expandedChartConfig.field"
        :color="expandedChartConfig.color"
        :height="380"
        :max-x-ticks="4"
      />
    </el-dialog>

    <el-dialog
      v-model="responseDialogVisible"
      title="错误响应详情"
      width="880px"
      class="response-detail-dialog"
    >
      <div class="response-detail-body">
        <div class="response-detail-section" v-if="responseDialogRecord.response_message">
          <div class="response-detail-label">响应信息</div>
          <pre>{{ responseDialogRecord.response_message }}</pre>
        </div>
        <div class="response-detail-section" v-if="responseDialogRecord.response_data">
          <div class="response-detail-label">响应内容</div>
          <pre>{{ responseDialogRecord.response_data }}</pre>
        </div>
        <div class="response-detail-section" v-if="responseDialogRecord.response_headers">
          <div class="response-detail-label">响应头</div>
          <pre>{{ responseDialogRecord.response_headers }}</pre>
        </div>
        <div class="response-detail-section" v-if="responseDialogRecord.failure_message">
          <div class="response-detail-label">失败原因</div>
          <pre>{{ responseDialogRecord.failure_message }}</pre>
        </div>
      </div>
    </el-dialog>

    <!-- 报告全屏查看 Dialog -->
    <el-dialog 
      v-model="reportFullscreen" 
      fullscreen 
      :show-close="true"
      class="report-fullscreen-dialog"
    >
      <template #header>
        <span>测试报告 - 全屏查看</span>
      </template>
      <iframe 
        :src="reportUrl" 
        class="report-fullscreen-iframe"
        frameborder="0"
      ></iframe>
    </el-dialog>

    <!-- 错误详情 Dialog -->
    <el-dialog
      v-model="errorDetailVisible"
      title="错误详情"
      width="800px"
      class="error-detail-dialog"
    >
      <div
        v-if="errorDetailNotice"
        class="error-detail-notice"
        :class="{ 'is-warning': !errorAnalysis?.detail_fields_available }"
      >
        <div class="error-detail-notice-title">
          {{ errorAnalysis?.detail_fields_available ? '详情字段状态' : '当前结果文件未保存完整 HTTP 明细' }}
        </div>
        <div class="error-detail-notice-text">{{ errorDetailNotice }}</div>
        <div v-if="errorAnalysis?.available_detail_fields?.length" class="error-detail-notice-meta">
          已检测到字段：{{ errorAnalysis.available_detail_fields.join(', ') }}
        </div>
      </div>
      <el-tabs v-model="errorDetailTab" class="error-detail-tabs">
        <!-- 请求信息 Tab -->
        <el-tab-pane label="请求信息" name="request">
          <div class="error-detail-content">
            <div class="error-detail-section">
              <div class="error-detail-label">来源节点</div>
              <pre class="error-detail-code">{{ currentErrorRecord?.source || '-' }}</pre>
            </div>
            <div class="error-detail-section">
              <div class="error-detail-label">URL</div>
              <pre class="error-detail-code">{{ currentErrorRecord?.url || '-' }}</pre>
            </div>
            <div class="error-detail-section">
              <div class="error-detail-label">请求头</div>
              <pre class="error-detail-code" v-if="currentErrorRecord?.request_headers">{{ currentErrorRecord.request_headers }}</pre>
              <div class="error-detail-empty" v-else>{{ getEmptyDetailText('requestHeaders') }}</div>
            </div>
            <div class="error-detail-section">
              <div class="error-detail-label">请求体/参数</div>
              <pre class="error-detail-code" v-if="currentErrorRecord?.request_body">{{ currentErrorRecord.request_body }}</pre>
              <div class="error-detail-empty" v-else>{{ getEmptyDetailText('samplerData') }}</div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 响应信息 Tab -->
        <el-tab-pane label="响应信息" name="response">
          <div class="error-detail-content">
            <div class="error-detail-section">
              <div class="error-detail-label">响应码</div>
              <div class="error-detail-value">
                <el-tag 
                  size="small" 
                  :style="{ backgroundColor: getErrorCodeColor(currentErrorRecord?.response_code) + '20', color: getErrorCodeColor(currentErrorRecord?.response_code), borderColor: getErrorCodeColor(currentErrorRecord?.response_code) + '40' }"
                >
                  {{ currentErrorRecord?.response_code || '-' }}
                </el-tag>
                <span class="error-detail-message">{{ currentErrorRecord?.response_message || '' }}</span>
              </div>
            </div>
            <div class="error-detail-section">
              <div class="error-detail-label">响应头</div>
              <pre class="error-detail-code" v-if="currentErrorRecord?.response_headers">{{ currentErrorRecord.response_headers }}</pre>
              <div class="error-detail-empty" v-else>{{ getEmptyDetailText('responseHeaders') }}</div>
            </div>
            <div class="error-detail-section">
              <div class="error-detail-label">响应内容</div>
              <pre class="error-detail-code scrollable" v-if="currentErrorRecord?.response_data">{{ currentErrorRecord.response_data }}</pre>
              <div class="error-detail-empty" v-else>{{ getEmptyDetailText('responseData') }}</div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 时序信息 Tab -->
        <el-tab-pane label="时序信息" name="timing">
          <div class="error-detail-content">
            <div class="error-detail-timing-grid">
              <div class="error-detail-timing-card">
                <div class="error-detail-timing-label">连接时间</div>
                <div class="error-detail-timing-value">{{ currentErrorRecord?.connect_time !== undefined ? currentErrorRecord.connect_time + ' ms' : '-' }}</div>
              </div>
              <div class="error-detail-timing-card">
                <div class="error-detail-timing-label">延迟</div>
                <div class="error-detail-timing-value">{{ currentErrorRecord?.latency !== undefined ? currentErrorRecord.latency + ' ms' : '-' }}</div>
              </div>
              <div class="error-detail-timing-card">
                <div class="error-detail-timing-label">响应时间</div>
                <div class="error-detail-timing-value">{{ currentErrorRecord?.elapsed !== undefined ? currentErrorRecord.elapsed + ' ms' : '-' }}</div>
              </div>
            </div>
            <div class="error-detail-section">
              <div class="error-detail-label">传输数据</div>
              <div class="error-detail-data-row">
                <div class="error-detail-data-item">
                  <span class="error-detail-data-label">发送字节数:</span>
                  <span class="error-detail-data-value">{{ currentErrorRecord?.sent_bytes !== undefined ? formatBytes(currentErrorRecord.sent_bytes) : '-' }}</span>
                </div>
                <div class="error-detail-data-item">
                  <span class="error-detail-data-label">接收字节数:</span>
                  <span class="error-detail-data-value">{{ currentErrorRecord?.received_bytes !== undefined ? formatBytes(currentErrorRecord.received_bytes) : '-' }}</span>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  ArrowLeft, 
  ArrowDown,
  CircleClose, 
  Document, 
  Refresh,
  Clock,
  Monitor,
  CopyDocument,
  Search,
  Download,
  WarningFilled,
  CircleCheckFilled,
  FolderOpened,
  DocumentCopy,
  Files,
  FullScreen,
  VideoPlay,
  VideoPause,
  TrendCharts,
  StarFilled
} from '@element-plus/icons-vue'
import { executionApi } from '@/api/execution'
import { formatDateTimeInShanghai, parseServerDateTime } from '@/utils/datetime'
import MetricTrendChart from '@/components/MetricTrendChart.vue'

const route = useRoute()
const router = useRouter()
const executionId = computed(() => route.params.id)
const executionNumericId = computed(() => {
  const value = Number.parseInt(route.params.id, 10)
  return Number.isFinite(value) ? value : null
})
const hasValidExecutionId = computed(() => executionNumericId.value !== null)

const loading = ref(false)
const stopping = ref(false)
const execution = ref({})
const logLinesData = ref([])
const logContainer = ref(null)
const sseConnected = ref(false)
const eventSource = ref(null)
const refreshTimer = ref(null)
const durationTimer = ref(null)
const isStreaming = ref(false)
const logSnapshotLoading = ref(false)
const logTrimmed = ref(false)
const pendingLogLines = ref([])
let logFlushTimer = null
let logSnapshotController = null
const liveMetrics = ref({ points: [] })
const detailRefreshing = ref(false)
const liveRefreshing = ref(false)
const errorAnalysisLoaded = ref(false)
const nowTick = ref(Date.now())
const expandedChartKey = ref('')
const expandedChartVisible = ref(false)
const pageVisible = ref(typeof document === 'undefined' ? true : document.visibilityState === 'visible')
let leavingPage = false

// 错误分析相关
const errorAnalysis = ref(null)
const errorLoading = ref(false)
const errorActiveTab = ref('types') // types | records
const errorPage = ref(1)
const errorPageSize = ref(50)
const selectedErrorType = ref(null)
const responseDialogVisible = ref(false)
const responseDialogRecord = ref({})

// 错误筛选相关
const errorFilterCode = ref('')
const errorFilterLabel = ref('')

// 报告全屏查看
const reportFullscreen = ref(false)

// 错误详情弹窗
const errorDetailVisible = ref(false)
const currentErrorRecord = ref(null)
const errorDetailTab = ref('request')

// 基准线对比相关
const baselineComparison = ref(null)
const baselineLoading = ref(false)

// 节点监控相关
const nodeMetrics = ref([])
let metricsTimer = null

const showErrorDetail = (record) => {
  currentErrorRecord.value = record
  errorDetailTab.value = 'request'
  errorDetailVisible.value = true
}

// 日志搜索和高亮
const logSearch = ref('')
const reconnectAttempts = ref(0)
const maxReconnectAttempts = 5
const MAX_LOG_LINES = 600
const LOG_FLUSH_INTERVAL = 300

// 日志行数组
const logLines = computed(() => {
  if (!logLinesData.value.length) return []
  return logLinesData.value.filter(line => line.trim() !== '')
})

// 报告 URL
const reportUrl = computed(() => {
  return `/reports/${executionId.value}/report/index.html`
})

// 解析 summary_data
const summary = computed(() => {
  if (!execution.value.summary_data) return {}
  try {
    return typeof execution.value.summary_data === 'string'
      ? JSON.parse(execution.value.summary_data)
      : execution.value.summary_data
  } catch {
    return {}
  }
})

// 是否有结果文件
const hasResultFile = computed(() => {
  return execution.value.result_path && execution.value.result_path !== ''
})

// 是否有报告目录
const hasReportDir = computed(() => {
  return execution.value.report_path && execution.value.report_path !== ''
})

// 是否有错误记录
const hasErrors = computed(() => {
  return errorAnalysis.value && errorAnalysis.value.total_errors > 0
})

const hasLiveMetrics = computed(() => Array.isArray(liveMetrics.value?.points) && liveMetrics.value.points.length > 0)
const isExecutionRunning = computed(() => execution.value.status === 'running')
const diagnostics = computed(() => execution.value?.diagnostics || {})

const summaryPrimaryThroughputLabel = computed(() => {
  return summary.value.primary_throughput_label || (Number(summary.value.transaction_samples || 0) > 0 ? 'TPS（事务/s）' : '请求次数（次/秒）')
})

const summaryPrimaryThroughputUnit = computed(() => {
  return summary.value.primary_throughput_unit || (Number(summary.value.transaction_samples || 0) > 0 ? 'tps' : 'req/s')
})

const summaryPrimaryThroughputField = computed(() => {
  return summary.value.primary_throughput_field || (Number(summary.value.transaction_samples || 0) > 0 ? 'transaction_tps' : 'request_rate')
})

const summaryPrimaryThroughputValue = computed(() => {
  const field = summaryPrimaryThroughputField.value
  if (field && summary.value[field] !== undefined) {
    return summary.value[field]
  }
  return summary.value.primary_throughput ?? summary.value.request_rate ?? summary.value.transaction_tps ?? summary.value.throughput ?? null
})

const livePrimaryThroughputTitle = computed(() => {
  if (isExecutionRunning.value) {
    return `当前 ${liveMetrics.value.primary_throughput_label || summaryPrimaryThroughputLabel.value}`
  }
  return `峰值 ${liveMetrics.value.primary_throughput_label || summaryPrimaryThroughputLabel.value}`
})

const livePrimaryThroughputChartTitle = computed(() => {
  return `${liveMetrics.value.primary_throughput_label || summaryPrimaryThroughputLabel.value}趋势`
})

const livePrimaryThroughputField = computed(() => {
  return liveMetrics.value.primary_throughput_field || (liveMetrics.value.has_transaction_samples ? 'tps' : 'request_rate')
})

const livePrimaryThroughputUnit = computed(() => {
  return liveMetrics.value.primary_throughput_unit || (liveMetrics.value.has_transaction_samples ? 'tps' : 'req/s')
})

const summaryCounts = computed(() => {
  const total = isExecutionRunning.value
    ? Number(liveMetrics.value.total_requests || 0)
    : Number(summary.value.total_samples || 0)
  const errors = isExecutionRunning.value
    ? Number(liveMetrics.value.error_requests ?? errorAnalysis.value?.total_errors ?? 0)
    : Number(summary.value.error_samples ?? errorAnalysis.value?.total_errors ?? 0)
  const success = isExecutionRunning.value
    ? Number(liveMetrics.value.success_requests ?? Math.max(total - errors, 0))
    : Number(summary.value.request_success_samples ?? summary.value.success_samples ?? Math.max(total - errors, 0))
  return { total, success, errors }
})

const overviewStatusTone = computed(() => {
  return execution.value.status_tone || (execution.value.status === 'failed' ? 'danger' : execution.value.status === 'running' ? 'info' : getErrorRateValue(summary.value.error_rate) > 5 ? 'warning' : 'success')
})

const overviewStatusNote = computed(() => {
  return execution.value.status_reason || (execution.value.status === 'running' ? '实时指标持续刷新中' : execution.value.status === 'failed' ? '本次执行存在明显失败样本' : getErrorRateValue(summary.value.error_rate) > 0 ? '执行完成，但存在错误请求' : '执行完成，整体表现稳定')
})

const overviewPrimaryMetrics = computed(() => {
  const s = summary.value
  const live = liveMetrics.value || {}
  const throughputValue = isExecutionRunning.value
    ? live.current_primary_throughput
    : summaryPrimaryThroughputValue.value
  const avgResponseTime = isExecutionRunning.value
    ? live.avg_rt
    : s.avg_response_time
  const throughputLabel = live.primary_throughput_label || summaryPrimaryThroughputLabel.value
  const throughputUnit = livePrimaryThroughputUnit.value
  const throughputCaption = isExecutionRunning.value
    ? `当前 ${formatNumber(live.current_primary_throughput)} ${throughputUnit} / 平均 ${formatNumber(live.avg_primary_throughput)} ${throughputUnit} / 峰值 ${formatNumber(live.peak_primary_throughput)} ${throughputUnit}`
    : s.sample_span_ms ? `基于 ${formatDurationFromMs(s.sample_span_ms)} 真实采样跨度` : '等待采样结果'
  const latencyCaption = isExecutionRunning.value
    ? `当前 ${live.current_rt ? `${formatNumber(live.current_rt)} ms` : '-'} / 请求速率 ${formatNumber(live.current_request_rate)} req/s`
    : s.p95 ? `P95 ${formatNumber(s.p95)} ms / P99 ${formatNumber(s.p99)} ms` : '等待延迟分布统计'
  return [
    {
      key: 'throughput',
      label: throughputLabel,
      value: throughputValue ? `${formatNumber(throughputValue)} ${throughputUnit}` : '-',
      caption: throughputCaption,
      tone: 'throughput'
    },
    {
      key: 'avg_response_time',
      label: '平均响应时间',
      value: avgResponseTime ? `${formatNumber(avgResponseTime)} ms` : '-',
      caption: latencyCaption,
      tone: 'latency'
    }
  ]
})

const overviewMiniMetrics = computed(() => {
  const s = summary.value
  const live = liveMetrics.value || {}
  const counts = summaryCounts.value
  const liveErrorRate = live.error_rate
  const liveSuccessRate = live.success_rate
  return [
    {
      key: 'total_samples',
      label: '总样本数',
      value: formatNumber(counts.total),
      caption: `成功 ${formatNumber(counts.success)} / 错误 ${formatNumber(counts.errors)}`,
      color: 'blue'
    },
    {
      key: 'error_rate',
      label: '错误率',
      value: (isExecutionRunning.value ? liveErrorRate : s.error_rate) !== undefined
        ? `${formatNumber(isExecutionRunning.value ? liveErrorRate : s.error_rate)}%`
        : '-',
      caption: (isExecutionRunning.value ? liveSuccessRate : s.success_rate) !== undefined
        ? `成功率 ${formatNumber(isExecutionRunning.value ? liveSuccessRate : s.success_rate)}%`
        : '暂无成功率统计',
      color: getErrorRateValue(isExecutionRunning.value ? liveErrorRate : s.error_rate) > 5 ? 'red' : 'green'
    },
    {
      key: 'p95',
      label: 'P95 延迟',
      value: isExecutionRunning.value ? '-' : (s.p95 ? `${formatNumber(s.p95)} ms` : '-'),
      caption: isExecutionRunning.value
        ? '运行中暂无峰值统计'
        : s.max_response_time !== undefined ? `最大 ${formatNumber(s.max_response_time)} ms` : '暂无峰值数据',
      color: 'purple'
    },
    {
      key: 'duration',
      label: '执行时长',
      value: formatDuration(displayDurationSeconds.value),
      caption: formatDateTime(execution.value.start_time || execution.value.created_at),
      color: 'blue'
    },
    {
      key: 'received_bytes',
      label: '接收流量',
      value: isExecutionRunning.value ? '-' : formatBytes(s.received_bytes),
      caption: isExecutionRunning.value
        ? '运行中暂不显示累计流量'
        : s.received_bytes_per_sec ? `${formatBytesRate(s.received_bytes_per_sec)}` : '暂无流量速率',
      color: 'green'
    },
    {
      key: 'sent_bytes',
      label: '发送流量',
      value: isExecutionRunning.value ? '-' : formatBytes(s.sent_bytes),
      caption: isExecutionRunning.value
        ? '运行中暂不显示累计流量'
        : s.sent_bytes_per_sec ? `${formatBytesRate(s.sent_bytes_per_sec)}` : '暂无流量速率',
      color: 'purple'
    }
  ]
})

const diagnosticCards = computed(() => {
  const diag = diagnostics.value || {}
  const modeMap = {
    local: '本地执行',
    distributed: '分布式执行',
    distributed_with_master: 'Master + Slave'
  }
  const detailStateMap = {
    disabled: '未开启',
    pending: '等待写入',
    local_captured: '本地已采集',
    partial: '部分到位',
    complete: '采集完整',
    missing: '未生成'
  }
  const resultReady = diag.result_merge_ready ? '已就绪' : '等待中'
  const resultCaption = Array.isArray(diag.result_files)
    ? diag.result_files.filter(item => item.exists).map(item => item.label).join(' / ') || '尚未生成任何结果文件'
    : '暂无结果文件信息'
  const detailCaption = diag.save_http_details
    ? `已回传 ${diag.received_detail_sources?.length || 0} / 预期 ${diag.expected_detail_sources?.length || 0}`
    : '当前执行未开启失败请求明细'
  const dependencyCaption = diag.csv_dependencies?.length || diag.file_dependencies?.length || diag.plugin_dependencies?.length
    ? `CSV ${diag.csv_dependencies?.length || 0} / 文件 ${diag.file_dependencies?.length || 0} / 插件 ${diag.plugin_dependencies?.length || 0}`
    : '未检测到额外依赖'
  const cards = [
    {
      key: 'mode',
      label: '执行模式',
      value: modeMap[diag.mode] || '本地执行',
      caption: diag.slave_hosts?.length ? `节点 ${diag.slave_hosts.join('、')}` : '当前未选择 Slave 节点',
      color: 'blue'
    },
    {
      key: 'result',
      label: '结果链路',
      value: resultReady,
      caption: resultCaption,
      color: diag.result_merge_ready ? 'green' : 'warning'
    },
    {
      key: 'details',
      label: 'HTTP 明细',
      value: detailStateMap[diag.detail_state] || '未开启',
      caption: detailCaption,
      color: diag.detail_state === 'complete' || diag.detail_state === 'local_captured' ? 'green' : diag.detail_state === 'disabled' ? 'blue' : 'warning'
    },
    {
      key: 'dependencies',
      label: '依赖概况',
      value: diag.split_csv ? '已启用 CSV 分片' : '按原脚本执行',
      caption: dependencyCaption,
      color: diag.warnings?.length ? 'warning' : 'blue'
    }
  ]

  if (diag.mode && diag.mode !== 'local') {
    cards.push({
      key: 'topology',
      label: '拓扑说明',
      value: diag.include_master ? 'Master 参与施压' : 'Master 仅调度',
      caption: `Slave ${diag.slave_count || 0} 台 / CSV分片 ${diag.split_csv ? '开启' : '关闭'}`,
      color: diag.include_master ? 'green' : 'blue'
    })
  }

  if (diag.save_http_details) {
    cards.push({
      key: 'detail-sources',
      label: '明细回传',
      value: `${diag.received_detail_sources?.length || 0}/${diag.expected_detail_sources?.length || 0}`,
      caption: (diag.missing_detail_sources?.length
        ? `缺失节点：${diag.missing_detail_sources.join('、')}`
        : '所有预期节点均已回传 HTTP 明细'),
      color: diag.missing_detail_sources?.length ? 'warning' : 'green'
    })
  }

  return cards
})

const diagnosticWarnings = computed(() => {
  return diagnostics.value?.warnings || []
})

const executionConclusion = computed(() => {
  const conclusion = summary.value?.conclusion
  return conclusion && typeof conclusion === 'object' ? conclusion : null
})

const conclusionHighlights = computed(() => {
  const items = executionConclusion.value?.highlights
  return Array.isArray(items) ? items : []
})

const conclusionRecommendations = computed(() => {
  const items = executionConclusion.value?.recommendations
  return Array.isArray(items) ? items : []
})

const conclusionLevelText = computed(() => {
  const level = executionConclusion.value?.level
  if (level === 'danger') return '高风险'
  if (level === 'warning') return '需关注'
  return '稳定'
})

const samplerStats = computed(() => {
  const items = summary.value?.sampler_stats
  return Array.isArray(items) ? items : []
})

const displaySamplerStats = computed(() => samplerStats.value.slice(0, 10))

const samplerOverviewCards = computed(() => {
  if (!samplerStats.value.length) return []
  const hottest = samplerStats.value.reduce((best, item) => (item.count > (best?.count || 0) ? item : best), null)
  const slowest = samplerStats.value.reduce((best, item) => (item.avg_rt > (best?.avg_rt || 0) ? item : best), null)
  const riskiest = samplerStats.value.reduce((best, item) => (item.error > (best?.error || 0) ? item : best), null)
  return [
    {
      key: 'hot',
      label: '样本最多',
      name: hottest?.label || '-',
      value: hottest ? `${formatNumber(hottest.count)} 次` : '-',
      caption: hottest?.url || '暂无接口数据'
    },
    {
      key: 'slow',
      label: '平均最慢',
      name: slowest?.label || '-',
      value: slowest ? `${formatNumber(slowest.avg_rt)} ms` : '-',
      caption: slowest?.url || '暂无接口数据'
    },
    {
      key: 'risk',
      label: '错误最多',
      name: riskiest?.label || '-',
      value: riskiest ? `${formatNumber(riskiest.error)} 次 / ${formatNumber(riskiest.error_rate)}%` : '-',
      caption: riskiest?.url || '暂无接口数据'
    }
  ]
})

const timelineStages = computed(() => {
  if (!execution.value?.id) return []
  const diag = diagnostics.value || {}
  const stages = []
  const pushStage = (key, step, name, time, description, tone = 'info') => {
    if (!time && !description) return
    stages.push({
      key,
      step,
      name,
      time: time || '-',
      description: description || '-',
      tone
    })
  }

  pushStage(
    'created',
    '01',
    '创建任务',
    formatDateTime(execution.value.created_at),
    execution.value.remarks || '已创建执行记录，等待脚本启动。',
    'blue'
  )
  if (execution.value.start_time) {
    pushStage(
      'started',
      '02',
      '开始执行',
      formatDateTime(execution.value.start_time),
      diag.mode === 'local'
        ? '当前任务在 Master 本机执行。'
        : `执行模式：${diag.include_master ? 'Master + Slave' : '仅 Slave'}，节点数 ${diag.slave_count || 0}。`,
      'green'
    )
  }
  if (diag.runtime_scripts?.length) {
    pushStage(
      'runtime',
      '03',
      '生成运行时脚本',
      formatDateTime(execution.value.start_time),
      `已生成 ${diag.runtime_scripts.length} 份运行时脚本，CSV分片 ${diag.split_csv ? '开启' : '关闭'}。`,
      'blue'
    )
  }
  if (diag.save_http_details) {
    pushStage(
      'detail',
      '04',
      '错误明细回传',
      execution.value.status === 'running' ? '执行中' : formatDateTime(execution.value.end_time),
      `已回传 ${diag.received_detail_sources?.length || 0}/${diag.expected_detail_sources?.length || 0} 个来源。`,
      diag.missing_detail_sources?.length ? 'warning' : 'green'
    )
  }
  if (execution.value.end_time || execution.value.status !== 'running') {
    pushStage(
      'finished',
      '05',
      '结果落盘',
      formatDateTime(execution.value.end_time),
      diag.result_merge_ready ? '结果文件和报告链路已就绪。' : '执行已结束，但结果链路仍需补充检查。',
      diag.result_merge_ready ? 'green' : 'warning'
    )
  }
  return stages
})

const summaryMeta = computed(() => {
  const s = summary.value
  return [
    { label: '开始时间', value: formatDateTime(execution.value.start_time || execution.value.created_at) },
    { label: '结束时间', value: formatDateTime(execution.value.end_time) },
    { label: '采样跨度', value: formatDurationFromMs(s.sample_span_ms) },
    { label: '备注', value: execution.value.remarks || '-' }
  ]
})

// 详细统计数据
const detailStatGroups = computed(() => {
  const s = summary.value
  const live = liveMetrics.value || {}
  return [
    {
      title: '请求概况',
      items: [
        { name: '总样本数', value: formatNumber(isExecutionRunning.value ? live.total_requests : s.total_samples) },
        { name: '请求成功样本', value: formatNumber(isExecutionRunning.value ? live.success_requests : (s.request_success_samples ?? s.success_samples)) },
        { name: '请求错误样本', value: formatNumber(isExecutionRunning.value ? live.error_requests : (s.request_error_samples ?? s.error_samples)) },
        { name: '事务样本数', value: formatNumber(isExecutionRunning.value ? live.total_transactions : s.transaction_samples) },
        { name: '错误率', value: (isExecutionRunning.value ? live.error_rate : s.error_rate) !== undefined ? `${formatNumber(isExecutionRunning.value ? live.error_rate : s.error_rate)}%` : '-' },
        { name: '成功率', value: (isExecutionRunning.value ? live.success_rate : s.success_rate) !== undefined ? `${formatNumber(isExecutionRunning.value ? live.success_rate : s.success_rate)}%` : '-' },
        { name: summaryPrimaryThroughputLabel.value, value: isExecutionRunning.value ? `${formatNumber(live.current_primary_throughput)} ${livePrimaryThroughputUnit.value}` : summaryPrimaryThroughputValue.value !== null ? `${formatNumber(summaryPrimaryThroughputValue.value)} ${summaryPrimaryThroughputUnit.value}` : '-' },
        { name: '请求次数（次/秒）', value: isExecutionRunning.value ? `${formatNumber(live.current_request_rate)} req/s` : s.request_rate ? `${formatNumber(s.request_rate)} req/s` : '-' }
      ]
    },
    {
      title: '延迟分布',
      items: [
        { name: '平均响应时间', value: isExecutionRunning.value ? (live.avg_rt ? `${formatNumber(live.avg_rt)} ms` : '-') : s.avg_response_time ? `${formatNumber(s.avg_response_time)} ms` : '-' },
        { name: '最小响应时间', value: s.min_response_time !== undefined ? `${formatNumber(s.min_response_time)} ms` : '-' },
        { name: 'P50 响应时间', value: s.p50 ? `${formatNumber(s.p50)} ms` : '-' },
        { name: 'P90 响应时间', value: s.p90 ? `${formatNumber(s.p90)} ms` : '-' },
        { name: 'P95 响应时间', value: s.p95 ? `${formatNumber(s.p95)} ms` : '-' },
        { name: 'P99 响应时间', value: s.p99 ? `${formatNumber(s.p99)} ms` : '-' },
        { name: '最大响应时间', value: s.max_response_time !== undefined ? `${formatNumber(s.max_response_time)} ms` : '-' }
      ]
    },
    {
      title: '流量与传输',
      items: [
        { name: '执行时长', value: formatDuration(displayDurationSeconds.value) },
        { name: '采样跨度', value: formatDurationFromMs(s.sample_span_ms) },
        { name: '接收字节', value: formatBytes(s.received_bytes) },
        { name: '发送字节', value: formatBytes(s.sent_bytes) },
        { name: '接收速率', value: formatBytesRate(s.received_bytes_per_sec) },
        { name: '发送速率', value: formatBytesRate(s.sent_bytes_per_sec) }
      ]
    }
  ]
})

// 筛选后的错误记录
const filteredErrorRecordsRaw = computed(() => {
  let records = errorAnalysis.value?.records || []
  if (errorFilterCode.value) {
    records = records.filter(r => r.response_code === errorFilterCode.value)
  }
  if (errorFilterLabel.value) {
    records = records.filter(r => r.label === errorFilterLabel.value)
  }
  return records
})

// 筛选选项
const errorCodeOptions = computed(() => {
  const distribution = errorAnalysis.value?.response_code_distribution || []
  return distribution.map(item => ({
    label: `${item.code} (${item.count}次)`,
    value: item.code
  }))
})

const errorLabelOptions = computed(() => {
  const types = errorAnalysis.value?.error_types || []
  return types.map(item => ({
    label: `${item.label} (${item.count}次)`,
    value: item.label
  }))
})

// 分页的错误记录（基于筛选后数据）
const paginatedRecords = computed(() => {
  if (!filteredErrorRecordsRaw.value) return []
  const start = (errorPage.value - 1) * errorPageSize.value
  return filteredErrorRecordsRaw.value.slice(start, start + errorPageSize.value)
})

// 格式化样例错误数据，处理 null/undefined/"null" 值
const formatSampleErrors = (errors) => {
  if (!errors || !Array.isArray(errors)) return []
  return errors.map(err => ({
    ...err,
    failure_message: formatFailureMessage(err.failure_message)
  }))
}

// 格式化错误记录数据，处理 null/undefined/"null" 值
const formatErrorRecords = computed(() => {
  if (!paginatedRecords.value) return []
  return paginatedRecords.value.map(record => ({
    ...record,
    failure_message: formatFailureMessage(record.failure_message)
  }))
})

// 响应码颜色映射（用于饼图）
const getResponseCodeColor = (code) => {
  if (!code) return '#8c8c8c'  // 未知 - 中灰
  // 5xx 服务器错误
  if (code.startsWith('5')) return '#ff4d4f'  // 亮红
  // 4xx 客户端错误
  if (code.startsWith('4')) return '#faad14'  // 金黄
  // 3xx 重定向
  if (code.startsWith('3')) return '#1890ff'  // 蓝色
  // 2xx 成功
  if (code.startsWith('2')) return '#52c41a'  // 绿色
  // 连接/异常错误
  if (code.includes('Exception') || code.includes('error') || code.includes('Non HTTP')) return '#ff7a45'  // 橙红
  return '#8c8c8c'  // 其他 - 中灰
}

// 饼图数据计算
const pieSegments = computed(() => {
  const distribution = errorAnalysis.value?.response_code_distribution || []
  const radius = 40
  const circumference = 2 * Math.PI * radius  // ≈ 251.2
  let currentOffset = 0
  return distribution.map(item => {
    const length = (item.percentage / 100) * circumference
    const segment = {
      code: item.code,
      count: item.count,
      percentage: item.percentage,
      color: getResponseCodeColor(item.code),
      length,
      offset: -currentOffset
    }
    currentOffset += length
    return segment
  })
})

// 错误趋势数据转换
const errorTimelinePoints = computed(() => {
  const timeline = errorAnalysis.value?.error_timeline || []
  return timeline.map(p => ({
    timestamp: p.time_bucket,
    error_count: p.error_count,
    error_rate: p.error_rate
  }))
})

// 最新错误数（用于趋势图标题）
const latestErrorCount = computed(() => {
  const timeline = errorAnalysis.value?.error_timeline || []
  if (timeline.length === 0) return 0
  return timeline[timeline.length - 1].error_count
})

// Top错误信息
const topErrorMessages = computed(() => {
  return errorAnalysis.value?.top_error_messages || []
})

const errorDetailNotice = computed(() => {
  return errorAnalysis.value?.detail_storage_hint || ''
})

const hasCapturedDetailField = (fieldType) => {
  const fields = errorAnalysis.value?.available_detail_fields || []
  if (fieldType === 'requestHeaders') {
    return fields.includes('requestHeaders') || fields.includes('request_headers') || fields.includes('listener.request_headers')
  }
  if (fieldType === 'samplerData') {
    return fields.includes('samplerData') || fields.includes('sampler_data') || fields.includes('queryString') || fields.includes('listener.request_body')
  }
  if (fieldType === 'responseHeaders') {
    return fields.includes('responseHeaders') || fields.includes('response_headers') || fields.includes('listener.response_headers')
  }
  if (fieldType === 'responseData') {
    return fields.includes('responseData.onError') || fields.includes('responseData') || fields.includes('response_data') || fields.includes('listener.response_data')
  }
  return false
}

const getEmptyDetailText = (fieldType) => {
  if (!errorAnalysis.value?.detail_fields_available) {
    return '当前 JTL 未保存该类明细'
  }
  if (!hasCapturedDetailField(fieldType)) {
    return '当前结果文件未包含该字段'
  }
  return '该条样本未记录'
}

// 格式化失败原因，处理空值
const formatFailureMessage = (message) => {
  if (message === null || message === undefined || message === '' || message === 'null') {
    return '-'
  }
  return message
}

const formatResponsePreview = (record) => {
  const response = record?.response_preview || record?.response_data || record?.response_message || record?.failure_message
  if (!response || response === 'null') return '-'
  return response.replace(/\s+/g, ' ').trim()
}

const hasResponseDetail = (record) => {
  return Boolean(record?.response_data || record?.response_headers || record?.response_message || record?.failure_message)
}

const showResponseDetail = (record) => {
  responseDialogRecord.value = {
    response_message: record.response_message || '',
    response_data: record.response_data || '',
    response_headers: record.response_headers || '',
    failure_message: record.failure_message || ''
  }
  responseDialogVisible.value = true
}

// 显示完整失败原因
const showFailureMessage = (message) => {
  ElMessageBox.alert(message, '失败原因详情', {
    confirmButtonText: '关闭',
    customClass: 'failure-message-dialog'
  })
}

// 错误类型颜色映射
const getErrorCodeColor = (code) => {
  if (code?.startsWith('5')) return '#ff4d4f'  // 5xx 红色
  if (code?.startsWith('4')) return '#fa8c16'  // 4xx 橙色
  if (code?.startsWith('0') || code === 'Non HTTP response code') return '#722ed1'  // 连接错误 紫色
  return '#ff4d4f'
}

// 获取状态类型
const getStatusType = (status) => {
  const normalized = typeof status === 'object' ? (status?.status_tone || status?.status || 'info') : status
  const map = {
    running: 'primary',
    success: 'success',
    failed: 'danger',
    stopped: 'warning',
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
    success: '成功',
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

// 获取资源颜色（用于节点监控进度条）
const getResourceColor = (percent) => {
  if (percent >= 90) return '#ff4d4f'
  if (percent >= 70) return '#faad14'
  return '#52c41a'
}

// 获取节点监控数据
const fetchNodeMetrics = async () => {
  if (leavingPage || !hasValidExecutionId.value || execution.value?.status !== 'running') {
    nodeMetrics.value = []
    if (metricsTimer) {
      clearInterval(metricsTimer)
      metricsTimer = null
    }
    return
  }
  try {
    const res = await executionApi.getNodeMetrics(executionNumericId.value, { silent: true })
    nodeMetrics.value = (res.data?.nodes || []).map(node => {
      const stats = node.system_stats ? JSON.parse(node.system_stats) : null
      return { ...node, stats }
    })
  } catch (e) {
    console.error('获取节点监控失败:', e)
  }
}

// 启动节点监控轮询
const startMetricsPolling = () => {
  fetchNodeMetrics()
}

// 停止节点监控轮询
const stopMetricsPolling = () => {
  if (metricsTimer) {
    clearInterval(metricsTimer)
    metricsTimer = null
  }
  nodeMetrics.value = []
}

// 获取报告占位文本
const getReportPlaceholderText = () => {
  const status = execution.value.status
  if (status === 'running') return '测试运行中，报告将在执行完成后生成'
  if (status === 'failed') return '测试执行失败，无法生成报告'
  if (status === 'stopped') return '测试已停止，报告未生成'
  return '报告不可用'
}

// 格式化日期时间
const formatDateTime = (dateStr) => {
  return formatDateTimeInShanghai(dateStr, { withSeconds: true })
}

// 格式化数字
const formatNumber = (num) => {
  if (num === null || num === undefined || num === '') return '-'
  const n = parseFloat(num)
  if (isNaN(n)) return '-'
  if (Number.isInteger(n)) return n.toLocaleString()
  return n.toFixed(2).replace(/\.?0+$/, '')
}

const bytesKBPoints = computed(() => {
  return (liveMetrics.value?.points || []).map(p => ({
    ...p,
    bytes_per_sec: (p.bytes_per_sec || 0) / 1024
  }))
})

// 格式化字节速率为纯数值（用于图表显示）
const formatBytesRateValue = (bytesPerSec) => {
  if (bytesPerSec === null || bytesPerSec === undefined || bytesPerSec === 0) return '0'
  return (bytesPerSec / 1024).toFixed(1)
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

const getSecondsFromDateString = (value) => {
  const date = parseServerDateTime(value)
  if (!date) return 0
  return Math.floor(date.getTime() / 1000)
}

const displayDurationSeconds = computed(() => {
  const persistedDuration = Number(execution.value.duration || 0)
  const startSeconds = getSecondsFromDateString(execution.value.start_time || execution.value.created_at)
  const endSeconds = getSecondsFromDateString(execution.value.end_time)
  const summarySpanMs = Number(summary.value.sample_span_ms || 0)

  if (execution.value.status !== 'running') {
    if (persistedDuration > 0) return persistedDuration
    if (startSeconds && endSeconds && endSeconds >= startSeconds) {
      return endSeconds - startSeconds
    }
    if (summarySpanMs > 0) {
      return Math.max(Math.round(summarySpanMs / 1000), 0)
    }
    return persistedDuration
  }

  const liveDuration = Number(liveMetrics.value.duration_seconds || 0)
  if (liveDuration > 0) {
    return liveDuration
  }

  if (!startSeconds) return persistedDuration
  return Math.max(Math.floor(nowTick.value / 1000) - startSeconds, persistedDuration, 0)
})

const formatDurationFromMs = (ms) => {
  if (!ms || ms <= 0) return '-'
  if (ms < 1000) return `${formatNumber(ms)} ms`
  return formatDuration(Math.round(ms / 1000))
}

// 格式化字节数
const formatBytes = (bytes) => {
  if (bytes === null || bytes === undefined || bytes === 0) return '-'
  const b = parseFloat(bytes)
  if (isNaN(b) || b === 0) return '-'
  if (b < 1024) return `${b} B`
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(2)} KB`
  if (b < 1024 * 1024 * 1024) return `${(b / 1024 / 1024).toFixed(2)} MB`
  return `${(b / 1024 / 1024 / 1024).toFixed(2)} GB`
}

const formatBytesRate = (bytesPerSec) => {
  if (bytesPerSec === null || bytesPerSec === undefined || bytesPerSec === 0) return '-'
  return `${formatBytes(bytesPerSec)}/s`
}

// 获取错误率数值
const getErrorRateValue = (rate) => {
  if (rate === null || rate === undefined) return 0
  const num = parseFloat(rate)
  return isNaN(num) ? 0 : num
}

// 获取指标值样式类
const getMetricValueClass = (item) => {
  const classes = []
  if (item.name === '总样本数') classes.push('text-blue')
  if (item.name === '平均响应时间') classes.push('text-purple')
  if (item.name.includes('P')) classes.push('text-purple')
  if (item.name.includes('TPS') || item.name.includes('请求次数')) classes.push('text-green')
  if (item.name.includes('成功')) classes.push('text-green')
  if (item.name.includes('错误样本')) classes.push('text-red')
  if (item.name === '错误率') {
    const errorRate = getErrorRateValue(isExecutionRunning.value ? liveMetrics.value.error_rate : summary.value.error_rate)
    classes.push(errorRate > 5 ? 'text-red' : 'text-green')
  }
  return classes
}

// 获取执行详情
const fetchExecutionDetail = async () => {
  if (leavingPage || !hasValidExecutionId.value) return
  if (detailRefreshing.value) return
  detailRefreshing.value = true
  if (!execution.value?.id) {
    loading.value = true
  }
  try {
    const res = await executionApi.getDetail(executionNumericId.value, { silent: true })
    execution.value = res.data || {}
    // 加载完成后尝试加载基准线对比
    if (execution.value.status === 'success' && !execution.value.is_baseline && !baselineComparison.value && !baselineLoading.value) {
      loadBaselineComparison()
    }
  } catch (error) {
    console.error('获取执行详情失败:', error)
  } finally {
    loading.value = false
    detailRefreshing.value = false
  }
}

// 加载基准线对比数据
const loadBaselineComparison = async () => {
  if (!execution.value || execution.value.is_baseline) return
  baselineLoading.value = true
  try {
    // 获取该脚本的执行列表，查找基准线
    const listRes = await executionApi.getList({ 
      script_id: execution.value.script_id, 
      page_size: 100 
    })
    const baseline = (listRes.data?.list || []).find(e => e.is_baseline)
    if (baseline && baseline.id !== execution.value.id) {
      const res = await executionApi.compareExecutions(baseline.id, execution.value.id)
      baselineComparison.value = res.data
    }
  } catch (err) {
    console.warn('加载基准线对比失败', err)
    baselineComparison.value = null
  } finally {
    baselineLoading.value = false
  }
}

// 获取错误分析数据
const fetchErrors = async () => {
  if (leavingPage || !hasValidExecutionId.value) return
  if (errorLoading.value) return
  errorLoading.value = true
  try {
    const res = await executionApi.getErrors(executionNumericId.value, { silent: true })
    errorAnalysis.value = res.data
    errorPage.value = 1
    errorAnalysisLoaded.value = true
  } catch (e) {
    console.error('获取错误分析失败:', e)
  } finally {
    errorLoading.value = false
  }
}

const fetchLiveMetrics = async () => {
  if (leavingPage || !hasValidExecutionId.value) return
  if (liveRefreshing.value) return
  liveRefreshing.value = true
  try {
    const res = await executionApi.getLiveMetrics(executionNumericId.value, { silent: true })
    liveMetrics.value = res.data || { points: [] }
  } catch (error) {
    console.error('获取实时指标失败:', error)
  } finally {
    liveRefreshing.value = false
  }
}

const maybeFetchErrors = async (force = false) => {
  if (!force && execution.value.status === 'running' && !errorAnalysisLoaded.value) {
    return
  }
  await fetchErrors()
}

const expandedChartConfig = computed(() => {
  const configs = {
    primary_throughput: {
      title: livePrimaryThroughputChartTitle.value,
      value: formatNumber(primaryMetricValue('primary_throughput')),
      unit: livePrimaryThroughputUnit.value,
      subline: chartSubline('primary_throughput'),
      field: livePrimaryThroughputField.value,
      color: '#38bdf8'
    },
    request_rate: {
      title: '请求次数（次/秒）',
      value: formatNumber(primaryMetricValue('request_rate')),
      unit: 'req/s',
      subline: chartSubline('request_rate'),
      field: 'request_rate',
      color: '#22c55e'
    },
    avg_rt: {
      title: '平均RT',
      value: liveMetrics.value.avg_rt ? formatNumber(liveMetrics.value.avg_rt) : '-',
      unit: 'ms',
      subline: chartSubline('avg_rt'),
      field: 'avg_rt',
      color: '#a855f7'
    },
    response_time: {
      title: '响应时间',
      value: primaryMetricValue('response_time') ? formatNumber(primaryMetricValue('response_time')) : '-',
      unit: 'ms',
      subline: chartSubline('response_time'),
      field: 'avg_rt',
      color: '#ec4899'
    },
    concurrency: {
      title: '并发数',
      value: formatNumber(primaryMetricValue('concurrency')),
      unit: '',
      subline: chartSubline('concurrency'),
      field: 'concurrency',
      color: '#f59e0b'
    },
    success_rate: {
      title: '成功率',
      value: liveMetrics.value.success_rate !== undefined ? formatNumber(liveMetrics.value.success_rate) : '-',
      unit: '%',
      subline: `错误率 ${liveMetrics.value.error_rate !== undefined ? formatNumber(liveMetrics.value.error_rate) : '-'}%`,
      field: 'success_rate',
      color: '#84cc16'
    },
    p95_p99: {
      title: 'P95/P99 响应时间',
      value: formatNumber(primaryMetricValue('p95_rt')),
      unit: 'ms',
      subline: `P99: ${formatNumber(primaryMetricValue('p99_rt'))} ms`,
      field: 'p95_rt',
      color: '#f59e0b'
    },
    error_count: {
      title: '错误数趋势',
      value: String(primaryMetricValue('error_count') || 0),
      unit: 'errors',
      subline: '',
      field: 'error_count',
      color: '#ef4444'
    },
    bytes_per_sec: {
      title: '网络吞吐量',
      value: formatBytesRateValue(primaryMetricValue('bytes_per_sec')),
      unit: 'KB/s',
      subline: '',
      field: 'bytes_per_sec',
      color: '#22c55e'
    }
  }
  return configs[expandedChartKey.value] || null
})

const primaryMetricValue = (metricKey) => {
  const metrics = liveMetrics.value || {}
  switch (metricKey) {
    case 'primary_throughput':
      return isExecutionRunning.value ? metrics.current_primary_throughput : metrics.peak_primary_throughput
    case 'tps':
      return isExecutionRunning.value ? metrics.current_tps : metrics.peak_tps
    case 'request_rate':
      return isExecutionRunning.value ? metrics.current_request_rate : metrics.peak_request_rate
    case 'response_time':
      return isExecutionRunning.value ? metrics.current_rt : metrics.avg_rt
    case 'concurrency':
      return isExecutionRunning.value ? metrics.current_concurrency : metrics.peak_concurrency
    case 'p95_rt':
      return metrics.p95_rt
    case 'p99_rt':
      return metrics.p99_rt
    case 'error_count':
      return metrics.error_count
    case 'bytes_per_sec':
      return metrics.bytes_per_sec
    default:
      return null
  }
}

const chartSubline = (metricKey) => {
  const metrics = liveMetrics.value || {}
  switch (metricKey) {
    case 'primary_throughput':
      return isExecutionRunning.value
        ? `当前值 ${formatNumber(metrics.current_primary_throughput) || '-'} ${livePrimaryThroughputUnit.value} / 平均 ${formatNumber(metrics.avg_primary_throughput) || '-'} ${livePrimaryThroughputUnit.value} / 峰值 ${formatNumber(metrics.peak_primary_throughput) || '-'} ${livePrimaryThroughputUnit.value}`
        : `执行已结束，展示峰值 ${formatNumber(metrics.peak_primary_throughput) || '-'} ${livePrimaryThroughputUnit.value} / 平均 ${formatNumber(metrics.avg_primary_throughput) || '-'} ${livePrimaryThroughputUnit.value}`
    case 'tps':
      return isExecutionRunning.value
        ? `当前值 ${formatNumber(metrics.current_tps) || '-'} / 平均 ${formatNumber(metrics.avg_tps) || '-'} / 峰值 ${formatNumber(metrics.peak_tps) || '-'}`
        : `执行已结束，展示峰值 ${formatNumber(metrics.peak_tps) || '-'} / 平均 ${formatNumber(metrics.avg_tps) || '-'}`
    case 'request_rate':
      return isExecutionRunning.value
        ? `当前值 ${formatNumber(metrics.current_request_rate) || '-'} / 平均 ${formatNumber(metrics.avg_request_rate) || '-'} / 峰值 ${formatNumber(metrics.peak_request_rate) || '-'}`
        : `执行已结束，展示峰值 ${formatNumber(metrics.peak_request_rate) || '-'} / 平均 ${formatNumber(metrics.avg_request_rate) || '-'}`
    case 'avg_rt':
      return `整体平均 ${formatNumber(metrics.avg_rt) || '-'} ms`
    case 'response_time':
      return isExecutionRunning.value
        ? `当前响应时间 ${metrics.current_rt ? `${formatNumber(metrics.current_rt)} ms` : '-'}`
        : `执行已结束，展示整体平均 ${metrics.avg_rt ? `${formatNumber(metrics.avg_rt)} ms` : '-'}`
    case 'concurrency':
      return isExecutionRunning.value
        ? `当前并发 ${formatNumber(metrics.current_concurrency) || '-'} / 峰值 ${formatNumber(metrics.peak_concurrency) || '-'}`
        : `执行已结束，展示峰值 ${formatNumber(metrics.peak_concurrency) || '-'}`
    default:
      return ''
  }
}

const openExpandedChart = (chartKey) => {
  expandedChartKey.value = chartKey
  expandedChartVisible.value = true
}

const trimLogLines = (lines) => {
  const normalized = (lines || []).filter(line => line !== null && line !== undefined)
  if (normalized.length <= MAX_LOG_LINES) {
    logTrimmed.value = false
    return normalized
  }
  logTrimmed.value = true
  return normalized.slice(-MAX_LOG_LINES)
}

const setLogLines = (lines) => {
  logLinesData.value = trimLogLines(lines)
}

const appendLogLines = (lines) => {
  if (!lines || lines.length === 0) return
  logLinesData.value = trimLogLines(logLinesData.value.concat(lines))
}

const flushPendingLogLines = () => {
  if (pendingLogLines.value.length === 0) return
  appendLogLines(pendingLogLines.value)
  pendingLogLines.value = []
  nextTick(() => {
    scrollToBottom()
  })
}

const scheduleLogFlush = () => {
  if (logFlushTimer) return
  logFlushTimer = window.setTimeout(() => {
    logFlushTimer = null
    flushPendingLogLines()
  }, LOG_FLUSH_INTERVAL)
}

// 获取日志快照
const fetchLog = async () => {
  if (leavingPage || !hasValidExecutionId.value) return
  if (logSnapshotLoading.value) return
  if (logSnapshotController) {
    logSnapshotController.abort()
    logSnapshotController = null
  }
  logSnapshotController = new AbortController()
  logSnapshotLoading.value = true
  try {
    const res = await fetch(`/api/executions/${executionNumericId.value}/log?snapshot=1&tail=${MAX_LOG_LINES}`, {
      signal: logSnapshotController.signal
    })
    const text = await res.text()
    setLogLines(text.split('\n').filter(line => line.trim() !== ''))
    nextTick(() => {
      scrollToBottom()
    })
  } catch (error) {
    if (error?.name !== 'AbortError') {
      console.error('获取日志失败:', error)
      setLogLines(['获取日志失败'])
    }
  } finally {
    logSnapshotController = null
    logSnapshotLoading.value = false
  }
}

// 转义正则表达式特殊字符
const escapeRegExp = (str) => {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// 高亮搜索文本
const highlightLog = (line) => {
  if (!logSearch.value) return line
  const regex = new RegExp(`(${escapeRegExp(logSearch.value)})`, 'gi')
  return line.replace(regex, '<mark class="log-highlight">$1</mark>')
}

// 日志关键词颜色高亮
const colorizeLog = (line) => {
  // ERROR/FAIL 红色
  line = line.replace(/(ERROR|FAIL|FATAL|Exception)/gi, '<span class="log-error">$1</span>')
  // WARN 黄色  
  line = line.replace(/(WARN|WARNING)/gi, '<span class="log-warn">$1</span>')
  // INFO 蓝色
  line = line.replace(/(INFO)/gi, '<span class="log-info">$1</span>')
  // 数字高亮
  line = line.replace(/\b(\d+(?:\.\d+)?)\s*(ms|s|req\/s|%|KB|MB)\b/g, '<span class="log-number">$1 $2</span>')
  return line
}

// 格式化日志行（组合高亮和颜色）
const formatLogLine = (line) => {
  let formatted = colorizeLog(line)
  formatted = highlightLog(formatted)
  return formatted
}

// 计算搜索匹配数量
const matchCount = computed(() => {
  if (!logSearch.value || logLines.value.length === 0) return 0
  const regex = new RegExp(escapeRegExp(logSearch.value), 'gi')
  let count = 0
  logLines.value.forEach(line => {
    const matches = line.match(regex)
    if (matches) count += matches.length
  })
  return count
})

// 复制日志
const copyLogs = () => {
  const text = logLines.value.join('\n')
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('日志已复制到剪贴板')
  }).catch(() => {
    // fallback
    const textarea = document.createElement('textarea')
    textarea.value = text
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    ElMessage.success('日志已复制到剪贴板')
  })
}

// 导出日志
const exportLogs = () => {
  const text = logLines.value.join('\n')
  const blob = new Blob([text], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `execution_${executionId.value}_logs.txt`
  a.click()
  URL.revokeObjectURL(url)
}

// 连接 SSE
const connectSSE = () => {
  if (eventSource.value) {
    eventSource.value.close()
  }
  
  sseConnected.value = false
  isStreaming.value = true
  const es = new EventSource(`/api/executions/${executionId.value}/log`)
  
  es.onopen = () => {
    sseConnected.value = true
    reconnectAttempts.value = 0
  }
  
  es.onmessage = (event) => {
    pendingLogLines.value.push(event.data)
    scheduleLogFlush()
  }
  
  // 监听 complete 事件
  es.addEventListener('complete', () => {
    es.close()
    sseConnected.value = false
    isStreaming.value = false
  })
  
  es.onerror = () => {
    es.close()
    sseConnected.value = false
    isStreaming.value = false
    
    // 断线自动重连
    if (reconnectAttempts.value < maxReconnectAttempts && execution.value?.status === 'running') {
      reconnectAttempts.value++
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.value), 10000)
      setTimeout(connectSSE, delay)
    }
  }
  
  eventSource.value = es
}

const stopLogStream = () => {
  if (eventSource.value) {
    eventSource.value.close()
    eventSource.value = null
  }
  sseConnected.value = false
  isStreaming.value = false
  reconnectAttempts.value = 0
  if (logFlushTimer) {
    window.clearTimeout(logFlushTimer)
    logFlushTimer = null
  }
  flushPendingLogLines()
}

const toggleLogStream = () => {
  if (isStreaming.value) {
    stopLogStream()
    return
  }
  connectSSE()
}

// 滚动到底部
const scrollToBottom = () => {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

// 刷新日志
const refreshLog = () => {
  fetchLog()
}

const getAutoRefreshInterval = () => {
  if (!pageVisible.value) return 6000
  if (execution.value.status === 'running') return 2500
  return 8000
}

const handleVisibilityChange = () => {
  pageVisible.value = document.visibilityState === 'visible'
  if (!leavingPage) {
    if (pageVisible.value && hasValidExecutionId.value) {
      fetchExecutionDetail()
      fetchLiveMetrics()
    }
    setupAutoRefresh()
  }
}

// 停止执行
const handleStop = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要停止当前执行吗？',
      '确认停止',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    stopping.value = true
    await executionApi.stop(executionId.value)
    ElMessage.success('停止命令已发送')
    fetchExecutionDetail()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('停止执行失败:', error)
    }
  } finally {
    stopping.value = false
  }
}

// 返回
const goBack = () => {
  router.back()
}

// 下载 JTL 文件
const downloadJTL = () => {
  if (!hasResultFile.value) {
    ElMessage.warning('结果文件不存在')
    return
  }
  executionApi.downloadJTL(executionId.value)
}

// 下载 HTML 报告
const downloadReport = () => {
  if (!hasReportDir.value) {
    ElMessage.warning('报告目录不存在')
    return
  }
  executionApi.downloadReport(executionId.value)
}

// 下载错误记录
const downloadErrors = () => {
  if (!hasErrors.value) {
    ElMessage.warning('没有错误记录')
    return
  }
  executionApi.downloadErrors(executionId.value)
}

// 下载完整结果
const downloadAll = () => {
  executionApi.downloadAll(executionId.value)
}

// 设置自动刷新
const setupAutoRefresh = () => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
  }
  let detailTick = 0
  refreshTimer.value = setInterval(() => {
    if (leavingPage || !hasValidExecutionId.value) return
    detailTick += 1
    if (execution.value.status === 'running') {
      fetchLiveMetrics()
      if (detailTick % 2 === 0) {
        fetchExecutionDetail()
        fetchNodeMetrics()
      }
      if (detailTick % 3 === 0) {
        fetchErrors()
      }
    } else if (detailTick % 2 === 0) {
      fetchExecutionDetail()
    }
  }, getAutoRefreshInterval())
}

const setupDurationTicker = () => {
  if (durationTimer.value) {
    clearInterval(durationTimer.value)
    durationTimer.value = null
  }

  durationTimer.value = setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)
}

onMounted(() => {
  leavingPage = false
  Promise.allSettled([fetchExecutionDetail(), fetchLiveMetrics()]).then(() => {
    if (execution.value.status !== 'running') {
      fetchErrors()
    } else {
      // 执行运行中时启动节点监控
      startMetricsPolling()
      fetchErrors()
    }
    setupAutoRefresh()
    setupDurationTicker()
    setTimeout(() => {
      if (!leavingPage) {
        fetchLog()
      }
    }, 120)
  })
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', handleVisibilityChange)
  }
})

watch(() => execution.value.status, (status, prevStatus) => {
  if (status && status !== 'running' && status !== prevStatus && !errorAnalysisLoaded.value) {
    fetchErrors()
  }
  // 节点监控启停逻辑
  if (status === 'running' && prevStatus !== 'running') {
    startMetricsPolling()
  } else if (status !== 'running' && prevStatus === 'running') {
    stopMetricsPolling()
  }
})

onBeforeUnmount(() => {
  leavingPage = true
  stopLogStream()
  stopMetricsPolling()
  if (logSnapshotController) {
    logSnapshotController.abort()
    logSnapshotController = null
  }
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }
  if (durationTimer.value) {
    clearInterval(durationTimer.value)
    durationTimer.value = null
  }
  if (typeof document !== 'undefined') {
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  }
})
</script>

<style scoped lang="scss">
.execution-detail-page {
  padding: 20px;
}

.conclusion-panel {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 20px;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: linear-gradient(135deg, rgba(34, 197, 94, 0.08), rgba(255, 255, 255, 0.03));
}

.conclusion-panel.is-warning {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.1), rgba(255, 255, 255, 0.03));
}

.conclusion-panel.is-danger {
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.12), rgba(255, 255, 255, 0.03));
}

.conclusion-main {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.conclusion-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.conclusion-badge {
  padding: 6px 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  background: rgba(59, 130, 246, 0.14);
  color: #5ab0ff;
}

.conclusion-badge.is-warning {
  background: rgba(245, 158, 11, 0.14);
  color: #ffbf5f;
}

.conclusion-badge.is-danger {
  background: rgba(239, 68, 68, 0.14);
  color: #ff7a7a;
}

.conclusion-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
}

.conclusion-summary {
  font-size: 14px;
  line-height: 1.8;
  color: var(--text-secondary);
}

.conclusion-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.conclusion-list-card,
.timeline-card,
.sampler-overview-card {
  padding: 16px 18px;
  border-radius: var(--radius-md);
  background: rgba(10, 18, 32, 0.55);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.conclusion-list-title {
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.conclusion-list-item {
  position: relative;
  padding-left: 14px;
  margin-bottom: 8px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-secondary);
}

.conclusion-list-item::before {
  content: '';
  position: absolute;
  left: 0;
  top: 9px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent-blue);
}

.timeline-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 14px;
}

.timeline-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.timeline-card.is-green {
  border-color: rgba(34, 197, 94, 0.22);
}

.timeline-card.is-warning {
  border-color: rgba(245, 158, 11, 0.22);
}

.timeline-step {
  font-size: 11px;
  letter-spacing: 0.12em;
  color: var(--accent-blue);
}

.timeline-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.timeline-time {
  font-size: 13px;
  color: var(--text-secondary);
}

.timeline-desc {
  font-size: 12px;
  line-height: 1.7;
  color: rgba(214, 222, 237, 0.7);
}

.sampler-overview-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 16px;
}

.sampler-overview-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.sampler-overview-name {
  margin-top: 8px;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.sampler-overview-value {
  margin-top: 6px;
  font-size: 24px;
  font-weight: 700;
  color: #5ab0ff;
}

.sampler-overview-caption {
  margin-top: 8px;
  font-size: 12px;
  line-height: 1.6;
  color: rgba(214, 222, 237, 0.68);
  word-break: break-all;
}

.sampler-table :deep(.el-table__cell) {
  vertical-align: top;
}

// 区域卡片
.section-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid rgba(255, 255, 255, 0.06);
  padding: 24px;
  margin-bottom: 24px;
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
  margin-bottom: 16px;
}

.section-header {
  margin-bottom: 20px;
}

// 顶部信息栏
.header-section {
  padding: 16px 24px;
  margin-bottom: 24px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: rgba(0, 102, 255, 0.1);
  border: 1px solid rgba(0, 102, 255, 0.2);
  border-radius: var(--radius-md);
  color: var(--accent-blue);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.25s ease;
}

.back-btn:hover {
  background: rgba(0, 102, 255, 0.15);
  border-color: rgba(0, 102, 255, 0.3);
}

.script-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.script-name {
  font-size: 18px;
  font-weight: 600;
  margin: 0;
  color: var(--text-primary);
}

.status-tag {
  font-weight: 500;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.export-dropdown {
  :deep(.el-button) {
    background: var(--accent-blue);
    border-color: var(--accent-blue);
    
    &:hover {
      background: #4d8aff;
      border-color: #4d8aff;
    }
  }
}

.export-menu {
  min-width: 180px;
  
  :deep(.el-dropdown-menu__item) {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    
    .el-icon {
      font-size: 16px;
    }
    
    &.is-disabled {
      opacity: 0.5;
    }
    
    &:not(.is-disabled):hover {
      color: var(--accent-blue);
    }
  }
}

.execution-time {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  font-family: 'Consolas', monospace;
}

// 执行概览
.overview-panel {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr);
  gap: 18px;
}

// 基准线对比卡片
.baseline-compare-card {
  background: rgba(234, 179, 8, 0.08);
  border: 1px solid rgba(234, 179, 8, 0.2);
  border-radius: 12px;
  padding: 16px 20px;
  margin-bottom: 20px;
  
  .baseline-header {
    margin-bottom: 14px;
    
    .baseline-title {
      display: flex;
      align-items: center;
      gap: 10px;
      font-size: 14px;
      font-weight: 600;
      color: rgba(255, 255, 255, 0.9);
      
      .el-icon {
        font-size: 16px;
        color: #eab308;
      }
      
      .baseline-tag {
        background: rgba(234, 179, 8, 0.15);
        border-color: rgba(234, 179, 8, 0.3);
        color: #eab308;
        margin-left: 4px;
      }
    }
  }
  
  .baseline-metrics {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
  }
  
  .baseline-metric-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 10px 16px;
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.04);
    min-width: 100px;
    
    .metric-name {
      font-size: 12px;
      color: rgba(255, 255, 255, 0.6);
      margin-bottom: 6px;
      text-transform: uppercase;
      letter-spacing: 0.04em;
    }
    
    .metric-change {
      font-size: 16px;
      font-weight: 700;
      font-family: 'Consolas', 'Monaco', monospace;
      color: rgba(255, 255, 255, 0.5);
      display: flex;
      align-items: center;
      gap: 4px;
      
      .arrow-up {
        color: #22c55e;
        font-size: 12px;
      }
      
      .arrow-down {
        color: #ef4444;
        font-size: 12px;
      }
      
      .arrow-flat {
        color: rgba(255, 255, 255, 0.3);
        font-size: 12px;
      }
      
      &.improved {
        color: #22c55e;
      }
      
      &.worsened {
        color: #ef4444;
      }
    }
  }
}

.live-metrics-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.live-summary-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.live-summary-label {
  color: var(--text-secondary);
  font-size: 12px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.live-summary-value {
  color: var(--text-primary);
  font-size: 22px;
  font-weight: 700;
  font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
}

.live-charts-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.overview-hero {
  padding: 20px;
  border-radius: 18px;
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.14), transparent 36%),
    radial-gradient(circle at bottom right, rgba(37, 99, 235, 0.12), transparent 32%),
    rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.overview-status-strip {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 36px;
  padding: 0 14px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.72);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.overview-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.overview-status-dot.is-success { background: #22c55e; }
.overview-status-dot.is-info { background: #38bdf8; }
.overview-status-dot.is-warning { background: #f59e0b; }
.overview-status-dot.is-danger { background: #ef4444; }

.overview-status-text,
.overview-status-note {
  font-size: 12px;
  color: var(--text-primary);
}

.overview-status-text {
  font-weight: 700;
}

.overview-status-note {
  color: var(--text-secondary);
}

.overview-status-divider {
  width: 1px;
  height: 12px;
  background: rgba(255, 255, 255, 0.08);
}

.overview-primary-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-top: 20px;
}

.overview-primary-card,
.overview-mini-card {
  border-radius: 18px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(10, 17, 31, 0.72);
}

.overview-primary-card {
  min-height: 188px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.overview-primary-card.is-throughput {
  box-shadow: inset 0 1px 0 rgba(34, 197, 94, 0.08);
}

.overview-primary-card.is-latency {
  box-shadow: inset 0 1px 0 rgba(168, 85, 247, 0.08);
}

.overview-card-label,
.overview-mini-label {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-secondary);
}

.overview-card-value,
.overview-mini-value {
  font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
  color: var(--text-primary);
}

.overview-card-value {
  font-size: clamp(34px, 4vw, 54px);
  font-weight: 700;
  line-height: 1.02;
  margin: 18px 0;
  word-break: break-word;
}

.overview-card-caption,
.overview-mini-caption {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.overview-mini-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.overview-mini-card {
  min-height: 124px;
  padding: 18px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.overview-mini-value {
  font-size: 26px;
  font-weight: 700;
  line-height: 1.1;
  margin: 10px 0 8px;
}

.text-blue { color: var(--accent-blue); }
.text-purple { color: var(--accent-purple); }
.text-green { color: var(--accent-green); }
.text-red { color: var(--accent-red); }
.text-warning { color: #f59e0b; }

.diagnostic-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.diagnostic-card {
  min-height: 128px;
  padding: 18px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.diagnostic-label {
  color: var(--text-secondary);
  font-size: 12px;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.diagnostic-value {
  color: var(--text-primary);
  font-size: 22px;
  font-weight: 700;
  line-height: 1.1;
  word-break: break-word;
}

.diagnostic-caption {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.diagnostic-warning-stack {
  margin-top: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.diagnostic-warning-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 12px;
  color: #fbbf24;
  background: rgba(245, 158, 11, 0.08);
  border: 1px solid rgba(245, 158, 11, 0.18);
  line-height: 1.6;
}

.diagnostic-warning-item .el-icon {
  margin-top: 2px;
  flex-shrink: 0;
}

// 详细统计
.stats-meta-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.stats-meta-card,
.detail-group-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 14px;
}

.stats-meta-card {
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.stats-meta-label,
.detail-group-title {
  color: var(--text-secondary);
  font-size: 12px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.stats-meta-value {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
}

.detail-groups {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.detail-group-card {
  padding: 16px;
}

.detail-group-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 14px;
}

.detail-group-item {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

.detail-group-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.metric-name {
  color: var(--text-secondary);
  font-size: 14px;
  flex-shrink: 0;
}

.metric-value {
  font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
  font-size: 15px;
  font-weight: 600;
  text-align: right;
}

// JMeter报告区域
.report-wrapper {
  padding: 0;
}

.report-container {
  width: 100%;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.report-iframe {
  width: 100%;
  height: 600px;
  border: none;
  background: white;
}

.report-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  border: 1px dashed rgba(255, 255, 255, 0.06);
}

.placeholder-icon {
  color: var(--text-secondary);
  margin-bottom: 16px;
  opacity: 0.5;
}

.placeholder-text {
  color: var(--text-secondary);
  font-size: 14px;
  margin: 0;
}

// 终端区域
.terminal-section {
  padding: 24px;
}

.terminal-window {
  background: #0d1117;
  border-radius: var(--radius-lg);
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--bg-secondary);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.terminal-controls {
  display: flex;
  gap: 8px;
}

.control-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.control-dot.red { background: #ff5f56; }
.control-dot.yellow { background: #ffbd2e; }
.control-dot.green { background: #27c93f; }

.terminal-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.terminal-actions {
  display: flex;
  gap: 8px;
}

.terminal-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  margin: 0 20px;
}

.log-search-input {
  width: 200px;
}

.log-search-input :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.05) !important;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
  border-radius: 6px !important;
  box-shadow: none !important;
}

.log-search-input :deep(.el-input__inner) {
  color: var(--text-primary) !important;
  font-size: 13px;
}

.log-search-input :deep(.el-input__inner::placeholder) {
  color: var(--text-secondary) !important;
}

.search-count {
  color: var(--text-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.terminal-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.terminal-btn:hover {
  background: rgba(0, 102, 255, 0.1);
  border-color: rgba(0, 102, 255, 0.3);
  color: var(--accent-blue);
}

.terminal-body {
  height: 400px;
  background: #0d1117;
  overflow-y: auto;
  padding: 12px 16px;
  font-family: 'Consolas', 'Fira Code', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.terminal-body::-webkit-scrollbar {
  width: 8px;
}

.terminal-body::-webkit-scrollbar-track {
  background: #0d1117;
}

.terminal-body::-webkit-scrollbar-thumb {
  background: #2a3441;
  border-radius: 4px;
}

.terminal-body::-webkit-scrollbar-thumb:hover {
  background: #3a4551;
}

.terminal-empty {
  color: #475569;
  text-align: center;
  padding: 40px;
}

.log-line {
  display: flex;
  gap: 12px;
  color: var(--accent-green);
}

.log-line-number {
  color: #475569;
  user-select: none;
  min-width: 40px;
  text-align: right;
}

.log-line-content {
  flex: 1;
  word-break: break-all;
}

// 日志颜色高亮
.log-error {
  color: #ff453a;
  font-weight: bold;
}

.log-warn {
  color: #ff9f43;
}

.log-info {
  color: #0a84ff;
}

.log-number {
  color: #30d158;
}

.log-highlight {
  background: rgba(255, 214, 10, 0.3);
  color: #ffd60a;
  border-radius: 2px;
  padding: 0 2px;
}

.cursor-line::after {
  content: '▋';
  animation: blink 1s step-end infinite;
  margin-left: 2px;
}

@keyframes blink {
  50% { opacity: 0; }
}

.terminal-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  background: var(--bg-secondary);
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  font-size: 12px;
}

.connection-status {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-indicator.connected {
  background: var(--accent-green);
}

.status-indicator.disconnected {
  background: var(--accent-red);
}

.connection-status .status-text {
  color: var(--text-secondary);
}

.log-stats {
  color: var(--text-secondary);
  font-family: 'Consolas', monospace;
}

// 错误分析区块样式
.error-analysis-section {
  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;

    .section-title {
      display: flex;
      align-items: center;
      gap: 8px;
      color: #ff6b6b;
      flex: 1;
    }
  }

  .no-errors {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 32px;
    color: #52c41a;
    font-size: 15px;

    .success-icon {
      font-size: 20px;
    }
  }

  .error-stats-row {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 12px;
    margin-bottom: 20px;
  }

  .error-stat-card {
    background: rgba(255, 77, 79, 0.08);
    border: 1px solid rgba(255, 77, 79, 0.15);
    border-radius: 10px;
    padding: 16px;
    text-align: center;

    .error-stat-value {
      font-size: 24px;
      font-weight: 700;
      color: #ff4d4f;
      margin-bottom: 4px;
    }

    .error-stat-label {
      font-size: 12px;
      color: #808080;
    }
  }

  .error-tabs {
    :deep(.el-tabs__header) {
      margin-bottom: 16px;
    }

    :deep(.el-tabs__item) {
      color: var(--text-secondary);

      &.is-active {
        color: #ff6b6b;
      }
    }

    :deep(.el-tabs__active-bar) {
      background-color: #ff6b6b;
    }
  }

  .error-count {
    color: #ff4d4f;
    font-weight: 600;
  }

  .sample-errors-wrapper {
    padding: 16px;
    background: rgba(255, 77, 79, 0.04);
    border-radius: 8px;
    margin: 0 24px;

    .sample-errors-title {
      font-size: 13px;
      color: var(--text-secondary);
      margin-bottom: 12px;
      font-weight: 500;
    }

    .sample-errors-table {
      background: transparent;

      :deep(.el-table__header-wrapper) {
        display: none;
      }

      :deep(.el-table__row) {
        background: transparent;

        &:hover > td {
          background: rgba(255, 77, 79, 0.06);
        }
      }

      :deep(td) {
        border-bottom: 1px solid rgba(255, 77, 79, 0.08);
        padding: 8px 0;
      }
    }
  }

  .error-pagination {
    margin-top: 16px;
    justify-content: flex-end;

    :deep(.el-pagination__total),
    :deep(.el-pagination__sizes) {
      color: var(--text-secondary);
    }
  }

  // 错误分布图表行
  .error-charts-row {
    display: flex;
    gap: 16px;
    margin-bottom: 20px;

    @media (max-width: 900px) {
      flex-direction: column;
    }
  }

  .error-pie-section {
    flex: 0 0 40%;
    min-width: 320px;

    @media (max-width: 900px) {
      flex: 1;
      min-width: auto;
    }
  }

  .error-timeline-section {
    flex: 1;
    min-width: 0;
  }

  .error-chart-card {
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 12px;
    padding: 16px;
    height: 100%;
  }

  .error-chart-title {
    font-size: 13px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin-bottom: 12px;
  }

  // 饼图样式
  .error-pie-content {
    display: flex;
    align-items: center;
    gap: 20px;
  }

  .error-pie-chart {
    width: 140px;
    height: 140px;
    flex-shrink: 0;

    circle {
      transition: stroke-dasharray 0.3s ease;
    }

    .pie-center-label {
      fill: var(--text-secondary);
      font-size: 8px;
      text-transform: uppercase;
      letter-spacing: 0.04em;
    }

    .pie-center-value {
      fill: var(--text-primary);
      font-size: 14px;
      font-weight: 700;
      font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
    }
  }

  .error-pie-legend {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 140px;
    overflow-y: auto;

    &::-webkit-scrollbar {
      width: 4px;
    }

    &::-webkit-scrollbar-track {
      background: transparent;
    }

    &::-webkit-scrollbar-thumb {
      background: rgba(255, 255, 255, 0.1);
      border-radius: 2px;
    }

    .legend-item {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;

      .legend-color {
        width: 10px;
        height: 10px;
        border-radius: 2px;
        flex-shrink: 0;
      }

      .legend-code {
        color: var(--text-primary);
        font-weight: 500;
        min-width: 40px;
        max-width: 80px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .legend-count {
        color: var(--text-secondary);
        flex: 1;
        text-align: right;
      }

      .legend-percent {
        color: var(--text-secondary);
        min-width: 40px;
        text-align: right;
        font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
      }
    }
  }

  // Top 错误信息
  .top-errors-section {
    margin-bottom: 20px;

    .top-errors-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }

    .top-error-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 10px 12px;
      background: rgba(255, 255, 255, 0.02);
      border-radius: 8px;
      border: 1px solid rgba(255, 255, 255, 0.04);

      .rank {
        width: 22px;
        height: 22px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(239, 68, 68, 0.15);
        color: #ef4444;
        font-size: 11px;
        font-weight: 700;
        border-radius: 4px;
        flex-shrink: 0;
      }

      .message {
        flex: 1;
        color: var(--text-primary);
        font-size: 13px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        cursor: default;
      }

      .count {
        color: var(--text-secondary);
        font-size: 12px;
        font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
        flex-shrink: 0;
      }
    }
  }

  // 错误筛选栏
  .error-filters-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
    flex-wrap: wrap;

    .error-filter-select {
      width: 180px;

      :deep(.el-input__wrapper) {
        background: rgba(255, 255, 255, 0.05) !important;
        border: 1px solid rgba(255, 255, 255, 0.08) !important;
        box-shadow: none !important;
      }

      :deep(.el-input__inner) {
        color: var(--text-primary) !important;
      }

      :deep(.el-input__inner::placeholder) {
        color: var(--text-secondary) !important;
      }
    }

    .error-filter-result {
      color: var(--text-secondary);
      font-size: 12px;
      margin-left: auto;
    }
  }

  // 失败原因单元格样式
  .response-preview-cell,
  .failure-message-cell {
    display: block;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: pointer;
    color: var(--text-primary);
    transition: color 0.2s ease;

    &:hover:not(.is-empty) {
      color: var(--accent-blue);
      text-decoration: underline;
    }

    &.is-empty {
      color: var(--text-secondary);
      cursor: default;
    }
  }
}

.response-detail-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.response-detail-section {
  display: flex;
  flex-direction: column;
  gap: 8px;

  pre {
    margin: 0;
    padding: 14px 16px;
    max-height: 240px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-word;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.06);
    color: var(--text-primary);
    font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
    font-size: 12px;
    line-height: 1.7;
  }
}

.response-detail-label {
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

:deep(.chart-detail-dialog .el-dialog) {
  background: var(--bg-card);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 20px;
}

:deep(.chart-detail-dialog .el-dialog__header) {
  margin-right: 0;
  padding: 22px 24px 8px;
}

:deep(.chart-detail-dialog .el-dialog__body) {
  padding: 0 24px 24px;
}

// 错误详情弹窗样式
.error-detail-dialog {
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
    padding: 0;
  }
}

.error-detail-tabs {
  :deep(.el-tabs__header) {
    margin: 0;
    padding: 0 24px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  }

  :deep(.el-tabs__item) {
    color: var(--text-secondary);
    font-weight: 500;

    &.is-active {
      color: var(--accent-blue);
    }
  }

  :deep(.el-tabs__active-bar) {
    background-color: var(--accent-blue);
  }
}

.error-detail-notice {
  margin: 16px 24px 0;
  padding: 14px 16px;
  border-radius: 12px;
  border: 1px solid rgba(32, 184, 255, 0.18);
  background: rgba(32, 184, 255, 0.08);
}

.error-detail-notice.is-warning {
  border-color: rgba(250, 173, 20, 0.22);
  background: rgba(250, 173, 20, 0.08);
}

.error-detail-notice-title {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
}

.error-detail-notice-text {
  margin-top: 6px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.error-detail-notice-meta {
  margin-top: 8px;
  color: rgba(184, 194, 217, 0.78);
  font-size: 12px;
  line-height: 1.6;
  word-break: break-all;
}

.error-detail-content {
  padding: 20px 24px;
}

.error-detail-section {
  margin-bottom: 20px;

  &:last-child {
    margin-bottom: 0;
  }
}

.error-detail-label {
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 8px;
}

.error-detail-code {
  margin: 0;
  padding: 14px 16px;
  max-height: 300px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  border-radius: 10px;
  background: #0d1117;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--text-primary);
  font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
  font-size: 12px;
  line-height: 1.7;

  &.scrollable {
    max-height: 300px;
    overflow-y: auto;
  }

  &::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }

  &::-webkit-scrollbar-track {
    background: #0d1117;
  }

  &::-webkit-scrollbar-thumb {
    background: #2a3441;
    border-radius: 4px;
  }

  &::-webkit-scrollbar-thumb:hover {
    background: #3a4551;
  }
}

.error-detail-empty {
  padding: 20px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 13px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 10px;
}

.error-detail-value {
  display: flex;
  align-items: center;
  gap: 12px;
}

.error-detail-message {
  color: var(--text-secondary);
  font-size: 13px;
}

.error-detail-timing-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.error-detail-timing-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  padding: 16px;
  text-align: center;
}

.error-detail-timing-label {
  color: var(--text-secondary);
  font-size: 12px;
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.error-detail-timing-value {
  color: var(--text-primary);
  font-size: 20px;
  font-weight: 700;
  font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
}

.error-detail-data-row {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 10px;
  padding: 16px;
}

.error-detail-data-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.error-detail-data-label {
  color: var(--text-secondary);
  font-size: 13px;
}

.error-detail-data-value {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
  font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
}

// 响应式
@media (max-width: 1400px) {
  .overview-panel {
    grid-template-columns: 1fr;
  }

  .diagnostic-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .live-charts-grid {
    grid-template-columns: 1fr;
  }

  .detail-groups {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 1024px) {
  .overview-primary-grid,
  .overview-mini-grid {
    grid-template-columns: 1fr;
  }

  .stats-meta-row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .live-metrics-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .overview-hero,
  .overview-mini-card,
  .overview-primary-card,
  .diagnostic-card {
    padding: 16px;
  }

  .stats-meta-row {
    grid-template-columns: 1fr;
  }

  .diagnostic-grid,
  .live-metrics-summary {
    grid-template-columns: 1fr;
  }

  .overview-status-strip {
    width: 100%;
    justify-content: flex-start;
    flex-wrap: wrap;
    padding: 10px 14px;
    border-radius: 14px;
  }
  
  .header-content {
    flex-direction: column;
    gap: 16px;
    align-items: flex-start;
  }
  
  .header-right {
    width: 100%;
    justify-content: space-between;
  }
}

// 节点监控面板样式
.node-metrics-panel {
  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;
  }
  
  .node-card {
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 10px;
    padding: 16px;
    transition: all 0.25s ease;
    
    &:hover {
      background: rgba(255, 255, 255, 0.06);
      border-color: rgba(255, 255, 255, 0.12);
    }
  }
  
  .node-header {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 14px;
    padding-bottom: 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    
    .node-name {
      color: var(--text-primary);
      font-weight: 600;
      font-size: 15px;
    }
    
    .node-role {
      color: var(--text-secondary);
      font-size: 12px;
      background: rgba(255, 255, 255, 0.06);
      padding: 2px 8px;
      border-radius: 4px;
    }
    
    .node-status {
      margin-left: auto;
      font-size: 12px;
      font-weight: 500;
      padding: 2px 8px;
      border-radius: 4px;
      
      &.online {
        color: #52c41a;
        background: rgba(82, 196, 26, 0.15);
      }
      
      &.offline {
        color: #ff4d4f;
        background: rgba(255, 77, 79, 0.15);
      }
    }
  }
  
  .node-stats {
    .stat-item {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 10px;
      
      &:last-child {
        margin-bottom: 0;
      }
      
      .stat-label {
        color: var(--text-secondary);
        font-size: 13px;
        width: 48px;
        flex-shrink: 0;
      }
      
      :deep(.el-progress) {
        flex: 1;
      }
      
      .stat-value {
        color: var(--text-primary);
        font-size: 14px;
        font-weight: 500;
        min-width: 40px;
        text-align: right;
      }
    }
  }
  
  .node-offline {
    color: var(--text-secondary);
    font-size: 13px;
    text-align: center;
    padding: 20px 0;
    font-style: italic;
  }
}

// 响应式适配
@media (max-width: 768px) {
  .node-metrics-panel {
    .metrics-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style>

<!-- 全屏报告 Dialog 样式 - 非 scoped 因为 el-dialog 使用 Teleport 渲染到 body -->
<style lang="scss">
.report-fullscreen-dialog {
  .el-dialog__header {
    padding: 12px 20px;
    margin: 0;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .el-dialog__body {
    padding: 0 !important;
    height: calc(100vh - 55px) !important;
    overflow: hidden;
  }
}

.report-fullscreen-iframe {
  width: 100%;
  height: 100%;
  border: none;
  background: white;
  display: block;
}
</style>
