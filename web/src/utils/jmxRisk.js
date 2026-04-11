import { extractCSVDataSetFilesFromXML } from './jmxParser'

const FILE_PROP_NAMES = new Set([
  'File.path',
  'filename',
  'scriptFile',
  'BeanShellSampler.filename',
  'BeanShellPreProcessor.filename',
  'BeanShellPostProcessor.filename',
  'BSFSampler.filename',
  'BSFPreProcessor.filename',
  'BSFPostProcessor.filename',
  'JSR223Sampler.filename',
  'JSR223PreProcessor.filename',
  'JSR223PostProcessor.filename',
  'JSR223Assertion.filename',
  'JSR223Listener.filename',
  'HTTPSampler.file.path',
  'HTTPFileArg.path'
])

const PLUGIN_PATTERNS = [
  { pattern: 'kg.apc.', label: 'JMeter Plugins (kg.apc / jp@gc)' },
  { pattern: 'UltimateThreadGroup', label: 'Ultimate Thread Group' },
  { pattern: 'jp@gc', label: 'JMeter Plugins (jp@gc)' }
]

const isDynamicPath = (value) => value.includes('${') || value.startsWith('__P(')
const isUrlPath = (value) => /^https?:\/\//i.test(value)
const isAbsolutePath = (value) => /^(\/|[A-Za-z]:[\\/])/.test(value)

const unique = (values) => [...new Set(values.filter(Boolean))]

function parseXML(xmlContent) {
  if (typeof DOMParser === 'undefined') {
    return null
  }
  const parser = new DOMParser()
  const doc = parser.parseFromString(xmlContent, 'application/xml')
  if (doc.querySelector('parsererror')) {
    return null
  }
  return doc
}

