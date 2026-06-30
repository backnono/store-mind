package customerqa

import (
	"encoding/json"
	"testing"
)

// TestNormalizeToolArgs 验证防御性 args 规范化在各种 LLM 返回格式下都能正确工作。
func TestNormalizeToolArgs(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		args     json.RawMessage
		wantOK   bool   // json.Unmarshal 到目标 struct 能否成功
		wantKey  string // 期望解析出的关键字段值
	}{
		// Case 1: 正常 JSON object（Python json.loads 后）
		{
			name:     "valid json object",
			toolName: "search_products",
			args:     json.RawMessage(`{"query":"薯片","limit":5}`),
			wantOK:   true,
			wantKey:  "薯片",
		},
		// Case 2: JSON string 包裹的 object（Python 未做 json.loads）
		{
			name:     "json object wrapped as string",
			toolName: "search_products",
			args:     json.RawMessage(`"{\"query\":\"薯片\"}"`),
			wantOK:   true,
			wantKey:  "薯片",
		},
		// Case 3: 纯文本 string（LLM 直接填了关键词）
		{
			name:     "plain text string → auto-wrap to query",
			toolName: "search_products",
			args:     json.RawMessage(`"薯片"`),
			wantOK:   true,
			wantKey:  "薯片",
		},
		// Case 4: product_id 纯数字
		{
			name:     "plain string for product_id",
			toolName: "get_product_location",
			args:     json.RawMessage(`"106"`),
			wantOK:   true,
			wantKey:  "106",
		},
		// Case 5: 空 args
		{
			name:     "empty args",
			toolName: "search_products",
			args:     json.RawMessage(``),
			wantOK:   false,
			wantKey:  "",
		},
		// Case 6: 已经是数组（保持原样，由 tool 自行处理）
		{
			name:     "already json array — pass through",
			toolName: "search_products",
			args:     json.RawMessage(`["薯片"]`),
			wantOK:   false, // 数组无法映射到 query struct，但 normalize 不做破坏
			wantKey:  "",
		},
		// Case 7: DeepSeek 典型格式 — JSON string 内嵌完整 JSON object
		{
			name:     "deepseek typical: string-wrapped json object",
			toolName: "search_products",
			args:     json.RawMessage(`"{\"query\":\"可乐\",\"limit\":5}"`),
			wantOK:   true,
			wantKey:  "可乐",
		},
		// Case 8: get_inventory sku_id 数字 string
		{
			name:     "plain string for sku_id",
			toolName: "get_inventory",
			args:     json.RawMessage(`"2001"`),
			wantOK:   true,
			wantKey:  "2001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeToolArgs(tt.toolName, tt.args)

			// 测试能否成功 unmarshal 到目标 struct
			switch tt.toolName {
			case "search_products", "search_faq":
				var input struct {
					Query string `json:"query"`
				}
				err := json.Unmarshal(normalized, &input)
				if tt.wantOK && err != nil {
					t.Errorf("unmarshal failed: %v\nnormalized=%s", err, normalized)
				}
				if !tt.wantOK && err == nil {
					t.Logf("expected unmarshal to fail, but succeeded: input=%+v", input)
				}
				if tt.wantKey != "" && input.Query != tt.wantKey {
					t.Errorf("expected query=%q, got %q", tt.wantKey, input.Query)
				}
			case "get_product_location":
				var input struct {
					ProductID json.RawMessage `json:"product_id"`
				}
				err := json.Unmarshal(normalized, &input)
				if tt.wantOK && err != nil {
					t.Errorf("unmarshal failed: %v", err)
				}
				// product_id can be number or string
				if tt.wantKey != "" && len(input.ProductID) == 0 {
					t.Errorf("expected non-empty product_id")
				}
			}
			t.Logf("input: %s → normalized: %s", tt.args, normalized)
		})
	}
}

// TestNormalizeToolArgsRealError 复现 session 59 的错误：
// LLM 返回 {"query": "薯片"} 被序列化为 JSON string 再传给 tool。
func TestNormalizeToolArgsRealError(t *testing.T) {
	// 模拟 DeepSeek 返回的原始 args（Python 未做 json.loads 时）
	// Go 侧 json.Unmarshal 到 LLMToolCall.Args (json.RawMessage) 的结果：
	// 是一个 Go string: `"{\"query\":\"薯片\"}"` 对应的 json.RawMessage
	rawArgs := json.RawMessage(`"{\"query\":\"薯片\"}"`)

	// 旧代码：直接传给 tool.Run → 报错
	var oldInput struct {
		Query string `json:"query"`
	}
	oldErr := json.Unmarshal(rawArgs, &oldInput)
	if oldErr == nil {
		t.Error("old code: expected unmarshal error for string args, but got none")
	} else {
		t.Logf("old code error (expected): %v", oldErr)
	}

	// 新代码：normalizeToolArgs → 再传给 tool.Run → 成功
	normalized := normalizeToolArgs("search_products", rawArgs)
	var newInput struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(normalized, &newInput); err != nil {
		t.Fatalf("new code: unmarshal failed after normalization: %v\nnormalized=%s", err, normalized)
	}
	if newInput.Query != "薯片" {
		t.Errorf("expected query='薯片', got %q", newInput.Query)
	}
	t.Logf("new code: success! query=%q from normalized=%s", newInput.Query, normalized)
}
