package wechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

const baseURL = "https://api.weixin.qq.com/cgi-bin"

// Client 微信公众号 API 客户端
type Client struct {
	AppID     string
	AppSecret string
	Token     string
}

// Article 草稿文章
type Article struct {
	Title            string `json:"title"`
	Author           string `json:"author"`
	Digest           string `json:"digest"`
	Content          string `json:"content"`
	ContentSourceURL string `json:"content_source_url"`
	ThumbMediaID     string `json:"thumb_media_id"`
}

// --- 响应结构体 ---

type apiError struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (e *apiError) check() error {
	if e.ErrCode != 0 {
		return fmt.Errorf("微信API错误: code=%d msg=%s", e.ErrCode, e.ErrMsg)
	}
	return nil
}

type tokenResp struct {
	apiError
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type materialResp struct {
	apiError
	MediaID string `json:"media_id"`
	URL     string `json:"url"`
}

type draftResp struct {
	apiError
	MediaID string `json:"media_id"`
}

type publishResp struct {
	apiError
	PublishID string `json:"publish_id"`
}

type PublishStatus struct {
	apiError
	PublishID     string        `json:"publish_id"`
	PublishStatus int           `json:"publish_status"` // 0=成功 1=发布中 2+失败
	ArticleID     string        `json:"article_id"`
	ArticleDetail articleDetail `json:"article_detail"`
}

type articleDetail struct {
	Count int          `json:"count"`
	Item  []articleItem `json:"item"`
}

type articleItem struct {
	Idx        int    `json:"idx"`
	ArticleURL string `json:"article_url"`
}

func (s *PublishStatus) StatusText() string {
	switch s.PublishStatus {
	case 0:
		return "发布成功"
	case 1:
		return "发布中"
	case 2:
		return "原创审核不通过"
	case 3:
		return "发布失败"
	case 4:
		return "尚未提交发布"
	default:
		return fmt.Sprintf("未知状态(%d)", s.PublishStatus)
	}
}

// --- API 方法 ---

// GetAccessToken 获取 access_token
func (c *Client) GetAccessToken() error {
	url := fmt.Sprintf("%s/token?grant_type=client_credential&appid=%s&secret=%s",
		baseURL, c.AppID, c.AppSecret)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result tokenResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	if err := result.check(); err != nil {
		return err
	}
	c.Token = result.AccessToken
	return nil
}

// UploadThumbImage 上传永久素材（图片）作为封面
func (c *Client) UploadThumbImage(filePath string) (string, error) {
	url := fmt.Sprintf("%s/material/add_material?access_token=%s&type=image",
		baseURL, c.Token)

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("创建表单失败: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("上传失败: %w", err)
	}
	defer resp.Body.Close()

	var result materialResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if err := result.check(); err != nil {
		return "", err
	}
	return result.MediaID, nil
}

// AddDraft 新建草稿
func (c *Client) AddDraft(article Article) (string, error) {
	url := fmt.Sprintf("%s/draft/add?access_token=%s", baseURL, c.Token)

	payload := map[string]interface{}{
		"articles": []Article{article},
	}
	data, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result draftResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if err := result.check(); err != nil {
		return "", err
	}
	return result.MediaID, nil
}

// SubmitPublish 提交发布
func (c *Client) SubmitPublish(mediaID string) (string, error) {
	url := fmt.Sprintf("%s/freepublish/submit?access_token=%s", baseURL, c.Token)

	payload := map[string]string{"media_id": mediaID}
	data, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result publishResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if err := result.check(); err != nil {
		return "", err
	}
	return result.PublishID, nil
}

// GetPublishStatus 查询发布状态
func (c *Client) GetPublishStatus(publishID string) (*PublishStatus, error) {
	url := fmt.Sprintf("%s/freepublish/get?access_token=%s", baseURL, c.Token)

	payload := map[string]string{"publish_id": publishID}
	data, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result PublishStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if err := result.check(); err != nil {
		return nil, err
	}
	return &result, nil
}
