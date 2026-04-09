<template>
  <div class="script-list-page">
    <!-- 统计概览卡片 -->
    <div class="stats-overview">
      <div class="stat-card" v-loading="statsLoading">
        <div class="stat-label">总脚本数</div>
        <div class="stat-value">{{ stats.totalScripts }}</div>
      </div>
      <div class="stat-card" v-loading="statsLoading">
        <div class="stat-label">总文件数</div>
        <div class="stat-value">{{ stats.totalFiles }}</div>
      </div>
      <div class="stat-card" v-loading="statsLoading">
        <div class="stat-label">运行中</div>
        <div class="stat-value">{{ stats.runningCount }}</div>
      </div>
      <div class="stat-card" v-loading="statsLoading">
        <div class="stat-label">执行记录数</div>
        <div class="stat-value">{{ stats.executionCount }}</div>
      </div>
    </div>

    <!-- 上传区域 -->
    <div class="section-card upload-section">
      <div class="upload-layout">
        <div class="upload-panel-copy">
          <div class="section-label">UPLOAD</div>
          <div class="section-title">上传脚本</div>
          <div class="section-desc">支持拖拽或点选 .jmx 文件，数据文件仍在脚本编辑页中关联。</div>
        </div>

        <el-form
          ref="uploadFormRef"
          :model="uploadForm"
          :rules="uploadRules"
          label-position="top"
          class="upload-form upload-workbench"
        >
          <div class="upload-desc-row">
            <el-form-item label="描述" class="desc-input-item">
              <el-input
                v-model="uploadForm.description"
                placeholder="补充脚本用途、环境或数据说明"
                maxlength="500"
                show-word-limit
              />
            </el-form-item>
          </div>
          <div class="upload-file-row">
            <el-form-item label="选择文件" prop="files" class="file-input-item">
              <FileUpload
                v-model:fileList="uploadForm.files"
                accept=".jmx"
                :multiple="false"
                :limit="1"
                :compact="true"
                :show-file-list="false"
                :single-tile="true"
              />
            </el-form-item>
            <div class="upload-action-slot">
              <el-button
                type="primary"
                @click="handleUploadSubmit"
                :loading="uploadLoading"
                :disabled="uploadLoading || uploadForm.files.length !== 1"
                class="upload-submit-btn"
              >
                <el-icon class="btn-icon"><upload /></el-icon>
                上传脚本
              </el-button>
            </div>
          </div>
        </el-form>
      </div>
    </div>

    <!-- 脚本列表区域 -->
    <div class="section-card scripts-section">
      <div class="section-header-with-action">
        <div class="section-header">
          <div class="section-label">SCRIPTS</div>
          <div class="section-title">脚本列表</div>
          <div class="section-desc">管理和执行 JMeter 性能测试脚本</div>
        </div>
        <div class="section-actions">
          <el-button @click="showGuide = true" type="info" plain class="guide-btn">
            <el-icon><QuestionFilled /></el-icon>
            使用指南
          </el-button>
          <el-input
            v-model="searchKeyword"
            placeholder="搜索脚本名称..."
            clearable
            @keyup.enter="handleSearch"
            class="search-input"
          >
            <template #prefix>
              <el-icon><search /></el-icon>
            </template>
          </el-input>
          <el-button @click="fetchScriptList" class="refresh-btn">
            <el-icon><refresh /></el-icon>
            刷新列表
          </el-button>
        </div>
      </div>

      <!-- 脚本表格 -->
      <el-table
        v-loading="loading"
        :data="scriptList"
        class="scripts-table"
        stripe
      >
        <el-table-column label="脚本名称" min-width="180" sortable prop="name" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="script-name-cell">
              <el-icon class="script-icon"><document /></el-icon>
              <span class="script-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="描述" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="script-desc">{{ row.description || '暂无描述' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="主文件" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="file-name-text">{{ row.file_name || '未上传主文件' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160" sortable prop="created_at">
          <template #default="{ row }">
            <span class="time-text">{{ formatDate(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="修改时间" width="160" sortable prop="updated_at">
          <template #default="{ row }">
            <span class="time-text">{{ formatDate(row.updated_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <div class="action-btns">
              <el-button
                link
                type="info"
                @click="handleDownload(row)"
                :disabled="!row.file_path"
                class="action-btn download-btn"
              >
                <el-icon><download /></el-icon>
                下载
              </el-button>
              <el-button
                link
                type="primary"
                @click="handleEdit(row)"
                class="action-btn edit-btn"
              >
                <el-icon><edit /></el-icon>
                编辑
              </el-button>
              <el-button
                link
                type="success"
                @click="openExecuteDialog(row)"
                class="action-btn execute-btn"
              >
                <el-icon><video-play /></el-icon>
                执行
              </el-button>
              <el-button
                link
                type="danger"
                @click="handleDelete(row)"
                :loading="deletingId === row.id"
                :disabled="deletingId === row.id"
                class="action-btn delete-btn"
              >
                <el-icon v-if="deletingId !== row.id"><delete /></el-icon>
                删除
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 空状态 -->
      <div v-if="!loading && scriptList.length === 0" class="empty-state">
        <div class="empty-icon">
          <el-icon><document-delete /></el-icon>
        </div>
        <h3 class="empty-title">暂无脚本数据</h3>
        <p class="empty-desc">请在上方上传区域添加脚本</p>
      </div>

      <!-- 分页 -->
      <div class="pagination-wrapper" v-if="total > 0">
        <el-pagination
          v-model:current-page="pageNum"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>

    <!-- 执行弹窗 -->
    <ExecuteDialog
      v-model:visible="executeDialogVisible"
      :script-id="currentScriptId"
      :script-name="currentScriptName"
      @success="fetchScriptList"
    />

    <!-- 使用指南弹窗 -->
    <el-dialog
      v-model="showGuide"
      title="使用指南"
      width="680px"
      :close-on-click-modal="true"
      class="guide-dialog"
    >
      <div class="guide-content">
        <div class="guide-section">
          <h3>1. 上传脚本</h3>
          <p>在脚本管理页面上传 .jmx 脚本文件。名称会默认使用主文件名，你只需要补充描述后点击上传。</p>
        </div>
        <div class="guide-section">
          <h3>2. 编辑脚本</h3>
          <p>点击脚本列表中的"编辑"按钮进入编辑页面。支持两种编辑模式：</p>
          <ul>
            <li><strong>可视化编辑</strong>：以树形结构展示 JMeter 元素，可直接修改线程数、请求地址、Header 等配置</li>
            <li><strong>XML 源码</strong>：直接编辑原始 XML 内容</li>
          </ul>
        </div>
        <div class="guide-section">
          <h3>3. 关联数据文件</h3>
          <p>在编辑页面右侧的"关联文件"面板中上传 CSV、JSON 等数据文件。系统会自动检测 JMX 中引用的文件并提示关联状态。</p>
        </div>
        <div class="guide-section">
          <h3>4. 管理 Slave 节点</h3>
          <p>在 Slave 管理页面添加 JMeter Slave 节点地址，用于分布式压测。添加后点击"检测"验证连通性。</p>
        </div>
        <div class="guide-section">
          <h3>5. 执行测试</h3>
          <p>在脚本列表点击"执行"按钮，选择执行模式：</p>
          <ul>
            <li><strong>本地模式</strong>：在当前服务器执行</li>
            <li><strong>分布式模式</strong>：选择 Slave 节点进行分布式压测</li>
          </ul>
        </div>
        <div class="guide-section">
          <h3>6. 查看结果</h3>
          <p>在执行记录页面查看测试结果，包括样本数、响应时间、错误率、吞吐量等指标。点击详情可查看实时日志和完整报告。</p>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search,
  Upload,
  Document,
  Download,
  Edit,
  Delete,
  VideoPlay,
  DocumentDelete,
  Refresh,
  QuestionFilled
} from '@element-plus/icons-vue'
import { scriptApi } from '@/api/script'
import { executionApi } from '@/api/execution'
import FileUpload from '@/components/FileUpload.vue'
import ExecuteDialog from '@/components/ExecuteDialog.vue'
import { formatDateTimeInShanghai } from '@/utils/datetime'

const router = useRouter()

// 统计数据
const statsLoading = ref(false)
const stats = reactive({
  totalScripts: 0,
  totalFiles: 0,
  runningCount: 0,
  executionCount: 0
})

// 列表数据
const loading = ref(false)
const scriptList = ref([])
const pageNum = ref(1)
const pageSize = ref(10)
const total = ref(0)
const searchKeyword = ref('')
const deletingId = ref(null)

// 上传表单
const uploadLoading = ref(false)
const uploadFormRef = ref(null)
const uploadForm = reactive({
  description: '',
  files: []
})
const uploadRules = {
  files: [
    {
      validator: (rule, value, callback) => {
        if (!uploadForm.files || uploadForm.files.length === 0) {
          callback(new Error('请上传一个 .jmx 脚本文件'))
          return
        }

        if (uploadForm.files.length !== 1) {
          callback(new Error('只能上传一个主脚本文件'))
          return
        }

        if (!uploadForm.files[0]?.name?.toLowerCase().endsWith('.jmx')) {
          callback(new Error('只支持上传 .jmx 脚本文件'))
          return
        }

        callback()
      },
      trigger: 'change'
    }
  ]
}

// 执行弹窗
const executeDialogVisible = ref(false)
const currentScriptId = ref(null)
const currentScriptName = ref('')

// 使用指南弹窗
const showGuide = ref(false)

// 获取统计数据
const fetchStats = async () => {
  statsLoading.value = true
  try {
    // 获取脚本列表计算总数
    const scriptsRes = await scriptApi.getList({ page: 1, pageSize: 9999 })
    const scripts = scriptsRes.data?.list || []
    stats.totalScripts = scripts.length
    stats.totalFiles = scripts.reduce((sum, s) => sum + (s.file_count || 0), 0)

    // 获取执行记录数
    const execRes = await executionApi.getList({ page: 1, pageSize: 1 })
    stats.executionCount = execRes.data?.total || 0

    // 获取运行中的数量
    const runningRes = await executionApi.getList({ page: 1, pageSize: 9999, status: 'running' })
    stats.runningCount = runningRes.data?.total || 0
  } catch (error) {
    console.error('获取统计数据失败:', error)
  } finally {
    statsLoading.value = false
  }
}

// 获取脚本列表
const fetchScriptList = async () => {
  loading.value = true
  try {
    const res = await scriptApi.getList({
      page: pageNum.value,
      pageSize: pageSize.value,
      keyword: searchKeyword.value
    })
    scriptList.value = res.data?.list || []
    total.value = res.data?.total || 0
  } catch (error) {
    console.error('获取脚本列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  pageNum.value = 1
  fetchScriptList()
}

// 分页
const handleSizeChange = (size) => {
  pageSize.value = size
  pageNum.value = 1
  fetchScriptList()
}

const handlePageChange = (page) => {
  pageNum.value = page
  fetchScriptList()
}

// 上传提交
const handleUploadSubmit = async () => {
  const valid = await uploadFormRef.value?.validate().catch(() => false)
  if (!valid) return

  uploadLoading.value = true
  try {
    const selectedFile = uploadForm.files[0]?.raw || uploadForm.files[0]
    const fileName = selectedFile?.name || uploadForm.files[0]?.name || ''
    const derivedName = fileName.replace(/\.[^.]+$/, '').trim()

    // 构建 FormData，一次性提交 name、description 和 file
    const formData = new FormData()
    formData.append('name', derivedName)
    formData.append('description', uploadForm.description || '')
    formData.append('file', selectedFile)

    // 创建脚本并上传文件
    const scriptRes = await scriptApi.create(formData)

    if (scriptRes.data?.id) {
      ElMessage.success('脚本上传成功')
      // 重置表单
      uploadForm.description = ''
      uploadForm.files = []
      fetchScriptList()
      fetchStats()
    }
  } catch (error) {
    console.error('上传失败:', error)
    ElMessage.error('上传失败，请重试')
  } finally {
    uploadLoading.value = false
  }
}

// 编辑
const handleEdit = (row) => {
  router.push(`/scripts/${row.id}/edit`)
}

// 下载主脚本
const handleDownload = (row) => {
  if (!row.file_path) {
    ElMessage.warning('当前脚本没有可下载的主文件')
    return
  }
  scriptApi.download(row.id)
}

// 删除
const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除脚本 "${row.name}" 吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    deletingId.value = row.id
    await scriptApi.delete(row.id)
    ElMessage.success('删除成功')
    fetchScriptList()
    fetchStats()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除失败:', error)
      ElMessage.error('删除失败')
    }
  } finally {
    deletingId.value = null
  }
}

// 打开执行弹窗
const openExecuteDialog = (row) => {
  currentScriptId.value = row.id
  currentScriptName.value = row.name
  executeDialogVisible.value = true
}

// 格式化日期
const formatDate = (dateStr) => {
  return formatDateTimeInShanghai(dateStr)
}

onMounted(() => {
  fetchStats()
  fetchScriptList()
})
</script>

<style scoped lang="scss">
.script-list-page {
  padding: 16px;
}

// 统计概览卡片 - 小号指标条
.stats-overview {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 12px;

  .stat-card {
    background: var(--bg-card);
    border-radius: var(--radius-md);
    border: 1px solid rgba(255, 255, 255, 0.06);
    padding: 10px 12px;

    .stat-label {
      color: var(--text-secondary);
      font-size: 11px;
      margin-bottom: 4px;
    }

    .stat-value {
      color: var(--text-primary);
      font-size: 18px;
      font-weight: 700;
      line-height: 1;
    }
  }
}

// 区域卡片
.section-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid rgba(255, 255, 255, 0.06);
  padding: 14px;
  margin-bottom: 14px;
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
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 2px;
}

.section-desc {
  color: var(--text-secondary);
  font-size: 12px;
  margin-bottom: 12px;
}

.section-header-with-action {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;

  .section-header {
    flex: 1;
  }

  .section-actions {
    display: flex;
    gap: 12px;
    align-items: center;

    :deep(.el-button + .el-button) {
      margin-left: 0;
    }

    .search-input {
      width: 260px;

      :deep(.el-input__wrapper) {
        border-radius: var(--radius-md);
      }

      .el-icon {
        color: var(--text-secondary);
      }
    }

    .refresh-btn {
      border-radius: var(--radius-md);
      padding: 10px 20px;
      background: transparent;
      border: 1px solid rgba(255, 255, 255, 0.2);
      color: var(--text-primary);

      .el-icon {
        margin-right: 6px;
      }
    }
  }
}

// 上传区域 - 顶部工作台
.upload-section {
  padding: 16px 18px;
  background:
    linear-gradient(180deg, rgba(0, 102, 255, 0.06) 0%, rgba(0, 102, 255, 0) 42%),
    var(--bg-card);

  .upload-layout {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .upload-panel-copy {
    .section-desc {
      margin-bottom: 0;
      font-size: 12px;
      line-height: 1.6;
      color: var(--text-secondary);
    }
  }

  .upload-workbench {
    padding: 0;
    border: none;
    background: transparent;

    :deep(.el-form-item) {
      margin-bottom: 0;
    }

    :deep(.el-form-item__label) {
      padding-bottom: 8px;
      color: var(--text-secondary);
      font-size: 12px;
    }

    :deep(.el-input__wrapper) {
      min-height: 42px;
      border-radius: 14px;
    }
  }

  .upload-desc-row {
    margin-bottom: 12px;
  }

  .upload-file-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 12px;
    align-items: end;
    padding: 14px;
    border-radius: 16px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.06);
  }

  .desc-input-item,
  .file-input-item {
    width: 100%;
  }

  .desc-input-item {
    :deep(.el-textarea__wrapper),
    :deep(.el-input__wrapper) {
      border-radius: 14px;
    }
  }

  .file-input-item {
    min-width: 0;
    margin-bottom: 0;
  }

  .upload-action-slot {
    display: flex;
    align-items: flex-end;
    justify-content: flex-end;
    margin-bottom: 0;
  }

  .upload-submit-btn {
    border-radius: 14px;
    min-width: 188px;
    min-height: 42px;
    padding: 10px 20px;
    font-weight: 600;
    font-size: 14px;
    box-shadow: 0 10px 24px rgba(0, 102, 255, 0.18);

    .btn-icon {
      margin-right: 6px;
    }
  }
}

// 脚本列表区域
.scripts-section {
  .scripts-table {
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

    .script-name-cell {
      display: flex;
      align-items: center;
      gap: 10px;

      .script-icon {
        font-size: 18px;
        color: var(--accent-blue);
      }

      .script-name {
        color: var(--text-primary);
        font-weight: 500;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }

    .script-desc {
      color: var(--text-secondary);
    }

    .file-name-text {
      color: var(--text-secondary);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      display: inline-block;
      max-width: 100%;
    }

    .time-text {
      color: var(--text-secondary);
      font-size: 13px;
    }

    .action-btns {
      display: flex;
      align-items: center;
      flex-wrap: nowrap;
      gap: 4px;

      :deep(.el-button + .el-button) {
        margin-left: 0;
      }

      .action-btn {
        padding: 4px 6px;
        font-size: 13px;
        white-space: nowrap;

        .el-icon {
          margin-right: 4px;
          font-size: 14px;
        }
      }

      .edit-btn {
        color: var(--accent-blue);
      }

      .download-btn {
        color: var(--text-secondary);
      }

      .execute-btn {
        color: var(--accent-green);
      }

      .delete-btn {
        color: var(--accent-red) !important;
      }
      
      .delete-btn:hover {
        color: #ff5c52 !important;
      }
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

// 分页
.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

// 响应式
@media (max-width: 1200px) {
  .stats-overview {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .upload-section {
    .upload-file-row {
      grid-template-columns: 1fr;
    }

    .upload-action-slot {
      align-items: stretch;
    }
  }
}

@media (max-width: 768px) {
  .stats-overview {
    grid-template-columns: 1fr;
  }

  .section-header-with-action {
    flex-direction: column;
    gap: 16px;

    .section-actions {
      width: 100%;

      .search-input {
        flex: 1;
      }
    }
  }

  .upload-section {
    padding: 14px;

    .upload-file-row {
      padding: 12px;
    }

    .upload-submit-btn {
      width: 100%;
    }
  }
}

// 使用指南样式
.guide-content {
  .guide-section {
    margin-bottom: 20px;
    h3 {
      color: var(--text-primary);
      font-size: 15px;
      margin-bottom: 8px;
      display: flex;
      align-items: center;
      &::before {
        content: '';
        width: 4px;
        height: 16px;
        background: #0066ff;
        border-radius: 2px;
        margin-right: 8px;
      }
    }
    p {
      color: var(--text-secondary);
      font-size: 13px;
      line-height: 1.8;
      margin: 0;
    }
    ul {
      color: var(--text-secondary);
      font-size: 13px;
      line-height: 1.8;
      padding-left: 20px;
      margin: 4px 0 0 0;
      li {
        margin-bottom: 4px;
      }
    }
  }
}

.guide-btn {
  border-radius: var(--radius-md);
  padding: 10px 20px;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: var(--text-primary);

  .el-icon {
    margin-right: 6px;
  }
}
</style>
