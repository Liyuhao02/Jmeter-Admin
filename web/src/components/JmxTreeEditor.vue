<template>
  <div class="jmx-tree-editor">
    <!-- 左侧：元素树 -->
    <div class="tree-panel">
      <div class="panel-header">
        <div class="panel-header-row">
          <span class="panel-title">元素树</span>
          <div class="panel-header-actions">
            <el-tag size="small" type="info">{{ totalNodeCount }}</el-tag>
            <el-button
              link
              size="small"
              class="shortcut-help-btn"
              @click="shortcutDialogVisible = true"
              title="键盘快捷键"
            >
              <el-icon><InfoFilled /></el-icon>
            </el-button>
          </div>
        </div>
        <div class="search-row">
          <el-input
            v-model="treeFilterKeyword"
            clearable
            placeholder="搜索元素名称、类型或摘要"
            class="tree-search-input"
            aria-label="搜索 JMX 元素"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-button-group class="expand-collapse-group">
            <el-button size="small" @click="expandAllNodes" title="全部展开">
              <el-icon><Bottom /></el-icon>
            </el-button>
            <el-button size="small" @click="collapseAllNodes" title="全部折叠">
              <el-icon><Top /></el-icon>
            </el-button>
          </el-button-group>
        </div>

        <!-- 快捷添加工具栏 -->
        <div v-if="selectedNode && getQuickAddElements.length > 0" class="quick-add-toolbar">
          <div class="quick-add-label">
            <el-icon><Plus /></el-icon>
            <span>快捷添加</span>
          </div>
          <div class="quick-add-buttons">
            <el-tooltip
              v-for="elem in getQuickAddElements"
              :key="elem.type"
              :content="elem.label"
              placement="top"
              :show-after="300"
            >
              <el-button
                size="small"
                class="quick-add-btn"
                @click="quickAddElement(elem.type)"
              >
                <el-icon :style="{ color: getElementIconColor(elem.type) }">
                  <component :is="iconMap[elem.icon] || iconMap.QuestionFilled" />
                </el-icon>
                <span class="quick-add-text">{{ elem.label }}</span>
              </el-button>
            </el-tooltip>
          </div>
        </div>

        <!-- 多选批量操作工具栏 -->
        <div v-if="isMultiSelectMode" class="batch-toolbar">
          <div class="batch-info">
            <el-tag size="small" type="primary" effect="dark">
              已选 {{ multiSelectCount }} 个
            </el-tag>
            <el-button link size="small" @click="clearMultiSelect">
              清空
            </el-button>
          </div>
          <div class="batch-actions">
            <el-button size="small" type="success" @click="batchToggleEnabled(true)">
              <el-icon><CircleCheck /></el-icon> 启用
            </el-button>
            <el-button size="small" type="warning" @click="batchToggleEnabled(false)">
              <el-icon><CircleClose /></el-icon> 禁用
            </el-button>
            <el-button size="small" type="danger" @click="batchDelete">
              <el-icon><Delete /></el-icon> 删除
            </el-button>
          </div>
        </div>

        <div class="tree-toolbar">
          <div v-if="selectedNode" class="tree-selection-toolbar">
            <div class="tree-selection-meta">
              <span class="tree-selection-label">已选节点</span>
              <span class="tree-selection-name">{{ getNodeDisplayName(selectedNode) }}</span>
            </div>
            <div class="tree-selection-actions">
              <el-button
                v-if="selectedNode.testclass !== 'TestPlan'"
                size="small"
                class="tree-action-chip"
                :disabled="isFirstNode(selectedNode)"
                @click="moveNodeUp(selectedNode)"
              >
                上移
              </el-button>
              <el-button
                v-if="selectedNode.testclass !== 'TestPlan'"
                size="small"
                class="tree-action-chip"
                :disabled="isLastNode(selectedNode)"
                @click="moveNodeDown(selectedNode)"
              >
                下移
              </el-button>
              <el-dropdown
                v-if="canAddChild(selectedNode) || selectedNode.testclass !== 'TestPlan'"
                trigger="click"
                size="small"
                @command="(command) => handleAddCommand(command, selectedNode)"
              >
                <el-button size="small" class="tree-action-chip tree-action-chip--primary">
                  新增
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item v-if="canAddChild(selectedNode)" command="child">
                      <el-icon><Plus /></el-icon> 添加子元素
                    </el-dropdown-item>
                    <el-dropdown-item v-if="selectedNode.testclass !== 'TestPlan'" command="before">
                      <el-icon><ArrowUp /></el-icon> 在前面插入
                    </el-dropdown-item>
                    <el-dropdown-item v-if="selectedNode.testclass !== 'TestPlan'" command="after">
                      <el-icon><ArrowDown /></el-icon> 在后面插入
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <el-button
                v-if="selectedNode.testclass !== 'TestPlan'"
                size="small"
                class="tree-action-chip"
                @click="copyNode(selectedNode)"
              >
                复制
              </el-button>
              <el-button
                size="small"
                class="tree-action-chip"
                @click="toggleNodeEnabled(selectedNode)"
              >
                {{ selectedNode.enabled === false || selectedNode.enabled === 'false' ? '启用' : '禁用' }}
              </el-button>
              <el-button
                v-if="selectedNode.testclass !== 'TestPlan'"
                size="small"
                class="tree-action-chip tree-action-chip--danger"
                @click="deleteNode(selectedNode)"
              >
                删除
              </el-button>
            </div>
          </div>
        </div>
      </div>
      <div class="tree-content">
        <el-tree
          ref="treeRef"
          v-if="treeData.length > 0"
          :data="treeData"
          node-key="id"
          default-expand-all
          highlight-current
          :expand-on-click-node="false"
          :filter-node-method="filterNode"
          draggable
          :allow-drop="allowDrop"
          :allow-drag="allowDrag"
          @node-drop="handleDrop"
          @node-click="handleNodeClick"
          class="jmx-tree"
        >
          <template #default="{ data }">
            <div
              class="tree-node"
              :class="{
                'is-selected': selectedNode?.id === data.id,
                'is-multi-selected': isNodeMultiSelected(data),
                'is-disabled': data.enabled === false || data.enabled === 'false'
              }"
            >
              <div class="node-main">
                <div class="node-top-row">
                  <div class="node-title-wrap">
                    <el-icon class="node-icon" :style="{ color: getNodeIconColor(data) }">
                      <component :is="getNodeIcon(data)" />
                    </el-icon>
                    <span class="node-label" :title="data.testname || data.testclass">{{ data.testname || getNodeShortLabel(data) }}</span>
                  </div>
                  <div class="node-top-side">
                    <div class="node-badges">
                      <el-tag size="small" class="node-type-tag" :type="getNodeTagType(data)">{{ getNodeShortLabel(data) }}</el-tag>
                      <el-tag v-if="(data.children?.length || 0) > 0" size="small" type="info" effect="plain">
                        {{ data.children.length }} 子项
                      </el-tag>
                      <el-tag v-if="data.enabled === false || data.enabled === 'false'" size="small" type="danger" effect="plain">
                        已禁用
                      </el-tag>
                    </div>
                    <el-dropdown
                      class="node-menu"
                      trigger="click"
                      placement="bottom-end"
                      @command="(command) => handleNodeMenuCommand(command, data)"
                    >
                      <button
                        class="node-menu-btn"
                        type="button"
                        :aria-label="`${data.testname || getNodeShortLabel(data)} 更多操作`"
                        @click.stop
                      >
                        <el-icon><MoreFilled /></el-icon>
                      </button>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item
                            v-if="data.testclass !== 'TestPlan'"
                            command="move-up"
                            :disabled="isFirstNode(data)"
                          >
                            <el-icon><ArrowUp /></el-icon> 上移
                          </el-dropdown-item>
                          <el-dropdown-item
                            v-if="data.testclass !== 'TestPlan'"
                            command="move-down"
                            :disabled="isLastNode(data)"
                          >
                            <el-icon><ArrowDown /></el-icon> 下移
                          </el-dropdown-item>
                          <el-dropdown-item
                            v-if="data.testclass !== 'TestPlan'"
                            command="copy"
                          >
                            <el-icon><DocumentCopy /></el-icon> 复制
                          </el-dropdown-item>
                          <el-dropdown-item
                            v-if="canAddChild(data)"
                            command="child"
                          >
                            <el-icon><Plus /></el-icon> 添加子元素
                          </el-dropdown-item>
                          <el-dropdown-item
                            v-if="data.testclass !== 'TestPlan'"
                            command="before"
                          >
                            <el-icon><ArrowUp /></el-icon> 在前面插入
                          </el-dropdown-item>
                          <el-dropdown-item
                            v-if="data.testclass !== 'TestPlan'"
                            command="after"
                          >
                            <el-icon><ArrowDown /></el-icon> 在后面插入
                          </el-dropdown-item>
                          <el-dropdown-item command="toggle">
                            <el-icon>
                              <CircleCheck v-if="data.enabled === false || data.enabled === 'false'" />
                              <CircleClose v-else />
                            </el-icon>
                            {{ data.enabled === false || data.enabled === 'false' ? '启用' : '禁用' }}
                          </el-dropdown-item>
                          <el-dropdown-item
                            v-if="data.testclass !== 'TestPlan'"
                            command="delete"
                            divided
                          >
                            <el-icon><Delete /></el-icon> 删除
                          </el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </div>
                </div>
                <div class="node-meta-row">
                  <el-tooltip
                    :content="getNodeSummary(data) || getNodeMetaHint(data)"
                    placement="top"
                    :show-after="300"
                    :disabled="!(getNodeSummary(data) || getNodeMetaHint(data))"
                  >
                    <span class="node-summary" :class="{ 'is-empty': !getNodeSummary(data) }">
                      {{ getNodeSummary(data) || getNodeMetaHint(data) }}
                    </span>
                  </el-tooltip>
                </div>
              </div>
            </div>
          </template>
        </el-tree>
        <el-empty v-else description="暂无数据" />
      </div>
    </div>

    <!-- 右侧：属性编辑面板 -->
    <div class="property-panel">
      <div class="panel-header">
        <span class="panel-title">属性编辑</span>
      </div>
      <div class="property-content" v-if="selectedNode">
        <!-- 节点类型和名称 -->
        <div class="property-header">
          <div class="property-context">
            <div class="property-context-label">当前编辑节点</div>
            <div class="property-context-path">{{ selectedNodePath }}</div>
          </div>
          <div class="property-title-row">
            <el-icon class="property-icon" :style="{ color: getNodeIconColor(selectedNode) }">
              <component :is="getNodeIcon(selectedNode)" />
            </el-icon>
            <div class="property-title-info">
              <el-input
                v-model="selectedNode.testname"
                placeholder="元素名称"
                class="property-name-input"
                @change="handlePropertyChange"
              />
              <span class="property-type">{{ getElementTypeLabel(selectedNode.testclass) }}</span>
            </div>
            <div class="property-actions">
              <el-switch
                v-model="selectedNode.enabled"
                :active-value="true"
                :inactive-value="false"
                active-text="启用"
                inactive-text="禁用"
                @change="handlePropertyChange"
              />
            </div>
          </div>
          <div class="property-summary-row">
            <div class="summary-chip">
              <span class="summary-chip-label">类型</span>
              <span class="summary-chip-value">{{ getElementTypeLabel(selectedNode.testclass) }}</span>
            </div>
            <div class="summary-chip">
              <span class="summary-chip-label">子元素</span>
              <span class="summary-chip-value">{{ selectedNode.children?.length || 0 }}</span>
            </div>
            <div class="summary-chip summary-chip-wide">
              <span class="summary-chip-label">摘要</span>
              <span class="summary-chip-value">{{ getNodeSummary(selectedNode) || getNodeMetaHint(selectedNode) }}</span>
            </div>
          </div>
          <el-divider />
        </div>

        <!-- 属性表单 -->
        <div class="property-form" v-if="hasMetaDefinition(selectedNode)">
          <el-form label-position="top">
            <template v-for="prop in getPropertyDefinitions(selectedNode)" :key="prop.key">
              <!-- 字符串类型 -->
              <el-form-item :label="prop.label" v-if="prop.type === 'string'">
                <!-- CSVDataSet filename 字段特殊处理：支持下拉选择已上传文件 -->
                <el-select
                  v-if="prop.key === 'filename' && selectedNode.testclass === 'CSVDataSet' && csvFiles.length > 0"
                  v-model="selectedNode.properties[prop.key]"
                  filterable
                  allow-create
                  default-first-option
                  placeholder="选择或输入文件名"
                  style="width: 100%"
                  @change="handlePropertyChange"
                >
                  <el-option
                    v-for="file in csvFiles"
                    :key="file"
                    :label="file"
                    :value="file"
                  />
                </el-select>
                <el-input
                  v-else
                  v-model="selectedNode.properties[prop.key]"
                  @change="handlePropertyChange"
                />
              </el-form-item>

              <!-- 数字类型 -->
              <el-form-item :label="prop.label" v-else-if="prop.type === 'number'">
                <el-input-number
                  v-model="selectedNode.properties[prop.key]"
                  :min="prop.min"
                  :max="prop.max"
                  controls-position="right"
                  style="width: 100%"
                  @change="handlePropertyChange"
                />
              </el-form-item>

              <!-- 布尔类型 -->
              <el-form-item :label="prop.label" v-else-if="prop.type === 'boolean'">
                <el-switch
                  v-model="selectedNode.properties[prop.key]"
                  @change="handlePropertyChange"
                />
              </el-form-item>

              <!-- 选择类型 -->
              <el-form-item :label="prop.label" v-else-if="prop.type === 'select'">
                <el-select
                  v-model="selectedNode.properties[prop.key]"
                  style="width: 100%"
                  @change="handlePropertyChange"
                >
                  <el-option
                    v-for="opt in prop.options"
                    :key="opt.value"
                    :label="opt.label"
                    :value="opt.value"
                  />
                </el-select>
              </el-form-item>

              <!-- 文本域类型 -->
              <el-form-item :label="prop.label" v-else-if="prop.type === 'textarea'">
                <el-input
                  v-model="selectedNode.properties[prop.key]"
                  type="textarea"
                  :rows="prop.key === 'script' ? 15 : 6"
                  :class="{ 'script-textarea': prop.key === 'script' }"
                  @change="handlePropertyChange"
                />
              </el-form-item>

              <!-- 线程调度配置类型 -->
              <el-form-item :label="prop.label" v-else-if="prop.type === 'threadSchedule'">
                <div class="thread-schedule-editor">
                  <div class="schedule-header">
                    <span class="schedule-title">线程调度阶段</span>
                    <el-button type="primary" size="small" @click="addScheduleRow">
                      <el-icon><Plus /></el-icon> 添加阶段
                    </el-button>
                  </div>
                  <el-table :data="scheduleRows" border size="small" class="schedule-table">
                    <el-table-column label="线程数" min-width="100">
                      <template #default="{ row, $index }">
                        <el-input-number v-model="row.threads" :min="1" size="small" controls-position="right" 
                          @change="onScheduleChange" />
                      </template>
                    </el-table-column>
                    <el-table-column label="初始延迟(秒)" min-width="110">
                      <template #default="{ row }">
                        <el-input-number v-model="row.initialDelay" :min="0" size="small" controls-position="right"
                          @change="onScheduleChange" />
                      </template>
                    </el-table-column>
                    <el-table-column label="启动时间(秒)" min-width="110">
                      <template #default="{ row }">
                        <el-input-number v-model="row.startupTime" :min="0" size="small" controls-position="right"
                          @change="onScheduleChange" />
                      </template>
                    </el-table-column>
                    <el-table-column label="持续时间(秒)" min-width="110">
                      <template #default="{ row }">
                        <el-input-number v-model="row.holdTime" :min="0" size="small" controls-position="right"
                          @change="onScheduleChange" />
                      </template>
                    </el-table-column>
                    <el-table-column label="关闭时间(秒)" min-width="110">
                      <template #default="{ row }">
                        <el-input-number v-model="row.shutdownTime" :min="0" size="small" controls-position="right"
                          @change="onScheduleChange" />
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" width="70" align="center">
                      <template #default="{ $index }">
                        <el-button link type="danger" size="small" @click="removeScheduleRow($index)">
                          <el-icon><Delete /></el-icon>
                        </el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </div>
              </el-form-item>

              <!-- 键值对列表类型 -->
              <el-form-item :label="prop.label" v-else-if="prop.type === 'keyValueList'">
                <div class="key-value-list">
                  <el-table
                    :data="getKeyValueListData(selectedNode, prop)"
                    size="small"
                    class="key-value-table"
                  >
                    <el-table-column
                      v-for="(itemKey, index) in getKeyValueItemKeys(prop)"
                      :key="itemKey"
                      :label="getKeyValueItemLabels(prop)[index]"
                    >
                      <template #default="{ $index }">
                        <el-input
                          v-model="keyValueListData[$index][index]"
                          size="small"
                          @change="updateKeyValueList(prop)"
                        />
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" width="80">
                      <template #default="{ $index }">
                        <el-button
                          type="danger"
                          link
                          size="small"
                          @click="removeKeyValueItem(prop, $index)"
                        >
                          删除
                        </el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                  <el-button
                    type="primary"
                    link
                    class="add-btn"
                    @click="addKeyValueItem(prop)"
                  >
                    <el-icon><Plus /></el-icon>
                    添加一行
                  </el-button>
                </div>
              </el-form-item>

              <!-- 字符串列表类型（如 ResponseAssertion 的 test_strings） -->
              <el-form-item :label="prop.label" v-else-if="prop.type === 'stringList'">
                <div class="string-list-editor">
                  <div v-if="prop.description" class="string-list-hint">{{ prop.description }}</div>
                  <el-input
                    v-model="stringListData[prop.key]"
                    type="textarea"
                    :rows="4"
                    placeholder="每行一个字符串"
                    @change="updateStringList(prop)"
                  />
                </div>
              </el-form-item>
            </template>
          </el-form>
        </div>

        <!-- 无元数据定义时显示原始 XML -->
        <div class="raw-xml-panel" v-else>
          <div class="raw-xml-label">原始 XML（该元素类型暂无编辑器支持）</div>
          <el-input
            v-model="rawXmlContent"
            type="textarea"
            :rows="20"
            readonly
            class="raw-xml-textarea"
          />
        </div>
      </div>

      <!-- 未选择节点 -->
      <div class="empty-panel" v-else>
        <div class="empty-guide">
          <el-icon class="empty-icon"><Document /></el-icon>
          <div class="empty-title">选择一个元素</div>
          <div class="empty-desc">从左侧元素树中点击选择一个 JMeter 元素，即可在此处编辑其属性</div>
          <div class="empty-tips">
            <div class="tip-item">
              <el-icon><User /></el-icon>
              <span>线程组 - 配置并发用户数</span>
            </div>
            <div class="tip-item">
              <el-icon><Link /></el-icon>
              <span>HTTP请求 - 配置接口地址</span>
            </div>
            <div class="tip-item">
              <el-icon><Grid /></el-icon>
              <span>CSV数据文件 - 配置数据驱动</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加元素对话框 -->
    <el-dialog
      v-model="addElementDialogVisible"
      title="添加元素"
      width="650px"
      class="add-element-dialog"
    >
      <el-tabs v-model="addElementTab" class="element-tabs">
        <el-tab-pane
          v-for="cat in categoryConfig"
          :key="cat.key"
          :label="cat.label"
          :name="cat.key"
          :disabled="!getAllowedCategories(addElementTarget).includes(cat.key)"
        >
          <div class="element-grid">
            <div
              v-for="elem in getAvailableElements(cat.key)"
              :key="elem.type"
              class="element-card"
              @click="addElement(elem.type)"
            >
              <el-icon class="element-icon" :style="{ color: getElementIconColor(elem.type) }">
                <component :is="iconMap[elem.icon] || iconMap.QuestionFilled" />
              </el-icon>
              <span class="element-label">{{ elem.label }}</span>
            </div>
            <div v-if="getAvailableElements(cat.key).length === 0" class="empty-category">
              该类别没有可用元素
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>

    <!-- 键盘快捷键提示对话框 -->
    <el-dialog
      v-model="shortcutDialogVisible"
      title="键盘快捷键"
      width="480px"
      class="shortcut-dialog"
    >
      <div class="shortcut-list">
        <div class="shortcut-section">
          <h4>节点操作</h4>
          <div class="shortcut-item">
            <kbd>Delete</kbd> / <kbd>Backspace</kbd>
            <span>删除选中节点</span>
          </div>
          <div class="shortcut-item">
            <kbd>Ctrl</kbd> + <kbd>D</kbd>
            <span>复制选中节点</span>
          </div>
          <div class="shortcut-item">
            <kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>E</kbd>
            <span>启用/禁用选中节点</span>
          </div>
          <div class="shortcut-item">
            <kbd>Ctrl</kbd> + <kbd>↑</kbd>
            <span>上移节点</span>
          </div>
          <div class="shortcut-item">
            <kbd>Ctrl</kbd> + <kbd>↓</kbd>
            <span>下移节点</span>
          </div>
        </div>
        <div class="shortcut-section">
          <h4>多选操作</h4>
          <div class="shortcut-item">
            <kbd>Ctrl</kbd> + <kbd>Click</kbd>
            <span>多选/取消选择节点</span>
          </div>
          <div class="shortcut-item">
            <kbd>Delete</kbd>
            <span>批量删除选中的节点</span>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, watch, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Document,
  User,
  Link,
  Grid,
  FolderOpened,
  List,
  Setting,
  Tickets,
  Timer,
  Check,
  RefreshRight,
  Switch,
  Folder,
  Plus,
  QuestionFilled,
  Monitor,
  Coin,
  Cloudy,
  Cpu,
  Clock,
  Connection,
  DataAnalysis,
  Stamp,
  MagicStick,
  Sort,
  Opportunity,
  Odometer,
  Search,
  MoreFilled,
  EditPen,
  Histogram,
  Delete,
  Lock,
  DataLine,
  CircleClose,
  CircleCheck,
  DocumentCopy,
  ArrowUp,
  ArrowDown,
  ArrowRight,
  Bottom,
  Top,
  InfoFilled
} from '@element-plus/icons-vue'
import {
  parseJMX,
  serializeJMX,
  getElementMeta,
  getElementCategory,
  getElementsByCategory,
  parseKeyValueList,
  setKeyValueList,
  isLeafElement,
  getElementSummary,
  ELEMENT_META
} from '@/utils/jmxParser.js'

