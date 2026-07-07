package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const tavilyDefaultBaseURL = "https://api.tavily.com"

// searchTavily 调用 Tavily Search API。
func (c *Client) searchTavily(ctx context.Context, cfg ProviderConfig, input SearchInput) (SearchOutput, error) {
	apiKey := strings.TrimSpace(cfg.TavilyAPIKey)
	if apiKey == "" {
		return SearchOutput{}, fmt.Errorf("tavily api key is not configured")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.TavilyBaseURL), "/")
	if baseURL == "" {
		baseURL = tavilyDefaultBaseURL
	}

	payload := map[string]interface{}{
		"query":        input.Query,
		"max_results":  input.MaxResults,
		"search_depth": "basic",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return SearchOutput{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/search", bytes.NewReader(raw))
	if err != nil {
		return SearchOutput{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.trustedHTTPClient.Do(req)
	if err != nil {
		return SearchOutput{}, fmt.Errorf("tavily request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return SearchOutput{}, fmt.Errorf("tavily response read failed: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return SearchOutput{}, fmt.Errorf("tavily rejected the api key (status %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SearchOutput{}, fmt.Errorf("tavily returned status %d", resp.StatusCode)
	}

	var parsed struct {
		Results []struct {
			Title         string `json:"title"`
			URL           string `json:"url"`
			Content       string `json:"content"`
			PublishedDate string `json:"published_date"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return SearchOutput{}, fmt.Errorf("tavily response decode failed: %w", err)
	}

	results := make([]SearchResult, 0, len(parsed.Results))
	for _, item := range parsed.Results {
		results = append(results, SearchResult{
			Title:         item.Title,
			URL:           item.URL,
			Snippet:       item.Content,
			PublishedDate: item.PublishedDate,
		})
	}
	return SearchOutput{Query: input.Query, Results: trimResults(results, input.MaxResults)}, nil
}
