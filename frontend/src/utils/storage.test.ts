// ============================================================
// 存储 key 生成 — 单元测试（含 session-scoped keys）
// ============================================================
import { describe, it, expect } from 'vitest'
import { sessionKey, messagesKey, draftKey, sessionHistoryKey } from './storageKeys'

describe('sessionKey', () => {
  it('包含 store_id', () => {
    expect(sessionKey(1)).toBe('store-mind:session:1')
  })

  it('不同 store_id 生成不同 key', () => {
    expect(sessionKey(1)).not.toBe(sessionKey(2))
  })

  it('尾部以 store_id 结尾', () => {
    expect(sessionKey(42).endsWith(':42')).toBe(true)
  })
})

describe('messagesKey', () => {
  it('包含 store_id', () => {
    expect(messagesKey(5)).toBe('store-mind:messages:5:draft-session')
  })

  it('支持 session 作用域', () => {
    expect(messagesKey(1, 21)).toBe('store-mind:messages:1:21')
  })

  it('sessionId=0 时用 draft-session 后缀', () => {
    expect(messagesKey(1, 0)).toBe('store-mind:messages:1:draft-session')
  })
})

describe('draftKey', () => {
  it('包含 store_id', () => {
    expect(draftKey(10)).toBe('store-mind:draft:10:draft-session')
  })

  it('支持 session 作用域', () => {
    expect(draftKey(1, 21)).toBe('store-mind:draft:1:21')
  })
})

describe('sessionHistoryKey', () => {
  it('包含 store_id', () => {
    expect(sessionHistoryKey(1)).toBe('store-mind:session-history:1')
  })
})
