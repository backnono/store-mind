// ============================================================
// GuidanceChips — 组件测试
// ============================================================
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import GuidanceChips from './GuidanceChips.vue'

describe('GuidanceChips', () => {
  const chips = [
    { text: '📦 还有几瓶？', prompt: '还有几瓶？' },
    { text: '🥤 同品类还有什么？', prompt: '同品类的都有什么？' },
  ]

  it('渲染所有 chips', () => {
    const wrapper = mount(GuidanceChips, { props: { chips } })
    const chipEls = wrapper.findAll('.guidance-chip')
    expect(chipEls.length).toBe(2)
    expect(chipEls[0].text()).toContain('还有几瓶')
    expect(chipEls[1].text()).toContain('同品类')
  })

  it('空 chips 不渲染', () => {
    const wrapper = mount(GuidanceChips, { props: { chips: [] } })
    expect(wrapper.find('.guidance-row').exists()).toBe(false)
  })

  it('点击 chip 发出 select 事件，传 prompt 值', async () => {
    const wrapper = mount(GuidanceChips, { props: { chips } })
    await wrapper.findAll('.guidance-chip')[0].trigger('tap')
    expect(wrapper.emitted('select')).toBeTruthy()
    expect(wrapper.emitted('select')![0]).toEqual(['还有几瓶？'])
  })
})
