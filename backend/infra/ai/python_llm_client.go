package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"
	"store-mind/infra/retrieval"
)

const intentRequestTimeout = 8 * time.Second

// PythonLLMClient 通过 HTTP 调用 Python LLM Sidecar
// 实现 IntentAnalyzer 和 AnswerComposer 接口
type PythonLLMClient struct {
	Endpoint   string
	HTTPClient *http.Client
}

// NewPythonLLMClient 创建 Python LLM 客户端
func NewPythonLLMClient(endpoint string) *PythonLLMClient {
	endpoint = strings.TrimRight(endpoint, "/")
	return &PythonLLMClient{
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second, // 单个请求超时，3s 在 context 中控制
		},
	}
}

// ── IntentAnalyzer 实现 ─────────────────────────────

// AnalyzeIntent 调用 Python /llm/intent 进行意图分析
// 超时 8s，失败时返回 fallback_used=true 的降级决策
func (c *PythonLLMClient) AnalyzeIntent(ctx context.Context, req app.IntentRequest) (app.Decision, error) {
	ctx, cancel := context.WithTimeout(ctx, intentRequestTimeout)
	defer cancel()

	body := map[string]any{
		"message": req.Message,
	}
	// 当 SessionID > 0 时，可以从 repo 查询 context_stack 传入
	// S0 阶段：先传空上下文，让 Python 侧仅基于单条消息做意图分析

	resp, err := c.postJSON(ctx, "/llm/intent", body)
	if err != nil {
		// 降级：返回 fallback 标记的决策，由 orchestrator 走 fallbackOrchestrator
		return app.Decision{
			Intent:       "unsupported",
			Route:        app.RouteFallback,
			Confidence:   0.0,
			FallbackUsed: true,
		}, fmt.Errorf("python llm /llm/intent failed: %w", err)
	}

	// 标准化 route 字段：Python 侧的 route 可能以大写或不同格式返回
	route := resp.Route
	if route == "" {
		route = app.RouteFallback
	}

	return app.Decision{
		Intent:         resp.Intent,
		RewrittenQuery: resp.RewrittenQuery,
		Route:          route,
		NeedsHandoff:   resp.NeedsHandoff,
		Confidence:     resp.Confidence,
		ReasoningTags:  resp.ReasoningTags,
		FallbackUsed:   resp.FallbackUsed,
	}, nil
}

// ── AnswerComposer 实现 ──────────────────────────────

// ComposeAnswer 调用 Python /llm/answer 生成回答 + 引导建议
func (c *PythonLLMClient) ComposeAnswer(ctx context.Context, req app.AnswerRequest) (*app.AnswerResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	evidenceList := make([]map[string]any, 0, len(req.Evidence))
	for _, ev := range req.Evidence {
		evidenceList = append(evidenceList, map[string]any{
			"source":    ev.Source,
			"kind":      ev.Kind,
			"record_id": ev.RecordID,
			"title":     ev.Title,
			"content":   ev.Content,
		})
	}

	body := map[string]any{
		"decision": map[string]any{
			"intent":          req.Decision.Intent,
			"route":           req.Decision.Route,
			"confidence":      req.Decision.Confidence,
			"needs_handoff":   req.Decision.NeedsHandoff,
			"fallback_used":   req.Decision.FallbackUsed,
			"rewritten_query": req.Decision.RewrittenQuery,
		},
		"message":  req.Message,
		"evidence": evidenceList,
	}

	raw, err := c.postJSONRaw(ctx, "/llm/answer", body)
	if err != nil {
		return nil, fmt.Errorf("python llm /llm/answer failed: %w", err)
	}

	// 解析 Python 响应: {answer, guidance_chips: [{text, prompt}]}
	type answerResp struct {
		Answer        string             `json:"answer"`
		GuidanceChips []app.GuidanceChip `json:"guidance_chips"`
	}
	var result answerResp
	if err := json.Unmarshal(raw, &result); err != nil {
		// 解析失败时返回原始文本
		return &app.AnswerResult{
			Answer:        string(raw),
			GuidanceChips: nil,
		}, nil
	}
	return &app.AnswerResult{
		Answer:        result.Answer,
		GuidanceChips: result.GuidanceChips,
	}, nil
}

