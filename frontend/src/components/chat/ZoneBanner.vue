<template>
  <view v-if="zoneLabel" class="zone-banner">
    <view class="zone-icon">📍</view>
    <view class="zone-info">
      <view class="zone-label">您当前在：{{ zoneLabel }}</view>
      <view v-if="zoneDesc" class="zone-desc">{{ zoneDesc }}</view>
    </view>
  </view>
</template>

<script setup lang="ts">
defineProps<{
  zoneLabel: string
  zoneDesc?: string
}>()
</script>

<style lang="scss">
.zone-banner {
  margin: 6px 20px;
  padding: 18px 22px;
  background: rgba(61, 56, 42, 0.45);
  border: 1px solid #4d402a;
  border-radius: 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  flex-shrink: 0;
}

.zone-icon {
  width: 50px;
  height: 50px;
  border-radius: 16px;
  background: rgba(77, 64, 42, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 30px;
  flex-shrink: 0;
}

.zone-info {
  flex: 1;
  min-width: 0;
}

.zone-label {
  font-weight: 600;
  font-size: 26px;
  color: #ede6dc;
}

.zone-desc {
  font-size: 21px;
  color: #9e968a;
  margin-top: 2px;
}
</style>
