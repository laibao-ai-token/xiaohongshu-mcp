package xiaohongshu

// 提供对小红书首页 Feed 列表（初始渲染数据）的抓取：
// - 导航至首页并等待页面稳定；
// - 等待 window.__INITIAL_STATE__ 注入；
// - 反序列化为结构化的 Feed 列表返回。

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

// FeedsListAction 封装 rod.Page，提供抓取小红书首页 feed 列表的能力。
type FeedsListAction struct {
	page *rod.Page
}

// NewFeedsListAction 创建一个用于抓取 Feed 列表数据的动作对象。
// - 为页面操作设置 60s 的默认超时；
// - 导航到小红书首页并等待页面稳定；
// - 返回封装后的 FeedsListAction。
// 注意：Must* 系列方法在失败时会 panic（与本项目其它动作保持一致）。
func NewFeedsListAction(page *rod.Page) *FeedsListAction {
	pp := page.Timeout(60 * time.Second)

	pp.MustNavigate("https://www.xiaohongshu.com")
	// 使用 WaitStable（而非固定 sleep）避免早期 DOM 抖动
	pp.MustWaitStable()

	return &FeedsListAction{page: pp}
}

// GetFeedsList 读取首页初始渲染中的 Feed 列表。
// 实现：
// 1) 绑定 ctx；
// 2) 条件等待 __INITIAL_STATE__ 注入；
// 3) 读取并反序列化，提取 feed.feeds._value。
func (f *FeedsListAction) GetFeedsList(ctx context.Context) ([]Feed, error) {
	page := f.page.Context(ctx)

	// 等待初始化状态注入并包含 feed 路径，替代固定 Sleep 带来的不稳定
	page.MustWait(`() => {
        const s = window.__INITIAL_STATE__;
        return !!(s && s.feed && s.feed.feeds);
    }`)

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

	// 解析完整的 InitialState（使用统一的类型定义，避免重复）
	var state FeedResponse
	if err := json.Unmarshal([]byte(result), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal __INITIAL_STATE__: %w", err)
	}

	return state.Feed.Feeds.Value, nil
}
