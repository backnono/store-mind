<template>
  <view class="messages-inner">
    <!-- Context bridge -->
    <view v-if="contextBridge" class="context-bridge">
      您之前在看 <text class="cb-hl">{{ contextBridge }}</text>，需要继续吗？
    </view>

    <!-- 消息列表 -->
    <view
      v-for="msg in filteredMessages"
      :key="msg.id"
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

    <!-- Typing dots: 对齐无气泡 AI 样式 -->
    <view v-if="isThinking" class="typing-row">
      <view class="msg-avatar clerk-av">王</view>
      <view class="typing-content">
        <view class="ai-name">王</view>
        <view class="typing-dots">
          <view class="dot"></view>
          <view class="dot"></view>
          <view class="dot"></view>
        </view>
      </view>
    </view>

    <view class="messages-bottom-spacer"></view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
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
</script>

<style lang="scss">
.messages-inner {
  box-sizing: border-box;
  min-height: 100%;
  padding: 24px 20px 0;
}

.messages-bottom-spacer {
  height: 16px;
  flex-shrink: 0;
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

// Typing — 对齐无气泡 AI 样式
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
  margin-top: 2px;
}

.clerk-av {
  background: linear-gradient(135deg, #c9863a, #b87a2e);
  color: #1c1b18;
  box-shadow: 0 2px 10px rgba(201, 134, 58, 0.22);
}

.typing-content {
  display: flex;
  flex-direction: column;
}

.ai-name {
  font-size: 22px;
  font-weight: 600;
  color: #888;
  margin-bottom: 6px;
  letter-spacing: 0.3px;
}

.typing-dots {
  display: flex;
  gap: 8px;
  padding: 8px 0;
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
