package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
)

// searchSearXNG 调用自建 SearXNG 实例的 JSON 接口。
// 实例需要在 settings.yml 的 search.formats 中开启 json 输出。
func (c *Client) searchSearXNG(ctx context.Context, cfg ProviderConfig, input SearchInput) (SearchOutput, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.SearXNGBaseURL), "/")
	if baseURL == "" {
		return SearchOutput{}, fmt.Errorf("searxng base url is not configured")
	}
	if err := security.ValidateOutboundHTTPURL(baseURL, c.env, c.ssrfProtectionEnabled); err != nil {
		return SearchOutput{}, fmt.Errorf("searxng base url rejected: %w", err)
	}

	params := url.Values{}
	params.Set("q", input.Query)
	params.Set("format", "json")
	params.Set("safesearch", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/search?"+params.Encode(), nil)
	if err != nil {
		return SearchOutput{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fetchUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SearchOutput{}, fmt.Errorf("searxng request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return SearchOutput{}, fmt.Errorf("searxng response read failed: %w", err)
	}
	if resp.StatusCode == http.StatusForbidden {
		return SearchOutput{}, fmt.Errorf("searxng returned 403: enable the json format in settings.yml (search.formats: [html, json])")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SearchOutput{}, fmt.Errorf("searxng returned status %d", resp.StatusCode)
	}

	var payload struct {
		Results []struct {
			Title         string `json:"title"`
			URL           string `json:"url"`
			Content       string `json:"content"`
			PublishedDate string `json:"publishedDate"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return SearchOutput{}, fmt.Errorf("searxng response decode failed: %w", err)
	}

	results := make([]SearchResult, 0, len(payload.Results))
	for _, item := range payload.Results {
		results = append(results, SearchResult{
			Title:         item.Title,
			URL:           item.URL,
			Snippet:       item.Content,
			PublishedDate: item.PublishedDate,
		})
	}
	return SearchOutput{Query: input.Query, Results: trimResults(results, input.MaxResults)}, nil
}
