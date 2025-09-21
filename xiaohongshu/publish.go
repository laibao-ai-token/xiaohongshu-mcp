package xiaohongshu

// 小红书“发布图文”自动化流程说明：
// - 打开创作中心发布页，切换到“上传图文”Tab；
// - 上传图片；
// - 输入标题、正文与标签；
// - 点击提交完成发布。
//
// 这里使用 go-rod 驱动浏览器执行页面操作；
// 上层 service 负责图片来源处理（URL 下载/本地路径）与参数校验，这里只做页面交互。

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
)

// PublishImageContent 发布图文内容
// - Title: 标题（服务层已对显示宽度做 40 限制）
// - Content: 正文描述
// - Tags: 话题标签（不用带 #，后续逻辑会自动补）
// - ImagePaths: 本地图片路径（URL 已在上层被下载为本地文件）
type PublishImageContent struct {
	Title      string
	Content    string
	Tags       []string
	ImagePaths []string
}

type PublishAction struct {
	page *rod.Page
}

const (
	// 创作页地址（官方发布入口）
	urlOfPublic = `https://creator.xiaohongshu.com/publish/publish?source=official`
)

// NewPublishImageAction 打开创作页并初始化到“上传图文”界面
func NewPublishImageAction(page *rod.Page) (*PublishAction, error) {

	pp := page.Timeout(60 * time.Second)

	// 导航到创作页
	pp.MustNavigate(urlOfPublic)

	// 等待上传容器可见，确保页面可交互
	pp.MustElement(`div.upload-content`).MustWaitVisible()
	slog.Info("wait for upload-content visible success")

	// 等待一段时间确保页面完全加载
	time.Sleep(1 * time.Second)

	// 切换到“上传图文”Tab
	createElems := pp.MustElements("div.creator-tab")
	slog.Info("foundcreator-tab elements", "count", len(createElems))
	for _, elem := range createElems {
		text, err := elem.Text()
		if err != nil {
			slog.Error("获取元素文本失败", "error", err)
			continue
		}

		if text == "上传图文" {
			if err := elem.Click(proto.InputMouseButtonLeft, 1); err != nil {
				slog.Error("点击元素失败", "error", err)
				continue
			}
			break
		}
	}

	time.Sleep(1 * time.Second)

	return &PublishAction{
		page: pp,
	}, nil
}

// Publish 执行完整的“发布图文”流程
func (p *PublishAction) Publish(ctx context.Context, content PublishImageContent) error {
	if len(content.ImagePaths) == 0 {
		return errors.New("图片不能为空")
	}

	page := p.page.Context(ctx)

	if err := uploadImages(page, content.ImagePaths); err != nil {
		return errors.Wrap(err, "小红书上传图片失败")
	}

	if err := submitPublish(page, content.Title, content.Content, content.Tags); err != nil {
		return errors.Wrap(err, "小红书发布失败")
	}

	return nil
}

// uploadImages 选择上传文件并等待上传完成
func uploadImages(page *rod.Page, imagesPaths []string) error {
	pp := page.Timeout(30 * time.Second)

	// 等待上传输入框出现
	uploadInput := pp.MustElement(".upload-input")

	// 上传多个文件
	uploadInput.MustSetFiles(imagesPaths...)

	// 等待上传完成
	time.Sleep(3 * time.Second)

	return nil
}

// submitPublish 填写标题、正文与标签并点击提交
func submitPublish(page *rod.Page, title, content string, tags []string) error {

	titleElem := page.MustElement("div.d-input input")
	titleElem.MustInput(title)

	time.Sleep(1 * time.Second)

	if contentElem, ok := getContentElement(page); ok {
		contentElem.MustInput(content)

		inputTags(contentElem, tags)

	} else {
		return errors.New("没有找到内容输入框")
	}

	time.Sleep(1 * time.Second)

	submitButton := page.MustElement("div.submit div.d-button-content")
	submitButton.MustClick()

	time.Sleep(3 * time.Second)

	return nil
}

