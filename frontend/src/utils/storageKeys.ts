// ============================================================
// 存储 key 生成 — 纯函数，不依赖 Taro
// 新增 session 作用域: messagesKey(storeId, sessionId) / draftKey(storeId, sessionId)
// ============================================================

/** 存储 key 前缀 */
const PREFIX = 'store-mind'

/**
 * 生成带 store_id 的 session key
 * e.g. sessionKey(1) → 'store-mind:session:1'
 */
export function sessionKey(storeId: number): string {
  return `${PREFIX}:session:${storeId}`
}

/**
 * 消息存储 key，支持 session 作用域
 * - sessionId > 0: 'store-mind:messages:1:21'
 * - sessionId = 0: 'store-mind:messages:1:draft-session'（新会话未拿到后端 session_id 时用）
 */
export function messagesKey(storeId: number, sessionId = 0): string {
  return sessionId > 0
    ? `${PREFIX}:messages:${storeId}:${sessionId}`
    : `${PREFIX}:messages:${storeId}:draft-session`
}

/**
 * 草稿存储 key，支持 session 作用域
 */
export function draftKey(storeId: number, sessionId = 0): string {
  return sessionId > 0
    ? `${PREFIX}:draft:${storeId}:${sessionId}`
    : `${PREFIX}:draft:${storeId}:draft-session`
}

/**
 * 会话历史列表 key（按 store_id 分片）
 */
export function sessionHistoryKey(storeId: number): string {
  return `${PREFIX}:session-history:${storeId}`
}
