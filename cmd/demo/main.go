package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/example/wechat-autopub/internal/fetcher"
	"github.com/example/wechat-autopub/internal/wechat"
)

func main() {
	// 命令行参数
	appID := flag.String("app-id", os.Getenv("WECHAT_APP_ID"), "微信 AppID (或设置 WECHAT_APP_ID 环境变量)")
	appSecret := flag.String("app-secret", os.Getenv("WECHAT_APP_SECRET"), "微信 AppSecret (或设置 WECHAT_APP_SECRET 环境变量)")
	articleURL := flag.String("url", "", "要抓取的文章 URL")
	title := flag.String("title", "", "文章标题 (不填则从 URL 提取)")
	content := flag.String("content", "", "文章内容 HTML (不填则从 URL 提取)")
	author := flag.String("author", "Auto Publisher", "作者名称")
	thumbMediaID := flag.String("thumb-media-id", "", "封面图片 media_id")
	thumbImage := flag.String("thumb-image", "", "封面图片文件路径 (如果没有 thumb-media-id 则上传此图片)")
	doPublish := flag.Bool("publish", false, "是否提交发布 (默认只创建草稿)")
	pollStatus := flag.Bool("poll", false, "发布后轮询状态直到完成")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `微信公众号发布 Demo

用法:
  go run cmd/demo/main.go [参数]

示例:
  # 从 URL 抓取文章，上传封面图，创建草稿
  go run cmd/demo/main.go \
    --app-id=YOUR_APP_ID \
    --app-secret=YOUR_APP_SECRET \
    --url=https://example.com/article \
    --thumb-image=cover.jpg

  # 创建草稿并发布
  go run cmd/demo/main.go \
    --app-id=YOUR_APP_ID \
    --app-secret=YOUR_APP_SECRET \
    --url=https://example.com/article \
    --thumb-media-id=EXISTING_MEDIA_ID \
    --publish

参数:
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	// 校验必填参数
	if *appID == "" || *appSecret == "" {
		fatal("必须提供 --app-id 和 --app-secret (或设置环境变量 WECHAT_APP_ID / WECHAT_APP_SECRET)")
	}
	if *articleURL == "" && *content == "" {
		fatal("必须提供 --url (抓取文章) 或 --content (直接提供内容)")
	}
	if *thumbMediaID == "" && *thumbImage == "" {
		fatal("必须提供 --thumb-media-id 或 --thumb-image (封面图片)")
	}

	// ========== Step 1: 获取 access_token ==========
	step("1", "获取 access_token")
	client := &wechat.Client{
		AppID:     *appID,
		AppSecret: *appSecret,
	}
	if err := client.GetAccessToken(); err != nil {
		fatal("获取 access_token 失败: %v", err)
	}
	success("access_token: %s...%s", client.Token[:8], client.Token[len(client.Token)-8:])

	// ========== Step 2: 上传封面图片 (如果需要) ==========
	if *thumbMediaID == "" {
		step("2", "上传封面图片: %s", *thumbImage)
		mediaID, err := client.UploadThumbImage(*thumbImage)
		if err != nil {
			fatal("上传封面图片失败: %v", err)
		}
		*thumbMediaID = mediaID
		success("thumb_media_id: %s", mediaID)
	} else {
		step("2", "使用已有封面 thumb_media_id: %s", *thumbMediaID)
	}

	// ========== Step 3: 获取文章内容 ==========
	articleTitle := *title
	articleContent := *content

	if articleContent == "" {
		step("3", "抓取文章: %s", *articleURL)
		result, err := fetcher.FetchArticle(*articleURL)
		if err != nil {
			fatal("抓取文章失败: %v", err)
		}
		if articleTitle == "" {
			articleTitle = result.Title
		}
		articleContent = result.Content
		success("标题: %s", articleTitle)
		// 显示内容预览
		preview := articleContent
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		info("内容预览: %s", preview)
	} else {
		step("3", "使用手动提供的内容")
		if articleTitle == "" {
			articleTitle = "Demo Article"
		}
	}

	// ========== Step 4: 创建草稿 ==========
	step("4", "创建草稿")
	article := wechat.Article{
		Title:            articleTitle,
		Author:           *author,
		Content:          articleContent,
		ContentSourceURL: *articleURL,
		ThumbMediaID:     *thumbMediaID,
		Digest:           truncate(stripHTML(articleContent), 54),
	}
	draftMediaID, err := client.AddDraft(article)
	if err != nil {
		fatal("创建草稿失败: %v", err)
	}
	success("草稿 media_id: %s", draftMediaID)

	if !*doPublish {
		info("草稿创建成功！如需发布，请添加 --publish 参数")
		return
	}

	// ========== Step 5: 提交发布 ==========
	step("5", "提交发布")
	publishID, err := client.SubmitPublish(draftMediaID)
	if err != nil {
		fatal("提交发布失败: %v", err)
	}
	success("publish_id: %s", publishID)

	if !*pollStatus {
		info("发布已提交！如需等待结果，请添加 --poll 参数")
		info("也可以稍后手动查询发布状态")
		return
	}

	// ========== Step 6: 轮询发布状态 ==========
	step("6", "轮询发布状态")
	for i := 0; i < 30; i++ {
		time.Sleep(10 * time.Second)
		status, err := client.GetPublishStatus(publishID)
		if err != nil {
			warn("查询状态失败: %v (将重试)", err)
			continue
		}
		info("状态: %s", status.StatusText())

		switch status.PublishStatus {
		case 0: // 成功
			success("发布成功!")
			if status.ArticleDetail.Count > 0 {
				for _, item := range status.ArticleDetail.Item {
					success("文章链接: %s", item.ArticleURL)
				}
			}
			return
		case 1: // 发布中
			continue
		default: // 失败
			fatal("发布失败: %s", status.StatusText())
		}
	}
	warn("轮询超时，请稍后手动查询发布状态")
}

// --- 辅助函数 ---

func step(num, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("\n[Step %s] %s\n", num, msg)
}

func success(format string, args ...interface{}) {
	fmt.Printf("  ✓ %s\n", fmt.Sprintf(format, args...))
}

func info(format string, args ...interface{}) {
	fmt.Printf("  → %s\n", fmt.Sprintf(format, args...))
}

func warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "  ⚠ %s\n", fmt.Sprintf(format, args...))
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "  ✗ %s\n", fmt.Sprintf(format, args...))
	os.Exit(1)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func stripHTML(html string) string {
	var result strings.Builder
	inTag := false
	for _, r := range html {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}