const props = defineProps({
  xmlContent: {
    type: String,
    default: ''
  },
  modelValue: {
    type: String,
    default: ''
  },
  uploadedFiles: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue'])

// 图标映射
const iconMap = {
  Document,
  User,
  Link,
  Grid,
  FolderOpened,
  List,
  Setting,
  Tickets,
  Timer,
  Check,
  RefreshRight,
  Switch,
  Folder,
  QuestionFilled,
  Monitor,
  Coin,
  Cloudy,
  Cpu,
  Clock,
  Connection,
  DataAnalysis,
  Stamp,
  MagicStick,
  Sort,
  Opportunity,
  Odometer,
  Search,
  MoreFilled,
  EditPen,
  Histogram,
  Delete,
  Lock,
  DataLine,
  CircleClose,
  CircleCheck,
  DocumentCopy,
  ArrowUp,
  ArrowDown,
  ArrowRight,
  Bottom,
  Top,
  InfoFilled
}

// 状态
const treeData = ref([])
const selectedNode = ref(null)
const selectedNodeRef = ref(null) // el-tree 的 Node 对象引用（用于键盘快捷键）
const originalXml = ref('')
const keyValueListData = ref([])
const stringListData = ref({})
const treeFilterKeyword = ref('')
const treeRef = ref(null)
const lastLocalXml = ref('')

// 多选状态
const multiSelectedNodes = ref(new Set())
const isMultiSelectMode = computed(() => multiSelectedNodes.value.size > 0)
const multiSelectCount = computed(() => multiSelectedNodes.value.size)

// 添加元素相关状态
const addElementDialogVisible = ref(false)
const addElementTarget = ref(null)
const addElementTab = ref('sampler')
const insertMode = ref('child') // 'child' | 'before' | 'after'

// 快捷键提示对话框
const shortcutDialogVisible = ref(false)

// 常用快捷元素配置
const quickElements = [
  { type: 'HTTPSamplerProxy', label: 'HTTP请求', icon: 'Link', category: 'sampler' },
  { type: 'ThreadGroup', label: '线程组', icon: 'User', category: 'threadGroup' },
  { type: 'LoopController', label: '循环控制器', icon: 'RefreshRight', category: 'controller' },
  { type: 'TransactionController', label: '事务控制器', icon: 'Folder', category: 'controller' },
  { type: 'IfController', label: 'IF控制器', icon: 'Switch', category: 'controller' },
  { type: 'CSVDataSet', label: 'CSV数据', icon: 'Grid', category: 'config' },
  { type: 'HeaderManager', label: 'HTTP头', icon: 'List', category: 'config' },
  { type: 'ConstantTimer', label: '定时器', icon: 'Timer', category: 'timer' },
  { type: 'ResponseAssertion', label: '断言', icon: 'Check', category: 'assertion' },
  { type: 'JSONPostProcessor', label: 'JSON提取', icon: 'Search', category: 'postProcessor' },
  { type: 'RegexExtractor', label: '正则提取', icon: 'Search', category: 'postProcessor' },
  { type: 'ResultCollector', label: '结果树', icon: 'DataAnalysis', category: 'listener' }
]

// 初始化解析
const getSourceXml = () => props.modelValue || props.xmlContent || ''

const initParse = (xml = getSourceXml()) => {
  if (!xml) {
    treeData.value = []
    selectedNode.value = null
    return
  }
  
  try {
    originalXml.value = xml
    treeData.value = parseJMX(xml)
    selectedNode.value = null
    keyValueListData.value = []
    nextTick(() => {
      if (treeFilterKeyword.value) {
        treeRef.value?.filter(treeFilterKeyword.value)
      }
    })
  } catch (error) {
    console.error('解析 JMX 失败:', error)
    treeData.value = []
  }
}

watch(getSourceXml, (xml) => {
  if (xml === lastLocalXml.value) {
    originalXml.value = xml
    lastLocalXml.value = ''
    return
  }
  initParse(xml)
}, { immediate: true })

watch(treeFilterKeyword, (keyword) => {
  treeRef.value?.filter(keyword)
})

const totalNodeCount = computed(() => {
  const countNodes = (nodes) => nodes.reduce((count, node) => {
    return count + 1 + countNodes(node.children || [])
  }, 0)

  return countNodes(treeData.value)
})

const syncSelectedNodeEditorState = (node) => {
  if (!node) {
    keyValueListData.value = []
    stringListData.value = {}
    return
  }

  const meta = getElementMeta(node.testclass)
  if (meta && meta.properties) {
    // 处理 keyValueList 类型
    const keyValueProp = meta.properties.find((prop) => prop.type === 'keyValueList')
    if (keyValueProp) {
      refreshKeyValueListData(keyValueProp)
    } else {
      keyValueListData.value = []
    }

    // 处理 stringList 类型
    stringListData.value = {}
    meta.properties.forEach((prop) => {
      if (prop.type === 'stringList') {
        const list = node.properties[prop.key]
        if (Array.isArray(list)) {
          stringListData.value[prop.key] = list.join('\n')
        } else {
          stringListData.value[prop.key] = ''
        }
      }
    })
    return
  }

  keyValueListData.value = []
  stringListData.value = {}
}

const focusNode = (node, shouldScrollIntoView = false) => {
  selectedNode.value = node || null
  syncSelectedNodeEditorState(node)

  nextTick(() => {
    treeRef.value?.setCurrentKey?.(node?.id)

    if (shouldScrollIntoView) {
      const currentNodeEl = treeRef.value?.$el?.querySelector('.el-tree-node.is-current')
      currentNodeEl?.scrollIntoView?.({ block: 'nearest' })
    }
  })
}

const filterNode = (value, data) => {
  if (!value) return true

  const keyword = value.trim().toLowerCase()
  const label = getElementTypeLabel(data.testclass).toLowerCase()
  const name = (data.testname || '').toLowerCase()
  const testclass = (data.testclass || '').toLowerCase()
  const summary = (getNodeSummary(data) || '').toLowerCase()

  return [label, name, testclass, summary].some((item) => item.includes(keyword))
}

const normalizeTestclass = (testclass = '') => {
  return testclass.includes('.') ? testclass.split('.').pop() : testclass
}

const getNodeDisplayName = (node) => {
  if (!node) return ''
  return node.testname || getElementTypeLabel(node.testclass)
}

const getNodeMetaHint = (node) => {
  const category = getElementCategory(node.testclass)
  const hints = {
    threadGroup: '配置并发和执行节奏',
    sampler: '配置请求目标和发送内容',
    controller: '组织执行顺序和条件',
    config: '配置变量、文件和默认值',
    timer: '控制请求节奏和吞吐',
    assertion: '定义成功或失败判定',
    preProcessor: '请求前准备变量和参数',
    postProcessor: '响应后提取和清洗数据',
    listener: '收集日志、结果和指标'
  }

  return hints[category] || '编辑当前节点配置'
}

const buildNodePath = (nodes, targetNode, trail = []) => {
  for (const node of nodes) {
    const nextTrail = [...trail, getNodeDisplayName(node)]
    if (node === targetNode) {
      return nextTrail
    }
    if (node.children?.length) {
      const result = buildNodePath(node.children, targetNode, nextTrail)
      if (result.length) return result
    }
  }
  return []
}

const selectedNodePath = computed(() => {
  if (!selectedNode.value) return ''
  return buildNodePath(treeData.value, selectedNode.value).join(' / ')
})

// 获取节点图标
const getNodeIcon = (node) => {
  const meta = getElementMeta(node.testclass)
  if (meta && meta.icon && iconMap[meta.icon]) {
    return iconMap[meta.icon]
  }
  return iconMap.QuestionFilled
}

// 获取节点图标颜色
const getNodeIconColor = (node) => {
  const testclass = normalizeTestclass(node.testclass)
  const colors = {
    TestPlan: '#0a84ff',
    ThreadGroup: '#30d158',
    SetupThreadGroup: '#30d158',
    PostThreadGroup: '#30d158',
    UltimateThreadGroup: '#30d158',
    HTTPSamplerProxy: '#ff9f0a',
    DebugSampler: '#64d2ff',
    JDBCSampler: '#64d2ff',
    CSVDataSet: '#64d2ff',
    HeaderManager: '#bf5af2',
    CookieManager: '#bf5af2',
    ConstantTimer: '#ff375f',
    UniformRandomTimer: '#ff375f',
    GaussianRandomTimer: '#ff375f',
    ResponseAssertion: '#30d158',
    JSONPathAssertion: '#30d158',
    DurationAssertion: '#30d158',
    SizeAssertion: '#30d158',
    LoopController: '#ff9f0a',
    IfController: '#ff9f0a',
    WhileController: '#ff9f0a',
    TransactionController: '#8e8e93',
    CriticalSectionController: '#bf5af2',
    Arguments: '#bf5af2',
    ConfigTestElement: '#bf5af2',
    DataSourceElement: '#bf5af2',
    RegexExtractor: '#ff9f0a',
    JSONPostProcessor: '#ff9f0a',
    XPathExtractor: '#ff9f0a',
    BoundaryExtractor: '#ff9f0a',
    BeanShellPreProcessor: '#64d2ff',
    BeanShellPostProcessor: '#64d2ff',
    JSR223PreProcessor: '#64d2ff',
    JSR223PostProcessor: '#64d2ff',
    JSR223Sampler: '#64d2ff',
    JSR223Listener: '#ff9f0a',
    ResultCollector: '#0a84ff',
    ForeachController: '#ff9f0a',
    OnceOnlyController: '#8e8e93',
    RandomController: '#8e8e93',
  }
  return colors[node.testclass] || colors[testclass] || '#8e8e93'
}

// 获取元素类型中文标签
const getElementTypeLabel = (testclass) => {
  const meta = getElementMeta(testclass)
  return meta ? meta.label : testclass
}

// 获取节点显示标签
const getNodeLabel = (node) => {
  const meta = getElementMeta(node.testclass)
  let label = meta ? meta.label : node.testclass
  
  // 如果有 summary 函数，使用 summary
  if (meta && meta.summary && node.properties) {
    const summary = meta.summary(node.properties)
    if (summary) {
      return `${label} - ${summary}`
    }
  }
  
  // 否则显示 testname
  if (node.testname) {
    return `${label} - ${node.testname}`
  }
  
  return label
}

// 获取节点摘要信息
const getNodeSummary = (data) => {
  if (!data.testclass || !data.properties) return ''
  return getElementSummary(data.testclass, data.properties)
}

// 获取节点简短标签（用于树节点类型标签）
const getNodeShortLabel = (node) => {
  const meta = getElementMeta(node.testclass)
  if (meta && meta.label) {
    // 简化标签，取第一个词或前4个字
    const label = meta.label
    if (label.length <= 4) return label
    // 如果是中文，取前4个字
    if (/[\u4e00-\u9fa5]/.test(label)) {
      return label.substring(0, 4)
    }
    return label.split(' ')[0]
  }
  return node.testclass
}

// 获取节点标签类型（用于 el-tag 的 type 属性）
const getNodeTagType = (node) => {
  const testclass = normalizeTestclass(node.testclass)
  // 线程组 -> '' (primary)
  const threadGroups = ['ThreadGroup', 'SetupThreadGroup', 'PostThreadGroup', 'UltimateThreadGroup']
  if (threadGroups.includes(testclass)) return ''
  
  // 采样器 -> 'warning'
  const samplers = ['HTTPSamplerProxy', 'DebugSampler', 'JDBCSampler', 'JSR223Sampler']
  if (samplers.includes(testclass)) return 'warning'
  
  // 断言 -> 'danger'
  const assertions = ['ResponseAssertion', 'JSONPathAssertion', 'DurationAssertion', 'SizeAssertion']
  if (assertions.includes(testclass)) return 'danger'
  
  // 提取器 -> 'danger'
  const extractors = ['RegexExtractor', 'JSONPostProcessor', 'XPathExtractor', 'BoundaryExtractor']
  if (extractors.includes(testclass)) return 'danger'
  
  // 配置元素 -> 'success'
  const configs = ['CSVDataSet', 'HeaderManager', 'Arguments', 'CookieManager', 'ConfigTestElement', 'DataSourceElement']
  if (configs.includes(testclass)) return 'success'
  
  // 其他 -> 'info'
  return 'info'
}

// 切换节点启用状态
const toggleNodeEnabled = (node) => {
  const isDisabled = node.enabled === false || node.enabled === 'false'
  node.enabled = isDisabled
  handlePropertyChange()
}

// 删除节点
const deleteNode = (node) => {
  // TestPlan 不允许删除
  if (node.testclass === 'TestPlan') return

  const result = findParentNode(node)
  if (!result) return

  const { parent, siblings } = result
  const children = siblings

  const index = children.findIndex(c => c.id === node.id)
  if (index > -1) {
    const nextSelectedNode = children[index + 1] || children[index - 1] || parent || null
    children.splice(index, 1)
    emitUpdate()
    focusNode(nextSelectedNode, true)
  }
}

// 处理节点点击
const handleNodeClick = (data, node, treeNode, event) => {
  // 保存 el-tree 的 Node 对象引用（用于键盘快捷键）
  selectedNodeRef.value = node

  // 多选模式：按住 Ctrl/Cmd 点击
  if (event?.ctrlKey || event?.metaKey) {
    event?.preventDefault()
    if (multiSelectedNodes.value.has(data.id)) {
      multiSelectedNodes.value.delete(data.id)
    } else {
      multiSelectedNodes.value.add(data.id)
    }
    // 触发响应式更新
    multiSelectedNodes.value = new Set(multiSelectedNodes.value)
    // 单选当前节点但不清空多选
    focusNode(data)
    return
  }

  // 普通点击：清空多选
  if (!event?.shiftKey) {
    multiSelectedNodes.value.clear()
    multiSelectedNodes.value = new Set()
  }

  focusNode(data)
}

// 检查节点是否被多选
const isNodeMultiSelected = (data) => {
  return multiSelectedNodes.value.has(data.id)
}

// 获取多选节点数据数组
const getMultiSelectedNodesData = () => {
  const result = []
  const traverse = (nodes) => {
    for (const node of nodes) {
      if (multiSelectedNodes.value.has(node.id)) {
        result.push(node)
      }
      if (node.children?.length) {
        traverse(node.children)
      }
    }
  }
  traverse(treeData.value)
  return result
}

// 批量删除
const batchDelete = () => {
  const nodesToDelete = getMultiSelectedNodesData()
  if (nodesToDelete.length === 0) return

  // 过滤掉 TestPlan
  const deletableNodes = nodesToDelete.filter(n => n.testclass !== 'TestPlan')
  if (deletableNodes.length === 0) {
    ElMessage.warning('测试计划不能删除')
    return
  }

  // 逐个删除
  deletableNodes.forEach(node => {
    const result = findParentNode(node)
    if (!result) return
    const { siblings } = result
    const index = siblings.findIndex(c => c.id === node.id)
    if (index > -1) {
      siblings.splice(index, 1)
    }
  })

  multiSelectedNodes.value.clear()
  multiSelectedNodes.value = new Set()
  emitUpdate()
  focusNode(null)
  ElMessage.success(`已删除 ${deletableNodes.length} 个节点`)
}

// 批量启用/禁用
const batchToggleEnabled = (enabled) => {
  const nodesToToggle = getMultiSelectedNodesData()
  if (nodesToToggle.length === 0) return

  nodesToToggle.forEach(node => {
    node.enabled = enabled
  })

  emitUpdate()
  ElMessage.success(`已${enabled ? '启用' : '禁用'} ${nodesToToggle.length} 个节点`)
}

// 清空多选
const clearMultiSelect = () => {
  multiSelectedNodes.value.clear()
  multiSelectedNodes.value = new Set()
}

// 检查是否有元数据定义
const hasMetaDefinition = (node) => {
  const meta = getElementMeta(node.testclass)
  return meta && meta.properties && meta.properties.length > 0
}

const getKeyValueItemKeys = (prop) => {
  if (prop.itemKeys?.length) return prop.itemKeys
  if (prop.columns?.length) return prop.columns.map((item) => item.key)
  return []
}

const getKeyValueItemLabels = (prop) => {
  if (prop.itemLabels?.length) return prop.itemLabels
  if (prop.columns?.length) return prop.columns.map((item) => item.label)
  return []
}

// 获取属性定义
const getPropertyDefinitions = (node) => {
  const meta = getElementMeta(node.testclass)
  if (!meta || !meta.properties) return []
  
  // 处理嵌套属性
  return meta.properties.map(prop => {
    if (prop.nested && node.properties._nested) {
      // 从嵌套属性中读取值
      const nestedValue = node.properties._nested[prop.nested]?.[prop.key]
      if (nestedValue !== undefined) {
        node.properties[prop.key] = nestedValue
      }
    }
    return prop
  })
}

// 处理属性变化
const handlePropertyChange = () => {
  if (!originalXml.value || treeData.value.length === 0) return
  
  try {
    const newXml = serializeJMX(treeData.value, originalXml.value)
    originalXml.value = newXml
    lastLocalXml.value = newXml
    emit('update:modelValue', newXml)
  } catch (error) {
    console.error('序列化 JMX 失败:', error)
  }
}

const handleNodeMenuCommand = (command, node) => {
  switch (command) {
    case 'move-up':
      moveNodeUp(node)
      break
    case 'move-down':
      moveNodeDown(node)
      break
    case 'copy':
      copyNode(node)
      break
    case 'child':
    case 'before':
    case 'after':
      handleAddCommand(command, node)
      break
    case 'toggle':
      toggleNodeEnabled(node)
      break
    case 'delete':
      deleteNode(node)
      break
    default:
      break
  }
}

// 获取键值对列表数据
const getKeyValueListData = (node, prop) => {
  const data = parseKeyValueList(node.properties, prop.key, getKeyValueItemKeys(prop))
  keyValueListData.value = data.length > 0 ? data : []
  return keyValueListData.value
}

// 刷新键值对列表数据
const refreshKeyValueListData = (prop) => {
  if (!selectedNode.value) return
  const data = parseKeyValueList(selectedNode.value.properties, prop.key, getKeyValueItemKeys(prop))
  keyValueListData.value = data.length > 0 ? data : []
}

// 更新键值对列表
const updateKeyValueList = (prop) => {
  if (!selectedNode.value) return
  setKeyValueList(selectedNode.value.properties, prop.key, getKeyValueItemKeys(prop), keyValueListData.value)
  handlePropertyChange()
}

// 添加键值对项
const addKeyValueItem = (prop) => {
  keyValueListData.value.push(new Array(getKeyValueItemKeys(prop).length).fill(''))
  updateKeyValueList(prop)
}

// 删除键值对项
const removeKeyValueItem = (prop, index) => {
  keyValueListData.value.splice(index, 1)
  updateKeyValueList(prop)
}

// 更新字符串列表
const updateStringList = (prop) => {
  if (!selectedNode.value) return
  const text = stringListData.value[prop.key] || ''
  // 将文本按行分割成数组，过滤空行
  const list = text.split('\n').filter(line => line.trim() !== '')
  selectedNode.value.properties[prop.key] = list
  handlePropertyChange()
}

// 原始 XML 内容
const rawXmlContent = computed(() => {
  if (!selectedNode.value || !selectedNode.value._xmlNode) return ''
  const serializer = new XMLSerializer()
  return serializer.serializeToString(selectedNode.value._xmlNode)
})

// 已上传的 CSV 文件列表（用于 CSVDataSet filename 下拉选择）
const csvFiles = computed(() => {
  return props.uploadedFiles
    .filter(f => f.file_name && f.file_name.toLowerCase().endsWith('.csv'))
    .map(f => f.file_name)
})

// 线程调度配置数据
const scheduleRows = computed(() => {
  if (!selectedNode.value?.properties?.ultimatethreadgroupdata) return []
  return selectedNode.value.properties.ultimatethreadgroupdata.map(row => ({
    threads: Number(row.threads) || 0,
    initialDelay: Number(row.initialDelay) || 0,
    startupTime: Number(row.startupTime) || 0,
    holdTime: Number(row.holdTime) || 0,
    shutdownTime: Number(row.shutdownTime) || 0
  }))
})

const addScheduleRow = () => {
  if (!selectedNode.value?.properties) return
  if (!selectedNode.value.properties.ultimatethreadgroupdata) {
    selectedNode.value.properties.ultimatethreadgroupdata = []
  }
  selectedNode.value.properties.ultimatethreadgroupdata.push({
    threads: '100', initialDelay: '0', startupTime: '10', holdTime: '60', shutdownTime: '10'
  })
  emitUpdate()
}

const removeScheduleRow = (index) => {
  if (!selectedNode.value?.properties?.ultimatethreadgroupdata) return
  selectedNode.value.properties.ultimatethreadgroupdata.splice(index, 1)
  emitUpdate()
}

const onScheduleChange = () => {
  // 将数字回写到 properties
  if (!selectedNode.value?.properties?.ultimatethreadgroupdata) return
  scheduleRows.value.forEach((row, i) => {
    const original = selectedNode.value.properties.ultimatethreadgroupdata[i]
    if (original) {
      original.threads = String(row.threads)
      original.initialDelay = String(row.initialDelay)
      original.startupTime = String(row.startupTime)
      original.holdTime = String(row.holdTime)
      original.shutdownTime = String(row.shutdownTime)
    }
  })
  emitUpdate()
}

const emitUpdate = () => {
  if (!originalXml.value || treeData.value.length === 0) return
  try {
    const currentNodeId = selectedNode.value?.id
    // 触发 treeData 的响应式更新
    treeData.value = [...treeData.value]
    const newXml = serializeJMX(treeData.value, originalXml.value)
    originalXml.value = newXml
    lastLocalXml.value = newXml
    emit('update:modelValue', newXml)
    nextTick(() => {
      treeRef.value?.setCurrentKey?.(currentNodeId)
    })
  } catch (error) {
    console.error('序列化 JMX 失败:', error)
  }
}

const updateTreeExpansionState = (expanded) => {
  const rootNodes = treeRef.value?.store?.root?.childNodes || []
  const walkNodes = (nodes) => {
    nodes.forEach((treeNode) => {
      if (treeNode.childNodes?.length) {
        treeNode.expanded = expanded
        walkNodes(treeNode.childNodes)
      }
    })
  }

  walkNodes(rootNodes)
}

const expandAllNodes = () => {
  nextTick(() => updateTreeExpansionState(true))
}

const collapseAllNodes = () => {
  nextTick(() => updateTreeExpansionState(false))
}

// ========== 键盘快捷键处理 ==========

const handleKeydown = (e) => {
  // 确保不在输入框、文本域或内容编辑元素中时才触发
  if (['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target.tagName)) return
  if (e.target.contentEditable === 'true') return

  // 多选模式下，只支持 Delete 批量删除
  if (isMultiSelectMode.value) {
    if (e.key === 'Delete' || e.key === 'Backspace') {
      e.preventDefault()
      batchDelete()
    }
    return
  }

  // 单选模式下，需要选中节点才能操作
  if (!selectedNodeRef.value) return

  switch (e.key) {
    case 'Delete':
    case 'Backspace':
      e.preventDefault()
      if (selectedNode.value && selectedNode.value.testclass !== 'TestPlan') {
        deleteNode(selectedNode.value)
      }
      break
  }

  // Ctrl/Cmd 快捷键
  if (e.ctrlKey || e.metaKey) {
    switch (e.key) {
      case 'd':
      case 'D':
        e.preventDefault()
        if (selectedNode.value && selectedNode.value.testclass !== 'TestPlan') {
          copyNode(selectedNode.value)
        }
        break
      case 'ArrowUp':
        e.preventDefault()
        if (selectedNode.value && selectedNode.value.testclass !== 'TestPlan') {
          moveNodeUp(selectedNode.value)
        }
        break
      case 'ArrowDown':
        e.preventDefault()
        if (selectedNode.value && selectedNode.value.testclass !== 'TestPlan') {
          moveNodeDown(selectedNode.value)
        }
        break
    }

    // Ctrl+Shift+E 启用/禁用
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && (e.key === 'e' || e.key === 'E')) {
      e.preventDefault()
      if (selectedNode.value) {
        toggleNodeEnabled(selectedNode.value)
      }
    }
  }
}

// 注册和移除键盘事件监听
onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})

// ========== 快捷添加元素相关函数 ==========

// 获取当前选中节点可以快捷添加的元素列表
const getQuickAddElements = computed(() => {
  if (!selectedNode.value) return []

  const node = selectedNode.value
  const isLeaf = isLeafElement(node.testclass)

  // 如果是叶子节点，基于父节点判断可添加的元素（插入到后面）
  if (isLeaf) {
    const result = findParentNode(node)
    if (!result || !result.parent) return []
    const parentTestclass = result.parent.testclass
    return quickElements.filter(elem => isAllowedChild(parentTestclass, elem.type))
  }

  // 非叶子节点，基于当前节点判断可添加的子元素
  return quickElements.filter(elem => isAllowedChild(node.testclass, elem.type))
})

// 快捷添加元素
const quickAddElement = (elementType) => {
  if (!selectedNode.value) return

  const node = selectedNode.value
  const isLeaf = isLeafElement(node.testclass)

  // 如果是叶子节点，插入到后面
  if (isLeaf) {
    const result = findParentNode(node)
    if (!result || !result.siblings) {
      ElMessage.error('无法找到插入位置')
      return
    }
    insertMode.value = 'after'
    addElementTarget.value = node
  } else {
    // 非叶子节点，添加为子元素
    insertMode.value = 'child'
    addElementTarget.value = node
  }

  // 调用添加元素
  addElement(elementType)
}

// ========== 添加元素相关函数 ==========

// 类别配置
const categoryConfig = [
  { key: 'sampler', label: '采样器' },
  { key: 'threadGroup', label: '线程组' },
  { key: 'controller', label: '控制器' },
  { key: 'config', label: '配置元素' },
  { key: 'timer', label: '定时器' },
  { key: 'preProcessor', label: '前置处理器' },
  { key: 'postProcessor', label: '后置处理器' },
  { key: 'assertion', label: '断言' },
  { key: 'listener', label: '监听器' }
]

// 判断节点是否可以添加子元素
const canAddChild = (node) => {
  if (!node) return false
  
  // 如果节点是叶子元素，不允许添加子元素
  if (isLeafElement(node.testclass)) return false
  
  const category = getElementCategory(node.testclass)
  
  // TestPlan 可以添加线程组、配置元素、监听器、定时器、处理器、断言
  if (node.testclass === 'TestPlan') {
    return true
  }
  
  // 线程组可以添加采样器、控制器、配置元素、定时器、断言、处理器、监听器
  if (category === 'threadGroup') {
    return true
  }
  
  // 控制器可以添加采样器、控制器、配置元素、定时器、断言、处理器、监听器
  if (category === 'controller') {
    return true
  }
  
  // 采样器可以添加定时器、断言、处理器、监听器、配置元素
  if (category === 'sampler') {
    return true
  }
  
  return false
}

// 获取节点可添加的元素类别
const getAllowedCategories = (node) => {
  if (!node) return []
  const category = getElementCategory(node.testclass)
  
  // TestPlan - 添加 timer、preProcessor、postProcessor、assertion
  if (node.testclass === 'TestPlan') {
    return ['threadGroup', 'config', 'listener', 'timer', 'preProcessor', 'postProcessor', 'assertion']
  }
  
  // 线程组
  if (category === 'threadGroup') {
    return ['sampler', 'controller', 'config', 'timer', 'preProcessor', 'postProcessor', 'assertion', 'listener']
  }
  
  // 控制器
  if (category === 'controller') {
    return ['sampler', 'controller', 'config', 'timer', 'preProcessor', 'postProcessor', 'assertion', 'listener']
  }
  
  // 采样器
  if (category === 'sampler') {
    return ['timer', 'assertion', 'preProcessor', 'postProcessor', 'listener', 'config']
  }
  
  return []
}

// 检查是否为允许的子元素（用于拖拽和插入）
const isAllowedChild = (parentTestclass, childTestclass) => {
  const parentCategory = getElementCategory(parentTestclass)
  const childCategory = getElementCategory(childTestclass)
  
  // 获取父元素允许的子元素类型列表
  let allowed = []
  if (parentTestclass === 'TestPlan') {
    allowed = ['threadGroup', 'config', 'listener', 'timer', 'preProcessor', 'postProcessor', 'assertion']
  } else if (parentCategory === 'threadGroup') {
    allowed = ['sampler', 'controller', 'config', 'timer', 'preProcessor', 'postProcessor', 'assertion', 'listener']
  } else if (parentCategory === 'controller') {
    allowed = ['sampler', 'controller', 'config', 'timer', 'preProcessor', 'postProcessor', 'assertion', 'listener']
  } else if (parentCategory === 'sampler') {
    allowed = ['config', 'timer', 'preProcessor', 'postProcessor', 'assertion', 'listener']
  } else {
    return false
  }
  
  return allowed.includes(childCategory)
}

// 获取指定类别的可用元素列表
const getAvailableElements = (category) => {
  if (!addElementTarget.value) return []
  const allowedCategories = getAllowedCategories(addElementTarget.value)
  
  // 检查该类别是否被允许
  if (!allowedCategories.includes(category)) {
    return []
  }
  
  return getElementsByCategory(category)
}

// 处理添加元素下拉命令
const handleAddCommand = (command, node) => {
  if (command === 'child') {
    openAddElementDialog(node, 'child')
  } else if (command === 'before') {
    openAddElementDialog(node, 'before')
  } else if (command === 'after') {
    openAddElementDialog(node, 'after')
  }
}

// 打开添加元素对话框
const openAddElementDialog = (node, mode = 'child') => {
  insertMode.value = mode
  addElementTarget.value = node
  
  // 根据插入模式确定允许的元素类别
  let allowedCategories = []
  if (mode === 'child') {
    allowedCategories = getAllowedCategories(node)
  } else {
    // before/after: 基于父节点判断
    const result = findParentNode(node)
    if (result && result.parent) {
      allowedCategories = getAllowedCategories(result.parent)
    } else {
      // 根级别，使用 TestPlan 的规则
      allowedCategories = ['threadGroup', 'config', 'listener', 'timer', 'preProcessor', 'postProcessor', 'assertion']
    }
  }
  
  // 设置默认选中第一个允许的类别
  if (allowedCategories.length > 0) {
    addElementTab.value = allowedCategories[0]
  }
  
  addElementDialogVisible.value = true
}

// 生成唯一ID
const generateNodeId = () => {
  return 'node_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now()
}

// 添加元素
const addElement = (elementType) => {
  const meta = ELEMENT_META[elementType]
  if (!meta || !addElementTarget.value) return
  
  // 创建新节点
  const newNode = {
    id: generateNodeId(),
    testclass: elementType,
    testname: meta.label,
    enabled: true,
    properties: {},
    children: [],
    _xmlNode: null // 标记为新节点，序列化时会创建
  }
  
  // 填充默认属性值
  if (meta.properties) {
    meta.properties.forEach(prop => {
      if (prop.defaultValue !== undefined) {
        if (prop.type === 'boolean') {
          newNode.properties[prop.key] = prop.defaultValue === true || prop.defaultValue === 'true'
        } else if (prop.type === 'number' && prop.defaultValue !== '') {
          const normalizedValue = Number(prop.defaultValue)
          newNode.properties[prop.key] = Number.isNaN(normalizedValue) ? prop.defaultValue : normalizedValue
        } else {
          newNode.properties[prop.key] = prop.defaultValue
        }
      }
    })
  }
  
  if (insertMode.value === 'child') {
    // 原有逻辑：追加到 children 末尾
    if (!addElementTarget.value.children) {
      addElementTarget.value.children = []
    }
    addElementTarget.value.children.push(newNode)
  } else {
    // before/after: 找到父节点和目标位置
    const result = findParentNode(addElementTarget.value)
    if (!result || !result.siblings) {
      ElMessage.error('无法找到插入位置')
      addElementDialogVisible.value = false
      return
    }
    const { siblings } = result
    const index = siblings.indexOf(addElementTarget.value)
    if (index < 0) {
      ElMessage.error('目标节点不在父节点的子列表中')
      addElementDialogVisible.value = false
      return
    }
    if (insertMode.value === 'before') {
      siblings.splice(index, 0, newNode)
    } else { // after
      siblings.splice(index + 1, 0, newNode)
    }
  }
  
  // 关闭对话框
  addElementDialogVisible.value = false
  
  // 选中新添加的节点
  focusNode(newNode, true)
  
  // 触发更新
  emitUpdate()
}

// 获取元素图标颜色（用于对话框中的元素卡片）
const getElementIconColor = (testclass) => {
  // 复用 getNodeIconColor 的逻辑
  return getNodeIconColor({ testclass })
}

// ========== 拖拽排序相关函数 ==========

// 是否允许拖拽
const allowDrag = (draggingNode) => {
  // TestPlan 不允许拖拽
  return draggingNode.data.testclass !== 'TestPlan'
}

// 是否允许放置
const allowDrop = (draggingNode, dropNode, type) => {
  const dragData = draggingNode.data
  const dropData = dropNode.data
  
  // prev/next: 作为兄弟节点插入，需要检查父节点是否允许该类型
  if (type === 'prev' || type === 'next') {
    // 获取 dropNode 的父节点
    const parent = dropNode.parent
    // el-tree 的根节点 parent 存在但 parent.data 是数组
    if (!parent || !parent.data || Array.isArray(parent.data)) {
      // 根级别：只有 TestPlan 允许的子元素可以放在这里
      const dragCategory = getElementCategory(dragData.testclass)
      return ['threadGroup', 'config', 'listener', 'timer', 'preProcessor', 'postProcessor', 'assertion'].includes(dragCategory)
    }
    return isAllowedChild(parent.data.testclass, dragData.testclass)
  }
  
  // inner: 放入目标节点内部
  if (type === 'inner') {
    // 叶子节点不接受子元素
    if (isLeafElement(dropData.testclass)) return false
    return isAllowedChild(dropData.testclass, dragData.testclass)
  }
  
  return false
}

// 处理拖拽放置
const handleDrop = (draggingNode, dropNode, dropType, ev) => {
  // el-tree 已经自动更新了数据结构
  // 只需触发序列化更新
  emitUpdate()
}

// ========== 节点复制与移动相关函数 ==========

// 查找节点的父节点
const findParentNode = (targetNode) => {
  const search = (nodes, parent) => {
    for (const node of nodes) {
      if (node === targetNode) {
        return { parent, siblings: nodes }
      }
      if (node.children && node.children.length > 0) {
        const result = search(node.children, node)
        if (result) return result
      }
    }
    return null
  }
  return search(treeData.value, null)
}

// 复制节点
const copyNode = (node) => {
  const result = findParentNode(node)
  if (!result || !result.parent) return

  const { parent, siblings } = result

  // 深拷贝节点（包括所有子节点）
  const cloned = JSON.parse(JSON.stringify(node))

  // 生成新的 ID（递归处理所有子节点）
  const regenerateIds = (n) => {
    n.id = generateNodeId()
    if (n.children) {
      n.children.forEach(regenerateIds)
    }
  }
  regenerateIds(cloned)

  // 修改名称，添加 "_copy" 后缀
  if (cloned.testname) {
    cloned.testname = cloned.testname + '_copy'
  }
  if (cloned.attrs && cloned.attrs.testname) {
    cloned.attrs.testname = cloned.testname
  }

  // 在同级节点中，在当前节点之后插入副本
  const index = siblings.indexOf(node)
  if (index >= 0) {
    siblings.splice(index + 1, 0, cloned)
  } else {
    siblings.push(cloned)
  }

  // 选中新复制的节点
  focusNode(cloned, true)

  // 触发更新
  emitUpdate()
}

// 获取节点在同级中的索引
const getNodeIndex = (node) => {
  const result = findParentNode(node)
  if (!result) return -1
  return result.siblings.indexOf(node)
}

// 判断节点是否是第一个子节点
const isFirstNode = (node) => {
  return getNodeIndex(node) === 0
}

// 判断节点是否是最后一个子节点
const isLastNode = (node) => {
  const result = findParentNode(node)
  if (!result) return true
  return getNodeIndex(node) === result.siblings.length - 1
}

// 上移节点
const moveNodeUp = (node) => {
  const result = findParentNode(node)
  if (!result) return

  const { siblings } = result
  const index = siblings.indexOf(node)
  if (index <= 0) return

  // 使用 splice 实现交换
  siblings.splice(index, 1)
  siblings.splice(index - 1, 0, node)

  // 触发更新
  emitUpdate()
  focusNode(node)
}

// 下移节点
const moveNodeDown = (node) => {
  const result = findParentNode(node)
  if (!result) return

  const { siblings } = result
  const index = siblings.indexOf(node)
  if (index < 0 || index >= siblings.length - 1) return

  // 使用 splice 实现交换
  siblings.splice(index, 1)
  siblings.splice(index + 1, 0, node)

  // 触发更新
  emitUpdate()
  focusNode(node)
}
</script>

<style scoped lang="scss">
.jmx-tree-editor {
  display: flex;
  height: 100%;
  min-height: 500px;
  gap: 0;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.06);
}

