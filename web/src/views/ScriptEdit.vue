<template>
  <div class="script-edit-page" v-loading="loading">
    <!-- 顶部操作栏 -->
    <div class="section-card header-section">
      <div class="header-content">
        <div class="header-left">
          <button class="back-btn" @click="handleBack" title="返回">
            <el-icon><ArrowLeft /></el-icon>
          </button>
          <div class="script-info">
            <span class="script-name">{{ scriptInfo.name || '加载中...' }}</span>
            <span v-if="scriptInfo.id" class="script-id">ID: {{ scriptInfo.id }}</span>
          </div>
        </div>
        
        <!-- Tab 切换按钮组 -->
        <div class="tab-switcher">
          <button 
            class="tab-btn" 
            :class="{ active: editMode === 'visual' }"
            @click="switchToVisual"
          >
            <el-icon><Grid /></el-icon>
            <span>可视化编辑</span>
          </button>
          <button 
            class="tab-btn" 
            :class="{ active: editMode === 'xml' }"
            @click="switchToXml"
          >
            <el-icon><Document /></el-icon>
            <span>XML源码</span>
          </button>
        </div>

        <div class="history-actions">
          <el-tooltip content="撤销" placement="top">
            <el-button circle @click="undoChange" :disabled="!canUndo">
              <el-icon><RefreshLeft /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="重做" placement="top">
            <el-button circle @click="redoChange" :disabled="!canRedo">
              <el-icon><RefreshRight /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="版本历史" placement="top">
            <el-button circle @click="openVersionHistory">
              <el-icon><Clock /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tag
            size="small"
            :type="hasUnsavedChanges ? 'warning' : 'success'"
            effect="plain"
            class="change-status-tag"
          >
            {{ hasUnsavedChanges ? '有未保存修改' : '已同步' }}
          </el-tag>
        </div>
        
        <div class="header-actions">
          <el-button @click="handleBack">取消</el-button>
          <el-button @click="openSavePreview" :disabled="!hasUnsavedChanges">
            <el-icon class="btn-icon"><View /></el-icon>
            预览差异
          </el-button>
          <el-button type="primary" @click="openSavePreview" :loading="saving">
            <el-icon class="btn-icon"><Check /></el-icon>
            预览并保存
          </el-button>
        </div>
      </div>
    </div>

    <!-- 主体区域 -->
    <div class="main-area">
      <!-- 左侧编辑器区域 -->
      <div class="section-card editor-section">
        <!-- 可视化编辑模式 -->
        <div class="editor-wrapper" v-show="editMode === 'visual'">
          <JmxTreeEditor 
            ref="treeEditorRef"
            v-model="xmlContent"
            :uploaded-files="fileList"
          />
        </div>
        
        <!-- XML源码模式 -->
        <div class="editor-wrapper" v-show="editMode === 'xml'">
          <div v-if="monacoLoadingState" class="monaco-loading">
            <el-icon class="loading-icon"><Loading /></el-icon>
            <span>正在加载编辑器...</span>
          </div>
          <div ref="editorContainer" class="monaco-editor-container"></div>
        </div>
      </div>

      <!-- 右侧文件面板 -->
      <div class="section-card file-panel">
        <div class="section-header">
          <div class="section-label">FILES</div>
          <div class="section-title">关联文件</div>
        </div>

        <!-- 缺少引用文件警告 -->
        <div v-if="missingFiles.length > 0" class="missing-files-warning">
          <div class="warning-header">
            <el-icon><Warning /></el-icon>
            <span>缺少引用文件</span>
          </div>
          <div class="missing-files-list">
            <div v-for="filename in missingFiles" :key="filename" class="missing-file-item">
              <span class="missing-file-name" :title="filename">{{ filename }}</span>
              <el-button
                size="small"
                type="primary"
                link
                @click="triggerUploadForFile(filename)"
              >
                上传
              </el-button>
            </div>
          </div>
        </div>

        <!-- 脚本文件组 -->
        <div v-if="jmxFiles.length > 0" class="file-group">
          <div class="file-group-title">
            <el-icon><Document /></el-icon>
            <span>脚本文件</span>
            <el-tag size="small" type="info">{{ jmxFiles.length }}</el-tag>
          </div>
          <div class="file-list">
            <div
              v-for="(file, index) in jmxFiles"
              :key="file.id"
              class="file-item"
              :style="{ animationDelay: `${index * 0.05}s` }"
            >
              <div class="file-icon jmx-icon">
                <el-icon><Document /></el-icon>
              </div>
              <div class="file-details">
                <span class="file-name" :title="file.file_name">{{ file.file_name }}</span>
                <span class="file-time" v-if="file.created_at">{{ formatFileTime(file.created_at) }}</span>
                <div class="file-tags">
                  <el-tag size="small" type="primary">JMX</el-tag>
                  <el-tag v-if="isMainScript(file)" size="small" type="success">主脚本</el-tag>
                </div>
              </div>
              <div class="file-delete" @click="handleDeleteFile(file)">
                <el-icon><Delete /></el-icon>
              </div>
            </div>
          </div>
        </div>

        <!-- 数据文件组 -->
        <div v-if="dataFiles.length > 0" class="file-group">
          <div class="file-group-title">
            <el-icon><DataLine /></el-icon>
            <span>数据文件</span>
            <el-tag size="small" type="info">{{ dataFiles.length }}</el-tag>
          </div>
          <div class="file-list">
            <div
              v-for="(file, index) in dataFiles"
              :key="file.id"
              class="file-item"
              :class="{ 'is-referenced': isReferenced(file.file_name) }"
              :style="{ animationDelay: `${index * 0.05}s` }"
            >
              <div class="file-icon" :class="getFileIconClass(file.file_name)">
                <el-icon v-if="isCsvFile(file.file_name)"><DataLine /></el-icon>
                <el-icon v-else-if="isJsonFile(file.file_name)"><DocumentCopy /></el-icon>
                <el-icon v-else><DocumentCopy /></el-icon>
              </div>
              <div class="file-details">
                <span class="file-name" :title="file.file_name">{{ file.file_name }}</span>
                <span class="file-time" v-if="file.created_at">{{ formatFileTime(file.created_at) }}</span>
                <div class="file-tags">
                  <el-tag size="small" :type="getFileTagType(file.file_name)">{{ getFileTypeLabel(file.file_name) }}</el-tag>
                  <el-tag v-if="isReferenced(file.file_name)" size="small" type="success">已关联</el-tag>
                </div>
              </div>
              <div class="file-delete" @click="handleDeleteFile(file)">
                <el-icon><Delete /></el-icon>
              </div>
            </div>
          </div>
        </div>

        <!-- 空状态 -->
        <div v-if="fileList.length === 0" class="empty-files">
          <el-icon><FolderOpened /></el-icon>
          <span>暂无关联文件</span>
        </div>

        <!-- 上传按钮 -->
        <div class="upload-area">
          <el-upload
            ref="uploadRef"
            :show-file-list="false"
            :auto-upload="false"
            :on-change="handleFileSelect"
            :multiple="true"
            accept=".csv,.txt,.json,.properties,.xml,.yaml,.yml"
            class="upload-trigger"
          >
            <div class="upload-btn-area" :class="{ 'is-uploading': uploadingFile }">
              <el-icon><Plus /></el-icon>
              <span>上传数据文件</span>
            </div>
          </el-upload>
        </div>
      </div>
    </div>

    <!-- 文件上传确认弹窗 -->
    <el-dialog
      v-model="uploadDialogVisible"
      title="上传文件"
      width="500px"
      :close-on-click-modal="false"
      class="upload-dialog"
    >
      <div class="upload-preview">
        <p class="preview-title">即将上传以下文件：</p>
        <div class="preview-list">
          <div
            v-for="file in pendingFiles"
            :key="file.name"
            class="preview-item"
          >
            <div class="preview-icon">
              <el-icon><Document /></el-icon>
            </div>
            <span class="preview-name">{{ file.name }}</span>
            <span class="preview-size">{{ formatFileSize(file.size) }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="closeUploadDialog">取消</el-button>
        <el-button
          type="primary"
          @click="confirmUploadFiles"
          :loading="uploadingFile"
        >
          确认上传
        </el-button>
      </template>
    </el-dialog>

    <!-- 版本历史抽屉 -->
    <el-drawer
      v-model="versionDrawerVisible"
      title="版本历史"
      direction="rtl"
      size="420px"
      :destroy-on-close="false"
      class="version-drawer"
    >
      <div class="version-timeline" v-loading="versionsLoading">
        <div v-if="versions.length === 0" class="empty-versions">
          暂无版本记录
        </div>
        <div
          v-for="ver in versions"
          :key="ver.id"
          class="version-item"
          :class="{ active: selectedVersion?.id === ver.id }"
          @click="selectVersion(ver)"
        >
          <div class="version-header">
            <span class="version-number">v{{ ver.version_number }}</span>
            <span class="version-time">{{ formatFileTime(ver.created_at) }}</span>
          </div>
          <div class="version-summary">{{ ver.change_summary }}</div>
          <div class="version-actions" v-if="selectedVersion?.id === ver.id">
            <el-button size="small" @click.stop="previewVersion(ver)">
              预览内容
            </el-button>
            <el-button size="small" type="warning" @click.stop="confirmRestore(ver)">
              回滚到此版本
            </el-button>
          </div>
        </div>
      </div>
    </el-drawer>

    <!-- 版本预览弹窗 -->
    <el-dialog
      v-model="versionPreviewVisible"
      :title="`版本 ${previewingVersion?.version_number} 预览`"
      width="70%"
      top="5vh"
      class="version-preview-dialog"
    >
      <div class="version-preview-content" v-loading="previewLoading">
        <pre class="xml-preview">{{ previewContent }}</pre>
      </div>
      <template #footer>
        <el-button @click="versionPreviewVisible = false">关闭</el-button>
        <el-button type="warning" @click="confirmRestore(previewingVersion)">
          回滚到此版本
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="savePreviewVisible"
      title="保存前差异预览"
      width="min(1200px, 92vw)"
      top="4vh"
      :close-on-click-modal="false"
      class="diff-dialog"
    >
      <div class="diff-dialog-body">
        <div v-if="saveRiskReport.preflight || saveRiskReport.summary.length || saveRiskReport.blockingIssues.length || saveRiskReport.warnings.length" class="save-risk-panel">
          <div class="save-risk-header">
            <span class="save-risk-title">保存前风险检查</span>
            <div class="save-risk-chips" v-if="saveRiskReport.summary.length">
              <span v-for="item in saveRiskReport.summary" :key="item.label" class="save-risk-chip">
                {{ item.label }} {{ item.count }}
              </span>
            </div>
          </div>
          <div v-if="saveRiskReport.preflight" class="save-risk-overview" :class="`is-${saveRiskReport.preflight.level || 'success'}`">
            <div class="save-risk-overview-header">
              <div>
                <div class="save-risk-overview-row">
                  <span class="save-risk-overview-badge" :class="`is-${saveRiskReport.preflight.level || 'success'}`">
                    {{ saveRiskReport.preflight.level === 'danger' ? '高风险' : saveRiskReport.preflight.level === 'warning' ? '需关注' : '可保存' }}
                  </span>
                  <span class="save-risk-overview-title">结构体检</span>
                </div>
                <div class="save-risk-overview-summary">
                  当前主指标口径：{{ saveRiskReport.preflight.metricMode || '未知' }}
                </div>
              </div>
              <div class="save-risk-overview-score">
                <span class="save-risk-overview-score-label">健康分</span>
                <span class="save-risk-overview-score-value">{{ saveRiskReport.preflight.score }}</span>
              </div>
            </div>
            <div v-if="saveRiskReport.preflight.facts?.length" class="save-risk-facts-grid">
              <div v-for="fact in saveRiskReport.preflight.facts" :key="fact.label" class="save-risk-fact-card">
                <div class="save-risk-fact-label">{{ fact.label }}</div>
                <div class="save-risk-fact-value">{{ fact.value }}</div>
                <div class="save-risk-fact-detail">{{ fact.detail || '-' }}</div>
              </div>
            </div>
          </div>
          <div v-if="saveRiskReport.blockingIssues.length" class="save-risk-list is-blocking">
            <div v-for="item in saveRiskReport.blockingIssues" :key="item.title" class="save-risk-item">
              <strong>{{ item.title }}</strong>
              <span>{{ item.detail }}</span>
            </div>
          </div>
          <div v-if="saveRiskReport.warnings.length" class="save-risk-list is-warning">
            <div v-for="item in saveRiskReport.warnings" :key="item.title" class="save-risk-item">
              <strong>{{ item.title }}</strong>
              <span>{{ item.detail }}</span>
            </div>
          </div>
          <div v-if="saveRiskReport.preflight?.recommendations?.length" class="save-risk-list is-suggestion">
            <div v-for="item in saveRiskReport.preflight.recommendations" :key="item" class="save-risk-item">
              <strong>建议</strong>
              <span>{{ item }}</span>
            </div>
          </div>
        </div>
        <div class="diff-summary">
          <div class="diff-stat">
            <span class="diff-stat-label">原始行数</span>
            <span class="diff-stat-value">{{ diffStats.originalLines }}</span>
          </div>
          <div class="diff-stat">
            <span class="diff-stat-label">当前行数</span>
            <span class="diff-stat-value">{{ diffStats.modifiedLines }}</span>
          </div>
          <div class="diff-stat">
            <span class="diff-stat-label">变化行数</span>
            <span class="diff-stat-value diff-stat-highlight">{{ diffStats.changedLines }}</span>
          </div>
        </div>
        <div v-if="previewContent === originalContent" class="diff-empty">
          当前内容和已保存版本一致，没有需要保存的差异。
        </div>
        <div v-else ref="diffContainer" class="diff-editor-container"></div>
      </div>
      <template #footer>
        <el-button @click="savePreviewVisible = false">继续编辑</el-button>
        <el-button type="primary" @click="confirmSave" :loading="saving">
          确认保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  Check,
  Document,
  DataLine,
  DocumentCopy,
  Delete,
  Plus,
  FolderOpened,
  Grid,
  Warning,
  View,
  RefreshLeft,
  RefreshRight,
  Loading,
  Clock
} from '@element-plus/icons-vue'
import { scriptApi } from '@/api/script'
import JmxTreeEditor from '@/components/JmxTreeEditor.vue'
import { extractCSVDataSetFilesFromXML, parseJMX } from '@/utils/jmxParser.js'
import { analyzeJmxSaveRisks } from '@/utils/jmxRisk'
import { formatDateTimeInShanghai } from '@/utils/datetime'

