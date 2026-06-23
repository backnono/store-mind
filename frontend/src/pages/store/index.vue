<template>
  <view class="store-page">
    <!-- 门店 Hero -->
    <view class="store-hero">
      <view class="store-icon-wrap">
        <view class="store-icon">🏪</view>
        <view class="store-badge">
          <text class="badge-text">24h</text>
        </view>
      </view>
      <view class="store-name">{{ storeName }}</view>
      <view class="store-addr">{{ storeAddress }}</view>
    </view>

    <!-- 数字店员问候卡片（含 "问小王" + "扫码购物"） -->
    <StoreGreeting
      @ask-wang="navigateToAsk"
      @scan="handleScan"
    />

    <!-- 今日推荐 -->
    <StoreRecommendationList
      :items="recommendations"
      @select="navigateToPromo"
    />

    <!-- 错误提示 -->
    <InlineNotice v-if="scanError" :message="scanError" @dismiss="scanError = ''" />
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Taro from '@tarojs/taro'
import { useSessionStore } from '@/stores/session'
import { parseQR, buildZoneScanUrl } from '@/utils/qr'
import StoreGreeting from '@/components/store/StoreGreeting.vue'
import StoreRecommendationList from '@/components/store/StoreRecommendationList.vue'
import InlineNotice from '@/components/common/InlineNotice.vue'
import type { RecommendationItem } from '@/components/store/StoreRecommendationList.vue'

const sessionStore = useSessionStore()

const storeName = ref('阳光便利店 No.3')
const storeAddress = ref('建国路 128 号 · 24 小时无人店')
const scanError = ref('')

const recommendations = ref<RecommendationItem[]>([
  {
    id: 'promo_1',
    icon: '🏷',
    title: '饮料区第二件半价',
    desc: '可乐、雪碧、元气森林 · 截至今晚',
    context: 'promo',
    prompt: '饮料第二件半价',
  },
  {
    id: 'new_1',
    icon: '🆕',
    title: '新品上架 · 巧克力布朗尼',
    desc: '零食区 A-03 · ¥12.90',
    context: 'product_detail',
    prompt: '巧克力布朗尼',
  },
  {
    id: 'hot_1',
    icon: '🔥',
    title: '今日热销 · 元气森林白桃味',
    desc: '低糖汽水 · 饮料区 B-03 · ¥5.00',
    context: 'product_detail',
    prompt: '元气森林白桃味',
  },
])

onMounted(() => {
  sessionStore.setStore(1)
})

function navigateToAsk() {
  Taro.switchTab({ url: '/pages/chat/index' })
}

function navigateToPromo(_item: RecommendationItem) {
  Taro.switchTab({ url: '/pages/chat/index' })
}

async function handleScan() {
  try {
    const res = await Taro.scanCode({ onlyFromCamera: true })
    const parsed = parseQR(res.result)
    if (parsed) {
      sessionStore.setEntryContext({
        mode: 'zone_scan',
        storeId: parsed.storeId,
        zoneId: parsed.zoneId,
        shelfId: parsed.shelfId,
      })
      Taro.switchTab({ url: '/pages/chat/index' })
    } else {
      scanError.value = '无法识别该二维码，请扫描货架或商品上的码'
    }
  } catch (err: unknown) {
    const e = err as { errMsg?: string }
    if (e?.errMsg?.includes('cancel')) return
    scanError.value = '扫码失败，请重试'
  }
}
</script>

<style lang="scss" src="./index.scss"></style>
