import request from './request'

export const slaveApi = {
  // 获取Slave列表
  getList(params) {
    return request.get('/api/slaves', { params })
  },

  // 添加Slave
  create(data) {
    return request.post('/api/slaves', data)
  },

  // 更新Slave
  update(id, data) {
    return request.put(`/api/slaves/${id}`, data)
  },

  // 删除Slave
  delete(id) {
    return request.delete(`/api/slaves/${id}`)
  },

  // 检测连通性
  checkConnectivity(id) {
    return request.post(`/api/slaves/${id}/check`)
  },

  // 执行前体检
  getPreflight(params) {
    return request.get('/api/slaves/preflight', { params })
  },

  // 获取网络接口列表
  getNetworkInterfaces() {
    return request.get('/api/config/network-interfaces')
  },

  // 获取 Master 主机名配置
  getMasterHostname() {
    return request.get('/api/config/master-hostname')
  },

  // 更新 Master 主机名配置
  updateMasterHostname(hostname) {
    return request.put('/api/config/master-hostname', { master_hostname: hostname })
  },

  // 获取心跳状态
  getHeartbeatStatus(options = {}) {
    return request.get('/api/slaves/heartbeat-status', options)
  }
}