// Monaco Editor 动态导入
let monaco = null
let monacoLoading = false
let monacoLoadPromise = null

const loadMonaco = async () => {
  if (monaco) return monaco
  if (monacoLoadPromise) return monacoLoadPromise

  monacoLoading = true
  monacoLoadPromise = import('monaco-editor').then((module) => {
    monaco = module
    monacoLoading = false
    return monaco
  })
  return monacoLoadPromise
}

const route = useRoute()
const router = useRouter()
const scriptId = route.params.id

// 编辑模式：'visual' 或 'xml'
const editMode = ref('visual')

// 脚本信息
const scriptInfo = ref({})
const fileList = ref([])
const loading = ref(false)
const saving = ref(false)
const uploadingFile = ref(false)

// XML 内容（双模式共享）
const xmlContent = ref('')
const originalContent = ref('')
const previewContent = ref('')

// 编辑器引用
const treeEditorRef = ref(null)
const editorContainer = ref(null)
const diffContainer = ref(null)
let editor = null
let diffEditor = null
let diffOriginalModel = null
let diffModifiedModel = null

// Monaco 加载状态
const monacoLoadingState = ref(false)

// 文件上传
const uploadDialogVisible = ref(false)
const pendingFiles = ref([])
const uploadRef = ref(null)
const targetUploadFilename = ref('')
const savePreviewVisible = ref(false)
const saveRiskReport = ref({
  summary: [],
  blockingIssues: [],
  warnings: [],
  preflight: null
})