// 查找内容输入框 - 使用 Race 方法兼容两种页面样式：
// 1) 直接存在富文本容器 div.ql-editor；
// 2) 通过包含占位符“输入正文描述”的 p 元素向上回溯到 role=textbox 的父元素。
func getContentElement(page *rod.Page) (*rod.Element, bool) {
	var foundElement *rod.Element
	var found bool

	page.Race().
		Element("div.ql-editor").MustHandle(func(e *rod.Element) {
		foundElement = e
		found = true
	}).
		ElementFunc(func(page *rod.Page) (*rod.Element, error) {
			return findTextboxByPlaceholder(page)
		}).MustHandle(func(e *rod.Element) {
		foundElement = e
		found = true
	}).
		MustDo()

	if found {
		return foundElement, true
	}

	slog.Warn("no content element found by any method")
	return nil, false
}

func inputTags(contentElem *rod.Element, tags []string) {
	if len(tags) == 0 {
		return
	}

	time.Sleep(1 * time.Second)

	for i := 0; i < 20; i++ {
		contentElem.MustKeyActions().
			Type(input.ArrowDown).
			MustDo()
		time.Sleep(10 * time.Millisecond)
	}

	contentElem.MustKeyActions().
		Press(input.Enter).
		Press(input.Enter).
		MustDo()

	time.Sleep(1 * time.Second)

	for _, tag := range tags {
		tag = strings.TrimLeft(tag, "#")
		inputTag(contentElem, tag)
	}
}

// inputTag 输入单个标签：先键入“#”，随后输入标签文本；
// 若检测到联想下拉（#creator-editor-topic-container），优先选择第一项；
// 否则输入空格结束标签。
func inputTag(contentElem *rod.Element, tag string) {
	contentElem.MustInput("#")
	time.Sleep(200 * time.Millisecond)

	for _, char := range tag {
		contentElem.MustInput(string(char))
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)

	page := contentElem.Page()
	topicContainer, err := page.Element("#creator-editor-topic-container")
	if err == nil && topicContainer != nil {
		firstItem, err := topicContainer.Element(".item")
		if err == nil && firstItem != nil {
			firstItem.MustClick()
			slog.Info("成功点击标签联想选项", "tag", tag)
			time.Sleep(200 * time.Millisecond)
		} else {
			slog.Warn("未找到标签联想选项，直接输入空格", "tag", tag)
			// 如果没有找到联想选项，输入空格结束
			contentElem.MustInput(" ")
		}
	} else {
		slog.Warn("未找到标签联想下拉框，直接输入空格", "tag", tag)
		// 如果没有找到下拉框，输入空格结束
		contentElem.MustInput(" ")
	}

	time.Sleep(500 * time.Millisecond) // 等待标签处理完成
}

// findTextboxByPlaceholder 通过占位符文本“输入正文描述”定位正文输入区域：
// 1) 遍历页面中的 p 元素，找到 data-placeholder 包含指定文本的元素；
// 2) 自该元素向上回溯，查找最近的 role=textbox 父元素并返回。
func findTextboxByPlaceholder(page *rod.Page) (*rod.Element, error) {
	elements := page.MustElements("p")
	if elements == nil {
		return nil, errors.New("no p elements found")
	}

	// 查找包含指定placeholder的元素
	placeholderElem := findPlaceholderElement(elements, "输入正文描述")
	if placeholderElem == nil {
		return nil, errors.New("no placeholder element found")
	}

	// 向上查找textbox父元素
	textboxElem := findTextboxParent(placeholderElem)
	if textboxElem == nil {
		return nil, errors.New("no textbox parent found")
	}

	return textboxElem, nil
}

// findPlaceholderElement 在一组 p 元素中查找 data-placeholder 含 searchText 的元素，返回首个匹配项。
func findPlaceholderElement(elements []*rod.Element, searchText string) *rod.Element {
	for _, elem := range elements {
		placeholder, err := elem.Attribute("data-placeholder")
		if err != nil || placeholder == nil {
			continue
		}

		if strings.Contains(*placeholder, searchText) {
			return elem
		}
	}
	return nil
}

// findTextboxParent 自下而上回溯至最近的 role=textbox 父元素（最多向上 5 层）。
func findTextboxParent(elem *rod.Element) *rod.Element {
	currentElem := elem
	for i := 0; i < 5; i++ {
		parent, err := currentElem.Parent()
		if err != nil {
			break
		}

		role, err := parent.Attribute("role")
		if err != nil || role == nil {
			currentElem = parent
			continue
		}

		if *role == "textbox" {
			return parent
		}

		currentElem = parent
	}
	return nil
}
