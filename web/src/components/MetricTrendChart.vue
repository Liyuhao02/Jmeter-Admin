<template>
  <div class="metric-trend-card">
    <div class="metric-trend-head">
      <div>
        <div class="metric-trend-label">{{ title }}</div>
        <div class="metric-trend-value">
          {{ headline }}
          <span v-if="unit" class="metric-trend-unit">{{ unit }}</span>
        </div>
      </div>
      <div class="metric-trend-actions">
        <div class="metric-trend-meta" v-if="subline">{{ subline }}</div>
        <button
          v-if="showExpand"
          class="expand-btn"
          type="button"
          @click="$emit('expand')"
        >
          放大
        </button>
      </div>
    </div>

    <div
      ref="chartRef"
      class="metric-trend-chart"
      @mouseleave="clearHover"
    >
      <svg
        :viewBox="`0 0 ${svgWidth} ${svgHeight}`"
        preserveAspectRatio="none"
        class="metric-trend-svg"
        :style="{ height: `${height}px` }"
      >
        <line
          x1="0"
          :y1="chartBottom"
          :x2="svgWidth"
          :y2="chartBottom"
          class="axis-line"
        />
        <line
          :x1="chartLeft"
          y1="0"
          :x2="chartLeft"
          :y2="chartBottom"
          class="axis-line"
        />
        <line
          v-for="tick in yTicks"
          :key="`grid-${tick.y}`"
          :x1="chartLeft"
          :y1="tick.y"
          :x2="chartRight"
          :y2="tick.y"
          class="grid-line"
        />
        <text
          v-for="tick in yTicks"
          :key="`label-${tick.y}`"
          :x="chartLeft - 12"
          :y="tick.y + 4"
          class="axis-text axis-text-y"
        >
          {{ tick.label }}
        </text>
        <text
          v-for="tick in xTicks"
          :key="`x-${tick.index}`"
          :x="tick.x"
          :y="svgHeight - 10"
          class="axis-text axis-text-x"
        >
          {{ tick.label }}
        </text>
        <polyline
          v-if="linePoints"
          :points="linePoints"
          class="trend-line"
          :style="{ stroke: color }"
        />
        <polyline
          v-if="secondLinePoints && secondField"
          :points="secondLinePoints"
          class="trend-line second-line"
          :style="{ stroke: secondColor }"
        />
        <circle
          v-if="activePoint"
          :cx="activePoint.x"
          :cy="activePoint.y"
          r="4.5"
          class="active-dot"
          :style="{ stroke: color, fill: color }"
        />
        <circle
          v-if="secondActivePoint && secondField"
          :cx="secondActivePoint.x"
          :cy="secondActivePoint.y"
          r="4.5"
          class="active-dot second-active-dot"
          :style="{ stroke: secondColor, fill: secondColor }"
        />
        <line
          v-if="activePoint"
          :x1="activePoint.x"
          y1="8"
          :x2="activePoint.x"
          :y2="chartBottom"
          class="hover-line"
        />
        <rect
          :x="chartLeft"
          y="0"
          :width="chartWidth"
          :height="chartBottom"
          class="chart-overlay"
          @mousemove="handleHover"
        />
      </svg>
      <div
        v-if="tooltipVisible && activePoint"
        class="metric-tooltip"
        :style="tooltipStyle"
      >
        <div class="metric-tooltip-time">{{ activePoint.timestamp }}</div>
        <div class="metric-tooltip-row">
          <span class="metric-tooltip-dot" :style="{ background: color }"></span>
          <span>{{ title }}</span>
          <strong>{{ formatMetricValue(activePoint.rawValue) }}{{ unitSuffix }}</strong>
        </div>
        <div v-if="secondActivePoint && secondField" class="metric-tooltip-row second-row">
          <span class="metric-tooltip-dot" :style="{ background: secondColor }"></span>
          <span>{{ secondLabel || secondField }}</span>
          <strong>{{ formatMetricValue(secondActivePoint.rawValue) }}{{ unitSuffix }}</strong>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  title: { type: String, required: true },
  value: { type: [String, Number], default: '-' },
  unit: { type: String, default: '' },
  subline: { type: String, default: '' },
  points: { type: Array, default: () => [] },
  field: { type: String, required: true },
  color: { type: String, default: '#38bdf8' },
  height: { type: Number, default: 220 },
  showExpand: { type: Boolean, default: false },
  maxXTicks: { type: Number, default: 4 },
  // 第二条线配置
  secondField: { type: String, default: '' },
  secondColor: { type: String, default: '#ef4444' },
  secondLabel: { type: String, default: '' }
})

