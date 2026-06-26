<template>
  <view class="chat-page" :style="chatPageStyle">
    <!-- 顶部头部 -->
    <ChatHeader
      :store-name="storeName"
      @new-chat="onNewChat"
      @history="onHistory"
    />

    <!-- 货架上下文 Banner -->
    <ZoneBanner
      v-if="zoneBanner.show"
      :zone-label="zoneBanner.label"
      :zone-desc="zoneBanner.desc"
    />

    <view class="chat-body">
      <!-- 消息展示区：对齐 prototype-a-v3 的 chat-area -->
      <scroll-view
        class="chat-scroll-area"
        scroll-y
        :scroll-into-view="scrollIntoView"
        :scroll-with-animation="scrollWithAnimation"
        @scroll="handleMessageScroll"
        :enable-flex="true"
        :bounces="false"
        :enable-back-to-top="false"
        :enhanced="true"
        :show-scrollbar="true"
      >
        <MessageList
          :messages="chatStore.messages"
          :is-thinking="isThinking"
          :context-bridge="contextBridge"
          @chip-select="onChipSelect"
          @feedback-submit="onFeedbackSubmit"
          @retry="onRetry"
        />
        <view :id="bottomAnchorId" class="chat-scroll-bottom-anchor"></view>
      </scroll-view>
    </view>

    <!-- 消息输入区 -->
    <view class="chat-composer-slot">
      <ChatInput
        v-model="chatStore.draftText"
        :disabled="chatStore.isSending"
        placeholder="问小王任何问题…"
        @send="onSend"
      />
    </view>

    <view class="chat-tabbar-reserve"></view>

    <!-- 历史会话面板 -->
    <view v-if="showHistory" class="history-mask" @tap="showHistory = false">
      <view class="history-panel" @tap.stop>
        <view class="history-title">
          <text class="history-title-text">历史对话</text>
          <view class="history-close" @tap="showHistory = false">×</view>
        </view>
        <view
          v-for="item in sessionStore.sessionHistory"
          :key="item.sessionId"
          class="history-item"
          :class="{ active: item.sessionId === sessionStore.currentSessionId }"
          @tap="onSelectSession(item.sessionId)"
        >
          <view class="history-item-icon">💬</view>
          <view class="history-item-info">
            <view class="history-item-title">{{ item.title }}</view>
            <view class="history-item-preview">
              {{ item.lastMessagePreview || formatHistoryTime(item.updatedAt) }}
            </view>
          </view>
        </view>
        <view
          v-if="sessionStore.sessionHistory.length === 0"
          class="history-empty"
        >
          暂无历史会话
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, onMounted, onUnmounted, watch } from 'vue'
import Taro, { useLoad } from '@tarojs/taro'
import { useSessionStore } from '@/stores/session'
import { useChatStore } from '@/stores/chat'
import { chat as chatApi } from '@/services/customerQa'
import ChatHeader from '@/components/chat/ChatHeader.vue'
import ZoneBanner from '@/components/chat/ZoneBanner.vue'
import MessageList from '@/components/chat/MessageList.vue'
import ChatInput from '@/components/chat/ChatInput.vue'
import type { Message } from '@/types/customerQa'
import {
  CUSTOM_TABBAR_HEIGHT_EVENT,
  CUSTOM_TABBAR_HEIGHT_STORAGE_KEY,
} from '@/utils/layoutEvents'

const sessionStore = useSessionStore()
const chatStore = useChatStore()

const storeName = ref('阳光便利店 No.3')
const isThinking = ref(false)
const contextBridge = ref('')
const showHistory = ref(false)
const scrollIntoView = ref('')
const scrollWithAnimation = ref(true)
const autoStickToBottom = ref(true)
const bottomAnchorSerial = ref(0)
const bottomAnchorId = computed(() => `chat-scroll-bottom-${bottomAnchorSerial.value}`)
let releaseScrollTargetTimer: ReturnType<typeof setTimeout> | undefined

const DESIGN_WIDTH_RPX = 750
const TABBAR_CONTENT_RPX = 104
const DEFAULT_COMPOSER_HEIGHT_PX = 72
const DEFAULT_HEADER_ZONE_HEIGHT_PX = 210
const MIN_MESSAGE_VIEWPORT_HEIGHT_PX = 160

type SystemInfoLike = {
  windowWidth?: number
  windowHeight?: number
  screenHeight?: number
  safeArea?: {
    bottom?: number
  }
}

const initialSystemInfo = readSystemInfo()
const tabbarReservePx = ref(estimateTabbarReservePx(initialSystemInfo))
const composerHeightPx = ref(DEFAULT_COMPOSER_HEIGHT_PX)
const messageViewportHeight = ref(estimateInitialMessageViewportHeight(initialSystemInfo))

