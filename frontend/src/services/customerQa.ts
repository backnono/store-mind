// ============================================================
// Customer QA API — 后端 /api/v1/customer-qa/ 封装
// ============================================================
import { post, get } from './api'
import type {
  ChatRequest,
  ChatResponse,
  FeedbackRequest,
  Promotion,
  PromotionListResponse,
} from '@/types/customerQa'

const BASE = '/api/v1/customer-qa'

/** 发起聊天 */
export async function chat(request: ChatRequest): Promise<ChatResponse> {
  return post<ChatResponse>(`${BASE}/chat`, request as unknown as Record<string, unknown>)
}

/** 提交反馈 */
export async function submitFeedback(request: FeedbackRequest): Promise<void> {
  await post(`${BASE}/feedback`, request as unknown as Record<string, unknown>)
}

/** 获取门店当前活动 */
export async function listActivePromotions(storeId: number): Promise<Promotion[]> {
  const res = await get<PromotionListResponse>(`${BASE}/promotions/active`, { store_id: storeId })
  return res.items ?? []
}