defineEmits(['expand'])

const svgWidth = 420
const svgHeight = computed(() => props.height)
const chartLeft = computed(() => (props.height >= 320 ? 72 : 60))
const chartRight = computed(() => svgWidth - 14)
const chartTop = 12
const chartBottom = computed(() => svgHeight.value - (props.height >= 320 ? 42 : 30))
const chartWidth = computed(() => chartRight.value - chartLeft.value)
const chartHeight = computed(() => chartBottom.value - chartTop)

const chartRef = ref(null)
const activeIndex = ref(-1)
const tooltipVisible = ref(false)

const chartData = computed(() => {
  return props.points.map((item, index) => ({
    index,
    timestamp: item?.timestamp || '--:--:--',
    epochSecond: Number(item?.epoch_second || 0),
    rawValue: Number(item?.[props.field] || 0),
    secondRawValue: props.secondField ? Number(item?.[props.secondField] || 0) : 0
  }))
})

const chartValues = computed(() => chartData.value.map(item => item.rawValue))
const secondChartValues = computed(() => props.secondField ? chartData.value.map(item => item.secondRawValue) : [])

const valueMax = computed(() => {
  const values = chartValues.value
  const secondValues = secondChartValues.value
  if (!values.length && !secondValues.length) return 1
  const max1 = values.length ? Math.max(...values) : 0
  const max2 = secondValues.length ? Math.max(...secondValues) : 0
  return Math.max(max1, max2, 1)
})

const yTicks = computed(() => {
  const count = 4
  const max = valueMax.value
  return Array.from({ length: count + 1 }, (_, index) => {
    const ratio = index / count
    const value = max * (1 - ratio)
    return {
      y: chartTop + chartHeight.value * ratio,
      label: formatTickValue(value)
    }
  })
})

const chartPoints = computed(() => {
  const values = chartValues.value
  if (!values.length) return []
  const stepX = values.length > 1 ? chartWidth.value / (values.length - 1) : 0

  return chartData.value.map((item, index) => {
    const ratio = valueMax.value > 0 ? item.rawValue / valueMax.value : 0
    const x = chartLeft.value + index * stepX
    const y = chartBottom.value - ratio * chartHeight.value
    return {
      ...item,
      x,
      y: Math.max(chartTop, Math.min(chartBottom.value, y))
    }
  })
})

const secondChartPoints = computed(() => {
  if (!props.secondField) return []
  const values = secondChartValues.value
  if (!values.length) return []
  const stepX = values.length > 1 ? chartWidth.value / (values.length - 1) : 0

  return chartData.value.map((item, index) => {
    const ratio = valueMax.value > 0 ? item.secondRawValue / valueMax.value : 0
    const x = chartLeft.value + index * stepX
    const y = chartBottom.value - ratio * chartHeight.value
    return {
      ...item,
      x,
      y: Math.max(chartTop, Math.min(chartBottom.value, y)),
      rawValue: item.secondRawValue
    }
  })
})

const secondLinePoints = computed(() => {
  if (secondChartPoints.value.length < 2) return ''
  return secondChartPoints.value.map(point => `${point.x},${point.y}`).join(' ')
})

const linePoints = computed(() => {
  if (chartPoints.value.length < 2) return ''
  return chartPoints.value.map(point => `${point.x},${point.y}`).join(' ')
})

const xTicks = computed(() => {
  const points = chartPoints.value
  if (!points.length) return []

  const tickCapacity = Math.max(2, Math.floor(chartWidth.value / 96))
  const desired = Math.min(props.maxXTicks, tickCapacity, points.length - 1)
  if (desired <= 0) {
    return [{
      index: 0,
      x: points[0].x,
      label: points[0].timestamp
    }]
  }

  const indices = Array.from({ length: desired + 1 }, (_, idx) => {
    return Math.round((idx / desired) * (points.length - 1))
  }).filter((value, idx, arr) => arr.indexOf(value) === idx)

  return indices.map(index => ({
    index,
    x: points[index].x,
    label: points[index].timestamp
  }))
})

const activePoint = computed(() => {
  if (activeIndex.value < 0) return null
  return chartPoints.value[activeIndex.value] || null
})

const secondActivePoint = computed(() => {
  if (activeIndex.value < 0 || !props.secondField) return null
  return secondChartPoints.value[activeIndex.value] || null
})

const tooltipStyle = computed(() => {
  if (!activePoint.value) return {}
  const positionX = Math.min(Math.max((activePoint.value.x / svgWidth) * 100, 18), 82)
  const positionY = Math.min(Math.max((activePoint.value.y / svgHeight.value) * 100 - 6, 10), 74)
  return {
    left: `${positionX}%`,
    top: `${positionY}%`
  }
})

