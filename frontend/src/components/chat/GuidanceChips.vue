<template>
  <view v-if="chips.length > 0" class="guidance-row">
    <view
      v-for="chip in chips"
      :key="chip.prompt"
      class="guidance-chip"
      @tap="$emit('select', chip.prompt)"
    >
      {{ chip.text }}
    </view>
  </view>
</template>

<script setup lang="ts">
import type { GuidanceChip } from '@/types/customerQa'

defineProps<{
  chips: GuidanceChip[]
}>()

defineEmits<{
  select: [prompt: string]
}>()
</script>

<style lang="scss">
.guidance-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 16px;
}

.guidance-chip {
  font-size: 23px;
  padding: 10px 20px;
  border-radius: 28px;
  border: 1px solid #4d402a;
  background: transparent;
  color: #deb370;
  font-weight: 500;
  transition: all 0.15s ease;
  white-space: nowrap;

  &:active {
    background: rgba(201, 134, 58, 0.18);
    border-color: #c9863a;
  }
}
</style>
