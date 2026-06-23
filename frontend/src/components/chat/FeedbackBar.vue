<template>
  <view class="feedback-row">
    <view
      class="feedback-btn"
      :class="{ active: selected === 1 }"
      @tap="$emit('submit', 1)"
    >👍</view>
    <view
      class="feedback-btn"
      :class="{ active: selected === 0 }"
      @tap="$emit('submit', 0)"
    >👎</view>
  </view>
</template>

<script setup lang="ts">
defineProps<{
  messageId?: number
  selected?: 0 | 1
}>()

defineEmits<{
  submit: [value: 0 | 1]
}>()
</script>

<style lang="scss">
.feedback-row {
  display: flex;
  gap: 6px;
  margin-top: 10px;
  padding-left: 6px;
}

.feedback-btn {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 1px solid #3d3a35;
  background: transparent;
  color: #7a6e5c;
  font-size: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;

  &.active {
    background: rgba(92, 184, 92, 0.16);
    border-color: #5cb85c;
    color: #5cb85c;
  }
}
</style>
