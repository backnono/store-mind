// ============================================================
// QR 解析工具 — 单元测试
// ============================================================
import { describe, it, expect } from 'vitest'
import { parseQR, buildZoneScanUrl } from './qr'

describe('parseQR', () => {
  it('解析 storemind:// 协议', () => {
    const result = parseQR('storemind://zone?store_id=1&zone_id=2&shelf_id=5')
    expect(result).toEqual({ storeId: 1, zoneId: 2, shelfId: 5 })
  })

  it('解析 https:// 链接', () => {
    const result = parseQR('https://example.com/miniapp/chat?store_id=1&zone_id=2&shelf_id=5')
    expect(result).toEqual({ storeId: 1, zoneId: 2, shelfId: 5 })
  })

  it('缺少 zone_id 和 shelf_id', () => {
    const result = parseQR('storemind://zone?store_id=42')
    expect(result).toEqual({ storeId: 42, zoneId: undefined, shelfId: undefined })
  })

  it('无效字符串返回 null', () => {
    expect(parseQR('')).toBeNull()
    expect(parseQR('not a url')).toBeNull()
  })

  it('缺少 store_id 返回 null', () => {
    expect(parseQR('storemind://zone?zone_id=1')).toBeNull()
  })

  it('store_id 为 0 返回 null', () => {
    expect(parseQR('storemind://zone?store_id=0')).toBeNull()
  })

  it('只传 zone_id', () => {
    const result = parseQR('https://example.com/miniapp/chat?store_id=10&zone_id=3')
    expect(result).toEqual({ storeId: 10, zoneId: 3, shelfId: undefined })
  })
})

describe('buildZoneScanUrl', () => {
  it('完整参数', () => {
    expect(buildZoneScanUrl({ storeId: 1, zoneId: 2, shelfId: 5 })).toBe(
      '/pages/chat/index?entry=zone_scan&store_id=1&zone_id=2&shelf_id=5',
    )
  })

  it('仅有 storeId', () => {
    expect(buildZoneScanUrl({ storeId: 7 })).toBe(
      '/pages/chat/index?entry=zone_scan&store_id=7',
    )
  })
})
