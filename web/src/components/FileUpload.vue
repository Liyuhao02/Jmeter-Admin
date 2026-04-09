<template>
  <div class="file-upload" :class="{ 'is-compact': compact, 'is-single-tile': singleTile }">
    <el-upload
      :drag="!singleTile"
      :accept="accept"
      :multiple="multiple"
      :file-list="fileList"
      :auto-upload="false"
      :on-change="handleChange"
      :on-remove="handleRemove"
      class="upload-area"
    >
      <div class="upload-content">
        <template v-if="singleTile">
          <span class="upload-select-btn">选择文件</span>
        </template>
        <el-icon v-else class="upload-icon"><upload-filled /></el-icon>
        <div class="upload-text">
          <div class="upload-text-title">
            {{ currentFileName || singleTileTitle }}
          </div>
          <div v-if="compact && !singleTile && !currentFileName" class="upload-text-subtitle">
            {{ multiple ? '支持一次选择多个文件' : '每次上传一个文件' }}
          </div>
        </div>
      </div>
      <template v-if="defaultTip" #tip>
        <div class="upload-tip">
          <slot name="tip">{{ defaultTip }}</slot>
        </div>
      </template>
    </el-upload>

    <!-- 已选文件列表 -->
    <div v-if="showFileList && fileList.length > 0" class="file-list">
      <div class="file-list-header">
        <span class="list-title">已选文件</span>
        <span class="file-count">{{ fileList.length }} 个</span>
      </div>
      <div class="file-items">
        <div
          v-for="file in fileList"
          :key="file.uid || file.name"
          class="file-item"
        >
          <div class="file-info">
            <div class="file-type-icon">
              <el-icon><document /></el-icon>
            </div>
            <div class="file-details">
              <span class="file-name" :title="file.name">{{ file.name }}</span>
              <span class="file-size">{{ formatFileSize(file.size) }}</span>
            </div>
          </div>
          <div class="file-remove" @click="removeFile(file)" title="移除">
            <el-icon><delete /></el-icon>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, Document, Delete } from '@element-plus/icons-vue'