// 版本管理相关状态
const versionDrawerVisible = ref(false)
const versions = ref([])
const versionsLoading = ref(false)
const selectedVersion = ref(null)
const versionPreviewVisible = ref(false)
const previewingVersion = ref(null)
const previewLoading = ref(false)

// 编辑历史
const historyStack = ref([])
const historyIndex = ref(-1)
const historyReady = ref(false)
const applyingHistory = ref(false)
let historyTimer = null

const HISTORY_LIMIT = 100
const HISTORY_DEBOUNCE_MS = 350

// 计算属性：脚本文件列表
const jmxFiles = computed(() => {
  return fileList.value.filter(f => isJmxFile(f.file_name))
})

// 计算属性：数据文件列表
const dataFiles = computed(() => {
  return fileList.value.filter(f => !isJmxFile(f.file_name))
})

// 计算属性：JMX 中引用的所有文件名
const referencedFiles = computed(() => {
  return extractCSVDataSetFilesFromXML(xmlContent.value)
})

// 计算属性：缺失的引用文件
const missingFiles = computed(() => {
  const uploadedFilenames = fileList.value.map(f => f.file_name)
  return referencedFiles.value.filter(filename => !uploadedFilenames.includes(filename))
})

// 检查文件是否被引用
const isReferenced = (filename) => {
  return referencedFiles.value.includes(filename)
}