// 左侧树面板
.tree-panel {
  width: clamp(300px, 29%, 470px);
  min-width: 300px;
  max-width: 470px;
  border-right: 1px solid rgba(255, 255, 255, 0.06);
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  border-radius: var(--radius-lg) 0 0 var(--radius-lg);

  .panel-header {
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    flex-shrink: 0;
    background: rgba(255, 255, 255, 0.02);
  }

  .panel-header-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
  }

  .panel-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--text-primary);
  }
}

.tree-search-input {
  :deep(.el-input__wrapper) {
    background: rgba(255, 255, 255, 0.04);
    box-shadow: none;
    border: 1px solid rgba(255, 255, 255, 0.08);
  }
}

// 搜索行（包含展开/折叠按钮）
.search-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;

  .tree-search-input {
    flex: 1;
  }

  .expand-collapse-group {
    flex-shrink: 0;
  }
}

// 面板头部操作区
.panel-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.shortcut-help-btn {
  padding: 4px;
  height: auto;
  color: var(--text-secondary);

  &:hover {
    color: var(--accent-blue);
  }
}

// 快捷添加工具栏
.quick-add-toolbar {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  margin-bottom: 12px;
  border-radius: 12px;
  background: rgba(10, 132, 255, 0.06);
  border: 1px solid rgba(10, 132, 255, 0.12);

  .quick-add-label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 500;
    color: var(--accent-blue);

    .el-icon {
      font-size: 14px;
    }
  }

  .quick-add-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .quick-add-btn {
    height: 28px;
    padding: 0 10px;
    border-radius: 6px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    background: rgba(255, 255, 255, 0.05);
    color: var(--text-secondary);
    font-size: 12px;
    display: flex;
    align-items: center;
    gap: 4px;

    &:hover {
      background: rgba(255, 255, 255, 0.1);
      border-color: rgba(255, 255, 255, 0.15);
      color: var(--text-primary);
    }

    .el-icon {
      font-size: 14px;
    }

    .quick-add-text {
      max-width: 80px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
}

// 多选批量操作工具栏
.batch-toolbar {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  margin-bottom: 12px;
  border-radius: 12px;
  background: rgba(10, 132, 255, 0.1);
  border: 1px solid rgba(10, 132, 255, 0.2);

  .batch-info {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .batch-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;

    .el-button {
      flex: 1;
      min-width: 80px;
    }
  }
}

.tree-toolbar {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 12px;
}

.tree-toolbar-row,
.tree-selection-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tree-tool-btn,
.tree-action-chip {
  height: 30px;
  padding: 0 11px;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-secondary);
}

