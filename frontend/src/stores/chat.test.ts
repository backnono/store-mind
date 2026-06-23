// ============================================================
// Chat Store — 单元测试
// ============================================================
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useChatStore } from './chat'
import { useSessionStore } from './session'

// Mock the customerQa service
vi.mock('@/services/customerQa', () => ({
  chat: vi.fn(),
  submitFeedback: vi.fn(),
  listActivePromotions: vi.fn(),
}))

import { chat, submitFeedback } from '@/services/customerQa'

describe('chatStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    const chatStore = useChatStore()
    const sessionStore = useSessionStore()
    chatStore.clearChat()
    sessionStore.setStore(1)
  })

  it('sendMessage 在 API 返回前先添加本地用户消息', async () => {
    const store = useChatStore()
    store.setDraftText('可乐在哪？')

    const mockChat = chat as ReturnType<typeof vi.fn>
    // 延迟 resolve
    mockChat.mockImplementation(
      () =>
        new Promise((resolve) =>
          setTimeout(
            () =>
              resolve({
                session_id: 100,
                message_id: 200,
                intent: 'inventory',
                answer: '可乐在 B-02 货架',
                cards: [],
                guidance_chips: [],
                handoff_required: false,
              }),
            50,
          ),
        ),
    )

    const sendPromise = store.sendMessage()

    // 消息发送后，立刻检查用户消息是否已渲染
    expect(store.messages.length).toBe(1)
    expect(store.messages[0].role).toBe('user')
    expect(store.messages[0].text).toBe('可乐在哪？')
    expect(store.messages[0].sendStatus).toBe('sending')

    await sendPromise

    // API 返回后
    expect(store.messages.length).toBe(2)
    expect(store.messages[1].role).toBe('assistant')
    expect(store.messages[1].text).toBe('可乐在 B-02 货架')
    expect(store.messages[0].sendStatus).toBe('sent')
  })

  it('成功响应保存 session_id 到 sessionStore', async () => {
    const sessionStore = useSessionStore()
    const store = useChatStore()
    store.setDraftText('测试')

    const mockChat = chat as ReturnType<typeof vi.fn>
    mockChat.mockResolvedValue({
      session_id: 999,
      message_id: 888,
      intent: 'faq',
      answer: '回答',
      cards: [],
      guidance_chips: [],
      handoff_required: false,
    })

    expect(sessionStore.currentSessionId).toBe(0)

    await store.sendMessage()

    expect(sessionStore.currentSessionId).toBe(999)
  })

  it('发送失败时标记用户消息为 failed 并设置 lastError', async () => {
    const store = useChatStore()
    store.setDraftText('测试')

    const mockChat = chat as ReturnType<typeof vi.fn>
    mockChat.mockRejectedValue(new Error('网络错误'))

    await store.sendMessage()

    expect(store.messages.length).toBe(1)
    expect(store.messages[0].sendStatus).toBe('failed')
    expect(store.lastError).not.toBeNull()
    expect(store.lastError?.retryable).toBe(true)
  })

  it('反馈只能应用于助理消息', async () => {
    const store = useChatStore()
    // 直接添加消息测试
    store.addUserMessage('用户消息')
    store.addAssistantMessage('助理消息', [], [], 1)

    // 对用户消息尝试反馈 — 应该被拒绝（不会调用 API）
    const mockSubmit = submitFeedback as ReturnType<typeof vi.fn>
    await store.submitFeedback(999, 1) // 不存在的 messageId
    expect(mockSubmit).not.toHaveBeenCalled()

    // 对助理消息反馈
    await store.submitFeedback(1, 1)
    expect(mockSubmit).toHaveBeenCalledWith({
      message_id: 1,
      session_id: 0, // sessionStore.currentSessionId 默认为 0
      feedback_value: 1,
    })
  })

  it('重复反馈被阻止', async () => {
    const store = useChatStore()
    store.addAssistantMessage('助理消息', [], [], 1)
    const sessionStore = useSessionStore()
    sessionStore.setSessionId(10)

    const mockSubmit = submitFeedback as ReturnType<typeof vi.fn>
    mockSubmit.mockResolvedValue(undefined)

    // 第一次
    await store.submitFeedback(1, 1)
    expect(mockSubmit).toHaveBeenCalledTimes(1)
    expect(store.getFeedback(1)).toBe(1)

    // 第二次 — 不应该再调用 API
    await store.submitFeedback(1, 0)
    expect(mockSubmit).toHaveBeenCalledTimes(1)
    expect(store.getFeedback(1)).toBe(1) // 保持第一次的值
  })

  it('draftText 更新后 canSend 变化', () => {
    const store = useChatStore()
    expect(store.canSend).toBe(false)

    store.setDraftText('你好')
    expect(store.canSend).toBe(true)

    store.setDraftText('   ')
    expect(store.canSend).toBe(false)
  })

  it('空消息不能发送', async () => {
    const store = useChatStore()
    store.setDraftText('   ')
    await store.sendMessage()
    expect(store.messages.length).toBe(0)
    expect(chat).not.toHaveBeenCalled()
  })
})
