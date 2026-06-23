// ============================================================
// Store Mind 顾客端 — 共享类型定义
// 与后端 /api/v1/customer-qa/ 接口对齐
// ============================================================

// ── 聊天 ──────────────────────────────────────────────

/** 聊天请求 */
export interface ChatRequest {
  store_id: number
  session_id?: number
  user_id?: number
  channel: 'miniapp'
  message: string
  entry_mode?: 'first_open' | 'zone_scan' | 'resume' | 'promo' | 'product_detail'
  zone_id?: number
  shelf_id?: number
}

/** 卡片类型枚举 */
export type CardType = 'product' | 'inventory' | 'promotion' | 'faq'

/** 结构化卡片 */
export interface ChatCard {
  type: CardType
  sku_id?: number
  name?: string
  location?: string
  quantity?: number
  title?: string
  content?: string
  validity?: string
  price?: string
  image_url?: string
}

/** 引导建议芯片 */
export interface GuidanceChip {
  text: string
  prompt: string
}

/** 聊天响应元数据 */
export interface ChatResponseMeta {
  request_id?: string
  route?: string
  confidence?: number
  rewrite_query?: string
  fallback_used?: boolean
  evidence_count?: number
}

/** 聊天响应 */
export interface ChatResponse {
  session_id: number
  message_id: number
  intent: string
  answer: string
  cards: ChatCard[]
  guidance_chips: GuidanceChip[]
  handoff_required: boolean
  meta?: ChatResponseMeta
}

/** 前端归一化后的 API 错误 */
export interface ApiError {
  code: 'network' | 'timeout' | 'bad_request' | 'server_error' | 'unknown'
  message: string
  retryable: boolean
}

// ── 反馈 ──────────────────────────────────────────────

/** 反馈请求 */
export interface FeedbackRequest {
  message_id: number
  session_id: number
  feedback_value: 0 | 1 // 0=👎 / 1=👍
}

// ── 活动 ──────────────────────────────────────────────

/** 活动/促销 */
export interface Promotion {
  id: number
  store_id: number
  title: string
  description: string
  product_scope?: string[]
  start_at: string
  end_at: string
  status: string
}

// ── 消息 (前端视图模型) ────────────────────────────────

/** 消息来源 */
export type MessageRole = 'user' | 'assistant'

/** 发送状态 */
export type SendStatus = 'sending' | 'sent' | 'failed'

/** 引导启动状态 */
export type BootstrapStatus = 'idle' | 'loading' | 'ready' | 'error'

/** 单条消息 — 前端视图模型 */
export interface Message {
  /** 客户端生成的唯一 id */
  id: string
  /** 后端返回的 message_id，用户消息在发送前为 undefined */
  messageId?: number
  role: MessageRole
  text: string
  /** 助理消息可能带卡片 */
  cards?: ChatCard[]
  /** 助理消息可能带引导建议 */
  guidanceChips?: GuidanceChip[]
  /** 时间戳 */
  createdAt: number
  /** 用户消息的发送状态 */
  sendStatus?: SendStatus
  /** 是否人工转接需求 */
  handoffRequired?: boolean
}

/** 入口上下文 */
export interface EntryContext {
  mode: 'first_open' | 'zone_scan' | 'resume' | 'promo' | 'product_detail'
  storeId: number
  zoneId?: number
  shelfId?: number
  prompt?: string
  promoTitle?: string
}

// ── 活动列表响应 ───────────────────────────────────────

export interface PromotionListResponse {
  items: Promotion[]
  meta?: { request_id?: string }
}
