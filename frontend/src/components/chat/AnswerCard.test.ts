// ============================================================
// AnswerCard — 组件测试
// ============================================================
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AnswerCard from './AnswerCard.vue'

describe('AnswerCard', () => {
  it('渲染 product 卡片', () => {
    const wrapper = mount(AnswerCard, {
      props: {
        card: {
          type: 'product',
          name: '可口可乐',
          location: 'B-02 货架',
          price: '¥3.50',
        },
      },
    })

    expect(wrapper.find('.product-name').text()).toBe('可口可乐')
    expect(wrapper.find('.product-spec').text()).toBe('B-02 货架')
    expect(wrapper.find('.product-price').text()).toBe('¥3.50')
  })

  it('渲染 inventory 卡片带库存', () => {
    const wrapper = mount(AnswerCard, {
      props: {
        card: {
          type: 'inventory',
          name: '元气森林',
          location: 'B-03 货架',
          quantity: 42,
          sku_id: 101,
        },
      },
    })

    expect(wrapper.find('.product-name').text()).toBe('元气森林')
    expect(wrapper.find('.stock-badge').text()).toBe('库存充足')
    expect(wrapper.find('.stock-badge').classes()).toContain('high')
    expect(wrapper.find('.stock-count').text()).toContain('42')
  })

  it('渲染 inventory 卡片 — 即将售罄', () => {
    const wrapper = mount(AnswerCard, {
      props: {
        card: {
          type: 'inventory',
          name: '限量商品',
          quantity: 2,
        },
      },
    })

    expect(wrapper.find('.stock-badge').text()).toBe('即将售罄')
    expect(wrapper.find('.stock-badge').classes()).toContain('low')
  })

  it('渲染 inventory 卡片 — 已售罄', () => {
    const wrapper = mount(AnswerCard, {
      props: {
        card: {
          type: 'inventory',
          name: '热门商品',
          quantity: 0,
        },
      },
    })

    expect(wrapper.find('.stock-badge').text()).toBe('已售罄')
    expect(wrapper.find('.stock-badge').classes()).toContain('none')
  })

  it('渲染 promotion 卡片', () => {
    const wrapper = mount(AnswerCard, {
      props: {
        card: {
          type: 'promotion',
          title: '饮料第二件半价',
          content: '可乐、雪碧等参与活动',
          validity: '06-25 23:59',
        },
      },
    })

    expect(wrapper.find('.promo-title').text()).toBe('饮料第二件半价')
    expect(wrapper.find('.promo-content').text()).toBe('可乐、雪碧等参与活动')
    expect(wrapper.find('.promo-validity').text()).toContain('06-25 23:59')
  })

  it('渲染 faq 卡片', () => {
    const wrapper = mount(AnswerCard, {
      props: {
        card: {
          type: 'faq',
          title: '如何付款？',
          content: '支持微信支付、支付宝。',
        },
      },
    })

    expect(wrapper.find('.faq-title').text()).toBe('如何付款？')
    expect(wrapper.find('.faq-content').text()).toBe('支持微信支付、支付宝。')
  })
})