.tree-toolbar-row :deep(.el-button),
.tree-selection-actions :deep(.el-button) {
  margin-left: 0;
}

.tree-action-chip--primary {
  background: rgba(10, 132, 255, 0.14);
  border-color: rgba(10, 132, 255, 0.22);
  color: #9fd0ff;
}

.tree-action-chip--danger {
  background: rgba(255, 69, 58, 0.1);
  border-color: rgba(255, 69, 58, 0.18);
  color: #ff8e88;
}

.tree-selection-toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 8px;
  padding: 10px;
  border-radius: 14px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(255, 255, 255, 0.03);
}

.tree-selection-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tree-selection-label {
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--accent-blue);
}

.tree-selection-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-content {
  flex: 1;
  overflow: auto;
  padding: 8px;
}

// 右侧属性面板
.property-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;

  .panel-header {
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    flex-shrink: 0;
    background: rgba(255, 255, 255, 0.02);
  }

  .panel-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--text-primary);
  }
}

.property-content {
  flex: 1;
  overflow: auto;
}

// 属性面板头部
.property-header {
  padding: 20px 20px 0;

  .property-context {
    margin-bottom: 16px;
  }

  .property-context-label {
    font-size: 11px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--accent-blue);
    margin-bottom: 6px;
  }

  .property-context-path {
    font-size: 12px;
    color: var(--text-secondary);
    line-height: 1.6;
  }

  .property-title-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .property-icon {
    font-size: 32px;
    flex-shrink: 0;
  }

  .property-title-info {
    flex: 1;
    min-width: 0;

    .property-name-input {
      :deep(.el-input__wrapper) {
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        box-shadow: none;
        
        &:hover {
          border-color: rgba(255, 255, 255, 0.2);
        }
        
        &.is-focus {
          border-color: var(--accent-blue);
        }
      }
      
      :deep(.el-input__inner) {
        font-size: 16px;
        font-weight: 600;
        color: var(--text-primary);
      }
    }

    .property-type {
      font-size: 12px;
      color: var(--text-secondary);
      margin-top: 4px;
      display: block;
    }
  }

  .property-actions {
    flex-shrink: 0;
    
    :deep(.el-switch__label) {
      color: var(--text-secondary);
      font-size: 12px;
    }
  }

  .property-summary-row {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    margin-top: 16px;
  }

  .summary-chip {
    min-width: 120px;
    padding: 10px 12px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.06);
  }

  .summary-chip-wide {
    flex: 1;
    min-width: 220px;
  }

  .summary-chip-label {
    display: block;
    font-size: 11px;
    color: var(--text-secondary);
    margin-bottom: 4px;
  }

  .summary-chip-value {
    display: block;
    color: var(--text-primary);
    font-size: 13px;
    line-height: 1.5;
    word-break: break-word;
  }

  :deep(.el-divider) {
    margin: 16px 0 0;
  }
}

