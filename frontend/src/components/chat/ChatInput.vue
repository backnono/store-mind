<template>
  <view class="composer">
    <view class="composer-input-wrap">
      <input
        class="composer-input"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        confirm-type="send"
        @input="onInput"
        @confirm="$emit('send')"
      />
    </view>
    <view
      class="composer-send"
      :class="{ disabled: !canSend }"
      @tap="$emit('send')"
    >↑</view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  modelValue: string
  disabled: boolean
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  send: []
}>()

const canSend = computed(() => props.modelValue.trim().length > 0 && !props.disabled)

function onInput(e: InputEvent) {
  const payload = e as unknown as { detail?: { value?: string }; target?: { value?: string } }
  emit('update:modelValue', payload.detail?.value ?? payload.target?.value ?? '')
}
</script>

<style lang="scss">
.composer {
  flex-shrink: 0;
  width: 100%;
  box-sizing: border-box;
  padding: 16px 20px 18px;
  background: #1c1b18;
  border-top: 1px solid #2b2925;
  display: flex;
  gap: 14px;
  align-items: flex-end;
}

.composer-input-wrap {
  flex: 1;
  background: #2b2820;
  border-radius: 40px;
  padding: 18px 28px;
  display: flex;
  align-items: center;
  border: 1px solid #3d3a35;
  transition: border-color 0.2s ease;
}

.composer-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  font-size: 29px;
  color: #ede6dc;
  font-family: inherit;
  width: 100%;
  line-height: 1.4;

  &::placeholder {
    color: #6b6358;
  }
}

.composer-send {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: linear-gradient(135deg, #c9863a, #a85e1a);
  color: #1c1b18;
  font-size: 34px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: transform 0.12s ease, opacity 0.2s ease;
  box-shadow: 0 4px 18px rgba(201, 134, 58, 0.24);

  &:active {
    transform: scale(0.93);
  }

  &.disabled {
    opacity: 0.38;
    box-shadow: none;
  }
}
</style>
