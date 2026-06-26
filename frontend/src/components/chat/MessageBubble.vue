<template>
  <view class="msg-row" :class="role">
    <!-- 助理头像 (左侧) -->
    <view v-if="role === 'assistant'" class="msg-avatar clerk-av">王</view>

    <view class="msg-body-col">
      <!-- AI 名称标注 -->
      <view v-if="role === 'assistant'" class="ai-name">王</view>

      <!-- AI 文本（无气泡，纯文字贴背景） -->
      <view v-if="role === 'assistant' && text" class="ai-text">{{ text }}</view>

      <!-- 用户气泡 -->
      <view v-if="role === 'user'" class="msg-bubble user">
        <text class="bubble-text" v-if="text">{{ text }}</text>
      </view>

      <!-- 卡片（仅 AI 消息） -->
      <view v-if="cards && cards.length > 0" class="bubble-cards">
        <AnswerCard v-for="(card, idx) in cards" :key="idx" :card="card" />
      </view>

      <!-- 引导芯片（仅 AI 消息，内嵌） -->
      <GuidanceChips
        v-if="chips && chips.length > 0"
        :chips="chips"
        @select="$emit('chipSelect', $event)"
      />

      <!-- 反馈栏 — 仅助理消息 -->
      <FeedbackBar
        v-if="role === 'assistant' && messageId"
        :message-id="messageId"
        :selected="feedbackValue"
        @submit="(v: 0 | 1) => $emit('feedbackSubmit', messageId!, v)"
      />

      <!-- 发送失败重试 -->
      <view v-if="sendStatus === 'failed'" class="msg-status-fail">
        <text>发送失败</text>
        <text class="retry-link" @tap="$emit('retry')">重试</text>
      </view>
    </view>

    <!-- 用户头像 (右侧) -->
    <view v-if="role === 'user'" class="msg-avatar user-av">我</view>
  </view>
</template>

<script setup lang="ts">
import type { ChatCard, GuidanceChip } from '@/types/customerQa'
import AnswerCard from './AnswerCard.vue'
import GuidanceChips from './GuidanceChips.vue'
import FeedbackBar from './FeedbackBar.vue'

defineProps<{
  text: string
  role: 'user' | 'assistant'
  messageId?: number
  cards?: ChatCard[]
  chips?: GuidanceChip[]
  feedbackValue?: 0 | 1
  sendStatus?: string
}>()

defineEmits<{
  chipSelect: [prompt: string]
  feedbackSubmit: [messageId: number, value: 0 | 1]
  retry: []
}>()
</script>

<style lang="scss">
.msg-row {
  display: flex;
  gap: 10px;
  margin-bottom: 28px;
  max-width: 100%;

  &.user { justify-content: flex-end; }
  &.assistant { justify-content: flex-start; }
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

.user-av {
  background: #38352e;
  color: #bfb6a4;
}

.msg-body-col {
  display: flex;
  flex-direction: column;
  max-width: 82%;
}

// ── AI 名称标注 ──
.ai-name {
  font-size: 22px;
  font-weight: 600;
  color: #888;
  margin-bottom: 6px;
  letter-spacing: 0.3px;
}

// ── AI 纯文本（无气泡容器） ──
.ai-text {
  font-size: 29px;
  line-height: 1.65;
  color: #bfb6a4;
  word-break: break-word;
  white-space: pre-wrap;
  padding: 0;
}

// ── 用户气泡: 冷灰色 + 右下尖角 ──
.msg-bubble {
  &.user {
    background: #3e3e4e;
    color: #f0f0f5;
    padding: 16px 22px;
    border-radius: 28px;
    border-bottom-right-radius: 4px;
    font-size: 29px;
    line-height: 1.55;
    word-break: break-word;
  }
}

.bubble-text {
  word-break: break-word;
  white-space: pre-wrap;
}

.bubble-cards {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-top: 14px;
}

.msg-status-fail {
  text-align: right;
  font-size: 20px;
  color: #d9534f;
  padding-top: 4px;
  padding-right: 4px;
}

.retry-link {
  color: #deb370;
  margin-left: 8px;
}
</style>
