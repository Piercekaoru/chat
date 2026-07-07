package llm

import (
	"encoding/json"
	"testing"
)

func chatStreamChunk(t *testing.T, payload string) map[string]interface{} {
	t.Helper()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		t.Fatalf("invalid chunk fixture: %v", err)
	}
	return parsed
}

// 上游按 OpenAI 规范带 index：并行调用分槽累积。
func TestMergeChatStreamToolCallsWithIndex(t *testing.T) {
	result := &GenerateOutput{}
	chunks := []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_a","type":"function","function":{"name":"web_search","arguments":""}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"query\":\"a\"}"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_b","type":"function","function":{"name":"web_search","arguments":"{\"query\":\"b\"}"}}]}}]}`,
	}
	for _, chunk := range chunks {
		mergeChatStreamToolCalls(chatStreamChunk(t, chunk), result)
	}
	if len(result.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d: %+v", len(result.ToolCalls), result.ToolCalls)
	}
	if result.ToolCalls[0].ArgumentsJSON != `{"query":"a"}` || result.ToolCalls[1].ArgumentsJSON != `{"query":"b"}` {
		t.Fatalf("unexpected arguments: %+v", result.ToolCalls)
	}
}

// 部分网关省略 index：靠 id 区分并行调用，不能把参数拼接进同一个调用。
func TestMergeChatStreamToolCallsWithoutIndexUsesID(t *testing.T) {
	result := &GenerateOutput{}
	chunks := []string{
		`{"choices":[{"delta":{"tool_calls":[{"id":"call_a","type":"function","function":{"name":"web_search","arguments":"{\"query\":\"AI新闻\"}"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"id":"call_b","type":"function","function":{"name":"web_search","arguments":"{\"query\":\"polymarket\"}"}}]}}]}`,
	}
	for _, chunk := range chunks {
		mergeChatStreamToolCalls(chatStreamChunk(t, chunk), result)
	}
	if len(result.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d: %+v", len(result.ToolCalls), result.ToolCalls)
	}
	if result.ToolCalls[0].ArgumentsJSON != `{"query":"AI新闻"}` {
		t.Fatalf("first call arguments corrupted: %q", result.ToolCalls[0].ArgumentsJSON)
	}
	if result.ToolCalls[1].ArgumentsJSON != `{"query":"polymarket"}` {
		t.Fatalf("second call arguments corrupted: %q", result.ToolCalls[1].ArgumentsJSON)
	}
}

// 省略 index 且后续分片不带 id：参数续片归并到最近一个调用。
func TestMergeChatStreamToolCallsWithoutIndexContinuation(t *testing.T) {
	result := &GenerateOutput{}
	chunks := []string{
		`{"choices":[{"delta":{"tool_calls":[{"id":"call_a","type":"function","function":{"name":"web_search","arguments":"{\"query\":"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"\"tokyo\"}"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"id":"call_b","type":"function","function":{"name":"web_fetch","arguments":"{\"url\":\"https://x.test\"}"}}]}}]}`,
	}
	for _, chunk := range chunks {
		mergeChatStreamToolCalls(chatStreamChunk(t, chunk), result)
	}
	if len(result.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d: %+v", len(result.ToolCalls), result.ToolCalls)
	}
	if result.ToolCalls[0].ArgumentsJSON != `{"query":"tokyo"}` {
		t.Fatalf("continuation not merged into first call: %q", result.ToolCalls[0].ArgumentsJSON)
	}
	if result.ToolCalls[1].ArgumentsJSON != `{"url":"https://x.test"}` {
		t.Fatalf("second call arguments corrupted: %q", result.ToolCalls[1].ArgumentsJSON)
	}
}