const props = defineProps({
  accept: {
    type: String,
    default: '*'
  },
  multiple: {
    type: Boolean,
    default: true
  },
  limit: {
    type: Number,
    default: 0
  },
  fileList: {
    type: Array,
    default: () => []
  },
  tip: {
    type: String,
    default: ''
  },
  compact: {
    type: Boolean,
    default: false
  },
  showFileList: {
    type: Boolean,
    default: true
  },
  singleTile: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:fileList', 'onChange'])

const defaultTip = computed(() => {
  if (props.singleTile) return ''
  return props.tip || `支持 ${props.accept} 格式的文件`
})

const currentFileName = computed(() => {
  if (!props.fileList.length) return ''
  return props.fileList[props.fileList.length - 1]?.name || ''
})

const singleTileTitle = computed(() => {
  return props.accept === '.jmx' ? '选择主脚本文件' : '选择文件'
})

const handleChange = (uploadFile, uploadFiles) => {
  let nextFiles = uploadFiles
  if (!props.multiple || props.limit === 1) {
    nextFiles = uploadFiles.slice(-1)
  } else if (props.limit > 0 && uploadFiles.length > props.limit) {
    nextFiles = uploadFiles.slice(-props.limit)
  }

  if (props.limit > 0 && uploadFiles.length > props.limit) {
    ElMessage.warning(`最多只能选择 ${props.limit} 个文件`)
  }

  emit('update:fileList', nextFiles)
  emit('onChange', nextFiles)
}

const handleRemove = (uploadFile, uploadFiles) => {
  emit('update:fileList', uploadFiles)
  emit('onChange', uploadFiles)
}

const removeFile = (file) => {
  const newList = props.fileList.filter(f => (f.uid || f.name) !== (file.uid || file.name))
  emit('update:fileList', newList)
  emit('onChange', newList)
}

const formatFileSize = (size) => {
  if (!size && size !== 0) return '未知大小'
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(2) + ' KB'
  return (size / 1024 / 1024).toFixed(2) + ' MB'
}
</script>

<style scoped lang="scss">
.file-upload {
  width: 100%;
}

.file-upload.is-compact {
  .upload-area {
    :deep(.el-upload-dragger) {
      padding: 16px;
      border-width: 1px;
      min-height: 112px;
    }
  }

  .upload-content {
    flex-direction: row;
    align-items: center;
    justify-content: flex-start;
    gap: 12px;
    text-align: left;
  }

  .upload-icon {
    font-size: 24px;
  }

  .upload-text {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 12px;
  }

  .upload-text-title {
    font-size: 13px;
  }

  .upload-text-subtitle {
    color: var(--text-secondary);
    font-size: 11px;
    line-height: 1.5;
  }

  .upload-tip {
    margin-top: 8px;
    font-size: 11px;
    text-align: left;
  }

  .file-list {
    margin-top: 8px;
  }

  .file-items {
    max-height: 108px;
  }

  .file-item {
    padding: 6px 8px;
  }
}

.file-upload.is-single-tile {
  .upload-area {
    :deep(.el-upload) {
      width: 100%;
      min-height: 56px;
      height: 56px;
      padding: 0 14px;
      border-width: 1px;
      border-style: solid;
      border-color: rgba(255, 255, 255, 0.1);
      background: rgba(255, 255, 255, 0.02);
      border-radius: 14px;
      display: flex;
      align-items: center;
      box-sizing: border-box;
      cursor: pointer;
      transition: all 0.25s ease;

      &:hover {
        border-color: rgba(120, 156, 255, 0.32);
        background: rgba(255, 255, 255, 0.04);
      }
    }
  }

  .upload-content {
    flex-direction: row;
    align-items: center;
    justify-content: flex-start;
    gap: 10px;
    text-align: left;
    width: 100%;
    min-width: 0;
  }

  .upload-select-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 36px;
    padding: 0 18px;
    border-radius: 12px;
    background: linear-gradient(180deg, rgba(116, 160, 255, 0.24) 0%, rgba(72, 115, 207, 0.28) 100%);
    border: 1px solid rgba(120, 156, 255, 0.2);
    color: var(--text-primary);
    font-size: 14px;
    font-weight: 600;
    flex-shrink: 0;
  }

  .upload-text {
    min-width: 0;
    flex: 1;
  }

  .upload-text-title {
    font-size: 13px;
    line-height: 1.4;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.upload-area {
  :deep(.el-upload) {
    width: 100%;
  }

  :deep(.el-upload-dragger) {
    width: 100%;
    padding: 16px 12px;
    background-color: var(--bg-card);
    border: 2px dashed rgba(255, 255, 255, 0.1);
    border-radius: var(--radius-md);
    transition: all 0.25s ease;

    &:hover {
      border-color: rgba(255, 255, 255, 0.2);
      background-color: var(--bg-hover);
    }

    &.is-dragover {
      border-color: var(--accent-blue);
      background-color: rgba(0, 102, 255, 0.05);
    }
  }
}

.upload-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.upload-icon {
  font-size: 28px;
  color: var(--accent-blue);
  opacity: 0.8;
  transition: all 0.25s ease;

  :deep(.el-upload-dragger:hover) & {
    opacity: 1;
  }
}

.upload-text {
  color: var(--text-secondary);
  font-size: 13px;
  text-align: center;

  em {
    color: var(--accent-blue);
    font-style: normal;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.25s ease;

    &:hover {
      text-decoration: underline;
    }
  }
}

.upload-text-title {
  color: var(--text-secondary);
}

.upload-text-subtitle {
  color: var(--text-secondary);
}

.upload-tip {
  color: var(--text-secondary);
  font-size: 11px;
  margin-top: 8px;
  text-align: center;
  opacity: 0.7;
}

// 已选文件列表
.file-list {
  margin-top: 12px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
  background-color: var(--bg-card);
  overflow: hidden;
  animation: slideInUp 0.3s ease-out;
}

.file-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background-color: var(--bg-secondary);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);

  .list-title {
    font-size: 12px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .file-count {
    font-size: 11px;
    color: var(--text-secondary);
    background-color: var(--bg-card);
    padding: 2px 6px;
    border-radius: var(--radius-sm);
  }
}

.file-items {
  max-height: 150px;
  overflow-y: auto;
  padding: 6px;
}

.file-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  margin-bottom: 4px;
  background-color: var(--bg-secondary);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-sm);
  transition: all 0.25s ease;

  &:last-child {
    margin-bottom: 0;
  }

  &:hover {
    border-color: rgba(255, 255, 255, 0.1);
    background-color: var(--bg-hover);

    .file-remove {
      opacity: 1;
    }
  }
}

.file-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.file-type-icon {
  width: 30px;
  height: 30px;
  border-radius: var(--radius-sm);
  background-color: rgba(0, 102, 255, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent-blue);
  flex-shrink: 0;

  .el-icon {
    font-size: 14px;
  }
}

.file-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.file-name {
  font-size: 12px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  font-size: 10px;
  color: var(--text-secondary);
}

.file-remove {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  opacity: 0;
  transition: all 0.25s ease;
  color: var(--text-secondary);
  flex-shrink: 0;

  &:hover {
    background-color: rgba(255, 69, 58, 0.1);
    color: var(--accent-red);
  }

  .el-icon {
    font-size: 14px;
  }
}
</style>
