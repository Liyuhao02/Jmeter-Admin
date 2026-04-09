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
              :type="getStatusType(execution.status)"
              size="small"
              class="status-tag"
            >
              {{ getStatusText(execution.status) }}
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
      <div class="overview-panel">
        <div class="overview-hero">
          <div class="overview-status-strip">
            <span class="overview-status-dot" :class="`is-${overviewStatusTone}`"></span>
            <span class="overview-status-text">{{ getStatusText(execution.status) }}</span>
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

    <div class="section-card">
      <div class="section-header">
        <div class="section-label">LIVE METRICS</div>
        <div class="section-title">实时趋势</div>
      </div>
      <div class="live-metrics-summary">
        <div class="live-summary-card">
          <span class="live-summary-label">{{ isExecutionRunning ? '当前 TPS' : '峰值 TPS' }}</span>
          <span class="live-summary-value text-blue">{{ formatNumber(primaryMetricValue('tps')) || '-' }}</span>
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
          title="TPS 趋势"
          :value="formatNumber(primaryMetricValue('tps'))"
          unit="req/s"
          :subline="chartSubline('tps')"
          :points="liveMetrics.points || []"
          field="tps"
          color="#38bdf8"
          :show-expand="true"
          :max-x-ticks="4"
          @expand="openExpandedChart('tps')"
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
              :total="errorAnalysis.records?.length || 0"
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
  VideoPause
} from '@element-plus/icons-vue'
import { executionApi } from '@/api/execution'
import { formatDateTimeInShanghai, parseServerDateTime } from '@/utils/datetime'
import MetricTrendChart from '@/components/MetricTrendChart.vue'

const route = useRoute()
const router = useRouter()
const executionId = computed(() => route.params.id)

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
const liveMetrics = ref({ points: [] })
const detailRefreshing = ref(false)
const liveRefreshing = ref(false)
const errorAnalysisLoaded = ref(false)
const LIVE_REFRESH_INTERVAL = 3000
const nowTick = ref(Date.now())
const expandedChartKey = ref('')
const expandedChartVisible = ref(false)

// 错误分析相关
const errorAnalysis = ref(null)
const errorLoading = ref(false)
const errorActiveTab = ref('types') // types | records
const errorPage = ref(1)
const errorPageSize = ref(50)
const selectedErrorType = ref(null)
const responseDialogVisible = ref(false)
const responseDialogRecord = ref({})

// 报告全屏查看
const reportFullscreen = ref(false)

// 错误详情弹窗
const errorDetailVisible = ref(false)
const currentErrorRecord = ref(null)
const errorDetailTab = ref('request')

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

const summaryCounts = computed(() => {
  const total = isExecutionRunning.value
    ? Number(liveMetrics.value.total_requests || 0)
    : Number(summary.value.total_samples || 0)
  const errors = isExecutionRunning.value
    ? Number(liveMetrics.value.error_requests ?? errorAnalysis.value?.total_errors ?? 0)
    : Number(summary.value.error_samples ?? errorAnalysis.value?.total_errors ?? 0)
  const success = Number(summary.value.success_samples ?? Math.max(total - errors, 0))
  return { total, success, errors }
})

const overviewStatusTone = computed(() => {
  if (execution.value.status === 'failed') return 'danger'
  if (execution.value.status === 'running') return 'info'
  if (getErrorRateValue(summary.value.error_rate) > 5) return 'warning'
  return 'success'
})

const overviewStatusNote = computed(() => {
  if (execution.value.status === 'running') return '实时指标持续刷新中'
  if (execution.value.status === 'failed') return '本次执行存在明显失败样本'
  if (getErrorRateValue(summary.value.error_rate) > 0) return '执行完成，但存在错误请求'
  return '执行完成，整体表现稳定'
})

