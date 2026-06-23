// ============================================================
// Session Store — 管理门店会话元数据
// ============================================================
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { EntryContext } from '@/types/customerQa'
import { sessionKey, setItem, getItem } from '@/utils/storage'

export const useSessionStore = defineStore('session', () => {
  // ── State ────────────────────────────────────────────
  const storeId = ref<number>(0)
  /** 按 store_id 缓存 session_id */
  const sessionIdByStore = ref<Record<number, number>>({})
  const entryContext = ref<EntryContext | null>(null)
  const lastActiveAt = ref<number>(0)

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

  /** 开始新会话（清除当前门店的 session_id） */
  function startNewSession() {
    if (storeId.value <= 0) return
    const sids = { ...sessionIdByStore.value }
    delete sids[storeId.value]
    sessionIdByStore.value = sids
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
    currentSessionId,
    hasActiveSession,
    setStore,
    setSessionId,
    setEntryContext,
    startNewSession,
    restoreSessionId,
  }
})
