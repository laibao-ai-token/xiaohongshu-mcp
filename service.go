package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/xpzouying/headless_browser"
	"github.com/xpzouying/xiaohongshu-mcp/browser"
	"github.com/xpzouying/xiaohongshu-mcp/configs"
	"github.com/xpzouying/xiaohongshu-mcp/pkg/downloader"
	"github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"
)

// XiaohongshuService 小红书业务服务
type XiaohongshuService struct{}

// NewXiaohongshuService 创建小红书服务实例
func NewXiaohongshuService() *XiaohongshuService {
	return &XiaohongshuService{}
}

// PublishRequest 发布请求
type PublishRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Images  []string `json:"images" binding:"required,min=1"`
	Tags    []string `json:"tags,omitempty"`
}

// LoginStatusResponse 登录状态响应
type LoginStatusResponse struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	Username   string `json:"username,omitempty"`
}

// PublishResponse 发布响应
type PublishResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Images  int    `json:"images"`
	Status  string `json:"status"`
	PostID  string `json:"post_id,omitempty"`
}

// FeedsListResponse Feeds列表响应
type FeedsListResponse struct {
	Feeds []xiaohongshu.Feed `json:"feeds"`
	Count int                `json:"count"`
}

// CheckLoginStatus 检查登录状态
func (s *XiaohongshuService) CheckLoginStatus(ctx context.Context) (*LoginStatusResponse, error) {
	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	loginAction := xiaohongshu.NewLogin(page)

	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		return nil, err
	}

	response := &LoginStatusResponse{
		IsLoggedIn: isLoggedIn,
		Username:   configs.Username,
	}

	return response, nil
}

// PublishContent 发布内容
func (s *XiaohongshuService) PublishContent(ctx context.Context, req *PublishRequest) (*PublishResponse, error) {
	// 验证标题长度
	// 小红书限制：最大40个单位长度
	// 中文/日文/韩文占2个单位，英文/数字占1个单位
	if titleWidth := runewidth.StringWidth(req.Title); titleWidth > 40 {
		return nil, fmt.Errorf("标题长度超过限制")
	}

	// 处理图片：下载URL图片或使用本地路径
	imagePaths, err := s.processImages(req.Images)
	if err != nil {
		return nil, err
	}

	// 构建发布内容
	content := xiaohongshu.PublishImageContent{
		Title:      req.Title,
		Content:    req.Content,
		Tags:       req.Tags,
		ImagePaths: imagePaths,
	}

	// 执行发布
	if err := s.publishContent(ctx, content); err != nil {
		return nil, err
	}

	response := &PublishResponse{
		Title:   req.Title,
		Content: req.Content,
		Images:  len(imagePaths),
		Status:  "发布完成",
	}

	return response, nil
}

// processImages 处理图片列表，支持URL下载和本地路径
func (s *XiaohongshuService) processImages(images []string) ([]string, error) {
	processor := downloader.NewImageProcessor()
	return processor.ProcessImages(images)
}

// publishContent 执行内容发布
func (s *XiaohongshuService) publishContent(ctx context.Context, content xiaohongshu.PublishImageContent) error {
	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	action, err := xiaohongshu.NewPublishImageAction(page)
	if err != nil {
		return err
	}

	// 执行发布
	return action.Publish(ctx, content)
}

// ListFeeds 获取Feeds列表
func (s *XiaohongshuService) ListFeeds(ctx context.Context) (*FeedsListResponse, error) {
	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	// 创建 Feeds 列表 action
	action := xiaohongshu.NewFeedsListAction(page)

	// 获取 Feeds 列表
	feeds, err := action.GetFeedsList(ctx)
	if err != nil {
		return nil, err
	}

	response := &FeedsListResponse{
		Feeds: feeds,
		Count: len(feeds),
	}

	return response, nil
}

func (s *XiaohongshuService) SearchFeeds(ctx context.Context, keyword string) (*FeedsListResponse, error) {
	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	action := xiaohongshu.NewSearchAction(page)

	feeds, err := action.Search(ctx, keyword)
	if err != nil {
		return nil, err
	}

	response := &FeedsListResponse{
		Feeds: feeds,
		Count: len(feeds),
	}

	return response, nil
}

// GetFeedDetail 获取Feed详情
func (s *XiaohongshuService) GetFeedDetail(ctx context.Context, feedID, xsecToken string) (*FeedDetailResponse, error) {
	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	// 创建 Feed 详情 action
	action := xiaohongshu.NewFeedDetailAction(page)

	// 获取 Feed 详情
	result, err := action.GetFeedDetail(ctx, feedID, xsecToken)
	if err != nil {
		return nil, err
	}

	response := &FeedDetailResponse{
		FeedID: feedID,
		Data:   result,
	}

	return response, nil
}

// PostCommentToFeed 发表评论到Feed
func (s *XiaohongshuService) PostCommentToFeed(ctx context.Context, feedID, xsecToken, content string) (*PostCommentResponse, error) {
	// 使用非无头模式以便查看操作过程
	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	// 创建 Feed 评论 action
	action := xiaohongshu.NewCommentFeedAction(page)

	// 发表评论
	err := action.PostComment(ctx, feedID, xsecToken, content)
	if err != nil {
		return nil, err
	}

	response := &PostCommentResponse{
		FeedID:  feedID,
		Success: true,
		Message: "评论发表成功",
	}

	return response, nil
}

