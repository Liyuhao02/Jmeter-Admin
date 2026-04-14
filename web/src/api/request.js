import axios from 'axios'
import { ElMessage } from 'element-plus'

const request = axios.create({
  baseURL: '',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求去重 - 存储正在进行的请求
const pendingRequests = new Map()
const BUSINESS_CODES_WITH_LOCAL_HANDLING = new Set([40901])
let lastErrorMessage = ''
let lastErrorAt = 0

// 生成请求唯一标识
function generateRequestKey(config) {
  return `${config.method}:${config.url}:${JSON.stringify(config.params || {})}:${JSON.stringify(config.data || {})}`
}

function shouldSkipGlobalMessage(config = {}, code) {
  const customSuppressedCodes = Array.isArray(config.suppressBusinessCodes) ? config.suppressBusinessCodes : []
  return Boolean(
    config.silent ||
    config.skipGlobalErrorHandler ||
    BUSINESS_CODES_WITH_LOCAL_HANDLING.has(code) ||
    customSuppressedCodes.includes(code)
  )
}

function showErrorOnce(message) {
  const normalized = String(message || '').trim() || '请求失败'
  const now = Date.now()
  if (normalized === lastErrorMessage && now - lastErrorAt < 1400) {
    return
  }
  lastErrorMessage = normalized
  lastErrorAt = now
  ElMessage.error(normalized)
}

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    const key = generateRequestKey(config)
    const previousController = pendingRequests.get(key)
    if (previousController) {
      previousController.abort('Duplicate request')
      pendingRequests.delete(key)
    }

    const controller = new AbortController()
    config.signal = controller.signal
    config.__requestKey = key
    pendingRequests.set(key, controller)
    return config
  },
  (error) => {
    if (!error?.config?.silent && !error?.config?.skipGlobalErrorHandler) {
      showErrorOnce('请求发送失败')
    }
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    const key = response.config?.__requestKey || generateRequestKey(response.config)
    pendingRequests.delete(key)
    
    const res = response.data
    if (res.code !== 0) {
      if (!shouldSkipGlobalMessage(response.config, res.code)) {
        showErrorOnce(res.message || '请求失败')
      }
      const businessError = new Error(res.message || '请求失败')
      businessError.response = response
      businessError.config = response.config
      businessError.isBusinessError = true
      return Promise.reject(businessError)
    }
    return res
  },
  (error) => {
    if (error.config) {
      const key = error.config.__requestKey || generateRequestKey(error.config)
      pendingRequests.delete(key)
    }

    if (
      axios.isCancel(error) ||
      error.code === 'ERR_CANCELED' ||
      error.message === 'Duplicate request' ||
      error?.config?.signal?.aborted
    ) {
      return Promise.reject(error)
    }

    const responseCode = error.response?.data?.code
    if (!shouldSkipGlobalMessage(error.config, responseCode)) {
      if (error.code === 'ECONNABORTED') {
        showErrorOnce('请求超时，请检查网络')
      } else if (!error.response) {
        showErrorOnce('网络连接失败，请检查服务是否启动')
      } else if (error.response.status >= 500) {
        showErrorOnce('服务器内部错误')
      } else if (error.response.status === 404) {
        showErrorOnce('请求的资源不存在')
      } else if (error.response.status === 403) {
        showErrorOnce('没有权限执行此操作')
      } else if (error.response.status === 401) {
        showErrorOnce('未登录或登录已过期')
      } else {
        const message = error.response?.data?.message || error.message || '网络错误'
        showErrorOnce(message)
      }
    }
    return Promise.reject(error)
  }
)

// 上传专用函数（支持进度回调）
export function uploadWithProgress(url, formData, onProgress) {
  return request.post(url, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: (e) => {
      if (onProgress && e.total) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    }
  })
}

export default request
