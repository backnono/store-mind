<template>
  <view v-if="items.length > 0" class="recommendation-section">
    <view class="section-label">今日推荐</view>
    <view class="recommendation-list">
      <view
        v-for="item in items"
        :key="item.id"
        class="rec-item"
        @tap="$emit('select', item)"
      >
        <view class="rec-icon">{{ item.icon }}</view>
        <view class="rec-body">
          <view class="rec-title">{{ item.title }}</view>
          <view class="rec-desc">{{ item.desc }}</view>
        </view>
        <view class="rec-arrow">›</view>
      </view>
    </view>
  </view>
  <view v-else class="recommendation-section">
    <view class="section-label">今日推荐</view>
    <view class="rec-empty">
      <text class="rec-empty-text">暂无推荐</text>
    </view>
  </view>
</template>

<script setup lang="ts">
export interface RecommendationItem {
  id: string
  icon: string
  title: string
  desc: string
  context: 'promo' | 'product_detail'
  prompt: string
}

defineProps<{
  items: RecommendationItem[]
}>()

defineEmits<{
  select: [item: RecommendationItem]
}>()
</script>

<style lang="scss">
.recommendation-section {
  margin-top: 0;
  padding-bottom: 12px;
}

.section-label {
  font-size: 24px;
  font-weight: 600;
  color: #8c8070;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin-bottom: 16px;
  padding: 0 4px;
}

.recommendation-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rec-item {
  display: flex;
  align-items: center;
  gap: 22px;
  padding: 26px 22px;
  border-radius: 28px;
  background: #23211d;
  border: 1px solid #2e2b27;
  transition: background 0.15s ease;

  &:active {
    background: #2b2925;
  }
}

.rec-icon {
  font-size: 44px;
  width: 76px;
  height: 76px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #2b2820;
  border-radius: 20px;
  flex-shrink: 0;
}

.rec-body {
  flex: 1;
  min-width: 0;
}

.rec-title {
  font-size: 28px;
  font-weight: 600;
  color: #e5dbca;
  line-height: 1.3;
}

.rec-desc {
  font-size: 22px;
  color: #9e968a;
  margin-top: 4px;
  line-height: 1.3;
}

.rec-arrow {
  color: #6b6358;
  font-size: 36px;
  font-weight: 300;
  flex-shrink: 0;
}

.rec-empty {
  text-align: center;
  padding: 40px 0;
}

.rec-empty-text {
  font-size: 24px;
  color: #6b6358;
}
</style>
