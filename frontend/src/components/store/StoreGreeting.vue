<template>
  <view class="greeting-card">
    <!-- 顶部渐变描边 — 使用真实 View 避免小程序伪元素兼容性问题 -->
    <view class="card-top-bar"></view>

    <!-- 小王信息区 -->
    <view class="greeting-header">
      <view class="greeting-avatar">王</view>
      <view class="greeting-info">
        <view class="greeting-name-row">
          <text class="greeting-name">小王</text>
          <view class="greeting-live">
            <view class="live-dot-sm"></view>
            <text class="live-text">在线</text>
          </view>
        </view>
        <text class="greeting-sub">您的数字店员</text>
      </view>
    </view>

    <!-- 问候语 -->
    <view class="greeting-msg">
      欢迎光临！今天<text class="em">饮料区第二件半价</text>，零食区上了几款新品。想找什么随时问我。
    </view>

    <!-- CTA 按钮组 -->
    <view class="greeting-cta">
      <view class="cta-btn cta-primary" @tap="$emit('askWang')">
        <text>💬 问小王</text>
      </view>
      <view class="cta-btn cta-secondary" @tap="$emit('scan')">
        <text>📱 扫码购物</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
defineEmits<{
  askWang: []
  scan: []
}>()
</script>

<style lang="scss">
.greeting-card {
  margin: 36px 0 20px;
  background: #2b2820;
  border-radius: 36px;
  border: 1px solid #3d3a30;
  overflow: hidden; // 让 top-bar 圆角生效
  position: relative;
}

// 顶部渐变描边
.card-top-bar {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 5px;
  background: linear-gradient(90deg, #c9863a, #a85e1a, #c9863a);
}

// ── 头部 ──
.greeting-header {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 32px 28px 0;
  margin-top: 5px; // 留出渐变条位置
}

.greeting-avatar {
  width: 84px;
  height: 84px;
  border-radius: 50%;
  background: linear-gradient(135deg, #c9863a, #b87a2e);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 38px;
  font-weight: 700;
  color: #1c1b18;
  box-shadow: 0 8px 28px rgba(201, 134, 58, 0.30);
  flex-shrink: 0;
}

.greeting-info {
  flex: 1;
}

.greeting-name-row {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 4px;
}

.greeting-name {
  font-size: 36px;
  font-weight: 700;
  color: #ede6dc;
}

.greeting-live {
  display: flex;
  align-items: center;
  gap: 7px;
}

.live-dot-sm {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #5cb85c;
  animation: pulse-live 2s ease-in-out infinite;
  box-shadow: 0 0 6px rgba(92, 184, 92, 0.5);
}

.live-text {
  font-size: 22px;
  color: #5cb85c;
  font-weight: 500;
}

.greeting-sub {
  font-size: 24px;
  color: #9e968a;
}

// ── 问候语 ──
.greeting-msg {
  font-size: 30px;
  line-height: 1.6;
  color: #d9d0c0;
  margin: 22px 28px 28px;
}

.em {
  font-weight: 600;
  color: #deb370;
}

// ── CTA ──
.greeting-cta {
  display: flex;
  gap: 18px;
  padding: 0 28px 28px;
}

.cta-btn {
  flex: 1;
  padding: 22px 0;
  border-radius: 26px;
  font-size: 30px;
  font-weight: 600;
  text-align: center;
  transition: transform 0.12s ease, opacity 0.12s ease;

  &:active {
    transform: scale(0.96);
  }
}

.cta-primary {
  background: linear-gradient(135deg, #c9863a, #a85e1a);
  color: #1c1b18;
  box-shadow: 0 6px 24px rgba(201, 134, 58, 0.28);
}

.cta-secondary {
  background: #302c26;
  color: #d9d0c0;
  border: 1px solid #47433a;
}

@keyframes pulse-live {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}
</style>
