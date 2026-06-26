<template>
  <view v-if="chips.length > 0" class="guidance-row">
    <view
      v-for="chip in chips"
      :key="chip.prompt"
      class="guidance-chip"
      @tap="$emit('select', chip.prompt)"
    >
      <text class="chip-icon">{{ chipIcon(chip) }}</text>
      <text>{{ chip.text }}</text>
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

function chipIcon(chip: GuidanceChip): string {
  const t = chip.text
  if (t.includes('几包') || t.includes('库存') || t.includes('补货')) return '📦'
  if (t.includes('活动') || t.includes('优惠') || t.includes('打折')) return '🏷️'
  if (t.includes('饮料') || t.includes('喝') || t.includes('咖啡')) return '🥤'
  if (t.includes('付款') || t.includes('支付') || t.includes('会员')) return '💳'
  if (t.includes('推荐') || t.includes('还有什么')) return '🔍'
  if (t.includes('便宜') || t.includes('价格') || t.includes('多少钱')) return '💰'
  if (t.includes('积分')) return '🎫'
  if (t.includes('位置') || t.includes('哪里') || t.includes('在哪')) return '📍'
  return '💡'
}
</script>

<style lang="scss">
.guidance-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 16px;
}

.guidance-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 24px;
  padding: 12px 22px;
  border-radius: 28px;
  border: 1px solid #4d402a;
  background: transparent;
  color: #d4a44c;
  font-weight: 500;
  transition: all 0.15s ease;
  white-space: nowrap;

  &:active {
    background: rgba(201, 134, 58, 0.18);
    border-color: #c9863a;
  }
}

.chip-icon {
  font-size: 26px;
  line-height: 1;
}
</style>
