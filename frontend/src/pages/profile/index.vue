<template>
  <view class="tab-page">
    <EmptyState icon="👤" title="个人中心" description="会员卡、积分、优惠券等" />
  </view>
</template>

<script setup lang="ts">
import EmptyState from '@/components/common/EmptyState.vue'
</script>

<style>
.tab-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #1c1b18;
}
</style>
