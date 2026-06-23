// ============================================================
// 存储 key 生成 — 纯函数，不依赖 Taro
// ============================================================

/** 存储 key 前缀 */
const PREFIX = 'store-mind'

/**
 * 生成带 store_id 的存储 key
 * e.g. sessionKey(1) → 'store-mind:session:1'
 */
export function sessionKey(storeId: number): string {
  return `${PREFIX}:session:${storeId}`
}

export function messagesKey(storeId: number): string {
  return `${PREFIX}:messages:${storeId}`
}

export function draftKey(storeId: number): string {
  return `${PREFIX}:draft:${storeId}`
}
