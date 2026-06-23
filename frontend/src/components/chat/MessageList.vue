<template>
  <scroll-view
    class="messages"
    scroll-y
    :scroll-with-animation="true"
    :scroll-into-view="scrollIntoView"
    :enhanced="true"
    :show-scrollbar="false"
  >
    <!-- Context bridge -->
    <view v-if="contextBridge" class="context-bridge">
      您之前在看 <text class="cb-hl">{{ contextBridge }}</text>，需要继续吗？
    </view>

    <!-- 消息列表 -->
    <view
      v-for="msg in filteredMessages"
      :key="msg.id"
      :id="`msg-${msg.id}`"
    >
      <MessageBubble
        :text="msg.text"
        :role="msg.role"
        :message-id="msg.messageId"
        :cards="msg.cards"
        :chips="msg.guidanceChips"
        :feedback-value="getFeedback(msg.messageId)"
        :send-status="msg.sendStatus"
        @chip-select="(p: string) => $emit('chipSelect', p)"
        @feedback-submit="(mid: number, v: 0 | 1) => $emit('feedbackSubmit', mid, v)"
        @retry="$emit('retry', msg.id)"
      />
    </view>

    <!-- Typing dots -->
    <view v-if="isThinking" class="typing-row">
      <view class="msg-avatar clerk-av">王</view>
      <view class="typing-bubble">
        <view class="typing-dots">
          <view class="dot"></view>
          <view class="dot"></view>
          <view class="dot"></view>
        </view>
      </view>
    </view>

    <!-- 滚动锚点 -->
    <view id="scroll-bottom" style="height:1px"></view>
  </scroll-view>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { Message } from '@/types/customerQa'
import { useChatStore } from '@/stores/chat'
import MessageBubble from './MessageBubble.vue'

const props = defineProps<{
  messages: Message[]
  isThinking: boolean
  contextBridge?: string
}>()

defineEmits<{
  chipSelect: [prompt: string]
  feedbackSubmit: [messageId: number, value: 0 | 1]
  retry: [messageId: string]
}>()

const chatStore = useChatStore()

const filteredMessages = computed(() =>
  props.messages.filter((m) => m.sendStatus !== 'failed'),
)

const getFeedback = (messageId?: number): 0 | 1 | undefined => {
  if (!messageId) return undefined
  return chatStore.getFeedback(messageId)
}

const scrollIntoView = ref('')

watch(() => [
  filteredMessages.value.map((m) => `${m.id}:${m.sendStatus ?? ''}`).join('|'),
  props.isThinking ? 'thinking' : 'idle',
], async () => {
  await nextTick()
  scrollIntoView.value = ''
  await nextTick()
  scrollIntoView.value = 'scroll-bottom'
}, {
  immediate: true,
})
</script>

<style lang="scss">
.messages {
  flex: 1;
  min-height: 0;
  height: 0;
  box-sizing: border-box;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 24px 20px 360px;
  scroll-behavior: smooth;
}

.context-bridge {
  text-align: center;
  padding: 10px 24px;
  margin: 0 8px 20px;
  font-size: 22px;
  color: #8c8070;
  background: #23211d;
  border-radius: 20px;
}

.cb-hl {
  color: #deb370;
  font-weight: 500;
}

// Typing
.typing-row {
  display: flex;
  gap: 10px;
}

.msg-avatar {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 700;
  margin-top: 4px;
}

.clerk-av {
  background: linear-gradient(135deg, #c9863a, #b87a2e);
  color: #1c1b18;
  box-shadow: 0 2px 10px rgba(201, 134, 58, 0.22);
}

.typing-bubble {
  background: #24211d;
  border-radius: 28px;
  border-bottom-left-radius: 8px;
  padding: 18px 24px;
}

.typing-dots {
  display: flex;
  gap: 6px;
  padding: 4px 0;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #7a6e5c;
  animation: dot-bounce 1.4s ease-in-out infinite;

  &:nth-child(2) { animation-delay: 0.2s; }
  &:nth-child(3) { animation-delay: 0.4s; }
}

@keyframes dot-bounce {
  0%, 80%, 100% { transform: scale(0.7); opacity: 0.4; }
  40% { transform: scale(1); opacity: 1; }
}
</style>
