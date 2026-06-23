// ============================================================
// QR 码解析 — 支持两种格式：
//   1. storemind://zone?store_id=1&zone_id=2&shelf_id=5
//   2. https://.../miniapp/chat?store_id=1&zone_id=2&shelf_id=5
// ============================================================

export interface ParsedQR {
  storeId: number
  zoneId?: number
  shelfId?: number
}

/**
 * 解析货架/商品 QR 码
 * @returns 解析结果，失败返回 null
 */
export function parseQR(raw: string): ParsedQR | null {
  if (!raw) return null

  let url: URL

  try {
    url = new URL(raw)
  } catch {
    // 可能是 storemind:// 协议
    const replaced = raw.replace(/^storemind:\/\//, 'https://storemind.local/')
    try {
      url = new URL(replaced)
    } catch {
      return null
    }
  }

  const params = url.searchParams
  const storeId = parseInt(params.get('store_id') ?? '', 10)
  if (isNaN(storeId) || storeId <= 0) return null

  const zoneIdRaw = params.get('zone_id')
  const shelfIdRaw = params.get('shelf_id')

  return {
    storeId,
    zoneId: zoneIdRaw ? parseInt(zoneIdRaw, 10) || undefined : undefined,
    shelfId: shelfIdRaw ? parseInt(shelfIdRaw, 10) || undefined : undefined,
  }
}

/**
 * 构建 zone_scan 入口跳转 URL
 */
export function buildZoneScanUrl(params: ParsedQR): string {
  let url = `/pages/chat/index?entry=zone_scan&store_id=${params.storeId}`
  if (params.zoneId !== undefined) url += `&zone_id=${params.zoneId}`
  if (params.shelfId !== undefined) url += `&shelf_id=${params.shelfId}`
  return url
}
