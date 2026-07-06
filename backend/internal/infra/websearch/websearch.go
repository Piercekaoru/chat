// Package websearch 提供平台自执行的联网搜索与网页抓取能力。
// 搜索源可插拔（SearXNG / Tavily），由管理员在系统设置中配置；
// 所有出站请求复用平台统一的 SSRF 防护出站通道，避免模型诱导访问内网地址。
package websearch

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	platformtracing "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/observability/tracing"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
)

// 搜索源类型。
const (
	ProviderSearXNG = "searxng"
	ProviderTavily  = "tavily"
)

const (
	defaultMaxResults    = 5
	maxAllowedResults    = 20
	defaultTimeout       = 15 * time.Second
	maxResponseBodyBytes = 4 << 20 // 搜索响应与网页抓取的原始字节上限
	fetchUserAgent       = "Mozilla/5.0 (compatible; DEEIX-Chat-WebSearch/1.0)"
)

// ProviderConfig 定义一次调用使用的搜索源配置（来自系统设置快照）。
type ProviderConfig struct {
	Provider       string // searxng | tavily
	SearXNGBaseURL string
	TavilyAPIKey   string
	TavilyBaseURL  string // 空串使用官方地址；供测试或代理覆盖
}

// SearchInput 定义一次联网搜索请求。
type SearchInput struct {
	Query      string
	MaxResults int
	TimeoutMS  int
}

// SearchResult 是单条搜索结果。
type SearchResult struct {
	Title         string `json:"title"`
	URL           string `json:"url"`
	Snippet       string `json:"snippet,omitempty"`
	PublishedDate string `json:"publishedDate,omitempty"`
}

// SearchOutput 是联网搜索结果集，序列化后回灌给模型。
type SearchOutput struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

// Client 执行联网搜索与网页抓取。
type Client struct {
	httpClient            *http.Client
	env                   string
	ssrfProtectionEnabled bool
}

// NewClientWithEnv 创建带运行环境的联网搜索客户端。
func NewClientWithEnv(env string, ssrfProtectionEnabled bool) *Client {
	transport := security.NewOutboundHTTPTransport(env, ssrfProtectionEnabled, 10*time.Second)
	return &Client{
		httpClient: &http.Client{
			Transport: platformtracing.NewHTTPTransport(transport),
		},
		env:                   env,
		ssrfProtectionEnabled: ssrfProtectionEnabled,
	}
}

// Search 按配置的搜索源执行一次联网搜索。
func (c *Client) Search(ctx context.Context, cfg ProviderConfig, input SearchInput) (SearchOutput, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return SearchOutput{}, fmt.Errorf("search query is required")
	}
	input.Query = query
	input.MaxResults = normalizeMaxResults(input.MaxResults)

	requestCtx, cancel := context.WithTimeout(ctx, resolveTimeout(input.TimeoutMS))
	defer cancel()

	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case ProviderTavily:
		return c.searchTavily(requestCtx, cfg, input)
	case ProviderSearXNG, "":
		return c.searchSearXNG(requestCtx, cfg, input)
	default:
		return SearchOutput{}, fmt.Errorf("unsupported web search provider: %s", cfg.Provider)
	}
}

func normalizeMaxResults(value int) int {
	if value <= 0 {
		return defaultMaxResults
	}
	if value > maxAllowedResults {
		return maxAllowedResults
	}
	return value
}

func resolveTimeout(ms int) time.Duration {
	if ms <= 0 {
		return defaultTimeout
	}
	return time.Duration(ms) * time.Millisecond
}

func trimResults(results []SearchResult, max int) []SearchResult {
	cleaned := make([]SearchResult, 0, len(results))
	for _, item := range results {
		item.Title = strings.TrimSpace(item.Title)
		item.URL = strings.TrimSpace(item.URL)
		item.Snippet = strings.TrimSpace(item.Snippet)
		item.PublishedDate = strings.TrimSpace(item.PublishedDate)
		if item.URL == "" {
			continue
		}
		cleaned = append(cleaned, item)
		if len(cleaned) >= max {
			break
		}
	}
	return cleaned
}
