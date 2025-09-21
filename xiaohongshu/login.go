package xiaohongshu

// 登录与登录状态检查逻辑
// 使用 rod 驱动浏览器打开小红书网站，判断是否已登录；
// 若未登录，触发扫码登录并等待登录完成。
// 注意：Cookies 的加载/保存由其他模块负责，这里只进行页面动作。

import (
	"context"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
)

// LoginAction 封装 rod.Page，在该页面上执行登录相关操作。
type LoginAction struct {
	page *rod.Page
}

// NewLogin 创建登录动作实例。调用方负责传入已创建的 rod.Page。
func NewLogin(page *rod.Page) *LoginAction {
	return &LoginAction{page: page}
}

// CheckLoginStatus 打开小红书首页并检查是否已登录。
// 返回 (isLoggedIn, error)。仅“未登录”时通常返回 (false, nil)；
// 只有当 rod 查询发生异常时才返回非空 error。
// 判定依据：页面中是否存在仅登录后才会出现的元素：
//
//	`.main-container .user .link-wrapper .channel`
func (a *LoginAction) CheckLoginStatus(ctx context.Context) (bool, error) {
	// 将页面操作绑定到上层 ctx，便于超时/取消。
	pp := a.page.Context(ctx)
	// 导航到首页并等待加载完成。
	pp.MustNavigate("https://www.xiaohongshu.com/explore").MustWaitLoad()

	// 额外等待，确保 DOM 稳定，避免元素刚渲染导致误判。
	time.Sleep(1 * time.Second)

	// 检查“仅登录后才出现”的元素是否存在。
	exists, _, err := pp.Has(`.main-container .user .link-wrapper .channel`)
	// Rod 查询发生异常（网络/脚本等）
	if err != nil {
		return false, errors.Wrap(err, "check login status failed")
	}

	// 未登录：未找到该元素（errors.Wrap(nil, ...) 依然返回 nil）
	if !exists {
		return false, errors.Wrap(err, "login status element not found")
	}

	// 已登录：找到该元素
	return true, nil
}

// Login 执行扫码登录流程：打开首页 → 若已登录直接返回 → 否则等待“登录后元素”出现。
func (a *LoginAction) Login(ctx context.Context) error {
	// 绑定上层 ctx
	pp := a.page.Context(ctx)

	// 导航到小红书首页，这会触发二维码弹窗
	// 导航到首页并等待加载完成（通常会触发二维码弹窗）
	pp.MustNavigate("https://www.xiaohongshu.com/explore").MustWaitLoad()

	// 等待一小段时间让页面完全加载
	// 稍等，确保二维码/页面元素渲染完成
	time.Sleep(2 * time.Second)

	// 检查是否已经登录
	// 若已登录则直接返回
	if exists, _, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		// 已经登录，直接返回
		return nil
	}

	// 等待扫码成功提示或者登录完成
	// 这里我们等待登录成功的元素出现，这样更简单可靠
	// 阻塞等待登录后才会出现的元素，视为登录成功
	pp.MustElement(".main-container .user .link-wrapper .channel")

	return nil
}
