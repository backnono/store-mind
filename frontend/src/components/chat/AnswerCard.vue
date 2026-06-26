<template>
  <view class="answer-card" :class="`card-${card.type}`">
    <!-- Product — 图一风格: emoji + 标题 + 副标题 + 位置标签 -->
    <template v-if="card.type === 'product'">
      <view class="card-header">
        <view class="card-emoji">{{ productEmoji }}</view>
        <view class="card-main">
          <view class="card-title">{{ card.name ?? '商品' }}</view>
          <view v-if="card.location" class="card-sub">{{ card.location }}</view>
        </view>
      </view>
      <view v-if="hasMeta" class="card-tags">
        <text v-if="card.location" class="tag location">📍 {{ card.location }}</text>
        <text v-if="card.price" class="tag price">¥{{ card.price }}</text>
      </view>
    </template>

    <!-- Inventory -->
    <template v-if="card.type === 'inventory'">
      <view class="card-header">
        <view class="card-emoji">📦</view>
        <view class="card-main">
          <view class="card-title">{{ card.name ?? '商品' }}</view>
          <view v-if="card.location" class="card-sub">{{ card.location }}</view>
        </view>
        <view class="stock-badge" :class="stockLevel">{{ stockLabel }}</view>
      </view>
      <view class="card-body">
        <view class="stock-detail">
          <text class="stock-count">库存数量：{{ card.quantity ?? '--' }}</text>
        </view>
      </view>
    </template>

    <!-- Promotion -->
    <template v-if="card.type === 'promotion'">
      <view class="promo-title">{{ card.title }}</view>
      <view class="promo-content">{{ card.content }}</view>
      <view v-if="card.validity" class="promo-validity">有效期至 {{ card.validity }}</view>
    </template>

    <!-- FAQ / Fallback -->
    <template v-if="card.type === 'faq'">
      <view class="faq-title">{{ card.title }}</view>
      <view class="faq-content">{{ card.content }}</view>
    </template>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ChatCard } from '@/types/customerQa'

const props = defineProps<{
  card: ChatCard
}>()

const productEmoji = computed(() => {
  const name = (props.card.name ?? '').toLowerCase()
  if (name.includes('可乐') || name.includes('cola')) return '🥤'
  if (name.includes('雪碧') || name.includes('sprite')) return '🫧'
  if (name.includes('元气') || name.includes('汽水')) return '🍑'
  if (name.includes('薯片') || name.includes('零食') || name.includes('乐事')) return '🍿'
  if (name.includes('巧克力') || name.includes('布朗尼')) return '🍫'
  if (name.includes('蛋糕') || name.includes('面包')) return '🍰'
  return '🛒'
})

const hasMeta = computed(() => !!props.card.location || props.card.price !== undefined)

const stockLevel = computed(() => {
  const q = props.card.quantity
  if (q === undefined) return 'unknown'
  if (q > 20) return 'high'
  if (q > 5) return 'medium'
  if (q > 0) return 'low'
  return 'none'
})

const stockLabel = computed(() => {
  const q = props.card.quantity
  if (q === undefined) return ''
  if (q > 20) return '库存充足'
  if (q > 5) return '库存有限'
  if (q > 0) return '即将售罄'
  return '已售罄'
})
</script>

<style lang="scss">
.answer-card {
  background: #2a2a36;
  border-radius: 22px;
  overflow: hidden;
  border: 1px solid #333345;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px 20px;
}

.card-emoji {
  width: 64px;
  height: 64px;
  border-radius: 14px;
  background: #34323a;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 34px;
  flex-shrink: 0;
  line-height: 1;
}

.card-main {
  flex: 1;
  min-width: 0;
}

.card-title {
  font-size: 28px;
  font-weight: 600;
  color: #f0f0f5;
  line-height: 1.3;
}

.card-sub {
  font-size: 22px;
  color: #888;
  margin-top: 4px;
}

.card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  padding: 0 20px 18px;
}

.tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 20px;
  padding: 6px 14px;
  border-radius: 20px;
  font-weight: 500;

  &.location {
    background: rgba(232, 166, 69, 0.14);
    border: 1px solid rgba(232, 166, 69, 0.25);
    color: #e8a645;
  }

  &.price {
    background: rgba(92, 184, 92, 0.12);
    color: #6bc46b;
  }
}

.card-body {
  padding: 0 20px 18px;
}

// Inventory badges
.stock-badge {
  font-size: 20px;
  padding: 5px 14px;
  border-radius: 18px;
  font-weight: 600;
  flex-shrink: 0;

  &.high { background: rgba(92, 184, 92, 0.14); color: #5cb85c; }
  &.medium { background: rgba(240, 173, 78, 0.14); color: #f0ad4e; }
  &.low { background: rgba(249, 107, 46, 0.14); color: #f96b2e; }
  &.none { background: rgba(217, 83, 79, 0.11); color: #d9534f; }
  &.unknown { background: rgba(158, 150, 138, 0.10); color: #8c8070; }
}

.stock-detail { padding-top: 2px; }
.stock-count { font-size: 24px; color: #bfb6a4; }

// Promotion
.promo-title { font-size: 26px; font-weight: 600; color: #deb370; padding: 18px 20px 6px; }
.promo-content { font-size: 24px; color: #ede6dc; padding: 0 20px 10px; line-height: 1.5; }
.promo-validity { font-size: 20px; color: #9e968a; padding: 0 20px 18px; }

// FAQ
.faq-title { font-size: 26px; font-weight: 600; color: #ede6dc; padding: 18px 20px 6px; }
.faq-content { font-size: 24px; color: #bfb6a4; padding: 0 20px 18px; line-height: 1.5; }
</style>
