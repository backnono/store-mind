// ============================================================
// 环境配置 — 可随渠道与部署环境切换
// ============================================================

/** 当前渠道 */
export const CHANNEL = 'miniapp' as const

/** API 基础地址 — 开发/测试时可覆盖为内网穿透地址 */
export const API_BASE_URL = 'http://localhost:8080'

/** 请求超时时间 (ms) */
export const REQUEST_TIMEOUT = 12000
