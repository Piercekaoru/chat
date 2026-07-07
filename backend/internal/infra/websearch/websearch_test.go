package websearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient() *Client {
	return NewClientWithEnv("dev", false)
}

func TestSearchSearXNGParsesResults(t *testing.T) {
	var gotPath, gotQuery, gotFormat string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("q")
		gotFormat = r.URL.Query().Get("format")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]string{
				{"title": "Result A", "url": "https://a.example.com", "content": "snippet a", "publishedDate": "2026-01-02"},
				{"title": "Result B", "url": "https://b.example.com", "content": "snippet b"},
				{"title": "No URL", "url": "", "content": "dropped"},
				{"title": "Result C", "url": "https://c.example.com", "content": "snippet c"},
			},
		})
	}))
	defer server.Close()

	output, err := newTestClient().Search(context.Background(), ProviderConfig{
		Provider:       ProviderSearXNG,
		SearXNGBaseURL: server.URL,
	}, SearchInput{Query: "hello world", MaxResults: 2})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if gotPath != "/search" || gotQuery != "hello world" || gotFormat != "json" {
		t.Fatalf("unexpected request: path=%s q=%s format=%s", gotPath, gotQuery, gotFormat)
	}
	if len(output.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(output.Results))
	}
	if output.Results[0].Title != "Result A" || output.Results[0].URL != "https://a.example.com" || output.Results[0].Snippet != "snippet a" {
		t.Fatalf("unexpected first result: %+v", output.Results[0])
	}
	if output.Results[0].PublishedDate != "2026-01-02" {
		t.Fatalf("expected published date, got %q", output.Results[0].PublishedDate)
	}
}

func TestSearchSearXNGForbiddenHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := newTestClient().Search(context.Background(), ProviderConfig{
		Provider:       ProviderSearXNG,
		SearXNGBaseURL: server.URL,
	}, SearchInput{Query: "hello"})
	if err == nil || !strings.Contains(err.Error(), "json format") {
		t.Fatalf("expected 403 hint about json format, got: %v", err)
	}
}

func TestSearchSearXNGRequiresBaseURL(t *testing.T) {
	_, err := newTestClient().Search(context.Background(), ProviderConfig{Provider: ProviderSearXNG}, SearchInput{Query: "hello"})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected missing base url error, got: %v", err)
	}
}

func TestSearchTavilyRequestAndParse(t *testing.T) {
	var gotAuth string
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]string{
				{"title": "Tavily A", "url": "https://ta.example.com", "content": "content a", "published_date": "2026-03-04"},
			},
		})
	}))
	defer server.Close()

	output, err := newTestClient().Search(context.Background(), ProviderConfig{
		Provider:      ProviderTavily,
		TavilyAPIKey:  "tvly-test",
		TavilyBaseURL: server.URL,
	}, SearchInput{Query: "hello", MaxResults: 3})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if gotAuth != "Bearer tvly-test" {
		t.Fatalf("expected bearer auth, got %q", gotAuth)
	}
	if gotBody["query"] != "hello" || gotBody["max_results"] != float64(3) {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
	if len(output.Results) != 1 || output.Results[0].URL != "https://ta.example.com" || output.Results[0].PublishedDate != "2026-03-04" {
		t.Fatalf("unexpected results: %+v", output.Results)
	}
}

func TestSearchTavilyRequiresAPIKey(t *testing.T) {
	_, err := newTestClient().Search(context.Background(), ProviderConfig{Provider: ProviderTavily}, SearchInput{Query: "hello"})
	if err == nil || !strings.Contains(err.Error(), "api key") {
		t.Fatalf("expected missing api key error, got: %v", err)
	}
}

func TestSearchRejectsEmptyQuery(t *testing.T) {
	_, err := newTestClient().Search(context.Background(), ProviderConfig{Provider: ProviderSearXNG, SearXNGBaseURL: "http://127.0.0.1:9"}, SearchInput{Query: "   "})
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("expected empty query error, got: %v", err)
	}
}

func TestFetchExtractsHTMLMainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html>
<head><title>示例页面</title><script>var x = "ignored";</script></head>
<body>
<nav>Ignored nav</nav>
<main>
<h1>标题一</h1>
<p>第一段正文内容。</p>
<p>第二段正文内容。</p>
</main>
<footer>Ignored footer</footer>
</body>
</html>`))
	}))
	defer server.Close()

	output, err := newTestClient().Fetch(context.Background(), FetchInput{URL: server.URL})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if output.Title != "示例页面" {
		t.Fatalf("expected title, got %q", output.Title)
	}
	if !strings.Contains(output.Content, "第一段正文内容。") || !strings.Contains(output.Content, "第二段正文内容。") {
		t.Fatalf("expected body text, got %q", output.Content)
	}
	if strings.Contains(output.Content, "Ignored nav") || strings.Contains(output.Content, "Ignored footer") || strings.Contains(output.Content, "ignored") {
		t.Fatalf("expected nav/footer/script to be skipped, got %q", output.Content)
	}
}

func TestFetchTruncatesContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(strings.Repeat("字", 5000)))
	}))
	defer server.Close()

	output, err := newTestClient().Fetch(context.Background(), FetchInput{URL: server.URL, MaxChars: 1000})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if !output.Truncated {
		t.Fatalf("expected truncated output")
	}
	if got := len([]rune(output.Content)); got != 1000 {
		t.Fatalf("expected 1000 runes, got %d", got)
	}
}

func TestFetchRejectsUnsupportedContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0x00, 0x01})
	}))
	defer server.Close()

	_, err := newTestClient().Fetch(context.Background(), FetchInput{URL: server.URL})
	if err == nil || !strings.Contains(err.Error(), "unsupported content type") {
		t.Fatalf("expected unsupported content type error, got: %v", err)
	}
}

func TestFetchRejectsInvalidURL(t *testing.T) {
	_, err := newTestClient().Fetch(context.Background(), FetchInput{URL: "ftp://example.com/file"})
	if err == nil {
		t.Fatalf("expected invalid url error")
	}
}

// 管理员配置的搜索源地址是可信内网服务：生产环境开启 SSRF 防护时也必须可达；
// 而 Fetch 抓取模型提供的 URL，内网地址仍要被拦截。
func TestSSRFProtectionAllowsAdminConfiguredProviderButGuardsFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]string{
				{"title": "Internal OK", "url": "https://a.example.com", "content": "snippet"},
			},
		})
	}))
	defer server.Close()

	client := NewClientWithEnv("prod", true)

	output, err := client.Search(context.Background(), ProviderConfig{
		Provider:       ProviderSearXNG,
		SearXNGBaseURL: server.URL, // httptest 监听 127.0.0.1，属于会被 SSRF 防护拦截的地址
	}, SearchInput{Query: "test"})
	if err != nil {
		t.Fatalf("admin-configured searxng on loopback must be reachable, got: %v", err)
	}
	if len(output.Results) != 1 || output.Results[0].Title != "Internal OK" {
		t.Fatalf("unexpected results: %+v", output.Results)
	}

	if _, err := client.Fetch(context.Background(), FetchInput{URL: server.URL}); err == nil {
		t.Fatal("fetch of model-provided loopback url must be rejected under ssrf protection")
	}
}
