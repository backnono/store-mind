<template>
  <view class="empty-state">
    <view class="empty-icon">{{ icon }}</view>
    <view class="empty-title">{{ title }}</view>
    <view v-if="description" class="empty-desc">{{ description }}</view>
  </view>
</template>

<script setup lang="ts">
defineProps<{
  icon: string
  title: string
  description?: string
}>()
</script>

<style lang="scss">
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 24px;
  padding: 80px 40px;
}

.empty-icon {
  font-size: 96px;
  opacity: 0.45;
}

.empty-title {
  font-size: 32px;
  font-weight: 600;
  color: #bfb6a4;
}

.empty-desc {
  font-size: 24px;
  color: #8c8070;
  text-align: center;
}
</style>
