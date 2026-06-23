// ============================================================
// API 错误规范化 & Taro.request 封装
// ============================================================
import Taro from '@tarojs/taro'
import { API_BASE_URL, REQUEST_TIMEOUT } from '@/utils/env'

/** 规范化后的 API 错误 */
export interface ApiError {
  code: 'network' | 'timeout' | 'bad_request' | 'server_error' | 'unknown'
  message: string
  retryable: boolean
}

/**
 * 把 Taro 请求错误转成用户可读的 ApiError
 */
export function normalizeError(err: unknown): ApiError {
  if (typeof err === 'object' && err !== null) {
    const e = err as Record<string, unknown>
    const errMsg = String(e.errMsg ?? e.message ?? '')

    if (errMsg.includes('timeout') || errMsg.includes('超时')) {
      return { code: 'timeout', message: '请求超时，请稍后重试', retryable: true }
    }
    if (errMsg.includes('fail') && errMsg.includes('url')) {
      return { code: 'network', message: '网络连接失败，请检查网络', retryable: true }
    }
    if (errMsg.includes('abort')) {
      return { code: 'network', message: '请求已取消', retryable: true }
    }
  }
  return { code: 'unknown', message: '出错了，请稍后重试', retryable: true }
}

/**
 * 把 HTTP 状态码映射为 ApiError
 */
export function httpError(status: number, body?: string): ApiError {
  let message = '服务器繁忙，请稍后重试'
  let code: ApiError['code'] = 'server_error'
  let retryable = true

  if (status === 400) {
    code = 'bad_request'
    message = '请求参数有误'
    retryable = false
  } else if (status === 401 || status === 403) {
    code = 'bad_request'
    message = '暂无访问权限'
    retryable = false
  } else if (status === 404) {
    code = 'bad_request'
    message = '请求的资源不存在'
    retryable = false
  } else if (status >= 500) {
    code = 'server_error'
    message = '服务器繁忙，请稍后重试'
    retryable = true
  }

  // 尝试从 body 中提取更友好的文案
  if (body) {
    try {
      const parsed = JSON.parse(body)
      if (parsed?.error && typeof parsed.error === 'string') {
        message = parsed.error
      }
    } catch {
      // ignore parse error
    }
  }

  return { code, message, retryable }
}

/**
 * 发起 GET 请求
 */
export async function get<T = unknown>(
  path: string,
  params?: Record<string, string | number | undefined>,
): Promise<T> {
  const query = buildQuery(params)
  const url = `${API_BASE_URL}${path}${query ? `?${query}` : ''}`

  const res = await Taro.request({
    url,
    method: 'GET',
    timeout: REQUEST_TIMEOUT,
    header: { 'Content-Type': 'application/json' },
  })

  if (res.statusCode < 200 || res.statusCode >= 300) {
    throw httpError(res.statusCode, typeof res.data === 'string' ? res.data : JSON.stringify(res.data))
  }

  return res.data as T
}

/**
 * 发起 POST 请求
 */
export async function post<T = unknown>(
  path: string,
  body?: Record<string, unknown>,
): Promise<T> {
  const url = `${API_BASE_URL}${path}`

  const res = await Taro.request({
    url,
    method: 'POST',
    data: body,
    timeout: REQUEST_TIMEOUT,
    header: { 'Content-Type': 'application/json' },
  })

  if (res.statusCode < 200 || res.statusCode >= 300) {
    throw httpError(res.statusCode, typeof res.data === 'string' ? res.data : JSON.stringify(res.data))
  }

  return res.data as T
}

function buildQuery(params?: Record<string, string | number | undefined>): string {
  if (!params) return ''
  const parts: string[] = []
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null) {
      parts.push(`${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
    }
  }
  return parts.join('&')
}
