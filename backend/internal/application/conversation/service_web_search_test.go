package conversation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/websearch"
)

func webSearchTestConfig(baseURL string) config.Config {
	return config.Config{
		WebSearchEnable:         true,
		WebSearchProvider:       "searxng",
		WebSearchSearXNGBaseURL: baseURL,
		WebSearchMaxResults:     5,
		WebSearchTimeoutSeconds: 5,
		WebSearchFetchEnable:    true,
		WebSearchFetchMaxChars:  8000,
	}
}

func TestAppendWebSearchRuntimeAddsBuiltinTools(t *testing.T) {
	svc := &Service{webSearch: websearch.NewClientWithEnv("dev", false)}
	runtime := selectedToolRuntime{}

	svc.appendWebSearchRuntime(&runtime, webSearchTestConfig("http://searxng.example.com"))

	if len(runtime.definitions) != 2 {
		t.Fatalf("expected web_search and web_fetch definitions, got %d", len(runtime.definitions))
	}
	if runtime.definitions[0].Name != webSearchToolName || runtime.definitions[1].Name != webFetchToolName {
		t.Fatalf("unexpected tool names: %s, %s", runtime.definitions[0].Name, runtime.definitions[1].Name)
	}
	if runtime.builtinHandlers[webSearchToolName] == nil || runtime.builtinHandlers[webFetchToolName] == nil {
		t.Fatalf("expected builtin handlers to be registered")
	}
	if runtime.nameMap[webSearchToolName] != webSearchToolName {
		t.Fatalf("expected identity name mapping, got %q", runtime.nameMap[webSearchToolName])
	}
	if len(runtime.schemas[webSearchToolName]) == 0 {
		t.Fatalf("expected web_search schema to be registered")
	}
}

func TestAppendWebSearchRuntimeSkipsWhenDisabledOrUnconfigured(t *testing.T) {
	svc := &Service{webSearch: websearch.NewClientWithEnv("dev", false)}

	disabled := webSearchTestConfig("http://searxng.example.com")
	disabled.WebSearchEnable = false
	runtime := selectedToolRuntime{}
	svc.appendWebSearchRuntime(&runtime, disabled)
	if len(runtime.definitions) != 0 {
		t.Fatalf("expected no tools when disabled")
	}

	unconfigured := webSearchTestConfig("")
	runtime = selectedToolRuntime{}
	svc.appendWebSearchRuntime(&runtime, unconfigured)
	if len(runtime.definitions) != 0 {
		t.Fatalf("expected no tools when searxng base url missing")
	}

	tavily := webSearchTestConfig("")
	tavily.WebSearchProvider = "tavily"
	tavily.WebSearchTavilyAPIKey = "tvly-key"
	runtime = selectedToolRuntime{}
	svc.appendWebSearchRuntime(&runtime, tavily)
	if len(runtime.definitions) != 2 {
		t.Fatalf("expected tavily config to enable tools")
	}
}

func TestAppendWebSearchRuntimeRespectsFetchDisable(t *testing.T) {
	svc := &Service{webSearch: websearch.NewClientWithEnv("dev", false)}
	cfg := webSearchTestConfig("http://searxng.example.com")
	cfg.WebSearchFetchEnable = false
	runtime := selectedToolRuntime{}

	svc.appendWebSearchRuntime(&runtime, cfg)

	if len(runtime.definitions) != 1 || runtime.definitions[0].Name != webSearchToolName {
		t.Fatalf("expected only web_search, got %#v", runtime.definitions)
	}
}

func TestAppendWebSearchRuntimeRenamesOnMCPNameCollision(t *testing.T) {
	svc := &Service{webSearch: websearch.NewClientWithEnv("dev", false)}
	runtime := selectedToolRuntime{
		nameMap: map[string]string{webSearchToolName: "mcp_web_search"},
		schemas: map[string]json.RawMessage{},
	}

	svc.appendWebSearchRuntime(&runtime, webSearchTestConfig("http://searxng.example.com"))

	renamed := webSearchToolName + "_builtin"
	if runtime.builtinHandlers[renamed] == nil {
		t.Fatalf("expected builtin web_search to be renamed to %s", renamed)
	}
	if runtime.nameMap[webSearchToolName] != "mcp_web_search" {
		t.Fatalf("expected mcp mapping to stay intact")
	}
}