const chatPageStyle = computed<Record<string, string>>(() => ({
  '--chat-tabbar-reserve': `${Math.ceil(tabbarReservePx.value)}px`,
  '--chat-composer-height': `${Math.ceil(composerHeightPx.value)}px`,
  '--chat-message-viewport-height': `${Math.ceil(messageViewportHeight.value)}px`,
}))

const zoneBanner = ref<{
  show: boolean
  label: string
  desc: string
}>({ show: false, label: '', desc: '' })

type SelectorRect = {
  height?: number
}

type ScrollDetail = {
  scrollTop?: number
  scrollHeight?: number
}

type ScrollEvent = {
  detail?: ScrollDetail
}

const AUTO_SCROLL_THRESHOLD_PX = 80

function readSystemInfo(): SystemInfoLike {
  try {
    return Taro.getSystemInfoSync() as SystemInfoLike
  } catch {
    return {}
  }
}

function getWindowHeightPx(system: SystemInfoLike = readSystemInfo()): number {
  return Math.floor(system.windowHeight ?? system.safeArea?.bottom ?? 667)
}

function getSafeAreaBottomInsetPx(system: SystemInfoLike = readSystemInfo()): number {
  const screenHeight = system.screenHeight ?? system.windowHeight ?? 0
  const safeBottom = system.safeArea?.bottom
  if (!screenHeight || !safeBottom) return 0
  return Math.max(0, Math.ceil(screenHeight - safeBottom))
}

function rpxToPx(rpx: number, system: SystemInfoLike = readSystemInfo()): number {
  const windowWidth = system.windowWidth ?? 375
  return rpx * (windowWidth / DESIGN_WIDTH_RPX)
}

function estimateTabbarReservePx(system: SystemInfoLike = readSystemInfo()): number {
  return Math.ceil(rpxToPx(TABBAR_CONTENT_RPX, system) + getSafeAreaBottomInsetPx(system))
}

function estimateInitialMessageViewportHeight(system: SystemInfoLike = readSystemInfo()): number {
  const availableHeight = getWindowHeightPx(system)
    - DEFAULT_HEADER_ZONE_HEIGHT_PX
    - DEFAULT_COMPOSER_HEIGHT_PX
    - estimateTabbarReservePx(system)

  return Math.max(MIN_MESSAGE_VIEWPORT_HEIGHT_PX, Math.floor(availableHeight))
}

function normalizeRectHeight(rect: SelectorRect | SelectorRect[] | null): number {
  const box = Array.isArray(rect) ? rect[0] : rect
  return box?.height && box.height > 0 ? Math.ceil(box.height) : 0
}

function measureSelectorHeight(selector: string): Promise<number> {
  return new Promise((resolve) => {
    Taro.createSelectorQuery()
      .select(selector)
      .boundingClientRect((rect: SelectorRect | SelectorRect[] | null) => {
        resolve(normalizeRectHeight(rect))
      })
      .exec()
  })
}

function handleTabbarHeight(height: number): void {
  if (!Number.isFinite(height) || height <= 0) return
  tabbarReservePx.value = Math.ceil(height)
  void updateLayoutMetrics()
}

function restoreCachedTabbarHeight(): void {
  void Taro.getStorage({ key: CUSTOM_TABBAR_HEIGHT_STORAGE_KEY })
    .then(({ data }) => {
      const cachedHeight = Number(data)
      handleTabbarHeight(cachedHeight)
    })
    .catch(() => undefined)
}

async function updateLayoutMetrics(): Promise<void> {
  await nextTick()

  const system = readSystemInfo()
  const windowHeight = getWindowHeightPx(system)
  const [headerHeight, zoneHeight, measuredComposerHeight] = await Promise.all([
    measureSelectorHeight('.chat-header'),
    measureSelectorHeight('.zone-banner'),
    measureSelectorHeight('.chat-composer-slot'),
  ])

  if (measuredComposerHeight > 0) {
    composerHeightPx.value = measuredComposerHeight
  }

  const viewportHeight = windowHeight
    - headerHeight
    - zoneHeight
    - composerHeightPx.value
    - tabbarReservePx.value

  if (windowHeight > 0) {
    messageViewportHeight.value = Math.max(
      MIN_MESSAGE_VIEWPORT_HEIGHT_PX,
      Math.floor(viewportHeight),
    )
  }

  await updateMessageViewportHeight()
}

async function updateMessageViewportHeight(): Promise<void> {
  await nextTick()
  Taro.createSelectorQuery()
    .select('.chat-scroll-area')
    .boundingClientRect((rect: SelectorRect | SelectorRect[] | null) => {
      const box = Array.isArray(rect) ? rect[0] : rect
      if (box?.height && box.height > 0) {
        messageViewportHeight.value = Math.floor(box.height)
      }
    })
    .exec()
}

