package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	deepseekBaseURL = "https://api.deepseek.com/v1/chat/completions"
	deepseekModel   = "deepseek-chat"
)

// 60s 超时：LLM 生成可能比较慢，但也不能无限等
var deepseekClient = &http.Client{Timeout: 60 * time.Second}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []message       `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// callDeepSeek 调用 DeepSeek（OpenAI 兼容接口），返回 content 字符串。
// systemPrompt 内嵌 Etsy SEO 规则；userPrompt 是用户输入。
func callDeepSeek(systemPromptMsg, userPrompt string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY not set")
	}

	reqBody := chatRequest{
		Model: deepseekModel,
		Messages: []message{
			{Role: "system", Content: systemPromptMsg},
			{Role: "user", Content: userPrompt},
		},
		// 强制返回 JSON，避免模型输出 markdown 代码块或解释文字
		ResponseFormat: &responseFormat{Type: "json_object"},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, deepseekBaseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := deepseekClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call deepseek: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	// 非 200 时把响应体一起带上，方便排查
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("deepseek api error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("deepseek error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("deepseek returned empty choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}
