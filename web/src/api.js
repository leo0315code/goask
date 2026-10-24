// 统一的 API 请求封装：携带 JWT、解析统一响应结构、401 自动清理登录态。
const BASE = '/api/v1'

export class ApiError extends Error {
  constructor(code, message) {
    super(message)
    this.code = code
  }
}

export async function api(method, path, body) {
  const headers = { 'Content-Type': 'application/json' }
  const token = localStorage.getItem('goask_token')
  if (token) headers.Authorization = 'Bearer ' + token

  let res
  try {
    res = await fetch(BASE + path, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiError(-1, '网络错误，请稍后重试')
  }

  let json
  try {
    json = await res.json()
  } catch {
    throw new ApiError(50000, '服务器响应异常')
  }
  if (json.code !== 0) {
    if (json.code === 40100) {
      localStorage.removeItem('goask_token')
      window.dispatchEvent(new CustomEvent('goask:unauthorized'))
    }
    throw new ApiError(json.code, json.message || '请求失败')
  }
  return json.data
}

// 文件上传（multipart/form-data）
export async function upload(path, formData) {
  const headers = {}
  const token = localStorage.getItem('goask_token')
  if (token) headers.Authorization = 'Bearer ' + token

  let res
  try {
    res = await fetch(BASE + path, { method: 'POST', headers, body: formData })
  } catch {
    throw new ApiError(-1, '网络错误，请稍后重试')
  }
  let json
  try {
    json = await res.json()
  } catch {
    throw new ApiError(50000, '服务器响应异常')
  }
  if (json.code !== 0) {
    if (json.code === 40100) {
      localStorage.removeItem('goask_token')
      window.dispatchEvent(new CustomEvent('goask:unauthorized'))
    }
    throw new ApiError(json.code, json.message || '上传失败')
  }
  return json.data
}

// 分页查询串
export function pageParams(page, pageSize = 15) {
  return `page=${page}&page_size=${pageSize}`
}
