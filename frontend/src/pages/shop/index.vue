<template>
  <view class="tab-page">
    <EmptyState icon="🛒" title="购物车是空的" description="扫码商品条码加入购物车" />
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
