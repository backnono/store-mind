// @vitest-environment node

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('store page layout', () => {
  it('reserves the visible custom tab bar area at the bottom', () => {
    const styles = readFileSync(resolve(__dirname, './index.scss'), 'utf8')

    expect(styles).toContain('padding-bottom: calc(32px + 220px + env(safe-area-inset-bottom));')
  })
})
