package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"
)

// 前端 -> 后端的请求体
type generateRequest struct {
	Product  string `json:"product"`
	Keywords string `json:"keywords"`
	Category string `json:"category"`
}

// 后端 -> 前端的响应体
type generateResponse struct {
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

// DeepSeek 返回的 content 字段本身就是一段 JSON，需要二次解析
type seoContent struct {
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if strings.TrimSpace(req.Product) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "product description is required"})
		return
	}

	userPrompt := buildUserPrompt(req.Product, req.Keywords, req.Category)

	content, err := callDeepSeek(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("deepseek error: %v", err)
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "failed to generate content, please try again"})
		return
	}

	// DeepSeek 返回的是 JSON 字符串（因为强制了 response_format: json_object）
	var seo seoContent
	if err := json.Unmarshal([]byte(content), &seo); err != nil {
		log.Printf("parse seo content failed: %v, raw: %s", err, content)
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "failed to parse generated content"})
		return
	}

	if seo.Tags == nil {
		seo.Tags = []string{}
	}

	// 后处理兜底：Etsy 标签限制 20 字符。
	// 1. 用 rune 计数（非 ASCII 也安全），超长截到 20 字符内最后一个单词边界。
	// 2. 去掉结尾停用词，避免 "guest book for wedding" 被截成 "guest book for" 这类碎片。
	for i, tag := range seo.Tags {
		if utf8.RuneCountInString(tag) > 20 {
			runes := []rune(tag)
			cut := string(runes[:20])
			if sp := strings.LastIndex(cut, " "); sp > 0 {
				cut = cut[:sp]
			}
			cut = trimTrailingStopwords(cut)
			seo.Tags[i] = cut
		}
	}

	// 兜底硬校验：剔除封店级高危词（部落/品牌/医疗）。
	// tag 命中整条丢弃；标题/描述命中移除该词。宁可少一个 tag，不留法律风险。
	sanitizeSEO(&seo)
	// 声明校验：vegan/sterling silver 等声明词只有逐字出现在卖家输入里才保留。
	sanitizeClaims(&seo, req.Product, req.Keywords, req.Category)

	writeJSON(w, http.StatusOK, generateResponse{
		Title:       seo.Title,
		Tags:        seo.Tags,
		Description: seo.Description,
	})
}

// trailingStopwords 是标签末尾应去掉的停用词，防止截断产生 "guest book for" 碎片。
var trailingStopwords = map[string]bool{
	"for": true, "the": true, "a": true, "of": true, "to": true,
	"and": true, "with": true, "in": true, "on": true, "at": true,
	"or": true, "by": true, "is": true, "an": true,
}

// trimTrailingStopwords 循环去掉标签末尾的停用词，直到以实义词结尾。
func trimTrailingStopwords(tag string) string {
	for {
		sp := strings.LastIndex(tag, " ")
		if sp < 0 {
			return tag
		}
		if trailingStopwords[strings.ToLower(tag[sp+1:])] {
			tag = tag[:sp]
			continue
		}
		return tag
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