// 属性表单
.property-form {
  padding: 0 20px 20px;

  :deep(.el-form-item) {
    margin-bottom: 18px;

    .el-form-item__label {
      font-size: 13px;
      color: var(--text-secondary);
      font-weight: 500;
      padding-bottom: 6px;
    }
  }
}

// 脚本编辑 textarea 特殊样式
.script-textarea {
  :deep(.el-textarea__inner) {
    font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
    font-size: 12px;
    line-height: 1.6;
    background: rgba(0, 0, 0, 0.3);
    color: #c9d1d9;
    border-color: rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    padding: 12px;
  }
}

// 空状态
.empty-panel {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-guide {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 40px;

  .empty-icon {
    font-size: 48px;
    color: var(--accent-blue);
    opacity: 0.5;
    margin-bottom: 16px;
  }

  .empty-title {
    font-size: 18px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 8px;
  }

  .empty-desc {
    font-size: 13px;
    color: var(--text-secondary);
    margin-bottom: 24px;
    max-width: 280px;
    line-height: 1.6;
  }

  .empty-tips {
    display: flex;
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;

    .tip-item {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 13px;
      color: var(--text-secondary);

      .el-icon {
        font-size: 16px;
        color: var(--accent-blue);
      }
    }
  }
}

/* 树样式 */
.jmx-tree {
  background: transparent;
}

// 树节点样式
.tree-node {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 4px;
  padding: 8px 8px 7px;
  border-radius: 12px;
  transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
  cursor: pointer;
  width: 100%;
  box-sizing: border-box;
  overflow: hidden;
  border: 1px solid transparent;
  min-height: 0;
  position: relative;

  &:hover,
  &.is-selected {
    background: rgba(255, 255, 255, 0.03);
    border-color: rgba(255, 255, 255, 0.08);
  }

  &.is-selected {
    background: rgba(10, 132, 255, 0.1);
    border-color: rgba(10, 132, 255, 0.22);
    box-shadow: inset 3px 0 0 rgba(10, 132, 255, 0.88);
  }

  &.is-multi-selected {
    background: rgba(10, 132, 255, 0.15);
    border-color: rgba(10, 132, 255, 0.3);
    box-shadow: inset 0 0 0 1px rgba(10, 132, 255, 0.4);
  }

  &.is-disabled {
    opacity: 0.45;

    .node-name {
      text-decoration: line-through;
    }
  }

  .node-main {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .node-top-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .node-title-wrap {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    flex: 1 1 180px;
  }

  .node-icon {
    font-size: 16px;
    flex-shrink: 0;
  }

  .node-label {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .node-badges {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-start;
    gap: 4px;
    flex: 1 1 auto;
    min-width: 0;
  }

  .node-top-side {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 6px;
    min-width: 0;
    flex: 0 0 auto;
  }

  .node-type-tag {
    flex-shrink: 0;
    font-size: 11px;
    height: 18px;
    line-height: 16px;
    padding: 0 5px;
    border-radius: 999px;
  }

  .node-meta-row {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    padding-left: 24px;
  }

  .node-summary {
    color: rgba(255, 255, 255, 0.55);
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
  }

  .node-summary.is-empty {
    color: rgba(255, 255, 255, 0.3);
    font-style: italic;
  }

  .node-menu {
    flex-shrink: 0;
  }

  .node-menu-btn {
    width: 24px;
    height: 24px;
    padding: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.06);
    background: rgba(255, 255, 255, 0.04);
    color: var(--text-secondary);
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.18s ease, background-color 0.18s ease, color 0.18s ease;

    &:hover {
      color: var(--text-primary);
      background: rgba(255, 255, 255, 0.08);
    }
  }

  &:hover .node-menu-btn,
  &.is-selected .node-menu-btn {
    opacity: 1;
  }
}

/* 添加元素对话框样式 */
.element-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  padding: 10px 0;
  max-height: 400px;
  overflow-y: auto;
}

.element-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 8px;
  cursor: pointer;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.02);
  transition: all 0.2s ease;
  
  &:hover {
    background: rgba(64, 158, 255, 0.1);
    border-color: var(--accent-blue, #409eff);
    transform: translateY(-1px);
  }
  
  .element-icon {
    font-size: 20px;
    flex-shrink: 0;
  }
  
  .element-label {
    font-size: 13px;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.empty-category {
  grid-column: span 3;
  text-align: center;
  padding: 40px;
  color: var(--text-secondary);
  font-size: 14px;
}

.element-tabs {
  :deep(.el-tabs__nav-wrap::after) {
    background-color: rgba(255, 255, 255, 0.06);
  }
  
  :deep(.el-tabs__item) {
    color: var(--text-secondary);
    
    &.is-active {
      color: var(--accent-blue);
    }
    
    &.is-disabled {
      opacity: 0.4;
    }
  }
  
  :deep(.el-tabs__active-bar) {
    background-color: var(--accent-blue);
  }
}

/* 键值对列表样式 */
.key-value-list {
  width: 100%;
}

.key-value-table {
  background: transparent;
  
  :deep(.el-table__header-wrapper) {
    th {
      background: rgba(255, 255, 255, 0.03);
      color: var(--text-secondary);
      font-weight: 500;
      font-size: 12px;
    }
  }
  
  :deep(.el-table__body-wrapper) {
    td {
      background: transparent;
    }
  }
  
  :deep(.el-table__row) {
    background: transparent;
    
    &:hover > td {
      background: rgba(255, 255, 255, 0.02) !important;
    }
  }
}

.add-btn {
  margin-top: 8px;
}

/* 字符串列表样式 */
.string-list-editor {
  width: 100%;
}

.string-list-hint {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 8px;
  opacity: 0.7;
}

/* 线程调度配置编辑器 */
.thread-schedule-editor {
  .schedule-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    
    .schedule-title {
      font-size: 14px;
      font-weight: 500;
      color: #e0e0e0;
    }
  }
  
  .schedule-table {
    .el-input-number {
      width: 100%;
    }
  }
}

/* 原始 XML 面板 */
.raw-xml-panel {
  padding: 0 20px 20px;

  .raw-xml-label {
    font-size: 12px;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }
}

.raw-xml-textarea {
  :deep(.el-textarea__inner) {
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.08);
    color: var(--text-secondary);
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    font-size: 12px;
    border-radius: 8px;
  }
}