// 检查是否为主脚本文件
const isMainScript = (file) => {
  return scriptInfo.value.file_path && file.file_path === scriptInfo.value.file_path
}

const canUndo = computed(() => historyIndex.value > 0)
const canRedo = computed(() => historyIndex.value >= 0 && historyIndex.value < historyStack.value.length - 1)

const diffStats = computed(() => {
  const originalLines = (originalContent.value || '').split('\n')
  const modifiedLines = (previewContent.value || getCurrentContent()).split('\n')
  const maxLength = Math.max(originalLines.length, modifiedLines.length)
  let changedLines = 0

  for (let index = 0; index < maxLength; index += 1) {
    if (originalLines[index] !== modifiedLines[index]) {
      changedLines += 1
    }
  }

  return {
    originalLines: originalContent.value ? originalLines.length : 0,
    modifiedLines: previewContent.value || getCurrentContent() ? modifiedLines.length : 0,
    changedLines
  }
})

const clearHistoryTimer = () => {
  if (historyTimer) {
    clearTimeout(historyTimer)
    historyTimer = null
  }
}

const getCurrentContent = () => {
  if (editMode.value === 'xml' && editor) {
    return editor.getValue()
  }
  return xmlContent.value
}

const syncEditorContent = (content) => {
  if (editor && editor.getValue() !== content) {
    editor.setValue(content)
  }
}

const resetHistory = (content) => {
  clearHistoryTimer()
  historyStack.value = [content]
  historyIndex.value = 0
  historyReady.value = true
}

const commitHistorySnapshot = (content) => {
  if (!historyReady.value || applyingHistory.value) return
  if (historyIndex.value >= 0 && historyStack.value[historyIndex.value] === content) return

  const previousContent = historyStack.value[historyIndex.value - 1]
  if (historyIndex.value > 0 && previousContent === content) {
    historyIndex.value -= 1
    return
  }

  const nextContent = historyStack.value[historyIndex.value + 1]
  if (historyIndex.value < historyStack.value.length - 1 && nextContent === content) {
    historyIndex.value += 1
    return
  }

  const nextHistory = historyStack.value.slice(0, historyIndex.value + 1)
  nextHistory.push(content)

  if (nextHistory.length > HISTORY_LIMIT) {
    nextHistory.shift()
  }

  historyStack.value = nextHistory
  historyIndex.value = nextHistory.length - 1
}

const scheduleHistorySnapshot = (content, immediate = false) => {
  if (!historyReady.value || applyingHistory.value) return
  clearHistoryTimer()

  if (immediate) {
    commitHistorySnapshot(content)
    return
  }

  historyTimer = setTimeout(() => {
    commitHistorySnapshot(content)
    historyTimer = null
  }, HISTORY_DEBOUNCE_MS)
}

const flushPendingHistory = () => {
  if (!historyReady.value || applyingHistory.value || historyIndex.value < 0) return false

  const currentContent = getCurrentContent()
  if (historyStack.value[historyIndex.value] === currentContent) {
    clearHistoryTimer()
    return false
  }

  clearHistoryTimer()
  commitHistorySnapshot(currentContent)
  return true
}

const applyHistoryContent = (content) => {
  applyingHistory.value = true
  clearHistoryTimer()
  xmlContent.value = content
  syncEditorContent(content)
  window.setTimeout(() => {
    applyingHistory.value = false
  }, 0)
}

const undoChange = () => {
  flushPendingHistory()
  if (!canUndo.value) return
  historyIndex.value -= 1
  applyHistoryContent(historyStack.value[historyIndex.value])
}

const redoChange = () => {
  if (flushPendingHistory()) return
  if (!canRedo.value) return
  historyIndex.value += 1
  applyHistoryContent(historyStack.value[historyIndex.value])
}

const ensureValidCurrentContent = () => {
  const currentContent = getCurrentContent()

  if (!currentContent) {
    ElMessage.warning('内容为空，无法继续')
    return null
  }

  try {
    parseJMX(currentContent)
  } catch (error) {
    ElMessage.error('JMX 格式有误，请先修复 XML 再继续')
    return null
  }

  xmlContent.value = currentContent
  scheduleHistorySnapshot(currentContent, true)
  return currentContent
}

// 获取脚本详情
const fetchScriptDetail = async () => {
  loading.value = true
  try {
    const res = await scriptApi.getDetail(scriptId)
    scriptInfo.value = res.data?.script || {}
    fileList.value = res.data?.files || []
  } catch (error) {
    console.error('获取脚本详情失败:', error)
    ElMessage.error('获取脚本详情失败')
  } finally {
    loading.value = false
  }
}

// 获取脚本内容
const fetchScriptContent = async () => {
  try {
    const res = await scriptApi.getContent(scriptId)
    const content = res.data?.content || ''
    historyReady.value = false
    xmlContent.value = content
    originalContent.value = content

    // 只有在 XML 模式下才初始化 Monaco Editor
    if (editMode.value === 'xml') {
      await initEditor(content)
    }
    resetHistory(content)
  } catch (error) {
    console.error('获取脚本内容失败:', error)
    ElMessage.error('获取脚本内容失败')
    resetHistory('')
  }
}

// 初始化 Monaco Editor
const initEditor = async (content) => {
  if (!editorContainer.value) return
  if (editor) {
    editor.setValue(content || '')
    return
  }

  // 动态加载 Monaco
  monacoLoadingState.value = true
  try {
    const monacoModule = await loadMonaco()

    editor = monacoModule.editor.create(editorContainer.value, {
      value: content,
      language: 'xml',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: true },
      fontSize: 14,
      wordWrap: 'on',
      scrollBeyondLastLine: false,
      lineNumbers: 'on',
      roundedSelection: false,
      scrollbar: {
        useShadows: false,
        verticalHasArrows: true,
        horizontalHasArrows: true,
        vertical: 'auto',
        horizontal: 'auto'
      }
    })

    editor.onDidChangeModelContent(() => {
      xmlContent.value = editor?.getValue() || ''
    })
  } finally {
    monacoLoadingState.value = false
  }
}

