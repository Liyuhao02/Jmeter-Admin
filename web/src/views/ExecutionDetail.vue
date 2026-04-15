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
            <div class="script-eyebrow-row">
              <span class="script-eyebrow">EXECUTION RUN</span>
              <span
                v-for="item in headerMetaItems"
                :key="item.label"
                class="script-meta-chip"
              >
                <span class="script-meta-chip-label">{{ item.label }}</span>
                <span class="script-meta-chip-value">{{ item.value }}</span>
              </span>
            </div>
            <div class="script-title-row">
              <h1 class="script-name">{{ execution.script_name || '执行详情' }}</h1>
              <el-tag
                :type="getStatusType(execution)"
                size="small"
                class="status-tag"
              >
                {{ getStatusText(execution) }}
              </el-tag>
            </div>
            <div class="script-subtitle">{{ overviewStatusNote }}</div>
          </div>
        </div>
        <div class="header-right">
          <div class="header-actions-row">
            <span class="execution-time" v-if="execution.created_at">
              <el-icon><Clock /></el-icon>
              {{ formatDateTime(execution.created_at) }}
            </span>
            <span class="detail-sync-chip" :class="`is-${liveSyncTone}`">
              <span class="detail-sync-dot"></span>
              {{ liveSyncLabel }}
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
          <div class="header-metric-strip">
            <div
              v-for="metric in headerMetricCards"
              :key="metric.key"
              class="header-metric-card"
              :class="`is-${metric.tone}`"
            >
              <span class="header-metric-label">{{ metric.label }}</span>
              <strong class="header-metric-value">{{ metric.value }}</strong>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="detail-layout">
      <aside class="detail-sidebar">
        <div class="detail-sidebar-card">
          <div class="detail-sidebar-eyebrow">SECTIONS</div>
          <div class="detail-sidebar-title">页面目录</div>
          <div class="detail-sidebar-desc">左侧直接跳转，不用再上下找模块。</div>
          <div class="detail-sidebar-list">
            <button
              v-for="item in detailNavItems"
              :key="`sidebar-${item.key}`"
              type="button"
              class="detail-sidebar-item"
              :class="{ active: activeDetailSection === item.key }"
              @click="scrollToDetailSection(item.key)"
            >
              <span class="detail-sidebar-kicker">{{ item.kicker }}</span>
              <span class="detail-sidebar-label">{{ item.label }}</span>
            </button>
          </div>
        </div>
      </aside>

      <div class="detail-main">
        <nav class="detail-section-nav" aria-label="执行详情分区导航">
          <button
            v-for="item in detailNavItems"
            :key="item.key"
            type="button"
            class="detail-nav-item"
            :class="{ active: activeDetailSection === item.key }"
            @click="scrollToDetailSection(item.key)"
          >
            <span class="detail-nav-kicker">{{ item.kicker }}</span>
            <span class="detail-nav-label">{{ item.label }}</span>
          </button>
        </nav>

        <!-- 执行概览 -->
        <div class="section-card" :ref="(el) => setDetailSectionRef('overview', el)">
      <div class="section-header">
        <div class="section-label">OVERVIEW</div>
        <div class="section-title">执行概览</div>
      </div>

      <div class="overview-stage-intro">
        <div>
          <div class="overview-stage-kicker">RUN SNAPSHOT</div>
          <div class="overview-stage-title">{{ overviewStageTitle }}</div>
          <div class="overview-stage-desc">{{ overviewStageDesc }}</div>
        </div>
        <span class="overview-stage-badge">{{ formatNumber(summaryCounts.total) }} 样本</span>
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

    <div
      v-if="diagnosticCards.length || diagnosticWarnings.length"
      class="section-card"
      :ref="(el) => setDetailSectionRef('diagnostics', el)"
    >
      <div class="section-header">
        <div class="section-label">DIAGNOSTICS</div>
        <div class="section-title">执行诊断</div>
      </div>
      <div class="diagnostic-stage">
        <div class="diagnostic-stage-hero">
          <div class="diagnostic-stage-kicker">DIAGNOSTIC FOCUS</div>
          <div class="diagnostic-stage-title">{{ diagnosticStageTitle }}</div>
          <div class="diagnostic-stage-desc">{{ diagnosticStageDesc }}</div>
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
    </div>

    <div
      v-if="distributedTopologyRows.length"
      class="section-card"
      :ref="(el) => setDetailSectionRef('topology', el)"
    >
      <div class="section-header">
        <div class="section-label">TOPOLOGY</div>
        <div class="section-title">分布式链路</div>
      </div>
      <div class="topology-stage-intro">
        <div>
          <div class="topology-stage-kicker">DISTRIBUTED PATH</div>
          <div class="topology-stage-title">Master、Slave 与回调链路</div>
          <div class="topology-stage-desc">
            把调度、回调和明细上传放在同一视角里看，优先确认分布式执行的关键链路是否完整。
          </div>
        </div>
        <span class="topology-stage-badge">{{ distributedTopologyRows.length }} 节点</span>
      </div>
      <div class="topology-meta-row">
        <div class="topology-meta-chip">
          <span class="topology-meta-label">Master 回调基地址</span>
          <code>{{ diagnostics.master_callback_base_url || '未配置' }}</code>
        </div>
        <div class="topology-meta-chip">
          <span class="topology-meta-label">HTTP 明细回传</span>
          <code>{{ diagnostics.save_http_details ? '已开启' : '未开启' }}</code>
        </div>
        <div v-if="diagnostics.master_detail_upload_url" class="topology-meta-chip">
          <span class="topology-meta-label">明细上传端点</span>
          <code>{{ diagnostics.master_detail_upload_url }}</code>
        </div>
      </div>
      <div class="topology-grid">
        <div v-for="node in distributedTopologyRows" :key="node.key" class="topology-card" :class="`is-${node.tone}`">
          <div class="topology-card-header">
            <div>
              <div class="topology-card-role">{{ node.role }}</div>
              <div class="topology-card-name">{{ node.name }}</div>
            </div>
            <span class="topology-card-status" :class="`is-${node.tone}`">{{ node.status }}</span>
          </div>
          <div class="topology-card-note">{{ node.note }}</div>
          <div class="topology-card-foot">{{ node.foot }}</div>
        </div>
      </div>
    </div>

    <div
      v-if="executionConclusion"
      class="section-card"
      :ref="(el) => setDetailSectionRef('conclusion', el)"
    >
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

    <div
      v-if="timelineStages.length"
      class="section-card"
      :ref="(el) => setDetailSectionRef('timeline', el)"
    >
      <div class="section-header">
        <div class="section-label">TIMELINE</div>
        <div class="section-title">执行链路</div>
      </div>
      <div class="timeline-stage-intro">
        <div>
          <div class="timeline-stage-kicker">EXECUTION PATH</div>
          <div class="timeline-stage-title">执行链路节奏</div>
          <div class="timeline-stage-desc">
            从创建、启动、回传到结束逐段复盘，快速判断卡点究竟落在哪一个阶段。
          </div>
        </div>
        <span class="timeline-stage-badge">{{ timelineStages.length }} 阶段</span>
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

    <div
      v-if="samplerStats.length"
      class="section-card"
      :ref="(el) => setDetailSectionRef('samplers', el)"
    >
      <div class="section-header">
        <div class="section-label">SAMPLERS</div>
        <div class="section-title">接口维度分析</div>
        <div class="section-desc">这里适合看热点接口、慢接口和高错误接口，和上面的趋势区配合着一起判断更快。</div>
      </div>
      <div class="sampler-stage-intro">
        <div>
          <div class="sampler-stage-kicker">API BREAKDOWN</div>
          <div class="sampler-stage-title">接口维度结果</div>
          <div class="sampler-stage-desc">
            先看热点接口和慢接口，再结合下方明细表定位吞吐瓶颈、异常接口和高延迟请求。
          </div>
        </div>
        <span class="sampler-stage-badge">{{ samplerStats.length }} 接口</span>
      </div>
      <div class="sampler-overview-grid">
        <div v-for="card in samplerOverviewCards" :key="card.key" class="sampler-overview-card">
          <div class="sampler-overview-label">{{ card.label }}</div>
          <div class="sampler-overview-name">{{ card.name }}</div>
          <div class="sampler-overview-value">{{ card.value }}</div>
          <div class="sampler-overview-caption">{{ card.caption }}</div>
        </div>
      </div>
      <div class="sampler-table-shell">
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
    </div>

    <!-- 节点监控面板 - 仅运行中显示 -->
    <div
      v-if="execution?.status === 'running' && nodeMetrics.length > 0"
      class="section-card node-metrics-panel"
      :ref="(el) => setDetailSectionRef('nodes', el)"
    >
      <div class="section-header with-tools">
        <div>
          <div class="section-label">NODE MONITORING</div>
          <div class="section-title">节点实时监控</div>
          <div class="section-desc">优先看节点负载、内存和连接数，先排除机器资源瓶颈，再继续判断业务层面的异常波动。</div>
        </div>
        <div class="section-header-tools">
          <span class="live-section-meta">{{ nodeMetrics.length }} 个节点</span>
          <span class="live-section-meta">{{ liveMetricsFreshnessText }}</span>
        </div>
      </div>
      <div class="metrics-grid node-metrics-grid">
        <div v-for="node in nodeMetrics" :key="node.id" class="node-card">
          <div class="node-header">
            <div class="node-header-main">
              <div class="node-name-row">
                <span class="node-name">{{ node.name }}</span>
                <span class="node-role">{{ getNodeRoleText(node) }}</span>
              </div>
              <div class="node-host">
                {{ node.host || '未知地址' }}<template v-if="node.port">:{{ node.port }}</template>
              </div>
            </div>
            <span class="node-status" :class="node.online ? 'online' : 'offline'">
              {{ node.online ? '在线' : '离线' }}
            </span>
          </div>
          <div v-if="node.stats" class="node-stats detailed">
            <div class="node-resource-card">
              <div class="node-resource-head">
                <span>CPU</span>
                <strong>{{ formatPercent(node.stats.cpu?.percent) }}</strong>
              </div>
              <el-progress
                :percentage="Math.round(node.stats.cpu?.percent || 0)"
                :stroke-width="6"
                :show-text="false"
                :color="getResourceColor(node.stats.cpu?.percent || 0)"
              />
              <div class="node-resource-meta">{{ node.stats.cpu?.count || '--' }} 核</div>
            </div>
            <div class="node-resource-card">
              <div class="node-resource-head">
                <span>内存</span>
                <strong>{{ formatPercent(node.stats.memory?.percent) }}</strong>
              </div>
              <el-progress
                :percentage="Math.round(node.stats.memory?.percent || 0)"
                :stroke-width="6"
                :show-text="false"
                :color="getResourceColor(node.stats.memory?.percent || 0)"
              />
              <div class="node-resource-meta">
                {{ formatMB(node.stats.memory?.used ?? node.stats.memory?.used_mb) }} /
                {{ formatMB(node.stats.memory?.total ?? node.stats.memory?.total_mb) }}
              </div>
            </div>
            <div class="node-resource-card">
              <div class="node-resource-head">
                <span>磁盘</span>
                <strong>{{ formatPercent(node.stats.disk?.percent) }}</strong>
              </div>
              <el-progress
                :percentage="Math.round(node.stats.disk?.percent || 0)"
                :stroke-width="6"
                :show-text="false"
                :color="getResourceColor(node.stats.disk?.percent || 0)"
              />
              <div class="node-resource-meta">
                {{ formatMB(node.stats.disk?.used ?? node.stats.disk?.used_mb) }} /
                {{ formatMB(node.stats.disk?.total ?? node.stats.disk?.total_mb) }}
              </div>
            </div>
            <div class="node-meta-grid">
              <div class="node-meta-item">
                <span>连接数</span>
                <strong>{{ node.stats.network?.connections || 0 }}</strong>
              </div>
              <div class="node-meta-item">
                <span>Agent</span>
                <strong>{{ getNodeAgentText(node) }}</strong>
              </div>
            </div>
          </div>
          <div v-else class="node-offline">
            数据不可用
          </div>
        </div>
      </div>
    </div>

    <div class="section-card" :ref="(el) => setDetailSectionRef('metrics', el)">
      <div class="section-header with-tools">
        <div>
          <div class="section-label">LIVE METRICS</div>
          <div class="section-title">实时趋势</div>
          <div class="section-desc">先看摘要，再按吞吐、延迟、质量三条线下钻，每组图只负责回答一个问题。</div>
        </div>
        <div class="section-header-tools">
          <span class="live-section-meta">{{ liveMetricsFreshnessText }}</span>
          <el-button
            text
            type="primary"
            class="live-refresh-btn"
            :loading="liveRefreshing || detailRefreshing"
            @click="handleManualRefreshMetrics"
          >
            <el-icon><Refresh /></el-icon>
            立即刷新
          </el-button>
        </div>
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
      <div class="live-ops-strip">
        <span class="live-ops-pill">悬浮查看单个采样时间点</span>
        <span class="live-ops-pill">点击放大查看完整趋势</span>
        <span class="live-ops-pill">{{ (liveMetrics.points || []).length }} 个采样点</span>
      </div>
      <div class="live-chart-stage">
        <section class="live-chart-group is-throughput">
          <div class="live-chart-group-head">
            <div>
              <div class="live-chart-kicker">THROUGHPUT</div>
              <h3>吞吐与请求速率</h3>
              <p>先确认事务吞吐是否稳定，再对照请求进入速率判断压测节奏有没有跑偏。</p>
            </div>
            <div class="live-chart-group-meta">
              <span class="live-chart-group-chip">TPS / 请求速率</span>
            </div>
          </div>
          <div class="live-chart-cluster live-chart-cluster--two">
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
          </div>
        </section>

        <section class="live-chart-group is-latency">
          <div class="live-chart-group-head">
            <div>
              <div class="live-chart-kicker">LATENCY</div>
              <h3>延迟与并发</h3>
              <p>把平均值、瞬时响应、分位延迟和并发放在一起看，能更快判断是否已经进入抖动区。</p>
            </div>
            <div class="live-chart-group-meta">
              <span class="live-chart-group-chip">平均 / 瞬时 / 分位 / 并发</span>
            </div>
          </div>
          <div class="live-chart-cluster live-chart-cluster--two">
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
          </div>
        </section>

        <section class="live-chart-group is-quality">
          <div class="live-chart-group-head">
            <div>
              <div class="live-chart-kicker">QUALITY</div>
              <h3>成功率与错误走势</h3>
              <p>最后看成功率和错误数，确认当前问题是持续性失败、瞬时波峰，还是节点侧的局部异常。</p>
            </div>
            <div class="live-chart-group-meta">
              <span class="live-chart-group-chip">成功率 / 错误数</span>
            </div>
          </div>
          <div class="live-chart-cluster live-chart-cluster--two">
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
          </div>
        </section>
      </div>
    </div>

    <!-- 详细统计 -->
    <div class="section-card" :ref="(el) => setDetailSectionRef('statistics', el)">
      <div class="section-header">
        <div class="section-label">STATISTICS</div>
        <div class="section-title">详细统计</div>
        <div class="section-desc">把分位延迟、样本信息和流量维度压成紧凑分组，方便快速横向对照。</div>
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
    <div
      class="section-card error-analysis-section"
      v-loading="errorLoading && !errorAnalysis"
      :ref="(el) => setDetailSectionRef('errors', el)"
    >
      <div class="section-header with-tools">
        <div>
          <div class="section-label">ERRORS</div>
          <div class="section-title is-danger-title">
            <el-icon><WarningFilled /></el-icon>
            错误分析
          </div>
          <div class="section-desc">按错误类型、来源节点和失败接口聚合，先定位影响面最大的失败，再进入单条错误详情。</div>
        </div>
        <div class="error-actions">
          <el-button size="small" @click="fetchErrors" :loading="errorLoading">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
          <el-button size="small" @click="downloadErrorReport" :disabled="!errorAnalysis">
            <el-icon><Document /></el-icon> 导出错误报告
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
        <div class="error-overview-shell">
          <div class="error-overview-hero">
            <div class="error-overview-kicker">ERROR SNAPSHOT</div>
            <div class="error-overview-title">{{ errorHeroTitle }}</div>
            <div class="error-overview-desc">{{ errorHeroDesc }}</div>

            <div class="error-context-strip">
              <div
                v-for="item in errorContextCards"
                :key="item.label"
                class="error-context-card"
                :class="`is-${item.tone || 'neutral'}`"
              >
                <span class="error-context-label">{{ item.label }}</span>
                <strong class="error-context-value">{{ item.value }}</strong>
                <span v-if="item.meta" class="error-context-meta">{{ item.meta }}</span>
              </div>
            </div>
          </div>

          <div class="error-overview-side">
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
          </div>
        </div>

        <div
          class="error-insight-shell"
          v-if="errorCategoryClusters.length || errorSourceClusters.length || errorApiClusters.length || errorReportHighlights.length"
        >
          <div class="error-insight-shell-head">
            <div>
              <div class="error-chart-title">INSIGHTS</div>
              <div class="error-insight-title">聚类洞察</div>
              <div class="error-insight-desc">按错误类别、节点来源和热点接口收束问题面，优先排查影响范围最大的主因。</div>
            </div>
          </div>
          <div class="error-insight-grid">
            <div class="error-insight-card" v-if="errorCategoryClusters.length">
              <div class="error-chart-title">错误分类聚类</div>
              <div class="error-cluster-list">
                <div v-for="item in errorCategoryClusters" :key="item.key" class="error-cluster-item">
                  <div class="error-cluster-main">
                    <span class="error-cluster-label">{{ item.label }}</span>
                    <span class="error-cluster-count">{{ item.count }} 条</span>
                  </div>
                  <div class="error-cluster-meta">
                    <span>{{ item.percentage.toFixed(2) }}%</span>
                    <span v-if="item.top_labels?.length">接口：{{ item.top_labels.join('、') }}</span>
                  </div>
                </div>
              </div>
            </div>

            <div class="error-insight-card" v-if="errorSourceClusters.length">
              <div class="error-chart-title">来源节点分布</div>
              <div class="error-cluster-list">
                <div v-for="item in errorSourceClusters" :key="item.key" class="error-cluster-item">
                  <div class="error-cluster-main">
                    <span class="error-cluster-label">{{ item.label }}</span>
                    <span class="error-cluster-count">{{ item.count }} 条</span>
                  </div>
                  <div class="error-cluster-meta">
                    <span>{{ item.percentage.toFixed(2) }}%</span>
                    <span v-if="item.top_labels?.length">热点：{{ item.top_labels.join('、') }}</span>
                  </div>
                </div>
              </div>
            </div>

            <div class="error-insight-card" v-if="errorApiClusters.length">
              <div class="error-chart-title">失败接口 Top</div>
              <div class="error-cluster-list">
                <div v-for="item in errorApiClusters" :key="item.key" class="error-cluster-item">
                  <div class="error-cluster-main">
                    <span class="error-cluster-label">{{ item.label || item.url || '未命名请求' }}</span>
                    <span class="error-cluster-count">{{ item.count }} 条</span>
                  </div>
                  <div class="error-cluster-meta">
                    <span>{{ item.percentage.toFixed(2) }}%</span>
                    <span class="truncate-text" :title="item.url">{{ item.url || 'URL 未记录' }}</span>
                  </div>
                </div>
              </div>
            </div>

            <div class="error-insight-card" v-if="errorReportHighlights.length">
              <div class="error-chart-title">错误复盘结论</div>
              <div class="error-report-list">
                <div v-for="item in errorReportHighlights" :key="item" class="error-report-item">
                  {{ item }}
                </div>
              </div>
            </div>
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
            <div class="error-workbench">
              <div class="error-workbench-toolbar">
                <div>
                  <div class="error-workbench-title">错误类型分布</div>
                  <div class="error-workbench-desc">按请求名称、响应码与响应信息聚合同类错误，适合先看主故障面。</div>
                </div>
                <span class="error-workbench-meta">{{ errorAnalysis.error_types?.length || 0 }} 类</span>
              </div>
              <div class="error-table-shell">
                <el-table
                  class="error-types-table"
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
              </div>
            </div>
          </el-tab-pane>
          <el-tab-pane label="错误记录明细" name="records">
            <div class="error-workbench">
              <div class="error-workbench-toolbar">
                <div>
                  <div class="error-workbench-title">错误记录明细</div>
                  <div class="error-workbench-desc">适合直接按响应码、请求名和具体失败原因下钻，定位到单条错误。</div>
                </div>
                <span class="error-workbench-meta">{{ filteredErrorRecordsRaw.length }} 条</span>
              </div>

              <div class="error-note-stack">
                <div v-if="!errorRecordsLoaded && errorAnalysis.total_errors > 0" class="error-note-card is-info">
                  当前正在展示实时错误摘要，打开明细表时会按需加载完整错误记录。
                </div>
                <div v-if="errorAnalysis.truncated" class="error-note-card is-warning">
                  错误记录超过 10000 条，当前仅展示前 10000 条。
                </div>
                <div v-if="errorAnalysis.detail_upload_warning" class="error-note-card is-warning">
                  <div>{{ errorAnalysis.detail_upload_warning }}</div>
                  <div v-if="errorAnalysis.missing_detail_sources?.length" class="error-note-meta">
                    未回传节点：{{ errorAnalysis.missing_detail_sources.join('、') }}
                  </div>
                </div>
              </div>

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

              <div class="error-table-shell">
                <el-table class="error-records-table" :data="formatErrorRecords" style="width: 100%" v-loading="errorLoading">
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
              </div>
              <el-pagination
                v-model:current-page="errorPage"
                v-model:page-size="errorPageSize"
                :page-sizes="[50, 100, 200]"
                :total="filteredErrorRecordsRaw.length"
                layout="total, sizes, prev, pager, next"
                class="error-pagination"
              />
            </div>
          </el-tab-pane>
        </el-tabs>
      </template>
    </div>

    <!-- 测试报告 -->
    <div class="section-card" :ref="(el) => setDetailSectionRef('report', el)">
      <div class="section-header with-tools">
        <div>
          <div class="section-label">REPORT</div>
          <div class="section-title">测试报告</div>
          <div class="section-desc">{{ reportHeaderHint }}</div>
        </div>
        <div v-if="execution.status === 'success'" class="section-header-tools report-actions">
          <el-button size="small" @click="openReportInNewWindow">
            新窗口打开
          </el-button>
          <el-button size="small" type="primary" @click="reportFullscreen = true">
            <el-icon><FullScreen /></el-icon> 全屏查看
          </el-button>
        </div>
      </div>
      <div class="report-wrapper">
        <div v-if="execution.status === 'success'" class="report-browser-frame">
          <div class="report-browser-bar">
            <div class="report-browser-copy">
              <span class="report-browser-label">HTML Report</span>
              <span class="report-browser-tip">内嵌报告适合快速复盘，也可以切到新窗口做细节排查。</span>
            </div>
            <code class="report-browser-url">{{ reportUrl }}</code>
          </div>
          <div class="report-container">
            <iframe 
              :src="reportUrl" 
              class="report-iframe"
              frameborder="0"
            ></iframe>
          </div>
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
        <div class="section-card terminal-section" :ref="(el) => setDetailSectionRef('logs', el)">
      <div class="section-header with-tools">
        <div>
          <div class="section-label">LOGS</div>
          <div class="section-title">执行日志</div>
          <div class="section-desc">运行中可盯实时流，结束后可搜索、复制和导出完整日志。</div>
        </div>
        <div class="section-header-tools terminal-section-meta">
          <span class="detail-sync-chip" :class="`is-${terminalSyncTone}`">
            <span class="detail-sync-dot"></span>
            {{ terminalModeLabel }}
          </span>
          <span class="terminal-meta-pill">{{ logLines.length }} 行</span>
          <span v-if="logSearch" class="terminal-meta-pill">{{ matchCount }} 匹配</span>
        </div>
      </div>
      <div class="terminal-window">
        <!-- 终端标题栏 -->
        <div class="terminal-header">
          <div class="terminal-title-group">
            <div class="terminal-title">
              <el-icon><Monitor /></el-icon>
              <span>Runtime Console</span>
            </div>
            <div class="terminal-subtitle">{{ terminalModeLabel }}</div>
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
          </div>
          <div class="terminal-actions">
            <button
              v-if="execution.status === 'running'"
              class="terminal-btn"
              :class="{ 'is-active': isStreaming }"
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
              {{ isStreaming ? (sseConnected ? '实时流已连接' : '正在连接') : '静态查看模式' }}
            </span>
          </div>
          <div class="log-stats">
            <span>{{ logLines.length }} 行</span>
            <span v-if="logTrimmed">仅保留最近 {{ MAX_LOG_LINES }} 行</span>
          </div>
        </div>
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
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
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
import { formatDateTimeInShanghai, formatRelativeTimeInShanghai, parseServerDateTime } from '@/utils/datetime'
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
const detailStreamConnected = ref(false)
const detailEventSource = ref(null)
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
const nodeMetricsRefreshing = ref(false)
const lastDetailStreamAt = ref(0)
const errorAnalysisLoaded = ref(false)
const errorRecordsLoaded = ref(false)
const nowTick = ref(Date.now())
const expandedChartKey = ref('')
const expandedChartVisible = ref(false)
const pageVisible = ref(typeof document === 'undefined' ? true : document.visibilityState === 'visible')
let leavingPage = false
const activeDetailSection = ref('overview')
const detailSectionRefs = ref({})
let detailSectionRaf = 0

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

