<template>
  <view v-if="message" class="inline-notice">
    <text class="notice-text">{{ message }}</text>
    <text class="notice-dismiss" @tap="$emit('dismiss')">✕</text>
  </view>
</template>

<script setup lang="ts">
defineProps<{
  message: string
}>()

defineEmits<{
  dismiss: []
}>()
</script>

<style lang="scss">
.inline-notice {
  margin: 16px 32px;
  padding: 20px 24px;
  background: #3a2010;
  border: 1px solid #6b3a1a;
  border-radius: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
}

.notice-text {
  flex: 1;
  font-size: 26px;
  color: #e8a86a;
  line-height: 1.5;
}

.notice-dismiss {
  font-size: 28px;
  color: #c97d3a;
  padding: 4px 8px;
}
</style>
