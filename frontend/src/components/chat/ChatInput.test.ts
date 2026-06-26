// @vitest-environment node

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('ChatInput layout', () => {
  it('stays in the chat flex column so it cannot overlap the message scroll area', () => {
    const component = readFileSync(resolve(__dirname, './ChatInput.vue'), 'utf8')

    expect(component).toContain('flex-shrink: 0;')
    expect(component).not.toMatch(/^\s*position:\s*fixed;/m)
    expect(component).not.toContain('bottom: calc(104px + env(safe-area-inset-bottom));')
    expect(component).not.toMatch(/^\s*left:\s*0;/m)
    expect(component).not.toMatch(/^\s*right:\s*0;/m)
  })
})
