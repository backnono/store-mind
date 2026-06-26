// @vitest-environment node

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('MessageList content renderer', () => {
  it('renders message content without owning the scroll-view', () => {
    const component = readFileSync(resolve(__dirname, './MessageList.vue'), 'utf8')

    expect(component).not.toContain('<scroll-view')
    expect(component).not.toContain('scroll-y')
    expect(component).not.toContain('scrollIntoView')
    expect(component).not.toContain('scrollTop')
    expect(component).not.toContain('viewportHeight')
    expect(component).not.toContain('stickToBottomRequest')
    expect(component).toContain('<view class="messages-inner">')
    expect(component).toContain('class="messages-bottom-spacer"')
    expect(component).toContain('filteredMessages')
    expect(component).toContain('MessageBubble')
  })

  it('keeps spacing on the content wrapper, not on an owned scroll surface', () => {
    const component = readFileSync(resolve(__dirname, './MessageList.vue'), 'utf8')

    expect(component).toContain('.messages-inner')
    expect(component).toContain('padding: 24px 20px 0;')
    expect(component).toContain('.messages-bottom-spacer')
    expect(component).toContain('height: 16px;')
    expect(component).not.toContain('.messages {')
  })
})