function isNearBottom(detail: ScrollDetail): boolean {
  const scrollTop = detail.scrollTop ?? 0
  const scrollHeight = detail.scrollHeight ?? 0
  const viewportHeight = messageViewportHeight.value

  if (!scrollHeight || !viewportHeight) return true

  return (scrollHeight - (scrollTop + viewportHeight)) <= AUTO_SCROLL_THRESHOLD_PX
}

function handleMessageScroll(event: ScrollEvent): void {
  autoStickToBottom.value = isNearBottom(event.detail ?? {})
}

function requestStickToBottom(animated = true): void {
  void scrollToBottom(true, animated)
}

async function scrollToBottom(force = false, animated = true): Promise<void> {
  if (!force && !autoStickToBottom.value) return
  await nextTick()
  bottomAnchorSerial.value += 1
  await nextTick()
  scrollWithAnimation.value = animated
  scrollIntoView.value = bottomAnchorId.value
  releaseScrollControl()
}

function releaseScrollControl(): void {
  if (releaseScrollTargetTimer) clearTimeout(releaseScrollTargetTimer)
  releaseScrollTargetTimer = setTimeout(() => {
    scrollIntoView.value = ''
  }, 160)
}

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

  // 恢复本地历史（始终执行）
  await sessionStore.restoreSessionHistory(sessionStore.storeId)

  // 恢复 session_id（仅 resume 入口自动取回）
  if (entry === 'resume') {
    const savedSid = await sessionStore.restoreSessionId(sessionStore.storeId)
    if (savedSid) {
      try {
        await restoreSession()
        chatStore.setBootstrapStatus('ready')
        return
      } catch { /* fall through */ }
    }
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
  // 防止重复添加欢迎消息
  if (chatStore.messages.length > 0) return

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
  const sending = chatStore.sendMessage()
  requestStickToBottom()
  await sending
  isThinking.value = false
  requestStickToBottom()
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
  const sending = chatStore.sendMessage()
  requestStickToBottom()
  await sending
  requestStickToBottom()
}

// ── 新聊天 ───────────────────────────────────────────
async function onNewChat(): Promise<void> {
  await chatStore.persistDraft(sessionStore.storeId)
  await chatStore.persistMessages()
  await sessionStore.startNewSession()
  chatStore.clearChat()
  contextBridge.value = ''
  zoneBanner.value = { show: false, label: '', desc: '' }
  await handleFirstOpen()
  requestStickToBottom(false)
}

// ── 历史面板 ─────────────────────────────────────────
async function onHistory(): Promise<void> {
  await sessionStore.restoreSessionHistory(sessionStore.storeId)
  showHistory.value = true
}

async function onSelectSession(sid: number): Promise<void> {
  await chatStore.persistDraft(sessionStore.storeId)
  await chatStore.persistMessages()
  await sessionStore.switchSession(sid)
  chatStore.clearChat()
  await chatStore.restoreMessages(sessionStore.storeId)
  await chatStore.restoreDraft(sessionStore.storeId)
  contextBridge.value = '历史会话'
  showHistory.value = false
  requestStickToBottom(false)
}

function formatHistoryTime(timestamp: number): string {
  if (!timestamp) return '继续这段对话'
  const date = new Date(timestamp)
  const now = new Date()
  const isToday = date.toDateString() === now.toDateString()

  if (isToday) {
    const hh = `${date.getHours()}`.padStart(2, '0')
    const mm = `${date.getMinutes()}`.padStart(2, '0')
    return `今天 ${hh}:${mm}`
  }

  return `${date.getMonth() + 1}月${date.getDate()}日`
}

// ── 生命周期 ─────────────────────────────────────────
watch(() => zoneBanner.value.show, () => {
  updateLayoutMetrics()
})

watch(
  () => [
    chatStore.messages[0]?.id ?? 'empty',
    chatStore.messages.map((m: Message) => m.id).join('|'),
    isThinking.value ? 'thinking' : 'idle',
  ],
  ([firstMessageId], previous) => {
    const conversationChanged = firstMessageId !== previous?.[0]
    if (conversationChanged) {
      autoStickToBottom.value = true
      requestStickToBottom(false)
      return
    }
    void scrollToBottom(false, true)
  },
  { immediate: true },
)

onMounted(() => {
  restoreCachedTabbarHeight()
  Taro.eventCenter.on(CUSTOM_TABBAR_HEIGHT_EVENT, handleTabbarHeight)
  void updateLayoutMetrics()
  window.setTimeout(() => void updateLayoutMetrics(), 120)
  window.setTimeout(() => void updateLayoutMetrics(), 320)
})
onUnmounted(() => {
  if (releaseScrollTargetTimer) clearTimeout(releaseScrollTargetTimer)
  Taro.eventCenter.off(CUSTOM_TABBAR_HEIGHT_EVENT, handleTabbarHeight)
  chatStore.persistDraft(sessionStore.storeId)
  chatStore.persistMessages()
})
</script>

<style lang="scss" src="./index.scss"></style>
