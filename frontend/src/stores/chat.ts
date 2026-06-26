// ============================================================
// Chat Store — 管理对话消息、发送状态、反馈
// 持久化 key 支持 session 作用域
// ============================================================
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  Message,
  BootstrapStatus,
  ChatCard,
  GuidanceChip,
  ApiError,
} from '@/types/customerQa'
import { chat as chatApi, submitFeedback as submitFeedbackApi } from '@/services/customerQa'
import { normalizeError } from '@/services/api'
import { useSessionStore } from './session'
import { messagesKey, draftKey, setItem, getItem, removeItem } from '@/utils/storage'

/** 持久化保存的消息数量上限 */
const MAX_PERSISTED_MESSAGES = 20

let _nextId = 1
function uid(): string {
  return `msg_${_nextId++}_${Date.now()}`
}

export const useChatStore = defineStore('chat', () => {
  const sessionStore = useSessionStore()

  // ── State ────────────────────────────────────────────
  const messages = ref<Message[]>([])
  const draftText = ref('')
  const isSending = ref(false)
  const bootstrapStatus = ref<BootstrapStatus>('idle')
  const lastError = ref<ApiError | null>(null)
  /** 按 messageId 记录反馈值 */
  const feedbackByMessageId = ref<Record<number, 0 | 1>>({})

  // ── Getters ──────────────────────────────────────────
  const latestMessages = computed(() => messages.value.slice(-MAX_PERSISTED_MESSAGES))

  const canSend = computed(() => !isSending.value && draftText.value.trim().length > 0)

  const failedMessages = computed(() => messages.value.filter((m) => m.sendStatus === 'failed'))

  // ── Message CRUD ────────────────────────────────────

  function addMessage(msg: Message) {
    messages.value.push(msg)
  }

  function updateMessage(id: string, patch: Partial<Message>) {
    const idx = messages.value.findIndex((m) => m.id === id)
    if (idx !== -1) {
      messages.value[idx] = { ...messages.value[idx], ...patch }
    }
  }

  /** 添加用户消息并立即渲染 */
  function addUserMessage(text: string): Message {
    const msg: Message = {
      id: uid(),
      role: 'user',
      text,
      createdAt: Date.now(),
      sendStatus: 'sending',
    }
    messages.value.push(msg)
    return msg
  }

  /** 添加助理消息 */
  function addAssistantMessage(
    text: string,
    cards?: ChatCard[],
    chips?: GuidanceChip[],
    messageId?: number,
    handoffRequired?: boolean,
  ): Message {
    const msg: Message = {
      id: uid(),
      messageId,
      role: 'assistant',
      text,
      cards,
      guidanceChips: chips,
      createdAt: Date.now(),
      handoffRequired,
    }
    messages.value.push(msg)
    return msg
  }

  // ── Send Flow ───────────────────────────────────────

  /** 发送消息 */
  async function sendMessage(): Promise<void> {
    const text = draftText.value.trim()
    if (!text || isSending.value) return

    // 清空草稿
    draftText.value = ''
    isSending.value = true
    lastError.value = null

    // 立即添加用户消息
    const userMsg = addUserMessage(text)

    try {
      const response = await chatApi({
        store_id: sessionStore.storeId,
        session_id: sessionStore.currentSessionId || undefined,
        channel: 'miniapp',
        message: text,
      })

      // 标记用户消息为已发送
      updateMessage(userMsg.id, {
        messageId: response.message_id,
        sendStatus: 'sent',
      })

      // 持久化 session_id
      if (response.session_id) {
        sessionStore.setSessionId(response.session_id)
      }

      // 添加助理消息
      addAssistantMessage(
        response.answer,
        response.cards,
        response.guidance_chips,
        response.message_id,
        response.handoff_required,
      )

      // 记录到本地历史
      if (response.session_id) {
        await sessionStore.recordSession({
          sessionId: response.session_id,
          title: text,
          lastMessagePreview: response.answer.slice(0, 40),
        })
      }

      // 持久化消息快照（按 session 作用域）
      persistMessages()
    } catch (err) {
      const apiErr = normalizeError(err)
      lastError.value = apiErr
      updateMessage(userMsg.id, { sendStatus: 'failed' })
    } finally {
      isSending.value = false
    }
  }

  function setDraftText(text: string) {
    draftText.value = text
  }

  // ── Feedback ────────────────────────────────────────

  async function submitFeedback(messageId: number, value: 0 | 1): Promise<void> {
    const msg = messages.value.find((m) => m.messageId === messageId)
    if (!msg || msg.role !== 'assistant') return

    if (feedbackByMessageId.value[messageId] !== undefined) return

    feedbackByMessageId.value = { ...feedbackByMessageId.value, [messageId]: value }

    try {
      await submitFeedbackApi({
        message_id: messageId,
        session_id: sessionStore.currentSessionId,
        feedback_value: value,
      })
    } catch {
      const fb = { ...feedbackByMessageId.value }
      delete fb[messageId]
      feedbackByMessageId.value = fb
    }
  }

  function getFeedback(messageId: number): 0 | 1 | undefined {
    return feedbackByMessageId.value[messageId]
  }

  // ── Persistence (session-scoped) ────────────────────

  /** 当前会话的 session_id（持久化 key 用） */
  function _currentSessionId(): number {
    return sessionStore.currentSessionId
  }

  async function persistMessages(): Promise<void> {
    if (sessionStore.storeId <= 0) return
    const key = messagesKey(sessionStore.storeId, _currentSessionId())
    const snapshot = latestMessages.value
    try {
      await setItem(key, snapshot)
    } catch {
      // non-critical
    }
  }

  async function restoreMessages(storeId: number): Promise<Message[]> {
    try {
      const key = messagesKey(storeId, _currentSessionId())
      const saved = await getItem<Message[]>(key)
      if (saved && saved.length > 0) {
        messages.value = saved
        const maxExisting = Math.max(
          ...saved.map((m) => {
            const parts = m.id.split('_')
            return parseInt(parts[1] ?? '0', 10)
          }),
          0,
        )
        _nextId = maxExisting + 1
        return saved
      }
    } catch {
      // ignore
    }
    return []
  }

  async function persistDraft(storeId: number): Promise<void> {
    const key = draftKey(storeId, _currentSessionId())
    if (!draftText.value) {
      await removeItem(key)
      return
    }
    try {
      await setItem(key, draftText.value)
    } catch {
      // non-critical
    }
  }

  async function restoreDraft(storeId: number): Promise<string> {
    try {
      const key = draftKey(storeId, _currentSessionId())
      const saved = await getItem<string>(key)
      if (saved) {
        draftText.value = saved
        return saved
      }
    } catch {
      // ignore
    }
    return ''
  }

  // ── Lifecycle ───────────────────────────────────────

  function clearChat() {
    messages.value = []
    draftText.value = ''
    isSending.value = false
    bootstrapStatus.value = 'idle'
    lastError.value = null
    feedbackByMessageId.value = {}
  }

  function setBootstrapStatus(status: BootstrapStatus) {
    bootstrapStatus.value = status
  }

  return {
    messages,
    draftText,
    isSending,
    bootstrapStatus,
    lastError,
    feedbackByMessageId,
    latestMessages,
    canSend,
    failedMessages,
    addMessage,
    updateMessage,
    addUserMessage,
    addAssistantMessage,
    sendMessage,
    setDraftText,
    submitFeedback,
    getFeedback,
    persistMessages,
    restoreMessages,
    persistDraft,
    restoreDraft,
    clearChat,
    setBootstrapStatus,
  }
})
