import request from './request'

export const scriptApi = {
  // 获取脚本列表
  getList(params) {
    const normalizedParams = { ...params }
    if (normalizedParams.pageSize && !normalizedParams.page_size) {
      normalizedParams.page_size = normalizedParams.pageSize
      delete normalizedParams.pageSize
    }
    return request.get('/api/scripts', { params: normalizedParams })
  },

  // 创建脚本（使用 FormData 格式）
  create(formData) {
    return request.post('/api/scripts', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
  },

  // 获取脚本详情
  getDetail(id) {
    return request.get(`/api/scripts/${id}`)
  },

  // 更新脚本
  update(id, data) {
    return request.put(`/api/scripts/${id}`, data)
  },

  // 删除脚本
  delete(id) {
    return request.delete(`/api/scripts/${id}`)
  },

  // 获取脚本内容
  getContent(id) {
    return request.get(`/api/scripts/${id}/content`)
  },

  // 保存脚本内容
  saveContent(id, content) {
    return request.put(`/api/scripts/${id}/content`, { content })
  },

  // 上传文件
  uploadFile(id, file) {
    const formData = new FormData()
    formData.append('files', file)
    return request.post(`/api/scripts/${id}/files`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
  },

  // 删除文件
  deleteFile(id, fileId) {
    return request.delete(`/api/scripts/${id}/files/${fileId}`)
  },

  // 下载脚本主文件
  download(id) {
    const link = document.createElement('a')
    link.href = `/api/scripts/${id}/download`
    link.style.display = 'none'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }
}
