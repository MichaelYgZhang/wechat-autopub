package fetcher

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"strings"
)

// fetchDocx 下载并解析 .docx 文件
func fetchDocx(body io.Reader) (*Result, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("打开docx失败(非有效zip): %w", err)
	}

	// 找到 word/document.xml
	var docXML []byte
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("打开document.xml失败: %w", err)
			}
			docXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("读取document.xml失败: %w", err)
			}
			break
		}
	}
	if docXML == nil {
		return nil, fmt.Errorf("docx中未找到word/document.xml")
	}

	return parseDocxXML(docXML)
}

// parseDocxXML 用 token 方式解析 Word XML，转成 HTML
func parseDocxXML(data []byte) (*Result, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))

	var title string
	var htmlBuf strings.Builder

	// 状态跟踪
	var inParagraph bool
	var inRun bool
	var inText bool
	var pStyle string
	var runBold, runItalic, runUnderline bool
	var pTextBuf strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("解析XML失败: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p": // 段落开始
				inParagraph = true
				pStyle = ""
				pTextBuf.Reset()
			case "pStyle": // 段落样式
				for _, attr := range t.Attr {
					if attr.Name.Local == "val" {
						pStyle = attr.Value
					}
				}
			case "r": // run 开始
				inRun = true
				runBold = false
				runItalic = false
				runUnderline = false
			case "b": // 粗体
				if inRun {
					runBold = true
					// 检查 w:val="0" / "false" 的情况
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" && (attr.Value == "0" || attr.Value == "false") {
							runBold = false
						}
					}
				}
			case "i": // 斜体
				if inRun {
					runItalic = true
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" && (attr.Value == "0" || attr.Value == "false") {
							runItalic = false
						}
					}
				}
			case "u": // 下划线
				if inRun {
					runUnderline = true
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" && attr.Value == "none" {
							runUnderline = false
						}
					}
				}
			case "t": // 文本
				inText = true
			case "br": // 换行
				if inParagraph {
					pTextBuf.WriteString("<br>")
				}
			case "tab": // 制表符
				if inParagraph {
					pTextBuf.WriteString("&emsp;")
				}
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "p": // 段落结束 → 输出 HTML
				if inParagraph {
					text := pTextBuf.String()
					if strings.TrimSpace(text) != "" {
						tag := styleToHTMLTag(pStyle)
						htmlBuf.WriteString(fmt.Sprintf("<%s>%s</%s>\n", tag, text, tag))
						// 第一个标题作为文章标题
						if title == "" && isHeadingStyle(pStyle) {
							title = stripInlineTags(text)
						}
					}
					inParagraph = false
				}
			case "r": // run 结束
				inRun = false
			case "t": // 文本结束
				inText = false
			}

		case xml.CharData:
			if inParagraph && inText {
				text := html.EscapeString(string(t))
				// 应用行内格式
				if runBold {
					text = "<strong>" + text + "</strong>"
				}
				if runItalic {
					text = "<em>" + text + "</em>"
				}
				if runUnderline {
					text = "<u>" + text + "</u>"
				}
				pTextBuf.WriteString(text)
			}
		}
	}

	content := strings.TrimSpace(htmlBuf.String())
	if content == "" {
		return nil, fmt.Errorf("docx中未提取到文章内容")
	}

	return &Result{
		Title:   title,
		Content: content,
	}, nil
}

// styleToHTMLTag 将 Word 段落样式映射为 HTML 标签
func styleToHTMLTag(style string) string {
	s := strings.ToLower(style)
	switch {
	case s == "heading1" || s == "1" || s == "title":
		return "h1"
	case s == "heading2" || s == "2" || s == "subtitle":
		return "h2"
	case s == "heading3" || s == "3":
		return "h3"
	case s == "heading4" || s == "4":
		return "h4"
	case s == "heading5" || s == "5":
		return "h5"
	case s == "heading6" || s == "6":
		return "h6"
	default:
		return "p"
	}
}

func isHeadingStyle(style string) bool {
	tag := styleToHTMLTag(style)
	return len(tag) == 2 && tag[0] == 'h'
}

// stripInlineTags 去除行内 HTML 标签，保留纯文本
func stripInlineTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
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