/* Element Plus 树覆盖样式 */
:deep(.el-tree) {
  background: transparent;
  color: var(--text-primary);
}

:deep(.el-tree-node__content) {
  height: auto;
  padding: 0;
  border-radius: var(--radius-sm);
  align-items: stretch;
}

:deep(.el-tree-node__content:hover) {
  background: transparent;
}

:deep(.el-tree-node__content > .tree-node) {
  flex: 1;
  min-width: 0;
}

:deep(.el-tree-node:focus > .el-tree-node__content) {
  background: transparent;
}

:deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: transparent;
}

:deep(.el-tree-node__expand-icon) {
  color: var(--text-secondary);
}

:deep(.el-tree-node__expand-icon.is-leaf) {
  color: transparent;
}

/* 拖拽时的样式 */
:deep(.el-tree-node.is-drop-inner > .el-tree-node__content) {
  background-color: rgba(64, 158, 255, 0.15) !important;
  border-radius: 4px;
}

:deep(.el-tree__drop-indicator) {
  height: 2px !important;
  background-color: #409eff !important;
}

/* 拖拽时被拖动节点的样式 */
:deep(.el-tree-node.is-dragging > .el-tree-node__content) {
  opacity: 0.5;
}

/* 分割线 */
:deep(.el-divider) {
  border-color: rgba(255, 255, 255, 0.06);
}

