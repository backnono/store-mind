// ============================================================
// Session Store — 单元测试
// ============================================================
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSessionStore } from './session'
import { sessionKey } from '@/utils/storage'
import Taro from '@tarojs/taro'

describe('sessionStore', () => {
  beforeEach(async () => {
    setActivePinia(createPinia())
    // 重置 mock 存储状态
    await (Taro as unknown as { clearStorage: () => Promise<void> }).clearStorage()
    vi.clearAllMocks()
  })

  // ── Task 2 ──

  it('startNewSession removes active session from memory and storage', async () => {
    const store = useSessionStore()
    store.setStore(1)
    store.setSessionId(21)

    await store.startNewSession()

    expect(store.currentSessionId).toBe(0)
    await expect(Taro.getStorage({ key: sessionKey(1) })).rejects.toThrow()
  })

  // ── Task 3 ──

  it('records sessions in newest-first local history', async () => {
    const store = useSessionStore()
    store.setStore(1)

    await store.recordSession({ sessionId: 21, title: '薯片在哪里？' })
    await store.recordSession({ sessionId: 22, title: '怎么付款？' })

    expect(store.sessionHistory[0].sessionId).toBe(22)
    expect(store.sessionHistory[1].sessionId).toBe(21)
  })

  it('switchSession updates active session id', async () => {
    const store = useSessionStore()
    store.setStore(1)
    await store.switchSession(21)

    expect(store.currentSessionId).toBe(21)
  })

  it('recordSession caps history at 20 entries', async () => {
    const store = useSessionStore()
    store.setStore(1)

    for (let i = 1; i <= 25; i++) {
      await store.recordSession({ sessionId: i, title: `会话 ${i}` })
    }

    expect(store.sessionHistory.length).toBeLessThanOrEqual(20)
    // 最新的是 25
    expect(store.sessionHistory[0].sessionId).toBe(25)
    // 最老的是 6
    expect(store.sessionHistory[store.sessionHistory.length - 1].sessionId).toBe(6)
  })

  it('deduplicates same session_id', async () => {
    const store = useSessionStore()
    store.setStore(1)

    await store.recordSession({ sessionId: 21, title: '旧标题' })
    await store.recordSession({ sessionId: 21, title: '新标题' })

    expect(store.sessionHistory.length).toBe(1)
    expect(store.sessionHistory[0].title).toBe('新标题')
  })

  it('title defaults to 会话 <id> when empty', async () => {
    const store = useSessionStore()
    store.setStore(1)

    await store.recordSession({ sessionId: 99 })
    expect(store.sessionHistory[0].title).toBe('会话 99')
  })
})