const isBenignRequestCancel = (error) => {
  const message = String(error?.message || '').toLowerCase()
  return (
    error?.code === 'ERR_CANCELED' ||
    error?.name === 'CanceledError' ||
    message === 'canceled' ||
    message.includes('duplicate request')
  )
}

const isIgnorableExecutionRequestError = (error) => {
  if (isBenignRequestCancel(error)) return true
  const message = String(error?.response?.data?.message || error?.message || '')
  return error?.response?.data?.code === -1 && message.includes('无效的执行')
}

const showErrorDetail = (record) => {
  currentErrorRecord.value = record
  errorDetailTab.value = 'request'
  errorDetailVisible.value = true
}

// 日志搜索和高亮
const logSearch = ref('')
const reconnectAttempts = ref(0)
const maxReconnectAttempts = 5
const detailReconnectAttempts = ref(0)
const maxDetailReconnectAttempts = 6
const DETAIL_STREAM_STALE_MS = 1400
const DETAIL_STREAM_RECONNECT_MS = 4200
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

const reportHeaderHint = computed(() => {
  if (execution.value.status === 'success') {
    return '报告已生成，可直接内嵌查看，也可以切到新窗口做更细的复盘。'
  }
  if (execution.value.status === 'running') {
    return '报告会在执行结束并完成汇总后生成，当前可以先盯实时指标和日志。'
  }
  return '当前执行尚未产出可浏览的 HTML 报告，可先查看错误分析和原始日志。'
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

const liveMetricsLastTimestamp = computed(() => {
  if (lastDetailStreamAt.value) return lastDetailStreamAt.value
  const points = liveMetrics.value?.points || []
  const lastPoint = points.length ? points[points.length - 1] : null
  return lastPoint?.timestamp || lastPoint?.time_bucket || null
})

const shouldPollLive = computed(() => {
  if (execution.value.status === 'running') return true
  if (!execution.value.end_time && execution.value.status !== 'failed' && execution.value.status !== 'stopped') {
    return true
  }
  if (liveMetricsLastTimestamp.value) {
    return Date.now() - new Date(liveMetricsLastTimestamp.value).getTime() < 15000
  }
  return false
})

const liveMetricsFreshnessText = computed(() => {
  if (!shouldPollLive.value) return '结果已稳定，可查看最终统计'
  if (!liveMetricsLastTimestamp.value) return '等待实时采样数据'
  return `最近刷新 ${formatRelativeTimeInShanghai(liveMetricsLastTimestamp.value)}`
})

const splitCSVEnabled = computed(() => {
  return Boolean(diagnostics.value?.split_csv || execution.value?.split_csv)
})

const detailNavItems = computed(() => {
  const items = [
    { key: 'overview', label: '概览', kicker: '总览' },
  ]

  if (diagnosticCards.value.length || diagnosticWarnings.value.length) {
    items.push({ key: 'diagnostics', label: '诊断', kicker: '体征' })
  }
  if (distributedTopologyRows.value.length) {
    items.push({ key: 'topology', label: '链路', kicker: '拓扑' })
  }
  if (executionConclusion.value) {
    items.push({ key: 'conclusion', label: '结论', kicker: '判断' })
  }
  if (timelineStages.value.length) {
    items.push({ key: 'timeline', label: '时序', kicker: '路径' })
  }
  if (samplerStats.value.length) {
    items.push({ key: 'samplers', label: '接口', kicker: 'API' })
  }
  if (execution.value?.status === 'running' && nodeMetrics.value.length > 0) {
    items.push({ key: 'nodes', label: '节点', kicker: '资源' })
  }

  items.push(
    { key: 'metrics', label: '趋势', kicker: 'Live' },
    { key: 'statistics', label: '统计', kicker: 'Stats' },
    { key: 'errors', label: '错误', kicker: 'Issue' },
    { key: 'report', label: '报告', kicker: 'HTML' },
    { key: 'logs', label: '日志', kicker: 'Log' },
  )

  return items
})

const liveSyncTone = computed(() => {
  if (!isExecutionRunning.value) return 'stable'
  if (detailStreamConnected.value) return 'stream'
  if (liveRefreshing.value || detailRefreshing.value) return 'polling'
  return 'lag'
})

const liveSyncLabel = computed(() => {
  if (!isExecutionRunning.value) return '执行已结束'
  if (detailStreamConnected.value) return '实时流已连接'
  if (liveRefreshing.value || detailRefreshing.value) return '轮询刷新中'
  return '等待下一次刷新'
})

const setDetailSectionRef = (key, el) => {
  if (el) {
    detailSectionRefs.value[key] = el
    return
  }
  delete detailSectionRefs.value[key]
}

const updateActiveDetailSection = () => {
  if (typeof window === 'undefined') return
  const items = detailNavItems.value
  if (!items.length) return

  let current = items[0].key
  for (const item of items) {
    const target = detailSectionRefs.value[item.key]
    if (!target) continue
    if (target.getBoundingClientRect().top <= 150) {
      current = item.key
    } else {
      break
    }
  }
  activeDetailSection.value = current
}

const handleDetailSectionScroll = () => {
  if (typeof window === 'undefined' || detailSectionRaf) return
  detailSectionRaf = window.requestAnimationFrame(() => {
    detailSectionRaf = 0
    updateActiveDetailSection()
  })
}

const scrollToDetailSection = (key) => {
  if (typeof window === 'undefined') return
  const target = detailSectionRefs.value[key]
  if (!target) return
  activeDetailSection.value = key
  const stickyOffset = 90
  const top = target.getBoundingClientRect().top + window.scrollY - stickyOffset
  window.scrollTo({ top: Math.max(top, 0), behavior: 'smooth' })
}

const terminalSyncTone = computed(() => {
  if (execution.value.status === 'running') {
    if (isStreaming.value && sseConnected.value) return 'stream'
    if (isStreaming.value) return 'polling'
    return 'lag'
  }
  return 'stable'
})

const terminalModeLabel = computed(() => {
  if (execution.value.status === 'running') {
    if (isStreaming.value && sseConnected.value) return '实时日志流已连接'
    if (isStreaming.value) return '正在建立实时日志流'
    return '日志轮询模式'
  }
  return '静态日志复盘'
})

const executionModeLabel = computed(() => {
  const mode = diagnostics.value?.mode || 'local'
  if (mode === 'distributed_with_master') return 'Master + Slave'
  if (mode === 'distributed') return '分布式执行'
  return '本地执行'
})

const headerMetaItems = computed(() => {
  const items = [
    { label: 'ID', value: execution.value.id ? `#${execution.value.id}` : executionNumericId.value !== null ? `#${executionNumericId.value}` : '-' },
    { label: '模式', value: executionModeLabel.value }
  ]

  if (execution.value.remarks) {
    items.push({
      label: '备注',
      value: execution.value.remarks.length > 24 ? `${execution.value.remarks.slice(0, 24)}...` : execution.value.remarks
    })
  }

  if (diagnostics.value?.slave_count || diagnostics.value?.mode === 'distributed' || diagnostics.value?.mode === 'distributed_with_master') {
    const slaveCount = Number(diagnostics.value?.slave_count || 0)
    const masterFlag = diagnostics.value?.include_master ? ' + Master' : ''
    items.push({
      label: '节点',
      value: slaveCount > 0 ? `${slaveCount}${masterFlag}` : `Master${masterFlag}`
    })
  }

  return items.slice(0, 4)
})

const headerMetricCards = computed(() => {
  const throughputValue = isExecutionRunning.value
    ? liveMetrics.value.current_primary_throughput
    : summaryPrimaryThroughputValue.value
  const avgRtValue = isExecutionRunning.value
    ? liveMetrics.value.avg_rt
    : summary.value.avg_response_time
  const errorRateValue = isExecutionRunning.value
    ? liveMetrics.value.error_rate
    : summary.value.error_rate

  return [
    {
      key: 'throughput',
      label: isExecutionRunning.value ? `当前${liveMetrics.value.primary_throughput_label || summaryPrimaryThroughputLabel.value}` : summaryPrimaryThroughputLabel.value,
      value: throughputValue !== null && throughputValue !== undefined
        ? `${formatNumber(throughputValue)} ${isExecutionRunning.value ? livePrimaryThroughputUnit.value : summaryPrimaryThroughputUnit.value}`
        : '-',
      tone: 'blue'
    },
    {
      key: 'avg_rt',
      label: '平均 RT',
      value: avgRtValue !== null && avgRtValue !== undefined ? `${formatNumber(avgRtValue)} ms` : '-',
      tone: 'purple'
    },
    {
      key: 'error_rate',
      label: '错误率',
      value: errorRateValue !== null && errorRateValue !== undefined ? `${formatNumber(errorRateValue)}%` : '-',
      tone: getErrorRateValue(errorRateValue) > 0 ? 'red' : 'green'
    },
    {
      key: 'duration',
      label: '执行时长',
      value: formatDuration(displayDurationSeconds.value),
      tone: 'amber'
    }
  ]
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

const overviewStageTitle = computed(() => {
  if (execution.value.status === 'running') return '执行正在进行，关键指标会持续刷新'
  if (execution.value.status === 'failed') return '执行已结束，但当前结果需要重点复盘'
  return '执行已完成，可以按概览与诊断顺序快速复盘'
})

const overviewStageDesc = computed(() => {
  const counts = summaryCounts.value
  return `当前累计 ${formatNumber(counts.total)} 样本，成功 ${formatNumber(counts.success)} / 错误 ${formatNumber(counts.errors)}，建议先看主指标，再看右侧细项和下面的诊断结果。`
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
      value: splitCSVEnabled.value ? '已启用 CSV 分片' : '按原脚本执行',
      caption: dependencyCaption,
      color: diag.warnings?.length ? 'warning' : 'blue'
    }
  ]

  if (diag.mode && diag.mode !== 'local') {
    cards.push({
      key: 'topology',
      label: '拓扑说明',
      value: diag.include_master ? 'Master 参与施压' : 'Master 仅调度',
      caption: `Slave ${diag.slave_count || 0} 台 / CSV分片 ${splitCSVEnabled.value ? '开启' : '关闭'}`,
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

  if (diag.master_callback_base_url) {
    cards.push({
      key: 'callback',
      label: '回调链路',
      value: '已配置',
      caption: diag.master_callback_base_url,
      color: 'blue'
    })
  }

  return cards
})

const diagnosticWarnings = computed(() => {
  return diagnostics.value?.warnings || []
})

const diagnosticStageTitle = computed(() => {
  if (diagnosticWarnings.value.length) return '执行链路里有需要优先确认的风险点'
  return '当前执行链路清晰，可以把注意力放在结果与趋势上'
})

const diagnosticStageDesc = computed(() => {
  const cardCount = diagnosticCards.value.length
  const warningCount = diagnosticWarnings.value.length
  if (warningCount) {
    return `已生成 ${cardCount} 项诊断卡，并检测到 ${warningCount} 条提醒。建议优先确认回调链路、明细回传和依赖一致性。`
  }
  return `已生成 ${cardCount} 项诊断卡，当前没有额外警告。你可以继续查看分布式链路、实时趋势和错误明细。`
})

const normalizeSourceKey = (value) => String(value || '').trim().toLowerCase()

const matchesDiagnosticSource = (expected, received) => {
  const exp = normalizeSourceKey(expected)
  const got = normalizeSourceKey(received)
  if (!exp || !got) return false
  if (exp === got) return true
  const expHost = exp.split('(')[0].trim()
  const gotHost = got.split('(')[0].trim()
  return exp.includes(gotHost) || got.includes(expHost)
}

const distributedTopologyRows = computed(() => {
  const diag = diagnostics.value || {}
  if (!diag.mode || diag.mode === 'local') return []

  const received = Array.isArray(diag.received_detail_sources) ? diag.received_detail_sources : []
  const missing = Array.isArray(diag.missing_detail_sources) ? diag.missing_detail_sources : []
  const rows = []

  const buildNodeStatus = (expectedSource) => {
    if (!diag.save_http_details) {
      return {
        status: '未开启',
        note: '当前任务未保存失败请求明细，只保留结果统计。',
        tone: 'blue'
      }
    }
    const hasReceived = received.some((item) => matchesDiagnosticSource(expectedSource, item))
    const isMissing = missing.some((item) => matchesDiagnosticSource(expectedSource, item))
    if (hasReceived && !isMissing) {
      return {
        status: '已回传',
        note: '该节点的 HTTP 失败明细已经回到 Master，可在错误分析页直接查看。',
        tone: 'green'
      }
    }
    if (hasReceived && isMissing) {
      return {
        status: '部分回传',
        note: '该节点已经回传部分明细，但链路仍不完整，建议结合日志继续核对。',
        tone: 'warning'
      }
    }
    if (execution.value.status === 'running') {
      return {
        status: '等待中',
        note: '执行仍在进行，等待该节点继续回传运行结果与失败明细。',
        tone: 'blue'
      }
    }
    return {
      status: '缺失',
      note: '执行已结束，但没有收到该节点的 HTTP 明细，需重点核查网络可达性或 Agent 回调日志。',
      tone: 'warning'
    }
  }

  const masterExpectedSource = 'master-local'
  if (diag.include_master || diag.master_callback_base_url) {
    const masterState = buildNodeStatus(masterExpectedSource)
    rows.push({
      key: 'master-local',
      role: diag.include_master ? 'Master + 施压' : 'Master 调度',
      name: 'master-local',
      status: masterState.status,
      note: masterState.note,
      foot: diag.master_callback_base_url || '未显式配置回调地址',
      tone: masterState.tone
    })
  }

  ;(diag.slave_hosts || []).forEach((host, index) => {
    const slaveState = buildNodeStatus(host)
    rows.push({
      key: `slave-${index}-${host}`,
      role: 'Slave 节点',
      name: host,
      status: slaveState.status,
      note: slaveState.note,
      foot: splitCSVEnabled.value ? 'CSV 数据分片已开启' : '按原脚本读取 CSV / 文件依赖',
      tone: slaveState.tone
    })
  })

  return rows
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
      `已生成 ${diag.runtime_scripts.length} 份运行时脚本，CSV分片 ${splitCSVEnabled.value ? '开启' : '关闭'}。`,
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

const errorContextCards = computed(() => {
  if (!errorAnalysis.value?.total_errors) return []
  return [
    {
      label: '错误样本',
      value: errorAnalysis.value.total_errors.toLocaleString(),
      meta: `覆盖 ${errorAnalysis.value.error_types?.length || 0} 类错误`,
      tone: 'danger'
    },
    {
      label: '错误率',
      value: `${Number(errorAnalysis.value.error_rate || 0).toFixed(2)}%`,
      meta: errorAnalysis.value.total_errors === summaryCounts.value.total ? '当前全部样本均失败' : '按当前已采样样本计算',
      tone: Number(errorAnalysis.value.error_rate || 0) >= 50 ? 'danger' : 'warning'
    },
    {
      label: 'HTTP 明细',
      value: errorAnalysis.value.detail_fields_available ? '字段齐全' : '字段缺失',
      meta: errorAnalysis.value.detail_fields_available ? '可直接查看请求/响应详情' : '建议下次执行时开启明细保存',
      tone: errorAnalysis.value.detail_fields_available ? 'success' : 'warning'
    },
    {
      label: '记录状态',
      value: errorAnalysis.value.truncated ? '已截断' : '完整',
      meta: errorAnalysis.value.truncated ? '当前仅展示前 10000 条错误记录' : '可直接进入明细表排查',
      tone: errorAnalysis.value.truncated ? 'warning' : 'neutral'
    }
  ]
})

const errorHeroTitle = computed(() => {
  if (!errorAnalysis.value?.total_errors) return '当前没有错误样本'
  const rate = Number(errorAnalysis.value.error_rate || 0)
  if (rate >= 90) return '当前执行几乎全部失败，建议先处理主错误类型'
  if (rate >= 50) return '当前执行失败占比过半，建议优先排查热点接口'
  return '错误已出现但还相对集中，可以先从热点接口和来源节点入手'
})

const errorHeroDesc = computed(() => {
  if (!errorAnalysis.value?.total_errors) return '错误聚类、热点接口和来源节点会在这里自动收束。'
  const sourceCount = errorAnalysis.value.source_distribution?.length || 0
  const apiCount = errorAnalysis.value.api_distribution?.length || 0
  return `共记录 ${errorAnalysis.value.total_errors.toLocaleString()} 条错误，当前聚合出 ${sourceCount} 个来源节点和 ${apiCount} 个热点接口，可直接沿着聚类结果往下钻取。`
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

const errorCategoryClusters = computed(() => {
  return (errorAnalysis.value?.category_distribution || []).slice(0, 6)
})

const errorSourceClusters = computed(() => {
  return (errorAnalysis.value?.source_distribution || []).slice(0, 6)
})

const errorApiClusters = computed(() => {
  return (errorAnalysis.value?.api_distribution || []).slice(0, 8)
})

const errorReportHighlights = computed(() => {
  return errorAnalysis.value?.report_highlights || []
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
  if (leavingPage || !hasValidExecutionId.value || !shouldPollLive.value) return
  if (nodeMetricsRefreshing.value) return
  nodeMetricsRefreshing.value = true
  try {
    const res = await executionApi.getNodeMetrics(executionNumericId.value, { silent: true })
    nodeMetrics.value = normalizeNodeMetrics(res.data?.nodes || [])
  } catch (e) {
    if (isIgnorableExecutionRequestError(e)) return
    console.error('获取节点监控失败:', e)
  } finally {
    nodeMetricsRefreshing.value = false
  }
}

// 启动节点监控轮询
const startMetricsPolling = () => {
  fetchNodeMetrics()
}

// 停止节点监控轮询
const stopMetricsPolling = () => {
  nodeMetricsRefreshing.value = false
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

const formatMB = (mb) => {
  if (mb === null || mb === undefined || mb === 0) return '--'
  const value = Number(mb)
  if (Number.isNaN(value)) return '--'
  if (value >= 1024) return `${(value / 1024).toFixed(1)} GB`
  return `${Math.round(value)} MB`
}

const formatPercent = (value) => {
  if (value === null || value === undefined) return '--'
  const num = Number(value)
  if (Number.isNaN(num)) return '--'
  return `${num.toFixed(0)}%`
}

const getNodeRoleText = (node) => {
  if (node.role === 'slave') return 'Slave'
  if (diagnostics.value?.include_master || node.participating) return 'Master + 施压'
  if (diagnostics.value?.mode === 'distributed') return 'Master 调度'
  return 'Master'
}

const getNodeAgentText = (node) => {
  if (node.role === 'master') {
    return diagnostics.value?.include_master || node.participating ? '本机执行' : '本机调度'
  }
  return node.agent_status === 'online' ? '在线' : '离线'
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

const normalizeNodeMetrics = (nodes = []) => {
  return (nodes || []).map(node => {
    let stats = null
    try {
      if (typeof node.system_stats === 'string') {
        stats = node.system_stats ? JSON.parse(node.system_stats) : null
      } else {
        stats = node.system_stats || null
      }
    } catch {
      stats = null
    }
    return { ...node, stats }
  })
}

const applyErrorOverview = (overview) => {
  if (!overview) return
  const existing = errorAnalysis.value || {}
  errorAnalysis.value = {
    ...overview,
    records: errorRecordsLoaded.value ? (existing.records || []) : []
  }
  errorAnalysisLoaded.value = true
}

const applyExecutionSnapshot = (payload) => {
  if (!payload || typeof payload !== 'object') return
  lastDetailStreamAt.value = Date.now()
  if (payload.execution) {
    execution.value = payload.execution
    if (execution.value.status === 'success' && !execution.value.is_baseline && !baselineComparison.value && !baselineLoading.value) {
      loadBaselineComparison()
    }
  }
  if (payload.live_metrics) {
    liveMetrics.value = payload.live_metrics || { points: [] }
  }
  if (Array.isArray(payload.node_metrics)) {
    nodeMetrics.value = normalizeNodeMetrics(payload.node_metrics)
  }
  if (payload.error_overview) {
    applyErrorOverview(payload.error_overview)
  }
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
    if (isIgnorableExecutionRequestError(error)) return
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
    errorRecordsLoaded.value = true
  } catch (e) {
    if (isIgnorableExecutionRequestError(e)) return
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
    if (isIgnorableExecutionRequestError(error)) return
    console.error('获取实时指标失败:', error)
  } finally {
    liveRefreshing.value = false
  }
}

const refreshLiveExecutionSnapshot = async () => {
  if (leavingPage || !hasValidExecutionId.value || !shouldPollLive.value) return
  await Promise.allSettled([
    fetchLiveMetrics(),
    fetchNodeMetrics()
  ])
}

const ensureDetailFreshness = async () => {
  if (leavingPage || !pageVisible.value || !hasValidExecutionId.value || !shouldPollLive.value) return

  if (!detailStreamConnected.value) {
    await refreshLiveExecutionSnapshot()
    return
  }

  const lastSnapshotAt = lastDetailStreamAt.value || 0
  const idleMs = lastSnapshotAt > 0 ? Date.now() - lastSnapshotAt : DETAIL_STREAM_RECONNECT_MS

  if (idleMs >= DETAIL_STREAM_STALE_MS) {
    await Promise.allSettled([
      refreshLiveExecutionSnapshot(),
      fetchExecutionDetail()
    ])
  }

  if (idleMs >= DETAIL_STREAM_RECONNECT_MS && pageVisible.value) {
    connectDetailStream()
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

const handleManualRefreshMetrics = async () => {
  await Promise.allSettled([
    fetchExecutionDetail(),
    fetchLiveMetrics(),
    fetchNodeMetrics()
  ])
}

const stopDetailStream = () => {
  if (detailEventSource.value) {
    detailEventSource.value.close()
    detailEventSource.value = null
  }
  detailStreamConnected.value = false
  lastDetailStreamAt.value = 0
  if (!leavingPage) {
    setupAutoRefresh()
  }
}

const connectDetailStream = () => {
  if (leavingPage || !hasValidExecutionId.value || !shouldPollLive.value) {
    stopDetailStream()
    return
  }

  stopDetailStream()
  const es = new EventSource(`/api/executions/${executionNumericId.value}/stream`)

  es.onopen = () => {
    detailStreamConnected.value = true
    detailReconnectAttempts.value = 0
    lastDetailStreamAt.value = Date.now()
    setupAutoRefresh()
  }

  es.addEventListener('snapshot', (event) => {
    try {
      const payload = JSON.parse(event.data)
      applyExecutionSnapshot(payload)
    } catch (error) {
      console.error('解析执行快照失败:', error)
    }
  })

  es.addEventListener('complete', () => {
    stopDetailStream()
    fetchExecutionDetail()
    if (!errorRecordsLoaded.value) {
      fetchErrors()
    }
  })

  es.onerror = () => {
    stopDetailStream()
    if (!leavingPage && execution.value?.status === 'running' && detailReconnectAttempts.value < maxDetailReconnectAttempts) {
      detailReconnectAttempts.value += 1
      const delay = Math.min(1000 * Math.pow(2, detailReconnectAttempts.value - 1), 8000)
      window.setTimeout(() => {
        if (!leavingPage && pageVisible.value) {
          connectDetailStream()
        }
      }, delay)
    }
  }

  detailEventSource.value = es
}

const getAutoRefreshInterval = () => {
  if (!pageVisible.value) return 4000
  if (shouldPollLive.value) return 1000
  return 8000
}

const handleVisibilityChange = () => {
  pageVisible.value = document.visibilityState === 'visible'
  if (!leavingPage) {
    if (pageVisible.value && hasValidExecutionId.value) {
      Promise.allSettled([fetchExecutionDetail(), fetchLiveMetrics(), fetchNodeMetrics()])
      if (shouldPollLive.value) {
        connectDetailStream()
      }
    } else {
      stopDetailStream()
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

const downloadErrorReport = () => {
  if (!errorAnalysis.value) {
    ElMessage.warning('错误分析尚未准备完成')
    return
  }
  executionApi.downloadErrorReport(executionId.value)
}

const openReportInNewWindow = () => {
  if (!reportUrl.value) return
  window.open(reportUrl.value, '_blank', 'noopener,noreferrer')
}

// 下载完整结果
const downloadAll = () => {
  executionApi.downloadAll(executionId.value)
}

// 设置自动刷新
const setupAutoRefresh = () => {
  if (refreshTimer.value) {
    clearTimeout(refreshTimer.value)
  }
  let detailTick = 0
  const tick = async () => {
    if (leavingPage || !hasValidExecutionId.value) return
    detailTick += 1
    if (shouldPollLive.value) {
      if (detailStreamConnected.value) {
        await ensureDetailFreshness()
        if (detailTick % 3 === 0) {
          await Promise.allSettled([fetchExecutionDetail(), fetchNodeMetrics()])
        } else {
          await fetchNodeMetrics()
        }
      } else {
        await Promise.allSettled([refreshLiveExecutionSnapshot(), fetchExecutionDetail()])
        if (detailTick % 3 === 0 && (errorActiveTab.value === 'records' || errorAnalysisLoaded.value)) {
          await fetchErrors()
        }
        if (pageVisible.value) {
          connectDetailStream()
        }
      }
    } else {
      if (detailTick % 2 === 0) {
        await fetchExecutionDetail()
      }
      if (detailTick % 3 === 0 && errorActiveTab.value === 'records' && !errorRecordsLoaded.value) {
        await fetchErrors()
      }
    }

    if (!leavingPage) {
      refreshTimer.value = window.setTimeout(tick, getAutoRefreshInterval())
    }
  }

  refreshTimer.value = window.setTimeout(tick, getAutoRefreshInterval())
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

const disposeExecutionDetail = () => {
  leavingPage = true
  stopDetailStream()
  stopLogStream()
  stopMetricsPolling()
  if (logSnapshotController) {
    logSnapshotController.abort()
    logSnapshotController = null
  }
  if (refreshTimer.value) {
    clearTimeout(refreshTimer.value)
    refreshTimer.value = null
  }
  if (durationTimer.value) {
    clearInterval(durationTimer.value)
    durationTimer.value = null
  }
  if (typeof window !== 'undefined') {
    window.removeEventListener('scroll', handleDetailSectionScroll)
    if (detailSectionRaf) {
      window.cancelAnimationFrame(detailSectionRaf)
      detailSectionRaf = 0
    }
  }
  if (typeof document !== 'undefined') {
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  }
}

onMounted(() => {
  leavingPage = false
  Promise.allSettled([fetchExecutionDetail(), fetchLiveMetrics()]).then(() => {
    if (!shouldPollLive.value) {
      fetchErrors()
    } else {
      fetchNodeMetrics()
      connectDetailStream()
    }
    setupAutoRefresh()
    setupDurationTicker()
    setTimeout(() => {
      if (!leavingPage) {
        fetchLog()
      }
    }, 120)
    nextTick(() => {
      updateActiveDetailSection()
    })
  })
  if (typeof window !== 'undefined') {
    window.addEventListener('scroll', handleDetailSectionScroll, { passive: true })
  }
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', handleVisibilityChange)
  }
})

watch(() => execution.value.status, (status, prevStatus) => {
  if (status && status !== 'running' && status !== prevStatus && !errorRecordsLoaded.value) {
    fetchErrors()
  }
  if (status === 'running' && prevStatus !== 'running') {
    fetchNodeMetrics()
    connectDetailStream()
  } else if (status !== 'running' && prevStatus === 'running') {
    stopDetailStream()
    stopMetricsPolling()
  }
})

watch(errorActiveTab, (tab) => {
  if (tab === 'records' && errorAnalysis.value?.total_errors > 0 && !errorRecordsLoaded.value) {
    fetchErrors()
  }
})

watch(detailNavItems, () => {
  nextTick(() => {
    updateActiveDetailSection()
  })
}, { flush: 'post' })

onBeforeRouteLeave(() => {
  disposeExecutionDetail()
})

onBeforeUnmount(() => {
  disposeExecutionDetail()
})
</script>

<style scoped lang="scss">
.execution-detail-page {
  --detail-sticky-offset: 116px;
  padding: 4px 0 18px;
}

.detail-layout {
  display: grid;
  grid-template-columns: 196px minmax(0, 1fr);
  gap: 18px;
  padding-top: 10px;
  align-items: start;
}

.detail-sidebar {
  position: sticky;
  top: var(--detail-sticky-offset);
  align-self: start;
  z-index: 8;
}

.detail-sidebar-card {
  padding: 14px 12px;
  max-height: calc(100vh - var(--detail-sticky-offset) - 24px);
  overflow: auto;
  border-radius: 22px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  background:
    radial-gradient(circle at top, rgba(37, 99, 235, 0.12), transparent 40%),
    linear-gradient(180deg, rgba(11, 19, 32, 0.9), rgba(10, 18, 30, 0.82));
  box-shadow: 0 18px 40px rgba(2, 8, 23, 0.18);
}

.detail-sidebar-card::-webkit-scrollbar {
  width: 6px;
}

.detail-sidebar-card::-webkit-scrollbar-thumb {
  background: rgba(71, 85, 105, 0.72);
  border-radius: 999px;
}

.detail-sidebar-eyebrow {
  color: var(--accent-blue);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}

.detail-sidebar-title {
  margin-top: 8px;
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 700;
}

.detail-sidebar-desc {
  margin-top: 6px;
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.6;
}

.detail-sidebar-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 12px;
}

.detail-sidebar-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  width: 100%;
  padding: 9px 10px 8px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  background: rgba(15, 23, 42, 0.76);
  color: var(--text-secondary);
  cursor: pointer;
  text-align: left;
  transition: all 0.2s ease;
}

.detail-sidebar-item:hover {
  transform: translateX(2px);
  border-color: rgba(56, 189, 248, 0.24);
  color: var(--text-primary);
}

.detail-sidebar-item.active {
  border-color: rgba(56, 189, 248, 0.34);
  background: linear-gradient(180deg, rgba(18, 79, 120, 0.34), rgba(14, 25, 40, 0.96));
  box-shadow: inset 0 0 0 1px rgba(56, 189, 248, 0.12);
  color: var(--text-primary);
}

.detail-sidebar-kicker {
  color: var(--accent-blue);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.detail-sidebar-label {
  font-size: 13px;
  font-weight: 700;
  line-height: 1.2;
}

.detail-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.detail-main > .section-card {
  margin-bottom: 0;
  scroll-margin-top: calc(var(--detail-sticky-offset) + 14px);
}

.detail-section-nav {
  position: sticky;
  top: var(--detail-sticky-offset);
  z-index: 24;
  display: none;
  gap: 8px;
  align-items: center;
  overflow-x: auto;
  padding: 8px;
  margin: 0 0 12px;
  border-radius: 16px;
  border: 1px solid rgba(148, 163, 184, 0.14);
  background: rgba(8, 15, 28, 0.82);
  backdrop-filter: blur(16px);
  box-shadow: 0 16px 36px rgba(2, 8, 23, 0.24);
  scrollbar-width: thin;
}

.detail-section-nav::-webkit-scrollbar {
  height: 6px;
}

.detail-section-nav::-webkit-scrollbar-thumb {
  background: rgba(71, 85, 105, 0.72);
  border-radius: 999px;
}

.detail-nav-item {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  min-width: 70px;
  padding: 8px 10px 7px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  background: rgba(15, 23, 42, 0.72);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.detail-nav-item:hover {
  transform: translateY(-1px);
  border-color: rgba(56, 189, 248, 0.24);
  color: var(--text-primary);
}

.detail-nav-item.active {
  border-color: rgba(56, 189, 248, 0.34);
  background: linear-gradient(180deg, rgba(18, 79, 120, 0.34), rgba(14, 25, 40, 0.9));
  box-shadow: inset 0 0 0 1px rgba(56, 189, 248, 0.12);
  color: var(--text-primary);
}

.detail-nav-kicker {
  color: var(--accent-blue);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.detail-nav-label {
  font-size: 13px;
  font-weight: 700;
  line-height: 1.2;
  white-space: nowrap;
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

.topology-stage-intro,
.timeline-stage-intro,
.sampler-stage-intro {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-wrap: wrap;
  padding: 16px 18px;
  margin-bottom: 16px;
  border-radius: 18px;
  border: 1px solid rgba(90, 176, 255, 0.08);
  background:
    radial-gradient(circle at top right, rgba(37, 99, 235, 0.1), transparent 38%),
    linear-gradient(135deg, rgba(9, 18, 33, 0.72), rgba(17, 27, 44, 0.76));
}

.topology-stage-kicker,
.timeline-stage-kicker,
.sampler-stage-kicker {
  color: rgba(90, 176, 255, 0.9);
  font-size: 11px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  font-weight: 700;
}

.topology-stage-title,
.timeline-stage-title,
.sampler-stage-title {
  margin-top: 8px;
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 700;
}

.topology-stage-desc,
.timeline-stage-desc,
.sampler-stage-desc {
  margin-top: 8px;
  max-width: 720px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.75;
}

.topology-stage-badge,
.timeline-stage-badge,
.sampler-stage-badge {
  display: inline-flex;
  align-items: center;
  min-height: 34px;
  padding: 0 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 700;
}

.conclusion-list-card,
.timeline-card,
.sampler-overview-card {
  padding: 16px 18px;
  border-radius: var(--radius-md);
  background:
    linear-gradient(180deg, rgba(15, 24, 40, 0.88), rgba(12, 20, 35, 0.72)),
    rgba(10, 18, 32, 0.55);
  border: 1px solid rgba(255, 255, 255, 0.06);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
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
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 14px;
}

.timeline-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 172px;
}

.timeline-card.is-green {
  border-color: rgba(34, 197, 94, 0.22);
  background:
    radial-gradient(circle at top right, rgba(34, 197, 94, 0.12), transparent 34%),
    linear-gradient(180deg, rgba(15, 24, 40, 0.9), rgba(12, 20, 35, 0.76));
}

.timeline-card.is-warning {
  border-color: rgba(245, 158, 11, 0.22);
  background:
    radial-gradient(circle at top right, rgba(245, 158, 11, 0.12), transparent 34%),
    linear-gradient(180deg, rgba(15, 24, 40, 0.9), rgba(12, 20, 35, 0.76));
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
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.sampler-overview-card {
  min-height: 138px;
  padding: 16px;
  border-radius: 16px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
    rgba(9, 16, 28, 0.54);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.sampler-overview-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.sampler-overview-name {
  margin-top: 8px;
  font-size: 17px;
  font-weight: 600;
  color: var(--text-primary);
}

.sampler-overview-value {
  margin-top: 6px;
  font-size: 22px;
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

.sampler-table-shell {
  overflow: hidden;
  border-radius: 18px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background:
    linear-gradient(180deg, rgba(11, 19, 32, 0.86), rgba(12, 20, 34, 0.72));
}

// 区域卡片
.section-card {
  background: var(--bg-card);
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  padding: 18px;
  margin-bottom: 16px;
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
  margin-bottom: 6px;
}

.section-title.is-danger-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #ff6b6b;
}

.section-desc {
  margin: 0;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.65;
}

.section-header {
  margin-bottom: 14px;
}

.section-header.with-tools {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.section-header-tools {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

// 顶部信息栏
.header-section {
  padding: 16px 18px;
  margin-bottom: 16px;
  background:
    radial-gradient(circle at top right, rgba(37, 99, 235, 0.14), transparent 28%),
    linear-gradient(180deg, rgba(24, 36, 58, 0.95), rgba(18, 28, 44, 0.94));
}

.header-content {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(340px, 0.84fr);
  align-items: start;
  gap: 18px;
}

.header-left {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  min-width: 0;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 42px;
  padding: 0 16px;
  background: rgba(0, 102, 255, 0.1);
  border: 1px solid rgba(0, 102, 255, 0.18);
  border-radius: 14px;
  color: var(--accent-blue);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.25s ease;
  flex-shrink: 0;
}

.back-btn:hover {
  background: rgba(0, 102, 255, 0.15);
  border-color: rgba(0, 102, 255, 0.3);
}

.script-info {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.script-eyebrow-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.script-eyebrow {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.14em;
  color: rgba(56, 189, 248, 0.95);
}

.script-meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.06);
  max-width: min(260px, 100%);
}

.script-meta-chip-label {
  font-size: 11px;
  color: var(--text-secondary);
}

.script-meta-chip-value {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.script-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.script-name {
  font-size: 28px;
  font-weight: 700;
  margin: 0;
  color: var(--text-primary);
  line-height: 1.12;
  word-break: break-word;
}

.script-subtitle {
  max-width: 720px;
  font-size: 14px;
  line-height: 1.7;
  color: rgba(226, 232, 240, 0.76);
}

.status-tag {
  font-weight: 700;
  border-radius: 999px;
  padding-inline: 10px;
}

.header-right {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 10px;
  min-width: 0;
}

.header-actions-row {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  width: 100%;
  flex-wrap: wrap;
}

.detail-sync-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 0 12px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.58);
  border: 1px solid rgba(148, 163, 184, 0.16);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;

  &.is-stream {
    color: #38bdf8;
    border-color: rgba(56, 189, 248, 0.24);
  }

  &.is-polling {
    color: #f59e0b;
    border-color: rgba(245, 158, 11, 0.22);
  }

  &.is-lag {
    color: #f97316;
    border-color: rgba(249, 115, 22, 0.22);
  }

  &.is-stable {
    color: #22c55e;
    border-color: rgba(34, 197, 94, 0.22);
  }
}

.detail-sync-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: currentColor;
  box-shadow: 0 0 0 4px rgba(255, 255, 255, 0.04);
}

.header-metric-strip {
  width: 100%;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 6px;
}

.header-metric-card {
  padding: 10px 12px;
  border-radius: 16px;
  background: rgba(8, 15, 28, 0.42);
  border: 1px solid rgba(255, 255, 255, 0.06);
  min-height: 74px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.header-metric-card.is-blue {
  border-color: rgba(59, 130, 246, 0.18);
}

.header-metric-card.is-purple {
  border-color: rgba(168, 85, 247, 0.18);
}

.header-metric-card.is-red {
  border-color: rgba(239, 68, 68, 0.18);
}

.header-metric-card.is-green {
  border-color: rgba(34, 197, 94, 0.18);
}

.header-metric-card.is-amber {
  border-color: rgba(245, 158, 11, 0.18);
}

.header-metric-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.header-metric-value {
  font-size: 18px;
  font-weight: 700;
  line-height: 1.2;
  color: var(--text-primary);
  word-break: break-word;
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
  grid-template-columns: minmax(0, 1.12fr) minmax(320px, 0.88fr);
  gap: 14px;
}

// 基准线对比卡片
.baseline-compare-card {
  background: rgba(234, 179, 8, 0.08);
  border: 1px solid rgba(234, 179, 8, 0.2);
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 16px;
  
  .baseline-header {
    margin-bottom: 12px;
    
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

.overview-stage-intro {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  flex-wrap: wrap;
  margin-bottom: 14px;
  padding: 12px 14px;
  border-radius: 16px;
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.1), transparent 38%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
    rgba(15, 23, 42, 0.58);
  border: 1px solid rgba(148, 163, 184, 0.12);
}

.overview-stage-kicker {
  color: #38bdf8;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.overview-stage-title {
  margin-top: 6px;
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.35;
}

.overview-stage-desc {
  margin-top: 6px;
  max-width: 760px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.7;
}

.overview-stage-badge {
  display: inline-flex;
  align-items: center;
  min-height: 36px;
  padding: 0 14px;
  border-radius: 999px;
  color: #bae6fd;
  font-size: 12px;
  font-weight: 700;
  background: rgba(56, 189, 248, 0.12);
  border: 1px solid rgba(56, 189, 248, 0.16);
}

.live-metrics-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 8px;
  margin-bottom: 12px;
}

.live-ops-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
}

.live-ops-pill {
  display: inline-flex;
  align-items: center;
  min-height: 30px;
  padding: 0 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(148, 163, 184, 0.12);
  color: var(--text-secondary);
  font-size: 12px;
}

.live-section-meta {
  display: inline-flex;
  align-items: center;
  min-height: 34px;
  padding: 0 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(148, 163, 184, 0.12);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.live-refresh-btn {
  min-height: 34px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(56, 189, 248, 0.08);
  border: 1px solid rgba(56, 189, 248, 0.14);
}

.live-summary-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border-radius: 12px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), rgba(255, 255, 255, 0.012)),
    rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.05);
  min-height: 80px;
}

.live-summary-label {
  color: var(--text-secondary);
  font-size: 12px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.live-summary-value {
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 700;
  font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
  line-height: 1.15;
}

.live-chart-stage {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.live-chart-group {
  padding: 16px;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
    rgba(15, 23, 42, 0.54);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);

  &.is-throughput {
    background:
      radial-gradient(circle at top left, rgba(56, 189, 248, 0.12), transparent 36%),
      rgba(15, 23, 42, 0.54);
  }

  &.is-latency {
    background:
      radial-gradient(circle at top left, rgba(168, 85, 247, 0.12), transparent 36%),
      rgba(15, 23, 42, 0.54);
  }

  &.is-quality {
    background:
      radial-gradient(circle at top left, rgba(132, 204, 22, 0.12), transparent 36%),
      rgba(15, 23, 42, 0.54);
  }
}

.live-chart-group-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  margin-bottom: 12px;
}

.live-chart-group-head > div:first-child {
  flex: 1;
  min-width: 0;
}

.live-chart-kicker {
  color: #38bdf8;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.live-chart-group h3 {
  margin: 6px 0;
  color: var(--text-primary);
  font-size: 16px;
  font-weight: 700;
}

.live-chart-group p {
  margin: 0;
  max-width: 680px;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.live-chart-group-meta {
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
}

.live-chart-group-chip {
  display: inline-flex;
  align-items: center;
  min-height: 30px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(148, 163, 184, 0.12);
  color: var(--text-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.live-chart-cluster {
  display: grid;
  gap: 12px;
}

.live-chart-cluster--two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.live-chart-group :deep(.metric-trend-card) {
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.032), rgba(255, 255, 255, 0.014)),
    rgba(8, 15, 28, 0.62);
  border-color: rgba(148, 163, 184, 0.12);
  box-shadow: none;
  border-radius: 18px;
}

.overview-hero {
  padding: 16px;
  border-radius: 16px;
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
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
  margin-top: 14px;
}

.overview-primary-card,
.overview-mini-card {
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(10, 17, 31, 0.72);
}

.overview-primary-card {
  min-height: 140px;
  padding: 16px;
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
  font-size: clamp(28px, 3.1vw, 42px);
  font-weight: 700;
  line-height: 1.02;
  margin: 12px 0;
  word-break: break-word;
}

.overview-card-caption,
.overview-mini-caption {
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.55;
}

.overview-mini-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 10px;
}

.overview-mini-card {
  min-height: 98px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.overview-mini-value {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.1;
  margin: 8px 0 6px;
}

.text-blue { color: var(--accent-blue); }
.text-purple { color: var(--accent-purple); }
.text-green { color: var(--accent-green); }
.text-red { color: var(--accent-red); }
.text-warning { color: #f59e0b; }
.text-danger { color: var(--accent-red); }

.diagnostic-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 12px;
}

.diagnostic-stage {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.diagnostic-stage-hero {
  padding: 12px 14px;
  border-radius: 16px;
  background:
    radial-gradient(circle at top left, rgba(99, 102, 241, 0.1), transparent 36%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.026), rgba(255, 255, 255, 0.012)),
    rgba(15, 23, 42, 0.56);
  border: 1px solid rgba(148, 163, 184, 0.12);
}

.diagnostic-stage-kicker {
  color: #8b5cf6;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.diagnostic-stage-title {
  margin-top: 6px;
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.35;
}

.diagnostic-stage-desc {
  margin-top: 6px;
  max-width: 760px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.7;
}

.diagnostic-card {
  min-height: 104px;
  padding: 14px;
  border-radius: 15px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
    rgba(9, 16, 28, 0.54);
  border: 1px solid rgba(255, 255, 255, 0.06);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.diagnostic-label {
  color: var(--text-secondary);
  font-size: 12px;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.diagnostic-value {
  color: var(--text-primary);
  font-size: 20px;
  font-weight: 700;
  line-height: 1.1;
  word-break: break-word;
}

.diagnostic-caption {
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.55;
}

.diagnostic-warning-stack {
  margin-top: 14px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
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

.diag-preflight-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 18px;
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(17, 24, 39, 0.98), rgba(30, 41, 59, 0.82));
  border: 1px solid rgba(56, 189, 248, 0.16);

  &.is-warning {
    border-color: rgba(245, 158, 11, 0.22);
  }

  &.is-danger {
    border-color: rgba(239, 68, 68, 0.24);
  }
}

.diag-preflight-hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 18px;
}

.diag-preflight-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.diag-preflight-badge {
  display: inline-flex;
  align-items: center;
  padding: 5px 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  background: rgba(34, 197, 94, 0.16);
  color: #4ade80;

  &.is-warning {
    background: rgba(245, 158, 11, 0.16);
    color: #fbbf24;
  }

  &.is-danger {
    background: rgba(239, 68, 68, 0.16);
    color: #f87171;
  }
}

.diag-preflight-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
}

.diag-preflight-summary {
  max-width: 660px;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.7;
}

.diag-preflight-score {
  min-width: 92px;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.diag-preflight-score-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.diag-preflight-score-value {
  font-size: 36px;
  line-height: 1;
  font-weight: 800;
  color: var(--text-primary);
}

.diag-preflight-facts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.diag-preflight-fact,
.topology-card {
  padding: 14px 16px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.diag-preflight-fact-label,
.topology-meta-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.diag-preflight-fact-value {
  margin-top: 8px;
  font-size: 22px;
  font-weight: 700;
}

.diag-preflight-fact-detail {
  margin-top: 8px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-secondary);
}

.diag-preflight-list-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 12px;
}

.diag-preflight-list-card {
  padding: 14px 16px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.diag-preflight-list-title {
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 700;
  color: var(--text-primary);
}

.diag-preflight-list-item {
  position: relative;
  padding-left: 12px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-secondary);

  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 9px;
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: rgba(56, 189, 248, 0.9);
  }
}

.topology-meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 16px;
}

.topology-meta-chip {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 220px;
  padding: 12px 14px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);

  code {
    color: var(--text-primary);
    word-break: break-all;
  }
}

.topology-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.topology-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 174px;
  padding: 18px;
  border-radius: 18px;
  background:
    linear-gradient(180deg, rgba(12, 20, 35, 0.86), rgba(16, 26, 43, 0.72));
  border: 1px solid rgba(255, 255, 255, 0.06);

  &.is-green {
    border-color: rgba(34, 197, 94, 0.18);
    background:
      radial-gradient(circle at top right, rgba(34, 197, 94, 0.12), transparent 34%),
      linear-gradient(180deg, rgba(12, 20, 35, 0.9), rgba(16, 26, 43, 0.76));
  }

  &.is-warning {
    border-color: rgba(245, 158, 11, 0.18);
    background:
      radial-gradient(circle at top right, rgba(245, 158, 11, 0.12), transparent 34%),
      linear-gradient(180deg, rgba(12, 20, 35, 0.9), rgba(16, 26, 43, 0.76));
  }

  &.is-blue {
    border-color: rgba(56, 189, 248, 0.18);
    background:
      radial-gradient(circle at top right, rgba(56, 189, 248, 0.12), transparent 34%),
      linear-gradient(180deg, rgba(12, 20, 35, 0.9), rgba(16, 26, 43, 0.76));
  }
}

.topology-card-header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.topology-card-role {
  font-size: 12px;
  color: var(--text-secondary);
}

.topology-card-name {
  margin-top: 4px;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  word-break: break-word;
}

.topology-card-status {
  display: inline-flex;
  align-items: center;
  height: fit-content;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  background: rgba(56, 189, 248, 0.12);
  color: #38bdf8;

  &.is-green {
    background: rgba(34, 197, 94, 0.12);
    color: #4ade80;
  }

  &.is-warning {
    background: rgba(245, 158, 11, 0.12);
    color: #fbbf24;
  }
}

.topology-card-note,
.topology-card-foot {
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-secondary);
}

.topology-card-foot {
  color: var(--text-primary-soft);
}

// 详细统计
.stats-meta-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
  margin-bottom: 16px;
}

.stats-meta-card,
.detail-group-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 14px;
}

.stats-meta-card {
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 4px;
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
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 12px;
}

.detail-group-card {
  padding: 14px;
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

.report-actions {
  align-items: center;
}

.report-browser-frame {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.report-browser-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 12px 14px;
  border-radius: 16px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
    rgba(15, 23, 42, 0.56);
  border: 1px solid rgba(148, 163, 184, 0.12);
}

.report-browser-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.report-browser-label {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.report-browser-tip {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.report-browser-url {
  color: var(--text-primary);
  font-size: 12px;
  word-break: break-all;
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
  padding: 20px;
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
  gap: 12px;
  flex-wrap: wrap;
  padding: 12px 16px;
  background: var(--bg-secondary);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.terminal-title-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.terminal-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--text-primary);
  font-weight: 700;
}

.terminal-subtitle {
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.terminal-actions {
  display: flex;
  gap: 8px;
}

.terminal-section-meta {
  align-items: center;
}

.terminal-meta-pill {
  display: inline-flex;
  align-items: center;
  min-height: 34px;
  padding: 0 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(148, 163, 184, 0.12);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.terminal-toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  flex: 1;
  min-width: 220px;
  margin: 0;
}

.log-search-input {
  width: 240px;
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

.terminal-btn.is-active {
  background: rgba(56, 189, 248, 0.12);
  border-color: rgba(56, 189, 248, 0.28);
  color: #38bdf8;
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
    margin-bottom: 16px;
  }

  .error-actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 10px;
  }

  .error-overview-shell {
    display: grid;
    grid-template-columns: minmax(0, 1.35fr) 320px;
    gap: 16px;
    margin-bottom: 20px;

    @media (max-width: 1200px) {
      grid-template-columns: 1fr;
    }
  }

  .error-overview-hero {
    padding: 18px;
    border-radius: 20px;
    background:
      radial-gradient(circle at top left, rgba(239, 68, 68, 0.16), transparent 34%),
      linear-gradient(180deg, rgba(255, 255, 255, 0.03), rgba(255, 255, 255, 0.015)),
      rgba(15, 23, 42, 0.6);
    border: 1px solid rgba(239, 68, 68, 0.14);
  }

  .error-overview-kicker {
    color: #fda4af;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.18em;
    text-transform: uppercase;
  }

  .error-overview-title {
    margin-top: 8px;
    color: var(--text-primary);
    font-size: 24px;
    font-weight: 700;
    line-height: 1.35;
  }

  .error-overview-desc {
    margin-top: 8px;
    max-width: 780px;
    color: var(--text-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .error-overview-side {
    display: flex;
  }

  .error-context-strip {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
    gap: 12px;
    margin-top: 16px;
  }

  .error-context-card {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-height: 110px;
    padding: 16px;
    border-radius: 14px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.06);

    &.is-danger {
      background: rgba(255, 77, 79, 0.08);
      border-color: rgba(255, 77, 79, 0.16);
    }

    &.is-warning {
      background: rgba(250, 173, 20, 0.09);
      border-color: rgba(250, 173, 20, 0.14);
    }

    &.is-success {
      background: rgba(34, 197, 94, 0.08);
      border-color: rgba(34, 197, 94, 0.14);
    }
  }

  .error-context-label {
    color: var(--text-secondary);
    font-size: 12px;
  }

  .error-context-value {
    color: var(--text-primary);
    font-size: 22px;
    font-weight: 700;
  }

  .error-context-meta {
    color: var(--text-secondary);
    font-size: 12px;
    line-height: 1.55;
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
    grid-template-columns: 1fr;
    gap: 12px;
    margin-bottom: 0;
    width: 100%;
  }

  .error-stat-card {
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: 116px;
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.032), rgba(255, 255, 255, 0.015)),
      rgba(38, 14, 18, 0.46);
    border: 1px solid rgba(255, 77, 79, 0.14);
    border-radius: 16px;
    padding: 16px;

    .error-stat-value {
      font-size: 28px;
      font-weight: 700;
      color: #ff4d4f;
      margin-bottom: 6px;
    }

    .error-stat-label {
      font-size: 12px;
      color: var(--text-secondary);
    }
  }

  .error-insight-shell {
    margin-bottom: 20px;
    padding: 18px;
    border-radius: 20px;
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
      rgba(15, 23, 42, 0.54);
    border: 1px solid rgba(148, 163, 184, 0.12);
  }

  .error-insight-shell-head {
    margin-bottom: 16px;
  }

  .error-insight-title {
    margin-top: 6px;
    color: var(--text-primary);
    font-size: 20px;
    font-weight: 700;
  }

  .error-insight-desc {
    margin-top: 6px;
    color: var(--text-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .error-insight-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 16px;
    margin-bottom: 0;
  }

  .error-insight-card {
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.032), rgba(255, 255, 255, 0.015)),
      rgba(9, 16, 28, 0.54);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 16px;
    padding: 18px;
    min-height: 180px;
  }

  .error-cluster-list,
  .error-report-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .error-cluster-item,
  .error-report-item {
    padding: 12px 14px;
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.05);
  }

  .error-cluster-main {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 6px;
  }

  .error-cluster-label {
    color: var(--text-primary);
    font-size: 13px;
    font-weight: 600;
  }

  .error-cluster-count {
    color: #ff6b6b;
    font-size: 12px;
    font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
    flex-shrink: 0;
  }

  .error-cluster-meta {
    display: flex;
    flex-direction: column;
    gap: 4px;
    color: var(--text-secondary);
    font-size: 12px;
    line-height: 1.5;
  }

  .error-report-item {
    color: var(--text-primary);
    line-height: 1.6;
    position: relative;
    padding-left: 18px;

    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 9px;
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: #ff6b6b;
      box-shadow: 0 0 12px rgba(255, 107, 107, 0.35);
    }
  }

  .error-workbench {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .error-workbench-toolbar {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    align-items: flex-start;
    flex-wrap: wrap;
    padding: 14px 16px;
    border-radius: 16px;
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
      rgba(11, 18, 31, 0.56);
    border: 1px solid rgba(148, 163, 184, 0.1);
  }

  .error-workbench-title {
    color: var(--text-primary);
    font-size: 16px;
    font-weight: 700;
  }

  .error-workbench-desc {
    margin-top: 6px;
    max-width: 720px;
    color: var(--text-secondary);
    font-size: 12px;
    line-height: 1.7;
  }

  .error-workbench-meta {
    display: inline-flex;
    align-items: center;
    min-height: 32px;
    padding: 0 12px;
    border-radius: 999px;
    color: #fda4af;
    font-size: 12px;
    font-weight: 700;
    background: rgba(255, 77, 79, 0.1);
    border: 1px solid rgba(255, 77, 79, 0.16);
  }

  .error-note-stack {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .error-note-card {
    padding: 12px 14px;
    border-radius: 14px;
    font-size: 13px;
    line-height: 1.7;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(148, 163, 184, 0.12);
    color: var(--text-secondary);

    &.is-info {
      background: rgba(56, 189, 248, 0.08);
      border-color: rgba(56, 189, 248, 0.14);
      color: #bae6fd;
    }

    &.is-warning {
      background: rgba(250, 173, 20, 0.1);
      border-color: rgba(250, 173, 20, 0.14);
      color: #fcd34d;
    }
  }

  .error-note-meta {
    margin-top: 6px;
    color: var(--text-secondary);
    font-size: 12px;
  }

  .error-tabs {
    :deep(.el-tabs__header) {
      margin-bottom: 18px;
    }

    :deep(.el-tabs__nav-wrap::after) {
      background: rgba(148, 163, 184, 0.12);
    }

    :deep(.el-tabs__item) {
      color: var(--text-secondary);
      min-height: 40px;

      &.is-active {
        color: #ff6b6b;
      }
    }

    :deep(.el-tabs__active-bar) {
      background-color: #ff6b6b;
    }
  }

  .error-table-shell {
    overflow: hidden;
    border-radius: 18px;
    border: 1px solid rgba(148, 163, 184, 0.1);
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.026), rgba(255, 255, 255, 0.012)),
      rgba(9, 16, 28, 0.54);
  }

  .error-types-table,
  .error-records-table {
    min-width: 1100px;

    :deep(.el-table__header-wrapper th) {
      background: rgba(255, 255, 255, 0.04);
    }

    :deep(.el-table__row:hover > td) {
      background: rgba(255, 255, 255, 0.03);
    }

    :deep(.el-table__cell) {
      border-bottom-color: rgba(148, 163, 184, 0.08);
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
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.032), rgba(255, 255, 255, 0.015)),
      rgba(9, 16, 28, 0.54);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 16px;
    padding: 18px;
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
    margin-bottom: 0;
    flex-wrap: wrap;
    padding: 14px 16px;
    border-radius: 16px;
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.028), rgba(255, 255, 255, 0.012)),
      rgba(11, 18, 31, 0.56);
    border: 1px solid rgba(148, 163, 184, 0.1);

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

  .truncate-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.sampler-table {
  min-width: 980px;
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

  .live-chart-cluster--two {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 1024px) {
  .header-content {
    grid-template-columns: 1fr;
  }

  .header-actions-row {
    justify-content: flex-start;
  }

  .live-chart-group-head {
    flex-direction: column;
    align-items: flex-start;
  }

  .error-analysis-section {
    .error-workbench-toolbar,
    .error-filters-row {
      flex-direction: column;
      align-items: stretch;
    }

    .error-filter-result {
      margin-left: 0 !important;
    }
  }
}

@media (max-width: 768px) {
  .execution-detail-page {
    --detail-sticky-offset: 100px;
  }

  .detail-section-nav {
    display: flex;
    padding: 8px;
    margin-bottom: 12px;
    border-radius: 16px;
  }

  .detail-nav-item {
    min-width: 64px;
    padding: 8px 10px 7px;
  }

  .detail-nav-label {
    font-size: 12px;
  }

  .overview-hero,
  .overview-mini-card,
  .overview-primary-card,
  .diagnostic-card {
    padding: 16px;
  }

  .section-header.with-tools {
    flex-direction: column;
    align-items: stretch;
  }

  .diag-preflight-hero,
  .topology-card-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .overview-status-strip {
    width: 100%;
    justify-content: flex-start;
    flex-wrap: wrap;
    padding: 10px 14px;
    border-radius: 14px;
  }

  .live-chart-group {
    padding: 16px;
    border-radius: 18px;
  }
  
  .header-right {
    width: 100%;
  }

  .header-actions-row,
  .header-metric-strip {
    width: 100%;
  }

  .section-header-tools {
    justify-content: flex-start;
  }

  .report-browser-bar,
  .terminal-section-meta {
    flex-direction: column;
    align-items: stretch;
  }

  .terminal-toolbar {
    width: 100%;
    justify-content: stretch;
  }

  .header-metric-strip {
    grid-template-columns: 1fr;
  }

  .script-name {
    font-size: 24px;
  }

  .section-card {
    padding: 18px;
    border-radius: 18px;
  }
}

@media (max-width: 1200px) {
  .detail-layout {
    grid-template-columns: 1fr;
  }

  .detail-sidebar {
    display: none;
  }

  .detail-section-nav {
    display: flex;
  }
}

// 节点监控面板样式
.node-metrics-panel {
  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;
  }

  .node-metrics-grid {
    align-items: stretch;
  }
  
  .node-card {
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 16px;
    padding: 18px;
    transition: all 0.25s ease;
    
    &:hover {
      background: rgba(255, 255, 255, 0.06);
      border-color: rgba(255, 255, 255, 0.12);
    }
  }
  
  .node-header {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    margin-bottom: 14px;
    padding-bottom: 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    
    .node-header-main {
      min-width: 0;
      flex: 1;
    }

    .node-name-row {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 6px;
      flex-wrap: wrap;
    }

    .node-name {
      color: var(--text-primary);
      font-weight: 700;
      font-size: 16px;
    }

    .node-role {
      color: var(--text-secondary);
      font-size: 12px;
      background: rgba(255, 255, 255, 0.06);
      padding: 4px 10px;
      border-radius: 999px;
      font-weight: 600;
    }

    .node-host {
      color: var(--text-secondary);
      font-size: 12px;
      line-height: 1.6;
      word-break: break-all;
    }
    
    .node-status {
      margin-left: auto;
      font-size: 12px;
      font-weight: 500;
      padding: 4px 10px;
      border-radius: 999px;
      flex-shrink: 0;
      
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
    &.detailed {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 12px;
    }

    .node-resource-card {
      padding: 14px;
      border-radius: 14px;
      background: rgba(15, 23, 42, 0.5);
      border: 1px solid rgba(255, 255, 255, 0.04);
    }

    .node-resource-head {
      display: flex;
      justify-content: space-between;
      gap: 8px;
      align-items: center;
      margin-bottom: 10px;

      span {
        color: var(--text-secondary);
        font-size: 12px;
      }

      strong {
        color: var(--text-primary);
        font-size: 14px;
        font-weight: 700;
      }
    }

    .node-resource-meta {
      color: var(--text-secondary);
      font-size: 12px;
      margin-top: 8px;
    }
  }

  .node-meta-grid {
    grid-column: 1 / -1;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }

  .node-meta-item {
    padding: 14px;
    border-radius: 14px;
    background: rgba(15, 23, 42, 0.44);
    border: 1px solid rgba(255, 255, 255, 0.04);
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: center;

    span {
      color: var(--text-secondary);
      font-size: 12px;
    }

    strong {
      color: var(--text-primary);
      font-size: 14px;
      font-weight: 700;
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

    .node-stats.detailed,
    .node-meta-grid {
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