func TestExecuteAssistantToolCallsDispatchesBuiltinHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]string{
				{"title": "Result", "url": "https://example.com", "content": "snippet"},
			},
		})
	}))
	defer server.Close()

	svc := &Service{
		cfg:       config.NewRuntime(config.Config{}),
		webSearch: websearch.NewClientWithEnv("dev", false),
	}
	runtime := selectedToolRuntime{}
	svc.appendWebSearchRuntime(&runtime, webSearchTestConfig(server.URL))

	result := svc.executeAssistantToolCalls(context.Background(), executeAssistantToolCallsInput{
		RunID: "run_ws",
		ToolCalls: []llm.ToolCall{{
			ToolCallID:    "toolu_ws",
			ToolType:      "function",
			ToolName:      webSearchToolName,
			ArgumentsJSON: `{"query":"deeix chat"}`,
			Status:        "requested",
		}},
		ToolNameMap:     runtime.nameMap,
		MCPConfigs:      runtime.mcpConfigs,
		ToolSchemas:     runtime.schemas,
		BuiltinHandlers: runtime.builtinHandlers,
		Ledger:          newToolExecutionLedger(),
	})

	if result.FatalErr != nil {
		t.Fatalf("unexpected fatal error: %v", result.FatalErr)
	}
	if len(result.Rows) != 1 || result.Rows[0].Status != "success" {
		t.Fatalf("expected success row, got %#v", result.Rows)
	}
	if !strings.Contains(result.Rows[0].OutputJSON, "https://example.com") {
		t.Fatalf("expected search result url in output, got %s", result.Rows[0].OutputJSON)
	}
	if len(result.ToolResults) != 1 || result.ToolResults[0].Status != "success" {
		t.Fatalf("expected success tool result, got %#v", result.ToolResults)
	}
}

func TestExecuteAssistantToolCallsBuiltinErrorIsNotFatal(t *testing.T) {
	svc := &Service{
		cfg:       config.NewRuntime(config.Config{}),
		webSearch: websearch.NewClientWithEnv("dev", false),
	}
	runtime := selectedToolRuntime{}
	// 指向不可达地址，触发执行错误但不是 fatal（模型可自行调整或回答）。
	cfg := webSearchTestConfig("http://127.0.0.1:1")
	svc.appendWebSearchRuntime(&runtime, cfg)

	result := svc.executeAssistantToolCalls(context.Background(), executeAssistantToolCallsInput{
		RunID: "run_ws_err",
		ToolCalls: []llm.ToolCall{{
			ToolCallID:    "toolu_ws_err",
			ToolType:      "function",
			ToolName:      webSearchToolName,
			ArgumentsJSON: `{"query":"deeix chat"}`,
			Status:        "requested",
		}},
		ToolNameMap:     runtime.nameMap,
		MCPConfigs:      runtime.mcpConfigs,
		ToolSchemas:     runtime.schemas,
		BuiltinHandlers: runtime.builtinHandlers,
		Ledger:          newToolExecutionLedger(),
	})

	if result.FatalErr != nil {
		t.Fatalf("builtin execution error must not be fatal, got %v", result.FatalErr)
	}
	if len(result.Rows) != 1 || result.Rows[0].Status != "error" {
		t.Fatalf("expected error row, got %#v", result.Rows)
	}
}

func TestInjectWebSearchGuidance(t *testing.T) {
	runtime := selectedToolRuntime{builtinHandlers: map[string]builtinToolHandler{
		webSearchToolName: func(ctx context.Context, argumentsJSON string) (string, error) { return "{}", nil },
	}}
	messages := []llm.Message{
		{Role: "system", Content: "base prompt"},
		{Role: "user", Content: "hi"},
	}

	next := injectWebSearchGuidance(messages, runtime, "")
	if len(next) != 3 {
		t.Fatalf("expected guidance message injected, got %d messages", len(next))
	}
	if next[1].Role != "system" || !strings.Contains(next[1].Content, "web_search") {
		t.Fatalf("expected web search guidance after system prefix, got %#v", next[1])
	}
	if !strings.Contains(next[1].Content, "Current date:") {
		t.Fatalf("expected current date in guidance, got %s", next[1].Content)
	}

	custom := injectWebSearchGuidance(messages, runtime, "custom guidance")
	if custom[1].Content != "custom guidance" {
		t.Fatalf("expected custom guidance override, got %s", custom[1].Content)
	}

	unchanged := injectWebSearchGuidance(messages, selectedToolRuntime{}, "")
	if len(unchanged) != len(messages) {
		t.Fatalf("expected no injection without builtin web search")
	}
}