const initDiffEditor = async () => {
  if (!diffContainer.value) return

  // 动态加载 Monaco
  const monacoModule = await loadMonaco()

  if (!diffEditor) {
    diffEditor = monacoModule.editor.createDiffEditor(diffContainer.value, {
      theme: 'vs-dark',
      readOnly: true,
      automaticLayout: true,
      renderSideBySide: false,
      originalEditable: false,
      scrollBeyondLastLine: false,
      minimap: { enabled: false },
      wordWrap: 'on'
    })
  }

  diffOriginalModel?.dispose()
  diffModifiedModel?.dispose()

  diffOriginalModel = monacoModule.editor.createModel(originalContent.value || '', 'xml')
  diffModifiedModel = monacoModule.editor.createModel(previewContent.value || '', 'xml')
  diffEditor.setModel({
    original: diffOriginalModel,
    modified: diffModifiedModel
  })
}

watch(xmlContent, (value) => {
  if (!historyReady.value || applyingHistory.value) return
  scheduleHistorySnapshot(value, editMode.value === 'visual')
})

watch(savePreviewVisible, (visible) => {
  if (!visible) return
  nextTick(() => {
    initDiffEditor()
  })
})

// 切换到可视化模式
const switchToVisual = () => {
  if (editMode.value === 'visual') return
  
  // 从 Monaco Editor 同步内容到 JmxTreeEditor
  if (editor) {
    const latestXml = editor.getValue()

    try {
      parseJMX(latestXml)
      xmlContent.value = latestXml
      scheduleHistorySnapshot(latestXml, true)
    } catch (error) {
      ElMessage.error('XML 解析失败，请先修复后再切换到可视化编辑')
      return
    }
  }
  
  editMode.value = 'visual'
}

// 切换到 XML 模式
const switchToXml = async () => {
  if (editMode.value === 'xml') return

  // 先切换到 XML 模式，显示加载状态
  editMode.value = 'xml'

  // 等待 DOM 更新
  await nextTick()

  // 动态加载 Monaco 并初始化编辑器
  await initEditor(xmlContent.value || '')
}

const openSavePreview = async () => {
  const contentToSave = ensureValidCurrentContent()
  if (!contentToSave) return
  if (contentToSave === originalContent.value) {
    ElMessage.info('当前没有需要保存的变更')
    return
  }

  previewContent.value = contentToSave
  saveRiskReport.value = analyzeJmxSaveRisks(contentToSave, fileList.value)
  savePreviewVisible.value = true
}

const confirmSave = async () => {
  const contentToSave = previewContent.value || ensureValidCurrentContent()
  if (!contentToSave) return

  saving.value = true
  try {
    await scriptApi.saveContent(scriptId, contentToSave)
    ElMessage.success('保存成功')
    originalContent.value = contentToSave
    previewContent.value = contentToSave
    savePreviewVisible.value = false
    scheduleHistorySnapshot(contentToSave, true)
  } catch (error) {
    console.error('保存失败:', error)
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

// 返回
const handleBack = () => {
  router.push('/scripts')
}

// 判断文件类型
const isJmxFile = (filename) => {
  return filename.toLowerCase().endsWith('.jmx')
}

const isCsvFile = (filename) => {
  return filename.toLowerCase().endsWith('.csv')
}

const isJsonFile = (filename) => {
  return filename.toLowerCase().endsWith('.json')
}

const getFileIconClass = (filename) => {
  if (isJmxFile(filename)) return 'jmx-icon'
  if (isCsvFile(filename)) return 'csv-icon'
  return 'other-icon'
}

const getFileTagType = (filename) => {
  const ext = filename.toLowerCase().split('.').pop()
  switch (ext) {
    case 'csv': return 'success'
    case 'json': return 'warning'
    case 'txt': return 'info'
    case 'properties': return ''
    default: return 'info'
  }
}

const getFileTypeLabel = (filename) => {
  const ext = filename.toLowerCase().split('.').pop()
  switch (ext) {
    case 'csv': return 'CSV'
    case 'json': return 'JSON'
    case 'txt': return 'TXT'
    case 'properties': return 'PROPERTIES'
    case 'xml': return 'XML'
    case 'yaml':
    case 'yml': return 'YAML'
    default: return ext.toUpperCase()
  }
}

const formatFileSize = (size) => {
  if (!size && size !== 0) return ''
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB'
  return (size / 1024 / 1024).toFixed(1) + ' MB'
}

const formatFileTime = (time) => {
  if (!time) return ''
  try {
    return formatDateTimeInShanghai(time)
  } catch {
    return time
  }
}

// 选择文件
const handleFileSelect = (uploadFile) => {
  // 如果指定了目标文件名，重命名文件
  if (targetUploadFilename.value && uploadFile.raw) {
    const renamedFile = new File([uploadFile.raw], targetUploadFilename.value, {
      type: uploadFile.raw.type
    })
    pendingFiles.value.push(renamedFile)
    targetUploadFilename.value = '' // 重置
  } else {
    pendingFiles.value.push(uploadFile.raw)
  }
  uploadDialogVisible.value = true
}

// 触发上传特定文件
const triggerUploadForFile = (filename) => {
  targetUploadFilename.value = filename
  // 触发文件选择
  const input = uploadRef.value?.$el.querySelector('input[type="file"]')
  if (input) {
    input.click()
  }
}

// 确认上传文件
const confirmUploadFiles = async () => {
  uploadingFile.value = true
  try {
    const uploadPromises = pendingFiles.value.map(file =>
      scriptApi.uploadFile(scriptId, file)
    )
    await Promise.all(uploadPromises)

    ElMessage.success('文件上传成功')
    closeUploadDialog()
    // 刷新文件列表
    await fetchScriptDetail()
  } catch (error) {
    console.error('上传失败:', error)
    ElMessage.error('上传失败')
  } finally {
    uploadingFile.value = false
  }
}

const closeUploadDialog = () => {
  uploadDialogVisible.value = false
  pendingFiles.value = []
  targetUploadFilename.value = ''
  uploadRef.value?.clearFiles?.()
}

// 删除文件
const handleDeleteFile = async (file) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除文件 "${file.file_name}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await scriptApi.deleteFile(scriptId, file.id)
    ElMessage.success('删除成功')
    await fetchScriptDetail()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除失败:', error)
      ElMessage.error('删除失败')
    }
  }
}

