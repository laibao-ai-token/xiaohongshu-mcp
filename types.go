package main

// HTTP API 响应类型

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details any    `json:"details,omitempty"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

// JSON-RPC 相关类型

// JSONRPCRequest JSON-RPC 请求
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      any    `json:"id"`
}

// JSONRPCResponse JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      any           `json:"id"`
}

// JSONRPCError JSON-RPC 错误
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP 相关类型

// MCPToolCall MCP 工具调用
type MCPToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// MCPToolResult MCP 工具结果
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent MCP 内容
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// FeedDetailRequest Feed详情请求
type FeedDetailRequest struct {
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
}

// FeedDetailResponse Feed详情响应
type FeedDetailResponse struct {
	FeedID string `json:"feed_id"`
	Data   any    `json:"data"`
}

// PostCommentRequest 发表评论请求
type PostCommentRequest struct {
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
	Content   string `json:"content" binding:"required"`
}

// PostCommentResponse 发表评论响应
type PostCommentResponse struct {
	FeedID  string `json:"feed_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SaveFeedsRequest 保存推荐内容请求
type SaveFeedsRequest struct {
	// Limit 保存前 N 条推荐，默认 10
	Limit int `json:"limit"`
	// OutputDir 输出目录（可选），默认使用工作目录下的 content 目录
	OutputDir string `json:"output_dir"`
}

// SaveFeedsResponse 保存推荐内容响应
type SaveFeedsResponse struct {
	Saved int      `json:"saved"`
	Files []string `json:"files"`
}
