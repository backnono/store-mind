// @vitest-environment node

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('custom tabbar layout metrics', () => {
  it('measures the rendered tabbar height and broadcasts it to pages', () => {
    const component = readFileSync(resolve(__dirname, './index.vue'), 'utf8')

    expect(component).toContain('CUSTOM_TABBAR_HEIGHT_EVENT')
    expect(component).toContain('measureTabbarHeight')
    expect(component).toContain("select('.tab-bar')")
    expect(component).toContain('boundingClientRect')
    expect(component).toContain('Taro.eventCenter.trigger(CUSTOM_TABBAR_HEIGHT_EVENT')
    expect(component).toContain('useDidShow')
    expect(component).toContain('onMounted')
  })
})
