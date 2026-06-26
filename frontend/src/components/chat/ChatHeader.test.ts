// @vitest-environment node

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('ChatHeader miniapp styles', () => {
  it('keeps header layout styles outside Vue scoped selectors for WeChat WXSS', () => {
    const component = readFileSync(resolve(__dirname, './ChatHeader.vue'), 'utf8')
    const pageStyles = readFileSync(resolve(__dirname, '../../pages/chat/index.scss'), 'utf8')

    expect(component).not.toContain('<style lang="scss" scoped>')
    expect(pageStyles).toContain('.chat-header')
    expect(component).toContain('class="chat-title-bar"')
    expect(component).toContain('class="btn-new-chat"')
    expect(component).toContain('class="btn-history"')
    expect(pageStyles).toContain('.chat-title-bar')
    expect(pageStyles).toContain('.btn-new-chat')
    expect(pageStyles).toContain('.btn-history')
  })
})
