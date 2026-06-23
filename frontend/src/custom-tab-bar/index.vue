<template>
  <view class="tab-bar">
    <view
      v-for="tab in tabs"
      :key="tab.key"
      class="tab-item"
      :class="{ active: selected === tab.index }"
      @tap="onTabTap(tab.key, tab.index)"
    >
      <!-- 小王 Tab：中间突出样式 -->
      <template v-if="tab.key === 'chat'">
        <view class="wang-avatar" :class="{ active: selected === tab.index }">
          <text class="wang-char">王</text>
        </view>
        <text class="tab-label">{{ tab.label }}</text>
      </template>

      <!-- 其他 Tab -->
      <template v-else>
        <text class="tab-icon">{{ tab.icon }}</text>
        <text class="tab-label">{{ tab.label }}</text>
      </template>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Taro, { useDidShow } from '@tarojs/taro'

const tabs = [
  { key: 'home', label: '首页', icon: '🏪', index: 0 },
  { key: 'shop', label: '购物', icon: '🛒', index: 1 },
  { key: 'chat', label: '小王', icon: '', index: 2 },
  { key: 'orders', label: '订单', icon: '📋', index: 3 },
  { key: 'profile', label: '我的', icon: '👤', index: 4 },
]

const pageMap: Record<string, string> = {
  home: '/pages/store/index',
  shop: '/pages/shop/index',
  chat: '/pages/chat/index',
  orders: '/pages/orders/index',
  profile: '/pages/profile/index',
}

const routeToTab: Record<string, number> = {
  'pages/store/index': 0,
  'pages/shop/index': 1,
  'pages/chat/index': 2,
  'pages/orders/index': 3,
  'pages/profile/index': 4,
}

const selected = ref(0)

// ── 每次页面显示时重新检测当前 tab ──────────────────────
useDidShow(() => {
  const pages = Taro.getCurrentPages()
  if (pages.length > 0) {
    const cur = pages[pages.length - 1]
    const route = cur.route ?? ''
    const matched = routeToTab[route]
    if (matched !== undefined) {
      selected.value = matched
    }
  }
})

function onTabTap(key: string, index: number) {
  // 乐观更新 UI（tab 切换无需等路由确认）
  selected.value = index
  Taro.switchTab({ url: pageMap[key] })
}
</script>

<style lang="scss">
.tab-bar {
  flex-shrink: 0;
  display: flex;
  align-items: flex-end;
  background: #1c1b18;
  border-top: 1px solid #2b2925;
  padding: 8px 0 0;
  padding-bottom: calc(env(safe-area-inset-bottom));
  height: calc(104px + env(safe-area-inset-bottom));
}

.tab-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
  gap: 3px;
  padding-bottom: 10px;
  color: #7a6e5c;
  font-size: 20px;
  font-weight: 500;
  transition: color 0.2s ease;

  &.active {
    color: #d4a44c;

    .wang-avatar {
      box-shadow: 0 4px 20px rgba(201, 134, 58, 0.45);
    }
  }
}

.tab-icon {
  font-size: 36px;
  line-height: 1;
}

.tab-label {
  font-size: 19px;
  line-height: 1;
}

// ── 小王头像样式 ─────────────────────────────────────
.wang-avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: linear-gradient(135deg, #c9863a, #a85e1a);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 2px;
  transition: box-shadow 0.2s ease;
  // 默认已有发光，非 active 状态稍弱
  box-shadow: 0 4px 16px rgba(201, 134, 58, 0.22);

  &.active {
    box-shadow: 0 4px 20px rgba(201, 134, 58, 0.45);
  }
}

.wang-char {
  font-size: 34px;
  font-weight: 700;
  color: #1c1b18;
}
</style>
