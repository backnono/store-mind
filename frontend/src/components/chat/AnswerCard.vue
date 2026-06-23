<template>
  <view class="answer-card" :class="`card-${card.type}`">
    <!-- Product -->
    <template v-if="card.type === 'product'">
      <view class="card-header">
        <view class="product-icon">{{ productEmoji }}</view>
        <view class="product-main">
          <view class="product-name">{{ card.name ?? '商品' }}</view>
          <view v-if="card.location" class="product-spec">{{ card.location }}</view>
        </view>
        <view v-if="card.price" class="product-price">{{ card.price }}</view>
      </view>
      <view v-if="hasMeta" class="card-body">
        <view class="meta-row">
          <text v-if="card.location" class="tag location">📍 {{ card.location }}</text>
          <text v-if="card.quantity !== undefined" class="tag stock">📦 库存 {{ card.quantity }}</text>
        </view>
      </view>
    </template>

    <!-- Inventory -->
    <template v-if="card.type === 'inventory'">
      <view class="card-header">
        <view class="product-icon">📦</view>
        <view class="product-main">
          <view class="product-name">{{ card.name ?? '商品' }}</view>
          <view v-if="card.location" class="product-spec">{{ card.location }}</view>
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
  if (name.includes('薯片') || name.includes('零食')) return '🍿'
  if (name.includes('巧克力') || name.includes('布朗尼')) return '🍫'
  return '🛒'
})

const hasMeta = computed(() => !!props.card.location || props.card.quantity !== undefined)

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
  background: #342f28;
  border-radius: 22px;
  overflow: hidden;
  border: 1px solid #3d3a35;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px 20px;
}

.product-icon {
  width: 68px;
  height: 68px;
  border-radius: 14px;
  background: #3a352e;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 36px;
  flex-shrink: 0;
}

.product-main {
  flex: 1;
  min-width: 0;
}

.product-name {
  font-size: 26px;
  font-weight: 600;
  color: #ede6dc;
  line-height: 1.3;
}

.product-spec {
  font-size: 20px;
  color: #9e968a;
  margin-top: 2px;
}

.product-price {
  font-size: 30px;
  font-weight: 700;
  color: #deb370;
  white-space: nowrap;
  flex-shrink: 0;
}

.card-body {
  padding: 0 20px 18px;
}

.meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.tag {
  font-size: 20px;
  padding: 5px 14px;
  border-radius: 18px;
  font-weight: 500;

  &.location {
    background: rgba(201, 134, 58, 0.14);
    color: #d4a44c;
  }

  &.stock {
    background: rgba(92, 184, 92, 0.12);
    color: #6bc46b;
  }
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