const overviewPrimaryMetrics = computed(() => {
  const s = summary.value
  const live = liveMetrics.value || {}
  const throughputValue = isExecutionRunning.value
    ? live.current_request_rate
    : s.throughput
  const avgResponseTime = isExecutionRunning.value
    ? live.avg_rt
    : s.avg_response_time
  const throughputCaption = isExecutionRunning.value
    ? `当前 ${formatNumber(live.current_request_rate)} req/s / 平均 ${formatNumber(live.avg_request_rate)} req/s`
    : s.sample_span_ms ? `基于 ${formatDurationFromMs(s.sample_span_ms)} 真实采样跨度` : '等待采样结果'
  const latencyCaption = isExecutionRunning.value
    ? `当前 ${live.current_rt ? `${formatNumber(live.current_rt)} ms` : '-'} / 峰值TPS ${formatNumber(live.peak_tps)}`
    : s.p95 ? `P95 ${formatNumber(s.p95)} ms / P99 ${formatNumber(s.p99)} ms` : '等待延迟分布统计'
  return [
    {
      key: 'throughput',
      label: '吞吐量',
      value: throughputValue ? `${formatNumber(throughputValue)} req/s` : '-',
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
        { name: '成功样本', value: formatNumber(isExecutionRunning.value ? live.success_requests : s.success_samples) },
        { name: '错误样本', value: formatNumber(isExecutionRunning.value ? live.error_requests : s.error_samples) },
        { name: '错误率', value: (isExecutionRunning.value ? live.error_rate : s.error_rate) !== undefined ? `${formatNumber(isExecutionRunning.value ? live.error_rate : s.error_rate)}%` : '-' },
        { name: '成功率', value: (isExecutionRunning.value ? live.success_rate : s.success_rate) !== undefined ? `${formatNumber(isExecutionRunning.value ? live.success_rate : s.success_rate)}%` : '-' },
        { name: '吞吐量', value: isExecutionRunning.value ? `${formatNumber(live.current_request_rate)} req/s` : s.throughput ? `${formatNumber(s.throughput)} req/s` : '-' }
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

// 分页的错误记录
const paginatedRecords = computed(() => {
  if (!errorAnalysis.value?.records) return []
  const start = (errorPage.value - 1) * errorPageSize.value
  return errorAnalysis.value.records.slice(start, start + errorPageSize.value)
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
  const map = {
    running: 'primary',
    success: 'success',
    failed: 'danger',
    stopped: 'warning'
  }
  return map[status] || 'info'
}

// 获取状态显示文本
const getStatusText = (status) => {
  const textMap = {
    running: '运行中',
    success: '成功',
    failed: '失败',
    stopped: '已停止'
  }
  return textMap[status] || status
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
  if (item.name === '吞吐量') classes.push('text-green')
  if (item.name.includes('成功')) classes.push('text-green')
  if (item.name.includes('错误样本')) classes.push('text-red')
  if (item.name === '错误率') {
    const errorRate = getErrorRateValue(summary.value.error_rate)
    classes.push(errorRate > 5 ? 'text-red' : 'text-green')
  }
  return classes
}

// 获取执行详情
const fetchExecutionDetail = async () => {
  if (detailRefreshing.value) return
  detailRefreshing.value = true
  loading.value = true
  try {
    const res = await executionApi.getDetail(executionId.value)
    execution.value = res.data || {}
  } catch (error) {
    console.error('获取执行详情失败:', error)
  } finally {
    loading.value = false
    detailRefreshing.value = false
  }
}

// 获取错误分析数据
const fetchErrors = async () => {
  if (!executionId.value) return
  if (errorLoading.value) return
  errorLoading.value = true
  try {
    const res = await executionApi.getErrors(executionId.value)
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
  if (!executionId.value) return
  if (liveRefreshing.value) return
  liveRefreshing.value = true
  try {
    const res = await executionApi.getLiveMetrics(executionId.value)
    liveMetrics.value = res.data || { points: [] }
  } catch (error) {
    console.error('获取实时指标失败:', error)
  } finally {
    liveRefreshing.value = false
  }
}

const maybeFetchErrors = async (force = false) => {
  if (!force && execution.value.status === 'running' && errorActiveTab.value !== 'records' && errorActiveTab.value !== 'types') {
    return
  }
  if (!force && execution.value.status === 'running' && errorActiveTab.value !== 'records' && errorActiveTab.value !== 'types') {
    return
  }
  if (!force && execution.value.status === 'running' && !errorAnalysisLoaded.value) {
    return
  }
  await fetchErrors()
}

const expandedChartConfig = computed(() => {
  const configs = {
    tps: {
      title: 'TPS 趋势',
      value: formatNumber(primaryMetricValue('tps')),
      unit: 'req/s',
      subline: chartSubline('tps'),
      field: 'tps',
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
    }
  }
  return configs[expandedChartKey.value] || null
})

const primaryMetricValue = (metricKey) => {
  const metrics = liveMetrics.value || {}
  switch (metricKey) {
    case 'tps':
      return isExecutionRunning.value ? metrics.current_tps : metrics.peak_tps
    case 'request_rate':
      return isExecutionRunning.value ? metrics.current_request_rate : metrics.peak_request_rate
    case 'response_time':
      return isExecutionRunning.value ? metrics.current_rt : metrics.avg_rt
    case 'concurrency':
      return isExecutionRunning.value ? metrics.current_concurrency : metrics.peak_concurrency
    default:
      return null
  }
}

const chartSubline = (metricKey) => {
  const metrics = liveMetrics.value || {}
  switch (metricKey) {
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
  if (logSnapshotLoading.value) return
  logSnapshotLoading.value = true
  try {
    const res = await fetch(`/api/executions/${executionId.value}/log?snapshot=1&tail=${MAX_LOG_LINES}`)
    const text = await res.text()
    setLogLines(text.split('\n').filter(line => line.trim() !== ''))
    nextTick(() => {
      scrollToBottom()
    })
  } catch (error) {
    console.error('获取日志失败:', error)
    setLogLines(['获取日志失败'])
  } finally {
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
  refreshTimer.value = setInterval(() => {
    if (execution.value.status === 'running') {
      fetchExecutionDetail()
      fetchLiveMetrics()
    }
  }, LIVE_REFRESH_INTERVAL)
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
  fetchExecutionDetail().then(() => {
    fetchLiveMetrics()
    fetchLog()
    if (execution.value.status !== 'running') {
      fetchErrors()
    }
    setupAutoRefresh()
    setupDurationTicker()
  })
})

watch(() => execution.value.status, (status, prevStatus) => {
  if (status && status !== 'running' && status !== prevStatus && !errorAnalysisLoaded.value) {
    fetchErrors()
  }
})

onBeforeUnmount(() => {
  stopLogStream()
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }
  if (durationTimer.value) {
    clearInterval(durationTimer.value)
    durationTimer.value = null
  }
})
</script>

<style scoped lang="scss">
.execution-detail-page {
  padding: 20px;
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
  .overview-primary-card {
    padding: 16px;
  }

  .stats-meta-row {
    grid-template-columns: 1fr;
  }

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