export function analyzeJmxSaveRisks(xmlContent, attachedFiles = []) {
  const report = {
    blockingIssues: [],
    warnings: [],
    summary: []
  }

  if (!xmlContent) {
    return report
  }

  const attachedSet = new Set(attachedFiles.map((item) => item.file_name || item).filter(Boolean))
  const csvFiles = unique(extractCSVDataSetFilesFromXML(xmlContent).map((item) => item.trim()))
  const doc = parseXML(xmlContent)
  const fileDependencies = []
  const pluginDependencies = []
  const absolutePaths = []
  const csvDuplicateBasenames = []
  const csvHeaderConflicts = []
  let hasTransactionController = false
  let hasCriticalSectionController = false

  if (doc) {
    const csvRefs = []
    const csvNodes = [...doc.querySelectorAll('CSVDataSet')]
    csvNodes.forEach((node) => {
      const filename = node.querySelector('stringProp[name="filename"]')?.textContent?.trim()
      if (!filename) return
      const ignoreFirstLineValue = node.querySelector('boolProp[name="ignoreFirstLine"]')?.textContent?.trim()
      csvRefs.push({
        filename,
        baseName: filename.split(/[\\/]/).pop(),
        ignoreFirstLine: String(ignoreFirstLineValue).toLowerCase() === 'true'
      })
    })

    const basenameMap = new Map()
    const configMap = new Map()
    csvRefs.forEach((ref) => {
      if (!basenameMap.has(ref.baseName)) basenameMap.set(ref.baseName, new Set())
      basenameMap.get(ref.baseName).add(ref.filename)

      if (!configMap.has(ref.filename)) configMap.set(ref.filename, new Set())
      configMap.get(ref.filename).add(ref.ignoreFirstLine ? 'true' : 'false')
    })

    basenameMap.forEach((owners, baseName) => {
      if (owners.size > 1) {
        csvDuplicateBasenames.push(`${baseName}（${[...owners].join('、')}）`)
      }
    })

    configMap.forEach((values, filename) => {
      if (values.size > 1) {
        csvHeaderConflicts.push(filename)
      }
    })

    const stringProps = [...doc.querySelectorAll('stringProp')]
    stringProps.forEach((node) => {
      const name = node.getAttribute('name')?.trim()
      const value = node.textContent?.trim()
      if (!name || !value || !FILE_PROP_NAMES.has(name) || isDynamicPath(value) || isUrlPath(value)) {
        return
      }
      const baseName = value.split(/[\\/]/).pop()
      if (csvFiles.some((csv) => csv.split(/[\\/]/).pop() === baseName)) {
        return
      }
      fileDependencies.push(value)
      if (isAbsolutePath(value)) {
        absolutePaths.push(value)
      }
    })

    hasTransactionController = Boolean(doc.querySelector('[testclass="TransactionController"]'))
    hasCriticalSectionController = Boolean(doc.querySelector('[testclass="CriticalSectionController"]'))
  }

  PLUGIN_PATTERNS.forEach(({ pattern, label }) => {
    if (xmlContent.includes(pattern)) {
      pluginDependencies.push(label)
    }
  })

  const missingDependencies = unique([...csvFiles, ...fileDependencies].filter((dep) => {
    if (isAbsolutePath(dep) || isDynamicPath(dep) || isUrlPath(dep)) {
      return false
    }
    return !attachedSet.has(dep.split(/[\\/]/).pop())
  }))

  if (missingDependencies.length > 0) {
    report.blockingIssues.push({
      title: '存在缺失依赖文件',
      detail: `以下文件当前未在脚本关联文件中找到：${missingDependencies.join('、')}`
    })
  }

  if (csvHeaderConflicts.length > 0) {
    report.blockingIssues.push({
      title: 'CSV 首行配置存在冲突',
      detail: `以下 CSV 在多个 CSVDataSet 中的 ignoreFirstLine 配置不一致：${csvHeaderConflicts.join('、')}。这会导致分布式拆分结果不可预期。`
    })
  }

  if (absolutePaths.length > 0) {
    report.warnings.push({
      title: '脚本包含绝对路径引用',
      detail: `检测到 ${absolutePaths.length} 处绝对路径，这在分布式执行或环境迁移时非常容易失效。`
    })
  }

  if (!hasTransactionController) {
    report.warnings.push({
      title: '未检测到事务控制器',
      detail: '当前脚本没有 Transaction Controller，执行结果中的 TPS 将退化为请求/s 口径。'
    })
  }

  if (hasCriticalSectionController) {
    report.warnings.push({
      title: '存在临界区控制器',
      detail: 'Critical Section Controller 在高并发和分布式下容易造成锁竞争或线程提前释放，请重点关注执行告警。'
    })
  }

  if (pluginDependencies.length > 0) {
    report.warnings.push({
      title: '存在第三方插件依赖',
      detail: `检测到插件组件：${unique(pluginDependencies).join('、')}。执行前请确保 Master 和所有 Slave 安装完全一致的插件版本。`
    })
  }

  if (csvDuplicateBasenames.length > 0) {
    report.warnings.push({
      title: '存在同名 CSV 文件',
      detail: `检测到多个不同路径下的同名 CSV：${csvDuplicateBasenames.join('；')}。系统会在执行期自动重命名分发，但建议你在脚本层保持命名清晰。`
    })
  }

  if (csvFiles.length > 1) {
    report.warnings.push({
      title: '存在多 CSV 输入源',
      detail: '如果这些 CSV 在业务上需要按行一一对应，请确保它们的数据量和拆分策略一致，否则分布式执行后不同节点拿到的数据区间可能不再严格对齐。'
    })
  }

  if (csvFiles.length > 0) {
    report.summary.push({ label: 'CSV 依赖', count: csvFiles.length })
  }
  if (fileDependencies.length > 0) {
    report.summary.push({ label: '文件依赖', count: unique(fileDependencies).length })
  }
  if (pluginDependencies.length > 0) {
    report.summary.push({ label: '插件依赖', count: unique(pluginDependencies).length })
  }
  if (missingDependencies.length > 0) {
    report.summary.push({ label: '缺失依赖', count: missingDependencies.length })
  }

  return report
}