// 检测是否有修改
const hasUnsavedChanges = computed(() => {
  return xmlContent.value !== originalContent.value
})

// 路由守卫 - 离开页面前检查是否有未保存的修改
onBeforeRouteLeave((to, from, next) => {
  if (hasUnsavedChanges.value) {
    ElMessageBox.confirm('有未保存的修改，确定离开吗？', '提示', {
      confirmButtonText: '离开',
      cancelButtonText: '继续编辑',
      type: 'warning'
    }).then(() => next()).catch(() => next(false))
  } else {
    next()
  }
})

// 浏览器关闭提示
const handleBeforeUnload = (e) => {
  if (hasUnsavedChanges.value) {
    e.preventDefault()
    e.returnValue = ''
  }
}

const handleKeydown = (event) => {
  const withModifier = event.metaKey || event.ctrlKey
  if (!withModifier) return

  const key = event.key.toLowerCase()
  const target = event.target
  const inTextEditor = target instanceof HTMLElement && (
    target.tagName === 'INPUT' ||
    target.tagName === 'TEXTAREA' ||
    target.isContentEditable ||
    Boolean(target.closest('.monaco-editor'))
  )

  if (key === 'z' && !event.shiftKey && !inTextEditor) {
    event.preventDefault()
    undoChange()
    return
  }

  if (((key === 'z' && event.shiftKey) || key === 'y') && !inTextEditor) {
    event.preventDefault()
    redoChange()
    return
  }

  if (key === 's') {
    event.preventDefault()
    openSavePreview()
  }
}

// 加载版本列表
const loadVersions = async () => {
  versionsLoading.value = true
  try {
    const res = await scriptApi.getVersions(scriptId)
    versions.value = res.data || []
  } catch (err) {
    console.error('加载版本列表失败', err)
    ElMessage.error('加载版本列表失败')
  } finally {
    versionsLoading.value = false
  }
}

// 打开版本历史
const openVersionHistory = () => {
  versionDrawerVisible.value = true
  selectedVersion.value = null
  loadVersions()
}

// 选中版本
const selectVersion = (ver) => {
  selectedVersion.value = selectedVersion.value?.id === ver.id ? null : ver
}

// 预览版本
const previewVersion = async (ver) => {
  previewingVersion.value = ver
  previewLoading.value = true
  versionPreviewVisible.value = true
  try {
    const res = await scriptApi.getVersionContent(scriptId, ver.id)
    previewContent.value = res.data?.content || ''
  } catch (err) {
    previewContent.value = '加载失败'
    ElMessage.error('加载版本内容失败')
  } finally {
    previewLoading.value = false
  }
}

// 确认回滚
const confirmRestore = (ver) => {
  if (!ver) return
  ElMessageBox.confirm(
    `确定要回滚到版本 ${ver.version_number} 吗？当前未保存的更改将丢失。`,
    '确认回滚',
    {
      confirmButtonText: '确认回滚',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await scriptApi.restoreVersion(scriptId, ver.id)
      ElMessage.success(`已回滚到版本 ${ver.version_number}`)
      // 重新加载脚本内容
      await fetchScriptContent()
      // 刷新版本列表
      await loadVersions()
      // 关闭预览弹窗和抽屉
      versionPreviewVisible.value = false
      versionDrawerVisible.value = false
    } catch (err) {
      ElMessage.error('回滚失败: ' + (err.message || '未知错误'))
    }
  }).catch(() => {})
}

onMounted(async () => {
  await fetchScriptDetail()
  await nextTick()
  await fetchScriptContent()
  window.addEventListener('beforeunload', handleBeforeUnload)
  window.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  clearHistoryTimer()
  if (editor) {
    editor.dispose()
    editor = null
  }
  diffOriginalModel?.dispose()
  diffModifiedModel?.dispose()
  if (diffEditor) {
    diffEditor.dispose()
    diffEditor = null
  }
  window.removeEventListener('beforeunload', handleBeforeUnload)
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped lang="scss">
.script-edit-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: var(--bg-primary);
  overflow: hidden;
  padding: 4px 0 10px;
  gap: 12px;
}

// 区域卡片
.section-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid rgba(255, 255, 255, 0.06);
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
  margin-bottom: 12px;
}

.section-header {
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  margin-bottom: 12px;
}

// 顶部操作栏
.header-section {
  padding: 12px 16px;
  flex-shrink: 0;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.back-btn {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  background-color: var(--bg-secondary);
  border: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.25s ease;

  .el-icon {
    font-size: 18px;
    color: var(--text-secondary);
    transition: all 0.25s ease;
  }

  &:hover {
    background-color: var(--bg-hover);

    .el-icon {
      color: var(--text-primary);
    }
  }
}

.script-info {
  display: flex;
  align-items: center;
  gap: 12px;

  .script-name {
    font-size: 18px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .script-id {
    font-size: 12px;
    color: var(--text-secondary);
    background-color: var(--bg-secondary);
    padding: 2px 8px;
    border-radius: var(--radius-sm);
  }
}

// Tab 切换按钮组
.tab-switcher {
  display: flex;
  gap: 0;
  background: var(--bg-secondary);
  border-radius: 24px;
  padding: 4px;
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 16px;
  border-radius: 20px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.25s ease;

  .el-icon {
    font-size: 16px;
  }

  &:hover:not(.active) {
    color: var(--text-primary);
    background: rgba(255, 255, 255, 0.05);
  }

  &.active {
    background: var(--accent-blue);
    color: #fff;
    border-color: var(--accent-blue);
  }
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;

  .btn-icon {
    margin-right: 6px;
  }

  :deep(.el-button) {
    margin-left: 0;
  }
}

.history-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--radius-full);
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);

  :deep(.el-button) {
    margin-left: 0;
  }
}

