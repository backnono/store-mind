<template>
  <view class="tab-page">
    <EmptyState icon="📋" title="暂无订单" description="完成购物后这里会显示订单记录" />
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
