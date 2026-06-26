// ============================================================
// Session Store — 管理门店会话元数据 + 本地历史列表
// ============================================================
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { EntryContext, LocalSessionSummary } from '@/types/customerQa'
import {
  sessionKey,
  sessionHistoryKey,
  setItem,
  getItem,
  removeItem,
} from '@/utils/storage'

/** 本地历史记录上限 */
const MAX_HISTORY = 20

export const useSessionStore = defineStore('session', () => {
  // ── State ────────────────────────────────────────────
  const storeId = ref<number>(0)
  /** 按 store_id 缓存 session_id */
  const sessionIdByStore = ref<Record<number, number>>({})
  const entryContext = ref<EntryContext | null>(null)
  const lastActiveAt = ref<number>(0)
  /** 本地会话历史（最新在前） */
  const sessionHistory = ref<LocalSessionSummary[]>([])

  // ── Getters ──────────────────────────────────────────
  const currentSessionId = computed(() => sessionIdByStore.value[storeId.value] ?? 0)

  const hasActiveSession = computed(() => currentSessionId.value > 0)

  // ── Actions ──────────────────────────────────────────
  function setStore(id: number) {
    storeId.value = id
    lastActiveAt.value = Date.now()
  }

  function setSessionId(sid: number) {
    if (storeId.value <= 0) return
    sessionIdByStore.value = { ...sessionIdByStore.value, [storeId.value]: sid }
    persistSessionId(storeId.value, sid)
  }

  function setEntryContext(ctx: EntryContext | null) {
    entryContext.value = ctx
  }

  /** 开始新会话（清除当前门店的 session_id 并删除存储） */
  async function startNewSession(): Promise<void> {
    if (storeId.value <= 0) return
    const id = storeId.value
    const sids = { ...sessionIdByStore.value }
    delete sids[id]
    sessionIdByStore.value = sids
    entryContext.value = null
    await removeItem(sessionKey(id))
  }

  // ── Session History ─────────────────────────────────

  /** 记录会话到本地历史（最新在前，最多 20 条） */
  async function recordSession(input: {
    sessionId: number
    title?: string
    lastMessagePreview?: string
  }): Promise<void> {
    if (input.sessionId <= 0 || storeId.value <= 0) return

    const title =
      (input.title?.trim() ?? '').slice(0, 24) || `会话 ${input.sessionId}`

    const entry: LocalSessionSummary = {
      sessionId: input.sessionId,
      storeId: storeId.value,
      title,
      lastMessagePreview: input.lastMessagePreview,
      updatedAt: Date.now(),
    }

    // 去重：如果已存在同 sessionId，移除旧的
    const filtered = sessionHistory.value.filter(
      (s) => s.sessionId !== input.sessionId,
    )

    sessionHistory.value = [entry, ...filtered].slice(0, MAX_HISTORY)

    // 持久化
    try {
      await setItem(sessionHistoryKey(storeId.value), sessionHistory.value)
    } catch {
      // non-critical
    }
  }

  /** 切换活跃会话 */
  async function switchSession(sessionId: number): Promise<void> {
    if (storeId.value <= 0) return
    sessionIdByStore.value = { ...sessionIdByStore.value, [storeId.value]: sessionId }
    await persistSessionId(storeId.value, sessionId)
  }

  /** 从本地存储恢复会话历史 */
  async function restoreSessionHistory(sid: number): Promise<LocalSessionSummary[]> {
    try {
      const saved = await getItem<LocalSessionSummary[]>(sessionHistoryKey(sid))
      if (saved && saved.length > 0) {
        sessionHistory.value = saved
        return saved
      }
    } catch {
      // ignore
    }
    return []
  }

  // ── Persistence ──────────────────────────────────────
  async function persistSessionId(id: number, sid: number) {
    try {
      await setItem(sessionKey(id), sid)
    } catch {
      // storage write failed — non-critical
    }
  }

  /** 从本地存储恢复 session_id */
  async function restoreSessionId(id: number): Promise<number> {
    try {
      const saved = await getItem<number>(sessionKey(id))
      if (saved && saved > 0) {
        sessionIdByStore.value = { ...sessionIdByStore.value, [id]: saved }
        return saved
      }
    } catch {
      // ignore
    }
    return 0
  }

  return {
    storeId,
    sessionIdByStore,
    entryContext,
    lastActiveAt,
    sessionHistory,
    currentSessionId,
    hasActiveSession,
    setStore,
    setSessionId,
    setEntryContext,
    startNewSession,
    recordSession,
    switchSession,
    restoreSessionHistory,
    restoreSessionId,
  }
})
