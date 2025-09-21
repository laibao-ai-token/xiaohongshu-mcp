package xiaohongshu

// 本文件实现“站内搜索”动作：
// - 构造搜索页 URL 并导航；
// - 等待页面渲染与全局状态注入；
// - 从 window.__INITIAL_STATE__ 解析结构化搜索结果；
// - 返回统一的 Feed 列表（由 FeedsValue.Value 承载）。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/go-rod/rod"
)

// SearchResult 对应 __INITIAL_STATE__ 中与搜索结果相关的片段
type SearchResult struct {
	Search struct {
		Feeds FeedsValue `json:"feeds"`
	} `json:"search"`
}

// SearchAction 封装一个 rod.Page，在该页面上执行搜索相关操作
type SearchAction struct {
	page *rod.Page
}

// NewSearchAction 绑定页面并设置默认超时
func NewSearchAction(page *rod.Page) *SearchAction {
	pp := page.Timeout(60 * time.Second)

	return &SearchAction{page: pp}
}

// Search 以关键字执行站内搜索并返回 Feed 列表
// 步骤：
// 1) 构造搜索 URL 并导航；
// 2) 等待页面稳定（MustWaitStable），避免早期 DOM 抖动；
// 3) 等待 window.__INITIAL_STATE__ 注入；
// 4) 读取并反序列化为 SearchResult，返回 feeds 值。
func (s *SearchAction) Search(ctx context.Context, keyword string) ([]Feed, error) {
	page := s.page.Context(ctx)

	searchURL := makeSearchURL(keyword)
	page.MustNavigate(searchURL)
	page.MustWaitStable()

	page.MustWait(`() => window.__INITIAL_STATE__ !== undefined`)

	// 获取 window.__INITIAL_STATE__ 并转换为 JSON 字符串
	result := page.MustEval(`() => {
			if (window.__INITIAL_STATE__) {
				return JSON.stringify(window.__INITIAL_STATE__);
			}
			return "";
		}`).String()

	if result == "" {
		return nil, fmt.Errorf("__INITIAL_STATE__ not found")
	}

	var searchResult SearchResult
	if err := json.Unmarshal([]byte(result), &searchResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal __INITIAL_STATE__: %w", err)
	}

	return searchResult.Search.Feeds.Value, nil
}

// makeSearchURL 生成搜索结果页的 URL，携带 keyword 与 source 等查询参数
func makeSearchURL(keyword string) string {

	values := url.Values{}
	values.Set("keyword", keyword)
	values.Set("source", "web_explore_feed")

	return fmt.Sprintf("https://www.xiaohongshu.com/search_result?%s", values.Encode())
}