/* 快捷键提示对话框样式 */
.shortcut-dialog {
  .shortcut-list {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .shortcut-section {
    h4 {
      font-size: 13px;
      font-weight: 600;
      color: var(--text-primary);
      margin: 0 0 12px 0;
      padding-bottom: 8px;
      border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    }
  }

  .shortcut-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 0;
    font-size: 13px;
    color: var(--text-secondary);

    kbd {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 24px;
      height: 24px;
      padding: 0 8px;
      font-family: inherit;
      font-size: 12px;
      font-weight: 500;
      color: var(--text-primary);
      background: rgba(255, 255, 255, 0.08);
      border: 1px solid rgba(255, 255, 255, 0.12);
      border-radius: 6px;
      box-shadow: 0 2px 0 rgba(0, 0, 0, 0.2);
    }
  }
}

@media (max-width: 1400px) {
  .tree-node .node-meta-row {
    padding-left: 0;
  }
}

@media (max-width: 1100px) {
  .jmx-tree-editor {
    flex-direction: column;
  }

  .tree-panel {
    width: 100%;
    max-width: none;
    min-width: 0;
    max-height: 42%;
    border-right: none;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: var(--radius-lg) var(--radius-lg) 0 0;
  }

  .property-panel {
    min-height: 380px;
  }
}

@media (hover: none) {
  .tree-node .node-menu-btn {
    opacity: 1;
  }
}

@media (max-width: 720px) {
  .property-header .property-title-row {
    flex-wrap: wrap;
  }

  .property-header .property-actions {
    width: 100%;
  }

  .tree-selection-toolbar {
    padding: 8px;
  }

  .tree-node .node-meta-row {
    padding-left: 0;
  }

  .tree-node .node-top-row {
    flex-wrap: wrap;
  }
}
</style>
