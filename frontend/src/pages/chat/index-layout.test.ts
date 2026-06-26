// @vitest-environment node

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('chat page layout', () => {
  it('uses the prototype flex column layout so messages scroll above the composer', () => {
    const page = readFileSync(resolve(__dirname, './index.vue'), 'utf8')
    const styles = readFileSync(resolve(__dirname, './index.scss'), 'utf8')
    const blockFor = (selector: string) => {
      const start = styles.indexOf(`${selector} {`)
      const end = styles.indexOf('\n}', start)
      return styles.slice(start, end)
    }
    const bodyBlock = blockFor('.chat-body')
    const scrollAreaBlock = blockFor('.chat-scroll-area')
    const composerSlotBlock = blockFor('.chat-composer-slot')
    const tabbarReserveBlock = blockFor('.chat-tabbar-reserve')

    // Template: chat page receives runtime layout metrics
    expect(page).toContain(':style="chatPageStyle"')

    // Template: chat-body as positioning scope
    expect(page).toContain('class="chat-body"')
    expect(page).not.toContain('class="chat-conversation"')

    // Template: scroll-view owned by page, not by MessageList
    expect(page).toContain('<scroll-view')
    expect(page).toContain('class="chat-scroll-area"')
    expect(page).toContain('scroll-y')
    expect(page).toContain(':scroll-into-view="scrollIntoView"')
    expect(page).toContain('@scroll="handleMessageScroll"')
    expect(page).toContain(':enable-flex="true"')

    // Template: scroll content ends inside chat-body; composer and tabbar reserve are sibling modules
    expect(page).not.toContain('class="composer-spacer"')

    // Template: composer slot with ChatInput
    expect(page).toContain('class="chat-composer-slot"')
    expect(page).toContain('class="chat-tabbar-reserve"')
    expect(page).toContain("select('.chat-scroll-area')")

    // Template: scroll state in script
    expect(page).toContain('const scrollIntoView = ref')
    expect(page).toContain("scrollIntoView.value = ''")
    expect(page).toContain('const chatPageStyle = computed')
    expect(page).toContain('--chat-message-viewport-height')
    expect(page).toContain('--chat-tabbar-reserve')
    expect(page).toContain('measureSelectorHeight')
    expect(page).toContain("measureSelectorHeight('.chat-header')")
    expect(page).toContain("measureSelectorHeight('.chat-composer-slot')")
    expect(page).toContain('Taro.getSystemInfoSync()')
    expect(page).toContain('Taro.eventCenter.on(CUSTOM_TABBAR_HEIGHT_EVENT')
    expect(page).toContain('Taro.eventCenter.off(CUSTOM_TABBAR_HEIGHT_EVENT')

    // Template: composer and tabbar reserve are in normal page flow after chat-body
    expect(page).toMatch(/<view class="chat-body">[\s\S]*<\/scroll-view>\s*<\/view>\s*[\s\S]*<view class="chat-composer-slot">[\s\S]*<ChatInput/)
    expect(page).toMatch(/<view class="chat-composer-slot">[\s\S]*<\/view>\s*<view class="chat-tabbar-reserve"><\/view>/)

    // Styles: design tokens
    expect(styles).toContain('--chat-tabbar-reserve')
    expect(styles).toContain('--chat-composer-height')
    expect(styles).toContain('--chat-message-viewport-height')

    // Styles: chat-body is only the message scroll module
    expect(styles).toContain('.chat-body')
    expect(bodyBlock).toContain('overflow: hidden;')
    expect(bodyBlock).toContain('box-sizing: border-box;')
    expect(bodyBlock).toContain('flex: 0 0 var(--chat-message-viewport-height);')
    expect(bodyBlock).toContain('height: var(--chat-message-viewport-height);')
    expect(bodyBlock).toContain('min-height: 0;')
    expect(bodyBlock).toContain('display: flex;')
    expect(bodyBlock).toContain('flex-direction: column;')
    expect(bodyBlock).not.toContain('position: relative;')
    expect(bodyBlock).not.toContain('padding-bottom:')

    // Styles: scroll-view has an explicit measured viewport for WeChat miniapp scrolling
    expect(styles).toContain('.chat-scroll-area')
    expect(scrollAreaBlock).toContain('height: var(--chat-message-viewport-height);')
    expect(scrollAreaBlock).toContain('min-height: 0;')
    expect(scrollAreaBlock).toContain('overflow-y: auto;')
    expect(scrollAreaBlock).not.toContain('position: absolute;')
    expect(scrollAreaBlock).not.toContain('position: fixed;')

    // Styles: composer slot is a normal flow module, not an overlay
    expect(styles).toContain('.chat-composer-slot')
    expect(composerSlotBlock).toContain('flex-shrink: 0;')
    expect(composerSlotBlock).not.toContain('position: absolute;')
    expect(composerSlotBlock).not.toContain('position: fixed;')
    expect(composerSlotBlock).not.toContain('bottom:')

    // Styles: tabbar reserve is a normal flow module for WeChat custom tabbar overlay
    expect(styles).toContain('.chat-tabbar-reserve')
    expect(tabbarReserveBlock).toContain('height: var(--chat-tabbar-reserve);')
    expect(tabbarReserveBlock).toContain('flex-shrink: 0;')

    // Styles: composer child still has its own styles
    expect(styles).toContain('.chat-composer-slot .composer')

    // Styles: no old layout artifacts
    expect(styles).not.toContain('.chat-conversation')
    expect(styles).not.toContain('padding-bottom: calc(var(--chat-tabbar-height) + env(safe-area-inset-bottom));')
    expect(styles).not.toContain('bottom: calc(104px')
    expect(styles).not.toContain('bottom: calc(112px + 220px + env(safe-area-inset-bottom));')
  })
})
