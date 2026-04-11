import request from './request'

export const executionApi = {
  // 获取执行统计
  getStats() {
    return request.get('/api/executions/stats')
  },

  // 获取执行列表
  getList(params) {
    return request.get('/api/executions', { params })
  },

  // 创建执行（启动测试）
  create(data, options = {}) {
    return request.post('/api/executions', data, options)
  },

  // 获取执行详情
  getDetail(id, options = {}) {
    return request.get(`/api/executions/${id}`, options)
  },

  // 获取执行中的实时指标
  getLiveMetrics(id, options = {}) {
    return request.get(`/api/executions/${id}/live-metrics`, options)
  },

  // 获取执行节点的实时系统指标
  getNodeMetrics(id, options = {}) {
    return request.get(`/api/executions/${id}/node-metrics`, options)
  },

  // 停止执行
  stop(id) {
    return request.post(`/api/executions/${id}/stop`)
  },

  // 删除执行记录
  delete(id) {
    return request.delete(`/api/executions/${id}`)
  },

  // 获取错误分析
  getErrors(id, options = {}) {
    return request.get(`/api/executions/${id}/errors`, options)
  },

  // 下载 JTL 结果文件
  downloadJTL(id) {
    const url = `/api/executions/${id}/download/jtl`
    this.triggerDownload(url)
  },

  // 下载 HTML 报告（ZIP）
  downloadReport(id) {
    const url = `/api/executions/${id}/download/report`
    this.triggerDownload(url)
  },

  // 下载错误记录（CSV）
  downloadErrors(id) {
    const url = `/api/executions/${id}/download/errors`
    this.triggerDownload(url)
  },

  // 下载完整结果（ZIP）
  downloadAll(id) {
    const url = `/api/executions/${id}/download/all`
    this.triggerDownload(url)
  },

  // 辅助方法：触发文件下载
  triggerDownload(url) {
    const link = document.createElement('a')
    link.href = url
    link.style.display = 'none'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  },

  // 设置/取消基准线
  setBaseline(id, action = 'set') {
    return request.put(`/api/executions/${id}/baseline`, { action })
  },

  // 对比两次执行
  compareExecutions(id1, id2) {
    return request.get('/api/executions/compare', { params: { id1, id2 } })
  }
}
