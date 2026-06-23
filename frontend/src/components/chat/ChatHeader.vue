<template>
  <view class="chat-header">
    <view class="clerk-avatar">王</view>
    <view class="header-info">
      <view class="header-name">小王 · 数字店员</view>
      <view class="header-sub">
        <view class="live-dot"></view>
        <text>在线 · {{ storeName }}</text>
      </view>
    </view>
    <view class="header-actions">
      <view class="header-btn" @tap="$emit('more')">⋯</view>
    </view>
  </view>
</template>

<script setup lang="ts">
defineProps<{
  storeName: string
}>()

defineEmits<{
  more: []
}>()
</script>

<style lang="scss">
.chat-header {
  flex-shrink: 0;
  padding: 18px 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  border-bottom: 1px solid #2b2925;
  background: #1c1b18;
  z-index: 10;
}

.clerk-avatar {
  width: 68px;
  height: 68px;
  border-radius: 50%;
  background: linear-gradient(135deg, #c9863a, #b87a2e);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 30px;
  font-weight: 700;
  color: #1c1b18;
  box-shadow: 0 4px 16px rgba(201, 134, 58, 0.28);
  flex-shrink: 0;
}

.header-info {
  flex: 1;
  min-width: 0;
}

.header-name {
  font-size: 30px;
  font-weight: 600;
  color: #ede6dc;
  line-height: 1.3;
}

.header-sub {
  font-size: 22px;
  color: #d4a44c;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 2px;
}

.live-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #5cb85c;
  animation: pulse-live 2s ease-in-out infinite;
  box-shadow: 0 0 5px rgba(92, 184, 92, 0.5);
}

@keyframes pulse-live {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}

.header-actions {
  display: flex;
  gap: 10px;
}

.header-btn {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: #2b2820;
  border: none;
  color: #bfb6a4;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 30px;
  font-weight: 300;
  line-height: 1;
}
</style>