func newBrowser() *headless_browser.Browser {
	return browser.NewBrowser(configs.IsHeadless(), browser.WithBinPath(configs.GetBinPath()))
}

// SaveRecommendedFeedsContent 抓取首页推荐前 N 条的详情内容，按标题排序保存为 Markdown 文件
func (s *XiaohongshuService) SaveRecommendedFeedsContent(ctx context.Context, limit int, outputDir string) (*SaveFeedsResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	// 默认输出目录：工作目录下 content
	if strings.TrimSpace(outputDir) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		outputDir = filepath.Join(cwd, "content")
	}

	// 为本次保存创建子目录（按时间戳），避免不同批次混在一起
	ts := time.Now().Format("20060102_150405")
	batchDir := filepath.Join(outputDir, ts)
	if err := os.MkdirAll(batchDir, 0o755); err != nil {
		return nil, err
	}

	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	// 1) 获取首页推荐 feeds
	feedsAction := xiaohongshu.NewFeedsListAction(page)
	feeds, err := feedsAction.GetFeedsList(ctx)
	if err != nil {
		return nil, err
	}
	if len(feeds) == 0 {
		return &SaveFeedsResponse{Saved: 0, Files: nil}, nil
	}

	if len(feeds) > limit {
		feeds = feeds[:limit]
	}

	// 2) 获取每条详情
	detailAction := xiaohongshu.NewFeedDetailAction(page)

	type item struct {
		Title   string
		Content string
	}

	items := make([]item, 0, len(feeds))
	for _, f := range feeds {
		d, err := detailAction.GetFeedDetail(ctx, f.ID, f.XsecToken)
		if err != nil {
			// 跳过失败项，继续抓取其它
			continue
		}

		note := d.Note
		title := strings.TrimSpace(note.Title)
		if title == "" {
			title = f.NoteCard.DisplayTitle
		}
		if title == "" {
			title = f.ID
		}

		var sb strings.Builder
		sb.WriteString("# ")
		sb.WriteString(title)
		sb.WriteString("\n\n")

		sb.WriteString("- NoteID: ")
		sb.WriteString(note.NoteID)
		sb.WriteString("\n- Type: ")
		sb.WriteString(note.Type)
		if note.User.Nickname != "" {
			sb.WriteString("\n- Author: ")
			sb.WriteString(note.User.Nickname)
			if note.User.UserID != "" {
				sb.WriteString(" (@")
				sb.WriteString(note.User.UserID)
				sb.WriteString(")")
			}
		}
		if note.InteractInfo.LikedCount != "" {
			sb.WriteString("\n- Likes: ")
			sb.WriteString(note.InteractInfo.LikedCount)
		}
		if note.IPLocation != "" {
			sb.WriteString("\n- Location: ")
			sb.WriteString(note.IPLocation)
		}
		sb.WriteString("\n\n## 内容\n\n")
		if strings.TrimSpace(note.Desc) != "" {
			sb.WriteString(note.Desc)
			sb.WriteString("\n\n")
		}

		if len(note.ImageList) > 0 {
			sb.WriteString("## 图片\n")
			for _, img := range note.ImageList {
				if img.URLDefault != "" {
					sb.WriteString("- ")
					sb.WriteString(img.URLDefault)
					sb.WriteString("\n")
				} else if img.URLPre != "" {
					sb.WriteString("- ")
					sb.WriteString(img.URLPre)
					sb.WriteString("\n")
				}
			}
			sb.WriteString("\n")
		}

		items = append(items, item{Title: title, Content: sb.String()})
	}

	if len(items) == 0 {
		return &SaveFeedsResponse{Saved: 0, Files: nil}, nil
	}

	// 3) 按标题排序
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})

	// 4) 保存为文件（按排序后的顺序，前缀序号确保文件名排序与标题排序一致）
	files := make([]string, 0, len(items))
	width := len(fmt.Sprintf("%d", len(items)))
	for idx, it := range items {
		safeTitle := sanitizeFilename(it.Title)
		baseName := fmt.Sprintf("%0*d_%s.md", width, idx+1, safeTitle)
		fullPath := filepath.Join(batchDir, baseName)
		// 避免重名（极端情况）
		fullPath = ensureUniquePath(fullPath)
		if err := os.WriteFile(fullPath, []byte(it.Content), 0o644); err != nil {
			continue
		}
		files = append(files, fullPath)
	}

	return &SaveFeedsResponse{Saved: len(files), Files: files}, nil
}

// sanitizeFilename 移除/替换 Windows 非法文件名字符
func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "untitled"
	}
	// 替换非法字符
	illegal := []string{"<", ">", ":", "\"", "\\", "/", "|", "?", "*"}
	for _, ch := range illegal {
		name = strings.ReplaceAll(name, ch, "_")
	}
	// 避免 Windows 保留名
	reserved := map[string]struct{}{
		"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
		"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
		"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
	}
	upper := strings.ToUpper(name)
	if _, ok := reserved[upper]; ok {
		name = name + "_"
	}
	// 限制长度，避免路径过长问题
	if len(name) > 80 {
		name = name[:80]
	}
	return name
}

// ensureUniquePath 如果文件已存在，则在文件名后追加 (-1), (-2) ...
func ensureUniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	for i := 1; i < 1000; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s-(%d)%s", name, i, ext))
		if _, err := os.Stat(cand); os.IsNotExist(err) {
			return cand
		}
	}
	return path
}
