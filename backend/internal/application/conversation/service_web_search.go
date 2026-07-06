package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/websearch"
)

// 内置联网工具在模型侧的名称；与前端 message-tool-trace 的 web_search 渲染约定保持一致。
const (
	webSearchToolName = "web_search"
	webFetchToolName  = "web_fetch"
)

// builtinToolHandler 是平台内置工具的执行函数，返回回灌给模型的 JSON 文本。
type builtinToolHandler func(ctx context.Context, argumentsJSON string) (string, error)

var webSearchToolSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"query": {
			"type": "string",
			"description": "Search keywords or a question, in the language most likely to match relevant sources."
		},
		"max_results": {
			"type": "integer",
			"minimum": 1,
			"maximum": 10,
			"description": "Optional number of results to return."
		}
	},
	"required": ["query"]
}`)

var webFetchToolSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"url": {
			"type": "string",
			"description": "The http(s) URL of a public web page to read."
		}
	},
	"required": ["url"]
}`)

// appendWebSearchRuntime 将平台内置联网搜索工具追加到本轮工具运行时。
// 与 MCP 工具共用同一套定义/执行/回灌链路，仅执行阶段改为本地分发。
func (s *Service) appendWebSearchRuntime(runtime *selectedToolRuntime, cfg config.Config) {
	if s == nil || s.webSearch == nil || runtime == nil || !webSearchConfigured(cfg) {
		return
	}
	if runtime.nameMap == nil {
		runtime.nameMap = map[string]string{}
	}
	if runtime.schemas == nil {
		runtime.schemas = map[string]json.RawMessage{}
	}
	if runtime.builtinHandlers == nil {
		runtime.builtinHandlers = map[string]builtinToolHandler{}
	}

	s.appendBuiltinTool(runtime, llm.ToolDefinition{
		Name:        webSearchToolName,
		Description: "Search the public web for up-to-date information. Returns results with title, url and snippet. Use it for current events, recent data, or anything you are unsure about.",
		InputSchema: webSearchToolSchema,
	}, s.webSearchToolHandler(cfg))

	if cfg.WebSearchFetchEnable {
		s.appendBuiltinTool(runtime, llm.ToolDefinition{
			Name:        webFetchToolName,
			Description: "Fetch a public web page by URL and return its main text content. Use it after web_search when a snippet is not enough.",
			InputSchema: webFetchToolSchema,
		}, s.webFetchToolHandler(cfg))
	}
}

func (s *Service) appendBuiltinTool(runtime *selectedToolRuntime, definition llm.ToolDefinition, handler builtinToolHandler) {
	name := definition.Name
	// 与已选择的 MCP 工具重名时让位，内置工具改用带后缀的名称。
	if _, exists := runtime.nameMap[name]; exists {
		name = name + "_builtin"
		if _, stillExists := runtime.nameMap[name]; stillExists {
			return
		}
		definition.Name = name
	}
	runtime.definitions = append(runtime.definitions, definition)
	runtime.nameMap[name] = definition.Name
	runtime.schemas[name] = definition.InputSchema
	runtime.builtinHandlers[name] = handler
}

// webSearchConfigured 判断内置联网搜索是否启用且搜索源配置完整。
func webSearchConfigured(cfg config.Config) bool {
	if !cfg.WebSearchEnable {
		return false
	}
	switch strings.TrimSpace(cfg.WebSearchProvider) {
	case websearch.ProviderTavily:
		return strings.TrimSpace(cfg.WebSearchTavilyAPIKey) != ""
	default:
		return strings.TrimSpace(cfg.WebSearchSearXNGBaseURL) != ""
	}
}

func webSearchProviderConfig(cfg config.Config) websearch.ProviderConfig {
	return websearch.ProviderConfig{
		Provider:       cfg.WebSearchProvider,
		SearXNGBaseURL: cfg.WebSearchSearXNGBaseURL,
		TavilyAPIKey:   cfg.WebSearchTavilyAPIKey,
	}
}

func (s *Service) webSearchToolHandler(cfg config.Config) builtinToolHandler {
	return func(ctx context.Context, argumentsJSON string) (string, error) {
		var args struct {
			Query      string `json:"query"`
			MaxResults int    `json:"max_results"`
		}
		if raw := strings.TrimSpace(argumentsJSON); raw != "" {
			if err := json.Unmarshal([]byte(raw), &args); err != nil {
				return "", fmt.Errorf("web_search arguments must be a JSON object: %w", err)
			}
		}
		if strings.TrimSpace(args.Query) == "" {
			return "", fmt.Errorf("web_search requires a non-empty query")
		}
		maxResults := cfg.WebSearchMaxResults
		if args.MaxResults > 0 && args.MaxResults < maxResults {
			maxResults = args.MaxResults
		}
		output, err := s.webSearch.Search(ctx, webSearchProviderConfig(cfg), websearch.SearchInput{
			Query:      args.Query,
			MaxResults: maxResults,
			TimeoutMS:  cfg.WebSearchTimeoutSeconds * 1000,
		})
		if err != nil {
			return "", err
		}
		encoded, err := json.Marshal(output)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}
}

func (s *Service) webFetchToolHandler(cfg config.Config) builtinToolHandler {
	return func(ctx context.Context, argumentsJSON string) (string, error) {
		var args struct {
			URL string `json:"url"`
		}
		if raw := strings.TrimSpace(argumentsJSON); raw != "" {
			if err := json.Unmarshal([]byte(raw), &args); err != nil {
				return "", fmt.Errorf("web_fetch arguments must be a JSON object: %w", err)
			}
		}
		if strings.TrimSpace(args.URL) == "" {
			return "", fmt.Errorf("web_fetch requires a non-empty url")
		}
		output, err := s.webSearch.Fetch(ctx, websearch.FetchInput{
			URL:       args.URL,
			MaxChars:  cfg.WebSearchFetchMaxChars,
			TimeoutMS: cfg.WebSearchTimeoutSeconds * 1000,
		})
		if err != nil {
			return "", err
		}
		encoded, err := json.Marshal(output)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}
}

// injectWebSearchGuidance 在启用内置联网搜索时注入使用规则与当前日期。
func injectWebSearchGuidance(messages []llm.Message, runtime selectedToolRuntime, customPrompt string) []llm.Message {
	if _, ok := runtime.builtinHandlers[webSearchToolName]; !ok {
		return messages
	}

	content := strings.TrimSpace(customPrompt)
	if content == "" {
		content = defaultWebSearchGuidancePrompt()
	}

	insertAt := 0
	for insertAt < len(messages) && messages[insertAt].Role == "system" {
		insertAt++
	}
	next := make([]llm.Message, 0, len(messages)+1)
	next = append(next, messages[:insertAt]...)
	next = append(next, llm.Message{Role: "system", Content: content})
	next = append(next, messages[insertAt:]...)
	return next
}

func defaultWebSearchGuidancePrompt() string {
	var builder strings.Builder
	builder.WriteString("# web_search\n")
	builder.WriteString(fmt.Sprintf("- Current date: %s.\n", time.Now().Format("2006-01-02")))
	builder.WriteString("- Use the web_search tool for time-sensitive questions, recent events, or facts you are unsure about; do not guess.\n")
	builder.WriteString("- Prefer one to three focused searches; refine the query instead of repeating it.\n")
	builder.WriteString("- Use web_fetch to read a specific result when its snippet is not enough.\n")
	builder.WriteString("- Cite sources in the final answer as markdown links.\n")
	return strings.TrimSpace(builder.String())
}