.change-status-tag {
  margin-left: 4px;
}

// 主体区域
.main-area {
  display: grid;
  grid-template-columns: minmax(0, 1fr) clamp(220px, 13vw, 268px);
  flex: 1;
  overflow: hidden;
  gap: 12px;
  min-height: 0;
  align-items: stretch;
}

// 左侧编辑器
.editor-section {
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
  min-width: 0;
  
  // 移除内边距，让编辑器占满
  background: transparent;
  border: none;
}

.editor-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
  position: relative;

  // 让 JmxTreeEditor 和 Monaco 都能撑满
  & > * {
    flex: 1;
    min-height: 0;
  }
}

.monaco-editor-container {
  flex: 1;
  min-height: 300px;
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--bg-card);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.monaco-loading {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  z-index: 10;
  color: var(--text-secondary);
  font-size: 14px;

  .loading-icon {
    font-size: 32px;
    animation: spin 1s linear infinite;
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

// 右侧文件面板
.file-panel {
  width: clamp(220px, 13vw, 268px);
  max-width: clamp(220px, 13vw, 268px);
  min-width: clamp(220px, 13vw, 268px);
  display: flex;
  flex-direction: column;
  padding: 14px;
  margin-bottom: 0;
}

.file-list {
  flex: 1;
  overflow-y: auto;
  padding: 0 0 12px 0;
  min-height: 0;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  margin-bottom: 8px;
  background-color: var(--bg-secondary);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
  position: relative;

  &:hover {
    background-color: rgba(255, 255, 255, 0.02);

    .file-delete {
      opacity: 1;
    }
  }

  .file-icon {
    width: 34px;
    height: 34px;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;

    .el-icon {
      font-size: 18px;
    }

    &.jmx-icon {
      background-color: rgba(0, 102, 255, 0.1);
      color: var(--accent-blue);
    }

    &.csv-icon {
      background-color: rgba(0, 204, 106, 0.1);
      color: var(--accent-green);
    }

    &.other-icon {
      background-color: rgba(148, 163, 184, 0.1);
      color: var(--text-secondary);
    }
  }

  .file-details {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;

    .file-name {
      font-size: 13px;
      color: var(--text-primary);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .file-type {
      font-size: 11px;
      color: var(--text-secondary);
    }

    .file-time {
      font-size: 11px;
      color: #666;
      margin-top: 2px;
    }
  }

  .file-delete {
    width: 28px;
    height: 28px;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    opacity: 0;
    transition: all 0.25s ease;
    color: var(--text-secondary);

    &:hover {
      background-color: rgba(255, 69, 58, 0.1);
      color: var(--accent-red);
    }

    .el-icon {
      font-size: 16px;
    }
  }
}

.empty-files {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px 18px;
  color: var(--text-secondary);

  .el-icon {
    font-size: 48px;
    margin-bottom: 12px;
    opacity: 0.5;
  }

  span {
    font-size: 13px;
  }
}

.upload-area {
  padding-top: 14px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.upload-trigger {
  width: 100%;

  :deep(.el-upload) {
    width: 100%;
  }
}

.upload-btn-area {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 14px;
  border: 2px dashed rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-md);
  cursor: pointer;
  color: var(--text-secondary);

  .el-icon {
    font-size: 18px;
  }

  span {
    font-size: 14px;
  }

  &:hover {
    border-color: rgba(255, 255, 255, 0.2);
    color: var(--text-primary);
  }

  &.is-uploading {
    opacity: 0.6;
    pointer-events: none;
  }
}

// 文件分组
.file-group {
  margin-bottom: 16px;

  .file-group-title {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;
    font-size: 13px;
    color: var(--text-secondary);
    font-weight: 500;

    .el-icon {
      font-size: 16px;
    }

    .el-tag {
      margin-left: 4px;
    }
  }
}

// 缺失文件警告
.missing-files-warning {
  background: rgba(255, 193, 7, 0.1);
  border: 1px solid rgba(255, 193, 7, 0.3);
  border-radius: var(--radius-md);
  padding: 12px 14px;
  margin-bottom: 14px;

  .warning-header {
    display: flex;
    align-items: center;
    gap: 8px;
    color: #ffc107;
    font-size: 13px;
    font-weight: 500;
    margin-bottom: 8px;

    .el-icon {
      font-size: 16px;
    }
  }

  .missing-files-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .missing-file-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 10px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: var(--radius-sm);

    .missing-file-name {
      font-size: 12px;
      color: var(--text-primary);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      max-width: 160px;
    }
  }
}

// 文件标签
.file-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  margin-top: 4px;
}

// 已引用文件高亮
.file-item.is-referenced {
  border-color: rgba(0, 204, 106, 0.3);
  background: rgba(0, 204, 106, 0.05);
}

// 上传预览弹窗
.upload-dialog {
  .upload-preview {
    .preview-title {
      font-size: 14px;
      color: var(--text-secondary);
      margin-bottom: 12px;
    }

    .preview-list {
      max-height: 200px;
      overflow-y: auto;
      border: 1px solid rgba(255, 255, 255, 0.06);
      border-radius: var(--radius-md);
      padding: 12px;
      background-color: var(--bg-secondary);
    }

    .preview-item {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 10px;
      border-bottom: 1px solid rgba(255, 255, 255, 0.04);

      &:last-child {
        border-bottom: none;
      }

      &:hover {
        background-color: rgba(255, 255, 255, 0.02);
        border-radius: 6px;
      }

      .preview-icon {
        width: 32px;
        height: 32px;
        border-radius: var(--radius-sm);
        background-color: rgba(0, 102, 255, 0.1);
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--accent-blue);

        .el-icon {
          font-size: 16px;
        }
      }

      .preview-name {
        flex: 1;
        font-size: 13px;
        color: var(--text-primary);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .preview-size {
        font-size: 12px;
        color: var(--text-secondary);
      }
    }
  }
}

.diff-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.save-risk-panel {
  padding: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(255, 255, 255, 0.03);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.save-risk-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.save-risk-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.save-risk-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.save-risk-chip {
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(0, 153, 255, 0.12);
  color: var(--accent-blue);
  font-size: 12px;
  font-weight: 600;
}

.save-risk-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.save-risk-overview {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-radius: 16px;
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.92), rgba(24, 34, 52, 0.82));
  border: 1px solid rgba(34, 197, 94, 0.16);

  &.is-warning {
    border-color: rgba(255, 184, 0, 0.2);
  }

  &.is-danger {
    border-color: rgba(255, 92, 92, 0.24);
  }
}

