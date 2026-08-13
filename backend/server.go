package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
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

	// 后处理兜底：Etsy 标签限制 20 字符，超长的截断到最后一个空格保持词完整
	for i, tag := range seo.Tags {
		if len(tag) > 20 {
			cut := tag[:20]
			if sp := strings.LastIndex(cut, " "); sp > 0 {
				cut = cut[:sp]
			}
			seo.Tags[i] = cut
		}
	}

	writeJSON(w, http.StatusOK, generateResponse{
		Title:       seo.Title,
		Tags:        seo.Tags,
		Description: seo.Description,
	})
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
