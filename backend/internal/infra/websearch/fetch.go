package websearch

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
)

const (
	defaultFetchMaxChars = 8000
	maxAllowedFetchChars = 50000
)

// FetchInput 定义一次网页抓取请求。URL 由模型给出，必须走 SSRF 校验。
type FetchInput struct {
	URL       string
	MaxChars  int
	TimeoutMS int
}

// FetchOutput 是网页抓取结果，序列化后回灌给模型。
type FetchOutput struct {
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated,omitempty"`
}

// Fetch 抓取公开网页并提取正文文本。
func (c *Client) Fetch(ctx context.Context, input FetchInput) (FetchOutput, error) {
	target := strings.TrimSpace(input.URL)
	if target == "" {
		return FetchOutput{}, fmt.Errorf("fetch url is required")
	}
	if err := security.ValidateOutboundHTTPURL(target, c.env, c.ssrfProtectionEnabled); err != nil {
		return FetchOutput{}, fmt.Errorf("fetch url rejected: %w", err)
	}
	maxChars := normalizeFetchMaxChars(input.MaxChars)

	requestCtx, cancel := context.WithTimeout(ctx, resolveTimeout(input.TimeoutMS))
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, target, nil)
	if err != nil {
		return FetchOutput{}, err
	}
	req.Header.Set("User-Agent", fetchUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.5")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return FetchOutput{}, fmt.Errorf("fetch request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FetchOutput{}, fmt.Errorf("fetch returned status %d", resp.StatusCode)
	}

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	mediaType := contentType
	if parsed, _, parseErr := mime.ParseMediaType(contentType); parseErr == nil {
		mediaType = parsed
	}

	limited := io.LimitReader(resp.Body, maxResponseBodyBytes)
	switch {
	case strings.Contains(mediaType, "html"):
		title, text, extractErr := extractHTMLText(limited, contentType)
		if extractErr != nil {
			return FetchOutput{}, fmt.Errorf("fetch content parse failed: %w", extractErr)
		}
		content, truncated := truncateRunes(text, maxChars)
		return FetchOutput{URL: target, Title: title, Content: content, Truncated: truncated}, nil
	case strings.HasPrefix(mediaType, "text/"),
		strings.Contains(mediaType, "json"),
		strings.Contains(mediaType, "xml"),
		strings.Contains(mediaType, "markdown"):
		body, readErr := io.ReadAll(limited)
		if readErr != nil {
			return FetchOutput{}, fmt.Errorf("fetch response read failed: %w", readErr)
		}
		content, truncated := truncateRunes(strings.TrimSpace(string(body)), maxChars)
		return FetchOutput{URL: target, Content: content, Truncated: truncated}, nil
	default:
		return FetchOutput{}, fmt.Errorf("fetch unsupported content type: %s", mediaType)
	}
}

func normalizeFetchMaxChars(value int) int {
	if value <= 0 {
		return defaultFetchMaxChars
	}
	if value > maxAllowedFetchChars {
		return maxAllowedFetchChars
	}
	return value
}

// skippedHTMLElements 是正文提取时整体跳过的标签。
var skippedHTMLElements = map[string]struct{}{
	"script":   {},
	"style":    {},
	"noscript": {},
	"template": {},
	"svg":      {},
	"iframe":   {},
	"head":     {},
	"nav":      {},
	"footer":   {},
	"aside":    {},
	"form":     {},
	"button":   {},
}

// blockHTMLElements 结束时补换行，保持段落结构。
var blockHTMLElements = map[string]struct{}{
	"p": {}, "div": {}, "section": {}, "article": {}, "main": {},
	"h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {}, "h6": {},
	"li": {}, "tr": {}, "br": {}, "blockquote": {}, "pre": {}, "table": {},
}

// extractHTMLText 解析 HTML 并提取标题与正文文本。
// 未引入 readability 依赖：跳过导航/脚本类标签、按块级元素分段，足够供模型阅读。
func extractHTMLText(reader io.Reader, contentType string) (string, string, error) {
	decoded, err := charset.NewReader(reader, contentType)
	if err != nil {
		return "", "", err
	}
	root, err := html.Parse(decoded)
	if err != nil {
		return "", "", err
	}

	var title string
	var builder strings.Builder
	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			name := strings.ToLower(node.Data)
			if name == "title" && title == "" {
				title = strings.TrimSpace(nodeText(node))
				return
			}
			if _, skip := skippedHTMLElements[name]; skip {
				return
			}
		}
		if node.Type == html.TextNode {
			text := strings.TrimSpace(node.Data)
			if text != "" {
				builder.WriteString(text)
				builder.WriteString(" ")
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if node.Type == html.ElementNode {
			if _, block := blockHTMLElements[strings.ToLower(node.Data)]; block {
				builder.WriteString("\n")
			}
		}
	}
	// <title> 位于 <head> 中，先单独提取再遍历正文。
	if head := findHTMLElement(root, "head"); head != nil {
		if titleNode := findHTMLElement(head, "title"); titleNode != nil {
			title = strings.TrimSpace(nodeText(titleNode))
		}
	}
	body := findHTMLElement(root, "body")
	if body == nil {
		body = root
	}
	walk(body)

	return title, collapseBlankLines(builder.String()), nil
}

func findHTMLElement(node *html.Node, name string) *html.Node {
	if node == nil {
		return nil
	}
	if node.Type == html.ElementNode && strings.EqualFold(node.Data, name) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findHTMLElement(child, name); found != nil {
			return found
		}
	}
	return nil
}

func nodeText(node *html.Node) string {
	var builder strings.Builder
	var walk func(current *html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return builder.String()
}

func collapseBlankLines(value string) string {
	lines := strings.Split(value, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(result) > 0 && result[len(result)-1] == "" {
				continue
			}
			result = append(result, "")
			continue
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

func truncateRunes(value string, maxChars int) (string, bool) {
	runes := []rune(value)
	if maxChars <= 0 || len(runes) <= maxChars {
		return value, false
	}
	return strings.TrimSpace(string(runes[:maxChars])), true
}