// ── 指代消解辅助方法 ─────────────────────────────────

// ResolveAnaphora 调用 Python /llm/resolve 进行指代消解
func (c *PythonLLMClient) ResolveAnaphora(ctx context.Context, message string, contextStack []domain.ContextStackItem, focusEntities *domain.FocusEntityIDs) (*app.AnaphoraLLMResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	body := map[string]any{
		"message":        message,
		"context_stack":  contextStack,
		"focus_entities": focusEntities,
	}

	raw, err := c.postJSONRaw(ctx, "/llm/resolve", body)
	if err != nil {
		return nil, fmt.Errorf("python llm /llm/resolve failed: %w", err)
	}

	// 解析 resolved_entities JSON
	type resolveResp struct {
		ResolvedEntities []domain.ResolvedEntity `json:"resolved_entities"`
		Confidence       float64                 `json:"confidence"`
		Explanation      string                  `json:"explanation"`
	}
	var result resolveResp
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse /llm/resolve response: %w", err)
	}
	return &app.AnaphoraLLMResult{
		ResolvedEntities: result.ResolvedEntities,
		Confidence:       result.Confidence,
		Explanation:      result.Explanation,
	}, nil
}

// ── 语义重排序 ─────────────────────────────────────

// SemanticRankFAQ 调用 Python /llm/semantic_search 对 FAQ 候选进行语义重排序。
// 实现 retrieval.SemanticReranker 接口。
// 超时 4s，失败时返回错误，由调用方降级回退。
func (c *PythonLLMClient) SemanticRankFAQ(ctx context.Context, query string, candidates []retrieval.SemanticCandidate) ([]retrieval.SemanticRankResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	body := map[string]any{
		"query":      query,
		"candidates": candidates,
	}

	raw, err := c.postJSONRaw(ctx, "/llm/semantic_search", body)
	if err != nil {
		return nil, fmt.Errorf("python llm /llm/semantic_search failed: %w", err)
	}

	type rankResp struct {
		Ranked []retrieval.SemanticRankResult `json:"ranked"`
	}
	var result rankResp
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse /llm/semantic_search response: %w", err)
	}
	return result.Ranked, nil
}

// ── 内部 HTTP 辅助方法 ───────────────────────────────

type llmIntentResponse struct {
	Intent           string   `json:"intent"`
	RewrittenQuery   string   `json:"rewritten_query"`
	Route            string   `json:"route"`
	NeedsHandoff     bool     `json:"needs_handoff"`
	Confidence       float64  `json:"confidence"`
	ReasoningTags    []string `json:"reasoning_tags"`
	FallbackUsed     bool     `json:"fallback_used"`
	Extra            map[string]any
	ExtraConfidence  float64
	ExtraExplanation string
}

func (c *PythonLLMClient) postJSON(ctx context.Context, path string, body map[string]any) (*llmIntentResponse, error) {
	raw, err := c.postJSONRaw(ctx, path, body)
	if err != nil {
		return nil, err
	}

	var result llmIntentResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// 捕获额外字段
	var extra map[string]any
	json.Unmarshal(raw, &extra)
	// 移除已知字段
	delete(extra, "intent")
	delete(extra, "rewritten_query")
	delete(extra, "route")
	delete(extra, "needs_handoff")
	delete(extra, "confidence")
	delete(extra, "reasoning_tags")
	delete(extra, "fallback_used")
	delete(extra, "resolved_entities")
	delete(extra, "explanation")

	result.Extra = extra

	// 尝试提取某些 endpoint 特有的字段
	var rawMap map[string]any
	json.Unmarshal(raw, &rawMap)
	if v, ok := rawMap["confidence"].(float64); ok {
		result.ExtraConfidence = v
	}
	if v, ok := rawMap["explanation"].(string); ok {
		result.ExtraExplanation = v
	}

	return &result, nil
}

func (c *PythonLLMClient) postJSONRaw(ctx context.Context, path string, body map[string]any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	url := c.Endpoint + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ── 指代消解结果 ─────────────────────────────────────

type ResolveResult struct {
	ResolvedEntities any     `json:"resolved_entities"`
	Confidence       float64 `json:"confidence"`
	Explanation      string  `json:"explanation"`
}
