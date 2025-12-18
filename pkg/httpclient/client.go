package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client HTTP 客户端
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// Option 客户端配置选项
type Option func(*Client)

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithBaseURL 设置基础 URL
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// New 创建 HTTP 客户端
func New(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Request 请求参数
type Request struct {
	Method  string
	Path    string
	Query   url.Values
	Headers map[string]string
}

// Get 发送 GET 请求并解析 JSON 响应
func (c *Client) Get(ctx context.Context, path string, query url.Values, result interface{}) error {
	return c.DoJSON(ctx, &Request{
		Method: http.MethodGet,
		Path:   path,
		Query:  query,
	}, result)
}

// DoJSON 发送请求并解析 JSON 响应
func (c *Client) DoJSON(ctx context.Context, req *Request, result interface{}) error {
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 错误 %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("JSON 解析失败: %w, body: %s", err, string(body))
		}
	}

	return nil
}

// Do 发送 HTTP 请求
func (c *Client) Do(ctx context.Context, req *Request) (*http.Response, error) {
	// 构建完整 URL
	fullURL := c.baseURL + req.Path
	if len(req.Query) > 0 {
		fullURL += "?" + req.Query.Encode()
	}

	// 创建请求
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 发送请求
	return c.httpClient.Do(httpReq)
}

// SetBaseURL 设置基础 URL
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}