.save-risk-overview-header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
}

.save-risk-overview-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}

.save-risk-overview-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(34, 197, 94, 0.16);
  color: #4ade80;
  font-size: 11px;
  font-weight: 700;

  &.is-warning {
    background: rgba(255, 184, 0, 0.16);
    color: #fbbf24;
  }

  &.is-danger {
    background: rgba(255, 92, 92, 0.16);
    color: #f87171;
  }
}

.save-risk-overview-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
}

.save-risk-overview-summary {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.save-risk-overview-score {
  min-width: 90px;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.save-risk-overview-score-label {
  font-size: 11px;
  color: var(--text-secondary);
}

.save-risk-overview-score-value {
  font-size: 30px;
  line-height: 1;
  font-weight: 800;
  color: var(--text-primary);
}

.save-risk-facts-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.save-risk-fact-card {
  padding: 12px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.save-risk-fact-label {
  color: var(--text-secondary);
  font-size: 11px;
}

.save-risk-fact-value {
  margin-top: 8px;
  color: var(--text-primary);
  font-size: 16px;
  font-weight: 700;
}

.save-risk-fact-detail {
  margin-top: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.save-risk-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border-radius: var(--radius-md);
}

.save-risk-item strong {
  color: var(--text-primary);
  font-size: 13px;
}

.save-risk-item span {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.save-risk-list.is-blocking .save-risk-item {
  border: 1px solid rgba(255, 92, 92, 0.35);
  background: rgba(255, 92, 92, 0.08);
}

.save-risk-list.is-warning .save-risk-item {
  border: 1px solid rgba(255, 184, 0, 0.24);
  background: rgba(255, 184, 0, 0.06);
}

.save-risk-list.is-suggestion .save-risk-item {
  border: 1px solid rgba(0, 153, 255, 0.18);
  background: rgba(0, 153, 255, 0.06);
}

.diff-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.diff-stat {
  padding: 12px 14px;
  border-radius: var(--radius-md);
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(255, 255, 255, 0.03);
}

.diff-stat-label {
  display: block;
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.diff-stat-value {
  display: block;
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 600;
}

.diff-stat-highlight {
  color: var(--accent-blue);
}

.diff-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 240px;
  border-radius: var(--radius-lg);
  border: 1px dashed rgba(255, 255, 255, 0.08);
  color: var(--text-secondary);
  background: rgba(255, 255, 255, 0.02);
}

.diff-editor-container {
  height: min(68vh, 720px);
  border-radius: var(--radius-lg);
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.06);
}

@media (max-width: 1280px) {
  .main-area {
    grid-template-columns: 1fr;
  }

  .file-panel {
    width: auto;
    max-width: none;
    min-width: 0;
  }
}

// 版本历史抽屉样式
.version-drawer {
  :deep(.el-drawer__header) {
    margin-bottom: 0;
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    color: var(--text-primary);
    font-weight: 600;
  }

  :deep(.el-drawer__body) {
    padding: 16px;
    background: var(--bg-primary);
  }
}

.version-timeline {
  padding: 0 4px;
}

.version-item {
  padding: 12px 16px;
  margin-bottom: 8px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;

  &:hover {
    background: rgba(255, 255, 255, 0.08);
  }

  &.active {
    border-color: rgba(56, 189, 248, 0.5);
    background: rgba(56, 189, 248, 0.08);
  }
}

.version-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.version-number {
  font-weight: 600;
  color: #38bdf8;
  font-size: 14px;
}

.version-time {
  color: rgba(255, 255, 255, 0.5);
  font-size: 12px;
}

.version-summary {
  color: rgba(255, 255, 255, 0.7);
  font-size: 13px;
  margin-bottom: 8px;
}

.version-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}

.empty-versions {
  text-align: center;
  color: rgba(255, 255, 255, 0.4);
  padding: 40px 0;
}

// 版本预览弹窗样式
.version-preview-dialog {
  :deep(.el-dialog__body) {
    padding: 16px 20px;
    background: var(--bg-card);
  }
}

.version-preview-content {
  min-height: 200px;
}

.xml-preview {
  background: rgba(0, 0, 0, 0.3);
  border-radius: 8px;
  padding: 16px;
  max-height: 60vh;
  overflow: auto;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.5;
  color: rgba(255, 255, 255, 0.85);
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}

@media (max-width: 900px) {
  .script-edit-page {
    padding: 8px;
    gap: 12px;
  }

  .header-left,
  .script-info,
  .history-actions,
  .header-actions {
    width: 100%;
  }

  .history-actions {
    justify-content: flex-start;
  }

  .tab-switcher {
    width: 100%;
    justify-content: space-between;
  }

  .tab-btn {
    flex: 1;
    justify-content: center;
  }

  .file-item .file-delete {
    opacity: 1;
  }

  .diff-summary {
    grid-template-columns: 1fr;
  }

  .diff-editor-container {
    height: 56vh;
  }
}
</style>
