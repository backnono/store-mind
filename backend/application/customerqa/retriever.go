package customerqa

import "context"

// RetrievalRequest 检索请求，描述需要从知识库中召回的证据条件。
type RetrievalRequest struct {
	StoreID int64  // 门店 ID，限定知识范围
	Query   string // 检索查询文本（可能是改写后的）
	Intent  string // 当前意图，用于过滤/排序召回结果
	Limit   int    // 最大返回条数
}

// Retriever 证据检索器，从门店知识库中召回与查询相关的证据片段。
// 证据片段通常是 FAQ 条目、商品描述等结构化文本，后续由 AnswerComposer 组装为自然语言回答。
type Retriever interface {
	Retrieve(ctx context.Context, req RetrievalRequest) ([]Evidence, error)
}
