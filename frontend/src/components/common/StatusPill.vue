<template>
  <view class="status-pill" :class="variant">
    <view v-if="variant === 'online' || variant === 'live'" class="dot"></view>
    <text>{{ label }}</text>
  </view>
</template>

<script setup lang="ts">
defineProps<{
  variant: 'online' | 'live' | 'info' | 'offline'
  label: string
}>()
</script>

<style lang="scss">
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 20px;
  border-radius: 32px;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
}

.online,
.live {
  background: rgba(76, 175, 80, 0.15);
  color: #5cb85c;

  .dot {
    background: #5cb85c;
    animation: pulse-live 2s ease-in-out infinite;
  }
}

.info {
  background: rgba(201, 134, 58, 0.15);
  color: #d4a44c;
}

.offline {
  background: rgba(158, 150, 138, 0.1);
  color: #8c8070;
}

@keyframes pulse-live {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}
</style>
