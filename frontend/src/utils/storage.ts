// ============================================================
// Taro 本地存储封装 — 带 store_id 分片
// ============================================================
import Taro from '@tarojs/taro'

export { sessionKey, messagesKey, draftKey } from './storageKeys'

/**
 * 存 JSON
 */
export function setItem<T>(key: string, value: T): Promise<void> {
  return Taro.setStorage({ key, data: value }).then(() => undefined)
}

/**
 * 读 JSON
 */
export async function getItem<T>(key: string): Promise<T | null> {
  try {
    const res = await Taro.getStorage({ key })
    return res.data as T
  } catch {
    return null
  }
}

/**
 * 删 key
 */
export function removeItem(key: string): Promise<void> {
  return Taro.removeStorage({ key }).then(() => undefined)
}
