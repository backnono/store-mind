<template>
  <view class="chat-page">
    <!-- 顶部头部 -->
    <ChatHeader :store-name="storeName" />

    <!-- 货架上下文 Banner -->
    <ZoneBanner
      v-if="zoneBanner.show"
      :zone-label="zoneBanner.label"
      :zone-desc="zoneBanner.desc"
    />

    <!-- 消息列表 -->
    <MessageList
      :messages="chatStore.messages"
      :is-thinking="isThinking"
      :context-bridge="contextBridge"
      @chip-select="onChipSelect"
      @feedback-submit="onFeedbackSubmit"
      @retry="onRetry"
    />

    <!-- 输入区域 -->
    <ChatInput
      v-model="chatStore.draftText"
      :disabled="chatStore.isSending"
      placeholder="问小王任何问题…"
      @send="onSend"
    />
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import Taro, { useLoad } from '@tarojs/taro'
import { useSessionStore } from '@/stores/session'
import { useChatStore } from '@/stores/chat'
import { chat as chatApi } from '@/services/customerQa'
import ChatHeader from '@/components/chat/ChatHeader.vue'
import ZoneBanner from '@/components/chat/ZoneBanner.vue'
import MessageList from '@/components/chat/MessageList.vue'
import ChatInput from '@/components/chat/ChatInput.vue'
import type { Message } from '@/types/customerQa'

const sessionStore = useSessionStore()
const chatStore = useChatStore()

const storeName = ref('阳光便利店 No.3')
const isThinking = ref(false)
const contextBridge = ref('')

const zoneBanner = ref<{
  show: boolean
  label: string
  desc: string
}>({ show: false, label: '', desc: '' })

// ── 页面加载 ─────────────────────────────────────────
useLoad((options) => {
  const storeId = Number(options.store_id) || 1
  const entry = (options.entry as string) || 'first_open'

  sessionStore.setStore(storeId)
  bootstrapEntry(entry, options)
})

async function bootstrapEntry(
  entry: string,
  options: Record<string, string | undefined>,
): Promise<void> {
  chatStore.setBootstrapStatus('loading')

  // 尝试恢复
  const savedSid = await sessionStore.restoreSessionId(sessionStore.storeId)

  if (entry === 'resume' && savedSid) {
    try {
      await restoreSession()
      chatStore.setBootstrapStatus('ready')
      return
    } catch { /* fall through */ }
  }

  // 从 sessionStore 读取入口上下文（zone_scan 通过它传参）
  const ctx = sessionStore.entryContext
  const effectiveEntry = ctx?.mode ?? entry

  try {
    if (effectiveEntry === 'zone_scan') {
      await handleZoneScan(ctx)
    } else if (effectiveEntry === 'first_open') {
      await handleFirstOpen()
    } else if (effectiveEntry === 'promo' || effectiveEntry === 'product_detail') {
      await handlePromoEntry(ctx?.prompt ?? options.prompt)
    }
    chatStore.setBootstrapStatus('ready')
  } catch (err) {
    chatStore.setBootstrapStatus('error')
  }
}

// ── 入口处理 ──────────────────────────────────────────

async function handleFirstOpen(): Promise<void> {
  chatStore.addAssistantMessage(
    '您好！我是小王，您身边的数字店员。\n\n店里的一切都可以问我：商品在哪儿、还有没有货、今天有什么活动、怎么付款……',
    [],
    [
      { text: '📍 薯片在哪里？', prompt: '薯片在哪里？' },
      { text: '🏷 今天有什么活动？', prompt: '今天有什么活动？' },
      { text: '🥤 低糖饮料有哪些？', prompt: '低糖饮料有哪些？' },
      { text: '💳 怎么付款？', prompt: '怎么付款？' },
    ],
  )
}

async function handleZoneScan(ctx?: { zoneId?: number; shelfId?: number } | null): Promise<void> {
  zoneBanner.value = {
    show: true,
    label: `货架 ${ctx?.shelfId ?? ctx?.zoneId ?? ''}`,
    desc: '当前区域陈列商品',
  }

  try {
    const response = await chatApi({
      store_id: sessionStore.storeId,
      channel: 'miniapp',
      message: '',
      entry_mode: 'zone_scan',
      zone_id: ctx?.zoneId,
      shelf_id: ctx?.shelfId,
    })

    if (response.session_id) sessionStore.setSessionId(response.session_id)

    if (response.answer) {
      chatStore.addAssistantMessage(
        response.answer, response.cards, response.guidance_chips,
        response.message_id, response.handoff_required,
      )
    }
  } catch {
    chatStore.addAssistantMessage('我看到您在当前区域。想了解什么商品，告诉我名字就行。')
  }
}

async function handlePromoEntry(prompt?: string): Promise<void> {
  const response = await chatApi({
    store_id: sessionStore.storeId,
    channel: 'miniapp',
    message: prompt || '',
    entry_mode: sessionStore.entryContext?.mode ?? 'promo',
  })

  if (response.session_id) sessionStore.setSessionId(response.session_id)

  chatStore.addAssistantMessage(
    response.answer, response.cards, response.guidance_chips,
    response.message_id, response.handoff_required,
  )
}

async function restoreSession(): Promise<void> {
  const response = await chatApi({
    store_id: sessionStore.storeId,
    session_id: sessionStore.currentSessionId,
    channel: 'miniapp',
    message: '',
    entry_mode: 'resume',
  })

  if (response.session_id) sessionStore.setSessionId(response.session_id)
  contextBridge.value = '最近对话'

  if (response.answer) {
    chatStore.addAssistantMessage(
      response.answer, response.cards, response.guidance_chips,
      response.message_id, response.handoff_required,
    )
  }

  await chatStore.restoreMessages(sessionStore.storeId)
  await chatStore.restoreDraft(sessionStore.storeId)
}

// ── 发送消息 ──────────────────────────────────────────
async function onSend(): Promise<void> {
  if (!chatStore.canSend) return
  if (contextBridge.value) contextBridge.value = ''
  isThinking.value = true
  await chatStore.sendMessage()
  isThinking.value = false
}

// ── 引导芯片 ──────────────────────────────────────────
function onChipSelect(prompt: string): void {
  chatStore.setDraftText(prompt)
}

// ── 反馈 ────────────────────────────────────────────
function onFeedbackSubmit(messageId: number, value: 0 | 1): void {
  chatStore.submitFeedback(messageId, value)
}

// ── 重试 ─────────────────────────────────────────────
async function onRetry(msgId: string): Promise<void> {
  const msg = chatStore.messages.find((m: Message) => m.id === msgId)
  if (!msg || msg.role !== 'user') return
  const idx = chatStore.messages.findIndex((m: Message) => m.id === msgId)
  if (idx !== -1) chatStore.messages.splice(idx, 1)
  chatStore.setDraftText(msg.text)
  await chatStore.sendMessage()
}

// ── 生命周期 ─────────────────────────────────────────
onMounted(() => {})
onUnmounted(() => {
  chatStore.persistDraft(sessionStore.storeId)
  chatStore.persistMessages()
})
</script>

<style lang="scss" src="./index.scss"></style>