const unitSuffix = computed(() => {
  return props.unit ? ` ${props.unit}` : ''
})

const headline = computed(() => props.value ?? '-')

const formatMetricValue = (value) => {
  if (value === null || value === undefined || Number.isNaN(value)) return '-'
  if (Math.abs(value) >= 1000) return Number(value).toLocaleString(undefined, { maximumFractionDigits: 2 })
  return Number(value).toFixed(2).replace(/\.?0+$/, '')
}

const formatTickValue = (value) => {
  if (value >= 1000000) return `${(value / 1000000).toFixed(1).replace(/\.0$/, '')}m`
  if (value >= 1000) return `${(value / 1000).toFixed(1).replace(/\.0$/, '')}k`
  return formatMetricValue(value)
}

const clearHover = () => {
  tooltipVisible.value = false
  activeIndex.value = -1
}

const handleHover = (event) => {
  if (!chartRef.value || !chartPoints.value.length) return
  const rect = chartRef.value.getBoundingClientRect()
  const relativeX = event.clientX - rect.left
  const ratio = rect.width > 0 ? relativeX / rect.width : 0
  const svgX = ratio * svgWidth

  const nearestIndex = chartPoints.value.reduce((bestIndex, point, index, points) => {
    if (bestIndex === -1) return index
    return Math.abs(point.x - svgX) < Math.abs(points[bestIndex].x - svgX) ? index : bestIndex
  }, -1)

  activeIndex.value = nearestIndex
  tooltipVisible.value = nearestIndex >= 0
}
</script>

<style scoped lang="scss">
.metric-trend-card {
  padding: 18px;
  border-radius: 18px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(255, 255, 255, 0.03);
}

.metric-trend-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
  align-items: flex-start;
}

.metric-trend-label {
  color: var(--text-secondary);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.metric-trend-value {
  margin-top: 8px;
  color: var(--text-primary);
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
  font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
}

.metric-trend-unit {
  font-size: 14px;
  color: var(--text-secondary);
  margin-left: 6px;
}

.metric-trend-meta {
  color: var(--text-secondary);
  font-size: 12px;
  text-align: right;
}

.metric-trend-actions {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.expand-btn {
  min-width: 54px;
  height: 30px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-primary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.expand-btn:hover {
  border-color: rgba(56, 189, 248, 0.4);
  color: #7dd3fc;
  background: rgba(56, 189, 248, 0.12);
}

.metric-trend-chart {
  position: relative;
}

.metric-trend-svg {
  width: 100%;
  display: block;
}

.axis-line {
  stroke: rgba(255, 255, 255, 0.12);
  stroke-width: 1;
}

.grid-line {
  stroke: rgba(255, 255, 255, 0.08);
  stroke-dasharray: 4 6;
}

.axis-text {
  fill: var(--text-secondary);
  font-size: 10px;
}

.axis-text-y {
  text-anchor: end;
}

.axis-text-x {
  text-anchor: middle;
}

.trend-line {
  fill: none;
  stroke-width: 3;
  stroke-linejoin: round;
  stroke-linecap: round;
}

.trend-line.second-line {
  stroke-width: 2;
  stroke-dasharray: 6 4;
}

.chart-overlay {
  fill: transparent;
  cursor: crosshair;
}

.hover-line {
  stroke: rgba(255, 255, 255, 0.24);
  stroke-dasharray: 4 4;
}

.active-dot {
  stroke-width: 2;
  filter: drop-shadow(0 0 10px rgba(56, 189, 248, 0.35));
}

.active-dot.second-active-dot {
  filter: drop-shadow(0 0 10px rgba(239, 68, 68, 0.35));
}

.metric-tooltip {
  position: absolute;
  z-index: 2;
  min-width: 148px;
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(9, 14, 26, 0.96);
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.28);
  transform: translate(-50%, -100%);
  pointer-events: none;
}

.metric-tooltip-time {
  color: var(--text-secondary);
  font-size: 11px;
  margin-bottom: 8px;
}

.metric-tooltip-row {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-primary);
  font-size: 12px;
}

.metric-tooltip-row strong {
  margin-left: auto;
  font-family: 'Consolas', 'Monaco', 'Fira Code', monospace;
}

.metric-tooltip-row.second-row {
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

.metric-tooltip-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

@media (max-width: 768px) {
  .metric-trend-actions {
    flex-direction: column;
    align-items: flex-end;
  }
}
</style>
