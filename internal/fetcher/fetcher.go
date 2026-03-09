package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// Result 抓取结果
type Result struct {
	Title   string
	Content string // HTML 格式
}

// FetchArticle 从 URL 或本地文件路径获取文章内容
func FetchArticle(source string) (*Result, error) {
	// 判断是本地文件还是 URL
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		return fetchLocalFile(source)
	}

	resp, err := http.Get(source)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	ext := strings.ToLower(path.Ext(source))
	contentType := resp.Header.Get("Content-Type")

	switch {
	case ext == ".docx" || strings.Contains(contentType, "officedocument.wordprocessingml"):
		return fetchDocx(resp.Body)
	case ext == ".md" || ext == ".markdown" ||
		strings.Contains(contentType, "text/markdown"):
		return fetchMarkdown(resp.Body)
	default:
		return fetchHTML(resp.Body)
	}
}

// fetchLocalFile 读取本地文件
func fetchLocalFile(filePath string) (*Result, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	ext := strings.ToLower(path.Ext(filePath))
	switch ext {
	case ".docx":
		return fetchDocx(f)
	case ".md", ".markdown":
		return fetchMarkdown(f)
	default:
		return fetchHTML(f)
	}
}

func fetchMarkdown(body io.Reader) (*Result, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("读取内容失败: %w", err)
	}

	mdText := string(raw)

	// 从 Markdown 提取标题（第一个 # 开头的行）
	title := ""
	for _, line := range strings.Split(mdText, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title = strings.TrimPrefix(trimmed, "# ")
			break
		}
	}

	// Markdown -> HTML
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	renderer := html.NewRenderer(html.RendererOptions{Flags: htmlFlags})
	htmlContent := string(markdown.ToHTML(raw, p, renderer))

	if strings.TrimSpace(htmlContent) == "" {
		return nil, fmt.Errorf("未能提取到文章内容")
	}

	return &Result{
		Title:   title,
		Content: htmlContent,
	}, nil
}

func fetchHTML(body io.Reader) (*Result, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %w", err)
	}

	title := extractTitle(doc)
	content := extractContent(doc)

	if content == "" {
		return nil, fmt.Errorf("未能提取到文章内容")
	}

	return &Result{
		Title:   title,
		Content: content,
	}, nil
}

func extractTitle(doc *goquery.Document) string {
	// 优先使用 og:title
	if title, exists := doc.Find(`meta[property="og:title"]`).Attr("content"); exists && title != "" {
		return strings.TrimSpace(title)
	}
	// 然后 h1
	if title := doc.Find("h1").First().Text(); title != "" {
		return strings.TrimSpace(title)
	}
	// 最后 <title>
	return strings.TrimSpace(doc.Find("title").First().Text())
}

func extractContent(doc *goquery.Document) string {
	// 移除不需要的元素
	doc.Find("script, style, nav, header, footer, .sidebar, .comments, .ad").Remove()

	// 按优先级尝试常见的文章容器选择器
	selectors := []string{
		"article",
		".article-content",
		".post-content",
		".entry-content",
		".article-body",
		".rich_media_content", // 微信文章
		"#content",
		".content",
		"main",
	}

	for _, sel := range selectors {
		s := doc.Find(sel).First()
		if s.Length() > 0 {
			html, err := s.Html()
			if err == nil && strings.TrimSpace(html) != "" {
				return cleanHTML(html)
			}
		}
	}

	// 兜底：用 body
	html, _ := doc.Find("body").Html()
	return cleanHTML(html)
}

func cleanHTML(html string) string {
	html = strings.TrimSpace(html)
	// 基本清理：移除多余空行
	lines := strings.Split(html, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.Join(cleaned, "\n")
}
