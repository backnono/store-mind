package retrieval

import (
	"context"

	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"
)

// ── 语义重排序类型 ───────────────────────────────────

// SemanticCandidate 表示一个待语义排序的 FAQ 候选条目。
type SemanticCandidate struct {
	ID       int64  `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Category string `json:"category"`
}

// SemanticRankResult 表示单条语义排序结果。
type SemanticRankResult struct {
	ID             int64   `json:"id"`
	RelevanceScore float64 `json:"relevance_score"`
}

// SemanticReranker 语义重排序能力接口。
// 由 infra/ai.PythonLLMClient 实现，通过 SetLLMClient 注入到 MySQLRetriever。
type SemanticReranker interface {
	SemanticRankFAQ(ctx context.Context, query string, candidates []SemanticCandidate) ([]SemanticRankResult, error)
}

// ── MySQLRetriever ───────────────────────────────────

// preFetchMultiplier 两阶段检索时，第一阶段预取的倍数。
// 比如 limit=3 时预取 10 条，保证 LLM 有足够候选做重排序。
const preFetchLimit = 10

type MySQLRetriever struct {
	repo      domain.KnowledgeRepository
	llmClient SemanticReranker // 可选：注入后启用语义重排序
}

func NewMySQLRetriever(repo domain.KnowledgeRepository) *MySQLRetriever {
	return &MySQLRetriever{repo: repo}
}

// SetLLMClient 注入语义重排序客户端。可为 nil（表示禁用语义重排序）。
// 向后兼容：不调用 SetLLMClient 时，Retrieve 仅使用 LIKE 搜索。
func (r *MySQLRetriever) SetLLMClient(client SemanticReranker) {
	r.llmClient = client
}

func (r *MySQLRetriever) Retrieve(ctx context.Context, req app.RetrievalRequest) ([]app.Evidence, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 3
	}

	// 两阶段检索：Phase 1 预取 preFetchLimit 条，Phase 2 LLM 重排序
	if r.llmClient != nil {
		return r.twoPhaseRetrieve(ctx, req, limit)
	}

	// 传统路径：直接 LIKE 搜索
	return r.likeOnlyRetrieve(ctx, req, limit)
}

// twoPhaseRetrieve 两阶段检索：
//
//	Phase 1: LIKE 预取 top-N 候选（快速初筛）
//	Phase 2: 发送给 LLM 语义重排序，保留最终 limit 条
//	若 LLM 调用失败，回退到 LIKE 原始顺序。
func (r *MySQLRetriever) twoPhaseRetrieve(ctx context.Context, req app.RetrievalRequest, finalLimit int) ([]app.Evidence, error) {
	// Phase 1: 预取 top preFetchLimit
	chunks, err := r.repo.SearchKnowledge(ctx, req.StoreID, req.Query, knowledgeTypesForIntent(req.Intent), preFetchLimit)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, nil
	}

	// 构建候选列表
	candidates := make([]SemanticCandidate, len(chunks))
	for i, ch := range chunks {
		candidates[i] = SemanticCandidate{
			ID:       ch.ID,
			Question: ch.Title,
			Answer:   ch.Content,
			Category: ch.KnowledgeType,
		}
	}

	// Phase 2: LLM 语义重排序
	ranked, err := r.llmClient.SemanticRankFAQ(ctx, req.Query, candidates)
	if err != nil {
		// LLM 失败时回退到原始 LIKE 顺序
		return chunksToEvidence(chunks, finalLimit), nil
	}

	// 按 LLM 排序结果重排 chunks
	reranked := reorderChunks(chunks, ranked, finalLimit)
	return chunksToEvidence(reranked, finalLimit), nil
}

// likeOnlyRetrieve 传统路径：仅使用 LIKE 搜索（向后兼容）。
func (r *MySQLRetriever) likeOnlyRetrieve(ctx context.Context, req app.RetrievalRequest, limit int) ([]app.Evidence, error) {
	chunks, err := r.repo.SearchKnowledge(ctx, req.StoreID, req.Query, knowledgeTypesForIntent(req.Intent), limit)
	if err != nil {
		return nil, err
	}
	return chunksToEvidence(chunks, limit), nil
}

// chunksToEvidence 将 KnowledgeChunk 切片转换为 Evidence 切片，最多取 limit 条。
func chunksToEvidence(chunks []domain.KnowledgeChunk, limit int) []app.Evidence {
	n := len(chunks)
	if limit < n {
		n = limit
	}
	out := make([]app.Evidence, 0, n)
	for i := 0; i < n; i++ {
		chunk := chunks[i]
		out = append(out, app.Evidence{
			Source:   "retriever",
			Kind:     chunk.KnowledgeType,
			RecordID: chunk.ID,
			Title:    chunk.Title,
			Content:  chunk.Content,
		})
	}
	return out
}

// reorderChunks 按 LLM 排序结果重新排列 chunks。
// ranked 是 LLM 返回的排序列表（相关性降序），取前 finalLimit 条。
// 如果某个 ID 在 chunks 中不存在，则跳过。
func reorderChunks(chunks []domain.KnowledgeChunk, ranked []SemanticRankResult, finalLimit int) []domain.KnowledgeChunk {
	// 构建 ID -> chunk 映射
	chunkMap := make(map[int64]domain.KnowledgeChunk, len(chunks))
	for _, ch := range chunks {
		chunkMap[ch.ID] = ch
	}

	result := make([]domain.KnowledgeChunk, 0, finalLimit)
	for _, r := range ranked {
		if len(result) >= finalLimit {
			break
		}
		if ch, ok := chunkMap[r.ID]; ok {
			result = append(result, ch)
			delete(chunkMap, r.ID) // 去重
		}
	}

	// 兜底：LLM 未覆盖的 chunk（如果有），追加到末尾
	for _, ch := range chunks {
		if len(result) >= finalLimit {
			break
		}
		if _, exists := chunkMap[ch.ID]; exists {
			result = append(result, ch)
		}
	}

	return result
}

func knowledgeTypesForIntent(intent string) []string {
	switch intent {
	case "faq":
		return []string{"faq", "store_policy"}
	default:
		return nil
	}
}
