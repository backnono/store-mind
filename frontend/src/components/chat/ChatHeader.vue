<template>
  <view class="chat-header">
    <view class="chat-top-nav">
      <view class="clerk-avatar">王</view>
      <view class="header-info">
        <view class="header-name">小王 · 数字店员</view>
        <view class="header-sub">
          <view class="live-dot"></view>
          <text>{{ storeName }}</text>
        </view>
      </view>
    </view>

    <view class="chat-title-bar">
      <view class="chat-title-label">💬 对话</view>
      <view class="chat-title-shop">{{ storeName }}</view>
      <view class="title-bar-btns">
        <view class="btn-new-chat" aria-label="新聊天" @tap="$emit('new-chat')">
          <text class="btn-new-chat-icon">+</text>
          <text>新聊天</text>
        </view>
        <view class="btn-history" aria-label="历史会话" @tap="$emit('history')">
          <text class="btn-history-icon">🕐</text>
          <text>历史</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
defineProps<{
  storeName: string
}>()

defineEmits<{
  'new-chat': []
  history: []
}>()
</script>
